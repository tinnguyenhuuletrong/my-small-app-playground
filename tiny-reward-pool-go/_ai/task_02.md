<!-- Read _ai/doc/*.md first -->

# Target 
    - Implement 2.2. Processing Model
    - It should have test along with implementations

## Iter 01
### Plan
  - Implement the 2.2. Processing Model as described in the PRD:
    - Create a single-threaded core using a dedicated goroutine for all state-mutating operations.
    - Use a buffered Go channel for incoming requests.
    - Ensure all draw, WAL logging, and quantity updates happen in the goroutine.
    - Generate unique, incrementing request IDs.
    - WAL entry is written before updating memory and responding.
  - Provide unit tests for the processing model.

### Problem
  - Read the source of internal modules make sure you understand what 's have
  - The processing.Draw() should small tweak 
    - generate requestId and return to the caller first then somehow callback for actually resonse later
    - example
```go
func onResult (itemId) {
    ....
}
var requestId := processing.Draw(onResult)

```

## Iter 02
### Plan
  - Refactor processing.Draw() to:
    - Generate and return the request ID immediately to the caller
    - Accept a callback function (e.g., onResult(itemId)) that will be called asynchronously when the draw result is ready
  - Update the processing model and its test to use this callback pattern
  
### Problem
  - processing module should utilize the types.go - Context
  - the requestID can be concurency access. so better to make it safe with atomic

## Iter 03
### Plan
  - Refactor processing module to:
    - Change NewProcessor to accept a *types.Context and use it for all operations
    - Make request ID generation concurrency-safe using atomic
    - Remove direct dependency injection of modules; use Context only
  - Update CLI and tests to use the new Context-based API

## Iter 04
### Target
  - Refactor wal, pool module prepare for next task
  - Pool module 
      Support save, load snapshot. which is current state of reward pool
  - Now it only have a Add Draw log. It should have 
    - Add Snapshot path
    - Flush. Makesure all log save and flush into disk - supporting batch write or transaction like later 
### Plan
  - Refactor wal and pool modules to prepare for snapshot and flush features
  - Pool module:
    - Add support for saving the current state as a snapshot (e.g., to a JSON file)
    - Add support for loading pool state from a snapshot file
  - WAL module:
    - Add support for specifying a snapshot path
    - Implement a Flush() method to ensure all logs are written and flushed to disk
    - Prepare for future batch write or transaction-like operations
  - Update interfaces in types.go to reflect new methods
  - Add unit tests for snapshot save/load and WAL flush

### Result
  - Processing model implemented: single-threaded goroutine, buffered channel, atomic request IDs, WAL-first logging, callback pattern for draw results
  - All modules refactored to use centralized Context for dependency injection
  - Pool module supports save/load snapshot (JSON), tested and integrated in CLI
  - WAL module supports Flush and snapshot path, tested and integrated in CLI
  - CLI demonstrates loading snapshot on start, periodic snapshot save, WAL rotation, and graceful shutdown
  - All interfaces updated in types.go; all unit tests passing