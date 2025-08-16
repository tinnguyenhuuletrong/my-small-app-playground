package actor

import (
	"context"
	"fmt"

	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"
)

// RewardProcessorActor encapsulates the state and behavior of the reward processing.
// It is designed to be run in a single goroutine, processing messages from its mailbox.
type RewardProcessorActor struct {
	ctx             *types.Context
	pool            types.RewardPool
	mailbox         chan interface{}
	flushAfterNDraw int
	stagedDraws     int
}

// NewRewardProcessorActor creates a new actor instance.
func NewRewardProcessorActor(ctx *types.Context, pool types.RewardPool, mailboxSize, flushAfterNDraw int) *RewardProcessorActor {
	return &RewardProcessorActor{
		ctx:             ctx,
		pool:            pool,
		mailbox:         make(chan interface{}, mailboxSize),
		flushAfterNDraw: flushAfterNDraw,
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
	case StateMessage:
		// This is a read-only operation, so it's safe to do directly.
		// Note: In a more complex actor, even reads might be message-based
		// to ensure sequential consistency with writes.
		m.ResponseChan <- a.pool.State()
	}
}

func (a *RewardProcessorActor) handleDraw(m DrawMessage) {
	item, err := a.pool.SelectItem(a.ctx)
	var walErr error

	logItem := types.WalLogDrawItem{
		WalLogEntryBase: types.WalLogEntryBase{Type: types.LogTypeDraw},
		RequestID:       m.RequestID, // Use RequestID from message
		Success:         err == nil,
	}

	if logItem.Success {
		logItem.ItemID = item
	} else if err == types.ErrEmptyRewardPool {
		logItem.Error = types.ErrorPoolEmpty
	}

	walErr = a.ctx.WAL.LogDraw(logItem)
	a.stagedDraws++

	if a.stagedDraws >= a.flushAfterNDraw {
		a.flush()
	}

	resp := DrawResponse{RequestID: m.RequestID, Err: err} // Add RequestID to response
	if walErr == nil {
		resp.Item = item
	} else {
		resp.Err = walErr
	}

	m.ResponseChan <- resp
}

func (a *RewardProcessorActor) flush() error {
	if a.stagedDraws <= 0 {
		return nil
	}

	flushErr := a.ctx.WAL.Flush()

	if flushErr != nil {
		if flushErr == types.ErrWALFull {
			// WAL is full. The buffer is not flushed. Revert draws.
			a.pool.RevertDraw()
			a.stagedDraws = 0
			if logger := a.ctx.Utils.GetLogger(); logger != nil {
				logger.Info("WAL is full. Reverting draws and rotating WAL.")
			}

			// Now rotate WAL
			rotatedPath := a.ctx.Utils.GenRotatedWALPath()
			if rotatedPath != nil {
				if err := a.ctx.WAL.Rotate(*rotatedPath); err != nil {
					if logger := a.ctx.Utils.GetLogger(); logger != nil {
						logger.Error("Failed to rotate WAL.", "error", err)
					}
					return err
				}
			} else {
				if logger := a.ctx.Utils.GetLogger(); logger != nil {
					logger.Error("Wall is full, rotatedPath not set. Noting to do. Stop here")
				}

				// too strict
				// panic(1)
			}
			a.ctx.WAL.Reset()

			// Create snapshot and log it to the new WAL
			if err := a.snapshot(); err != nil {
				return err
			}
			// And flush the snapshot log
			return a.ctx.WAL.Flush()

		} else {
			// Another flush error. Revert draws.
			a.pool.RevertDraw()
			a.stagedDraws = 0
			if logger := a.ctx.Utils.GetLogger(); logger != nil {
				logger.Error("[Actor] WAL Flush failed, reverting draws.", "error", flushErr)
			}
			return flushErr
		}
	}

	// Flush was successful. Commit draws.
	a.pool.CommitDraw()

	if logger := a.ctx.Utils.GetLogger(); logger != nil {
		logger.Debug(fmt.Sprintf("[Actor] WAL Flush and Commit - %d", a.stagedDraws))
	}
	a.stagedDraws = 0
	return nil
}

func (a *RewardProcessorActor) snapshot() error {
	snapshotPath := a.ctx.Utils.GenSnapshotPath()
	if snapshotPath != nil {
		if logger := a.ctx.Utils.GetLogger(); logger != nil {
			logger.Info("Creating snapshot.", "path", *snapshotPath)
		}
		if err := a.pool.SaveSnapshot(*snapshotPath); err != nil {
			if logger := a.ctx.Utils.GetLogger(); logger != nil {
				logger.Error("Failed to create snapshot after WAL rotation.", "error", err)
			}
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
