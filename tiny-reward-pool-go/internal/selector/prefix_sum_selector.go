package selector

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"
)

// PrefixSumSelector implements the ItemSelector interface using a prefix sum array.
type PrefixSumSelector struct {
	// prefixSums stores the cumulative sums of item probabilities.
	prefixSums []int64

	// items stores the original reward data.
	items []types.PoolReward

	// itemIDs maps the index in the prefixSums array back to the actual ItemID.
	itemIDs []string

	// itemIndex maps ItemID to its index in the prefixSums and itemIDs slices.
	itemIndex map[string]int

	// itemInfo tracks the current state (quantity) of each item.
	itemInfo map[string]*types.PoolReward

	// totalWeight stores the sum of all probabilities in the selector.
	totalWeight int64

	// rand is the random number generator for selection.
	rand *rand.Rand
}

var _ types.ItemSelector = (*PrefixSumSelector)(nil)

// NewPrefixSumSelector creates a new PrefixSumSelector.
func NewPrefixSumSelector() *PrefixSumSelector {
	return &PrefixSumSelector{
		itemIndex: make(map[string]int),
		itemInfo:  make(map[string]*types.PoolReward),
		rand:      rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Reset initializes or re-initializes the selector with a new catalog.
func (pss *PrefixSumSelector) Reset(catalog []types.PoolReward) {
	pss.items = make([]types.PoolReward, len(catalog))
	pss.itemIDs = make([]string, len(catalog))
	pss.itemIndex = make(map[string]int)
	pss.itemInfo = make(map[string]*types.PoolReward, len(catalog))
	pss.prefixSums = make([]int64, len(catalog))
	pss.totalWeight = 0

	var currentWeight int64
	for i, item := range catalog {
		itemCopy := item
		pss.items[i] = itemCopy
		pss.itemIDs[i] = item.ItemID
		pss.itemIndex[item.ItemID] = i
		pss.itemInfo[item.ItemID] = &pss.items[i]

		if item.Quantity > 0 {
			currentWeight += item.Probability
		}
		pss.prefixSums[i] = currentWeight
	}
	pss.totalWeight = currentWeight
}

// Select chooses an item based on its availability.
func (pss *PrefixSumSelector) Select(ctx *types.Context) (string, error) {
	if pss.totalWeight <= 0 {
		return "", types.ErrEmptyRewardPool
	}

	randVal := pss.rand.Int63n(pss.totalWeight) + 1

	idx := pss.findItemIndex(randVal)

	if idx == -1 || idx >= len(pss.itemIDs) {
		return "", fmt.Errorf("internal error: failed to find item for random value %d (total weight: %d)", randVal, pss.totalWeight)
	}

	selectedItemID := pss.itemIDs[idx]
	if pss.itemInfo[selectedItemID].Quantity <= 0 {
		return "", fmt.Errorf("internal error: selected item %s has zero quantity", selectedItemID)
	}

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
		return
	}

	item := pss.itemInfo[itemID]
	oldQuantity := item.Quantity
	newQuantity := oldQuantity + int(delta)
	item.Quantity = newQuantity

	var weightChange int64
	if oldQuantity > 0 && newQuantity <= 0 {
		weightChange = -item.Probability
	} else if oldQuantity <= 0 && newQuantity > 0 {
		weightChange = item.Probability
	}

	if weightChange != 0 {
		for i := idx; i < len(pss.prefixSums); i++ {
			pss.prefixSums[i] += weightChange
		}
		pss.totalWeight += weightChange
	}
}

// TotalAvailable returns the total weight of all items currently available for selection.
func (pss *PrefixSumSelector) TotalAvailable() int64 {
	return pss.totalWeight
}

// GetItemRemaining returns the remaining quantity of a specific item.
func (pss *PrefixSumSelector) GetItemRemaining(itemID string) int {
	if item, ok := pss.itemInfo[itemID]; ok {
		return item.Quantity
	}
	return -1 // Item not found
}
