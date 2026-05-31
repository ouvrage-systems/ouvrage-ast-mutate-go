# ADR 0009: Policy Verification Engine via Assertion Actions

## Status
Accepted

## Context
SRE and GitOps workflows need to verify configuration security and compliance policies (such as checking if a container runs as root, or if replicas are within bounds). 
We need a native, declarative validation mechanism. However, using the root-level `if:` block of a rule to perform the verification creates a major logic flaw: a policy violation (evaluating to false) would cause the engine to skip the rule entirely instead of raising a failure.

## Decision
We will support a native **`assert`** action. 

To maintain logical correctness, we strictly separate rule eligibility from assertion verification:
* **Root-level `if`**: Determines whether the rule runs at all (e.g. only in production).
* **Action-level `properties.if`**: The actual condition payload to verify against the document AST, sharing the exact same Dual-Mode Condition schema as the root-level `if`.

If the assertion `properties.if` evaluates to `false`, the engine raises an **`assertion_failed`** error. The outcome is handled by the standard `on_error` block of the instruction, allowing blocking or non-blocking policy behaviors.

### 1. Blocking Assertion (Default)
If the policy is omitted or set to `fail`, a failed assertion immediately halts execution:
```yaml
- name: "Ensure deployment is non-root (Prod only)"
  action: "assert"
  if: "vars.env == 'prod'" # Runs check only in production
  properties:
    if: "$.spec.template.spec.containers[*].securityContext.runAsNonRoot == true"
  on_error:
    - type: "assertion_failed"
      policy: "fail"
      message: "Security breach: Root container detected."
```

### 2. Non-blocking Assertion (Audit/Warning-only)
By setting `policy: ignore`, the engine logs the violation to the auditing pipeline and proceeds with the rest of the stream:
```yaml
- name: "Log compliance warnings"
  action: "assert"
  properties:
    if: "$.spec.replicas > 1"
  on_error:
    - type: "assertion_failed"
      policy: "ignore" # Non-blocking, logs warning and continues
      message: "Compliance warning: Deployment has only 1 replica."
      tags: ["compliance", "warning"]
```

## Consequences
* **Pros**:
  - Logical correctness: Separates rule gating (root-level `if`) from policy check (`properties.if`), preventing silent bypass of failing policies.
  - Unified syntax: The `properties.if` block supports both string expressions and structured schemas natively, reusing the existing Condition AST parser.
  - Policy-as-Code: Turns `ast-mutate` into a fast, lightweight, dependency-free validation engine.
* **Cons**:
  - Requires paying attention to which `if` check is at the root vs nested inside `properties`.
