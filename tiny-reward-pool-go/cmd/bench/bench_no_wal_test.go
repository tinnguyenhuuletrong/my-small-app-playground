package main

import (
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/processing"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/rewardpool"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/utils"
)

func BenchmarkPoolDrawNoWalChannel(b *testing.B) {
	pool := rewardpool.NewPool(
		[]types.PoolReward{
			{ItemID: "gold", Quantity: 1000000, Probability: 1.0},
		},
	)
	ctx := &types.Context{
		WAL:   &utils.MockWAL{},
		Utils: &utils.MockUtils{},
	}

	opt := &processing.ProcessorOptional{RequestBufferSize: b.N}
	proc := processing.NewProcessor(ctx, pool, opt)

	b.ResetTimer()
	start := time.Now()
	var memStatsStart, memStatsEnd runtime.MemStats

	runtime.ReadMemStats(&memStatsStart)

	resChans := make([]<-chan processing.DrawResponse, b.N)
	for i := 0; i < b.N; i++ {
		resChans[i] = proc.Draw()
	}

	for _, ch := range resChans {
		<-ch
	}

	runtime.ReadMemStats(&memStatsEnd)
	elapsed := time.Since(start)

	b.StopTimer()

	b.ReportMetric(float64(b.N)/elapsed.Seconds(), "draws/sec")
	b.ReportMetric(float64(memStatsEnd.TotalAlloc-memStatsStart.TotalAlloc)/float64(b.N), "bytes/draw")
	b.ReportMetric(float64(memStatsEnd.NumGC-memStatsStart.NumGC), "gc_count")

}

func BenchmarkPoolDrawNoWalCallback(b *testing.B) {
	pool := rewardpool.NewPool(
		[]types.PoolReward{
			{ItemID: "gold", Quantity: 1000000, Probability: 1.0},
		},
	)
	ctx := &types.Context{
		WAL:   &utils.MockWAL{},
		Utils: &utils.MockUtils{},
	}

	opt := &processing.ProcessorOptional{RequestBufferSize: b.N}
	proc := processing.NewProcessor(ctx, pool, opt)

	var wg sync.WaitGroup

	b.ResetTimer()
	start := time.Now()
	var memStatsStart, memStatsEnd runtime.MemStats

	runtime.ReadMemStats(&memStatsStart)

	for i := 0; i < b.N; i++ {
		wg.Add(1)

		proc.DrawWithCallback(func(resp processing.DrawResponse) {
			wg.Done()
		})

	}

	wg.Wait()

	runtime.ReadMemStats(&memStatsEnd)
	elapsed := time.Since(start)

	b.StopTimer()

	b.ReportMetric(float64(b.N)/elapsed.Seconds(), "draws/sec")
	b.ReportMetric(float64(memStatsEnd.TotalAlloc-memStatsStart.TotalAlloc)/float64(b.N), "bytes/draw")
	b.ReportMetric(float64(memStatsEnd.NumGC-memStatsStart.NumGC), "gc_count")
}
