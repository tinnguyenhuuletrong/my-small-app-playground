package types

import "log/slog"

// LogType defines the type of a WAL log entry.
type LogType byte

const (
	LogTypeDraw LogType = iota + 1
)

// LogError defines the type of a WAL log error.
type LogError byte

const (
	ErrorNone LogError = iota
	ErrorPoolEmpty
	ErrorItemNotFound
)

// ConfigPool represents the configuration for the reward pool
type ConfigPool struct {
	Catalog []PoolReward `json:"catalog"`
}

// PoolReward represents a reward item in the pool
type PoolReward struct {
	ItemID      string `json:"item_id"`
	Quantity    int    `json:"quantity"`
	Probability int64  `json:"probability"`
}

// WalLogItem represents a WAL log entry
type WalLogItem struct {
	Type  LogType  `json:"type"`
	Error LogError `json:"error,omitempty"`
}

// WalLogDrawItem represents a WAL log entry for a draw operation
type WalLogDrawItem struct {
	WalLogItem
	RequestID uint64 `json:"request_id"`
	ItemID    string `json:"item_id,omitempty"`
	Success   bool   `json:"success"`
}

// RewardPool interface
type RewardPool interface {
	SelectItem(ctx *Context) (string, error)
	CommitDraw()
	RevertDraw()
	Load(config ConfigPool) error
	SaveSnapshot(path string) error
	LoadSnapshot(path string) error
}

// WAL interface
// WAL interface with buffered logging
type WAL interface {
	// LogDraw appends a log entry to the buffer (does not write to disk immediately)
	LogDraw(item WalLogDrawItem) error
	// Flush writes all buffered log entries to disk
	Flush() error
	// Close closes the WAL file
	Close() error
	// Rotate file
	Rotate(path string) error
}

// Config interface
type Config interface {
	LoadConfig(path string) (ConfigPool, error)
}

// Context for dependency injection
type Context struct {
	WAL   WAL
	Utils Utils
}

// Utils interface for random selection
type Utils interface {
	GetLogger() *slog.Logger
}

// ItemSelector defines the contract for selecting items from a reward pool.
// It abstracts the underlying data structure used for efficient selection.
type ItemSelector interface {
	// Select chooses an item based on its availability and returns its ID.
	Select(ctx *Context) (string, error)

	// Update adjusts the quantity of a specific item in the selector.
	// A positive value increases availability, a negative value decreases it.
	Update(itemID string, delta int64)

	// Reset clears the selector's state and re-initializes it with a new catalog.
	Reset(catalog []PoolReward)

	// TotalAvailable returns the total count of all items currently available for selection.
	TotalAvailable() int64

	// GetItemRemaining returns the remaining quantity of a specific item.
	GetItemRemaining(itemID string) int

	// Return PoolReward[] for Snapshot
	SnapshotCatalog() []PoolReward
}

// Error
type errString string

func (e errString) Error() string {
	return string(e)
}

const ErrWalBufferNotEmpty = errString("Wal buffer is not empty. Should Flush before rotate")
const ErrEmptyRewardPool = errString("reward pool is empty")
const ErrPendingDrawsNotEmpty = errString("PendingDraws remaining. Please CommitDraw or RevertDraw before")
const ErrShutingDown = errString("request cancelled: processor shutting down")
