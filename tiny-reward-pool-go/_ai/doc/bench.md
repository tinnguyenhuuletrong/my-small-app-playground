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
BenchmarkPoolDrawNoWal-8          6,996,973   175.4 ns/op   16.00 bytes/draw   5,699,800 draws/sec   29.00 gc_count
```

### Mmap WAL (memory-mapped WAL)
```
BenchmarkPoolDrawWithMmapWAL-8    2,298,444   485.4 ns/op   54.64 bytes/draw   2,060,333 draws/sec   33.00 gc_count
```

### Real WAL (file logging)
```
BenchmarkPoolDrawWithBasicWAL-8    198,525   5,101 ns/op   59.99 bytes/draw   196,038 draws/sec   3.00 gc_count   16.44 wal_bytes/draw   3,263,820 wal_file_size
```

## Summary & Analysis

### Difference
- **No WAL (mock WAL):** Achieves much higher throughput and lower latency because no file I/O is performed. Memory usage is minimal, and there is no WAL file growth.
- **Mmap WAL:** Throughput and latency are significantly improved compared to real WAL. Memory-mapped WAL achieves ~2M draws/sec and ~485 ns/op, much faster than file WAL but slower than mock WAL. GC count is slightly higher than mock WAL, likely due to buffer allocations. WAL file size and bytes/draw are reduced compared to file WAL.
- **Real WAL (file logging):** Throughput drops significantly and latency increases due to the overhead of writing each draw to disk. WAL file size grows with each operation, and GC count may increase due to more allocations.

### Reason
- The main performance bottleneck in the real WAL scenario is disk I/O. Each draw operation triggers a file write, which is much slower than in-memory operations. This also increases memory pressure and can trigger more frequent garbage collection.
- Mmap WAL reduces disk I/O overhead by writing directly to memory-mapped regions, allowing faster flushes and less syscall overhead. However, it is still slower than pure in-memory mock WAL due to OS-level syncs and memory management.

### Metrics Collection (Aug 2025)
| WAL Type      | draws/sec   | ns/op   | bytes/draw | gc_count | wal_bytes/draw | wal_file_size |
|--------------|-------------|---------|------------|----------|----------------|---------------|
| No WAL       | 5,699,800   | 175.4   | 16.00      | 29.00    | N/A            | N/A           |
| Mmap WAL     | 2,060,333   | 485.4   | 54.64      | 33.00    | N/A            | N/A           |
| Basic WAL    |   196,038   | 5101    | 59.99      | 3.00     | 16.44          | 3,263,820     |

- Mmap WAL is a strong middle ground, offering much better performance than file WAL, with durability and auditability, but not matching pure in-memory speed.

### Plan for Improvement
- **Optimize WAL Format:** Reduce the size of each WAL entry and use more efficient serialization to minimize bytes/draw and WAL file size.
- **Snapshot & WAL Rotation:** Periodically save snapshots and rotate WAL files to keep file sizes manageable and improve recovery speed. Consider automating rotation based on size or time.
- **Benchmark Variants:** Add more benchmark scenarios to measure the impact of WAL rotation, and different mmap flush strategies.
- **Tune mmap WAL:** Experiment with different mmap region sizes, flush intervals, and OS sync strategies to further improve mmap WAL performance.
- **Advanced Recovery:** Explore parallel WAL replay and incremental snapshotting for faster recovery and lower downtime.

These improvements should help close the performance gap between the mock WAL and real WAL scenarios, while maintaining durability and auditability. Mmap WAL is a promising middle ground, but further tuning and batching may yield even better results.

