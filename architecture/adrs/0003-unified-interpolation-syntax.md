# ADR 0003: Unified Interpolation Syntax and Metadata Separation

## Status
Accepted

## Context
When mutating AST structures, the engine must resolve dynamic variables, registers, and loop iterators. Historically, combining path navigation with dynamic variable expansion led to two problems:
1. **Parser Complexity & Conflation**: Distinguishing a literal string segment from a variable reference requires unambiguous delimiters.
2. **Namespace Collision**: Accessing document fields vs engine metadata (like absolute path, element index, or node type) can conflict if they share the same naming conventions (e.g. if a JSON document has a key named `path`).

## Decision
We will adopt a unified interpolation syntax and a strict boundary between document traversal and engine reflection.

### 1. Unified `{}` Interpolation
All dynamic references (from namespaces `vars.*`, `register.*`, `loop.*`, etc.) are wrapped in curly braces `{}`. 
* Curly braces are the **only** symbol used to denote dynamic lookup, replacing previous drafts that proposed brackets `[]` for paths and braces for values.
* This applies uniformly to string values, targets, and selectors:
  - Value: `"myregistry.io/{loop.container.name}"`
  - Target Path: `"$.metadata.labels.{vars.target_label_key}"`

### 2. Nested (Recursive) Interpolation
Since the engine implements a custom parser, we support recursive nested evaluations. The parser evaluates tokens from the innermost braces outwards at runtime:
* Example: `{vars.{vars.env}_replicas}`
* If `{vars.env}` resolves to `"prod"`, the outer token becomes `{vars.prod_replicas}`, which evaluates to the final integer value.

### 3. Path vs Metadata Boundary (`::` Delimiter)
To separate target configuration traversal from engine metadata queries, we introduce the double-colon `::` delimiter:
* **Left of `::`**: Standard path navigation (`.` and `[]`) to traverse the JSON/YAML structure.
* **Right of `::`**: Metadata selectors queried from the engine (`path`, `type`, `value`, `index`).
* If `::` is omitted, the engine defaults to retrieving the payload value (`::value`).

#### Examples:
* `{vars.proxy_name}` is equivalent to `{vars.proxy_name::value}`.
* `{loop.panel::path}` returns the absolute path of the current iteration node (e.g. `"$.panels[3]"`).
* `{register.legacy_panels[0]::path}` returns the path of the first captured panel.
* `{loop.panel::type}` returns `"Mapping"`.

This boundary prevents collisions: accessing `{$.panels.0.path}` returns the JSON key `path` inside the panel, while `{$.panels.0::path}` returns the AST path of that panel.

## Consequences
* **Pros**:
  - Unified syntax: Developers only need to learn one format `{}` for all dynamic evaluations.
  - Safe escaping: Eliminates conflicts between document keys and engine variables.
  - Flexibility: Nested evaluations allow inline environment-variable indirections.
* **Cons**:
  - Requires recursive parsing logic in the custom lexer/parser, which increases initial implementation effort.
