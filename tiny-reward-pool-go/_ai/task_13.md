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
[AI_TODO]

## Iteration 2: Interactive TUI with Bubble Tea

### Plan
1.  **Add Dependency**: Add `github.com/charmbracelet/bubbletea` to `go.mod`.
2.  **Define TUI Model**: Create a `model` struct to hold the TUI's state (e.g., input fields, status panes, logs).
3.  **Implement `Init`, `Update`, `View`**: Structure the TUI application according to The Elm Architecture (Model-View-Update) provided by `bubbletea`. The `Update` function will handle both user input and asynchronous messages from our actor system.
4.  **Integrate in `main.go`**: The `main` function will initialize and run the `bubbletea` program, passing it a reference to the actor system.

### Result
[AI_TODO]

## Iteration 3: gRPC Service API

### Plan
1.  **Add Dependencies**: Add `google.golang.org/grpc` and `google.golang.org/protobuf` to `go.mod`.
2.  **Define Protobuf API**: Create an `api/reward.proto` file to define the `RewardService` with RPCs for `Draw`, `UpdateItem`, and `GetStatus`.
3.  **Generate Go Code**: Use `protoc` to generate the necessary Go gRPC server and client code from the `.proto` file.

### Result
[AI_TODO]

## Iteration 4: gRPC Server Implementation

### Plan
1.  **Implement Server Logic**: Create a new `internal/grpc` package. Implement the generated `RewardServiceServer` interface. The server will translate incoming gRPC requests into messages for the actor system and return the responses.
2.  **Integrate Server**: In `main.go`, add logic (e.g., controlled by a config value) to start the gRPC server in a separate goroutine.
3.  **Graceful Shutdown**: Ensure the gRPC server is shut down gracefully when the application exits.

### Result
[AI_TODO]
