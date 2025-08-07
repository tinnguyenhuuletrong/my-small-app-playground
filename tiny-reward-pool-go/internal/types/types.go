package types

// ConfigPool represents the configuration for the reward pool
type ConfigPool struct {
	Catalog []PoolReward `json:"catalog"`
}

// PoolReward represents a reward item in the pool
type PoolReward struct {
	ItemID      string  `json:"item_id"`
	Quantity    int     `json:"quantity"`
	Probability float64 `json:"probability"`
}

// WalLogItem represents a WAL log entry
type WalLogItem struct {
	RequestID uint64
	ItemID    string
	Success   bool
}

// RewardPool interface
type RewardPool interface {
	SelectItem(ctx *Context) (*PoolReward, error)
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
	LogDraw(item WalLogItem) error
	// Flush writes all buffered log entries to disk
	Flush() error
	// Close closes the WAL file
	Close() error
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
	RandomItem(items []PoolReward) (int, error)
}
