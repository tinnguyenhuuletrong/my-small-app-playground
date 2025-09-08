
# Task 16: Create pkg/raft-service

## Target
- Create a raft base services for poolReward
- Master service serve the request, replica service keep synchonize via wal-stream

## Iter 1

### Plan

1.  **Research and Scaffolding:**
    *   Read the `dragonboat` documentation and examples to understand the core concepts.
    *   Create a new package `pkg/raft-service`.
    *   Define the `RewardPoolStateMachine` struct that will implement `dragonboat.IStateMachine`. This struct will encapsulate the `rewardpool.Pool`.

2.  **Implement `IStateMachine`:**
    *   Implement the `Update(entries []dragonboat.Entry) ([]dragonboat.Entry, error)` method. This method will be responsible for applying the committed log entries to the `rewardpool.Pool`. The log entries will be our serialized `types.WalLogEntry`.
    *   Implement the `Lookup(query interface{}) (interface{}, error)` method. This will be used for read-only queries on the state machine.
    *   Implement the `SaveSnapshot(w io.Writer, fc sm.ISnapshotFileCollection, done <-chan struct{}) error` method. This will create a snapshot of the `rewardpool.Pool` state.
    *   Implement the `RecoverFromSnapshot(r io.Reader, files []sm.SnapshotFile, done <-chan struct{}) error` method. This will restore the `rewardpool.Pool` state from a snapshot.
    *   Implement the `Close() error` method.

3.  **Raft Node Management:**
    *   Create a `Node` struct that will encapsulate the `dragonboat.NodeHost` and the `RewardPoolStateMachine`.
    *   Create a `NewNode(...)` function to initialize and start a `dragonboat` node. This function will handle the configuration, state machine creation, and starting the `NodeHost`.

4.  **Basic Testing:**
    *   Create a test file `pkg/raft-service/raft_test.go`.
    *   Write a test to start a single-node raft cluster.
    *   Write a test to propose a log entry (e.g., a `WalLogDrawItem`) and verify that it is applied to the state machine.

### Result

...

### Problem

...

---

## Iter 2

### Plan

1.  **Integrate Raft into the Actor:**
    *   Modify `RewardProcessorActor` to hold a reference to the `raft.Node`.
    *   In `handleDraw` and `handleUpdate`, instead of calling `a.ctx.WAL.LogDraw` and `a.ctx.WAL.LogUpdate`, the actor will serialize the `WalLogEntry` and propose it to the raft cluster using `node.Propose()`.
    *   The proposal will be asynchronous. The actor will need to handle the response from the proposal, which will indicate whether the log entry was committed.

2.  **Remove old WAL logic:**
    *   The `flush`, `handleWALFull`, and `replayAndRelog` methods in the actor will be removed or significantly simplified, as `dragonboat` will handle log persistence and replication.
    *   The `wal.WAL` struct and its related components might be deprecated or repurposed.

3.  **Update gRPC Service:**
    *   Modify the `RewardPoolService` to interact with the raft node.
    *   The `Draw` and `UpdateItem` methods will now trigger proposals to the raft cluster.
    *   The `GetState` method will use `node.Lookup()` to perform a linearizable read of the state machine.

4.  **Configuration:**
    *   Update the `config.yaml` to include raft-related configuration, such as the list of nodes in the cluster, the raft address, etc.

5.  **Testing:**
    *   Update existing tests to work with the new raft-based implementation.
    *   Write new tests for the multi-node scenario, verifying that the state is correctly replicated between nodes.

### Result

...

### Problem

...
