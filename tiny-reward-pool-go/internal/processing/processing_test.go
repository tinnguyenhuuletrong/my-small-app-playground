package processing_test

import (
	"testing"

	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/processing"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/utils"
)

type mockPool struct {
	item   types.PoolReward
	called bool
}

func (m *mockPool) Draw(ctx *types.Context) (*types.PoolReward, error) {
	if m.item.Quantity > 0 && !m.called {
		m.item.Quantity--
		m.called = true
		return &m.item, nil
	}
	return nil, nil
}
func (m *mockPool) Load(cfg types.ConfigPool) error { return nil }

type mockWAL struct {
	logged []types.WalLogItem
}

func (m *mockWAL) LogDraw(item types.WalLogItem) error {
	m.logged = append(m.logged, item)
	return nil
}
func (m *mockWAL) Close() error { return nil }

func TestProcessor_Draw(t *testing.T) {
	pool := &mockPool{item: types.PoolReward{ItemID: "gold", Quantity: 1, Probability: 1.0}}
	wal := &mockWAL{}
	utils := &utils.UtilsImpl{}
	ctx := &types.Context{WAL: wal, Utils: utils}
	proc := processing.NewProcessor(ctx, pool)
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
	proc.Stop()
}
