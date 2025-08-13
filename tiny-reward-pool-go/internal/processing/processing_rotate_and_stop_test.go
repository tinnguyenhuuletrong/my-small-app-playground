package processing_test

import (
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/processing"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/recovery"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/rewardpool"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/wal"
	walformatter "github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/wal/formatter"
	walstorage "github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/wal/storage"
)

type mockStopUtils struct {
	rotatedPath       string
	snapshotPath      string
	genRotatedCalled  bool
	genSnapshotCalled bool
}

func (m *mockStopUtils) GetLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func (m *mockStopUtils) GenRotatedWALPath() *string {
	m.genRotatedCalled = true
	return &m.rotatedPath
}

func (m *mockStopUtils) GenSnapshotPath() *string {
	m.genSnapshotCalled = true
	return &m.snapshotPath
}

func TestProcessor_StopWithWALRotationRaceCondition(t *testing.T) {
	// 1. Setup
	tmpDir := t.TempDir()
	walPath := filepath.Join(tmpDir, "test.wal")
	rotatedPath := filepath.Join(tmpDir, "test.wal.rotated")
	snapshotPath := filepath.Join(tmpDir, "test.snapshot")
	configPath := filepath.Join(tmpDir, "config.json")
	initialQuantity := int64(10)

	// Create a dummy config file
	configContent := []byte(`{"catalog":[{"item_id":"gold","quantity":10,"probability":1}]}`)
	require.NoError(t, os.WriteFile(configPath, configContent, 0644))

	// Use a real mmap storage with a tiny size to force rotation
	// A single draw log is ~70 bytes. 2 logs = 140 bytes.
	// Set size to 150 to ensure the 3rd write fails.
	mmapStorage, err := walstorage.NewFileMMapStorage(walPath, walstorage.FileMMapStorageOps{
		MMapFileSizeInBytes: 150,
	})
	require.NoError(t, err)

	realWAL, err := wal.NewWAL(walPath, walformatter.NewJSONFormatter(), mmapStorage)
	require.NoError(t, err)

	pool := rewardpool.NewPool([]types.PoolReward{
		{ItemID: "gold", Quantity: int(initialQuantity), Probability: 1},
	})

	mockUtils := &mockStopUtils{
		rotatedPath:  rotatedPath,
		snapshotPath: snapshotPath,
	}

	ctx := &types.Context{
		WAL:   realWAL,
		Utils: mockUtils,
	}

	// Flush after every draw to trigger the check
	proc := processing.NewProcessor(ctx, pool, &processing.ProcessorOptional{FlushAfterNDraw: 1})

	// 2. Execution
	// Draw twice. These should succeed and be written to the WAL buffer.
	<-proc.Draw()
	<-proc.Draw()

	// Give the processor a moment to process the draws
	time.Sleep(100 * time.Millisecond)

	// Let's draw one more time to ensure the WAL is full on the next flush, which happens in Stop()
	<-proc.Draw()
	time.Sleep(100 * time.Millisecond)

	proc.Stop() // This will trigger rotation

	// 3. Assertions
	assert.True(t, mockUtils.genRotatedCalled, "GenRotatedWALPath should have been called")
	assert.True(t, mockUtils.genSnapshotCalled, "GenSnapshotPath should have been called")

	// Check that the snapshot was created
	snapshotData, err := os.ReadFile(snapshotPath)
	require.NoError(t, err, "Snapshot file should exist and be readable")

	var snapshotContent struct {
		Items []types.PoolReward `json:"catalog"`
	}
	err = json.Unmarshal(snapshotData, &snapshotContent)
	require.NoError(t, err, "Snapshot should be valid JSON")

	// The snapshot should reflect the state *after* all 3 draws were committed.
	expectedQuantityAfterStop := initialQuantity - 3
	assert.Equal(t, int(expectedQuantityAfterStop), snapshotContent.Items[0].Quantity, "Snapshot should have the final quantity")

	// 4. Recovery Test
	// Now, let's recover from the state we just saved.
	// The recovery process should load the snapshot and then apply any logs in the *new* WAL.
	recoveredPool, err := recovery.RecoverPool(snapshotPath, walPath, configPath, walformatter.NewJSONFormatter(), mockUtils)
	require.NoError(t, err, "Recovery process should succeed")

	// Verify the final state of the recovered pool
	finalQuantity := recoveredPool.GetItemRemaining("gold")
	assert.Equal(t, int(expectedQuantityAfterStop), finalQuantity, "Recovered pool quantity should match the snapshot and not have logs applied twice")
}