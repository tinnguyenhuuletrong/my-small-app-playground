# Task 16: Implement Raft-based Reward Pool Service

## Target

- Create a robust Raft-based service for the reward pool using `dragonboat`.
- The service will expose methods for drawing rewards and querying the pool state.
- The Raft leader will serve write requests (draws, updates), while any node can serve read-only requests.
- This implementation will use Raft's built-in replication, making the separate `wal-stream` module unnecessary for replica synchronization in this context.

---

## Iter 1

### Plan

1.  **Refine the `Node` struct in `pkg/raft-service/raft.go`**:
    *   Add methods to the `Node` struct to expose a clean API for interacting with the Raft cluster.
    *   `Draw(ctx context.Context, itemID string, quantity int) error`: This method will serialize a `WalLogDrawItem` and propose it to the Raft group using `nh.SyncPropose`.
    *   `Update(ctx context.Context, itemID string, quantity int, probability int64) error`: This will serialize a `WalLogUpdateItem` and propose it.
    *   `GetState(ctx context.Context) ([]types.PoolReward, error)`: This will use `nh.SyncRead` to get the current state from the state machine.

2.  **Enhance `NewNode` in `pkg/raft-service/raft.go`**:
    *   Modify `NewNode` to be more flexible. It should take the `NodeHostConfig` and `Config` as parameters rather than hardcoding them. This will allow for easier testing and configuration.
    *   The `datadir` should be constructed based on the replica ID to avoid conflicts when running multiple nodes locally for testing. The `helloworld` `dragonboat` example shows a good way to do this (`example-data/nodex`).

3.  **Create a Multi-Node Test in `pkg/raft-service/raft_test.go`**:
    *   Create a new test, e.g., `TestRaftCluster_ThreeNodes`, that sets up a 3-node Raft cluster.
    *   Each node will need a unique `ReplicaID`, `RaftAddress`, and data directory.
    *   The test will:
        a. Start three `Node` instances.
        b. Wait for a leader to be elected. `nh.GetLeaderID(shardID)` can be used for this, possibly in a retry loop.
        c. Connect to the leader node.
        d. Propose an `Update` using the new `Node.Update` method on the leader.
        e. Use `Node.GetState` on all three nodes to verify that the state was replicated and is consistent across the cluster.

4.  **Update `RewardPoolStateMachine` in `pkg/raft-service/raft.go`**:
    *   The `Update` method currently handles `LogTypeDraw` and `LogTypeUpdate`. This is sufficient for the first iteration.
    *   The `Lookup` method currently returns the entire pool state. This is fine for now.

5.  **Build and Verify (`make check` and `make test`)**:
    *   Run `make check` to ensure no compilation errors.
    *   Run `make test` to run the existing and new unit tests to validate the implementation.

### Result
...

### Problem
...
