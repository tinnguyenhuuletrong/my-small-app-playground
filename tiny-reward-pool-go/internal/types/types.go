package types

import "log/slog"

// LogType defines the type of a WAL log entry.
type LogType byte

const (
	LogTypeDraw LogType = iota + 1
	LogTypeUpdate
	LogTypeSnapshot
	LogTypeRotate
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

// WalLogEntry defines the interface for a WAL log entry.
type WalLogEntry interface {
	GetType() LogType
}

// WalLogEntryBase is a base struct for log entries that implements the WalLogEntry interface.
type WalLogEntryBase struct {
	Type  LogType  `json:"type"`
	Error LogError `json:"error,omitempty"`
}

func (b WalLogEntryBase) GetType() LogType {
	return b.Type
}

// WalLogDrawItem represents a WAL log entry for a draw operation
type WalLogDrawItem struct {
	WalLogEntryBase
	RequestID uint64 `json:"request_id"`
	ItemID    string `json:"item_id,omitempty"`
	Success   bool   `json:"success"`
}

// WalLogUpdateItem represents a WAL log entry for an update operation
type WalLogUpdateItem struct {
	WalLogEntryBase
	ItemID      string `json:"item_id"`
	Quantity    int    `json:"quantity"`
	Probability int64  `json:"probability"`
}

// WalLogSnapshotItem represents a WAL log entry for a snapshot operation
type WalLogSnapshotItem struct {
	WalLogEntryBase
	Path string `json:"path"`
}

// WalLogRotateItem represents a WAL log entry for a rotate operation
type WalLogRotateItem struct {
	WalLogEntryBase
	OldPath string `json:"old_path"`
	NewPath string `json:"new_path"`
}

// RewardPool interface
type RewardPool interface {
	SelectItem(ctx *Context) (string, error)
	CommitDraw()
	RevertDraw()
	State() []PoolReward
	Load(config ConfigPool) error
	SaveSnapshot(path string) error
	LoadSnapshot(path string) error
	ApplyUpdateLog(itemID string, quantity int, probability int64)
}

// LogFormatter Interface: To handle serialization and deserialization.
type LogFormatter interface {
	// Batched encode. Should call in Flush
	Encode(items []WalLogEntry) ([]byte, error)

	// Batched decode. Should call in Parse
	Decode(data []byte) ([]WalLogEntry, error)
}

// Storage Interface: To handle the physical writing, reading, and management of the log medium.
type Storage interface {
	Write([]byte) error
	CanWrite(size int) bool
	Flush() error
	Close() error

	// Finalize current file and move to archivePath.
	// Then reset and continue to use the current one
	Rotate(archivePath string) error
}

// WAL interface
// WAL interface with buffered logging
type WAL interface {
	LogDraw(item WalLogDrawItem) error
	LogUpdate(item WalLogUpdateItem) error
	LogSnapshot(item WalLogSnapshotItem) error
	LogRotate(item WalLogRotateItem) error

	// Flush writes all buffered log entries to disk
	Flush() error
	// Close closes the WAL file
	Close() error
	// Rotate file
	Rotate(path string) error
	// Reset buffer
	Reset()
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

// Utils provides an interface for environment-specific operations like logging and path generation.
type Utils interface {
	GetLogger() *slog.Logger
	GenRotatedWALPath() *string // Path for the archived WAL. nil means skip archiving.
	GenSnapshotPath() *string   // Path for the new snapshot. nil means skip snapshotting.
}

// ItemSelector defines the contract for selecting items from a reward pool.
// It abstracts the underlying data structure used for efficient selection.
type ItemSelector interface {
	// Select chooses an item based on its availability and returns its ID.
	Select(ctx *Context) (string, error)

	// Update adjusts the quantity of a specific item in the selector.
	// A positive value increases availability, a negative value decreases it.
	Update(itemID string, delta int64)

	// UpdateItem updates the quantity and probability of a specific item.
	UpdateItem(itemID string, quantity int, probability int64)

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
const ErrWALFull = errString("WAL is full, rotation is required")
const ErrEmptyRewardPool = errString("reward pool is empty")
const ErrPendingDrawsNotEmpty = errString("PendingDraws remaining. Please CommitDraw or RevertDraw before")
const ErrShutingDown = errString("request cancelled: processor shutting down")
