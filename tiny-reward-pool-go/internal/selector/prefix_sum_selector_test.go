package selector

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/utils"
)

func TestNewPrefixSumSelector(t *testing.T) {
	pss := NewPrefixSumSelector()
	assert.NotNil(t, pss)
	assert.NotNil(t, pss.itemIndex)
	assert.NotNil(t, pss.rand)
	assert.Empty(t, pss.itemIDs)
	assert.Empty(t, pss.prefixSums)
	assert.Zero(t, pss.totalWeight)
}

func TestPrefixSumSelector_Reset(t *testing.T) {
	pss := NewPrefixSumSelector()

	// Test with empty catalog
	pss.Reset([]types.PoolReward{})
	assert.Empty(t, pss.itemIDs)
	assert.Empty(t, pss.itemIndex)
	assert.Empty(t, pss.prefixSums)
	assert.Zero(t, pss.totalWeight)

	// Test with a single item
	catalog1 := []types.PoolReward{
		{ItemID: "itemA", Quantity: 10, Probability: 5},
	}
	pss.Reset(catalog1)
	assert.Equal(t, []string{"itemA"}, pss.itemIDs)
	assert.Equal(t, 0, pss.itemIndex["itemA"])
	assert.Equal(t, []int64{5}, pss.prefixSums)
	assert.Equal(t, int64(5), pss.totalWeight)
	assert.Equal(t, 10, pss.GetItemRemaining("itemA"))

	// Test with multiple items
	catalog2 := []types.PoolReward{
		{ItemID: "itemA", Quantity: 10, Probability: 10},
		{ItemID: "itemB", Quantity: 20, Probability: 20},
		{ItemID: "itemC", Quantity: 0, Probability: 30}, // Zero quantity
	}
	pss.Reset(catalog2)
	assert.Equal(t, []string{"itemA", "itemB", "itemC"}, pss.itemIDs)
	assert.Equal(t, []int64{10, 30, 30}, pss.prefixSums) // 10, 10+20, 10+20+0
	assert.Equal(t, int64(30), pss.totalWeight)
	assert.Equal(t, 10, pss.GetItemRemaining("itemA"))
	assert.Equal(t, 20, pss.GetItemRemaining("itemB"))
	assert.Equal(t, 0, pss.GetItemRemaining("itemC"))
}

func TestPrefixSumSelector_Select(t *testing.T) {
	pss := NewPrefixSumSelector()
	ctx := &types.Context{}

	_, err := pss.Select(ctx)
	assert.Equal(t, types.ErrEmptyRewardPool, err)

	catalog := []types.PoolReward{
		{ItemID: "itemA", Quantity: 10, Probability: 10},
		{ItemID: "itemB", Quantity: 20, Probability: 20},
		{ItemID: "itemC", Quantity: 30, Probability: 30},
	}
	pss.Reset(catalog)

	// Mock rand source
	// Cumulative probabilities: A: 10, B: 30, C: 60
	mockSource := &utils.MockRandSource{Values: []int64{
		0,  // randVal=1 -> selects itemA
		10, // randVal=11 -> selects itemB
		30, // randVal=31 -> selects itemC
	}}
	pss.rand = rand.New(mockSource)

	selected, err := pss.Select(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "itemA", selected)

	selected, err = pss.Select(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "itemB", selected)

	selected, err = pss.Select(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "itemC", selected)

	// Test distribution
	pss.Reset(catalog)
	counts := make(map[string]int)
	numSelections := 60000
	pss.rand = rand.New(rand.NewSource(42))

	for i := 0; i < numSelections; i++ {
		selected, err := pss.Select(ctx)
		assert.NoError(t, err)
		counts[selected]++
	}

	assert.InDelta(t, 10000, counts["itemA"], float64(numSelections)*0.03)
	assert.InDelta(t, 20000, counts["itemB"], float64(numSelections)*0.03)
	assert.InDelta(t, 30000, counts["itemC"], float64(numSelections)*0.03)
}

func TestPrefixSumSelector_Update(t *testing.T) {
	pss := NewPrefixSumSelector()
	catalog := []types.PoolReward{
		{ItemID: "itemA", Quantity: 1, Probability: 10},
		{ItemID: "itemB", Quantity: 20, Probability: 20},
	}
	pss.Reset(catalog)

	assert.Equal(t, int64(30), pss.TotalAvailable())
	assert.Equal(t, []int64{10, 30}, pss.prefixSums)

	// Decrease quantity of itemA to 0
	pss.Update("itemA", -1)
	assert.Equal(t, int64(20), pss.TotalAvailable())
	assert.Equal(t, 0, pss.GetItemRemaining("itemA"))
	assert.Equal(t, []int64{0, 20}, pss.prefixSums)

	// Increase quantity of itemA back to 1
	pss.Update("itemA", 1)
	assert.Equal(t, int64(30), pss.TotalAvailable())
	assert.Equal(t, 1, pss.GetItemRemaining("itemA"))
	assert.Equal(t, []int64{10, 30}, pss.prefixSums)
}

func TestPrefixSumSelector_IntegrationWithDraw(t *testing.T) {
	pss := NewPrefixSumSelector()
	ctx := &types.Context{}

	catalog := []types.PoolReward{
		{ItemID: "itemA", Quantity: 1, Probability: 100},
		{ItemID: "itemB", Quantity: 1, Probability: 100},
	}
	pss.Reset(catalog)
	assert.Equal(t, int64(200), pss.TotalAvailable())

	// Draw itemA
	pss.rand = rand.New(&utils.MockRandSource{Values: []int64{0}}) // Selects itemA
	selected, err := pss.Select(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "itemA", selected)
	pss.Update(selected, -1)

	assert.Equal(t, int64(100), pss.TotalAvailable())
	assert.Equal(t, 0, pss.GetItemRemaining("itemA"))
	assert.Equal(t, 1, pss.GetItemRemaining("itemB"))

	// Next draw must be itemB
	pss.rand = rand.New(&utils.MockRandSource{Values: []int64{0}}) // Only one item left, any value works
	selected, err = pss.Select(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "itemB", selected)
	pss.Update(selected, -1)

	assert.Equal(t, int64(0), pss.TotalAvailable())
	_, err = pss.Select(ctx)
	assert.Equal(t, types.ErrEmptyRewardPool, err)
}
