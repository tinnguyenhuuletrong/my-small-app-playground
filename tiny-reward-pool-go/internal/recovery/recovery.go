package recovery

import (
	"fmt"
	"os"

	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/rewardpool"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/wal"
)

// RecoverPool loads the pool from snapshot, replays WAL, writes new snapshot, and rotates WAL.
func RecoverPool(snapshotPath, walPath, configPath string) (*rewardpool.Pool, error) {
	pool := &rewardpool.Pool{}
	// 1. Load snapshot or fallback to config
	if err := pool.LoadSnapshot(snapshotPath); err != nil {
		loaded, err := rewardpool.LoadPool(configPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load config: %w", err)
		}
		pool = loaded
	}

	// 2. Replay WAL log for recovery
	items, err := wal.ParseWAL(walPath)
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
