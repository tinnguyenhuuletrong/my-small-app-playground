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

func BenchmarkPoolDrawNoWal(b *testing.B) {
	pool := &rewardpool.Pool{
		Catalog: []types.PoolReward{
			{ItemID: "gold", Quantity: 1000000, Probability: 1.0},
		},
		PendingDraws: make(map[string]int),
	}
	ctx := &types.Context{
		WAL:   &mockWAL{},
		Utils: &utils.UtilsImpl{},
	}

	proc := processing.NewProcessor(ctx, pool, nil)

	var wg sync.WaitGroup

	b.ResetTimer()
	start := time.Now()
	var memStatsStart, memStatsEnd runtime.MemStats

	runtime.ReadMemStats(&memStatsStart)

	for i := 0; i < b.N; i++ {
		wg.Add(1)
		proc.Draw(func(resp processing.DrawResponse) {
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

type mockWAL struct {
}

func (m *mockWAL) LogDraw(item types.WalLogItem) error {
	return nil
}
func (m *mockWAL) Close() error                { return nil }
func (m *mockWAL) Flush() error                { return nil }
func (m *mockWAL) SetSnapshotPath(path string) {}
