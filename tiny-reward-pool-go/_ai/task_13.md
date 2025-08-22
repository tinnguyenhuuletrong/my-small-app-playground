# Task 13: TUI Improvement - Live Bar Chart

## Iteration 3

### Problem
The current TUI main view only shows a simple history of commands and their results. It doesn't provide a live, visual representation of the state of the reward pool. Furthermore, fetching state on every render cycle is inefficient and can spam the actor system.

### Target
Modify the main panel of the TUI to display a bar chart showing the remaining quantity of each item in the reward pool. The display should update automatically and efficiently after each operation.

### Plan

1.  **Refactor `tui.Model` for a new layout and caching:**
    *   Rename `viewport` to `chartView`.
    *   Add `historyView viewport.Model`.
    *   Add `initialQuantities map[string]int`.
    *   Add `cachedState []types.PoolReward` to hold the last known state of the pool.
    *   Add `cachedRequestID int64` to hold the last known request ID.
    *   Add a `time.Ticker` to trigger periodic state updates.
    *   Define new messages: `tickMsg` for the ticker, and `refreshStateMsg` to carry the updated state.

2.  **Update `NewModel()` constructor:**
    *   Initialize `chartView` and `historyView`.
    *   Fetch the initial state from `system.State()` and populate `initialQuantities` and `cachedState`.
    *   Fetch the initial request ID and populate `cachedRequestID`.
    *   Initialize and start the `time.Ticker` for periodic refreshes (e.g., every second).

3.  **Modify the `Init()` method:**
    *   Add a command to wait for the ticker's ticks (`waitForTick`).

4.  **Modify the `Update()` method for efficient state handling:**
    *   Handle `tickMsg`: This message will trigger a command to refresh the state from the actor system.
    *   Handle `refreshStateMsg`: This message will update `cachedState` and `cachedRequestID` with the new data from the actor.
    *   On `concurrentDrawsFinishedMsg` (after a draw):
        *   Update the `historyView` with the results.
        *   Send a command to immediately refresh the cached state.
    *   On user commands that modify state (e.g., `u` for update), also trigger an immediate state refresh.

5.  **Optimize the `View()` method:**
    *   The `headerView` will now use the `m.cachedRequestID`.
    *   Create a `renderChartView()` method that uses the `m.cachedState` to build the bar chart string. This prevents calling the actor system during the render phase.
    *   The main `View()` will be restructured to render the new layout using the cached data.

6.  **Handle Window Resizing in `onResize()`:**
    *   Update the dimensions of `chartView`, `historyView`, and other components when the terminal window is resized.

7.  **Ensure Graceful Shutdown:**
    *   Ensure the `ticker.Stop()` is called when the application quits (e.g., on `tea.KeyCtrlC` or `q` command) to prevent resource leaks.

### UI Mockup

```
+--------------------------------------------------------------------------------------------------+
| Reward Pool TUI                                                    Request ID: 1147              |
+--------------------------------------------------------------------------------------------------+
| Item Quantities:                                                                                 |
| diamond: [██████████████████████████████████████████████████] 100/100                              |
| gold:    [██████████████████████████████████████████████████] 100/100                              |
| silver:  [████████████████████████████████████████████████  ] 98/100                               |
| rock:    [██████████████████████████████████████████████████] 100/100                              |
+--------------------------------------------------------------------------------------------------+
| Command History:                                                                                 |
| > d 1                                                                                            |
| [Request 1145] You drew: diamond                                                                 |
| > d 2                                                                                            |
| [Request 1146] You drew: silver                                                                  |
| [Request 1147] You drew: silver                                                                  |
+--------------------------------------------------------------------------------------------------+
| > Enter command...                                                                               |
+--------------------------------------------------------------------------------------------------+
| Debug Log:                                                                                       |
| time=... level=INFO msg="WAL is empty..."                                                        |
+--------------------------------------------------------------------------------------------------+
```