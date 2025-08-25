package rewardpool_grpc_service_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/actor"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"
	generated "github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/pkg/rewardpool-grpc-service"
	grpc_service "github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/pkg/rewardpool-grpc-service"
)

type mockActorSystem struct{}

func (m *mockActorSystem) State() []types.PoolReward {
	return []types.PoolReward{
		{ItemID: "gold", Quantity: 10, Probability: 1},
		{ItemID: "silver", Quantity: 20, Probability: 2},
	}
}

func (m *mockActorSystem) Draw() <-chan actor.DrawResponse {
	return nil
}

func (m *mockActorSystem) Stop() {}

func (m *mockActorSystem) UpdateItem(id string, quantity int, weight int64) error {
	return nil
}

func (m *mockActorSystem) GetRequestID() uint64 {
	return 0
}

func (m *mockActorSystem) SetRequestID(id uint64) {}

func TestRewardPoolService_GetState(t *testing.T) {
	// 1. Setup
	mockSystem := &mockActorSystem{}
	service := grpc_service.NewRewardPoolService(mockSystem)

	// 2. Execution
	resp, err := service.GetState(context.Background(), &generated.GetStateRequest{})

	// 3. Assertions
	require.NoError(t, err)
	require.NotNil(t, resp)
	expectedState := mockSystem.State()
	require.Len(t, resp.Items, len(expectedState))

	for i, expectedItem := range expectedState {
		actualItem := resp.Items[i]
		assert.Equal(t, expectedItem.ItemID, actualItem.ItemId)
		assert.Equal(t, int32(expectedItem.Quantity), actualItem.Quantity)
		assert.Equal(t, expectedItem.Probability, actualItem.Probability)
	}
}
