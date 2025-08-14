
# Task 10: Refactor to Actor Model

## Target
Refactor the existing `processing` module to more explicitly align with the Actor Model terminology and patterns for better clarity, maintainability, and extensibility. A new `processing_actor` module will be created to allow for a side-by-side comparison with the existing implementation.

## Plan

### Iter 1: Formalize the Actor and Message Types
- **Problem:** The current `processing.Processor` acts like an actor but isn't formally defined as one. The request/response logic is coupled within the `DrawRequest` struct.
- **Plan:**
    1. Create a new module `internal/actor`.
    2. Define explicit message structs for different operations (e.g., `DrawMessage`, `StopMessage`, `FlushMessage`) to decouple requests from their handling logic.
    3. Create a `RewardProcessorActor` struct that encapsulates the core processing logic, state (the reward pool and WAL), and a mailbox (channel) for receiving messages.
    4. Implement a `Receive` method on the actor, containing the main `for-select` loop to handle incoming messages, similar to the current `run` method.

### Iter 2: Create an Actor Management System
- **Problem:** The lifecycle and client-facing API of the current processor are mixed with its internal logic.
- **Plan:**
    1. Create an `actor.System` that manages the `RewardProcessorActor`.
    2. The `System` will be responsible for creating, starting (launching the goroutine), and stopping the actor.
    3. It will expose a clean, high-level API to the rest of the application (e.g., `Draw()`, `Stop()`), hiding the underlying message-passing mechanism. This is analogous to the current `processing.NewProcessor` and its methods.

### Iter 3: Integration and Verification
- **Problem:** The new actor system needs to be integrated and verified.
- **Plan:**
    1. Adapt the existing unit tests from the `processing` module to test the new `actor` module, ensuring all functionalities, including WAL rotation and snapshotting, work correctly.
    2. Update the main CLI application (`cmd/cli`) to use the new `actor.System` instead of the `processing.Processor`.

### Iter 4: Benchmarking and Comparison
- **Problem:** The performance impact of the refactoring is unknown.
- **Plan:**
    1. Create a new set of benchmarks by adapting the existing ones in `cmd/bench`. Same as `cmd/bench/bench_draw_apis_test.go` focus on processing model so use no wal overhead
    2. Run benchmarks for both the old `processing` module and the new `actor` module.
    3. **Benchmarking Strategy:** The primary goal is to ensure the new, more structured implementation does not introduce performance regressions. We will compare:
        - **Throughput:** Operations per second (draws/sec).
        - **Memory Usage:** Bytes per operation.
        - **Garbage Collection:** Number of GC runs.

## Final Result
- Successfully refactored the `processing` module to the `actor` module, aligning with the Actor Model patterns for better clarity and maintainability.
- Removed the `internal/processing` directory and all its files.
- Refactored all the code that was using `internal/processing` to use `internal/actor` instead.
- Verified that all tests pass after the refactoring.
- Updated the documentation in `_ai/doc/agent_note.md` and `GEMINI.md` to reflect the changes.
- Updated the benchmark results in `_ai/doc/bench.md`.
