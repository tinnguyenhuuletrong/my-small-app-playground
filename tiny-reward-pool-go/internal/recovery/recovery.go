package recovery

import (
	"fmt"
	"os"

	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/rewardpool"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/wal"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"
)

// RecoverPool loads the pool from snapshot, replays WAL, writes new snapshot, and rotates WAL.
func RecoverPool(snapshotPath, walPath, configPath string, formatter types.LogFormatter, storage types.Storage) (*rewardpool.Pool, error) {
	var pool *rewardpool.Pool

	// Try to load from snapshot first
	initialPool := rewardpool.NewPool([]types.PoolReward{}) // Create a pool with an empty catalog initially
	if err := initialPool.LoadSnapshot(snapshotPath); err == nil {
		pool = initialPool
	} else {
		// If snapshot fails, load from config
		loaded, err := rewardpool.CreatePoolFromConfigPath(configPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load config: %w", err)
		}
		pool = loaded
	}

	// 2. Replay WAL log for recovery
	items, err := wal.ParseWAL(walPath, formatter, storage)
	if err == nil {
		for _, entry := range items {
			if entry.Success {
				pool.ApplyDrawLog(entry.ItemID)
			}
		}
		// 3. Write new snapshot after recovery
		if err := pool.SaveSnapshot(snapshotPath); err != nil {
			return nil, fmt.Errorf("failed to save recovered snapshot: %w", err)
		}
		// 4. Rotate WAL log
		os.Remove(walPath)
	} // else: no WAL log found for recovery

	return pool, nil
}
