# ADR 0008: Decoupled Audit Sinks and Event Routing Pipeline

## Status
Accepted

## Context
In enterprise SRE environments, different auditing files are required. For example, operational logs (successful mutations) should go to one system, while security violations (assertion failures) must go to a dedicated compliance audit file. 
Hardcoding file paths inside the playbook breaks portability, and evaluating metadata/tags on every rule adds CPU overhead.

## Decision
We will adopt a decoupled logging model based on **Sinks** and a **Global Event Routing Pipeline**. We will implement this as a reusable auditing package (`internal/audit`), designed to eventually be exported as the shared library `ouvrage-audit-go`.

### 1. Abstract Sinks
Playbooks define logic target streams called **Sinks** (e.g., `audit`, `security`, `compliance`).
* No paths are declared in the playbook.
* The execution environment binds these sinks at runtime:
  - CLI: `ast-mutate --sink security=/var/log/security.jsonl --sink audit=stdout`
  - Server: Config files map sinks to network destinations or local files.
* Unbound sinks are disabled automatically at runtime.

### 2. Global Event Routing Pipeline
Every execution event or error is processed through a sequential routing pipeline defined under `spec.pipeline` using dynamic filters:

```yaml
spec:
  pipeline:
    - filter: "event.type == 'assertion_failed' and 'security' in event.tags"
      sink: "security"
    - filter: "event.tags contains 'compliance'"
      sink: "compliance"
    - filter: "any" # Fallback
      sink: "audit"
```

### 3. Error-Level Enrichment
To keep the Happy Path clean and minimize CPU overhead, `tags` and `context` maps are defined exclusively inside the `on_error` block:
* If a mutation succeeds, no event is created and no context is evaluated.
* If an error occurs, the engine resolves the context variables (e.g. `{loop.index}`, `$.metadata.name`) and creates a rich event payload with the tags before sending it to the routing pipeline.

## Consequences
* **Pros**:
  - Decoupling: Playbooks remain completely environment-agnostic.
  - SecOps ready: Allows strict separation of security/compliance logs from standard operation audits.
  - Efficiency: Lazy-evaluation of context maps means no CPU overhead during successful nominal runs.
* **Cons**:
  - Introduces a global configuration block (`spec.pipeline`) which SREs must define to manage routing.
