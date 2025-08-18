package selector_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/selector"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"
)

func TestItemSelector_UpdateItem(t *testing.T) {
	catalog := []types.PoolReward{
		{ItemID: "item1", Quantity: 10, Probability: 20},
		{ItemID: "item2", Quantity: 5, Probability: 30},
		{ItemID: "item3", Quantity: 0, Probability: 50},
	}

	testCases := []struct {
		name     string
		selector types.ItemSelector
	}{
		{
			name:     "FenwickTreeSelector",
			selector: selector.NewFenwickTreeSelector(),
		},
		{
			name:     "PrefixSumSelector",
			selector: selector.NewPrefixSumSelector(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.selector.Reset(catalog)

			// --- Test Case 1: Update quantity and probability of an existing item ---
		t.Run("Update existing item", func(t *testing.T) {
				tc.selector.UpdateItem("item1", 5, 25)

				assert.Equal(t, 5, tc.selector.GetItemRemaining("item1"))
				// Check total probability
				assert.Equal(t, int64(55), tc.selector.TotalAvailable()) // 25 (item1) + 30 (item2)
			})

			// --- Test Case 2: Update item to have zero quantity ---
		t.Run("Update item to zero quantity", func(t *testing.T) {
				tc.selector.Reset(catalog) // Reset state
				tc.selector.UpdateItem("item2", 0, 40)

				assert.Equal(t, 0, tc.selector.GetItemRemaining("item2"))
				assert.Equal(t, int64(20), tc.selector.TotalAvailable()) // Only item1 is available
			})

			// --- Test Case 3: Update item from zero quantity to be available ---
			t.Run("Update item from zero quantity", func(t *testing.T) {
				tc.selector.Reset(catalog) // Reset state
				tc.selector.UpdateItem("item3", 10, 60)

				assert.Equal(t, 10, tc.selector.GetItemRemaining("item3"))
				assert.Equal(t, int64(110), tc.selector.TotalAvailable()) // 20 (item1) + 30 (item2) + 60 (item3)
			})

			// --- Test Case 4: Update a non-existent item ---
			t.Run("Update non-existent item", func(t *testing.T) {
				tc.selector.Reset(catalog) // Reset state
				tc.selector.UpdateItem("item4", 10, 100)

				assert.Equal(t, -1, tc.selector.GetItemRemaining("item4"))
				assert.Equal(t, int64(50), tc.selector.TotalAvailable()) // Should not have changed
			})
		})
	}
}
