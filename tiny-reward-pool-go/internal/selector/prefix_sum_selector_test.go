package selector

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"
)

func TestNewPrefixSumSelector(t *testing.T) {
	pss := NewPrefixSumSelector()
	assert.NotNil(t, pss)
	assert.NotNil(t, pss.itemIndex)
	assert.NotNil(t, pss.rand)
	assert.Empty(t, pss.itemIDs)
	assert.Empty(t, pss.prefixSums)
	assert.Zero(t, pss.totalAvailable)
}

func TestPrefixSumSelector_Reset(t *testing.T) {
	pss := NewPrefixSumSelector()

	// Test with empty catalog
	pss.Reset([]types.PoolReward{})
	assert.Empty(t, pss.itemIDs)
	assert.Empty(t, pss.itemIndex)
	assert.Empty(t, pss.prefixSums)
	assert.Zero(t, pss.totalAvailable)

	// Test with a single item
	catalog1 := []types.PoolReward{
		{ItemID: "itemA", Quantity: 10},
	}
	pss.Reset(catalog1)
	assert.Equal(t, []string{"itemA"}, pss.itemIDs)
	assert.Equal(t, 0, pss.itemIndex["itemA"])
	assert.Equal(t, []int64{10}, pss.prefixSums)
	assert.Equal(t, int64(10), pss.totalAvailable)
	assert.Equal(t, 10, pss.GetItemRemaining("itemA"))

	// Test with multiple items
	catalog2 := []types.PoolReward{
		{ItemID: "itemA", Quantity: 10},
		{ItemID: "itemB", Quantity: 20},
		{ItemID: "itemC", Quantity: 30},
	}
	pss.Reset(catalog2)
	assert.Equal(t, []string{"itemA", "itemB", "itemC"}, pss.itemIDs)
	assert.Equal(t, 0, pss.itemIndex["itemA"])
	assert.Equal(t, 1, pss.itemIndex["itemB"])
	assert.Equal(t, 2, pss.itemIndex["itemC"])
	assert.Equal(t, []int64{10, 30, 60}, pss.prefixSums)
	assert.Equal(t, int64(60), pss.totalAvailable)
	assert.Equal(t, 10, pss.GetItemRemaining("itemA"))
	assert.Equal(t, 20, pss.GetItemRemaining("itemB"))
	assert.Equal(t, 30, pss.GetItemRemaining("itemC"))

	// Test resetting with different catalog
	catalog3 := []types.PoolReward{
		{ItemID: "itemX", Quantity: 5},
		{ItemID: "itemY", Quantity: 15},
	}
	pss.Reset(catalog3)
	assert.Equal(t, []string{"itemX", "itemY"}, pss.itemIDs)
	assert.Equal(t, 0, pss.itemIndex["itemX"])
	assert.Equal(t, 1, pss.itemIndex["itemY"])
	assert.Equal(t, []int64{5, 20}, pss.prefixSums)
	assert.Equal(t, int64(20), pss.totalAvailable)
	assert.Equal(t, 5, pss.GetItemRemaining("itemX"))
	assert.Equal(t, 15, pss.GetItemRemaining("itemY"))
	assert.Equal(t, -1, pss.GetItemRemaining("itemA")) // Old item should not exist
}

func TestPrefixSumSelector_Select(t *testing.T) {
	pss := NewPrefixSumSelector()
	ctx := &types.Context{}

	// Test with empty pool
	_, err := pss.Select(ctx)
	assert.Equal(t, types.ErrEmptyRewardPool, err)

	catalog := []types.PoolReward{
		{ItemID: "itemA", Quantity: 10},
		{ItemID: "itemB", Quantity: 20},
		{ItemID: "itemC", Quantity: 30},
	}
	pss.Reset(catalog)

	// Test specific selections using mock rand source
	mockSource := &MockRandSource{values: []int64{
		0,  // Selects itemA (1-10)
		10, // Selects itemB (11-30)
		30, // Selects itemC (31-60)
	}}
	pss.rand = rand.New(mockSource) // Set the PrefixSumSelector's rand source

	selected, err := pss.Select(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "itemA", selected)

	selected, err = pss.Select(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "itemB", selected)

	selected, err = pss.Select(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "itemC", selected)

	// Test distribution over many selections
	pss.Reset(catalog) // Reset for distribution test
	counts := make(map[string]int)
	numSelections := 60000 // 1000 * total quantity

	// Use a new rand source for actual random distribution
	pss.rand = rand.New(rand.NewSource(42)) // Fixed seed for reproducibility
	for i := 0; i < numSelections; i++ {
		selected, err := pss.Select(ctx)
		assert.NoError(t, err)
		counts[selected]++
	}

	// Expected proportions: itemA: 1/6, itemB: 2/6, itemC: 3/6
	// With 60000 selections:
	// itemA: 10000
	// itemB: 20000
	// itemC: 30000
	assert.InDelta(t, 10000, counts["itemA"], float64(numSelections)*0.02) // 2% tolerance
	assert.InDelta(t, 20000, counts["itemB"], float64(numSelections)*0.02)
	assert.InDelta(t, 30000, counts["itemC"], float64(numSelections)*0.02)

	// Test internal error case (should not happen with correct logic)
	pss.Reset(catalog)
	// Temporarily manipulate the prefixSums to cause an invalid index
	pss.prefixSums = []int64{} // Make it empty to force idx out of bounds
	_, err = pss.Select(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "internal error")
}

func TestPrefixSumSelector_Update(t *testing.T) {
	pss := NewPrefixSumSelector()
	catalog := []types.PoolReward{
		{ItemID: "itemA", Quantity: 10},
		{ItemID: "itemB", Quantity: 20},
	}
	pss.Reset(catalog)

	assert.Equal(t, int64(30), pss.TotalAvailable())
	assert.Equal(t, 10, pss.GetItemRemaining("itemA"))
	assert.Equal(t, 20, pss.GetItemRemaining("itemB"))

	// Increase quantity of itemA by 5
	pss.Update("itemA", 5)
	assert.Equal(t, int64(35), pss.TotalAvailable())
	assert.Equal(t, 15, pss.GetItemRemaining("itemA"))
	assert.Equal(t, 20, pss.GetItemRemaining("itemB"))
	assert.Equal(t, []int64{15, 35}, pss.prefixSums)

	// Decrease quantity of itemB by 10
	pss.Update("itemB", -10)
	assert.Equal(t, int64(25), pss.TotalAvailable())
	assert.Equal(t, 15, pss.GetItemRemaining("itemA"))
	assert.Equal(t, 10, pss.GetItemRemaining("itemB"))
	assert.Equal(t, []int64{15, 25}, pss.prefixSums)

	// Decrease quantity of itemA by 15 (to zero)
	pss.Update("itemA", -15)
	assert.Equal(t, int64(10), pss.TotalAvailable())
	assert.Equal(t, 0, pss.GetItemRemaining("itemA"))
	assert.Equal(t, 10, pss.GetItemRemaining("itemB"))
	assert.Equal(t, []int64{0, 10}, pss.prefixSums)

	// Update an item that doesn't exist (should be ignored)
	pss.Update("itemX", 100)
	assert.Equal(t, int64(10), pss.TotalAvailable()) // Should remain unchanged
	assert.Equal(t, -1, pss.GetItemRemaining("itemX"))
}

func TestPrefixSumSelector_TotalAvailable(t *testing.T) {
	pss := NewPrefixSumSelector()
	assert.Zero(t, pss.TotalAvailable())

	catalog := []types.PoolReward{
		{ItemID: "itemA", Quantity: 10},
		{ItemID: "itemB", Quantity: 20},
	}
	pss.Reset(catalog)
	assert.Equal(t, int64(30), pss.TotalAvailable())
	t.Logf("%v %v", pss.itemIndex, pss.prefixSums)

	pss.Update("itemA", -5)
	assert.Equal(t, int64(25), pss.TotalAvailable())
	t.Logf("%v", pss.prefixSums)

	pss.Update("itemC", 100) // Non-existent item
	assert.Equal(t, int64(25), pss.TotalAvailable())
}

func TestPrefixSumSelector_GetItemRemaining(t *testing.T) {
	pss := NewPrefixSumSelector()
	catalog := []types.PoolReward{
		{ItemID: "itemA", Quantity: 10},
		{ItemID: "itemB", Quantity: 20},
		{ItemID: "itemC", Quantity: 0}, // Item with zero quantity
	}
	pss.Reset(catalog)

	assert.Equal(t, 10, pss.GetItemRemaining("itemA"))
	assert.Equal(t, 20, pss.GetItemRemaining("itemB"))
	assert.Equal(t, 0, pss.GetItemRemaining("itemC"))
	assert.Equal(t, -1, pss.GetItemRemaining("nonExistentItem"))

	pss.Update("itemA", -5)
	assert.Equal(t, 5, pss.GetItemRemaining("itemA"))

	pss.Update("itemC", 5) // Increase zero quantity item
	assert.Equal(t, 5, pss.GetItemRemaining("itemC"))
}

func TestPrefixSumSelector_IntegrationWithDraw(t *testing.T) {
	pss := NewPrefixSumSelector()
	ctx := &types.Context{}

	catalog := []types.PoolReward{
		{ItemID: "itemA", Quantity: 1},
		{ItemID: "itemB", Quantity: 1},
		{ItemID: "itemC", Quantity: 1},
	}
	pss.Reset(catalog)

	assert.Equal(t, int64(3), pss.TotalAvailable())

	// Simulate drawing items and updating quantities
	drawnItems := make(map[string]int)
	for i := 0; i < 3; i++ {
		selected, err := pss.Select(ctx)
		assert.NoError(t, err)
		pss.Update(selected, -1) // Decrement quantity after selection
		drawnItems[selected]++
	}

	assert.Equal(t, int64(0), pss.TotalAvailable())
	assert.Equal(t, 0, pss.GetItemRemaining("itemA"))
	assert.Equal(t, 0, pss.GetItemRemaining("itemB"))
	assert.Equal(t, 0, pss.GetItemRemaining("itemC"))

	// Ensure each item was drawn exactly once
	assert.Equal(t, 1, drawnItems["itemA"])
	assert.Equal(t, 1, drawnItems["itemB"])
	assert.Equal(t, 1, drawnItems["itemC"])

	// After all items are drawn, pool should be empty
	_, err := pss.Select(ctx)
	assert.Equal(t, types.ErrEmptyRewardPool, err)
}
