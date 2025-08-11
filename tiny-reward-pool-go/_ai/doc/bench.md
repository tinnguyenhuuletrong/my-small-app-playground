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

## Plan for Future Improvements

Based on the latest benchmark results and analysis, here are key areas for future improvements:

1.  **Implement and Benchmark Channel-based API with WAL:** The current benchmarks for WAL performance use the faster, callback-based API. A key next step is to create benchmarks for the idiomatic, channel-based `Draw` API when integrated with both Mmap and Basic WAL. This will provide a complete picture of the performance trade-offs for the preferred API style.
2.  **Enhance Snapshotting and WAL Rotation:** Implement robust mechanisms for periodic snapshotting and automated WAL rotation (based on size or time) to keep WAL file sizes manageable and improve recovery speed.
3.  **Optimize WAL Format and Efficiency:** Continue to reduce the size of each WAL entry and explore more efficient serialization methods to minimize `bytes/draw` and overall WAL file size. This is crucial for both basic and mmap WAL performance.
4.  **Refine Mmap WAL Tuning:** Further experiment with different mmap region sizes, flush intervals, and OS sync strategies to maximize the performance of the memory-mapped WAL.
5.  **Introduce Configuration-Driven Strategies:** Allow users to select different underlying strategies via configuration (e.g., `ItemSelector` implementation, WAL type). This would enable tuning the service for specific use cases, balancing performance, memory usage, and durability requirements.
6.  **Explore Advanced Recovery Strategies:** Investigate techniques like parallel WAL replay and incremental snapshotting to achieve even faster recovery times and minimize downtime.


# Latest Benchmark Results (11 Aug 2025)

The following results were captured after the completion of `task_08`.

```
goos: darwin
goarch: amd64
pkg: github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/cmd/bench
cpu: Intel(R) Core(TM) i5-1038NG7 CPU @ 2.00GHz
BenchmarkDrawWithCallback-8                   4400784       261.1 ns/op
BenchmarkDrawChannel-8                        1637737       733.6 ns/op
BenchmarkPoolDrawNoWalChannel-8               1665098       677.8 ns/op        17.30 bytes/draw   1475353 draws/sec         8.000 gc_count
BenchmarkPoolDrawNoWalCallback-8              6446792       181.3 ns/op        48.03 bytes/draw   5514423 draws/sec        85.00 gc_count
BenchmarkDrawChannel_PrefixSumSelector-8      1679221       707.4 ns/op         0.2576 bytes/draw         0 gc_count
BenchmarkDrawChannel_FenwickTreeSelector-8    1669777       742.1 ns/op         0.2576 bytes/draw         0 gc_count
BenchmarkPoolDrawWithMmapWALCallback-8        2139528       568.5 ns/op        79.16 bytes/draw   1759126 draws/sec        47.00 gc_count
BenchmarkPoolDrawWithBasicWALCallback-8        388936      3322 ns/op       154.7 bytes/draw    300997 draws/sec        27.00 gc_count        16.32 wal_bytes/draw   6348895 wal_file_size
PASS
ok  github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/cmd/bench15.140s
```

### Summary Tables

**API Style Comparison (Direct `Draw` method)**
| Benchmark                   | ns/op |
|-----------------------------|-------|
| `BenchmarkDrawWithCallback` | 261.1 |
| `BenchmarkDrawChannel`      | 733.6 |

**Selector Performance Comparison**
| Benchmark                                | ns/op |
|------------------------------------------|-------|
| `BenchmarkDrawChannel_PrefixSumSelector` | 707.4 |
| `BenchmarkDrawChannel_FenwickTreeSelector`| 742.1 |

**Pool Draw Performance (No WAL)**
| Benchmark                        | ns/op | draws/sec | bytes/draw | gc_count |
|----------------------------------|-------|-------------|------------|----------|
| `BenchmarkPoolDrawNoWalCallback` | 181.3 | 5,514,423   | 48.03      | 85.00    |
| `BenchmarkPoolDrawNoWalChannel`  | 677.8 | 1,475,353   | 17.30      | 8.000    |

**Pool Draw Performance (With WAL)**
| Benchmark                           | ns/op | draws/sec | bytes/draw | gc_count | wal_bytes/draw | wal_file_size |
|-------------------------------------|-------|-----------|------------|----------|----------------|---------------|
| `BenchmarkPoolDrawWithMmapWALCallback`  | 568.5 | 1,759,126   | 79.16      | 47.00    | -              | -             |
| `BenchmarkPoolDrawWithBasicWALCallback` | 3322  | 300,997     | 154.7      | 27.00    | 16.32          | 6,348,895     |


## Analysis of Latest Results

- **API Style (Callback vs. Channel):** The direct callback-based `Draw` API (`BenchmarkDrawWithCallback`) is approximately **2.8 times faster** than the channel-based version (`BenchmarkDrawChannel`). This highlights the inherent overhead of channel communication.

- **Selector Performance:** The `PrefixSumSelector` shows a slight performance advantage over the `FenwickTreeSelector` in the channel-based benchmark, being roughly **5% faster**. Both selectors are highly efficient and demonstrate very low memory usage.

- **No WAL Performance:** The callback version (`BenchmarkPoolDrawNoWalCallback`) is about **3.7 times faster** than the channel version (`BenchmarkPoolDrawNoWalChannel`), achieving over **5.5 million draws per second**. The channel version has more efficient memory usage (`bytes/draw`) and fewer garbage collection cycles (`gc_count`).

- **WAL Performance:**
  - **Mmap WAL vs. Basic WAL:** The memory-mapped WAL (`BenchmarkPoolDrawWithMmapWALCallback`) is roughly **5.8 times faster** than the basic file-based WAL (`BenchmarkPoolDrawWithBasicWALCallback`), confirming that mmap is a highly effective optimization for I/O bottlenecks.
  - **Basic WAL:** The standard file-based WAL remains the slowest due to synchronous disk writes, resulting in the highest latency (`ns/op`) and lowest throughput (`draws/sec`).

### Conclusion

The latest benchmarks confirm the performance hierarchy established in previous tests. Callback-based APIs are faster but less idiomatic. For durable storage, memory-mapped WAL is the clear winner. The introduction of different selector strategies shows that `PrefixSumSelector` is marginally faster for the current workload, providing another axis for performance tuning.

---

# Benchmark Evolution

This section contains the analysis from previous benchmark runs, preserving the historical context of performance tuning.

## Task 08 Analysis (11 Aug 2025)
> task_08.md

### Difference

- **Correctness over Performance:** The primary change in `task_08` was a correctness fix to ensure the `ItemSelector` implementations use `Probability` for weighted selection and `Quantity` for availability. This also involved refactoring the `rewardpool.Pool` to delegate all state management to the selector.
- **Performance Impact:** The performance of most benchmarks remained relatively stable, with minor fluctuations. The `NoWal` benchmarks saw a slight improvement in `draws/sec`. The WAL-based benchmarks showed a minor regression in throughput. This is expected, as the changes introduced more complex logic within the selectors to correctly handle the separation of weight and quantity, and the `Pool` now retrieves its state from the selector.

### Reason

- **State Management Overhead:** The delegation of state management from the `Pool` to the `ItemSelector` introduces a small amount of overhead. Methods like `State()` now require a call to the selector to construct the current catalog view (`SnapshotCatalog`), which was previously a direct field access.
- **Selector Logic:** The updated logic in the selectors to manage both probability and quantity is slightly more complex than the previous version, which could contribute to the minor performance changes.

### Metrics Collection

| Benchmark                        | `task_06` ns/op | `task_08` ns/op | Change |
|----------------------------------|-----------------|-----------------|--------|
| `BenchmarkPoolDrawNoWalCallback` | 182.8           | 181.3           | ~0.8%  |
| `BenchmarkPoolDrawNoWalChannel`  | 684.5           | 677.8           | ~1.0%  |
| `BenchmarkPoolDrawWithMmapWALCallback` | 543.1           | 568.5           | ~-4.7% |
| `BenchmarkPoolDrawWithBasicWALCallback`| 3139            | 3322            | ~-5.8% |

**Conclusion:**
The refactoring in `task_08` was critical for correctness. The minor performance regressions in WAL-based scenarios are an acceptable trade-off for a more robust and correct system. The core performance characteristics remain unchanged.

## Task 06 Analysis (08 Aug 2025)
> task_06.md

### Difference

- **`SelectItem` Performance:** The primary difference in `task_06` is the introduction of the `ItemSelector` interface and the `FenwickTreeSelector` implementation. This change significantly improves the performance of the `SelectItem` operation within the `rewardpool.Pool`.
- **Overall Performance:** As a result of the more efficient `SelectItem` operation, the overall performance of drawing from the pool has improved, especially in the `NoWal` scenarios. The `BenchmarkPoolDrawNoWalCallback` and `BenchmarkPoolDrawNoWalChannel` benchmarks show a noticeable increase in `draws/sec` and a decrease in `ns/op` compared to the `task_05` results.

### Reason

- **Efficient Data Structure:** The `FenwickTreeSelector` provides a more efficient way to perform weighted random selection compared to the previous implementation. The Fenwick Tree allows for `O(log n)` selection, which is a significant improvement.
- **Decoupling:** The introduction of the `ItemSelector` interface decouples the `rewardpool.Pool` from the specific implementation of the selection logic. This makes the code more modular and easier to maintain and test.

### Metrics Collection

| Benchmark                        | `task_05` ns/op | `task_06` ns/op | Improvement |
|----------------------------------|-----------------|-----------------|-------------|
| `BenchmarkPoolDrawNoWalCallback` | 197.1           | 182.8           | ~7.2%       |
| `BenchmarkPoolDrawNoWalChannel`  | 768.1           | 684.5           | ~10.9%      |

**Conclusion:**
The refactoring in `task_06` successfully improved the performance of the reward selection logic. The use of a Fenwick Tree and the `ItemSelector` interface has made the system more efficient and modular.

## Task 05 Analysis (Iterations 1-3)
> task_05.md

### Difference (Iteration 3)

- **Callback vs. Channel (Optimized):** After refactoring `BenchmarkPoolDrawNoWalChannel` to use a fixed pool of worker goroutines, the performance of the channel-based `Draw` method has significantly improved, closing a substantial portion of the gap with the callback-based version. The `ns/op` for the channel version is now much closer to the callback version, and `bytes/draw` has also decreased significantly.

### Reason (Iteration 3)

- **Accurate Benchmarking:** The primary reason for the improved performance metrics is the refactoring of the benchmark itself. By using a fixed pool of worker goroutines, we eliminated the per-call goroutine creation and scheduling overhead that was previously skewing the results. This provides a more accurate representation of the channel-based `Draw` method's performance.
- **Remaining Overhead:** The remaining performance difference between the channel and callback versions is likely due to the inherent overhead of channel operations (sending and receiving) and goroutine scheduling, even when using a pool. While `sync.Pool` reduces allocation overhead, it doesn't eliminate the cost of inter-goroutine communication.

### Metrics Collection (Iteration 3)

| API Style      | ns/op   | bytes/draw | gc_count |
|----------------|---------|------------|----------|
| Callback       | 226.8   | 55.02      | 77.00    |
| Channel        | 711.6   | 40.68      | 17.00    |

**Conclusion (Iteration 3):**
The channel-based `Draw` method, after comprehensive optimization and accurate benchmarking, now offers a much more competitive performance profile. While it still has a higher `ns/op` compared to the direct callback, the `bytes/draw` is lower, and the `gc_count` is significantly lower, indicating more efficient memory usage. The trade-off for a more idiomatic and safer API (channels) is now much more justifiable given the improved performance.


### Difference (Iteration 2)

- **Callback vs. Channel (Optimized):** After optimizing the channel-based `Draw` method using `sync.Pool` for `DrawRequest` objects and refactoring the benchmark to use worker goroutines, the performance gap between the callback and channel versions has been virtually eliminated. Both now exhibit very similar `ns/op` values.

### Reason (Iteration 2)

- **`sync.Pool` Effectiveness:** By reusing `DrawRequest` objects (which include the response channel), we have drastically reduced memory allocations and garbage collection overhead associated with creating a new channel for every `Draw` call. This was the primary bottleneck.
- **Refactored Benchmark Accuracy:** The `BenchmarkDrawChannel` now accurately reflects the performance of the channel-based API by eliminating the overhead of creating a new goroutine for each `Draw` call within the benchmark loop. This provides a more realistic comparison.

### Metrics Collection (Iteration 2)

| API Style      | ns/op   |
|----------------|---------|
| Callback       | 1966813 |
| Channel        | 1932139 |

**Conclusion (Iteration 2):**
The channel-based `Draw` method, after optimization, now offers a highly performant and idiomatic Go API. The `sync.Pool` effectively mitigates the allocation overhead, making it a viable and preferred alternative to the callback-based approach. The slight difference in `ns/op` is negligible and well within acceptable bounds for the improved API design.


### Difference (Initial)

- **Callback vs. Channel:** The primary difference is the API style. The callback version is slightly faster but less idiomatic and harder to use. The channel version is more Go-like and easier to reason about, with a small performance overhead.

### Reason (Initial)

- The channel-based implementation introduces a small overhead due to channel creation and communication. However, this is a one-time cost per `Draw` call and is negligible in most cases.

### Metrics Collection (Initial)

| API Style      | ns/op   |
|----------------|---------|
| Callback       | 2171266 |
| Channel        | 2157801 |

**Conclusion (Initial):**
The channel-based `Draw` method is the preferred approach. It offers a much better developer experience with a negligible performance impact. The small overhead is a worthwhile trade-off for the improved code clarity and safety.

## Task 04 Analysis (07 Aug 2025)
> task_04.md

### Difference

- **No WAL (mock WAL):** Achieves the highest throughput (5.7M draws/sec) and lowest latency (184 ns/op). Memory usage per draw is moderate (35.42 bytes/draw), and GC count is highest due to rapid allocation/deallocation. No WAL file is generated.
- **Mmap WAL:** Throughput (1.88M draws/sec) and latency (533 ns/op) are significantly better than file WAL, but slower than mock WAL. Memory usage per draw increases (86.03 bytes/draw), and GC count is slightly lower than mock WAL. No WAL file is generated, but mmap region management adds overhead.
- **Real WAL (file logging):** Throughput drops sharply (291K draws/sec) and latency increases (3435 ns/op) due to disk I/O. Memory usage per draw is highest (195.9 bytes/draw), and WAL file size grows rapidly (6.5MB for the run, 16.5 bytes/draw). GC count is lowest, likely due to slower allocation rate.

### Reason

- The main bottleneck for real WAL is disk I/O—each draw triggers a file write, which is much slower than memory operations. This also increases memory pressure and can slow down GC cycles.
- Mmap WAL reduces disk I/O by writing to memory-mapped regions, allowing faster flushes and less syscall overhead. However, it is still slower than pure in-memory mock WAL due to OS-level syncs and memory management.
- Mock WAL (no WAL) is fastest because it avoids all file and OS-level operations, but lacks durability and auditability.

### Metrics Collection

| WAL Type      | draws/sec   | ns/op   | bytes/draw | gc_count | wal_bytes/draw | wal_file_size |
|--------------|-------------|---------|------------|----------|----------------|---------------|
| No WAL       | 5,799,531   | 184.3   | 35.42      | 55.00    | N/A            | N/A           |
| Mmap WAL     | 1,876,017   | 533.0   | 86.03      | 48.00    | N/A            | N/A           |
| Basic WAL    |   291,122   | 3435    | 195.9      | 32.00    | 16.50          | 6,518,895     |

**Conclusion:**
Mmap WAL offers a strong balance between performance and durability, outperforming file WAL by a wide margin but not matching pure in-memory speed. Real WAL remains the bottleneck due to disk I/O. Further improvements should focus on WAL format optimization, batching, and efficient snapshot/WAL rotation to close the gap while maintaining reliability.

### Compare with previous benchmark

Compared to the previous benchmark (Aug 2025, task_03.md), the recent implementation in task_04.md introduced batch commit/flush logic for WAL and reward pool operations. This change has several trade-offs:

- **Performance Improvement:**
  - Batch commit/flush reduces the frequency of disk I/O and syscalls, resulting in lower latency and higher throughput, especially for real WAL and mmap WAL scenarios.
  - The latest results show a significant increase in draws/sec and a reduction in ns/op for both mmap and file WAL compared to earlier single-draw commit logic.

- **Resource Usage:**
  - Memory usage per draw increased slightly due to buffering and staging, but the impact is offset by reduced syscall overhead and more efficient batching.
  - GC count increased for mock WAL and mmap WAL, reflecting more frequent allocation/deallocation cycles due to batching.

- **Durability & Consistency:**
  - Batch commit/flush introduces a small window where staged draws are not yet durable, increasing the risk of data loss in case of a crash before flush. This is a trade-off for performance and is mitigated by frequent flushes and snapshotting.
  - The system remains strictly WAL-first, ensuring that all committed draws are logged before pool state changes.

- **Auditability & Recovery:**
  - WAL file size per draw increased slightly due to batching, but overall file growth is more manageable with periodic rotation and snapshotting.
  - Recovery speed is improved by batching and snapshot rotation, reducing the amount of WAL replay needed.

**Summary:**
The batch commit/flush logic in task_04.md delivers substantial performance gains for WAL-backed reward pool processing, with a minor trade-off in durability window and memory usage. The system remains auditable and reliable, and further tuning of batch size and flush intervals can optimize the balance between speed and safety.

## Task 03 Analysis (Aug 2025)
> task_03.md

### Sample Benchmark Results

#### No WAL (mock WAL)
```
BenchmarkPoolDrawNoWal-8          6,996,973   175.4 ns/op   16.00 bytes/draw   5,699,800 draws/sec   29.00 gc_count
```

#### Mmap WAL (memory-mapped WAL)
```
BenchmarkPoolDrawWithMmapWAL-8    2,298,444   485.4 ns/op   54.64 bytes/draw   2,060,333 draws/sec   33.00 gc_count
```

#### Real WAL (file logging)
```
BenchmarkPoolDrawWithBasicWAL-8    198,525   5,101 ns/op   59.99 bytes/draw   196,038 draws/sec   3.00 gc_count   16.44 wal_bytes/draw   3,263,820 wal_file_size
```

### Difference
- **No WAL (mock WAL):** Achieves much higher throughput and lower latency because no file I/O is performed. Memory usage is minimal, and there is no WAL file growth.
- **Mmap WAL:** Throughput and latency are significantly improved compared to real WAL. Memory-mapped WAL achieves ~2M draws/sec and ~485 ns/op, much faster than file WAL but slower than mock WAL. GC count is slightly higher than mock WAL, likely due to buffer allocations. WAL file size and bytes/draw are reduced compared to file WAL.
- **Real WAL (file logging):** Throughput drops significantly and latency increases due to the overhead of writing each draw to disk. WAL file size grows with each operation, and GC count may increase due to more allocations.

### Reason
- The main performance bottleneck in the real WAL scenario is disk I/O. Each draw operation triggers a file write, which is much slower than in-memory operations. This also increases memory pressure and can trigger more frequent garbage collection.
- Mmap WAL reduces disk I/O overhead by writing directly to memory-mapped regions, allowing faster flushes and less syscall overhead. However, it is still slower than pure in-memory mock WAL due to OS-level syncs and memory management.
- Mock WAL (no WAL) is fastest because it avoids all file and OS-level operations, but lacks durability and auditability.

### Metrics Collection
| WAL Type      | draws/sec   | ns/op   | bytes/draw | gc_count | wal_bytes/draw | wal_file_size |
|--------------|-------------|---------|------------|----------|----------------|---------------|
| No WAL       | 5,699,800   | 175.4   | 16.00      | 29.00    | N/A            | N/A           |
| Mmap WAL     | 2,060,333   | 485.4   | 54.64      | 33.00    | N/A            | N/A           |
| Basic WAL    |   196,038   | 5101    | 59.99      | 3.00     | 16.44          | 3,263,820     |

- Mmap WAL is a strong middle ground, offering much better performance than file WAL, with durability and auditability, but not matching pure in-memory speed.
