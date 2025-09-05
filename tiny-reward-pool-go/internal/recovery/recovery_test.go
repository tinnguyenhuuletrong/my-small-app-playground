package recovery_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/recovery"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/rewardpool"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/utils"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/wal"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/wal/formatter"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/wal/storage"
)

func setupTestPaths(t *testing.T) (string, string, string, string) {
	tempDir := t.TempDir()
	snapshotPath := filepath.Join(tempDir, "test_snapshot.json")
	walDir := filepath.Join(tempDir, "wal")
	walPath := filepath.Join(walDir, "wal.000")
	configPath := filepath.Join(tempDir, "test_config.json")

	require.NoError(t, os.MkdirAll(walDir, 0755))

	// Create a dummy config
	f, err := os.Create(configPath)
	require.NoError(t, err)
	_, err = f.WriteString(`{"catalog": [{"item_id": "gold", "quantity": 100, "probability": 50}]}`)
	require.NoError(t, err)
	f.Close()

	return snapshotPath, walPath, configPath, walDir
}

func TestRecoverPool_Basic(t *testing.T) {
	snapshotPath, walPath, configPath, walDir := setupTestPaths(t)

	// Create a valid WAL file using the WAL writer
	w, err := wal.NewWAL(walPath, 0, formatter.NewJSONFormatter(), nil)
	require.NoError(t, err)

	// Create a snapshot and log it
	pool, err := rewardpool.CreatePoolFromConfigPath(configPath)
	require.NoError(t, err)
	snap, err := pool.CreateSnapshot()
	require.NoError(t, err)
	snap.LastRequestID = 10
	// Manually create snapshot file for the log
	sf, err := os.Create(snapshotPath)
	require.NoError(t, err)
	require.NoError(t, json.NewEncoder(sf).Encode(snap))
	sf.Close()

	require.NoError(t, w.LogSnapshot(types.WalLogSnapshotItem{WalLogEntryBase: types.WalLogEntryBase{Type: types.LogTypeSnapshot}, Path: snapshotPath}))
	require.NoError(t, w.LogDraw(types.WalLogDrawItem{WalLogEntryBase: types.WalLogEntryBase{Type: types.LogTypeDraw}, RequestID: 11, ItemID: "gold", Success: true}))
	require.NoError(t, w.LogDraw(types.WalLogDrawItem{WalLogEntryBase: types.WalLogEntryBase{Type: types.LogTypeDraw}, RequestID: 12, ItemID: "gold", Success: true}))

	require.NoError(t, w.Flush())
	require.NoError(t, w.Close())

	// Now, recover
	jsonFormatter := formatter.NewJSONFormatter()
	recoveredPool, lastRequestID, _, err := recovery.RecoverPool(snapshotPath, configPath, jsonFormatter, utils.NewDefaultUtils(walDir, "", 0, nil))
	require.NoError(t, err)

	assert.Equal(t, uint64(12), lastRequestID)
	assert.Equal(t, 98, recoveredPool.GetItemRemaining("gold"))
}

func TestRecoverPool_MMap(t *testing.T) {
	snapshotPath, walPath, configPath, walDir := setupTestPaths(t)

	// Write WAL using mmap storage
	jsonFormatter := formatter.NewJSONFormatter()
	mmapStorage, err := storage.NewFileMMapStorage(walPath, 0)
	require.NoError(t, err)
	w, err := wal.NewWAL(walPath, 0, jsonFormatter, mmapStorage)
	require.NoError(t, err)

	// Create a snapshot and log it
	pool, err := rewardpool.CreatePoolFromConfigPath(configPath)
	require.NoError(t, err)
	snap, err := pool.CreateSnapshot()
	require.NoError(t, err)
	snap.LastRequestID = 20
	sf, err := os.Create(snapshotPath)
	require.NoError(t, err)
	require.NoError(t, json.NewEncoder(sf).Encode(snap))
	sf.Close()

	require.NoError(t, w.LogSnapshot(types.WalLogSnapshotItem{WalLogEntryBase: types.WalLogEntryBase{Type: types.LogTypeSnapshot}, Path: snapshotPath}))
	require.NoError(t, w.LogDraw(types.WalLogDrawItem{WalLogEntryBase: types.WalLogEntryBase{Type: types.LogTypeDraw}, RequestID: 21, ItemID: "gold", Success: true}))
	require.NoError(t, w.LogDraw(types.WalLogDrawItem{WalLogEntryBase: types.WalLogEntryBase{Type: types.LogTypeDraw}, RequestID: 22, ItemID: "gold", Success: true}))

	require.NoError(t, w.Flush())
	require.NoError(t, w.Close())

	// Now, recover
	recoveredPool, lastRequestID, _, err := recovery.RecoverPool(snapshotPath, configPath, jsonFormatter, utils.NewDefaultUtils(walDir, "", 0, nil))
	require.NoError(t, err)

	assert.Equal(t, uint64(22), lastRequestID)
	assert.Equal(t, 98, recoveredPool.GetItemRemaining("gold"))
}