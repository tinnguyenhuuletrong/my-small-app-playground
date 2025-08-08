package selector

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/utils"
)

// MockRandSource is a mock for rand.Source to control random number generation.
type MockRandSource struct {
	values []int64
	idx    int
}

func (m *MockRandSource) Int63() int64 {
	if m.idx >= len(m.values) {
		panic("not enough mock random values")
	}
	val := m.values[m.idx]
	m.idx++
	return val
}

func (m *MockRandSource) Seed(seed int64) {
	// Do nothing for mock
}

func TestNewFenwickTreeSelector(t *testing.T) {
	fts := NewFenwickTreeSelector()
	assert.NotNil(t, fts)
	assert.NotNil(t, fts.itemIndex)
	assert.Nil(t, fts.tree) // tree should be nil until Reset is called
	assert.Empty(t, fts.itemIDs)
	assert.Zero(t, fts.totalAvailable)
}

func TestFenwickTreeSelector_Reset(t *testing.T) {
	fts := NewFenwickTreeSelector()

	// Test with empty catalog
	fts.Reset([]types.PoolReward{})
	assert.NotNil(t, fts.tree)
	assert.Zero(t, fts.tree.Size())
	assert.Empty(t, fts.itemIDs)
	assert.Empty(t, fts.itemIndex)
	assert.Zero(t, fts.totalAvailable)

	// Test with a single item
	catalog1 := []types.PoolReward{
		{ItemID: "itemA", Quantity: 10},
	}
	fts.Reset(catalog1)
	assert.NotNil(t, fts.tree)
	assert.Equal(t, 1, fts.tree.Size())
	assert.Equal(t, []string{"itemA"}, fts.itemIDs)
	assert.Equal(t, 0, fts.itemIndex["itemA"])
	assert.Equal(t, int64(10), fts.totalAvailable)
	assert.Equal(t, 10, fts.GetItemRemaining("itemA"))

	// Test with multiple items
	catalog2 := []types.PoolReward{
		{ItemID: "itemA", Quantity: 10},
		{ItemID: "itemB", Quantity: 20},
		{ItemID: "itemC", Quantity: 30},
	}
	fts.Reset(catalog2)
	assert.NotNil(t, fts.tree)
	assert.Equal(t, 3, fts.tree.Size())
	assert.Equal(t, []string{"itemA", "itemB", "itemC"}, fts.itemIDs)
	assert.Equal(t, 0, fts.itemIndex["itemA"])
	assert.Equal(t, 1, fts.itemIndex["itemB"])
	assert.Equal(t, 2, fts.itemIndex["itemC"])
	assert.Equal(t, int64(60), fts.totalAvailable)
	assert.Equal(t, 10, fts.GetItemRemaining("itemA"))
	assert.Equal(t, 20, fts.GetItemRemaining("itemB"))
	assert.Equal(t, 30, fts.GetItemRemaining("itemC"))

	// Test resetting with different catalog
	catalog3 := []types.PoolReward{
		{ItemID: "itemX", Quantity: 5},
		{ItemID: "itemY", Quantity: 15},
	}
	fts.Reset(catalog3)
	assert.NotNil(t, fts.tree)
	assert.Equal(t, 2, fts.tree.Size())
	assert.Equal(t, []string{"itemX", "itemY"}, fts.itemIDs)
	assert.Equal(t, 0, fts.itemIndex["itemX"])
	assert.Equal(t, 1, fts.itemIndex["itemY"])
	assert.Equal(t, int64(20), fts.totalAvailable)
	assert.Equal(t, 5, fts.GetItemRemaining("itemX"))
	assert.Equal(t, 15, fts.GetItemRemaining("itemY"))
	assert.Equal(t, -1, fts.GetItemRemaining("itemA")) // Old item should not exist
}

func TestFenwickTreeSelector_Select(t *testing.T) {
	fts := NewFenwickTreeSelector()
	ctx := &types.Context{}

	// Test with empty pool
	_, err := fts.Select(ctx)
	assert.Equal(t, types.ErrEmptyRewardPool, err)

	catalog := []types.PoolReward{
		{ItemID: "itemA", Quantity: 10},
		{ItemID: "itemB", Quantity: 20},
		{ItemID: "itemC", Quantity: 30},
	}
	fts.Reset(catalog)

	// Save original rand.Source and restore after test
	// originalRandSource := rand.Source(rand.NewSource(0)) // Use a dummy source to get the type
	// defer func() {
	// 	rand.Seed(0) // Reset to default behavior
	// 	rand.New(originalRandSource)
	// }()

	// Test specific selections using mock rand source
	mockSource := &MockRandSource{values: []int64{
		0,  // Selects itemA (1-10)
		10, // Selects itemB (11-30)
		30, // Selects itemC (31-60)
	}}
	fts.rand = rand.New(mockSource) // Set the FenwickTreeSelector's rand source

	selected, err := fts.Select(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "itemA", selected)

	selected, err = fts.Select(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "itemB", selected)

	selected, err = fts.Select(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "itemC", selected)

	// Test distribution over many selections
	fts.Reset(catalog) // Reset for distribution test
	counts := make(map[string]int)
	numSelections := 60000 // 1000 * total quantity

	// Use a new rand source for actual random distribution
	fts.rand = rand.New(rand.NewSource(42)) // Fixed seed for reproducibility
	for i := 0; i < numSelections; i++ {
		selected, err := fts.Select(ctx)
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
	fts.Reset(catalog)
	// Temporarily manipulate the tree to cause an invalid index
	fts.tree = utils.NewFenwickTree(0) // Make it empty to force idx out of bounds
	_, err = fts.Select(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "internal error")
}

func TestFenwickTreeSelector_Update(t *testing.T) {
	fts := NewFenwickTreeSelector()
	catalog := []types.PoolReward{
		{ItemID: "itemA", Quantity: 10},
		{ItemID: "itemB", Quantity: 20},
	}
	fts.Reset(catalog)

	assert.Equal(t, int64(30), fts.TotalAvailable())
	assert.Equal(t, 10, fts.GetItemRemaining("itemA"))
	assert.Equal(t, 20, fts.GetItemRemaining("itemB"))

	// Increase quantity of itemA
	fts.Update("itemA", 5)
	assert.Equal(t, int64(35), fts.TotalAvailable())
	assert.Equal(t, 15, fts.GetItemRemaining("itemA"))
	assert.Equal(t, 20, fts.GetItemRemaining("itemB"))

	// Decrease quantity of itemB
	fts.Update("itemB", -10)
	assert.Equal(t, int64(25), fts.TotalAvailable())
	assert.Equal(t, 15, fts.GetItemRemaining("itemA"))
	assert.Equal(t, 10, fts.GetItemRemaining("itemB"))

	// Update an item to zero quantity
	fts.Update("itemA", -15)
	assert.Equal(t, int64(10), fts.TotalAvailable())
	assert.Equal(t, 0, fts.GetItemRemaining("itemA"))
	assert.Equal(t, 10, fts.GetItemRemaining("itemB"))

	// Update an item that doesn't exist (should be ignored)
	fts.Update("itemX", 100)
	assert.Equal(t, int64(10), fts.TotalAvailable()) // Should remain unchanged
	assert.Equal(t, -1, fts.GetItemRemaining("itemX"))
}

func TestFenwickTreeSelector_TotalAvailable(t *testing.T) {
	fts := NewFenwickTreeSelector()
	assert.Zero(t, fts.TotalAvailable())

	catalog := []types.PoolReward{
		{ItemID: "itemA", Quantity: 10},
		{ItemID: "itemB", Quantity: 20},
	}
	fts.Reset(catalog)
	assert.Equal(t, int64(30), fts.TotalAvailable())

	fts.Update("itemA", -5)
	assert.Equal(t, int64(25), fts.TotalAvailable())

	fts.Update("itemC", 100) // Non-existent item
	assert.Equal(t, int64(25), fts.TotalAvailable())
}

func TestFenwickTreeSelector_GetItemRemaining(t *testing.T) {
	fts := NewFenwickTreeSelector()
	catalog := []types.PoolReward{
		{ItemID: "itemA", Quantity: 10},
		{ItemID: "itemB", Quantity: 20},
		{ItemID: "itemC", Quantity: 0}, // Item with zero quantity
	}
	fts.Reset(catalog)

	assert.Equal(t, 10, fts.GetItemRemaining("itemA"))
	assert.Equal(t, 20, fts.GetItemRemaining("itemB"))
	assert.Equal(t, 0, fts.GetItemRemaining("itemC"))
	assert.Equal(t, -1, fts.GetItemRemaining("nonExistentItem"))

	fts.Update("itemA", -5)
	assert.Equal(t, 5, fts.GetItemRemaining("itemA"))

	fts.Update("itemC", 5) // Increase zero quantity item
	assert.Equal(t, 5, fts.GetItemRemaining("itemC"))
}

func TestFenwickTreeSelector_IntegrationWithDraw(t *testing.T) {
	fts := NewFenwickTreeSelector()
	ctx := &types.Context{}

	catalog := []types.PoolReward{
		{ItemID: "itemA", Quantity: 1},
		{ItemID: "itemB", Quantity: 1},
		{ItemID: "itemC", Quantity: 1},
	}
	fts.Reset(catalog)

	assert.Equal(t, int64(3), fts.TotalAvailable())

	// Simulate drawing items and updating quantities
	drawnItems := make(map[string]int)
	for i := 0; i < 3; i++ {
		selected, err := fts.Select(ctx)
		assert.NoError(t, err)
		fts.Update(selected, -1) // Decrement quantity after selection
		drawnItems[selected]++
	}

	assert.Equal(t, int64(0), fts.TotalAvailable())
	assert.Equal(t, 0, fts.GetItemRemaining("itemA"))
	assert.Equal(t, 0, fts.GetItemRemaining("itemB"))
	assert.Equal(t, 0, fts.GetItemRemaining("itemC"))

	// Ensure each item was drawn exactly once
	assert.Equal(t, 1, drawnItems["itemA"])
	assert.Equal(t, 1, drawnItems["itemB"])
	assert.Equal(t, 1, drawnItems["itemC"])

	// After all items are drawn, pool should be empty
	_, err := fts.Select(ctx)
	assert.Equal(t, types.ErrEmptyRewardPool, err)
}
