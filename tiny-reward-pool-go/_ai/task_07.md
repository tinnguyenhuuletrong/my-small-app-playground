# Task 07: Implement Prefix Sum Selector

## Target
Implement a new `ItemSelector` called `PrefixSumSelector` that uses a prefix sum array for weighted random item selection. This selector should adhere to the `types.ItemSelector` interface.

## Problem (from previous iteration, if any)
N/A

## Iter 01

### Plan
1.  **Create `internal/selector/prefix_sum_selector.go`:**
    *   Define the `PrefixSumSelector` struct, including fields for the prefix sum array, item IDs, item index map, `totalAvailable`, and an injectable `*rand.Rand` instance.
    *   Implement `NewPrefixSumSelector()` to initialize the struct and the `rand` field.
    *   Implement the `Reset(catalog []types.PoolReward)` method:
        *   Clear existing data.
        *   Initialize the prefix sum array based on the catalog.
        *   Populate the `itemIDs` and `itemIndex` maps.
        *   Calculate `totalAvailable`.
    *   Implement the `Select(ctx *types.Context)` method:
        *   Handle `types.ErrEmptyRewardPool` if `totalAvailable` is zero.
        *   Generate a random value using the injectable `rand` instance within `totalAvailable`.
        *   Use binary search (or a similar efficient method) on the prefix sum array to find the selected item's index.
        *   Return the `ItemID` of the selected item.
    *   Implement the `Update(itemID string, quantity int64)` method:
        *   Find the item's index.
        *   Update the prefix sum array and `totalAvailable` accordingly.
    *   Implement `TotalAvailable()` to return `totalAvailable`.
    *   Implement `GetItemRemaining(itemID string)` to return the remaining quantity of a specific item.

2.  **Create `internal/selector/prefix_sum_selector_test.go`:**
    *   Add comprehensive unit tests for `NewPrefixSumSelector`, `Reset`, `Select`, `Update`, `TotalAvailable`, and `GetItemRemaining`.
    *   For `Select` tests, use a mock random source for deterministic testing and statistical checks for distribution.
    *   Include tests for edge cases (empty catalog, updating non-existent items, items with zero quantity).
    *   Add an integration test that simulates drawing items and updating quantities, similar to `FenwickTreeSelector_IntegrationWithDraw`.

3.  **Update Documentation:**
    *   Update `_ai/doc/agent_note.md` to mention the new `PrefixSumSelector` as an alternative `ItemSelector` implementation.
    *   Update `GEMINI.md` to mention the new `PrefixSumSelector` in the context of `ItemSelector` implementations.

4.  **Verification:**
    *   Run all project tests (`make test`) to ensure no regressions and that the new tests pass.

### Result
All unit tests for `PrefixSumSelector` passed, including `NewPrefixSumSelector`, `Reset`, `Select`, `Update`, `TotalAvailable`, `GetItemRemaining`, and the integration test. The `Update` method now correctly interprets the `quantity` parameter as a delta, aligning with the `ItemSelector` interface and `FenwickTreeSelector` implementation. Documentation in `_ai/doc/agent_note.md` and `GEMINI.md` has been updated to reflect the new selector.

### Problem
None. Iteration 01 is complete.