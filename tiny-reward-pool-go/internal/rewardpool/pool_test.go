package rewardpool_test

import (
	"testing"

	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/rewardpool"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/utils"
)

func TestTransactionalDraw(t *testing.T) {
	pool := &rewardpool.Pool{
		Catalog: []types.PoolReward{
			{ItemID: "gold", Quantity: 1, Probability: 1.0},
		},
		// pendingDraws will be initialized by Load
	}
	pool.Load(types.ConfigPool{Catalog: pool.Catalog})
	ctx := &types.Context{
		WAL:   &mockWAL{},
		Utils: &utils.UtilsImpl{},
	}
	// SelectItem should stage the item
	item, err := pool.SelectItem(ctx)
	if err != nil {
		t.Fatalf("SelectItem failed: %v", err)
	}
	if item == nil || item.ItemID != "gold" {
		t.Fatalf("Expected gold, got %v", item)
	}
	// CommitDraw should decrement quantity
	pool.CommitDraw("gold")
	if pool.Catalog[0].Quantity != 0 {
		t.Fatalf("Expected quantity 0 after commit, got %d", pool.Catalog[0].Quantity)
	}
	// RevertDraw should not change quantity, but should clear pendingDraws
	pool.Catalog[0].Quantity = 1
	pool.SelectItem(ctx)
	pool.RevertDraw("gold")
	if pool.PendingDraws["gold"] != 0 {
		t.Fatalf("Expected PendingDraws 0 after revert, got %d", pool.PendingDraws["gold"])
	}
}

type mockWAL struct{}

func (m *mockWAL) LogDraw(item types.WalLogItem) error { return nil }
func (m *mockWAL) Close() error                        { return nil }
func (m *mockWAL) Flush() error                        { return nil }
func (m *mockWAL) SetSnapshotPath(path string)         {}
