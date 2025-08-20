package walstream

import "github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"

// WALStreamer defines the interface for streaming WAL logs to a replica.
type WALStreamer interface {
	// Stream sends a WAL log entry to the replica.
	// This method should be non-blocking.
	Stream(log types.WalLogEntry)
}
