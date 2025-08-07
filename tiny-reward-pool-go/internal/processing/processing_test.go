package processing_test

import (
	"errors"
	"testing"

	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/processing"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/utils"
)

type mockPool struct {
	item      types.PoolReward
	staged    int
	committed int
	reverted  int
}

func (m *mockPool) SelectItem(ctx *types.Context) (*types.PoolReward, error) {
	if m.item.Quantity-m.staged > 0 {
		m.staged++
		copyItem := m.item
		return &copyItem, nil
	}
	return nil, nil
}
func (m *mockPool) CommitDraw(itemID string) {
	if m.staged > 0 && m.item.ItemID == itemID {
		m.committed++
		m.item.Quantity--
		m.staged--
	}
}
func (m *mockPool) RevertDraw(itemID string) {
	if m.staged > 0 && m.item.ItemID == itemID {
		m.reverted++
		m.staged--
	}
}
func (m *mockPool) Load(cfg types.ConfigPool) error { return nil }
func (m *mockPool) LoadSnapshot(path string) error  { return nil }
func (m *mockPool) SaveSnapshot(path string) error  { return nil }

type mockWAL struct {
	logged []types.WalLogItem
	fail   bool
}

func (m *mockWAL) LogDraw(item types.WalLogItem) error {
	m.logged = append(m.logged, item)
	if m.fail {
		return errors.New("simulated WAL error")
	}
	return nil
}
func (m *mockWAL) Close() error                { return nil }
func (m *mockWAL) Flush() error                { return nil }
func (m *mockWAL) SetSnapshotPath(path string) {}

func TestProcessor_TransactionalDraw(t *testing.T) {
	pool := &mockPool{item: types.PoolReward{ItemID: "gold", Quantity: 1, Probability: 1.0}}
	wal := &mockWAL{}
	utils := &utils.UtilsImpl{}
	ctx := &types.Context{WAL: wal, Utils: utils}
	proc := processing.NewProcessor(ctx, pool)

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
	if pool.reverted != 1 {
		t.Fatalf("Expected reverted=1, got %d", pool.reverted)
	}
}
