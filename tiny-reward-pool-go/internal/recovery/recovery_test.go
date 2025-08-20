package recovery

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/rewardpool"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/utils"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/wal"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/wal/formatter"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/wal/storage"
)

func TestRecoverPool_Basic(t *testing.T) {
	snapshotPath := "../../tmp/test_snapshot.json"
	walPath := "../../tmp/test_wal.log"
	configPath := "../../samples/test_config.json"
	defer os.Remove(snapshotPath)
	defer os.Remove(walPath)
	defer os.Remove(configPath)

	// Create a dummy config
	f, err := os.Create(configPath)
	assert.NoError(t, err)
	_, err = f.WriteString(`{"catalog": [{"item_id": "gold", "quantity": 100, "probability": 50}]}`)
	assert.NoError(t, err)
	f.Close()

	// Create a snapshot
	pool, err := rewardpool.CreatePoolFromConfigPath(configPath)
	assert.NoError(t, err)
	snap, err := pool.CreateSnapshot()
	assert.NoError(t, err)
	snap.LastRequestID = 10
	sf, err := os.Create(snapshotPath)
	assert.NoError(t, err)
	assert.NoError(t, json.NewEncoder(sf).Encode(snap))
	sf.Close()

	// Create a WAL file
	wf, err := os.Create(walPath)
	assert.NoError(t, err)
	encoder := json.NewEncoder(wf)
	encoder.Encode(&types.WalLogSnapshotItem{WalLogEntryBase: types.WalLogEntryBase{Type: types.LogTypeSnapshot}, Path: snapshotPath})
	encoder.Encode(&types.WalLogDrawItem{WalLogEntryBase: types.WalLogEntryBase{Type: types.LogTypeDraw}, RequestID: 11, ItemID: "gold", Success: true})
	encoder.Encode(&types.WalLogDrawItem{WalLogEntryBase: types.WalLogEntryBase{Type: types.LogTypeDraw}, RequestID: 12, ItemID: "gold", Success: true})
	wf.Close()

	jsonFormatter := formatter.NewJSONFormatter()
	recoveredPool, lastRequestID, err := RecoverPool(snapshotPath, walPath, configPath, jsonFormatter, &utils.MockUtils{})
	assert.NoError(t, err)

	assert.Equal(t, uint64(12), lastRequestID)
	assert.Equal(t, 98, recoveredPool.GetItemRemaining("gold"))
}

func TestRecoverPool_MMap(t *testing.T) {
	snapshotPath := "../../tmp/test_snapshot_mmap.json"
	walPath := "../../tmp/test_wal_mmap.log"
	configPath := "../../samples/test_config_mmap.json"
	defer os.Remove(snapshotPath)
	defer os.Remove(walPath)
	defer os.Remove(configPath)

	// Create a dummy config
	f, err := os.Create(configPath)
	assert.NoError(t, err)
	_, err = f.WriteString(`{"catalog": [{"item_id": "gold", "quantity": 100, "probability": 50}]}`)
	assert.NoError(t, err)
	f.Close()

	// Create a snapshot
	pool, err := rewardpool.CreatePoolFromConfigPath(configPath)
	assert.NoError(t, err)
	snap, err := pool.CreateSnapshot()
	assert.NoError(t, err)
	snap.LastRequestID = 20
	sf, err := os.Create(snapshotPath)
	assert.NoError(t, err)
	assert.NoError(t, json.NewEncoder(sf).Encode(snap))
	sf.Close()

	// Write WAL using mmap storage
	jsonFormatter := formatter.NewJSONFormatter()
	mmapStorage, err := storage.NewFileMMapStorage(walPath)
	assert.NoError(t, err)
	w, err := wal.NewWAL(walPath, jsonFormatter, mmapStorage)
	assert.NoError(t, err)

	err = w.LogSnapshot(types.WalLogSnapshotItem{WalLogEntryBase: types.WalLogEntryBase{Type: types.LogTypeSnapshot}, Path: snapshotPath})
	assert.NoError(t, err)
	err = w.LogDraw(types.WalLogDrawItem{WalLogEntryBase: types.WalLogEntryBase{Type: types.LogTypeDraw}, RequestID: 21, ItemID: "gold", Success: true})
	assert.NoError(t, err)
	err = w.LogDraw(types.WalLogDrawItem{WalLogEntryBase: types.WalLogEntryBase{Type: types.LogTypeDraw}, RequestID: 22, ItemID: "gold", Success: true})
	assert.NoError(t, err)
	err = w.Flush()
	assert.NoError(t, err)
	err = w.Close()
	assert.NoError(t, err)

	recoveredPool, lastRequestID, err := RecoverPool(snapshotPath, walPath, configPath, jsonFormatter, &utils.MockUtils{})
	assert.NoError(t, err)

	assert.Equal(t, uint64(22), lastRequestID)
	assert.Equal(t, 98, recoveredPool.GetItemRemaining("gold"))
}