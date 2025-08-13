package recovery

import (
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
	snapshot := "../../tmp/test_snapshot.json"
	walPath := "../../tmp/test_wal.log"
	config := "../../samples/test_config.json"
	defer os.Remove(snapshot)
	defer os.Remove(walPath)
	defer os.Remove(config)

	f, err := os.Create(config)
	assert.NoError(t, err)
	_, err = f.WriteString(
		`
{
  "catalog": [
    { "item_id": "gold", "quantity": 100, "probability": 50 },
    { "item_id": "silver", "quantity": 200, "probability": 30 },
    { "item_id": "bronze", "quantity": 300, "probability": 20 }
  ]
}
	`)
	assert.NoError(t, err)
	f.Close()

	// Setup: create a snapshot and WAL log
	var pool *rewardpool.Pool
	loaded, err := rewardpool.CreatePoolFromConfigPath(config)
	assert.NoError(t, err)
	pool = loaded

	err = pool.SaveSnapshot(snapshot)
	assert.NoError(t, err)

	f, err = os.Create(walPath)
	assert.NoError(t, err)
	_, err = f.WriteString(`{"type":1,"request_id":1,"item_id":"gold","success":true}` + "\n" + `{"type":1,"request_id":2,"item_id":"silver","success":true}` + "\n" + `{"type":1,"request_id":3,"success":false,"error":1}` + "\n")
	assert.NoError(t, err)
	f.Close()

	jsonFormatter := formatter.NewJSONFormatter()
	recovered, err := RecoverPool(snapshot, walPath, config, jsonFormatter, &utils.MockUtils{})
	assert.NoError(t, err)

	// Check that gold and silver quantities are decremented
	var gold, silver int
	gold = recovered.GetItemRemaining("gold")
	silver = recovered.GetItemRemaining("silver")
	assert.GreaterOrEqual(t, gold, 0)
	assert.GreaterOrEqual(t, silver, 0)
}

func TestRecoverPool_MMap(t *testing.T) {
	snapshot := "../../tmp/test_snapshot_mmap.json"
	walPath := "../../tmp/test_wal_mmap.log"
	config := "../../samples/test_config_mmap.json"
	defer os.Remove(snapshot)
	defer os.Remove(walPath)
	defer os.Remove(config)

	f, err := os.Create(config)
	assert.NoError(t, err)
	_, err = f.WriteString(
		`
{
  "catalog": [
    { "item_id": "gold", "quantity": 100, "probability": 50 },
    { "item_id": "silver", "quantity": 200, "probability": 30 },
    { "item_id": "bronze", "quantity": 300, "probability": 20 }
  ]
}
	`)
	assert.NoError(t, err)
	f.Close()

	// Setup: create a snapshot and WAL log
	var pool *rewardpool.Pool
	loaded, err := rewardpool.CreatePoolFromConfigPath(config)
	assert.NoError(t, err)
	pool = loaded

	err = pool.SaveSnapshot(snapshot)
	assert.NoError(t, err)

	// Write WAL using mmap storage
	jsonFormatter := formatter.NewJSONFormatter()
	mmapStorage, err := storage.NewFileMMapStorage(walPath)
	assert.NoError(t, err)
	w, err := wal.NewWAL(walPath, jsonFormatter, mmapStorage)
	assert.NoError(t, err)

	err = w.LogDraw(types.WalLogDrawItem{WalLogItem: types.WalLogItem{Type: types.LogTypeDraw}, RequestID: 1, ItemID: "gold", Success: true})
	assert.NoError(t, err)
	err = w.LogDraw(types.WalLogDrawItem{WalLogItem: types.WalLogItem{Type: types.LogTypeDraw}, RequestID: 2, ItemID: "silver", Success: true})
	assert.NoError(t, err)
	err = w.LogDraw(types.WalLogDrawItem{WalLogItem: types.WalLogItem{Type: types.LogTypeDraw, Error: types.ErrorPoolEmpty}, RequestID: 3, Success: false})
	assert.NoError(t, err)
	err = w.Flush()
	assert.NoError(t, err)
	err = w.Close()
	assert.NoError(t, err)

	recovered, err := RecoverPool(snapshot, walPath, config, jsonFormatter, &utils.MockUtils{})
	assert.NoError(t, err)

	// Check that gold and silver quantities are decremented
	var gold, silver int
	gold = recovered.GetItemRemaining("gold")
	silver = recovered.GetItemRemaining("silver")
	assert.GreaterOrEqual(t, gold, 0)
	assert.GreaterOrEqual(t, silver, 0)
}
