package walstream

import "github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"

// NoOpStreamer is a WALStreamer that does nothing.
// It is used when WAL streaming is disabled.
type NoOpStreamer struct{}

// NewNoOpStreamer creates a new NoOpStreamer.
func NewNoOpStreamer() *NoOpStreamer {
	return &NoOpStreamer{}
}

// Stream does nothing.
func (s *NoOpStreamer) Stream(log types.WalLogEntry) {}
