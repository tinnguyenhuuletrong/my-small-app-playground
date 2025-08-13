package rewardpool

import (
	"testing"

	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/utils"
)

func TestTransactionalDraw(t *testing.T) {
	// Initial setup for the first part of the test
	initialCatalog := []types.PoolReward{
		{ItemID: "gold", Quantity: 1, Probability: 1.0},
	}
	pool := NewPool(initialCatalog)

	ctx := &types.Context{
		WAL:   &utils.MockWAL{},
		Utils: &utils.MockUtils{},
	}

	// SelectItem should stage the item
	item, err := pool.SelectItem(ctx)
	if err != nil {
		t.Fatalf("SelectItem failed: %v", err)
	}
	if item == "" || item != "gold" {
		t.Fatalf("Expected gold, got %v", item)
	}

	// Verify pendingDraws and selector state after SelectItem
	if pool.pendingDraws["gold"] != 1 {
		t.Errorf("Expected pendingDraws[gold] to be 1, got %d", pool.pendingDraws["gold"])
	}
	if pool.selector.GetItemRemaining("gold") != 0 {
		t.Errorf("Expected selector remaining gold to be 0, got %d", pool.selector.GetItemRemaining("gold"))
	}

	// CommitDraw should decrement quantity in catalog and clear pendingDraws
	pool.CommitDraw()
	val := pool.GetItemRemaining("gold")
	if val != 0 {
		t.Fatalf("Expected catalog quantity 0 after commit, got %d", val)
	}
	if pool.pendingDraws["gold"] != 0 {
		t.Errorf("Expected pendingDraws[gold] to be 0 after commit, got %d", pool.pendingDraws["gold"])
	}
	// Selector state should remain 0 for gold as it was already decremented by SelectItem
	if pool.selector.GetItemRemaining("gold") != 0 {
		t.Errorf("Expected selector remaining gold to be 0 after commit, got %d", pool.selector.GetItemRemaining("gold"))
	}

	// Test RevertDraw
	// Define a fresh catalog for this test section
	revertCatalog := []types.PoolReward{
		{ItemID: "gold", Quantity: 1, Probability: 1.0},
	}
	pool = NewPool(revertCatalog) // Reset pool for revert test
	t.Logf("Revert Test: Pool Total Available before SelectItem: %d", pool.selector.TotalAvailable())
	t.Logf("Revert Test: Gold Remaining before SelectItem: %d", pool.selector.GetItemRemaining("gold"))
	item, err = pool.SelectItem(ctx)
	if err != nil {
		t.Fatalf("SelectItem failed for revert test: %v", err)
	}
	if pool.pendingDraws["gold"] != 1 {
		t.Errorf("Revert test: Expected pendingDraws[gold] to be 1, got %d", pool.pendingDraws["gold"])
	}
	if pool.selector.GetItemRemaining("gold") != 0 {
		t.Errorf("Revert test: Expected selector remaining gold to be 0, got %d", pool.selector.GetItemRemaining("gold"))
	}

	pool.RevertDraw()
	if pool.pendingDraws["gold"] != 0 {
		t.Fatalf("Expected pendingDraws[gold] 0 after revert, got %d", pool.pendingDraws["gold"])
	}
	// Selector state should be back to initial quantity after revert
	if pool.selector.GetItemRemaining("gold") != 1 {
		t.Errorf("Expected selector remaining gold to be 1 after revert, got %d", pool.selector.GetItemRemaining("gold"))
	}

	// Test ApplyDrawLog
	// Define a fresh catalog for this test section
	applyDrawLogCatalog := []types.PoolReward{
		{ItemID: "gold", Quantity: 1, Probability: 1.0},
	}
	pool = NewPool(applyDrawLogCatalog) // Reset pool for ApplyDrawLog test
	pool.ApplyDrawLog("gold")
	val = pool.GetItemRemaining("gold")
	if val != 0 {
		t.Fatalf("Expected catalog quantity 0 after ApplyDrawLog, got %d", val)
	}
	if pool.selector.GetItemRemaining("gold") != 0 {
		t.Errorf("Expected selector remaining gold to be 0 after ApplyDrawLog, got %d", pool.selector.GetItemRemaining("gold"))
	}
}
