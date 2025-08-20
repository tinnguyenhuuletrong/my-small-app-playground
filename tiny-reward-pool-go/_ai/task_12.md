
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
