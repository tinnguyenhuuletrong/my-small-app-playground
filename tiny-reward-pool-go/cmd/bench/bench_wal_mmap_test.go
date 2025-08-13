package main

import (
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/processing"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/rewardpool"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/utils"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/wal"
	walformatter "github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/wal/formatter"
	walstorage "github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/wal/storage"
)

func BenchmarkPoolDrawWithMmapWALCallback(b *testing.B) {
	tmpDir := filepath.Join("_tmp")
	walPath := filepath.Join(tmpDir, "wal_mmap.log")
	_ = os.Remove(walPath)

	// Allocate 64MB for WAL file (adjust as needed)
	const walSize = 64 * 1024 * 1024

	jsonFormatter := walformatter.NewStringLineFormatter()
	fileStorage, err := walstorage.NewFileMMapStorage(walPath, walstorage.FileMMapStorageOps{
		MMapFileSizeInBytes: walSize,
	})
	if err != nil {
		b.Fatalf("failed to create file storage: %v", err)
	}

	w, err := wal.NewWAL(walPath, jsonFormatter, fileStorage)
	if err != nil {
		b.Fatalf("failed to create mmap WAL: %v", err)
	}
	defer w.Close()

	pool := rewardpool.NewPool(
		[]types.PoolReward{
			{ItemID: "gold", Quantity: 1000000, Probability: 1.0},
		},
	)
	ctx := &types.Context{
		WAL:   w,
		Utils: &utils.MockUtils{},
	}

	proc := processing.NewProcessor(ctx, pool, &processing.ProcessorOptional{
		FlushAfterNDraw: 10_000,
	})

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

	// Output basic stats for now
	// b.Logf("Elapsed: %v", elapsed)
	// b.Logf("Alloc: %d", memStatsEnd.Alloc-memStatsStart.Alloc)

	b.ReportMetric(float64(b.N)/elapsed.Seconds(), "draws/sec")
	b.ReportMetric(float64(memStatsEnd.TotalAlloc-memStatsStart.TotalAlloc)/float64(b.N), "bytes/draw")
	b.ReportMetric(float64(memStatsEnd.NumGC-memStatsStart.NumGC), "gc_count")
	// b.ReportMetric(walSize, "wal_file_size")
}
