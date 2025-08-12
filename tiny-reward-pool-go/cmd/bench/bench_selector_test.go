package main

import (
	"runtime"
	"testing"

	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/processing"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/rewardpool"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/selector"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/utils"
)

func BenchmarkDrawChannel_PrefixSumSelector(b *testing.B) {
	ctx := &types.Context{Utils: &utils.UtilsImpl{}}
	pool := rewardpool.NewPool(
		[]types.PoolReward{
			{ItemID: "gold", Quantity: b.N, Probability: 10},
			{ItemID: "silver", Quantity: b.N, Probability: 20},
			{ItemID: "bronze", Quantity: b.N, Probability: 30},
			{ItemID: "rock", Quantity: b.N, Probability: 90},
		},

		rewardpool.PoolOptional{
			Selector: selector.NewPrefixSumSelector(),
		},
	)
	w := &utils.MockWAL{}
	ctx.WAL = w

	opt := &processing.ProcessorOptional{RequestBufferSize: b.N, FlushAfterNDraw: 1000}
	p := processing.NewProcessor(ctx, pool, opt)

	var memStatsStart, memStatsEnd runtime.MemStats
	b.ResetTimer()
	runtime.ReadMemStats(&memStatsStart)

	resChans := make([]<-chan processing.DrawResponse, b.N)
	for i := 0; i < b.N; i++ {
		resChans[i] = p.Draw()
	}

	for _, ch := range resChans {
		<-ch
	}

	runtime.ReadMemStats(&memStatsEnd)
	p.Stop()

	b.ReportMetric(float64(memStatsEnd.TotalAlloc-memStatsStart.TotalAlloc)/float64(b.N), "bytes/draw")
	b.ReportMetric(float64(memStatsEnd.NumGC-memStatsStart.NumGC), "gc_count")
}

func BenchmarkDrawChannel_FenwickTreeSelector(b *testing.B) {
	ctx := &types.Context{Utils: &utils.UtilsImpl{}}
	pool := rewardpool.NewPool(
		[]types.PoolReward{
			{ItemID: "gold", Quantity: b.N, Probability: 10},
			{ItemID: "silver", Quantity: b.N, Probability: 20},
			{ItemID: "bronze", Quantity: b.N, Probability: 30},
			{ItemID: "rock", Quantity: b.N, Probability: 90},
		},

		rewardpool.PoolOptional{
			Selector: selector.NewFenwickTreeSelector(),
		},
	)
	w := &utils.MockWAL{}
	ctx.WAL = w

	opt := &processing.ProcessorOptional{RequestBufferSize: b.N, FlushAfterNDraw: 1000}
	p := processing.NewProcessor(ctx, pool, opt)

	var memStatsStart, memStatsEnd runtime.MemStats
	b.ResetTimer()
	runtime.ReadMemStats(&memStatsStart)

	resChans := make([]<-chan processing.DrawResponse, b.N)
	for i := 0; i < b.N; i++ {
		resChans[i] = p.Draw()
	}

	for _, ch := range resChans {
		<-ch
	}

	runtime.ReadMemStats(&memStatsEnd)
	p.Stop()

	b.ReportMetric(float64(memStatsEnd.TotalAlloc-memStatsStart.TotalAlloc)/float64(b.N), "bytes/draw")
	b.ReportMetric(float64(memStatsEnd.NumGC-memStatsStart.NumGC), "gc_count")
}
