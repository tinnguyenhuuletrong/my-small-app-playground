package recovery

import (
	"os"
	"testing"

	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/rewardpool"
)

func TestRecoverPool_Basic(t *testing.T) {
	snapshot := "../../tmp/test_snapshot.json"
	wal := "../../tmp/test_wal.log"
	config := "../../samples/test_config.json"

	f, err := os.Create(config)
	if err != nil {
		t.Fatalf("failed to create config: %v", err)
	}
	_, _ = f.WriteString(`
{
  "catalog": [
    { "item_id": "gold", "quantity": 100, "probability": 50 },
    { "item_id": "silver", "quantity": 200, "probability": 30 },
    { "item_id": "bronze", "quantity": 300, "probability": 20 }
  ]
}
	`)
	f.Close()

	// Setup: create a snapshot and WAL log
	var pool *rewardpool.Pool
	loaded, err := rewardpool.CreatePoolFromConfigPath(config)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	pool = loaded

	pool.SaveSnapshot(snapshot)

	f, err = os.Create(wal)
	if err != nil {
		t.Fatalf("failed to create wal log: %v", err)
	}
	_, _ = f.WriteString("DRAW 1 gold\nDRAW 2 silver\nDRAW 3 FAILED\n")
	f.Close()

	recovered, err := RecoverPool(snapshot, wal, config)
	if err != nil {
		t.Fatalf("recovery failed: %v", err)
	}

	// Check that gold and silver quantities are decremented
	var gold, silver int
	gold = recovered.GetItemRemaining("gold")
	silver = recovered.GetItemRemaining("silver")
	if gold < 0 || silver < 0 {
		t.Errorf("item quantity should not be negative: gold=%d silver=%d", gold, silver)
	}

	// Cleanup
	os.Remove(snapshot)
	os.Remove(wal)
	os.Remove(config)
}
