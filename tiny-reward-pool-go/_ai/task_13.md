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
    *   **Print (`p`)**: Print the detailed view of the reward pool.
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