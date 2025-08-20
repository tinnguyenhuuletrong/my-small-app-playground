package rewardpool

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"
)

func TestPoolSnapshotSaveLoad(t *testing.T) {
	initialCatalog := []types.PoolReward{
		{ItemID: "gold", Quantity: 10, Probability: 1.0},
	}
	pool := NewPool(initialCatalog)

	snapshotPath := "test_snapshot.json"
	defer os.Remove(snapshotPath)

	snap, err := pool.CreateSnapshot()
	if err != nil {
		t.Fatalf("CreateSnapshot failed: %v", err)
	}

	// Manually save the snapshot to a file
	file, err := os.Create(snapshotPath)
	if err != nil {
		t.Fatalf("Failed to create snapshot file: %v", err)
	}

	if err := json.NewEncoder(file).Encode(snap); err != nil {
		file.Close()
		t.Fatalf("Failed to encode snapshot: %v", err)
	}
	file.Close()

	// Create a new pool to load the snapshot into
	loadedPool := NewPool([]types.PoolReward{})
	if err := loadedPool.LoadSnapshot(snap); err != nil {
		t.Fatalf("LoadSnapshot failed: %v", err)
	}

	val := loadedPool.GetItemRemaining("gold")
	if val != 10 {
		t.Fatalf("Expected quantity 10, got %d", val)
	}
}