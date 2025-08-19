package replay

import (
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"
)

// ApplyLog applies a single log entry to the pool's state.
func ApplyLog(pool types.RewardPool, log types.WalLogEntry) {
	switch v := log.(type) {
	case *types.WalLogDrawItem:
		if v.Success {
			pool.ApplyDrawLog(v.ItemID)
		}
	case *types.WalLogUpdateItem:
		pool.ApplyUpdateLog(v.ItemID, v.Quantity, v.Probability)
		// Other log types like Rotate or Snapshot are not applied to the pool state itself.
	}
}

// ReplayLogs applies a series of log entries to the pool's state.
func ReplayLogs(pool types.RewardPool, logs []types.WalLogEntry) {
	for _, item := range logs {
		ApplyLog(pool, item)
	}
}
