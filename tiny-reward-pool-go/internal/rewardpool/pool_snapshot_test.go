package rewardpool

import (
	"os"
	"testing"

	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"
)

func TestPoolSnapshotSaveLoad(t *testing.T) {
	pool := &Pool{
		catalog: []types.PoolReward{
			{ItemID: "gold", Quantity: 10, Probability: 1.0},
		},
	}
	snapshot := "test_snapshot.json"
	defer os.Remove(snapshot)
	if err := pool.SaveSnapshot(snapshot); err != nil {
		t.Fatalf("SaveSnapshot failed: %v", err)
	}
	pool.catalog[0].Quantity = 0
	if err := pool.LoadSnapshot(snapshot); err != nil {
		t.Fatalf("LoadSnapshot failed: %v", err)
	}
	if pool.catalog[0].Quantity != 10 {
		t.Fatalf("Expected quantity 10, got %d", pool.catalog[0].Quantity)
	}
}
