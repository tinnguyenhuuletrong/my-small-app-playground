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
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/utils"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/wal"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/wal/formatter"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/wal/storage"
)

type mockUtilsForRestoreTest struct {
	snapshotPath string
	walDir       string
}

func (m *mockUtilsForRestoreTest) GetLogger() *slog.Logger {
	return nil
}

func (m *mockUtilsForRestoreTest) GenSnapshotPath() *string {
	return &m.snapshotPath
}

func (m *mockUtilsForRestoreTest) GetWALFiles() ([]string, error) {
	return utils.NewDefaultUtils(m.walDir, "", slog.LevelDebug, nil).GetWALFiles()
}

func (m *mockUtilsForRestoreTest) GenNextWALPath() (string, uint64, error) {
	return utils.NewDefaultUtils(m.walDir, "", slog.LevelDebug, nil).GenNextWALPath()
}

func TestActor_RestoreRequestID(t *testing.T) {
	// 1. Setup initial environment
	tmpDir := t.TempDir()
	walDir := filepath.Join(tmpDir, "wal")
	require.NoError(t, os.MkdirAll(walDir, 0755))
	walPath := filepath.Join(walDir, "wal.000")
	snapshotPath := filepath.Join(tmpDir, "test.snapshot")
	configPath := filepath.Join(tmpDir, "config.json")

	configContent := []byte(`{"catalog":[{"item_id":"gold","quantity":100,"probability":1}]}`)
	require.NoError(t, os.WriteFile(configPath, configContent, 0644))

	// 2. First run: Create a system, draw some items, and stop it.
	var lastRequestID uint64
	func() {
		pool, err := rewardpool.CreatePoolFromConfigPath(configPath)
		require.NoError(t, err)

		fileStorage, err := storage.NewFileStorage(walPath, 0)
		require.NoError(t, err)
		w, err := wal.NewWAL(walPath, 0, formatter.NewJSONFormatter(), fileStorage)
		require.NoError(t, err)

		mockUtils := &mockUtilsForRestoreTest{snapshotPath: snapshotPath, walDir: walDir}
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
		mockUtils := &mockUtilsForRestoreTest{snapshotPath: snapshotPath, walDir: walDir}
		recoveredPool, recoveredRequestID, _, err := recovery.RecoverPool(configPath, formatter.NewJSONFormatter(), mockUtils)
		require.NoError(t, err)
		require.Equal(t, uint64(5), recoveredRequestID)

		// Create a new system with the recovered state
		fileStorage, err := storage.NewFileStorage(walPath, 0)
		require.NoError(t, err)
		w, err := wal.NewWAL(walPath, 0, formatter.NewJSONFormatter(), fileStorage)
		require.NoError(t, err)

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
