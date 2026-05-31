---
date: 2026-05-31
location: "Ouvrage Workspace Room"
goal: "Establish the definitive specifications, rules, boundaries, and syntax conventions for ast-mutate (stateless AST mutation engine) before writing code."
participants:
  - name: "Guillaume"
    github: "gpineda-dev"
    type: "human"
    role: "SRE & Platform Architect"
  - name: "David"
    type: "agent"
    role: "UNIX & CLI Systems Persona"
  - name: "Lucas"
    type: "agent"
    role: "Python & Developer Experience Automation Persona"
  - name: "Nicolas"
    type: "agent"
    role: "Cloud-Native & GitOps Kubernetes Persona"
  - name: "Gemini"
    type: "agent"
    role: "Lead Systems Architect & AI Scribe"
---

# Design Meeting: `ast-mutate` Engine Architecture


## Part 1: Error Management
To enforce the "Everything Declarative" principle, the engine is strictly deterministic and uses an **Absolute Fail-Fast** strategy by default.

### 1. Global Settings (`spec.settings`)
```yaml
spec:
  settings:
    on_failure: fail           # Default for unhandled exceptions [fail | ignore]
    on_missing_path: fail      # Behavior when a targeted path is absent [fail | create]
    on_empty_match: fail       # Behavior when query returns no nodes [fail | ignore]
```

### 2. Instruction-Level Interception & Enrichment (`on_error`)
Error handling and log enrichment are isolated under a dedicated `on_error` block to keep the core rules readable, performant, and statically verifiable.
* Variables and context inside the `on_error` block are only evaluated if the error condition is actually triggered (saving CPU cycles on successful runs).

```yaml
on_error:
  - type: "missing_path"
    policy: "create"
  - type: "empty_match"
    policy: "ignore"
    message: "Optional component not present. Skipping."
    tags: ["ops", "optional-check"]
    context:
      container_index: "{loop.index}"
```

---

## Part 2: Path Selectors & Token Expansion
To avoid the risks of string-injection and parsing conflicts (e.g., when variable values contain dots, parentheses, or brackets), we use **AST-level Token Expansion** instead of naive string interpolation.

### 1. Syntax Separation: `Path::Metadata` (Double-Colon Delimiter)
We separate target configuration navigation (left of `::`) from engine metadata queries (right of `::`).
* **`.`** and **`[]`** navigation are strictly reserved for static document AST traversal.
* **`::`** is the boundary to query metadata from the engine (e.g., `path`, `type`, `value`, `index`).
* **`{}`** is the **unified syntax** for all dynamic token references (variables, registers, loop iterators). It is used in values, paths, and selectors alike.

#### Examples:
* `{vars.proxy_name}` (evaluates to `{vars.proxy_name::value}`)
* `$.metadata.labels.{vars.target_label_key}` -> Resolves the key name dynamically using the value of `{vars.target_label_key}` at runtime.
* `{register.legacy_panels[0]::path}` -> Returns the absolute path where the node was found (e.g., `"$.panels[3]"`).
* `{loop.panel::path}` -> Returns the absolute path of the current iteration.

### 2. Cardinality Safety: `find` vs `find_all`
* **`find`**: Strict single-node selection. Fails fast if multiple or zero nodes match.
* **`find_all`**: Collection selection. Returns a list of captured contexts (`[]CapturedNode`).
  * Accessing `@path` (now `::path`) directly on a list (e.g., `{register.my_list::path}`) triggers a `TypeMismatch` error.
  * Explicit index access is required: `{register.my_list[0]::path}`.

### 3. Safe Navigation Operator (`?.`)
To support optional fields and prevent verbose `if:` guards, path expressions support the safe navigation operator (`?.`).
* If a path segment prefixed with `?.` is missing or resolves to null (e.g., `{register?.debug_panel::path}`), the entire expression safely evaluates to `null` instead of raising a Fail-Fast exception.

---

## Part 3: Namespaces & Checkpoints
The namespaces of `ast-mutate` have strict R/W boundaries to ensure safety and isolation during execution.

### 1. Namespace Access Rights

| Namespace | Access Mode | Purpose |
| :--- | :--- | :--- |
| **`$`** (Workspace) | **R / W** | The primary mutation target, exported at the end. |
| **`origin`** | **Read-Only** | Copy of `$` at $t=0$, used for diffs/rollbacks. |
| **`draft.*`** | **R / W** | Isolated temporary workspaces (checkpoints / petri dishes). |
| **`register.*`** | **Write-Once / R** | Immutable clipboard populated via `register` actions. |
| **`source.*`** | **Read-Only** | Eager-loaded external JSON/YAML reference files (e.g., CMDB static exports). |
| **`vars.*`** | **Read-Only** | Input variables validated against the playbook contract. |
| **`loop.*`** | **Read-Only** | Volatile scope for active loop iterators. |

### 2. Checkpoints via Drafts
To keep the engine simple and prevent memory bloat, we do **not** introduce a separate `checkpoint` namespace. Since drafts are mutable and support full-document copies, creating a checkpoint is semantically identical to initializing a draft from the workspace root:
```yaml
- name: "Checkpoint workspace"
  action: "draft_create"
  target: "draft.backup"
  from: "$"
```

---

## Part 4: Rule Anatomy & Iteration
To enforce strict schema validation and prevent duplicate action declarations in YAML, rules explicitly separate the action verb from its parameters.

### 1. Action & Properties Schema
Each item in the playbook's `mutations` list defines its action and parameters using two explicit keys:
* **`action`** (string): The verb of the operation (e.g., `set`, `delete`, `inject`, `register`, `assert`, `skip`, `drop`, `for_each`).
* **`properties`** (map): The set of parameters specific to that action.
* Global metadata and flow controls remain at the root of the rule: `name`, `if`, `on_error`.

#### Example Structure:
```yaml
mutations:
  - name: "Set Title"
    action: "set"
    properties:
      target: "$.title"
      value: "cRSP {vars.proxy_name}"
    on_error:
      - type: "missing_path"
        policy: "create"
```

### 2. Loop Syntax (`for_each`)
A loop is declared using `action: "for_each"`. Its properties block contains loop parameters, and the nested `mutations` list declares the actions to run on each iteration:
```yaml
- name: "Loop over panels"
  action: "for_each"
  properties:
    on: "register.ts_panels"
    as: "panel"
  mutations:
    - name: "Delete panel node"
      action: "delete"
      properties:
        target: "{loop.panel::path}"
```

### 3. Volatile Scoping (`loop.*`)
Loop variables are registered under the `loop` namespace to prevent scope pollution.
* For each iteration, the iterator is accessible via `{loop.<alias>}` (e.g., `{loop.panel::value}`).
* In nested loops, variables are stacked, allowing inner scopes to access parent iterators without name masking (e.g., referencing `{loop.panel::path}` inside a sub-loop).

### 4. Stability under Mutation (Tombstone & Compaction)
To prevent index shifts from corrupting loop iterations when items are deleted from an array (e.g., deleting a panel during `for_each`), the engine employs a deferred deletion strategy:
* **Tombstoning**: During loop execution, `delete` actions on array elements do not resize the array in-place. Instead, targeted nodes are marked internally as deleted (tombstoned), keeping indices stable for `{loop.<alias>::path}`.
* **Compaction**: Upon completion of the loop step, the engine rebuilds the arrays, filtering out all tombstoned nodes.

---

## Part 5: Condition & Sandbox Evaluation (`if:`)
Expressions within `if:` blocks are evaluated using a **Dual-Mode Condition** architecture. The core execution engine only evaluates structured schema representations.

### 1. String Expressions (High-DX)
For human readability, standard string conditions are supported (e.g., `if: "loop.container.image == 'nginx:latest' and vars.env != 'prod'"`). The parser compiles this string into the structured Schema AST representation before execution.
* Truthy/falsy evaluations are supported (e.g., `if: "register.debug_panel"`).

### 2. Structured Schema (Machine-Readable / Complex Logic)
For complex nested conditions or machine generation, conditions can be declared directly as structured objects. If `if:` is an object, it must contain a `condition` key mapping to the root logical group of the structured condition tree.

#### Supported Logical Groups:
* `all` (AND logic)
* `any` (OR logic)
* `not` (NOT logic: negates the nested condition or group)

#### Supported Operators (`operator`):
`equals`, `not_equals`, `contains`, `starts_with`, `regex_match`, `greater_than`, `less_than`

#### Schema Example:
```yaml
if:
  condition:
    all:
      - operator: "equals"
        left: "loop.container.image"
        right: "nginx:latest"
      - operator: "not_equals"
        left: "vars.env"
        right: "prod"
```

### 3. Custom StdLib & Sandboxed Functions (`fct.*`)
The expression sandbox blocks access to host system primitives and arbitrary code execution to ensure safety and WASM compatibility.
* **Pure Functions only**: The sandbox exposes a restricted set of built-in pure functions prefixed with `fct.` (e.g., `fct.base64`, `fct.base64_decode`, `fct.sha256`, `fct.urlencode`, `fct.escape`).
* **No external execution**: Playbooks cannot declare custom functions executing external shell processes or binaries. All external logic must be executed out-of-band in pipeline workflows.

---

## Part 6: Streaming Execution Architecture
To guarantee $O(1)$ memory complexity and allow processing of multi-document YAML manifests or large JSONL log streams, the engine operates in a streaming fashion.
* The workspace **`$`** represents the AST of the *currently processed document* only, rather than the entire file.
* **`register.*`** and **`vars.*`** act as global, persistent spaces that persist throughout the entire stream, allowing cross-document data propagation.
* Two streaming control actions are supported:
  * **`skip`**: Skips all subsequent mutations on the current document, outputting it unchanged.
  * **`drop`**: Excludes the current document from the output stream.

---

## Part 7: Audit Trail & Traceability Design
Every execution run outputs line-delimited JSON logs distributed across logical destinations called **Sinks**.

### 1. Sinks (Abstract Destinations)
Sinks represent logical log channels (e.g., `audit`, `security`, `compliance`).
* File paths or outputs are never hardcoded in the playbook. Instead, the runtime environment (CLI flags or Server config) binds these logical sinks to actual output streams (e.g., `--sink security=/var/log/security.jsonl` or `--sink audit=stdout`).
* Unbound sinks are silently inactive at runtime.

### 2. Global Event Routing Pipeline
Every execution event/error is processed through a sequential routing pipeline defined in the playbook's `spec` to determine its destination sinks.
```yaml
spec:
  pipeline:
    - filter: "event.type == 'assertion_failed' and 'security' in event.tags"
      sink: "security"
    - filter: "event.tags contains 'compliance'"
      sink: "compliance"
    - filter: "any" # Catch-all rule
      sink: "audit"
```

### 3. Session and Instruction Identifiers
* **`run_id`** (UUIDv4): Generated at CLI/Server startup, identifying the execution session.
* **`instruction_id`** (UUIDv5 / Hash): A deterministic identifier for each playbook rule.
  * **Strategy (Option C)**: An optional `id` field can be defined in the playbook rule. If omitted, the engine automatically generates a deterministic UUIDv5 using the playbook name, the rule's index, and the rule's `name` property.

### 4. Log Line Payload representation
Each log entry contains:
```json
{
  "run_id": "a8b3f2c4-...",
  "instruction_id": "5e9d...",
  "timestamp": "2026-05-31T14:46:00Z",
  "document": { "index": 1, "kind": "Namespace", "name": "prod-alpha" },
  "path": "$.metadata.name",
  "action": "set",
  "before": "template-namespace",
  "after": "prod-alpha",
  "status": "success",
  "message": null
}
```

---

## Part 8: Policy & Verification Engine (`assert` action)
To support security and policy compliance without repeating conditional code blocks, the engine supports a native **`assert`** action.

### 1. Assertion Flow
An `assert` rule evaluates a condition (either in string or schema form) declared inside its `properties.if` field.
* **Root-level `if`**: Gating condition (e.g., determines if the rule executes at all).
* **Action-level `properties.if`**: The policy check logic (shares the same schema as the root-level `if`).
* If the assertion evaluates to `false`, the engine raises an **`assertion_failed`** error.
* The outcome is handled by the standard `on_error` block of the instruction, allowing blocking or non-blocking policy behaviors.

### 2. Example Declarations

#### Blocking Assertion (Default):
```yaml
- name: "Ensure deployment is non-root (Prod only)"
  action: "assert"
  if: "vars.env == 'prod'" # Gate: Runs the check only in production
  properties:
    if: "$.spec.template.spec.containers[*].securityContext.runAsNonRoot == true"
  on_error:
    - type: "assertion_failed"
      policy: "fail" # Aborts execution immediately
      message: "Security breach: Root container detected."
```

#### Non-blocking Assertion (Audit/Warning-only):
```yaml
- name: "Log warnings for dev configurations"
  action: "assert"
  properties:
    if: "$.spec.replicas > 1"
  on_error:
    - type: "assertion_failed"
      policy: "ignore" # Logs the message and continues execution
      message: "Warning: Deployment has only 1 replica."
```
