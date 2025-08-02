<!-- Read _ai/doc/*.md first -->

# Target 
    - Init project following PRD
    - Reward pool module 
    - WAL module 
    - Config module 
    - Cli cmd for testing

## Iter 01
### Plan
  - Initialize Go project structure (already present)
  - Design and implement basic Reward Pool module:
    - Define in-memory pool structure and item catalog
    - Load initial state from config file
  - Implement WAL module:
    - Log draw operations to WAL file
    - Support replaying WAL for recovery
  - Implement Config module:
    - Load configuration from config.json
  - Create CLI command for basic testing:
    - Allow drawing rewards and viewing pool state
### Result
  - Project structure and Go module initialized
  - Reward Pool module created with in-memory catalog and config loading
  - WAL module implemented for draw logging
  - Config module implemented for config loading
  - CLI command added for basic draw and pool state testing
### Problem
  - Should move type define into one file
    - Define Config Pool, PoolReward, WalLogItem
    - Define interface for each module
  - Should add context which can inject the module easier for testing
    - Example 
  - Each module should have a tests

```go
pool.draw(ctx) {
    // get wall module to write log 
    // get utils module for random 
}
```

## Iter 02
### Plan
  - Refactor type definitions into a single file:
    - Define ConfigPool, PoolReward, WalLogItem types
    - Create interfaces for RewardPool, WAL, Config modules
  - Implement a context struct to inject modules for easier testing
  - Refactor pool.draw(ctx) to use injected modules (WAL, utils)
  - Add unit tests for each module
### Result
  - All type definitions consolidated in one file (`types.go`)
  - Interfaces for RewardPool, WAL, Config, and Utils modules created
  - Context struct implemented for dependency injection and easier testing
  - `pool.draw(ctx)` refactored to use injected modules (WAL, utils)
  - Unit tests added for RewardPool, WAL, Config, and Utils modules; all tests passing