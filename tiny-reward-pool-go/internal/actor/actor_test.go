package actor_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/actor"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/rewardpool"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/utils"
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

func (m *mockPool) ApplyDrawLog(itemID string) {
	m.item.Quantity--
}

func (m *mockPool) Load(cfg types.ConfigPool) error                               { return nil }
func (m *mockPool) LoadSnapshot(snapshot *types.PoolSnapshot) error             { return nil }
func (m *mockPool) CreateSnapshot() (*types.PoolSnapshot, error)                  { return &types.PoolSnapshot{}, nil }
func (m *mockPool) ApplyUpdateLog(itemID string, quantity int, probability int64) {}
func (m *mockPool) UpdateItem(itemID string, quantity int, probability int64) error {
	m.item.Quantity = quantity
	m.item.Probability = probability
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

func (m *mockWAL) Close() error { return nil }
func (m *mockWAL) Reset()       {}

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