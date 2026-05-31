# ADR 0005: Streaming Execution Model

## Status
Accepted

## Context
Infrastructure configurations can consist of very large multi-document YAML files (such as combined Kubernetes manifests) or infinite JSON Lines (JSONL) streams. 
Loading the entire stream into memory as a single unified AST tree is memory-intensive and prevents treating the tool as a pipeline filter. We need a memory-efficient, stateless streaming architecture.

## Decision
We will implement a **document-by-document streaming model**. The engine will process the input stream sequentially, document by document, using a decoder/encoder loop.

### 1. Workspace Context Scope
* **`$`** (the Workspace) and **`origin`** represent the AST of the **currently processed document** only, rather than the entire file.
* At the start of each document in the stream:
  - `$` is initialized with the current document's AST.
  - `origin` is populated with a deep copy of `$`.
  - The loop context (`loop.*`) is reset.

### 2. Global Persistent Context Scope
* **`register.*`**, **`vars.*`**, and **`source.*`** act as global, persistent spaces that persist across the entire streaming session.
* This allows cross-document data propagation (e.g. registering a value in Document 1 and using it in Document 5).

### 3. Stream Controls (Skip & Drop Actions)
We introduce two streaming control actions to manage the output flow:
* **`skip`**: Immediately terminates the execution of the playbook for the current document. The document is emitted to the output stream unchanged.
* **`drop`**: Excludes the current document from the output stream. The engine stops processing it and discards it (does not write it to stdout/file).

### 4. Input-to-Output Flow
```text
[Input Stream] 
  │   (YAML docs or JSONL lines)
  ▼
[Decoder] ──(Next Doc)──► [Workspace $] ──► [Playbook Execution] 
                                                    │
                                                    ├──► Action: skip ──► [Encoder] ──► [Output]
                                                    ├──► Action: drop ──► (Discarded)
                                                    └──► Success ───────► [Encoder] ──► [Output]
```

## Consequences
* **Pros**:
  - $O(1)$ memory footprint: Memory usage is bounded by the size of a single document, not the total stream size.
  - Compatibility: Seamlessly supports both multi-document YAML and JSONL out of the box.
  - Cross-document state: Registers allow storing values dynamically to share them downstream.
* **Cons**:
  - Out-of-order dependency limit: If a document requires information from a document positioned later in the stream, the engine cannot resolve it in a single pass. Users must structure files with producers first, or chain multiple execution runs.
