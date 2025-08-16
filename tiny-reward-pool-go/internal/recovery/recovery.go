package recovery

import (
	"fmt"
	"os"

	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/rewardpool"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/wal"
)

// RecoverPool loads the pool from snapshot, replays WAL, writes new snapshot, and rotates WAL.
func RecoverPool(snapshotPath, walPath, configPath string, formatter types.LogFormatter, utils types.Utils) (*rewardpool.Pool, error) {
	var pool *rewardpool.Pool

	// 1. Parse the WAL file.
	logItems, err := wal.ParseWAL(walPath, formatter)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to parse WAL: %w", err)
		}
		// WAL file doesn't exist, try to load from snapshot or config.
		logItems = []types.WalLogEntry{}
	}

	// 2. Load the initial state.
	pool = rewardpool.NewPool([]types.PoolReward{}) // Create a pool with an empty catalog initially

	if len(logItems) == 0 {
		// No WAL, try snapshot then config.
		if err := pool.LoadSnapshot(snapshotPath); err != nil {
			// If snapshot fails, load from config as a last resort.
			loaded, cfgErr := rewardpool.CreatePoolFromConfigPath(configPath)
			if cfgErr != nil {
				return nil, fmt.Errorf("failed to load config after failed snapshot: %w", cfgErr)
			}
			pool = loaded
		}
	} else {
		// WAL exists, the first entry must be a snapshot.
		snapshotLog, ok := logItems[0].(*types.WalLogSnapshotItem)
		if !ok || snapshotLog.Type != types.LogTypeSnapshot {
			return nil, fmt.Errorf("first WAL entry must be a snapshot")
		}

		if err := pool.LoadSnapshot(snapshotLog.Path); err != nil {
			return nil, fmt.Errorf("failed to load snapshot from WAL: %w", err)
		}

		// Replay the rest of the WAL.
		for _, item := range logItems[1:] {
			switch v := item.(type) {
			case *types.WalLogDrawItem:
				if v.Success {
					pool.ApplyDrawLog(v.ItemID)
				}
			case *types.WalLogUpdateItem:
				pool.ApplyUpdateLog(v.ItemID, v.Quantity, v.Probability)
			// Other log types like Rotate are not applied to the pool state.
			}
		}
	}

	// 3. Write new snapshot after recovery.
	if err := pool.SaveSnapshot(snapshotPath); err != nil {
		return nil, fmt.Errorf("failed to save recovered snapshot: %w", err)
	}

	// 4. Rotate WAL log.
	archiveWalPath := utils.GenRotatedWALPath()
	if archiveWalPath != nil {
		// Rename the old file to the new path (archive it).
		if err := os.Rename(walPath, *archiveWalPath); err != nil {
			if !os.IsNotExist(err) {
				return nil, err
			}
		}
	}

	// Remove the old WAL file if it exists.
	if _, err := os.Stat(walPath); err == nil {
		if err := os.Remove(walPath); err != nil {
			return nil, err
		}
	}

	return pool, nil
}