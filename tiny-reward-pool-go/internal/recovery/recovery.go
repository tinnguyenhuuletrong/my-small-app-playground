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
func RecoverPool(snapshotPath, configPath string, formatter types.LogFormatter, utils types.Utils) (*rewardpool.Pool, uint64, string, error) {
	var pool *rewardpool.Pool
	var lastRequestID uint64

	// 1. Get all WAL files, sorted by sequence number.
	walFiles, err := utils.GetWALFiles()
	if err != nil {
		return nil, 0, "", fmt.Errorf("failed to get WAL files: %w", err)
	}

	// 2. Parse all WAL files to get all log entries.
	var allLogItems []types.WalLogEntry
	for _, walFile := range walFiles {
		entries, _, err := wal.ParseWAL(walFile, formatter)
		if err != nil {
			return nil, 0, "", fmt.Errorf("error parsing WAL file %s: %w", walFile, err)
		}
		allLogItems = append(allLogItems, entries...)
	}

	// 3. Determine the starting point for recovery.
	var snapshotToLoad string
	var logsToReplay []types.WalLogEntry

	// Find the last snapshot in the combined WAL entries.
	lastSnapshotIdx := -1
	for i := len(allLogItems) - 1; i >= 0; i-- {
		if s, ok := allLogItems[i].(*types.WalLogSnapshotItem); ok {
			snapshotToLoad = s.Path
			lastSnapshotIdx = i
			break
		}
	}

	if lastSnapshotIdx != -1 {
		// If a snapshot was found in the WAL, replay logs that came after it.
		logsToReplay = allLogItems[lastSnapshotIdx+1:]
	} else {
		// No snapshot in the WAL, so use the standalone snapshotPath and replay all WAL entries.
		snapshotToLoad = snapshotPath
		logsToReplay = allLogItems
	}

	// 4. Load the initial state from the chosen snapshot.
	pool = rewardpool.NewPool([]types.PoolReward{}) // Create an empty pool.

	// Attempt to load from the determined snapshot path.
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
		// Snapshot doesn't exist, fall back to the initial config file.
		loadedPool, cfgErr := rewardpool.CreatePoolFromConfigPath(configPath)
		if cfgErr != nil {
			return nil, 0, "", fmt.Errorf("failed to load from config after missing snapshot: %w", cfgErr)
		}
		pool = loadedPool
		lastRequestID = 0 // No snapshot, so request ID starts from 0.
	}

	// 5. Replay logs to bring the pool to its most recent state.
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

	var lastWalPath string
	if len(walFiles) > 0 {
		lastWalPath = walFiles[len(walFiles)-1]
	}

	return pool, lastRequestID, lastWalPath, nil
}

// RecoverPoolFromConfig loads the pool state from a snapshot and replays any subsequent WAL entries.
// It returns the recovered pool, the last used request ID, the path of the last WAL file, and any error that occurred.
func RecoverPoolFromConfig(snapshotPath string, initialPool *rewardpool.Pool, formatter types.LogFormatter, utils types.Utils) (*rewardpool.Pool, uint64, string, error) {
	var pool *rewardpool.Pool
	var lastRequestID uint64

	// 1. Get all WAL files, sorted by sequence number.
	walFiles, err := utils.GetWALFiles()
	if err != nil {
		return nil, 0, "", fmt.Errorf("failed to get WAL files: %w", err)
	}

	// 2. Parse all WAL files to get all log entries.
	var allLogItems []types.WalLogEntry
	for _, walFile := range walFiles {
		entries, _, err := wal.ParseWAL(walFile, formatter)
		if err != nil {
			return nil, 0, "", fmt.Errorf("error parsing WAL file %s: %w", walFile, err)
		}
		allLogItems = append(allLogItems, entries...)
	}

	// 3. Determine the starting point for recovery.
	var snapshotToLoad string
	var logsToReplay []types.WalLogEntry

	// Find the last snapshot in the WAL.
	lastSnapshotIdx := -1
	for i := len(allLogItems) - 1; i >= 0; i-- {
		if s, ok := allLogItems[i].(*types.WalLogSnapshotItem); ok {
			snapshotToLoad = s.Path
			lastSnapshotIdx = i
			break
		}
	}

	if lastSnapshotIdx != -1 {
		// If a snapshot was found in the WAL, replay logs that came after it.
		logsToReplay = allLogItems[lastSnapshotIdx+1:]
	} else {
		// No snapshot in the WAL, so use the standalone snapshotPath and replay all WAL entries.
		snapshotToLoad = snapshotPath
		logsToReplay = allLogItems
	}

	// 4. Load the initial state from the chosen snapshot.
	pool = rewardpool.NewPool([]types.PoolReward{}) // Create an empty pool.

	// Attempt to load from the determined snapshot path.
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
		// Snapshot doesn't exist, fall back to the initial config file.
		pool = initialPool
		lastRequestID = 0 // No snapshot, so request ID starts from 0.
	}

	// 5. Replay logs to bring the pool to its most recent state.
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

	var lastWalPath string
	if len(walFiles) > 0 {
		lastWalPath = walFiles[len(walFiles)-1]
	}

	return pool, lastRequestID, lastWalPath, nil
}