
# Task 12: Road to Replica-Sync

## Target

- Restore `requestId` to make it unique and always increasing.
- The `requestId` should be restored from WAL logs and snapshots.
- The `actor` should own the `requestId` and manage its restoration.

## Plan

### Iteration 1: Implement persistent request ID

1.  **`internal/types/types.go`**
    -   Move the `internal/rewardpool/pool.go` poolSnapshot into types
    -   Add `LastRequestID uint64` to the `poolSnapshot` struct in `internal/rewardpool/pool.go` (or a similar central types location if more appropriate).

2.  **`internal/rewardpool/pool.go`**
    -   Modify `SaveSnapshot` to return the partial of `poolSnapshot` -> actore can append the requestId
    -   Modify `LoadSnapshot` load partial of `poolSnapshot`.

3.  **`internal/actor/actor.go` & `internal/actor/system.go`**
    -   Add a `GetRequestID() uint64` method to `RewardProcessorActor` and `System`.
    -   Add a `SetRequestID(id uint64)` method to `RewardProcessorActor` and `System`.

4.  **`internal/recovery/recovery.go`**
    -   Update `RecoverPool` to return the restored pool and the last request ID.
    -   In `RecoverPool`, after loading the snapshot, get the `LastRequestID`.
    -   During WAL replay, track the maximum `RequestID` from `WalLogDrawItem` entries.
    -   The final `requestID` will be the maximum of the ID from the snapshot and the one from the WAL replay.

5.  **`cmd/cli/main.go`**
    -   Update the main function to get the last request ID from `RecoverPool`.
    -   Create the `actor.System` and then set the restored request ID on it using the new `SetRequestID` method.

6.  **Testing**
    -   Make sure fix all compile error passed `make check`
    -   Make sure existing test passed `make test`
    -   Create a new test file `internal/actor/actor_restore_request_id_test.go` to test the complete flow:
        1. Create a pool, draw some items (which generates request IDs).
        2. Stop the actor system (which saves a snapshot).
        3. Recover the pool and get the last request ID.
        4. Create a new actor system and set the request ID.
        5. Draw again and verify the new request IDs continue from the restored value.

## Result

I have implemented the persistent request ID feature.

- Moved `poolSnapshot` to `types.go` and added `LastRequestID`.
- Modified `rewardpool` to create snapshots without writing them to disk.
- The `actor` now owns the `requestID` and includes it in snapshots.
- `RecoverPool` now restores the `requestID` from snapshots and WAL files.
- The CLI now restores the `requestID` when starting the actor system.
- All existing tests are passing, and a new test for request ID restoration has been added and is passing.

### Iteration 2: Create a WAL streaming module

#### Plan

1.  **Create a new package `internal/walstream`:**
    -   This will contain the logic for the WAL streaming client.

2.  **Define the `WALStreamer` interface in `internal/walstream/streamer.go`:**
    -   This interface will define the contract for streaming WAL logs.
    -   It should have a method like `Stream(log *types.WalLogEntry) error`.
    -   The implementation should be non-blocking.

3.  **Create a `NoOpStreamer` in `internal/walstream/noop_streamer.go`:**
    -   This will be the default implementation that does nothing.
    -   This will be used when streaming is not configured.

4.  **Create a `LogStreamer` in `internal/walstream/log_streamer.go`:**
    -   This will be a simple implementation that logs the WAL entries using the standard logger.
    -   This is for testing and demonstration purposes.

5.  **Integrate `WALStreamer` into the `actor.RewardProcessorActor`:**
    -   Add a `walStreamer types.WALStreamer` field to the `RewardProcessorActor` struct.
    -   Modify the `flush()` method:
        -   After a successful `a.ctx.WAL.Flush()`, iterate through `a.pendingLogs`.
        -   For each `logEntry` in `a.pendingLogs`, call `a.walStreamer.Stream(logEntry)`.
        -   This streaming call should be done in a non-blocking way, potentially by sending the `logEntry` to a channel that a separate goroutine consumes and streams.
    -   The `replayAndRelog` function will re-apply and re-log the pending operations. These re-logged operations will then be flushed and streamed as part of the normal `flush()` process, avoiding duplicate streaming.

6.  **Update `cmd/cli/main.go` to demonstrate the usage:**
    -   Add a command-line flag to enable/disable WAL streaming.
    -   If enabled, create a `LogStreamer` and pass it to the `actor.NewSystem`.

7.  **Testing:**
    -   Make sure fix all compile error passed `make check`
    -   Make sure existing test passed `make test`
    -   Create unit tests for the `LogStreamer`.
    -   Create a mock `WALStreamer` to test the integration with the `actor.System`.

### Iteration 3: Refactor WAL Streaming with a Dedicated Actor

#### Plan

1.  **Create `internal/actor/streamer_actor.go`:**
    *   Define a new `StreamingActor` struct.
    *   This actor will manage the `WALStreamer` and process log entries from a dedicated mailbox channel.

2.  **Refactor `RewardProcessorActor` (`internal/actor/actor.go`):**
    *   Remove the `walStreamer` field.
    *   Add a channel to send `WalLogEntry` messages to the `StreamingActor`.
    *   Update the `flush()` method to send logs to this channel instead of calling the streamer directly.

3.  **Update `actor.System` (`internal/actor/system.go`):**
    *   The `System` will now create and manage both the `RewardProcessorActor` and the new `StreamingActor`.
    *   It will create the channel to communicate between the two actors.
    *   The `Stop()` method will be updated to gracefully shut down both actors.

4.  **Update Tests:**
    *   Make sure fix all compile error passed `make check`
    *   Make sure existing test passed `make test`
    *   Adjust the WAL streaming tests to reflect the new asynchronous, two-actor architecture.
