package distributiontest

import (
	"fmt"
	"testing"

	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/actor"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/rewardpool"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/selector"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/utils"
)

func TestRewardDistributionReport(t *testing.T) {
	selectors := []struct {
		name     string
		selector types.ItemSelector
	}{
		{"PrefixSumSelector", selector.NewPrefixSumSelector()},
		{"FenwickTreeSelector", selector.NewFenwickTreeSelector()},
	}

	const totalDraws = 1000000

	for _, s := range selectors {
		t.Run(s.name, func(t *testing.T) {
			ctx := &types.Context{Utils: &utils.MockUtils{}}
			rewards := []types.PoolReward{
				{ItemID: "gold", Quantity: 1000000, Probability: 10},
				{ItemID: "silver", Quantity: 1000000, Probability: 20},
				{ItemID: "rock", Quantity: 1000000, Probability: 90},
			}
			pool := rewardpool.NewPool(
				rewards,
				rewardpool.PoolOptional{
					Selector: s.selector,
				},
			)
			w := &utils.MockWAL{}
			ctx.WAL = w

			opt := &actor.SystemOptional{RequestBufferSize: 1000, FlushAfterNDraw: 1000}
			sys, err := actor.NewSystem(ctx, pool, opt)
			if err != nil {
				t.Error(err)
				return
			}

			counts := make(map[string]int)
			for i := 0; i < totalDraws; i++ {
				resp := <-sys.Draw()
				if resp.Err == nil {
					counts[resp.Item]++
				}
			}
			sys.Stop()

			fmt.Printf("\n--- Distribution Report for %s ---\n", s.name)
			fmt.Println("|   Item   |   Count   | Proportion |")
			fmt.Println("|----------|-----------|------------|")

			totalProbability := int64(0)
			for _, r := range rewards {
				totalProbability += r.Probability
			}

			for _, r := range rewards {
				expectedProp := float64(r.Probability) / float64(totalProbability)
				actualProp := float64(counts[r.ItemID]) / float64(totalDraws)
				fmt.Printf("| %-8s | %9d |   %.4f   (expected %.4f) |\n", r.ItemID, counts[r.ItemID], actualProp, expectedProp)
			}
			fmt.Println("-------------------------------------------------")
		})
	}
}

func TestQuantityExhaustion(t *testing.T) {
	selectors := []struct {
		name     string
		selector types.ItemSelector
	}{
		{"PrefixSumSelector", selector.NewPrefixSumSelector()},
		{"FenwickTreeSelector", selector.NewFenwickTreeSelector()},
	}

	for _, s := range selectors {
		t.Run(s.name, func(t *testing.T) {
			rewards := []types.PoolReward{
				{ItemID: "gold", Quantity: 1, Probability: 100}, // High probability, low quantity
				{ItemID: "silver", Quantity: 100, Probability: 10},
			}

			ctx := &types.Context{Utils: &utils.MockUtils{}}
			pool := rewardpool.NewPool(
				rewards,
				rewardpool.PoolOptional{
					Selector: s.selector,
				},
			)
			w := &utils.MockWAL{}
			ctx.WAL = w

			opt := &actor.SystemOptional{RequestBufferSize: 100, FlushAfterNDraw: 10}
			sys, err := actor.NewSystem(ctx, pool, opt)
			if err != nil {
				t.Error(err)
				return
			}

			goldCount := 0
			counts := make(map[string]int)
			for i := 0; i < 200; i++ {
				resp := <-sys.Draw()
				if resp.Err == nil && resp.Item == "gold" {
					goldCount++
				}
				if resp.Err == nil {
					counts[resp.Item]++
				}
			}

			sys.Stop()
			final_state := pool.State()

			// Make sure pool drained out
			for _, val := range final_state {
				if val.Quantity > 0 {
					t.Errorf("Expected %s to be empty, but got %d", val.ItemID, val.Quantity)
				}

			}

			// Make sure pool delivery distribution for each item equal the rewards
			for _, val := range rewards {
				deliveried_count := counts[val.ItemID]
				if deliveried_count != val.Quantity {
					t.Errorf("Expected %s to be delivery %d, but got %d", val.ItemID, val.Quantity, deliveried_count)
				}
			}

			// Make sure only one gold
			if goldCount != 1 {
				t.Errorf("Expected 'gold' to be drawn at most once, but got %d", goldCount)
			}
		})
	}
}

func TestItemUpdateDistribution(t *testing.T) {
	selectors := []struct {
		name     string
		selector types.ItemSelector
	}{
		{"PrefixSumSelector", selector.NewPrefixSumSelector()},
		{"FenwickTreeSelector", selector.NewFenwickTreeSelector()},
	}

	const drawsBeforeUpdate = 100000
	const drawsAfterUpdate = 100000

	for _, s := range selectors {
		t.Run(s.name, func(t *testing.T) {
			ctx := &types.Context{Utils: &utils.MockUtils{}}
			rewards := []types.PoolReward{
				{ItemID: "gold", Quantity: 1000000, Probability: 10},
				{ItemID: "silver", Quantity: 1000000, Probability: 90},
			}
			pool := rewardpool.NewPool(
				rewards,
				rewardpool.PoolOptional{
					Selector: s.selector,
				},
			)
			w := &utils.MockWAL{}
			ctx.WAL = w

			opt := &actor.SystemOptional{RequestBufferSize: 1000, FlushAfterNDraw: 1000}
			sys, err := actor.NewSystem(ctx, pool, opt)
			if err != nil {
				t.Error(err)
				return
			}

			// Phase 1: Draw before update
			countsBefore := make(map[string]int)
			for i := 0; i < drawsBeforeUpdate; i++ {
				resp := <-sys.Draw()
				if resp.Err == nil {
					countsBefore[resp.Item]++
				}
			}

			// Update item probabilities
			err = sys.UpdateItem("gold", 1000000, 90)
			if err != nil {
				t.Fatalf("UpdateItem failed for gold: %v", err)
			}
			err = sys.UpdateItem("silver", 1000000, 10)
			if err != nil {
				t.Fatalf("UpdateItem failed for silver: %v", err)
			}

			// Phase 2: Draw after update
			countsAfter := make(map[string]int)
			for i := 0; i < drawsAfterUpdate; i++ {
				resp := <-sys.Draw()
				if resp.Err == nil {
					countsAfter[resp.Item]++
				}
			}

			sys.Stop()

			// Assertions
			fmt.Printf("\n--- Update Distribution Report for %s ---\n", s.name)
			fmt.Println("Phase 1 (Gold Prob: 10, Silver Prob: 90)")
			assertDistribution(t, countsBefore, drawsBeforeUpdate, 10, 90)

			fmt.Println("\nPhase 2 (Gold Prob: 90, Silver Prob: 10)")
			assertDistribution(t, countsAfter, drawsAfterUpdate, 90, 10)
			fmt.Println("-------------------------------------------------")
		})
	}
}

func assertDistribution(t *testing.T, counts map[string]int, totalDraws int, goldProb, silverProb int) {
	t.Helper()
	goldProp := float64(counts["gold"]) / float64(totalDraws)
	silverProp := float64(counts["silver"]) / float64(totalDraws)

	expectedGoldProp := float64(goldProb) / float64(goldProb+silverProb)
	expectedSilverProp := float64(silverProb) / float64(goldProb+silverProb)

	fmt.Printf("|   Item   |   Count   | Proportion |\n")
	fmt.Printf("|----------|-----------|------------|\n")
	fmt.Printf("| %-8s | %9d |   %.4f   (expected ~%.4f) |\n", "gold", counts["gold"], goldProp, expectedGoldProp)
	fmt.Printf("| %-8s | %9d |   %.4f   (expected ~%.4f) |\n", "silver", counts["silver"], silverProp, expectedSilverProp)

	// Allow a tolerance for randomness
	if goldProp < expectedGoldProp*0.8 || goldProp > expectedGoldProp*1.2 {
		t.Errorf("Gold proportion %.4f is outside the expected range (~%.4f)", goldProp, expectedGoldProp)
	}
	if silverProp < expectedSilverProp*0.8 || silverProp > expectedSilverProp*1.2 {
		t.Errorf("Silver proportion %.4f is outside the expected range (~%.4f)", silverProp, expectedSilverProp)
	}
}

