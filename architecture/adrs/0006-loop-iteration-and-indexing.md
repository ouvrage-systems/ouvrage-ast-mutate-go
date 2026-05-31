# ADR 0006: Loop Iteration, Scoping, and Index Stability

## Status
Accepted

## Context
When performing mutations on lists (such as panels in a Grafana dashboard or containers in a Kubernetes pod), we need iteration support (`for_each`). 
This introduces two core challenges:
1. **Name Masking**: In nested loops, a single global `loop.item` namespace gets overwritten at each level, losing access to parent loop parameters.
2. **Index Shift Corruption**: Deleting an item from an array in-place during iteration shifts the indices of all subsequent items. Any target paths resolved dynamically using loop indices (e.g. `$.panels[3]`) are immediately invalidated, corrupting the AST.

## Decision
We will adopt a structured loop declaration, stacked scoping, and a deferred deletion model.

### 1. Loop Syntax (`for_each` Command)
Loops are declared as structured maps using `on` (target list) and `as` (iterator alias):
```yaml
action: "for_each"
properties:
  on: "register.ts_panels"
  as: "panel"
```

### 2. Stacked Volatile Scoping (`loop.*`)
Iterators are registered under the `loop` namespace using their alias (e.g., `loop.panel`).
* The current value is accessed via `{loop.panel::value}` and its path via `{loop.panel::path}`.
* In nested loops, variables are pushed onto a scope stack. Inner mutations can resolve parent variables (e.g., referencing `{loop.panel::path}` inside a sub-loop iterating over `loop.target`) without name masking.

### 3. Stable Indexing (Tombstoning & Compaction)
To prevent index shifts during loop iteration:
1. **Tombstoning (Soft Delete)**: When a `delete` action is executed on an array element during a loop, the engine does **not** shrink the array in the active workspace `$` immediately. Instead, it marks the node internally as deleted (tombstoned). The array maintains its original size and offsets, keeping all active `{loop.<alias>::path}` pointers stable.
2. **Compaction**: Once the loop step completes, the engine compacts the mutated array by copying it and filtering out all tombstoned nodes.

```text
[Loop Start: Array [A, B, C]]
   │
   ├── Iteration 0: Keep A
   ├── Iteration 1: Action Delete B ──► Mark B as Tombstone [A, B(Tombstone), C] (Size is still 3)
   ├── Iteration 2: Keep C (C is still at index 2, path $.array[2] remains valid)
   │
[Loop End] ──► Compaction ──► Rebuild Array omitting Tombstones ──► Final Array: [A, C]
```

## Consequences
* **Pros**:
  - Stable loops: Completely resolves the index shift issue for all array deletion mutations.
  - Nesting support: Stacked scoping allows writing complex nested loops cleanly.
  - Safety: Declarative schema makes `for_each` validation easy.
* **Cons**:
  - Memory: Temporary tombstone state increases memory overhead slightly during iteration, but it is reclaimed immediately upon compaction at the end of the step.
  - Multi-pass rebuild: Rebuilding arrays during compaction adds a minor CPU cost, which is negligible for configuration files.
