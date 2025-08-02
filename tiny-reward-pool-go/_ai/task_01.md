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
