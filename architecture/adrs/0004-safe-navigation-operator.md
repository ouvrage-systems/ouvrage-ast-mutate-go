# ADR 0004: Safe Navigation Operator (`?.`)

## Status
Accepted

## Context
Under the strict Absolute Fail-Fast policy (ADR 0002), attempting to resolve a path that does not exist in the workspace or a variable namespace triggers an immediate fatal exception. 
However, configuration files often contain optional fields. Wrapping every optional mutation in a verbose `if:` conditional block degrades developer experience (DX) and bloats the playbook size.

## Decision
We will support the safe navigation operator (`?.`) in path expressions.

* When the parser encounters a path segment prefixed with `?.` (e.g. `$.metadata?.labels` or `{register?.optional_panel::path}`):
  - If the parent node is null, undefined, or missing, the evaluator aborts navigation for that expression and immediately returns `null` (or an empty node).
  - The engine does **not** raise a Fail-Fast exception.
* If a standard dot (`.`) is used instead of `?.`, the engine maintains its strict Fail-Fast validation.

### Example Usage:
```yaml
- name: "Delete optional debug panel"
  action: "delete"
  properties:
    target: "{register?.debug_panel::path}"
  on_error:
    - type: "missing_path"
      policy: "ignore"
```
If `register.debug_panel` was never populated, `{register?.debug_panel::path}` resolves to `null`. The `delete` action targets `null`, raising a `missing_path` error, which is caught and ignored by the local `on_error` block. Without `?.`, the expression `{register.debug_panel::path}` would have crashed the run before the `delete` action could even start.

## Consequences
* **Pros**:
  - Improved DX: Simplifies playbooks by removing boilerplate existence checks.
  - Safe optional checks: Cleanly integrates with the `on_error` policy engine.
* **Cons**:
  - Overuse of `?.` can mask actual configuration bugs. Developers must be encouraged to use it only for truly optional fields.
