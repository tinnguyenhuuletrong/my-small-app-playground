package distributiontest

import (
	"fmt"
	"testing"

	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/processing"
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
			ctx := &types.Context{Utils: &utils.UtilsImpl{}}
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
			w := &selectorTestmockWAL{}
			ctx.WAL = w

			opt := &processing.ProcessorOptional{RequestBufferSize: 1000, FlushAfterNDraw: 1000}
			p := processing.NewProcessor(ctx, pool, opt)

			counts := make(map[string]int)
			for i := 0; i < totalDraws; i++ {
				resp := <-p.Draw()
				if resp.Err == nil {
					counts[resp.Item]++
				}
			}
			p.Stop()

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

			ctx := &types.Context{Utils: &utils.UtilsImpl{}}
			pool := rewardpool.NewPool(
				rewards,
				rewardpool.PoolOptional{
					Selector: s.selector,
				},
			)
			w := &selectorTestmockWAL{}
			ctx.WAL = w

			opt := &processing.ProcessorOptional{RequestBufferSize: 100, FlushAfterNDraw: 10}
			p := processing.NewProcessor(ctx, pool, opt)

			goldCount := 0
			counts := make(map[string]int)
			for i := 0; i < 200; i++ {
				resp := <-p.Draw()
				if resp.Err == nil && resp.Item == "gold" {
					goldCount++
				}
				if resp.Err == nil {
					counts[resp.Item]++
				}
			}

			p.Stop()
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

type selectorTestmockWAL struct {
}

func (m *selectorTestmockWAL) LogDraw(item types.WalLogItem) error {
	return nil
}
func (m *selectorTestmockWAL) Close() error                { return nil }
func (m *selectorTestmockWAL) Flush() error                { return nil }
func (m *selectorTestmockWAL) SetSnapshotPath(path string) {}
