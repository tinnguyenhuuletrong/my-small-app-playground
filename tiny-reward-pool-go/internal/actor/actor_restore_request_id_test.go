package actor_test

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/actor"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/recovery"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/rewardpool"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/wal"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/wal/formatter"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/wal/storage"
)

type mockUtilsForRestoreTest struct {
	snapshotPath string
}

func (m *mockUtilsForRestoreTest) GetLogger() *slog.Logger {
	return nil
}

func (m *mockUtilsForRestoreTest) GenRotatedWALPath() *string {
	return nil
}

func (m *mockUtilsForRestoreTest) GenSnapshotPath() *string {
	return &m.snapshotPath
}

func TestActor_RestoreRequestID(t *testing.T) {
	// 1. Setup initial environment
	tmpDir := t.TempDir()
	walPath := filepath.Join(tmpDir, "test.wal")
	snapshotPath := filepath.Join(tmpDir, "test.snapshot")
	configPath := filepath.Join(tmpDir, "config.json")

	configContent := []byte(`{"catalog":[{"item_id":"gold","quantity":100,"probability":1}]}`)
	require.NoError(t, os.WriteFile(configPath, configContent, 0644))

	// 2. First run: Create a system, draw some items, and stop it.
	var lastRequestID uint64
	func() {
		pool, err := rewardpool.CreatePoolFromConfigPath(configPath)
		require.NoError(t, err)

		fileStorage, err := storage.NewFileStorage(walPath)
		require.NoError(t, err)
		w, err := wal.NewWAL(walPath, formatter.NewJSONFormatter(), fileStorage)
		require.NoError(t, err)

		mockUtils := &mockUtilsForRestoreTest{snapshotPath: snapshotPath}
		ctx := &types.Context{WAL: w, Utils: mockUtils}

		sys, err := actor.NewSystem(ctx, pool, &actor.SystemOptional{FlushAfterNDraw: 1})
		require.NoError(t, err)

		// Draw 5 times
		for i := 0; i < 5; i++ {
			<-sys.Draw()
		}

		lastRequestID = sys.GetRequestID()
		require.Equal(t, uint64(5), lastRequestID)

		sys.Stop() // This will flush and create a final snapshot
	}()

	// 3. Second run: Recover the system and verify the request ID.
	func() {
		// Recover the pool and last request ID
		recoveredPool, recoveredRequestID, err := recovery.RecoverPool(snapshotPath, walPath, configPath, formatter.NewJSONFormatter(), &mockUtilsForRestoreTest{snapshotPath: snapshotPath})
		require.NoError(t, err)
		require.Equal(t, uint64(5), recoveredRequestID)

		// Create a new system with the recovered state
		fileStorage, err := storage.NewFileStorage(walPath)
		require.NoError(t, err)
		w, err := wal.NewWAL(walPath, formatter.NewJSONFormatter(), fileStorage)
		require.NoError(t, err)

		mockUtils := &mockUtilsForRestoreTest{snapshotPath: snapshotPath}
		ctx := &types.Context{WAL: w, Utils: mockUtils}

		sys, err := actor.NewSystem(ctx, recoveredPool, &actor.SystemOptional{FlushAfterNDraw: 1})
		require.NoError(t, err)

		// Set the restored request ID
		sys.SetRequestID(recoveredRequestID)

		// Draw again
		resp := <-sys.Draw()
		require.NoError(t, resp.Err)
		require.Equal(t, uint64(6), resp.RequestID)

		lastRequestID = sys.GetRequestID()
		require.Equal(t, uint64(6), lastRequestID)

		sys.Stop()
	}()
}