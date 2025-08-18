package actor

import "github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"

// DrawMessage is sent to the actor to request a reward draw.
type DrawMessage struct {
	RequestID    uint64
	ResponseChan chan DrawResponse
}

// DrawResponse is the response sent back for a DrawMessage.
type DrawResponse struct {
	RequestID uint64
	Item      string
	Err       error
}

// StopMessage is sent to the actor to request a graceful shutdown.
type StopMessage struct {
	ResponseChan chan struct{}
}

// FlushMessage is sent to the actor to manually trigger a WAL flush.
type FlushMessage struct {
	ResponseChan chan error
}

// SnapshotMessage is sent to the actor to manually trigger a snapshot.
type SnapshotMessage struct {
	ResponseChan chan error
}

// StateMessage is sent to the actor to request the current pool state.
type StateMessage struct {
	ResponseChan chan []types.PoolReward
}

// UpdateMessage is sent to the actor to update an item's properties.
type UpdateMessage struct {
	ItemID       string
	Quantity     int
	Probability  int64
	ResponseChan chan error
}
