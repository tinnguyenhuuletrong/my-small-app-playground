package raft_service

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/lni/dragonboat/v4"
	"github.com/lni/dragonboat/v4/config"
	"github.com/stretchr/testify/require"
)

func TestRaftCluster_ThreeNodes(t *testing.T) {
	// 1. Setup
	// Clean up the temp directories
	defer os.RemoveAll("wal-1")
	defer os.RemoveAll("dragonboat-1")

	shardID := uint64(1)
	initialMembers := map[uint64]string{
		1: "localhost:9091",
		2: "localhost:9092",
		3: "localhost:9093",
	}

	// Create a single NodeHost that will manage all replicas for this test
	nhc := config.NodeHostConfig{
		WALDir:         "wal-1", // a single WAL dir for the NH
		NodeHostDir:    "dragonboat-1",
		RaftAddress:    "localhost:9091",
		RTTMillisecond: 200,
	}
	nh, err := dragonboat.NewNodeHost(nhc)
	require.NoError(t, err)
	defer nh.Close()

	// Start the replicas on the NodeHost
	for replicaID, raftAddress := range initialMembers {
		rc := config.Config{
			ReplicaID:          replicaID,
			ShardID:            shardID,
			ElectionRTT:        10,
			HeartbeatRTT:       1,
			CheckQuorum:        true,
			SnapshotEntries:    100,
			CompactionOverhead: 50,
		}
		tempNHC := nhc
		tempNHC.RaftAddress = raftAddress
		tempNH, err := dragonboat.NewNodeHost(tempNHC)
		require.NoError(t, err)
		defer tempNH.Close()
		err = tempNH.StartReplica(initialMembers, false, NewRewardPoolStateMachine, rc)
		require.NoError(t, err)
	}

	// Create Node wrappers for interaction
	nodes := make(map[uint64]*Node)
	for replicaID := range initialMembers {
		nodes[replicaID] = &Node{nh: nh, shardID: shardID}
	}

	// 3. Wait for a leader to be elected.
	var leaderID uint64
	require.Eventually(t, func() bool {
		var err error
		var term uint64
		var valid bool
		leaderID, term, valid, err = nodes[1].GetLeaderID()
		return err == nil && valid && leaderID > 0 && term > 0
	}, 20*time.Second, 1*time.Second, "Leader not elected")

	leaderNode := nodes[leaderID]

	// 4. Propose an update to the leader node
	ctxUpdate, cancelUpdate := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelUpdate()
	err = leaderNode.Update(ctxUpdate, "item1", 10, 100)
	require.NoError(t, err)

	// 5. Verify that the state was replicated to all nodes
	require.Eventually(t, func() bool {
		for replicaID, node := range nodes {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			state, err := node.GetState(ctx)
			if err != nil {
				t.Logf("Failed to get state from replica %d: %v", replicaID, err)
				return false
			}
			if len(state) != 1 || state[0].ItemID != "item1" || state[0].Quantity != 10 {
				t.Logf("State not replicated on replica %d. Got: %+v", replicaID, state)
				return false
			}
		}
		return true
	}, 20*time.Second, 1*time.Second, "State not replicated")

	// 6. Propose a draw from a follower node
	followerID := uint64(0)
	for id := range nodes {
		if id != leaderID {
			followerID = id
			break
		}
	}
	require.NotZero(t, followerID)
	followerNode := nodes[followerID]

	ctxDraw, cancelDraw := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelDraw()
	err = followerNode.Draw(ctxDraw, "item1")
	require.NoError(t, err)

	// 7. Verify the draw was applied
	require.Eventually(t, func() bool {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		state, err := leaderNode.GetState(ctx)
		if err != nil {
			return false
		}
		return len(state) == 1 && state[0].Quantity == 9
	}, 20*time.Second, 1*time.Second, "Draw not applied")
}
