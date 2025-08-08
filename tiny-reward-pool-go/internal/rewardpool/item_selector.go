package rewardpool

import (
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"
)

// ItemSelector defines the contract for selecting items from a reward pool.
// It abstracts the underlying data structure used for efficient selection.
type ItemSelector interface {
	// Select chooses an item based on its availability and returns its ID.
	Select(ctx *types.Context) (string, error)

	// Update adjusts the quantity of a specific item in the selector.
	// A positive value increases availability, a negative value decreases it.
	Update(itemID string, quantity int64)

	// Reset clears the selector's state and re-initializes it with a new catalog.
	Reset(catalog []types.PoolReward)

	// TotalAvailable returns the total count of all items currently available for selection.
	TotalAvailable() int64

	// GetItemRemaining returns the remaining quantity of a specific item.
	GetItemRemaining(itemID string) int
}
