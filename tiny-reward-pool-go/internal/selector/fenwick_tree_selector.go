package selector

import (
	"fmt"
	"math/rand"

	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/utils"
)

// FenwickTreeSelector implements the ItemSelector interface using a Fenwick Tree.
type FenwickTreeSelector struct {
	// tree stores the cumulative quantities of items.
	tree *utils.FenwickTree

	// itemIDs maps the index in the Fenwick tree back to the actual ItemID.
	itemIDs []string

	// itemIndex maps ItemID to its index in the Fenwick tree and itemIDs slice.
	itemIndex map[string]int

	// totalAvailable stores the sum of all quantities in the tree.
	totalAvailable int64
}

// NewFenwickTreeSelector creates a new FenwickTreeSelector.
func NewFenwickTreeSelector() *FenwickTreeSelector {
	return &FenwickTreeSelector{
		itemIndex: make(map[string]int),
	}
}

// Reset initializes or re-initializes the selector with a new catalog.
func (fts *FenwickTreeSelector) Reset(catalog []types.PoolReward) {
	// fmt.Printf("FenwickTreeSelector.Reset called with catalog size: %d\n", len(catalog))
	// Clear existing data
	fts.itemIDs = make([]string, len(catalog))
	fts.itemIndex = make(map[string]int)
	fts.totalAvailable = 0

	// Initialize Fenwick Tree with the size of the catalog
	fts.tree = utils.NewFenwickTree(len(catalog))

	// Populate the tree and maps
	for i, item := range catalog {
		fts.itemIDs[i] = item.ItemID
		fts.itemIndex[item.ItemID] = i
		fts.tree.Add(i, int64(item.Quantity))
		fts.totalAvailable += int64(item.Quantity)
	}
	// fmt.Printf("FenwickTreeSelector.Reset finished. Total Available: %d\n", fts.totalAvailable)
}

// Select chooses an item based on its availability.
func (fts *FenwickTreeSelector) Select(ctx *types.Context) (string, error) {
	if fts.totalAvailable <= 0 {
		return "", types.ErrEmptyRewardPool
	}

	// Generate a random value within the total available range
	randVal := rand.Int63n(fts.totalAvailable) + 1 // +1 because FenwickTree.Find expects 1-based cumulative sum

	// Find the index of the item in Acc sum array. Where A[i] = sum(0...i]
	idx := fts.tree.Find(randVal)

	// This should ideally not happen if totalAvailable is correct and Find works as expected
	if idx == -1 || idx >= len(fts.itemIDs) {
		return "", fmt.Errorf("internal error: failed to find item for random value %d (total available: %d)", randVal, fts.totalAvailable)
	}

	selectedItemID := fts.itemIDs[idx]

	return selectedItemID, nil
}

// Update adjusts the quantity of a specific item in the selector.
func (fts *FenwickTreeSelector) Update(itemID string, quantity int64) {
	idx, ok := fts.itemIndex[itemID]
	if !ok {
		// Item not found in the selector, perhaps a new item or an error.
		// For now, we'll just ignore it as it shouldn't happen with existing items.
		return
	}

	fts.tree.Add(idx, quantity)
	fts.totalAvailable += quantity
}

// TotalAvailable returns the total count of all items currently available for selection.
func (fts *FenwickTreeSelector) TotalAvailable() int64 {
	return fts.totalAvailable
}

// GetItemRemaining returns the remaining quantity of a specific item.
func (fts *FenwickTreeSelector) GetItemRemaining(itemID string) int {
	idx, ok := fts.itemIndex[itemID]
	if !ok {
		return -1 // Item not found
	}

	// Query the current quantity of the item from the Fenwick tree.
	// This requires querying the prefix sum up to idx, and then subtracting the prefix sum up to idx-1.
	currentQuantity := fts.tree.Query(idx)
	if idx > 0 {
		currentQuantity -= fts.tree.Query(idx - 1)
	}

	return int(currentQuantity)
}
