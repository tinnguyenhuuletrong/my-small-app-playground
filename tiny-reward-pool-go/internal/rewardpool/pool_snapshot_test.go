package rewardpool

import (
	"os"
	"testing"

	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"
)

func TestPoolSnapshotSaveLoad(t *testing.T) {
	initialCatalog := []types.PoolReward{
		{ItemID: "gold", Quantity: 10, Probability: 1.0},
	}
	pool := NewPool(initialCatalog)

	snapshot := "test_snapshot.json"
	defer os.Remove(snapshot)

	if err := pool.SaveSnapshot(snapshot); err != nil {
		t.Fatalf("SaveSnapshot failed: %v", err)
	}

	// Create a new pool to load the snapshot into
	loadedPool := NewPool([]types.PoolReward{})
	if err := loadedPool.LoadSnapshot(snapshot); err != nil {
		t.Fatalf("LoadSnapshot failed: %v", err)
	}

	val := loadedPool.GetItemRemaining("gold")
	if val != 10 {
		t.Fatalf("Expected quantity 10, got %d", val)
	}
}
