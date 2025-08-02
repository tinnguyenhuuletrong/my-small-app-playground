package rewardpool_test

import (
	"testing"

	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/rewardpool"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/utils"
)

func TestDraw(t *testing.T) {
	pool := &rewardpool.Pool{
		Catalog: []types.PoolReward{
			{ItemID: "gold", Quantity: 1, Probability: 1.0},
		},
	}
	ctx := &types.Context{
		WAL:   &mockWAL{},
		Utils: &utils.UtilsImpl{},
	}
	item, err := pool.Draw(ctx)
	if err != nil {
		t.Fatalf("Draw failed: %v", err)
	}
	if item == nil || item.ItemID != "gold" {
		t.Fatalf("Expected gold, got %v", item)
	}
}

type mockWAL struct{}

func (m *mockWAL) LogDraw(item types.WalLogItem) error { return nil }
func (m *mockWAL) Close() error                        { return nil }
