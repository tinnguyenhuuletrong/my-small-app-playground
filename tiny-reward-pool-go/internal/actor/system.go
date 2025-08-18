package actor

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"
)

// System manages the lifecycle of an actor and provides a client-facing API.
type System struct {
	actor     *RewardProcessorActor
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	stopOnce  sync.Once
	requestID uint64 // Add requestID counter
}

// SystemOptional provides optional parameters for creating a new System.
type SystemOptional struct {
	FlushAfterNDraw   int
	RequestBufferSize int
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

	actorInstance := NewRewardProcessorActor(ctx, pool, bufSize, flushN)

	if err := actorInstance.Init(); err != nil {
		// If init fails, we must ensure the WAL is closed if it was opened.
		actorInstance.ctx.WAL.Close()
		return nil, fmt.Errorf("actor initialization failed: %w", err)
	}

	actorCtx, cancel := context.WithCancel(context.Background())

	sys := &System{
		actor:  actorInstance,
		cancel: cancel,
	}

	sys.wg.Add(1)
	go func() {
		defer sys.wg.Done()
		sys.actor.Receive(actorCtx)
	}()

	return sys, nil
}

// Draw sends a draw request to the actor and waits for a response.
func (s *System) Draw() <-chan DrawResponse {
	reqID := atomic.AddUint64(&s.requestID, 1) // Increment requestID
	respChan := make(chan DrawResponse, 1)
	msg := DrawMessage{RequestID: reqID, ResponseChan: respChan} // Pass requestID
	s.actor.mailbox <- msg
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
	s.actor.mailbox <- msg
	return <-respChan
}

// Snapshot manually triggers a snapshot.
func (s *System) Snapshot() error {
	respChan := make(chan error, 1)
	msg := SnapshotMessage{ResponseChan: respChan}
	s.actor.mailbox <- msg
	return <-respChan
}

// UpdateItem sends a message to the actor to update an item and waits for a response.
func (s *System) UpdateItem(itemID string, quantity int, probability int64) error {
	respChan := make(chan error, 1)
	s.actor.mailbox <- UpdateMessage{
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
	s.actor.mailbox <- StateMessage{ResponseChan: respChan}
	return <-respChan
}
