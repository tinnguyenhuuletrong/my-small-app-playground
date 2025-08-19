package actor_test

import (
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/actor"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/rewardpool"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/utils"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/wal"
	walformatter "github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/wal/formatter"
	walstorage "github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/wal/storage"
)

func TestSystem_TransactionalDraw(t *testing.T) {
	pool := &mockPool{item: types.PoolReward{ItemID: "gold", Quantity: 1, Probability: 1.0}}

	// wal contain something -> no need create snapshot
	wal := &mockWAL{size: 10}
	ctx := &types.Context{WAL: wal, Utils: &utils.MockUtils{}}
	sys, err := actor.NewSystem(ctx, pool, &actor.SystemOptional{FlushAfterNDraw: 1})
	require.NoError(t, err)
	defer sys.Stop()

	// Success path
	respChan := sys.Draw()
	gotResp := <-respChan
	if gotResp.Item == "" || gotResp.Item != "gold" {
		t.Fatalf("Expected gold, got %v", gotResp.Item)
	}
	if len(wal.logged) == 0 || !wal.logged[0].(*types.WalLogDrawItem).Success {
		t.Fatalf("Expected WAL log success, got %v", wal.logged)
	}
	if pool.committed != 1 {
		t.Fatalf("Expected committed=1, got %d", pool.committed)
	}

	// WAL failure path
	pool.item.Quantity = 1
	wal.fail = true
	respChan2 := sys.Draw()
	gotResp2 := <-respChan2
	if gotResp2.Item != "" {
		t.Fatalf("Expected nil item on WAL failure, got %v", gotResp2.Item)
	}
}

func TestSystem_FlushAndFlushAfterNDraw(t *testing.T) {
	// Test case 1: Flush after N draws
	pool := &mockPool{item: types.PoolReward{ItemID: "gold", Quantity: 10, Probability: 1.0}}

	// wal contain something -> no need create snapshot
	wal := &mockWAL{size: 10}
	ctx := &types.Context{WAL: wal, Utils: &utils.MockUtils{}}
	flushN := 3
	sys, err := actor.NewSystem(ctx, pool, &actor.SystemOptional{FlushAfterNDraw: flushN})
	require.NoError(t, err)

	// Perform N-1 draws, flush should not be called
	for i := 0; i < flushN-1; i++ {
		<-sys.Draw()
	}
	if wal.flushCount != 0 {
		t.Fatalf("Expected flushCount=0 after %d draws, got %d", flushN-1, wal.flushCount)
	}
	if pool.committed != 0 {
		t.Fatalf("Expected committed=0 after %d draws, got %d", flushN-1, pool.committed)
	}

	// Perform the Nth draw, flush should be called
	<-sys.Draw()
	if wal.flushCount != 1 {
		t.Fatalf("Expected flushCount=1 after %d draws, got %d", flushN, wal.flushCount)
	}
	if pool.committed != flushN {
		t.Fatalf("Expected committed=%d after %d draws, got %d", flushN, flushN, pool.committed)
	}
	if pool.reverted != 0 {
		t.Fatalf("Expected reverted=0, got %d", pool.reverted)
	}
	sys.Stop()
}

func TestSystem_FlushOnStop(t *testing.T) {
	// Test case 3: Flush on Stop with remaining staged draws
	pool := &mockPool{item: types.PoolReward{ItemID: "bronze", Quantity: 10, Probability: 1.0}}

	// wal contain something -> no need create snapshot
	wal := &mockWAL{size: 10}
	ctx := &types.Context{WAL: wal, Utils: &utils.MockUtils{}}
	sys, err := actor.NewSystem(ctx, pool, &actor.SystemOptional{FlushAfterNDraw: 100}) // High flushN to prevent auto-flush
	require.NoError(t, err)

	<-sys.Draw()
	if wal.flushCount != 0 { // Should not have flushed yet
		t.Fatalf("Expected flushCount=0 before stop, got %d", wal.flushCount)
	}
	sys.Stop() // This should trigger a final flush
	if wal.flushCount != 1 {
		t.Fatalf("Expected flushCount=1 after Stop, got %d", wal.flushCount)
	}
	if pool.committed != 1 {
		t.Fatalf("Expected committed=1 after Stop, got %d", pool.committed)
	}
	if pool.reverted != 0 {
		t.Fatalf("Expected reverted=0 after Stop, got %d", pool.reverted)
	}
}

// Additional tests for rotation

func TestSystem_WALRotation(t *testing.T) {
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

	initQuantity := 1000
	numberDraw := 20
	mockPool := &mockRotationPool{
		mockPool: mockPool{item: types.PoolReward{ItemID: "gold", Quantity: initQuantity, Probability: 1}},
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
	sys, err := actor.NewSystem(ctx, mockPool, &actor.SystemOptional{FlushAfterNDraw: 1})
	require.NoError(t, err)

	// 2. Execution: Write data until WAL is full
	// A single draw log is ~70 bytes. 1024 / 70 = ~15 draws needed. Let's do 20 to be safe.
	for i := 0; i < numberDraw; i++ {
		<-sys.Draw()
	}

	// The processor runs in a separate goroutine, so we need to wait a bit
	// for the last flush to be processed.
	time.Sleep(200 * time.Millisecond)

	// check state correct
	state := sys.State()
	remainingItem := initQuantity - numberDraw
	require.Equal(t, state[0].Quantity, remainingItem, "pool Quantity should correct")

	sys.Stop() // Final flush

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

func TestSystem_StopWithWALRotationRaceCondition(t *testing.T) {
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
	mmapStorage, err := walstorage.NewFileMMapStorage(walPath, walstorage.FileMMapStorageOps{
		MMapFileSizeInBytes: 150 * 2,
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
	sys, err := actor.NewSystem(ctx, pool, &actor.SystemOptional{FlushAfterNDraw: 1})
	require.NoError(t, err)

	// 2. Execution
	<-sys.Draw()
	<-sys.Draw()
	time.Sleep(100 * time.Millisecond)
	<-sys.Draw()
	time.Sleep(100 * time.Millisecond)
	sys.Stop() // This will trigger rotation

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

	// Snapshot should revert and apply remaining draw
	expectedQuantityAfterStop := initialQuantity - 2
	assert.Equal(t, int(expectedQuantityAfterStop), snapshotContent.Items[0].Quantity, "Snapshot should have pending")

	// Check active wal log. It should
	//	- snapshot
	//	- 1 draw log
	logItems, err := wal.ParseWAL(walPath, walformatter.NewJSONFormatter())
	assert.Equal(t, logItems[0].GetType(), types.LogTypeSnapshot, "1st item should be a snapshot")
	assert.Equal(t, logItems[1].GetType(), types.LogTypeDraw, "2st item should be a draw")

	snapshotItm := logItems[0].(*types.WalLogSnapshotItem)
	assert.Equal(t, snapshotItm.Path, snapshotPath, "1st item snapshot path should correct")
	updateItm := logItems[1].(*types.WalLogDrawItem)
	assert.Equal(t, updateItm.ItemID, "gold", "2st item should relay correct ItemID")

}

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

// Mocks
type mockPool struct {
	item      types.PoolReward
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
	return "", types.ErrEmptyRewardPool
}
func (m *mockPool) State() []types.PoolReward {
	if m.item.Quantity > 0 {
		return []types.PoolReward{m.item}
	}
	return []types.PoolReward{}
}

func (m *mockPool) CommitDraw() {
	m.committed += len(m.pending)
	for range m.pending {
		m.item.Quantity--
	}
	m.pending = nil
}

func (m *mockPool) RevertDraw() {
	m.reverted += len(m.pending)
	m.pending = nil
}

func (m *mockPool) Load(cfg types.ConfigPool) error                               { return nil }
func (m *mockPool) LoadSnapshot(path string) error                                { return nil }
func (m *mockPool) SaveSnapshot(path string) error                                { return nil }
func (m *mockPool) ApplyUpdateLog(itemID string, quantity int, probability int64) {}
func (m *mockPool) UpdateItem(itemID string, quantity int, probability int64) error {
	return nil
}

type mockWAL struct {
	logged     []types.WalLogEntry
	fail       bool
	flushCount int
	flushFail  bool
	size       int
}

func (m *mockWAL) LogDraw(item types.WalLogDrawItem) error {
	m.logged = append(m.logged, &item)
	if m.fail {
		return errors.New("simulated WAL error")
	}
	return nil
}
func (m *mockWAL) LogUpdate(item types.WalLogUpdateItem) error {
	m.logged = append(m.logged, &item)
	return nil
}
func (m *mockWAL) LogSnapshot(item types.WalLogSnapshotItem) error { return nil }
func (m *mockWAL) LogRotate(item types.WalLogRotateItem) error     { return nil }

func (m *mockWAL) Close() error { return nil }
func (m *mockWAL) Reset()       {}

func (m *mockWAL) Rotate(string) error  { return nil }
func (w *mockWAL) Size() (int64, error) { return int64(w.size), nil }
func (m *mockWAL) Flush() error {
	m.flushCount++
	if m.flushFail {
		return errors.New("simulated WAL flush error")
	}
	return nil
}

func TestSystem_UpdateItem(t *testing.T) {
	// 1. Setup
	catalog := []types.PoolReward{
		{ItemID: "item1", Quantity: 10, Probability: 20},
	}
	pool := rewardpool.NewPool(catalog)
	wal := &mockWAL{size: 10} // Non-empty WAL
	ctx := &types.Context{WAL: wal, Utils: &utils.MockUtils{}}
	sys, err := actor.NewSystem(ctx, pool, nil)
	require.NoError(t, err)
	defer sys.Stop()

	// 2. Execution
	updatedQuantity := 5
	updatedProbability := int64(50)
	err = sys.UpdateItem("item1", updatedQuantity, updatedProbability)
	require.NoError(t, err)

	// 3. Assertions
	// Check that the pool state is updated
	state := sys.State()
	require.Len(t, state, 1)
	assert.Equal(t, "item1", state[0].ItemID)
	assert.Equal(t, updatedQuantity, state[0].Quantity)
	assert.Equal(t, updatedProbability, state[0].Probability)

	// Check that the WAL has logged the update
	require.Len(t, wal.logged, 1)
	updateLog, ok := wal.logged[0].(*types.WalLogUpdateItem)
	require.True(t, ok, "Logged item is not an update log")
	assert.Equal(t, types.LogTypeUpdate, updateLog.Type)
	assert.Equal(t, "item1", updateLog.ItemID)
	assert.Equal(t, updatedQuantity, updateLog.Quantity)
	assert.Equal(t, updatedProbability, updateLog.Probability)
}
