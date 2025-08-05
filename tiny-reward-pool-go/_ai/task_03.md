<!-- Read _ai/doc/*.md first -->

# Target 
    - Create a go benchmark
      - Scenario - same as _ai/doc/requirement.md#2.2. Processing Model
    - processing as much as possible and get stats, process per sec, memory usage, gc... 

## Iter 01
### Plan
1. **Review Existing Code & Requirements**
   - Ensure understanding of the single-threaded processing model and WAL requirements from `_ai/doc/requirement.md`.
   - Identify entry points for benchmarking (core draw loop, request channel, WAL logging).
2. **Implement Benchmark**
  - code implement in `cmd/bench`
  - `bench_no_wal_test.go` - no wal 
  - `bench_wal_test.go` - with wal log into a file

3. **Script to run***
  - `make bench`

## Iter 02
### Target
  - create `bench_wal_mmap_test.go`. target is using memory mmap to speedup the write
### Plan
  - Read what we have in _ai/doc/*.md, _ai/doc/bench.md
  - Read what we have in _ai/doc/*.md, _ai/doc/bench.md

#### Plan
1. **Review Existing WAL Implementation**
   - Understand current WAL logic in `bench_wal_test.go` and related code in `internal/wal/`.
   - Identify how WAL writes and flushes are performed.

2. **Design Memory-Mapped WAL**
   - Research Go libraries for memory-mapped file support (e.g., `golang.org/x/exp/mmap` or `syscall`).
   - Plan how to replace or wrap file I/O with mmap for WAL writes.

3. **Implement `bench_wal_mmap_test.go`**
   - Create a new benchmark file in `cmd/bench/` for WAL using mmap.
   - Ensure the benchmark follows the same scenario as described in `_ai/doc/requirement.md#2.2. Processing Model`.
   - Log draw outcomes to a memory-mapped WAL file, flush as needed.

4. **Metrics Collection**
   - Measure and report: draws/sec, bytes/draw, gc_count, wal_bytes/draw, wal_file_size.
   - Compare results to previous benchmarks (no WAL, file WAL).

5. **Documentation**
   - Document the mmap approach, limitations, and any changes to the processing model.
   - Summarize findings and performance differences in `_ai/doc/bench.md`.