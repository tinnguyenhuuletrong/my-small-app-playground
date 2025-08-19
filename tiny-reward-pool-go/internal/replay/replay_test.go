package replay_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/replay"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/rewardpool"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"
)

func TestReplayLogsWithRealPool(t *testing.T) {
	initialCatalog := []types.PoolReward{
		{ItemID: "item1", Quantity: 10, Probability: 100},
		{ItemID: "item2", Quantity: 5, Probability: 50},
		{ItemID: "item3", Quantity: 1, Probability: 20},
	}

	pool := rewardpool.NewPool(initialCatalog)

	logs := []types.WalLogEntry{
		// Successful draw of item1
		&types.WalLogDrawItem{
			WalLogEntryBase: types.WalLogEntryBase{Type: types.LogTypeDraw},
			Success:         true,
			ItemID:          "item1",
		},
		// Update item2
		&types.WalLogUpdateItem{
			WalLogEntryBase: types.WalLogEntryBase{Type: types.LogTypeUpdate},
			ItemID:          "item2",
			Quantity:        20,
			Probability:     200,
		},
		// Unsuccessful draw of item3 (should have no effect)
		&types.WalLogDrawItem{
			WalLogEntryBase: types.WalLogEntryBase{Type: types.LogTypeDraw},
			Success:         false,
			ItemID:          "item3",
		},
		// Another successful draw of item1
		&types.WalLogDrawItem{
			WalLogEntryBase: types.WalLogEntryBase{Type: types.LogTypeDraw},
			Success:         true,
			ItemID:          "item1",
		},
		// A snapshot log (should have no effect on state)
		&types.WalLogSnapshotItem{
			WalLogEntryBase: types.WalLogEntryBase{Type: types.LogTypeSnapshot},
			Path:            "/some/path",
		},
	}

	replay.ReplayLogs(pool, logs)

	state := pool.State()

	// Create a map for easy lookup
	stateMap := make(map[string]types.PoolReward)
	for _, item := range state {
		stateMap[item.ItemID] = item
	}

	// Check item1 state: initial 10, drawn twice
	assert.Equal(t, 8, stateMap["item1"].Quantity)
	assert.Equal(t, int64(100), stateMap["item1"].Probability)

	// Check item2 state: updated
	assert.Equal(t, 20, stateMap["item2"].Quantity)
	assert.Equal(t, int64(200), stateMap["item2"].Probability)

	// Check item3 state: unchanged
	assert.Equal(t, 1, stateMap["item3"].Quantity)
	assert.Equal(t, int64(20), stateMap["item3"].Probability)
}

func TestApplyLogWithRealPool(t *testing.T) {
	initialCatalog := []types.PoolReward{
		{ItemID: "item1", Quantity: 1, Probability: 100},
	}
	pool := rewardpool.NewPool(initialCatalog)

	// Apply a draw log
	drawLog := &types.WalLogDrawItem{
		WalLogEntryBase: types.WalLogEntryBase{Type: types.LogTypeDraw},
		Success:         true,
		ItemID:          "item1",
	}
	replay.ApplyLog(pool, drawLog)
	assert.Equal(t, 0, pool.State()[0].Quantity)

	// Apply an update log
	updateLog := &types.WalLogUpdateItem{
		WalLogEntryBase: types.WalLogEntryBase{Type: types.LogTypeUpdate},
		ItemID:          "item1",
		Quantity:        5,
		Probability:     50,
	}
	replay.ApplyLog(pool, updateLog)
	state := pool.State()
	assert.Equal(t, 5, state[0].Quantity)
	assert.Equal(t, int64(50), state[0].Probability)
}