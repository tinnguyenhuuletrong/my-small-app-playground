package utils_test

import (
	"testing"

	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/utils"
)

func TestRandomItem(t *testing.T) {
	u := &utils.UtilsImpl{}
	items := []types.PoolReward{
		{ItemID: "gold", Probability: 50},
		{ItemID: "silver", Probability: 50},
	}
	idx, err := u.RandomItem(items)
	if err != nil {
		t.Fatalf("RandomItem failed: %v", err)
	}
	if idx < 0 || idx >= len(items) {
		t.Fatalf("RandomItem index out of range: %d", idx)
	}
}
