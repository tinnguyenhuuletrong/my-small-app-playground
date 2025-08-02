# PRD: High-Performance Reward Pool Service

This document outlines the Product Requirements and Implementation Details for a high-performance, in-memory Reward Pool Random Service, tailored for a Code AI Agent.

***

### 1. Introduction

This project is to build a **Reward Pool Random Service** for rapid and reliable reward distribution. The key requirements are **low-latency** and **high-throughput**, achieved by processing requests sequentially in-memory to eliminate mutex locks and minimize GC overhead. The system must ensure data integrity through a Write-Ahead Log (WAL). 

The initial prototype will be developed in **Golang**.

***

### 2. Core Features & Requirements

#### 2.1. Reward Pool Structure

* **In-Memory Pool**: The service manages a single, global reward pool in memory.
* **Item Catalog**: The pool is composed of multiple item catalogs defined in a configuration file.
    * `ItemID` (string): A unique identifier for the reward type.
    * `Quantity` (int): The number of available items.
    * `Probability` (float64): The selection weight for this item.
* **Initialization**: The pool's state is loaded from a `config.json` file or a snapshot file at startup.

#### 2.2. Processing Model

* **Single-Threaded Core**: All state-mutating operations (logging, drawing, updating quantity) **must** occur in a single dedicated goroutine. This is the central design constraint to avoid mutexes.
* **Request Channel**: Incoming requests are pushed into a buffered Go channel. The core processing loop reads from this channel one by one.
* **Write-Ahead Log (WAL)**:
    1.  After a draw is performed, a log entry containing the **outcome** must be written to a WAL file first. Then update the memory later. This makes recovery deterministic.
    2.  The format should be `DRAW <request_id> <item_id>` for a successful draw or `DRAW <request_id> FAILED` for a failure (e.g., pool empty).
    3.  The response to the client should only be sent **after** the WAL entry is successfully flushed to disk.
* **Internal Request ID**: The service generates a unique, incrementing `uint64` `request_id` for each operation.
* **Internal State and Snapshot**: Memory pool should init from config by default. After that maitain the state in memory structure. It support snapshot and restore for persistent storage later - see 3.2 below

### 3. WAL Recovery & Cleanup Procedures

These procedures are critical for ensuring data integrity and managing disk space.

#### 3.1. Startup Recovery Procedure

The service must automatically recover its state from the logs upon starting.

1.  **Check for Snapshot**: On startup, the service first looks for a state snapshot file (e.g., `snapshot.json`).
    * **If a snapshot exists**: Load the initial pool state from this snapshot file.
    * **If no snapshot exists**: Load the initial pool state from the base `config.json` file.
2.  **Check for WAL**: After loading the initial state, check for a `wal.log` file.
3.  **Replay WAL**: If `wal.log` exists, the service must read it from beginning to end and replay every logged operation against the in-memory pool.
    * For each line (`DRAW <request_id> <item_id>`), the service performs the corresponding action: decrement the quantity of the specified `item_id`.
    * Because the **outcome** is logged, there is no randomness during recovery, guaranteeing a deterministic state reconstruction.
    * The internal `request_id` counter must also be updated to the highest ID found in the log to prevent reuse.
4.  **Ready State**: Once the WAL has been fully replayed, the service is considered consistent and can begin accepting new client connections.
5.  **Post-Recovery Cleanup**: After a successful recovery, the replayed `wal.log` should be archived (e.g., renamed to `wal_<timestamp>.log.bak`) or deleted.

#### 3.2. WAL Cleanup & Snapshotting

To prevent the WAL from growing indefinitely and to shorten recovery times, the service will implement periodic snapshotting.

1.  **Trigger**: The snapshotting process will be triggered based on the number of WAL entries (e.g., every 100,000 draw operations).
2.  **Procedure**:
    * **Block New Requests**: Briefly pause accepting new connections or requests from the channel.
    * **Write Snapshot**: Write the current state of the entire reward pool to a temporary snapshot file (e.g., `snapshot.json.tmp`).
    * **Flush and Replace**: Once the temporary file is successfully written and flushed to disk, atomically rename it to `snapshot.json`, replacing the previous snapshot.
    * **Delete Old WAL**: The `wal.log` file that existed *before* the snapshot began can now be safely deleted.
    * **Resume Operations**: The service can now resume accepting new requests, writing them to a new, empty `wal.log`.

