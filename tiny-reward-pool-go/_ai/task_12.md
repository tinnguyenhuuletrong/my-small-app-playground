
# Task 12: Restore Request ID

## Target

- Restore `requestId` to make it unique and always increasing.
- The `requestId` should be restored from WAL logs and snapshots.
- The `actor` should own the `requestId` and manage its restoration.

## Plan

### Iteration 1: Implement persistent request ID

1.  **`internal/types/types.go`**
    -   Add `LastRequestID uint64` to the `poolSnapshot` struct in `internal/rewardpool/pool.go` (or a similar central types location if more appropriate).

2.  **`internal/rewardpool/pool.go`**
    -   Modify `SaveSnapshot` to accept the last request ID as an argument and save it in the snapshot.
    -   Modify `LoadSnapshot` to return the `LastRequestID` from the snapshot.

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
    -   Create a new test file `internal/actor/actor_restore_request_id_test.go` to test the complete flow:
        1. Create a pool, draw some items (which generates request IDs).
        2. Stop the actor system (which saves a snapshot).
        3. Recover the pool and get the last request ID.
        4. Create a new actor system and set the request ID.
        5. Draw again and verify the new request IDs continue from the restored value.
