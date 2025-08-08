package selector

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"
)

// PrefixSumSelector implements the ItemSelector interface using a prefix sum array.
type PrefixSumSelector struct {
	// prefixSums stores the cumulative sums of item quantities.
	prefixSums []int64

	// itemIDs maps the index in the prefixSums array back to the actual ItemID.
	itemIDs []string

	// itemIndex maps ItemID to its index in the prefixSums and itemIDs slices.
	itemIndex map[string]int

	// totalAvailable stores the sum of all quantities in the selector.
	totalAvailable int64

	// rand is the random number generator for selection.
	rand *rand.Rand
}

var _ types.ItemSelector = (*PrefixSumSelector)(nil)

// NewPrefixSumSelector creates a new PrefixSumSelector.
func NewPrefixSumSelector() *PrefixSumSelector {
	return &PrefixSumSelector{
		itemIndex: make(map[string]int),
		rand:      rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Reset initializes or re-initializes the selector with a new catalog.
func (pss *PrefixSumSelector) Reset(catalog []types.PoolReward) {
	// Clear existing data
	pss.itemIDs = make([]string, len(catalog))
	pss.itemIndex = make(map[string]int)
	pss.prefixSums = make([]int64, len(catalog))
	pss.totalAvailable = 0

	// Populate the prefix sums, itemIDs, and itemIndex maps
	var currentSum int64
	for i, item := range catalog {
		currentSum += int64(item.Quantity)
		pss.prefixSums[i] = currentSum
		pss.itemIDs[i] = item.ItemID
		pss.itemIndex[item.ItemID] = i
	}
	pss.totalAvailable = currentSum
}

// Select chooses an item based on its availability.
func (pss *PrefixSumSelector) Select(ctx *types.Context) (string, error) {
	if pss.totalAvailable <= 0 {
		return "", types.ErrEmptyRewardPool
	}

	// Generate a random value within the total available range
	randVal := pss.rand.Int63n(pss.totalAvailable) + 1 // +1 because we're looking for a value >= 1

	// Find the index of the item using binary search on prefixSums
	idx := pss.findItemIndex(randVal)

	// This should ideally not happen if totalAvailable is correct and findItemIndex works as expected
	if idx == -1 || idx >= len(pss.itemIDs) {
		return "", fmt.Errorf("internal error: failed to find item for random value %d (total available: %d)", randVal, pss.totalAvailable)
	}

	selectedItemID := pss.itemIDs[idx]

	return selectedItemID, nil
}

// findItemIndex performs a binary search to find the index of the item corresponding to the given value.
func (pss *PrefixSumSelector) findItemIndex(value int64) int {
	low := 0
	high := len(pss.prefixSums) - 1
	resultIdx := -1

	for low <= high {
		mid := low + (high-low)/2
		if pss.prefixSums[mid] >= value {
			resultIdx = mid
			high = mid - 1
		} else {
			low = mid + 1
		}
	}
	return resultIdx
}

// Update adjusts the quantity of a specific item in the selector.
func (pss *PrefixSumSelector) Update(itemID string, delta int64) {
	idx, ok := pss.itemIndex[itemID]
	if !ok {
		// Item not found in the selector, ignore.
		return
	}

	// The 'delta' parameter is the delta to apply.
	change := delta

	// Update prefix sums from the current item's index onwards
	for i := idx; i < len(pss.prefixSums); i++ {
		pss.prefixSums[i] += change
	}

	pss.totalAvailable += change
}

// TotalAvailable returns the total count of all items currently available for selection.
func (pss *PrefixSumSelector) TotalAvailable() int64 {
	return pss.totalAvailable
}

// GetItemRemaining returns the remaining quantity of a specific item.
func (pss *PrefixSumSelector) GetItemRemaining(itemID string) int {
	idx, ok := pss.itemIndex[itemID]
	if !ok {
		return -1 // Item not found
	}

	if idx == 0 {
		return int(pss.prefixSums[idx])
	}
	return int(pss.prefixSums[idx] - pss.prefixSums[idx-1])
}
