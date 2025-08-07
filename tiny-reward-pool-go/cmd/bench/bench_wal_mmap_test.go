package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/edsrzf/mmap-go"

	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/processing"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/rewardpool"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/utils"
)

func BenchmarkPoolDrawWithMmapWAL(b *testing.B) {
	tmpDir := filepath.Join("_tmp")
	walPath := filepath.Join(tmpDir, "wal_mmap.log")
	_ = os.Remove(walPath)

	w, err := NewMmapWAL(walPath)
	if err != nil {
		b.Fatalf("failed to create mmap WAL: %v", err)
	}
	defer w.Close()

	pool := &rewardpool.Pool{
		Catalog: []types.PoolReward{
			{ItemID: "gold", Quantity: 1000000, Probability: 1.0},
		},
		PendingDraws: make(map[string]int),
	}
	ctx := &types.Context{
		WAL:   w,
		Utils: &utils.UtilsImpl{},
	}

	proc := processing.NewProcessor(ctx, pool)

	var wg sync.WaitGroup

	b.ResetTimer()
	start := time.Now()
	var memStatsStart, memStatsEnd runtime.MemStats

	runtime.ReadMemStats(&memStatsStart)

	for i := 0; i < b.N; i++ {
		wg.Add(1)
		proc.Draw(func(resp processing.DrawResponse) {
			// Only wait for draw completion, WAL logging is handled inside Processor
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

type MmapWAL struct {
	file   *os.File
	mmap   mmap.MMap
	offset int64
	size   int64
}

func NewMmapWAL(path string) (*MmapWAL, error) {
	// Allocate 64MB for WAL file (adjust as needed)
	const walSize = 64 * 1024 * 1024
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}
	if err := f.Truncate(walSize); err != nil {
		f.Close()
		return nil, err
	}
	mm, err := mmap.Map(f, mmap.RDWR, 0)
	if err != nil {
		f.Close()
		return nil, err
	}
	return &MmapWAL{file: f, mmap: mm, size: walSize}, nil
}

func (w *MmapWAL) LogDraw(item types.WalLogItem) error {
	var line string
	if item.Success {
		line = fmt.Sprintf("DRAW %d %s\n", item.RequestID, item.ItemID)
	} else {
		line = fmt.Sprintf("DRAW %d FAILED\n", item.RequestID)
	}
	lineBytes := []byte(line)
	if w.offset+int64(len(lineBytes)) > w.size {
		return fmt.Errorf("WAL mmap full")
	}
	copy(w.mmap[w.offset:], lineBytes)
	w.offset += int64(len(lineBytes))
	return nil
}

func (w *MmapWAL) Flush() error {
	return w.mmap.Flush()
}

func (w *MmapWAL) Close() error {
	if err := w.mmap.Flush(); err != nil {
		return err
	}
	if err := w.mmap.Unmap(); err != nil {
		return err
	}
	return w.file.Close()
}
