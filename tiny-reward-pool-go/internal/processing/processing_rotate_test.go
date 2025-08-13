package processing_test

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/processing"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/wal"
	walformatter "github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/wal/formatter"
	walstorage "github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/wal/storage"
)

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
