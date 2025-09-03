package rewardpool

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/selector"
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

type mockSelectorForUpdate struct {
	selector.FenwickTreeSelector
	updateCalled       bool
	updatedItemID      string
	updatedQuantity    int
	updatedProbability int64
}

func (m *mockSelectorForUpdate) UpdateItem(itemID string, quantity int, probability int64) {
	m.updateCalled = true
	m.updatedItemID = itemID
	m.updatedQuantity = quantity
	m.updatedProbability = probability
}

func TestPool_UpdateItem(t *testing.T) {
	mockSelector := &mockSelectorForUpdate{}
	pool := NewPool([]types.PoolReward{}, PoolOptional{Selector: mockSelector})

	err := pool.UpdateItem("item1", 10, 50)

	require.NoError(t, err)
	assert.True(t, mockSelector.updateCalled)
	assert.Equal(t, "item1", mockSelector.updatedItemID)
	assert.Equal(t, 10, mockSelector.updatedQuantity)
	assert.Equal(t, int64(50), mockSelector.updatedProbability)
}

func TestTransactionalDrawWithUnlimitedQuantity(t *testing.T) {
	// Initial setup for the test
	initialCatalog := []types.PoolReward{
		{ItemID: "unlimited_item", Quantity: types.UnlimitedQuantity, Probability: 1.0},
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
	if item == "" || item != "unlimited_item" {
		t.Fatalf("Expected unlimited_item, got %v", item)
	}

	// Verify pendingDraws and selector state after SelectItem
	if pool.pendingDraws["unlimited_item"] != 1 {
		t.Errorf("Expected pendingDraws[unlimited_item] to be 1, got %d", pool.pendingDraws["unlimited_item"])
	}
	if pool.selector.GetItemRemaining("unlimited_item") != types.UnlimitedQuantity {
		t.Errorf("Expected selector remaining unlimited_item to be %d, got %d", types.UnlimitedQuantity, pool.selector.GetItemRemaining("unlimited_item"))
	}

	// CommitDraw should not change the quantity and clear pendingDraws
	pool.CommitDraw()
	val := pool.GetItemRemaining("unlimited_item")
	if val != types.UnlimitedQuantity {
		t.Fatalf("Expected catalog quantity %d after commit, got %d", types.UnlimitedQuantity, val)
	}
	if pool.pendingDraws["unlimited_item"] != 0 {
		t.Errorf("Expected pendingDraws[unlimited_item] to be 0 after commit, got %d", pool.pendingDraws["unlimited_item"])
	}
	// Selector state should remain unlimited
	if pool.selector.GetItemRemaining("unlimited_item") != types.UnlimitedQuantity {
		t.Errorf("Expected selector remaining unlimited_item to be %d after commit, got %d", types.UnlimitedQuantity, pool.selector.GetItemRemaining("unlimited_item"))
	}

	// Test RevertDraw
	item, err = pool.SelectItem(ctx)
	if err != nil {
		t.Fatalf("SelectItem failed for revert test: %v", err)
	}
	if pool.pendingDraws["unlimited_item"] != 1 {
		t.Errorf("Revert test: Expected pendingDraws[unlimited_item] to be 1, got %d", pool.pendingDraws["unlimited_item"])
	}
	if pool.selector.GetItemRemaining("unlimited_item") != types.UnlimitedQuantity {
		t.Errorf("Revert test: Expected selector remaining unlimited_item to be %d, got %d", types.UnlimitedQuantity, pool.selector.GetItemRemaining("unlimited_item"))
	}

	pool.RevertDraw()
	if pool.pendingDraws["unlimited_item"] != 0 {
		t.Fatalf("Expected pendingDraws[unlimited_item] 0 after revert, got %d", pool.pendingDraws["unlimited_item"])
	}
	// Selector state should be back to initial quantity after revert
	if pool.selector.GetItemRemaining("unlimited_item") != types.UnlimitedQuantity {
		t.Errorf("Expected selector remaining unlimited_item to be %d after revert, got %d", types.UnlimitedQuantity, pool.selector.GetItemRemaining("unlimited_item"))
	}

	// Test ApplyDrawLog
	pool.ApplyDrawLog("unlimited_item")
	val = pool.GetItemRemaining("unlimited_item")
	if val != types.UnlimitedQuantity {
		t.Fatalf("Expected catalog quantity %d after ApplyDrawLog, got %d", types.UnlimitedQuantity, val)
	}
	if pool.selector.GetItemRemaining("unlimited_item") != types.UnlimitedQuantity {
		t.Errorf("Expected selector remaining unlimited_item to be %d after ApplyDrawLog, got %d", types.UnlimitedQuantity, pool.selector.GetItemRemaining("unlimited_item"))
	}
}

func TestPool_CreateSnapshot_WithSHA256(t *testing.T) {
	// Test catalog with items in different orders
	catalog1 := []types.PoolReward{
		{ItemID: "gold", Quantity: 10, Probability: 50},
		{ItemID: "silver", Quantity: 20, Probability: 30},
		{ItemID: "bronze", Quantity: 30, Probability: 20},
	}

	catalog2 := []types.PoolReward{
		{ItemID: "silver", Quantity: 20, Probability: 30},
		{ItemID: "bronze", Quantity: 30, Probability: 20},
		{ItemID: "gold", Quantity: 10, Probability: 50},
	}

	// Create pools with different catalog orders
	pool1 := NewPool(catalog1)
	pool2 := NewPool(catalog2)

	// Create snapshots
	snapshot1, err := pool1.CreateSnapshot()
	require.NoError(t, err)
	require.NotNil(t, snapshot1)

	snapshot2, err := pool2.CreateSnapshot()
	require.NoError(t, err)
	require.NotNil(t, snapshot2)

	// Verify SHA256 fields are present and not empty
	assert.NotEmpty(t, snapshot1.SHA256)
	assert.NotEmpty(t, snapshot2.SHA256)

	// Verify that both snapshots have the same SHA256 hash (deterministic)
	assert.Equal(t, snapshot1.SHA256, snapshot2.SHA256, "SHA256 hashes should be identical for catalogs with same data in different orders")

	// Verify the original catalog order is preserved in the snapshot
	assert.Equal(t, catalog1, snapshot1.Catalog, "Original catalog order should be preserved")
	assert.Equal(t, catalog2, snapshot2.Catalog, "Original catalog order should be preserved")

	// Test that the same catalog always produces the same hash
	snapshot1Again, err := pool1.CreateSnapshot()
	require.NoError(t, err)
	assert.Equal(t, snapshot1.SHA256, snapshot1Again.SHA256, "Same catalog should always produce the same hash")

	// Test with empty catalog
	emptyPool := NewPool([]types.PoolReward{})
	emptySnapshot, err := emptyPool.CreateSnapshot()
	require.NoError(t, err)
	assert.NotEmpty(t, emptySnapshot.SHA256, "Empty catalog should still produce a hash")
	assert.Empty(t, emptySnapshot.Catalog, "Empty catalog should have empty catalog array")

	// Test that different catalogs produce different hashes
	differentCatalog := []types.PoolReward{
		{ItemID: "gold", Quantity: 5, Probability: 50}, // Different quantity
		{ItemID: "silver", Quantity: 20, Probability: 30},
		{ItemID: "bronze", Quantity: 30, Probability: 20},
	}
	differentPool := NewPool(differentCatalog)
	differentSnapshot, err := differentPool.CreateSnapshot()
	require.NoError(t, err)
	assert.NotEqual(t, snapshot1.SHA256, differentSnapshot.SHA256, "Different catalogs should produce different hashes")
}

func TestPool_CreateSnapshot_WithPendingDraws(t *testing.T) {
	catalog := []types.PoolReward{
		{ItemID: "gold", Quantity: 10, Probability: 50},
		{ItemID: "silver", Quantity: 20, Probability: 30},
	}
	pool := NewPool(catalog)

	ctx := &types.Context{
		WAL:   &utils.MockWAL{},
		Utils: &utils.MockUtils{},
	}

	// Select an item to create pending draws
	_, err := pool.SelectItem(ctx)
	require.NoError(t, err)

	// Try to create snapshot with pending draws - should fail
	_, err = pool.CreateSnapshot()
	assert.Error(t, err)
	assert.Equal(t, types.ErrPendingDrawsNotEmpty, err)

	// Commit the draw and try again - should succeed
	pool.CommitDraw()
	snapshot, err := pool.CreateSnapshot()
	require.NoError(t, err)
	assert.NotEmpty(t, snapshot.SHA256)
}
