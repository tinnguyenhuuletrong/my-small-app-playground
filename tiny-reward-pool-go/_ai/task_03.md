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