# Task 13: TUI Improvement - Live Bar Chart

## Iteration 3

### Problem
The current TUI main view only shows a simple history of commands and their results. It doesn't provide a live, visual representation of the state of the reward pool, which would be very useful for monitoring.

### Target
Modify the main panel of the TUI to display a bar chart showing the remaining quantity of each item in the reward pool. The display should update automatically after each draw operation.

### Plan

1.  **Refactor `tui.Model` for a new layout:**
    *   Rename the existing `viewport` to `chartView` which will be used to display the bar chart.
    *   Add a new `viewport.Model` named `historyView` to manage and display the command history.
    *   Add a map `initialQuantities map[string]int` to the model to store the initial quantities of each item for calculating bar chart percentages.

2.  **Update `NewModel()` constructor:**
    *   Initialize both `chartView` and `historyView`.
    *   Fetch the initial state from `system.State()` and populate the `initialQuantities` map.

3.  **Modify the `View()` method:**
    *   The main `View()` will be restructured to render the new layout.
    *   Create a new `renderChartView()` method:
        *   It will fetch the current pool state via `m.system.State()`.
        *   For each item, it will calculate the percentage of remaining quantity against the `initialQuantities`.
        *   It will generate and return an ASCII bar chart string.
    *   The main `View()` will now compose the `headerView`, the new chart view, the history view, the `footerView`, and the debug view.

4.  **Adjust the `Update()` method:**
    *   On `concurrentDrawsFinishedMsg`, the message handler will now update the `historyView` with the draw results instead of the main viewport.
    *   The `chartView` will be implicitly updated on every `View()` call, ensuring it's always in sync with the latest state.

5.  **Handle Window Resizing in `onResize()`:**
    *   Update the dimensions of both `chartView` and `historyView` when the terminal window is resized to ensure the layout doesn't break.

### UI Mockup

```
+--------------------------------------------------------------------------------------------------+
| Reward Pool TUI                                                    Request ID: 1147              |
+--------------------------------------------------------------------------------------------------+
|                                                                                                  |
| Item Quantities:                                                                                 |
| diamond: [██████████████████████████████████████████████████] 100/100                              |
| gold:    [██████████████████████████████████████████████████] 100/100                              |
| silver:  [████████████████████████████████████████████████  ] 98/100                               |
| rock:    [██████████████████████████████████████████████████] 100/100                              |
|                                                                                                  |
+--------------------------------------------------------------------------------------------------+
| Command History:                                                                                 |
| > d 1                                                                                            |
| [Request 1145] You drew: diamond                                                                 |
| > d 2                                                                                            |
| [Request 1146] You drew: silver                                                                  |
| [Request 1147] You drew: silver                                                                  |
|                                                                                                  |
+--------------------------------------------------------------------------------------------------+
| > Enter command...                                                                               |
+--------------------------------------------------------------------------------------------------+
| Debug Log:                                                                                       |
| time=... level=INFO msg="WAL is empty..."                                                        |
| ...                                                                                              |
+--------------------------------------------------------------------------------------------------+
```