package actor

import (
	"context"

	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/walstream"
)

// StreamingActor is responsible for streaming WAL logs to a replica.
// It runs in its own goroutine and processes log entries from its mailbox.
type StreamingActor struct {
	walStreamer walstream.WALStreamer
	mailbox     chan types.WalLogEntry
}

// NewStreamingActor creates a new StreamingActor.
func NewStreamingActor(walStreamer walstream.WALStreamer, mailboxSize int) *StreamingActor {
	return &StreamingActor{
		walStreamer: walStreamer,
		mailbox:     make(chan types.WalLogEntry, mailboxSize),
	}
}

func (a *StreamingActor) Init() error {
	return nil
}

// Receive starts the actor's message processing loop.
func (a *StreamingActor) Receive(ctx context.Context) {
	for {
		select {
		case logEntry := <-a.mailbox:
			a.walStreamer.Stream(logEntry)
		case <-ctx.Done():
			// Drain the mailbox before shutting down
			for logEntry := range a.mailbox {
				a.walStreamer.Stream(logEntry)
			}
			return
		}
	}
}
