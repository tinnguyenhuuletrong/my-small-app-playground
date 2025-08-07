package processing_test

import (
	"errors"
	"testing"

	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/processing"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/utils"
)

func TestProcessor_TransactionalDraw(t *testing.T) {
	pool := &mockPool{item: types.PoolReward{ItemID: "gold", Quantity: 1, Probability: 1.0}}
	wal := &mockWAL{}
	utils := &utils.UtilsImpl{}
	ctx := &types.Context{WAL: wal, Utils: utils}
	proc := processing.NewProcessor(ctx, pool, &processing.ProcessorOptional{FlushAfterNDraw: 1})

	// Success path
	done := make(chan struct{})
	var gotResp processing.DrawResponse
	reqID := proc.Draw(func(resp processing.DrawResponse) {
		gotResp = resp
		close(done)
	})
	<-done
	if gotResp.RequestID != reqID {
		t.Fatalf("Expected requestID %d, got %d", reqID, gotResp.RequestID)
	}
	if gotResp.Item == nil || gotResp.Item.ItemID != "gold" {
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
	done2 := make(chan struct{})
	var gotResp2 processing.DrawResponse
	_ = proc.Draw(func(resp processing.DrawResponse) {
		gotResp2 = resp
		close(done2)
	})
	<-done2
	if gotResp2.Item != nil {
		t.Fatalf("Expected nil item on WAL failure, got %v", gotResp2.Item)
	}
}

func TestProcessor_FlushAndFlushAfterNDraw(t *testing.T) {
	// Test case 1: Flush after N draws
	pool := &mockPool{item: types.PoolReward{ItemID: "gold", Quantity: 10, Probability: 1.0}}
	wal := &mockWAL{}
	utils := &utils.UtilsImpl{}
	ctx := &types.Context{WAL: wal, Utils: utils}
	flushN := 3
	proc := processing.NewProcessor(ctx, pool, &processing.ProcessorOptional{FlushAfterNDraw: flushN})

	// Perform N-1 draws, flush should not be called
	for i := 0; i < flushN-1; i++ {
		done := make(chan struct{})
		proc.Draw(func(resp processing.DrawResponse) {
			close(done)
		})
		<-done
	}
	if wal.flushCount != 0 {
		t.Fatalf("Expected flushCount=0 after %d draws, got %d", flushN-1, wal.flushCount)
	}
	if pool.committed != 0 {
		t.Fatalf("Expected committed=0 after %d draws, got %d", flushN-1, pool.committed)
	}

	// Perform the Nth draw, flush should be called
	done := make(chan struct{})
	proc.Draw(func(resp processing.DrawResponse) {
		close(done)
	})
	<-done
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

	done2 := make(chan struct{})
	proc.Draw(func(resp processing.DrawResponse) {
		close(done2)
	})
	<-done2
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

	done3 := make(chan struct{})
	proc.Draw(func(resp processing.DrawResponse) {
		close(done3)
	})
	<-done3
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

	done4 := make(chan struct{})
	proc.Draw(func(resp processing.DrawResponse) {
		close(done4)
	})
	<-done4
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

func (m *mockPool) SelectItem(ctx *types.Context) (*types.PoolReward, error) {
	if m.item.Quantity-len(m.pending) > 0 {
		copyItem := m.item
		m.pending = append(m.pending, copyItem.ItemID)
		return &copyItem, nil
	}
	return nil, nil
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
	logged     []types.WalLogItem
	fail       bool
	flushCount int
	flushFail  bool
}

func (m *mockWAL) LogDraw(item types.WalLogItem) error {
	m.logged = append(m.logged, item)
	if m.fail {
		return errors.New("simulated WAL error")
	}
	return nil
}
func (m *mockWAL) Close() error { return nil }
func (m *mockWAL) Flush() error {
	m.flushCount++
	if m.flushFail {
		return errors.New("simulated WAL flush error")
	}
	return nil
}
func (m *mockWAL) SetSnapshotPath(path string) {}
