package actor

import (
	"context"
	"fmt"
	"sync"

	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/wal"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/wal/formatter"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/wal/storage"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/walstream"
)

// System manages the lifecycle of an actor and provides a client-facing API.
type System struct {
	processorActor *RewardProcessorActor
	streamingActor *StreamingActor
	cancel         context.CancelFunc
	wg             sync.WaitGroup
	stopOnce       sync.Once
}

// SystemOptional provides optional parameters for creating a new System.
type SystemOptional struct {
	FlushAfterNDraw   int
	RequestBufferSize int
	LastRequestID     uint64
	WALStreamer       walstream.WALStreamer
	WALFactory        func(path string, seqNo uint64) (types.WAL, error)
}

// NewSystem creates, starts, and returns a new actor system.
func NewSystem(ctx *types.Context, pool types.RewardPool, opt *SystemOptional) (*System, error) {
	flushN := 10
	if opt != nil && opt.FlushAfterNDraw > 0 {
		flushN = opt.FlushAfterNDraw
	}
	bufSize := 100
	if opt != nil && opt.RequestBufferSize > 0 {
		bufSize = opt.RequestBufferSize
	}
	lastRequestID := uint64(0)
	if opt != nil && opt.LastRequestID > 0 {
		lastRequestID = opt.LastRequestID
	}

	var walFactory func(path string, seqNo uint64) (types.WAL, error)
	if opt != nil && opt.WALFactory != nil {
		walFactory = opt.WALFactory
	} else {
		// Default WALFactory
		walFactory = func(path string, seqNo uint64) (types.WAL, error) {
			fileStorage, err := storage.NewFileStorage(path, seqNo)
			if err != nil {
				return nil, err
			}
			return wal.NewWAL(path, seqNo, formatter.NewJSONFormatter(), fileStorage)
		}
	}

	processorActor := NewRewardProcessorActor(ctx, pool, bufSize, flushN, lastRequestID, walFactory)
	if err := processorActor.Init(); err != nil {
		// If init fails, we must ensure the WAL is closed if it was opened.
		processorActor.ctx.WAL.Close()
		return nil, fmt.Errorf("actor initialization failed: %w", err)
	}

	var streamingActor *StreamingActor = nil
	if opt != nil && opt.WALStreamer != nil {
		streamingActor = NewStreamingActor(opt.WALStreamer, bufSize)
		if err := streamingActor.Init(); err != nil {
			return nil, fmt.Errorf("streamingActor initialization failed: %w", err)
		}

		processorActor.SetStreamChannel(streamingActor.mailbox)
	}

	actorCtx, cancel := context.WithCancel(context.Background())

	sys := &System{
		processorActor: processorActor,
		streamingActor: streamingActor,
		cancel:         cancel,
	}

	sys.wg.Add(2)
	go func() {
		defer sys.wg.Done()
		sys.processorActor.Receive(actorCtx)
	}()
	go func() {
		defer sys.wg.Done()
		if sys.streamingActor == nil {
			return
		}
		sys.streamingActor.Receive(actorCtx)
	}()

	return sys, nil
}

// Draw sends a draw request to the actor and waits for a response.
func (s *System) Draw() <-chan DrawResponse {
	respChan := make(chan DrawResponse, 1)
	msg := DrawMessage{ResponseChan: respChan}
	s.processorActor.mailbox <- msg
	return respChan
}

// Stop gracefully shuts down the actor system.
func (s *System) Stop() {
	s.stopOnce.Do(func() {
		s.cancel()  // Signal the actor to stop
		s.wg.Wait() // Wait for the actor's goroutine to finish
	})
}

// Flush manually triggers a WAL flush.
func (s *System) Flush() error {
	respChan := make(chan error, 1)
	msg := FlushMessage{ResponseChan: respChan}
	s.processorActor.mailbox <- msg
	return <-respChan
}

// Snapshot manually triggers a snapshot.
func (s *System) Snapshot() error {
	respChan := make(chan error, 1)
	msg := SnapshotMessage{ResponseChan: respChan}
	s.processorActor.mailbox <- msg
	return <-respChan
}

// UpdateItem sends a message to the actor to update an item and waits for a response.
func (s *System) UpdateItem(itemID string, quantity int, probability int64) error {
	respChan := make(chan error, 1)
	s.processorActor.mailbox <- UpdateMessage{
		ItemID:       itemID,
		Quantity:     quantity,
		Probability:  probability,
		ResponseChan: respChan,
	}
	return <-respChan
}

// State returns the current state of the reward pool.
func (s *System) State() []types.PoolReward {
	respChan := make(chan []types.PoolReward, 1)
	s.processorActor.mailbox <- StateMessage{ResponseChan: respChan}
	return <-respChan
}

// GetRequestID returns the current request ID from the actor.
func (s *System) GetRequestID() uint64 {
	respChan := make(chan uint64, 1)
	s.processorActor.mailbox <- GetRequestIDMessage{ResponseChan: respChan}
	return <-respChan
}

// SetRequestID sets the request ID on the actor.
func (s *System) SetRequestID(id uint64) {
	respChan := make(chan struct{}, 1)
	s.processorActor.mailbox <- SetRequestIDMessage{ID: id, ResponseChan: respChan}
	<-respChan
}
