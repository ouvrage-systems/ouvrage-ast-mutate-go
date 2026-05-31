# ADR 0002: Error Interception and Fail-Fast Policy

## Status
Accepted

## Context
In configuration management and SRE pipelines, silent failures (such as missing keys or failed pattern matches resulting in empty deployments) lead to "ghost configurations." 
Traditional scripting tools often continue execution on warning, which is dangerous in production. We need a deterministic, transactional error policy that aborts execution on unexpected states, while allowing declarative error handling at both the global playbook level and individual rule level.

## Decision
We will enforce an **Absolute Fail-Fast** execution strategy by default. Unhandled exceptions will immediately abort the transaction, discard any in-memory workspace changes, and output a non-zero exit code.

To allow safe optional mutations, we will introduce a structured `on_error` block at the rule level. Error logic is isolated from the main action properties, keeping the nominal path clean.

### 1. Global Failure Configuration
The playbook defines default error behaviors under `spec.settings`:
* `on_failure` (default: `fail`): Overall fallback strategy for unhandled errors (`fail` or `ignore`).
* `on_missing_path` (default: `fail`): Strategy when a targeted AST path is absent (`fail` or `create`).
* `on_empty_match` (default: `fail`): Strategy when a search selector returns zero nodes (`fail` or `ignore`).

### 2. Instruction-Level Interception
Each mutation rule can intercept specific errors and override the global behavior using an `on_error` block. 

```yaml
on_error:
  - type: "missing_path" # Error type to intercept
    policy: "create"      # Overridden policy [fail | ignore | create]
    message: "Path not found, creating parent nodes."
    tags: ["ops", "optional"]
    context:
      container_index: "{loop.index}"
```

### 3. Execution Flow
1. When an operation encounters an error (e.g. `ErrPathNotFound`), the engine stops processing the current action.
2. It looks up the rule's `on_error` block for a matching error `type` (e.g. `missing_path` or `any`).
3. If a match is found:
   - If `policy: ignore`, the engine logs the event, marks it as `skipped` in the audit logs, and proceeds to the next rule.
   - If `policy: create` (only valid for `missing_path`), the engine builds the missing intermediate nodes in the AST and resumes the action.
   - If `policy: fail`, the run is aborted.
4. If no match is found, the engine falls back to the global settings under `spec.settings`. If those also evaluate to `fail`, execution halts immediately.

## Consequences
* **Pros**:
  - High resilience: No partial or corrupted configurations can be written out.
  - Flexibility: Explicitly documented handling of expected optional failures (idempotency).
  - Performance: Context variables and tags are only evaluated when an error is caught.
* **Cons**:
  - Requires verbose error declarations for complex optional configurations.
