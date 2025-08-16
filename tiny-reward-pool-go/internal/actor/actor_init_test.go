package actor_test

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/actor"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"
)

// mockWalWithSize is a mock WAL that includes the Size method.
type mockWalWithSize struct {
	mockWAL
	size      int64
	sizeErr   error
	flushed   bool
	loggedVal types.WalLogEntry
}

func (m *mockWalWithSize) Size() (int64, error) {
	return m.size, m.sizeErr
}

func (m *mockWalWithSize) Flush() error {
	m.flushed = true
	return m.mockWAL.Flush()
}

func (m *mockWalWithSize) LogSnapshot(item types.WalLogSnapshotItem) error {
	m.loggedVal = &item
	return m.mockWAL.LogSnapshot(item)
}

// mockPoolForInit is a mock pool for testing initialization.
type mockPoolForInit struct {
	mockPool
	snapshotPath       string
	saveSnapshotCalled bool
}

func (m *mockPoolForInit) SaveSnapshot(path string) error {
	m.saveSnapshotCalled = true
	m.snapshotPath = path
	return nil
}

// mockUtilsForInit is a mock utils for testing initialization.
type mockUtilsForInit struct {
	snapshotPath string
}

func (m *mockUtilsForInit) GetLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func (m *mockUtilsForInit) GenRotatedWALPath() *string {
	return nil
}

func (m *mockUtilsForInit) GenSnapshotPath() *string {
	return &m.snapshotPath
}

func TestSystem_InitialSnapshotOnEmptyWAL(t *testing.T) {
	tmpDir := t.TempDir()
	snapshotPath := filepath.Join(tmpDir, "test.snapshot")

	tests := []struct {
		name                 string
		walSize              int64
		walSizeErr           error
		expectSnapshot       bool
		expectSnapshotPath   string
		expectFlush          bool
		expectActorStartFail bool
	}{
		{
			name:               "WAL is empty",
			walSize:            0,
			walSizeErr:         nil,
			expectSnapshot:     true,
			expectSnapshotPath: snapshotPath,
			expectFlush:        true,
		},
		{
			name:           "WAL is not empty",
			walSize:        100,
			walSizeErr:     nil,
			expectSnapshot: false,
			expectFlush:    false,
		},
		{
			name:                 "WAL size check fails",
			walSize:              0,
			walSizeErr:           fmt.Errorf("size error"),
			expectSnapshot:       false,
			expectFlush:          false,
			expectActorStartFail: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 1. Setup
			wal := &mockWalWithSize{
				size:    tt.walSize,
				sizeErr: tt.walSizeErr,
			}
			pool := &mockPoolForInit{}
			mockUtils := &mockUtilsForInit{
				snapshotPath: snapshotPath,
			}
			ctx := &types.Context{
				WAL:   wal,
				Utils: mockUtils,
			}

			// 2. Execution
			sys, err := actor.NewSystem(ctx, pool, &actor.SystemOptional{})

			if tt.expectActorStartFail {
				require.Error(t, err)
				require.Nil(t, sys)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, sys)
			defer sys.Stop()

			// 3. Assertions
			assert.Equal(t, tt.expectSnapshot, pool.saveSnapshotCalled, "SaveSnapshot call expectation mismatch")
			assert.Equal(t, tt.expectFlush, wal.flushed, "Flush call expectation mismatch")

			if tt.expectSnapshot {
				assert.Equal(t, tt.expectSnapshotPath, pool.snapshotPath, "Snapshot saved to wrong path")
				require.NotNil(t, wal.loggedVal)
				loggedSnapshot, ok := wal.loggedVal.(*types.WalLogSnapshotItem)
				require.True(t, ok, "Logged item is not a snapshot")
				assert.Equal(t, types.LogTypeSnapshot, loggedSnapshot.Type)
				assert.Equal(t, snapshotPath, loggedSnapshot.Path)
			} else {
				assert.Nil(t, wal.loggedVal, "Snapshot should not have been logged")
			}
		})
	}
}

func TestSystem_StartWithNonSizableWAL(t *testing.T) {
	// 1. Setup
	// Use the original mockWAL which does not have the Size() method
	wal := &mockWAL{size: 10}
	pool := &mockPoolForInit{}
	mockUtils := &mockUtilsForInit{}
	ctx := &types.Context{
		WAL:   wal,
		Utils: mockUtils,
	}

	// 2. Execution
	sys, err := actor.NewSystem(ctx, pool, &actor.SystemOptional{})

	// 3. Assertions
	// Init should succeed without error, but no snapshot should be created.
	require.NoError(t, err)
	require.NotNil(t, sys)
	defer sys.Stop()
	assert.False(t, pool.saveSnapshotCalled, "Snapshot should not be created for non-sizable WAL")
}
