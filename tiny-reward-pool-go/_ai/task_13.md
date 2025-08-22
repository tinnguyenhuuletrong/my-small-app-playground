# Task 13: Service Evolution - Config, REPL, and gRPC

## Target
Evolve the application from a simple demo into a configurable service. This involves moving to a YAML-based configuration, adding an interactive REPL for administration, and exposing the core functionality via a gRPC API.

## Iteration 1: Archive Old CLI and Implement YAML Configuration

### Plan
1.  **Archive Old CLI**: Rename the directory `cmd/cli` to `cmd/cli_v1`.
2.  **Create New CLI Directory**: Create a new, empty `cmd/cli` directory.
3.  **Add Dependency**: Add a YAML parsing library (`gopkg.in/yaml.v3`) to the `go.mod` file.
4.  **Define `config.yaml`**: Create a `config.yaml` file in the `samples` directory to define settings like `working_dir`, `ConfigPool` configurable in yaml, and WAL parameters.
5.  **Implement Config Loader**: Update the `internal/config` package to load, parse, and validate the `config.yaml` file.
6.  **Create New `main.go`**: Create a new `cmd/cli/main.go` that uses the new configuration loader. For now, it will just load the config and print it to confirm it's working.

### Result
- Archived the old CLI to `cmd/cli_v1`.
- Created a new `cmd/cli` directory.
- Added `gopkg.in/yaml.v3` to `go.mod`.
- Created `samples/config.yaml` with the new configuration structure.
- Implemented a new YAML config loader in the `internal/config` package without introducing breaking changes.
- Created a new `main.go` in `cmd/cli` that successfully loads and prints the configuration from `samples/config.yaml`.
- Added `github.com/charmbracelet/bubbletea` to `go.mod`.
- Created a new `tui` package inside `cmd/cli`.
- Defined a basic TUI model with `Init`, `Update`, and `View` functions.
- Integrated the TUI into `main.go`, which now launches the Bubble Tea application.
- The application successfully displays a "Hello, World!" message and can be quit with 'q' or 'ctrl+c'.

## Iteration 2: Implement Interactive TUI with Commands

### Plan
1.  **Enhance TUI Model**:
    *   Add a `textinput` component for user command entry.
    *   Use a `viewport` to display logs and command output.
    *   Structure the UI with a help view, status view, and input area.
2.  **Command Handling**:
    *   Implement input handling to capture user commands.
    *   Create a simple command parser for commands like `h` (help), `s` (status), `d` (draw), `u` (update), `p` (print pool), `r` (reload), and `q` (quit).
3.  **Implement Core Commands**:
    *   **Help (`h`)**: Display a list of available commands and their usage.
    *   **Status (`s`)**: Show the current status of the reward pool actor.
    *   **Draw (`d`)**: Trigger a draw from the reward pool.
    *   **Update (`u <id> <quantity> <weight>`)**: Update an item's quantity and weight.
    *   **Reload (`r`)**: Reload the reward pool from the configuration.
4.  **Actor Integration**:
    *   Connect the TUI to the backend actor using channels.
    *   Use a `ChannelWriter` to stream actor logs and responses to the TUI's viewport.
    *   Send commands from the TUI to the actor for processing.
5.  **Refine `main.go`**:
    *   Update `main.go` to initialize the actor system and the TUI, and wire them together.
    *   Ensure graceful shutdown of the actor system when the TUI exits.
6.  **Testing:**
    -   Make sure fix all compile error passed `make check`
    -   Make sure existing test passed `make test`

## Iteration 3 - TUI Improvement - Live Bar Chart

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