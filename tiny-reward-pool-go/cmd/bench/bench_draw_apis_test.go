package main

import (
	"sync"
	"testing"

	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/processing"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/rewardpool"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/utils"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/wal"
)

func BenchmarkDrawWithCallback(b *testing.B) {
	ctx := &types.Context{Utils: &utils.UtilsImpl{}}
	pool := rewardpool.NewPool(
		[]types.PoolReward{
			{ItemID: "gold", Quantity: b.N, Probability: 1.0},
		},
	)
	w, _ := wal.NewWAL("/tmp/bench_draw_apis_test.go")
	ctx.WAL = w

	opt := &processing.ProcessorOptional{RequestBufferSize: b.N}
	p := processing.NewProcessor(ctx, pool, opt)

	var wg sync.WaitGroup
	wg.Add(b.N)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		p.DrawWithCallback(func(resp processing.DrawResponse) {
			wg.Done()
		})
	}
	wg.Wait()
	p.Stop()
}

func BenchmarkDrawChannel(b *testing.B) {
	ctx := &types.Context{Utils: &utils.UtilsImpl{}}
	pool := rewardpool.NewPool(
		[]types.PoolReward{
			{ItemID: "gold", Quantity: b.N, Probability: 1.0},
		},
	)
	w, _ := wal.NewWAL("/tmp/bench_draw_apis_test.go")
	ctx.WAL = w

	opt := &processing.ProcessorOptional{RequestBufferSize: b.N}
	p := processing.NewProcessor(ctx, pool, opt)

	var wg sync.WaitGroup
	wg.Add(b.N)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		go func() {
			<-p.Draw()
			wg.Done()
		}()
	}
	wg.Wait()
	p.Stop()
}
