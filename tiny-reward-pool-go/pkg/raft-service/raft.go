package raft_service

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/lni/dragonboat/v4"
	"github.com/lni/dragonboat/v4/config"
	"github.com/lni/dragonboat/v4/statemachine"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/replay"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/rewardpool"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"
)

// RewardPoolStateMachine is the state machine for the reward pool.
type RewardPoolStateMachine struct {
	ShardID   uint64
	ReplicaID uint64
	pool      *rewardpool.Pool
}

// NewRewardPoolStateMachine creates a new RewardPoolStateMachine.
func NewRewardPoolStateMachine(shardID uint64, replicaID uint64) statemachine.IStateMachine {
	return &RewardPoolStateMachine{
		ShardID:   shardID,
		ReplicaID: replicaID,
		pool:      rewardpool.NewPool(nil),
	}
}

// Update applies the committed log entries to the state machine.
func (s *RewardPoolStateMachine) Update(entry statemachine.Entry) (statemachine.Result, error) {
	var base types.WalLogEntryBase
	if err := json.Unmarshal(entry.Cmd, &base); err != nil {
		return statemachine.Result{}, err
	}

	var logEntry types.WalLogEntry
	switch base.Type {
	case types.LogTypeDraw:
		var drawLog types.WalLogDrawItem
		if err := json.Unmarshal(entry.Cmd, &drawLog); err != nil {
			return statemachine.Result{}, err
		}
		logEntry = &drawLog
	case types.LogTypeUpdate:
		var updateLog types.WalLogUpdateItem
		if err := json.Unmarshal(entry.Cmd, &updateLog); err != nil {
			return statemachine.Result{}, err
		}
		logEntry = &updateLog
	default:
		return statemachine.Result{Value: 0}, nil
	}

	fmt.Printf("Updating state machine with entry: %+v\n", entry)
	replay.ApplyLog(s.pool, logEntry)
	return statemachine.Result{Value: uint64(len(entry.Cmd))}, nil
}

// Lookup performs a read-only query on the state machine.
func (s *RewardPoolStateMachine) Lookup(query interface{}) (interface{}, error) {
	fmt.Printf("Looking up state: %+v\n", s.pool.State())
	state := s.pool.State()
	data, err := json.Marshal(state)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// SaveSnapshot creates a snapshot of the state machine.
func (s *RewardPoolStateMachine) SaveSnapshot(w io.Writer, fc statemachine.ISnapshotFileCollection, done <-chan struct{}) error {
	snap, err := s.pool.CreateSnapshot()
	if err != nil {
		return err
	}

	data, err := json.Marshal(snap)
	if err != nil {
		return err
	}

	_, err = w.Write(data)
	return err
}

// RecoverFromSnapshot restores the state machine from a snapshot.
func (s *RewardPoolStateMachine) RecoverFromSnapshot(r io.Reader, files []statemachine.SnapshotFile, done <-chan struct{}) error {
	var snap types.PoolSnapshot
	if err := json.NewDecoder(r).Decode(&snap); err != nil {
		return err
	}

	return s.pool.LoadSnapshot(&snap)
}

// Close closes the state machine.
func (s *RewardPoolStateMachine) Close() error {
	return nil
}

// Node is a wrapper around the dragonboat NodeHost.
type Node struct {
	nh *dragonboat.NodeHost
}

// NewNode creates and starts a new dragonboat node.
func NewNode(replicaID uint64, raftAddress string, initialMembers map[uint64]string) (*Node, error) {
	rc := config.Config{
		ReplicaID:          replicaID,
		ShardID:            1,
		ElectionRTT:        10,
		HeartbeatRTT:       1,
		CheckQuorum:        true,
		SnapshotEntries:    10000,
		CompactionOverhead: 5000,
	}

	nhc := config.NodeHostConfig{
		WALDir:         "wal",
		NodeHostDir:    "dragonboat",
		RaftAddress:    raftAddress,
		RTTMillisecond: 200,
	}

	nh, err := dragonboat.NewNodeHost(nhc)
	if err != nil {
		return nil, err
	}

	createStateMachine := func(shardID uint64, replicaID uint64) statemachine.IStateMachine {
		sm := NewRewardPoolStateMachine(shardID, replicaID)
		fmt.Printf("Created state machine at address: %p\n", sm)
		return sm
	}

	if err := nh.StartReplica(initialMembers, false, createStateMachine, rc); err != nil {
		return nil, err
	}

	return &Node{nh: nh}, nil
}
