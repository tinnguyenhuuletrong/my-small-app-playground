package main

import (
	"runtime"
	"testing"
	"time"

	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/actor"
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

	opt := &actor.SystemOptional{RequestBufferSize: b.N}
	sys := actor.NewSystem(ctx, pool, opt)

	b.ResetTimer()
	start := time.Now()
	var memStatsStart, memStatsEnd runtime.MemStats

	runtime.ReadMemStats(&memStatsStart)

	resChans := make([]<-chan actor.DrawResponse, b.N)
	for i := 0; i < b.N; i++ {
		resChans[i] = sys.Draw()
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
