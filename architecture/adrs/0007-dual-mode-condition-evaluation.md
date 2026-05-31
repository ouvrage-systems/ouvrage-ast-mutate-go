# ADR 0007: Dual-Mode Condition Evaluation

## Status
Accepted

## Context
Playbooks require conditional execution (`if:` guards). We must choose between:
1. **String Expressions** (e.g. `"vars.env == 'prod'"`): Very concise and natural for humans, but hard to parse and validate statically.
2. **Structured Objects** (e.g. MongoDB/ES style queries): Easy to generate programmatically and validate using schemas, but extremely verbose for simple human-written checks.

## Decision
We will implement a **Dual-Mode Condition Evaluation** architecture. The core execution engine only evaluates structured schema trees. The frontend parser accepts both formats, cleanly distinguishing between them by checking the type of the `if:` value.

### 1. High-DX String Mode (Human-Readable)
If the `if:` property is a string, it is treated as a logical expression:
```yaml
if: "(vars.env == 'prod' or vars.env == 'staging') and $.metadata.name != 'legacy-auth'"
```
The playbook compiler parses this string and translates it into the equivalent structured schema before running the execution loop.

### 2. Structured Schema Mode (Machine-Readable)
If the `if:` property is an object, it must contain a `condition` key mapping to the root logical group of the structured condition tree.

#### Supported Logical Groups:
* `all` (AND logic: list of conditions)
* `any` (OR logic: list of conditions)
* `not` (NOT logic: negates the nested condition or group)

#### Supported Operators (`operator`):
`equals`, `not_equals`, `contains`, `starts_with`, `regex_match`, `greater_than`, `less_than`

#### Schema Example:
```yaml
if:
  condition:
    all:
      - any:
          - operator: "equals"
            left: "vars.env"
            right: "prod"
          - operator: "equals"
            left: "vars.env"
            right: "staging"
      - operator: "not_equals"
        left: "$.metadata.name"
        right: "legacy-auth"
```

### 3. CLI Compilation Utilities
The CLI will expose helper commands (e.g. `ast-mutate tool parse-condition`) to read string expressions and print their structured YAML schemas, helping developers debug complex rules.

## Consequences
* **Pros**:
  - Best of both worlds: String format is concise for developers, while the structured format is robust for machine generation and static validation (KCL).
  - Explicit distinction: Wrapping the structured condition under `condition:` prevents parser ambiguity and makes schema validation cleaner.
  - Simpler Core Engine: The evaluator only has to process structured Go structs representing the schema tree.
* **Cons**:
  - Requires writing a robust frontend string expression compiler in the Go parser package.
  - Introduces one level of nesting (`condition:`) for the structured mode, increasing verbosity slightly.
