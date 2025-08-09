package selector

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"
)

func TestNewFenwickTreeSelector(t *testing.T) {
	fts := NewFenwickTreeSelector()
	assert.NotNil(t, fts)
	assert.NotNil(t, fts.itemIndex)
	assert.Nil(t, fts.tree)
	assert.Empty(t, fts.itemIDs)
	assert.Zero(t, fts.totalWeight)
}

func TestFenwickTreeSelector_Reset(t *testing.T) {
	fts := NewFenwickTreeSelector()

	// Test with empty catalog
	fts.Reset([]types.PoolReward{})
	assert.NotNil(t, fts.tree)
	assert.Zero(t, fts.tree.Size())
	assert.Empty(t, fts.itemIDs)
	assert.Empty(t, fts.itemIndex)
	assert.Zero(t, fts.totalWeight)

	// Test with a single item
	catalog1 := []types.PoolReward{
		{ItemID: "itemA", Quantity: 10, Probability: 5},
	}
	fts.Reset(catalog1)
	assert.Equal(t, 1, fts.tree.Size())
	assert.Equal(t, []string{"itemA"}, fts.itemIDs)
	assert.Equal(t, 0, fts.itemIndex["itemA"])
	assert.Equal(t, int64(5), fts.totalWeight)
	assert.Equal(t, 10, fts.GetItemRemaining("itemA"))

	// Test with multiple items
	catalog2 := []types.PoolReward{
		{ItemID: "itemA", Quantity: 10, Probability: 10},
		{ItemID: "itemB", Quantity: 20, Probability: 20},
		{ItemID: "itemC", Quantity: 0, Probability: 30}, // Zero quantity, should not be added to weight
	}
	fts.Reset(catalog2)
	assert.Equal(t, 3, fts.tree.Size())
	assert.Equal(t, int64(30), fts.totalWeight) // 10 + 20
	assert.Equal(t, 10, fts.GetItemRemaining("itemA"))
	assert.Equal(t, 20, fts.GetItemRemaining("itemB"))
	assert.Equal(t, 0, fts.GetItemRemaining("itemC"))
}

func TestFenwickTreeSelector_Select(t *testing.T) {
	fts := NewFenwickTreeSelector()
	ctx := &types.Context{}

	_, err := fts.Select(ctx)
	assert.Equal(t, types.ErrEmptyRewardPool, err)

	catalog := []types.PoolReward{
		{ItemID: "itemA", Quantity: 10, Probability: 10},
		{ItemID: "itemB", Quantity: 20, Probability: 20},
		{ItemID: "itemC", Quantity: 30, Probability: 30},
	}
	fts.Reset(catalog)

	// Mock rand source for predictable selections
	// Cumulative probabilities: A: 10, B: 30, C: 60
	mockSource := &MockRandSource{values: []int64{
		0,  // randVal=1 -> selects itemA
		10, // randVal=11 -> selects itemB
		30, // randVal=31 -> selects itemC
	}}
	fts.rand = rand.New(mockSource)

	selected, err := fts.Select(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "itemA", selected)

	selected, err = fts.Select(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "itemB", selected)

	selected, err = fts.Select(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "itemC", selected)

	// Test distribution
	fts.Reset(catalog)
	counts := make(map[string]int)
	numSelections := 60000
	fts.rand = rand.New(rand.NewSource(42))

	for i := 0; i < numSelections; i++ {
		selected, err := fts.Select(ctx)
		assert.NoError(t, err)
		counts[selected]++
	}

	assert.InDelta(t, 10000, counts["itemA"], float64(numSelections)*0.03) // 3% tolerance
	assert.InDelta(t, 20000, counts["itemB"], float64(numSelections)*0.03)
	assert.InDelta(t, 30000, counts["itemC"], float64(numSelections)*0.03)
}

func TestFenwickTreeSelector_Update(t *testing.T) {
	fts := NewFenwickTreeSelector()
	catalog := []types.PoolReward{
		{ItemID: "itemA", Quantity: 1, Probability: 10},
		{ItemID: "itemB", Quantity: 20, Probability: 20},
	}
	fts.Reset(catalog)

	assert.Equal(t, int64(30), fts.TotalAvailable())
	assert.Equal(t, 1, fts.GetItemRemaining("itemA"))

	// Decrease quantity of itemA to 0, should remove its weight
	fts.Update("itemA", -1)
	assert.Equal(t, int64(20), fts.TotalAvailable())
	assert.Equal(t, 0, fts.GetItemRemaining("itemA"))

	// Increase quantity of itemA back to 1, should add its weight back
	fts.Update("itemA", 1)
	assert.Equal(t, int64(30), fts.TotalAvailable())
	assert.Equal(t, 1, fts.GetItemRemaining("itemA"))

	// Update non-existent item, should be ignored
	fts.Update("itemX", 100)
	assert.Equal(t, int64(30), fts.TotalAvailable())
}

func TestFenwickTreeSelector_IntegrationWithDraw(t *testing.T) {
	fts := NewFenwickTreeSelector()
	ctx := &types.Context{}

	catalog := []types.PoolReward{
		{ItemID: "itemA", Quantity: 1, Probability: 100},
		{ItemID: "itemB", Quantity: 1, Probability: 100},
	}
	fts.Reset(catalog)

	assert.Equal(t, int64(200), fts.TotalAvailable())

	// Draw itemA
	fts.rand = rand.New(&MockRandSource{values: []int64{0}}) // Selects itemA
	selected, err := fts.Select(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "itemA", selected)
	fts.Update(selected, -1)

	assert.Equal(t, int64(100), fts.TotalAvailable()) // itemA's weight is removed
	assert.Equal(t, 0, fts.GetItemRemaining("itemA"))
	assert.Equal(t, 1, fts.GetItemRemaining("itemB"))

	// Next draw must be itemB
	fts.rand = rand.New(&MockRandSource{values: []int64{0}}) // Only one item left, any value works
	selected, err = fts.Select(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "itemB", selected)
	fts.Update(selected, -1)

	assert.Equal(t, int64(0), fts.TotalAvailable())
	_, err = fts.Select(ctx)
	assert.Equal(t, types.ErrEmptyRewardPool, err)
}