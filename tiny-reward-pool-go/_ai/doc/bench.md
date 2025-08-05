# Benchmarks in `cmd/bench`

This folder contains benchmark scenarios for the reward pool system.

- Temporary files (WAL logs, snapshots, configs) are saved in `_tmp/`.

## Usage

Run the benchmark:

```sh
make bench
# or
go test -bench=. ./cmd/bench/
```

## Output Metrics
- draws/sec
- bytes/draw
- gc_count
- wal_bytes/draw
- wal_file_size

## Sample Benchmark Results

### No WAL (mock WAL)
```
BenchmarkPoolDrawNoWal-8          7,131,360   162.0 ns/op   16.00 bytes/draw   6,173,266 draws/sec   30.00 gc_count
```

### Real WAL (file logging)
```
BenchmarkPoolDrawWithRealWAL-8    269,635   4,498 ns/op   61.06 bytes/draw   222,335 draws/sec   4.000 gc_count   16.59 wal_bytes/draw   4,472,690 wal_file_size
```

## Summary & Analysis

### Difference
- **No WAL (mock WAL):** Achieves much higher throughput and lower latency because no file I/O is performed. Memory usage is minimal, and there is no WAL file growth.
- **Real WAL (file logging):** Throughput drops significantly and latency increases due to the overhead of writing each draw to disk. WAL file size grows with each operation, and GC count may increase due to more allocations.

### Reason
- The main performance bottleneck in the real WAL scenario is disk I/O. Each draw operation triggers a file write, which is much slower than in-memory operations. This also increases memory pressure and can trigger more frequent garbage collection.

### Plan for Improvement
- **Batch WAL Writes:** Buffer multiple draw operations in memory and write them to disk in batches to reduce the number of file operations.
- **Optimize WAL Format:** Reduce the size of each WAL entry and use more efficient serialization.
- **Snapshot & WAL Rotation:** Periodically save snapshots and rotate WAL files to keep file sizes manageable and improve recovery speed.
- **Benchmark Variants:** Add more benchmark scenarios to measure the impact of batching, and WAL rotation.

These improvements should help close the performance gap between the mock WAL and real WAL scenarios, while maintaining durability and auditability.

