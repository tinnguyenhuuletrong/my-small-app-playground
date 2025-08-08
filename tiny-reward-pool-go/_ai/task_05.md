<!-- Read _ai/doc/*.md first -->

# Target
- Refactor the `Processor.Draw` method to return a channel (`<-chan DrawResponse`) for a more idiomatic and developer-friendly API. 
- Create a benchmark to compare the performance of the new channel-based implementation against the original callback-based version.

## Iter 01
### Problem
The current callback-based `Draw` method is not idiomatic Go, is cumbersome to use, and forces complex, error-prone concurrency management (manual locking) on the client side.

### Plan
We will refactor the `Draw` method to use a channel for returning results, which is a more standard and safer pattern in Go. We will then benchmark this new implementation against the original to quantify the performance trade-off.

1.  **Preserve Original for Benchmarking:**
    *   In `internal/processing/processing.go`, rename the current `Draw(callback func(DrawResponse)) uint64` method to `DrawWithCallback(callback func(DrawResponse)) uint64`.
    *   This preserves the original implementation for a direct performance comparison.

2.  **Refactor `DrawRequest` and Core Loop:**
    *   Modify the `DrawRequest` struct: remove the `Callback` field and add a `ResponseChan chan DrawResponse` field.
    *   Update the `run()` loop in the `Processor` to send the `DrawResponse` to the `ResponseChan` from the request instead of invoking a callback.

3.  **Implement New Channel-Based `Draw` Method:**
    *   Create the new primary `Draw` method with the signature `Draw() <-chan DrawResponse`.
    *   This method will create a response channel, package it in the `DrawRequest`, send it to the processor's internal request channel, and return the response channel to the caller.

4.  **Create Comparative Benchmark:**
    *   Make sure all test passed `make test` and existing bench work `make bench`

4.  **Create Comparative Benchmark:**
    *   Create a new benchmark file: `cmd/bench/bench_draw_apis_test.go`.
    *   This file will contain two benchmarks:
        *   `BenchmarkDrawWithCallback`: Measures the performance of the original implementation.
        *   `BenchmarkDrawChannel`: Measures the performance of the new channel-based implementation.

5.  **Analyze and Document:**
    *   Run the new benchmark using `make bench`.
    *   Document the results (ops/sec, allocs/op, B/op) in `_ai/doc/bench.md` to make an informed decision about the trade-offs.

6.  **Update Client and Tests:**
    *   Refactor the client code in `cmd/cli/main.go` and the unit tests in `internal/processing/processing_test.go` to use the new, more idiomatic `Draw() <-chan DrawResponse` method.

## Iter 02

### Problem
The channel-based `Draw` method is nearly 10x slower and allocates ~16x more memory per operation compared to the `DrawWithCallback` method. The performance degradation is caused by excessive channel allocations in the `Draw` method and goroutine creation overhead in the benchmark test itself.

### Plan
To fix this, I will use a `sync.Pool` to recycle `DrawRequest` objects, including their response channels. This will virtually eliminate channel allocations from the critical path. I will also refactor the benchmark to use a pool of worker goroutines, providing a more accurate performance measurement.

1.  **Introduce `sync.Pool` for `DrawRequest`:**
    *   In `internal/processing/processing.go`, add a `sync.Pool` to the `Processor` struct.
    *   This pool will manage `DrawRequest` objects. Each object in the pool will contain a pre-allocated response channel (`make(chan DrawResponse, 1)`).

2.  **Refactor `Draw()` Method:**
    *   Modify `Draw()` to get a `DrawRequest` object from the `sync.Pool`.
    *   It will then send this request into the processor's queue and return the pre-existing response channel from the pooled object to the caller.

3.  **Update Core `run()` Loop:**
    *   After processing a request and sending the response back on the channel, the `run()` loop will **not** close the channel.
    *   Instead, it will return the entire `DrawRequest` object to the `sync.Pool` for immediate reuse.

4.  **Refactor Benchmark `bench_draw_apis_test.go`:**
    *   Modify `BenchmarkDrawChannel` to create a fixed-size pool of worker goroutines *before* the benchmark timer starts.
    *   These workers will be responsible for calling `proc.Draw()` and receiving the responses, which eliminates the per-call goroutine creation overhead from the measurement.

5.  **Verify and Document:**
    *   Run the full suite of benchmarks again with the optimized implementation.
    *   Document the new results in `_ai/doc/bench.md`, comparing the optimized channel version against the callback version. The goal is to make their performance characteristics nearly identical.