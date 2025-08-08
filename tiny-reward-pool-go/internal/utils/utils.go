package utils

import (
	"math/rand"

	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"
)

type UtilsImpl struct{}

func (u *UtilsImpl) RandomItem(items []types.PoolReward) (int, error) {
	if len(items) == 0 {
		return -1, nil
	}
	// Simple random selection by probability
	total := int64(0)
	for _, item := range items {
		total += item.Probability
	}
	r := rand.Int63n(total)
	acc := int64(0)
	for i, item := range items {
		acc += item.Probability
		if r <= acc {
			return i, nil
		}
	}
	return 0, nil
}
