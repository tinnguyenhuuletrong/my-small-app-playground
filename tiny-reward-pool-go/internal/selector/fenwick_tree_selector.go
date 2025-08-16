package selector

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/utils"
)

// FenwickTreeSelector implements the ItemSelector interface using a Fenwick Tree.
type FenwickTreeSelector struct {
	// tree stores the cumulative probabilities of items.
	tree *utils.FenwickTree

	// items stores the original reward data.
	items []types.PoolReward

	// itemIDs maps the index in the Fenwick tree back to the actual ItemID.
	itemIDs []string

	// itemIndex maps ItemID to its index in the Fenwick tree and itemIDs slice.
	itemIndex map[string]int

	// itemInfo tracks the current state (quantity) of each item.
	itemInfo map[string]*types.PoolReward

	// totalWeight stores the sum of all probabilities in the tree.
	totalWeight int64

	// rand is the random number generator for selection.
	rand *rand.Rand
}

var _ types.ItemSelector = (*FenwickTreeSelector)(nil)

// NewFenwickTreeSelector creates a new FenwickTreeSelector.
func NewFenwickTreeSelector() *FenwickTreeSelector {
	return &FenwickTreeSelector{
		itemIndex: make(map[string]int),
		itemInfo:  make(map[string]*types.PoolReward),
		rand:      rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Reset initializes or re-initializes the selector with a new catalog.
func (fts *FenwickTreeSelector) Reset(catalog []types.PoolReward) {
	fts.items = make([]types.PoolReward, len(catalog))
	fts.itemIDs = make([]string, len(catalog))
	fts.itemIndex = make(map[string]int)
	fts.itemInfo = make(map[string]*types.PoolReward, len(catalog))
	fts.totalWeight = 0

	fts.tree = utils.NewFenwickTree(len(catalog))

	for i, item := range catalog {
		// Create a copy to avoid modifying the original catalog
		itemCopy := item
		fts.items[i] = itemCopy
		fts.itemIDs[i] = item.ItemID
		fts.itemIndex[item.ItemID] = i
		fts.itemInfo[item.ItemID] = &fts.items[i]

		if item.Quantity > 0 {
			fts.tree.Add(i, item.Probability)
			fts.totalWeight += item.Probability
		}
	}

}

// Select chooses an item based on its availability.
func (fts *FenwickTreeSelector) Select(ctx *types.Context) (string, error) {
	if fts.totalWeight <= 0 {
		return "", types.ErrEmptyRewardPool
	}

	randVal := fts.rand.Int63n(fts.totalWeight) + 1
	idx := fts.tree.Find(randVal)

	if idx == -1 || idx >= len(fts.itemIDs) {
		return "", fmt.Errorf("internal error: failed to find item for random value %d (total weight: %d)", randVal, fts.totalWeight)
	}

	selectedItemID := fts.itemIDs[idx]
	if fts.itemInfo[selectedItemID].Quantity <= 0 {
		return "", fmt.Errorf("internal error: selected item %s has zero quantity", selectedItemID)
	}

	return selectedItemID, nil
}

// Update adjusts the quantity of a specific item in the selector.
func (fts *FenwickTreeSelector) Update(itemID string, delta int64) {
	idx, ok := fts.itemIndex[itemID]
	if !ok {
		return
	}

	item := fts.itemInfo[itemID]
	oldQuantity := item.Quantity
	newQuantity := oldQuantity + int(delta)
	item.Quantity = newQuantity

	// If the item becomes exhausted, remove its probability from the tree.
	if oldQuantity > 0 && newQuantity <= 0 {
		fts.tree.Add(idx, -item.Probability)
		fts.totalWeight -= item.Probability
	} else if oldQuantity <= 0 && newQuantity > 0 {
		// If the item becomes available again, add its probability back.
		fts.tree.Add(idx, item.Probability)
		fts.totalWeight += item.Probability
	}
}

// UpdateItem updates the quantity and probability of a specific item.
func (fts *FenwickTreeSelector) UpdateItem(itemID string, quantity int, probability int64) {
	// TODO: Implement in Iter 2
}

// TotalAvailable returns the total weight of all items currently available for selection.
func (fts *FenwickTreeSelector) TotalAvailable() int64 {
	return fts.totalWeight
}

// GetItemRemaining returns the remaining quantity of a specific item.
func (fts *FenwickTreeSelector) GetItemRemaining(itemID string) int {
	if item, ok := fts.itemInfo[itemID]; ok {
		return item.Quantity
	}
	return -1 // Item not found
}

// Return PoolReward[] for Snapshot
func (fts *FenwickTreeSelector) SnapshotCatalog() []types.PoolReward {
	snapshot_catalog := make([]types.PoolReward, len(fts.items))
	for i, val := range fts.items {
		snapshot_catalog[i] = types.PoolReward{
			Quantity:    fts.GetItemRemaining(val.ItemID),
			ItemID:      val.ItemID,
			Probability: val.Probability,
		}
	}
	return snapshot_catalog
}
