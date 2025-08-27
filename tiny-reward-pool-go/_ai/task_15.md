# Task 15: Evaluate Ring Buffer for Performance Improvement

## Target

- **Goal**: Brainstorm and evaluate if it's worth applying principles from the LMAX Disruptor pattern (specifically, using a ring buffer) to the actor's mailbox and the WAL's log buffer to make the core processing engine faster and reduce GC overhead.
- **Artifacts**:
  - An experimental actor implementation using a ring buffer.
  - An experimental WAL implementation using a ring buffer.
  - A new benchmark test to compare the performance of all combinations (channel vs. ring buffer actor, slice vs. ring buffer WAL).
  - A concluding analysis based on benchmark results.

---

## Iter 1

### Plan

1.  **Develop Generic Ring Buffer:**

    - Create a new file `internal/utils/ring_buffer.go`.
    - Implement a basic, generic, single-producer, single-consumer (SPSC) lock-free ring buffer. It will manage sequence numbers and pre-allocated slots.
    - It will use `uint64` sequence numbers and cache-line padding to prevent false sharing, which are key performance concepts from the Disruptor.

2.  **Create Experimental Actor (`RingBufferActor`):**

    - Create a new file `internal/actor/ring_buffer_actor.go`.
    - Implement a `RingBufferActor` that uses the generic ring buffer for its message queue instead of a Go channel.
    - The ring buffer will contain pre-allocated message objects to minimize garbage collection.

3.  **Create Experimental WAL (`RingBufferWAL`):**

    - Create a new file `internal/wal/ring_buffer_wal.go`.
    - Implement a `RingBufferWAL` that uses the generic ring buffer for its internal log buffer instead of a slice.
    - The `LogDraw`, `LogUpdate`, etc., methods will claim the next pre-allocated `WalLogEntry` in the buffer, populate it, and publish it.

4.  **Implement Comprehensive Benchmark:**

    - Create a new benchmark test file `cmd/bench/bench_ring_buffer_test.go`.
    - The benchmark will compare the throughput and allocations for the four possible combinations:
      1.  `ChannelActor` + `SliceWAL` (Current baseline)
      2.  `RingBufferActor` + `SliceWAL` (Actor improvement only)
      3.  `ChannelActor` + `RingBufferWAL` (WAL improvement only)
      4.  `RingBufferActor` + `RingBufferWAL` (Combined improvement)
    - Mocks will be used for file I/O to isolate the performance of the in-memory buffering and message passing.

5.  **Analyze and Conclude:**
    - Run the benchmarks and collect performance data (ops/sec, ns/op, B/op, allocs/op).
    - Compare the results across the four combinations to precisely measure the impact of each change.
    - Based on the data, write a summary in the `Result` section, concluding whether the performance gains justify the added complexity for each use case (actor and/or WAL).

### Result

> TODO

### Problem

> TODO
