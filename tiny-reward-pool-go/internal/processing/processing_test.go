package processing_test

import (
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/processing"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/utils"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/wal"
	walformatter "github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/wal/formatter"
	walstorage "github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/wal/storage"
)

func TestProcessor_TransactionalDraw(t *testing.T) {
	pool := &mockPool{item: types.PoolReward{ItemID: "gold", Quantity: 1, Probability: 1.0}}
	wal := &mockWAL{}
	utils := &utils.MockUtils{}
	ctx := &types.Context{WAL: wal, Utils: utils}
	proc := processing.NewProcessor(ctx, pool, &processing.ProcessorOptional{FlushAfterNDraw: 1})

	// Success path
	respChan := proc.Draw()
	gotResp := <-respChan
	if gotResp.Item == "" || gotResp.Item != "gold" {
		t.Fatalf("Expected gold, got %v", gotResp.Item)
	}
	if len(wal.logged) == 0 || !wal.logged[0].Success {
		t.Fatalf("Expected WAL log success, got %v", wal.logged)
	}
	if pool.committed != 1 {
		t.Fatalf("Expected committed=1, got %d", pool.committed)
	}

	// WAL failure path
	pool.item.Quantity = 1
	wal.fail = true
	respChan2 := proc.Draw()
	gotResp2 := <-respChan2
	if gotResp2.Item != "" {
		t.Fatalf("Expected nil item on WAL failure, got %v", gotResp2.Item)
	}
}

func TestProcessor_FlushAndFlushAfterNDraw(t *testing.T) {
	// Test case 1: Flush after N draws
	pool := &mockPool{item: types.PoolReward{ItemID: "gold", Quantity: 10, Probability: 1.0}}
	wal := &mockWAL{}
	utils := &utils.MockUtils{}
	ctx := &types.Context{WAL: wal, Utils: utils}
	flushN := 3
	proc := processing.NewProcessor(ctx, pool, &processing.ProcessorOptional{FlushAfterNDraw: flushN})

	// Perform N-1 draws, flush should not be called
	for i := 0; i < flushN-1; i++ {
		<-proc.Draw()
	}
	if wal.flushCount != 0 {
		t.Fatalf("Expected flushCount=0 after %d draws, got %d", flushN-1, wal.flushCount)
	}
	if pool.committed != 0 {
		t.Fatalf("Expected committed=0 after %d draws, got %d", flushN-1, pool.committed)
	}

	// Perform the Nth draw, flush should be called
	<-proc.Draw()
	if wal.flushCount != 1 {
		t.Fatalf("Expected flushCount=1 after %d draws, got %d", flushN, wal.flushCount)
	}
	if pool.committed != flushN {
		t.Fatalf("Expected committed=%d after %d draws, got %d", flushN, flushN, pool.committed)
	}
	if pool.reverted != 0 {
		t.Fatalf("Expected reverted=0, got %d", pool.reverted)
	}

	// Test case 2: WAL Flush failure
	pool = &mockPool{item: types.PoolReward{ItemID: "silver", Quantity: 10, Probability: 1.0}}
	wal = &mockWAL{flushFail: true}
	ctx = &types.Context{WAL: wal, Utils: utils}
	proc = processing.NewProcessor(ctx, pool, &processing.ProcessorOptional{FlushAfterNDraw: 1})

	<-proc.Draw()
	if wal.flushCount != 1 {
		t.Fatalf("Expected flushCount=1 on WAL flush failure, got %d", wal.flushCount)
	}
	if pool.committed != 0 {
		t.Fatalf("Expected committed=0 on WAL flush failure, got %d", pool.committed)
	}
	if pool.reverted != 1 {
		t.Fatalf("Expected reverted=1 on WAL flush failure, got %d", pool.reverted)
	}

	// Test case 3: Flush on Stop with remaining staged draws
	pool = &mockPool{item: types.PoolReward{ItemID: "bronze", Quantity: 10, Probability: 1.0}}
	wal = &mockWAL{}
	ctx = &types.Context{WAL: wal, Utils: utils}
	proc = processing.NewProcessor(ctx, pool, &processing.ProcessorOptional{FlushAfterNDraw: 100}) // High flushN to prevent auto-flush

	<-proc.Draw()
	if wal.flushCount != 0 { // Should not have flushed yet
		t.Fatalf("Expected flushCount=0 before stop, got %d", wal.flushCount)
	}
	proc.Stop() // This should trigger a final flush
	if wal.flushCount != 1 {
		t.Fatalf("Expected flushCount=1 after Stop, got %d", wal.flushCount)
	}
	if pool.committed != 1 {
		t.Fatalf("Expected committed=1 after Stop, got %d", pool.committed)
	}
	if pool.reverted != 0 {
		t.Fatalf("Expected reverted=0 after Stop, got %d", pool.reverted)
	}

	// Test case 4: Flush on Stop with WAL flush failure
	pool = &mockPool{item: types.PoolReward{ItemID: "platinum", Quantity: 10, Probability: 1.0}}
	wal = &mockWAL{flushFail: true}
	ctx = &types.Context{WAL: wal, Utils: utils}
	proc = processing.NewProcessor(ctx, pool, &processing.ProcessorOptional{FlushAfterNDraw: 100})

	<-proc.Draw()
	if wal.flushCount != 0 {
		t.Fatalf("Expected flushCount=0 before stop, got %d", wal.flushCount)
	}
	proc.Stop()
	if wal.flushCount != 1 {
		t.Fatalf("Expected flushCount=1 after Stop, got %d", wal.flushCount)
	}
	if pool.committed != 0 {
		t.Fatalf("Expected committed=0 after Stop with flush failure, got %d", pool.committed)
	}
	if pool.reverted != 1 {
		t.Fatalf("Expected reverted=1 after Stop with flush failure, got %d", pool.reverted)
	}
}

type mockPool struct {
	item      types.PoolReward
	staged    int
	committed int
	reverted  int
	pending   []string // track staged itemIDs for batch commit/revert
}

func (m *mockPool) SelectItem(ctx *types.Context) (string, error) {
	if m.item.Quantity-len(m.pending) > 0 {
		copyItem := m.item
		m.pending = append(m.pending, copyItem.ItemID)
		return copyItem.ItemID, nil
	}
	return "", nil
}
func (m *mockPool) CommitDraw() {
	// Commit all pending draws
	for range m.pending {
		m.committed++
		m.item.Quantity--
	}
	m.pending = nil
}
func (m *mockPool) RevertDraw() {
	// Revert all pending draws
	m.reverted += len(m.pending)
	m.pending = nil
}
func (m *mockPool) Load(cfg types.ConfigPool) error { return nil }
func (m *mockPool) LoadSnapshot(path string) error  { return nil }
func (m *mockPool) SaveSnapshot(path string) error  { return nil }

type mockWAL struct {
	logged     []types.WalLogDrawItem
	fail       bool
	flushCount int
	flushFail  bool
}

func (m *mockWAL) LogDraw(item types.WalLogDrawItem) error {
	m.logged = append(m.logged, item)
	if m.fail {
		return errors.New("simulated WAL error")
	}
	return nil
}
func (m *mockWAL) Close() error        { return nil }
func (m *mockWAL) Reset()              {}
func (m *mockWAL) Rotate(string) error { return nil }
func (m *mockWAL) Flush() error {
	m.flushCount++
	if m.flushFail {
		return errors.New("simulated WAL flush error")
	}
	return nil
}
func (m *mockWAL) SetSnapshotPath(path string) {}

// --- New Integration Test ---

type mockRotationUtils struct {
	rotatedPath       string
	snapshotPath      string
	genRotatedCalled  bool
	genSnapshotCalled bool
}

func (m *mockRotationUtils) GetLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func (m *mockRotationUtils) GenRotatedWALPath() *string {
	m.genRotatedCalled = true
	return &m.rotatedPath
}

func (m *mockRotationUtils) GenSnapshotPath() *string {
	m.genSnapshotCalled = true
	return &m.snapshotPath
}

type mockRotationPool struct {
	mockPool
	snapshotPath       string
	saveSnapshotCalled bool
}

func (m *mockRotationPool) SaveSnapshot(path string) error {
	m.saveSnapshotCalled = true
	m.snapshotPath = path
	// Create a dummy snapshot file
	return os.WriteFile(path, []byte("snapshot_data"), 0644)
}

func TestProcessor_WALRotation(t *testing.T) {
	// 1. Setup
	tmpDir := t.TempDir()
	walPath := filepath.Join(tmpDir, "test.wal")
	rotatedPath := filepath.Join(tmpDir, "test.wal.rotated")
	snapshotPath := filepath.Join(tmpDir, "test.snapshot")

	// Use a real mmap storage with a tiny size to force rotation
	mmapStorage, err := walstorage.NewFileMMapStorage(walPath, walstorage.FileMMapStorageOps{
		MMapFileSizeInBytes: 1024, // 1KB, very small
	})
	require.NoError(t, err)

	realWAL, err := wal.NewWAL(walPath, walformatter.NewJSONFormatter(), mmapStorage)
	require.NoError(t, err)

	mockPool := &mockRotationPool{
		mockPool: mockPool{item: types.PoolReward{ItemID: "gold", Quantity: 1000, Probability: 1}},
	}
	mockUtils := &mockRotationUtils{
		rotatedPath:  rotatedPath,
		snapshotPath: snapshotPath,
	}

	ctx := &types.Context{
		WAL:   realWAL,
		Utils: mockUtils,
	}

	// Flush after every draw to trigger the check
	proc := processing.NewProcessor(ctx, mockPool, &processing.ProcessorOptional{FlushAfterNDraw: 1})

	// 2. Execution: Write data until WAL is full
	// A single draw log is ~70 bytes. 1024 / 70 = ~15 draws needed. Let's do 20 to be safe.
	for i := 0; i < 20; i++ {
		<-proc.Draw()
	}

	// The processor runs in a separate goroutine, so we need to wait a bit
	// for the last flush to be processed.
	time.Sleep(200 * time.Millisecond)
	proc.Stop() // Final flush

	// 3. Assertions
	assert.True(t, mockUtils.genRotatedCalled, "GenRotatedWALPath should have been called")
	assert.True(t, mockUtils.genSnapshotCalled, "GenSnapshotPath should have been called")
	assert.True(t, mockPool.saveSnapshotCalled, "SaveSnapshot should have been called")
	assert.Equal(t, snapshotPath, mockPool.snapshotPath, "Snapshot should be saved to the correct path")

	// Check if the rotated WAL file exists
	_, err = os.Stat(rotatedPath)
	assert.NoError(t, err, "Rotated WAL file should exist")

	// Check if the snapshot file exists
	_, err = os.Stat(snapshotPath)
	assert.NoError(t, err, "Snapshot file should exist")

	// Check if the new WAL file was created
	_, err = os.Stat(walPath)
	assert.NoError(t, err, "New WAL file should exist at the original path")
}
