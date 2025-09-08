package raft_service

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"
)

func TestRaftNode_ProposeAndApply(t *testing.T) {
	// 1. Setup
	// Clean up the temp directories
	defer os.RemoveAll("wal")
	defer os.RemoveAll("dragonboat")

	replicaID := uint64(1)
	raftAddress := "localhost:9090"
	initialMembers := map[uint64]string{1: raftAddress}

	// 2. Start the raft node
	node, err := NewNode(replicaID, raftAddress, initialMembers)
	require.NoError(t, err)
	defer node.nh.Close()

	// 3. Propose an update to initialize the state
	updateLog := &types.WalLogUpdateItem{
		WalLogEntryBase: types.WalLogEntryBase{Type: types.LogTypeUpdate},
		ItemID:          "item1",
		Quantity:        10,
		Probability:     100,
	}
	updateData, err := json.Marshal(updateLog)
	require.NoError(t, err)

	cs := node.nh.GetNoOPSession(1)

	// Retry loop for proposal
	for i := 0; i < 10; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		_, err = node.nh.SyncPropose(ctx, cs, updateData)
		cancel()
		if err == nil {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	require.NoError(t, err)

	// 5. Verify that the log entry was applied
	// Use SyncRead to get the state from the state machine
	readCtx, readCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer readCancel()

	result, err := node.nh.SyncRead(readCtx, 1, nil)
	require.NoError(t, err)

	t.Logf("State from SyncRead: %s", string(result.([]byte)))
	var state []types.PoolReward
	err = json.Unmarshal(result.([]byte), &state)
	require.NoError(t, err)

	// Find the item and check its quantity
	require.Len(t, state, 1)
	require.Equal(t, "item1", state[0].ItemID)
	require.Equal(t, 10, state[0].Quantity)
}
