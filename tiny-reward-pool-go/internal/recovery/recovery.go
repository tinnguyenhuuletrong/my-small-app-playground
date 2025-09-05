package recovery

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/replay"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/rewardpool"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/wal"
)

// RecoverPool loads the pool state from a snapshot and replays any subsequent WAL entries.
// It returns the recovered pool, the last used request ID, the path of the last WAL file, and any error that occurred.
func RecoverPool(configPath string, formatter types.LogFormatter, utils types.Utils) (*rewardpool.Pool, uint64, string, error) {
	var pool *rewardpool.Pool
	var lastRequestID uint64

	// 1. Get all WAL files, sorted by sequence number.
	walFiles, err := utils.GetWALFiles()
	if err != nil {
		return nil, 0, "", fmt.Errorf("failed to get WAL files: %w", err)
	}

	var lastWalPath string
	var logsToReplay []types.WalLogEntry
	var snapshotToLoad string // Initialize to empty

	if len(walFiles) > 0 {
		lastWalPath = walFiles[len(walFiles)-1]

		// Parse only the latest WAL file
		entries, _, err := wal.ParseWAL(lastWalPath, formatter)
		if err != nil {
			return nil, 0, "", fmt.Errorf("error parsing latest WAL file %s: %w", lastWalPath, err)
		}

		if len(entries) == 0 {
			// Latest WAL is empty or only contains header, indicate no WAL to continue from
			lastWalPath = ""
		} else {
			// The first entry must be a snapshot
			snapshotLog, ok := entries[0].(*types.WalLogSnapshotItem)
			if !ok {
				return nil, 0, "", fmt.Errorf("first entry in WAL %s is not a snapshot", lastWalPath)
			}
			snapshotToLoad = snapshotLog.Path
			logsToReplay = entries[1:] // Replay logs after the initial snapshot
		}
	}

	// 2. Load the initial state from the chosen snapshot or config.
	pool = rewardpool.NewPool([]types.PoolReward{}) // Create an empty pool.

	// Attempt to load from the determined snapshot path.
	if snapshotToLoad != "" {
		if _, err := os.Stat(snapshotToLoad); err == nil {
			file, err := os.Open(snapshotToLoad)
			if err != nil {
				return nil, 0, "", fmt.Errorf("failed to open snapshot file %s: %w", snapshotToLoad, err)
			}
			defer file.Close()

			var snap types.PoolSnapshot
			if err := json.NewDecoder(file).Decode(&snap); err != nil {
				return nil, 0, "", fmt.Errorf("failed to decode snapshot %s: %w", snapshotToLoad, err)
			}

			// Load the pool state and the last request ID from the snapshot.
			pool.LoadSnapshot(&snap)
			lastRequestID = snap.LastRequestID

		} else if !os.IsNotExist(err) {
			// Handle other errors from Stat, like permission issues.
			return nil, 0, "", fmt.Errorf("failed to stat snapshot file %s: %w", snapshotToLoad, err)
		} else {
			// Snapshot file not found, fall back to initial config
			loadedPool, cfgErr := rewardpool.CreatePoolFromConfigPath(configPath)
			if cfgErr != nil {
				return nil, 0, "", fmt.Errorf("failed to load from config after missing snapshot: %w", cfgErr)
			}
			pool = loadedPool
			lastRequestID = 0 // No snapshot, so request ID starts from 0.
		}
	} else {
		// No snapshot path from WAL, fall back to initial config
		loadedPool, cfgErr := rewardpool.CreatePoolFromConfigPath(configPath)
		if cfgErr != nil {
			return nil, 0, "", fmt.Errorf("failed to load from config after missing snapshot: %w", cfgErr)
		}
		pool = loadedPool
		lastRequestID = 0 // No snapshot, so request ID starts from 0.
	}

	// 3. Replay logs to bring the pool to its most recent state.
	if len(logsToReplay) > 0 {
		replay.ReplayLogs(pool, logsToReplay)

		// Find the maximum request ID from the replayed draw logs.
		for _, item := range logsToReplay {
			if drawLog, ok := item.(*types.WalLogDrawItem); ok {
				if drawLog.RequestID > lastRequestID {
					lastRequestID = drawLog.RequestID
				}
			}
		}
	}

	if logger := utils.GetLogger(); logger != nil {
		logger.Info(fmt.Sprintf("Recovered state: lastWalPath=%s, lastRequestID=%d, logsReplayed=%d", lastWalPath, lastRequestID, len(logsToReplay)))
	}

	return pool, lastRequestID, lastWalPath, nil
}

// RecoverPoolFromConfig loads the pool state from a snapshot and replays any subsequent WAL entries.
// It returns the recovered pool, the last used request ID, the path of the last WAL file, and any error that occurred.
func RecoverPoolFromConfig(initialPool *rewardpool.Pool, formatter types.LogFormatter, utils types.Utils) (*rewardpool.Pool, uint64, string, error) {
	var pool *rewardpool.Pool
	var lastRequestID uint64

	// 1. Get all WAL files, sorted by sequence number.
	walFiles, err := utils.GetWALFiles()
	if err != nil {
		return nil, 0, "", fmt.Errorf("failed to get WAL files: %w", err)
	}

	var lastWalPath string
	var logsToReplay []types.WalLogEntry
	var snapshotToLoad string // Initialize to empty

	if len(walFiles) > 0 {
		lastWalPath = walFiles[len(walFiles)-1]

		// Parse only the latest WAL file
		entries, _, err := wal.ParseWAL(lastWalPath, formatter)
		if err != nil {
			return nil, 0, "", fmt.Errorf("error parsing latest WAL file %s: %w", lastWalPath, err)
		}

		if len(entries) == 0 {
			// Latest WAL is empty or only contains header, indicate no WAL to continue from
			lastWalPath = ""
		} else {
			// The first entry must be a snapshot
			snapshotLog, ok := entries[0].(*types.WalLogSnapshotItem)
			if !ok {
				return nil, 0, "", fmt.Errorf("first entry in WAL %s is not a snapshot", lastWalPath)
			}
			snapshotToLoad = snapshotLog.Path
			logsToReplay = entries[1:] // Replay logs after the initial snapshot
		}
	}

	// 2. Load the initial state from the chosen snapshot or initialPool.
	pool = initialPool

	// Attempt to load from the determined snapshot path.
	if snapshotToLoad != "" {
		if _, err := os.Stat(snapshotToLoad); err == nil {
			file, err := os.Open(snapshotToLoad)
			if err != nil {
				return nil, 0, "", fmt.Errorf("failed to open snapshot file %s: %w", snapshotToLoad, err)
			}
			defer file.Close()

			var snap types.PoolSnapshot
			if err := json.NewDecoder(file).Decode(&snap); err != nil {
				return nil, 0, "", fmt.Errorf("failed to decode snapshot %s: %w", snapshotToLoad, err)
			}

			// Load the pool state and the last request ID from the snapshot.
			pool.LoadSnapshot(&snap)
			lastRequestID = snap.LastRequestID

		} else if !os.IsNotExist(err) {
			// Handle other errors from Stat, like permission issues.
			return nil, 0, "", fmt.Errorf("failed to stat snapshot file %s: %w", snapshotToLoad, err)
		} else {
			// Snapshot file not found, fall back to initialPool
			pool = initialPool
			lastRequestID = 0 // No snapshot, so request ID starts from 0.
		}
	} else {
		// No snapshot path from WAL, fall back to initialPool
		pool = initialPool
		lastRequestID = 0 // No snapshot, so request ID starts from 0.
	}

	// 3. Replay logs to bring the pool to its most recent state.
	if len(logsToReplay) > 0 {
		replay.ReplayLogs(pool, logsToReplay)

		// Find the maximum request ID from the replayed draw logs.
		for _, item := range logsToReplay {
			if drawLog, ok := item.(*types.WalLogDrawItem); ok {
				if drawLog.RequestID > lastRequestID {
					lastRequestID = drawLog.RequestID
				}
			}
		}
	}

	if logger := utils.GetLogger(); logger != nil {
		logger.Info(fmt.Sprintf("Recovered state: lastWalPath=%s, lastRequestID=%d, logsReplayed=%d", lastWalPath, lastRequestID, len(logsToReplay)))
	}

	return pool, lastRequestID, lastWalPath, nil
}
