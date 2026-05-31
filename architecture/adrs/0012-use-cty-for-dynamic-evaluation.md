# ADR 0012: Use of HashiCorp's `go-cty` for Dynamic Type and Expression Evaluation

## Status
Accepted

## Context
In ADR 0007 (Condition Evaluation) and ADR 0010 (Sandboxed Functions Stdlib), we established the need for a sandboxed expression evaluator. We must support dynamic, runtime type evaluations (e.g. comparisons, string operations, functions stdlib) while resolving inputs from the playbook's variables and registers.

We face a design choice between:
1. **Manual Go Reflection and Type Assertions**: Writing custom type assertion switches (`switch v := val.(type)`) and manual type-coercion logic for operations.
2. **Adopting `go-cty` (`github.com/zclconf/go-cty`)**: Leveraging the dynamic type system developed by HashiCorp for HCL2/Terraform.

## Decision
We will use **`go-cty`** as the core dynamic type representation and execution engine for expression evaluation, variable resolution, and sandboxed stdlib functions in `ast-mutate`.

This decision amends the implementation path for ADR 0007 and ADR 0010.

### Comparison: With `go-cty` vs. Without `go-cty`

| Feature / Scenario | With `go-cty` | Without `go-cty` (Manual Go `interface{}`) |
| :--- | :--- | :--- |
| **Type Coercion & Math** | **Automated & Predictable**: Natively handles number representations (`cty.Number` is backed by `math/big`) and prevents silent rounding or overflow errors. | **Brittle**: Requires writing manual casting rules for all permutations of `int`, `int64`, `float64`, `string` to prevent runtime mismatches. |
| **Type-Safety Interception** | **Safe Panics / Errors**: Incompatible operations (e.g., comparing a string to an integer) fail with structured, catchable type errors rather than crashing the process. | **Error-Prone**: Requires wrapping every reflective comparison in manual boundary checks to prevent standard Go runtime crashes. |
| **Dry-Run Validation** | **Unknown Values**: Natively supports `cty.Unknown` values. A dry-run can evaluate a conditional block with missing input variables, returning `Unknown` rather than failing or returning a false negative. | **Fake Placeholders**: Requires injecting dummy values (e.g. `""` or `0`), which can cause downstream assertions to fail, producing false positive warnings. |
| **Sandboxed Functions** | **Standardized API**: Custom functions (e.g., `fct.sha256`, `fct.base64`) are declared using `cty.Function` with strict input type signatures, returning typed values. | **Manual Reflection**: Requires writing custom reflection wrappers for every stdlib function, manually validating argument slices. |
| **Dependencies** | **One External Dependency**: Introduces `github.com/zclconf/go-cty`. | **Zero Dependencies**: Keeps the codebase free of third-party typing frameworks. |

## Consequences
* **Pros**:
  - Reuses a battle-tested type engine (used by Terraform to manage complex cloud states), ensuring high reliability.
  - Simplifies the writing of sandboxed functions and conditional expressions.
  - Provides a elegant dry-run mode via `cty.Unknown`.
* **Cons**:
  - Requires translating Go values parsed by `yaml.v3` into `cty.Value` before evaluation, adding a thin conversion layer in the parser package.
