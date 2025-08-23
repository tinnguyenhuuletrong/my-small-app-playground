# Task 14: Add gRPC Service

## Target

- **Scenario**: Add `/pkg/rewardpool-grpc-service`
- **Goal**:
    - Add a grpc service module for actor
    - **Requirements**:
        1. like @cmd/cli/tui/**. New function accept an actor as reference -> maximize flexibility extendable interface
        2. support grpc method State, Draws(batch)
        3. define proto file, version inside module -> we publish it into registry later
        4. update @cmd/cli/** to start a grpc service ( optional yaml config, default is false )

---

## Iter 1

### Plan

1.  **Directory and Proto Definition**:
    - Create a new directory `pkg/rewardpool-grpc-service/proto`.
    - Create a `rewardpool.proto` file inside it.
    - Define the service `RewardPoolService` with two RPCs:
        - `GetState(GetStateRequest) returns (GetStateResponse)`
        - `Draw(DrawRequest) returns (stream DrawResponse)`
    - Define the necessary message types: `RewardItem`, `GetStateRequest`, `GetStateResponse`, `DrawRequest`, `DrawResponse`.

2.  **gRPC Dependencies & Code Generation**:
    - Check `go.mod` for gRPC and protobuf dependencies (`google.golang.org/grpc`, `google.golang.org/protobuf`). If not present, add them using `go get`.
    - Check and update dep tool if need `protoc`, `protoc-gen-go`, and `protoc-gen-go-grpc`.
    - Add a new script/command to the `Makefile` to generate Go code from the `.proto` file. The generated code will be placed in `pkg/rewardpool-grpc-service/`.

3.  **gRPC Service Implementation**:
    - Create `pkg/rewardpool-grpc-service/service.go`.
    - Implement the `RewardPoolServiceServer` interface.
    - The service struct will hold a reference to `*actor.System`.
    - `GetState` implementation will call `system.State()` and return the result.
    - `Draw` implementation will handle a stream of draw requests, calling `system.Draw()` for each and streaming the responses back.

4.  **Configuration Update**:
    - Modify `internal/config/config.go` and `internal/config/yaml_config.go`.
    - Add a `GRPC` struct to the `Config` with `Enabled` (bool) and `ListenAddress` (string) fields.
    - Update `samples/config.yaml` to include the new `grpc` section, with `enabled: false` by default.

5.  **CLI Integration**:
    - Modify `cmd/cli/main.go`.
    - In the `main` function, after setting up the `actor.System`, check `cfg.GRPC.Enabled`.
    - If enabled, create a new goroutine to:
        - Create a TCP listener on `cfg.GRPC.ListenAddress`.
        - Create a new gRPC server.
        - Register the `RewardPoolService` implementation.
        - Start serving requests.
    - Ensure graceful shutdown of the gRPC server when the application exits.

6.  **Testing**:
    - Add a new test file `pkg/rewardpool-grpc-service/service_test.go`.
    - Write unit tests for the gRPC service, mocking the `actor.System` if necessary.

### Result

I have completed all the steps in the plan.

- Created a new gRPC service in `pkg/rewardpool-grpc-service`.
- Defined the protobuf file and generated the Go code.
- Implemented the gRPC service.
- Updated the configuration to enable/disable the gRPC service.
- Integrated the gRPC server into the CLI application.
- Added unit tests for the gRPC service.
- Refactored the code to improve the structure and handle graceful shutdown.
