package recovery

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"

	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/replay"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/rewardpool"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/wal"
)

// RecoverPool loads the pool state from a snapshot and replays any subsequent WAL entries.
// It returns the recovered pool and the last used request ID.
func RecoverPool(snapshotPath, walPath, configPath string, formatter types.LogFormatter, utils types.Utils) (*rewardpool.Pool, uint64, error) {
	var pool *rewardpool.Pool
	var lastRequestID uint64

	// 1. Unroll the WAL chain to get all log entries.
	allLogItems, processedWalFiles, err := unrollWALChain(walPath, formatter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to unroll WAL chain: %w", err)
	}

	// 2. Determine the starting point for recovery.
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

	// 3. Load the initial state from the chosen snapshot.
	pool = rewardpool.NewPool([]types.PoolReward{}) // Create an empty pool.

	// Attempt to load from the determined snapshot path.
	if _, err := os.Stat(snapshotToLoad); err == nil {
		file, err := os.Open(snapshotToLoad)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to open snapshot file %s: %w", snapshotToLoad, err)
		}
		defer file.Close()

		var snap types.PoolSnapshot
		if err := json.NewDecoder(file).Decode(&snap); err != nil {
			return nil, 0, fmt.Errorf("failed to decode snapshot %s: %w", snapshotToLoad, err)
		}

		// Load the pool state and the last request ID from the snapshot.
		pool.LoadSnapshot(&snap)
		lastRequestID = snap.LastRequestID

	} else if !os.IsNotExist(err) {
		// Handle other errors from Stat, like permission issues.
		return nil, 0, fmt.Errorf("failed to stat snapshot file %s: %w", snapshotToLoad, err)
	} else {
		// Snapshot doesn't exist, fall back to the initial config file.
		loadedPool, cfgErr := rewardpool.CreatePoolFromConfigPath(configPath)
		if cfgErr != nil {
			return nil, 0, fmt.Errorf("failed to load from config after missing snapshot: %w", cfgErr)
		}
		pool = loadedPool
		lastRequestID = 0 // No snapshot, so request ID starts from 0.
	}

	// 4. Replay logs to bring the pool to its most recent state.
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

	// 5. Clean up old WAL files.
	for _, path := range processedWalFiles {
		if _, err := os.Stat(path); err == nil {
			if err := os.Remove(path); err != nil {
				// Log this error but don't fail the recovery.
				if utils.GetLogger() != nil {
					utils.GetLogger().Error("failed to remove old WAL file", "path", path, "error", err)
				}
			}
		}
	}

	return pool, lastRequestID, nil
}

// RecoverPoolFromConfig loads the pool state from a snapshot and replays any subsequent WAL entries.
// It returns the recovered pool and the last used request ID.
func RecoverPoolFromConfig(snapshotPath, walPath string, initialPool *rewardpool.Pool, formatter types.LogFormatter, utils types.Utils) (*rewardpool.Pool, uint64, error) {
	var pool *rewardpool.Pool
	var lastRequestID uint64

	// 1. Unroll the WAL chain to get all log entries.
	allLogItems, processedWalFiles, err := unrollWALChain(walPath, formatter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to unroll WAL chain: %w", err)
	}

	// 2. Determine the starting point for recovery.
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

	// 3. Load the initial state from the chosen snapshot.
	pool = rewardpool.NewPool([]types.PoolReward{}) // Create an empty pool.

	// Attempt to load from the determined snapshot path.
	if _, err := os.Stat(snapshotToLoad); err == nil {
		file, err := os.Open(snapshotToLoad)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to open snapshot file %s: %w", snapshotToLoad, err)
		}
		defer file.Close()

		var snap types.PoolSnapshot
		if err := json.NewDecoder(file).Decode(&snap); err != nil {
			return nil, 0, fmt.Errorf("failed to decode snapshot %s: %w", snapshotToLoad, err)
		}

		// Load the pool state and the last request ID from the snapshot.
		pool.LoadSnapshot(&snap)
		lastRequestID = snap.LastRequestID

	} else if !os.IsNotExist(err) {
		// Handle other errors from Stat, like permission issues.
		return nil, 0, fmt.Errorf("failed to stat snapshot file %s: %w", snapshotToLoad, err)
	} else {
		// Snapshot doesn't exist, fall back to the initial config file.
		pool = initialPool
		lastRequestID = 0 // No snapshot, so request ID starts from 0.
	}

	// 4. Replay logs to bring the pool to its most recent state.
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

	// 5. Clean up old WAL files.
	for _, path := range processedWalFiles {
		if _, err := os.Stat(path); err == nil {
			if err := os.Remove(path); err != nil {
				// Log this error but don't fail the recovery.
				if utils.GetLogger() != nil {
					utils.GetLogger().Error("failed to remove old WAL file", "path", path, "error", err)
				}
			}
		}
	}

	return pool, lastRequestID, nil
}

// unrollWALChain reads a chain of WAL files starting from startPath and returns all log entries.
func unrollWALChain(startPath string, formatter types.LogFormatter) ([]types.WalLogEntry, []string, error) {
	var allEntries []types.WalLogEntry
	var processedFiles []string
	walQueue := []string{startPath}

	for len(walQueue) > 0 {
		currentPath := walQueue[0]
		walQueue = walQueue[1:]

		// Avoid processing the same file twice in case of a loop (should not happen in normal operation)
		isProcessed := false
		for _, p := range processedFiles {
			if p == currentPath {
				isProcessed = true
				break
			}
		}
		if isProcessed {
			continue
		}

		entries, hdr, err := wal.ParseWAL(currentPath, formatter)
		if err != nil {
			if os.IsNotExist(err) {
				continue // A file in the chain might not exist, which is acceptable.
			}
			return nil, nil, fmt.Errorf("error parsing WAL file %s: %w", currentPath, err)
		}

		if entries != nil {
			allEntries = append(allEntries, entries...)
		}
		processedFiles = append(processedFiles, currentPath)

		if hdr != nil && hdr.Status == types.WALStatusClosed {
			nextPathBytes := bytes.Trim(hdr.NextWALPath[:], "\x00")
			if len(nextPathBytes) > 0 {
				walQueue = append(walQueue, string(nextPathBytes))
			}
		}
	}

	return allEntries, processedFiles, nil
}
