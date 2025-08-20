package actor

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync/atomic"

	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/replay"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"
)

// RewardProcessorActor encapsulates the state and behavior of the reward processing.
// It is designed to be run in a single goroutine, processing messages from its mailbox.
type RewardProcessorActor struct {
	ctx             *types.Context
	pool            types.RewardPool
	mailbox         chan interface{}
	flushAfterNDraw int
	pendingLogs     []types.WalLogEntry
	requestID       uint64
}

// Init performs the initial setup for the actor, like creating an initial
// snapshot if the WAL is empty. It's called once when the actor starts.
func (a *RewardProcessorActor) Init() error {

	size, err := a.ctx.WAL.Size()
	if err != nil {
		return fmt.Errorf("could not determine WAL size: %w", err)
	}

	if size == 0 {
		if logger := a.ctx.Utils.GetLogger(); logger != nil {
			logger.Info("WAL is empty, creating initial snapshot.")
		}
		if err := a.snapshot(); err != nil {
			return fmt.Errorf("failed to create initial snapshot: %w", err)
		}
		// The snapshot log is staged in the WAL's buffer, flush it to disk.
		return a.ctx.WAL.Flush()
	}

	return nil
}

// NewRewardProcessorActor creates a new actor instance.
func NewRewardProcessorActor(ctx *types.Context, pool types.RewardPool, mailboxSize, flushAfterNDraw int, requestID uint64) *RewardProcessorActor {
	return &RewardProcessorActor{
		ctx:             ctx,
		pool:            pool,
		mailbox:         make(chan interface{}, mailboxSize),
		flushAfterNDraw: flushAfterNDraw,
		pendingLogs:     make([]types.WalLogEntry, 0, flushAfterNDraw*2),
		requestID:       requestID,
	}
}

// Receive starts the actor's message processing loop.
// This method is expected to be called in its own goroutine.
func (a *RewardProcessorActor) Receive(ctx context.Context) {
	for {
		select {
		case msg := <-a.mailbox:
			a.handleMessage(msg)
		case <-ctx.Done():
			// Context was cancelled, perform graceful shutdown.
			a.shutdown()
			return
		}
	}
}

func (a *RewardProcessorActor) handleMessage(msg interface{}) {
	switch m := msg.(type) {
	case DrawMessage:
		a.handleDraw(m)
	case StopMessage:
		a.shutdown()
		close(m.ResponseChan)
	case FlushMessage:
		m.ResponseChan <- a.flush()
	case SnapshotMessage:
		m.ResponseChan <- a.snapshot()
	case UpdateMessage:
		a.handleUpdate(m)
	case StateMessage:
		// This is a read-only operation, so it's safe to do directly.
		// Note: In a more complex actor, even reads might be message-based
		// to ensure sequential consistency with writes.
		m.ResponseChan <- a.pool.State()
	case GetRequestIDMessage:
		m.ResponseChan <- a.requestID
	case SetRequestIDMessage:
		a.requestID = m.ID
		close(m.ResponseChan)
	}
}

func (a *RewardProcessorActor) handleDraw(m DrawMessage) {
	reqID := atomic.AddUint64(&a.requestID, 1)
	item, err := a.pool.SelectItem(a.ctx)
	var walErr error

	logItem := types.WalLogDrawItem{
		WalLogEntryBase: types.WalLogEntryBase{Type: types.LogTypeDraw},
		RequestID:       reqID,
		Success:         err == nil,
	}

	if logItem.Success {
		logItem.ItemID = item
	} else if err == types.ErrEmptyRewardPool {
		logItem.Error = types.ErrorPoolEmpty
	}

	walErr = a.ctx.WAL.LogDraw(logItem)
	a.pendingLogs = append(a.pendingLogs, &logItem)

	if len(a.pendingLogs) >= a.flushAfterNDraw {
		a.flush()
	}

	resp := DrawResponse{RequestID: reqID, Err: err}
	if walErr == nil {
		resp.Item = item
	} else {
		resp.Err = walErr
	}

	m.ResponseChan <- resp
}

func (a *RewardProcessorActor) handleUpdate(m UpdateMessage) {
	err := a.pool.UpdateItem(m.ItemID, m.Quantity, m.Probability)
	if err != nil {
		m.ResponseChan <- err
		return
	}

	logItem := types.WalLogUpdateItem{
		WalLogEntryBase: types.WalLogEntryBase{Type: types.LogTypeUpdate},
		ItemID:          m.ItemID,
		Quantity:        m.Quantity,
		Probability:     m.Probability,
	}

	walErr := a.ctx.WAL.LogUpdate(logItem)
	if walErr == nil {
		a.pendingLogs = append(a.pendingLogs, &logItem)
	}
	m.ResponseChan <- walErr
}

func (a *RewardProcessorActor) flush() error {
	if len(a.pendingLogs) == 0 {
		return nil
	}

	flushErr := a.ctx.WAL.Flush()

	if flushErr != nil {
		if flushErr == types.ErrWALFull {
			return a.handleWALFull()
		}

		// Another flush error. Revert draws.
		a.pool.RevertDraw()
		a.pendingLogs = a.pendingLogs[:0]
		a.ctx.WAL.Reset() // Clear the unflushed buffer
		if logger := a.ctx.Utils.GetLogger(); logger != nil {
			logger.Error("[Actor] WAL Flush failed, reverting draws.", "error", flushErr)
		}
		return flushErr
	}

	// Flush was successful. Commit draws.
	a.pool.CommitDraw()

	if logger := a.ctx.Utils.GetLogger(); logger != nil {
		logger.Debug(fmt.Sprintf("[Actor] WAL Flush and Commit - %d logs", len(a.pendingLogs)))
	}
	a.pendingLogs = a.pendingLogs[:0]
	return nil
}

func (a *RewardProcessorActor) handleWALFull() error {
	if logger := a.ctx.Utils.GetLogger(); logger != nil {
		logger.Info("WAL is full. Reverting draws, rotating WAL, and re-applying logs.")
	}

	// 1. Preserve pending logs and revert in-memory state
	logsToReplay := make([]types.WalLogEntry, len(a.pendingLogs))
	copy(logsToReplay, a.pendingLogs)
	a.pool.RevertDraw()
	a.pendingLogs = a.pendingLogs[:0]
	a.ctx.WAL.Reset() // Clear the unflushed buffer in the WAL

	// 2. Rotate WAL file
	rotatedPath := a.ctx.Utils.GenRotatedWALPath()
	if rotatedPath != nil {
		if err := a.ctx.WAL.Rotate(*rotatedPath); err != nil {
			if logger := a.ctx.Utils.GetLogger(); logger != nil {
				logger.Error("Failed to rotate WAL.", "error", err)
			}
			// This is a critical failure, can't proceed.
			return err
		}
	}

	// 3. Create and log a snapshot to the new WAL
	if err := a.snapshot(); err != nil {
		// Also a critical failure.
		return err
	}
	// Flush the snapshot log immediately to secure the new WAL's starting state.
	if err := a.ctx.WAL.Flush(); err != nil {
		if logger := a.ctx.Utils.GetLogger(); logger != nil {
			logger.Error("CRITICAL: Could not flush snapshot to new WAL. State may be inconsistent.", "error", err)
		}
		return err
	}

	// 4. Re-apply and re-log the preserved operations
	a.replayAndRelog(logsToReplay)

	// 5. Final flush attempt on the new WAL
	if err := a.ctx.WAL.Flush(); err != nil {
		if logger := a.ctx.Utils.GetLogger(); logger != nil {
			logger.Error("CRITICAL: Flush failed even after WAL rotation. Data may be lost.", "error", err)
		}
		// At this point, recovery is difficult. We've already rotated and snapshotted.
		// The best we can do is revert the re-staged draws and report the error.
		a.pool.RevertDraw()
		a.pendingLogs = a.pendingLogs[:0]
		return err
	}

	return nil
}

func (a *RewardProcessorActor) replayAndRelog(logsToReplay []types.WalLogEntry) {
	if logger := a.ctx.Utils.GetLogger(); logger != nil {
		logger.Info("Replaying pending logs to the new WAL.", "count", len(logsToReplay))
	}
	for _, logEntry := range logsToReplay {
		// Re-apply the operation to the in-memory pool
		replay.ApplyLog(a.pool, logEntry)

		// Re-log the operation to the new WAL's buffer and the actor's pending list
		switch v := logEntry.(type) {
		case *types.WalLogDrawItem:
			a.ctx.WAL.LogDraw(*v)
			a.pendingLogs = append(a.pendingLogs, v)
		case *types.WalLogUpdateItem:
			a.ctx.WAL.LogUpdate(*v)
			a.pendingLogs = append(a.pendingLogs, v)
		}
	}
}

func (a *RewardProcessorActor) snapshot() error {
	snapshotPath := a.ctx.Utils.GenSnapshotPath()
	if snapshotPath == nil {
		return nil // Snapshotting is disabled
	}

	if logger := a.ctx.Utils.GetLogger(); logger != nil {
		logger.Info("Creating snapshot.", "path", *snapshotPath)
	}

	snap, err := a.pool.CreateSnapshot()
	if err != nil {
		if logger := a.ctx.Utils.GetLogger(); logger != nil {
			logger.Error("Failed to create snapshot data.", "error", err)
		}
		return err
	}

	// The actor is the owner of the request ID, so it sets it on the snapshot.
	snap.LastRequestID = a.requestID

	file, err := os.Create(*snapshotPath)
	if err != nil {
		return err
	}
	defer file.Close()

	enc := json.NewEncoder(file)
	if err := enc.Encode(snap); err != nil {
		return err
	}

	logItem := types.WalLogSnapshotItem{
		WalLogEntryBase: types.WalLogEntryBase{Type: types.LogTypeSnapshot},
		Path:            *snapshotPath,
	}
	if err := a.ctx.WAL.LogSnapshot(logItem); err != nil {
		if logger := a.ctx.Utils.GetLogger(); logger != nil {
			logger.Error("Failed to log snapshot to WAL.", "error", err)
		}
		return err
	}

	return nil
}

func (a *RewardProcessorActor) shutdown() {
	if a.ctx.Utils.GetLogger() != nil {
		a.ctx.Utils.GetLogger().Debug("[Actor] Shutdown")
	}

	// Drain mailbox and cancel pending requests
	close(a.mailbox)
	for msg := range a.mailbox {
		if m, ok := msg.(DrawMessage); ok {
			m.ResponseChan <- DrawResponse{Err: types.ErrShutingDown}
		}
	}

	a.flush()
	a.ctx.WAL.Close()
}
