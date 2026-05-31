# ADR 0010: Sandboxed Evaluation and Pure Function Library

## Status
Accepted

## Context
When performing calculations or string formatting inside conditions or values (e.g. converting a string to base64, calculating a sha256 checksum, or checking string properties), we must evaluate expressions at runtime. 
Allowing arbitrary shell commands or external binary execution (such as `action: exec` or custom process hooks) inside the engine:
1. Destroys the stateless, pure data-transformation function model.
2. Introduces severe security vulnerabilities (command injection).
3. Breaks portability, making it impossible to compile and run the engine in WebAssembly (WASM) environments (which have no access to host shells).

## Decision
We will enforce a strictly sandboxed, pure-function evaluation environment for all calculations, values, and conditions.

### 1. Sandboxed Evaluator
* The engine blocks all access to host operating system primitives, environment variables (except those explicitly declared in `vars.*`), disk I/O, network I/O, and process execution.
* The evaluation engine runs in a clean, isolated memory space.

### 2. Built-in Pure Standard Library (`fct.*`)
The evaluator exposes a dedicated, immutable set of pure helper functions prefixed with `fct.`:
* **Encoding**: `fct.base64(s)`, `fct.base64_decode(s)`, `fct.urlencode(s)`, `fct.urldecode(s)`
* **Hashing**: `fct.sha256(s)`
* **Escaping**: `fct.escape(s)`
* **Collections**: `fct.len(obj)`, `fct.is_empty(obj)`, `fct.has_key(obj, key)`

### 3. Rejection of Custom Shell Process Execution
Any requirement to invoke external binaries (such as running encryption tools like `sops` or retrieving data from Vault) must be executed **out-of-band** in pipeline workflows. 
The playbook cannot define or trigger shell execution hooks. The engine receives static data outputs from these external runs through its eager-loaded namespaces (`source.*` or `vars.*`).

## Consequences
* **Pros**:
  - Security: Guaranteed safety against malicious playbooks (e.g. they cannot read host files or run arbitrary commands).
  - WebAssembly (WASM) compatibility: The entire engine can be compiled to WASM and run directly in browsers or sandboxed gateways.
  - Reproducibility: Playbooks are pure functions; running them with the same inputs will always produce the exact same output.
* **Cons**:
  - Decreased flexibility: Users cannot write custom scripts directly inside the playbook and must rely on standard system orchestration tools to chain executions.
