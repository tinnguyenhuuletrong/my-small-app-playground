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

5.  **Create Comparative Benchmark:**
    *   Create a new benchmark file: `cmd/bench/bench_draw_apis_test.go`.
    *   This file will contain two benchmarks:
        *   `BenchmarkDrawWithCallback`: Measures the performance of the original implementation.
        *   `BenchmarkDrawChannel`: Measures the performance of the new channel-based implementation.

6.  **Analyze and Document:**
    *   Run the new benchmark using `make bench`.
    *   Document the results (ops/sec, allocs/op, B/op) in `_ai/doc/bench.md` to make an informed decision about the trade-offs.

7.  **Update Client and Tests:**
    *   Refactor the client code in `cmd/cli/main.go` and the unit tests in `internal/processing/processing_test.go` to use the new, more idiomatic `Draw() <-chan DrawResponse` method.