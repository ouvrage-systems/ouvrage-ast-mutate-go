# ADR 0001: Tri-modal Architecture and Go AST Mutation Engine

## Status
Accepted

## Context
In the Ouvrage ecosystem, we need a declarative engine (`ast-mutate`) to safely mutate JSON/YAML configurations (such as Grafana dashboards, Kubernetes manifests, or custom application configs). The requirements are:
1. Parse JSON/YAML files into an Abstract Syntax Tree (AST).
2. Apply deterministic mutation playbooks based on an intermediate representation (Ouvrage IR).
3. Write back the modified files without corrupting original formatting, comments, or key orders.
4. Provide strict observability by exposing Prometheus `/metrics` at runtime.
5. Offer flexible execution modes:
   - **Library (Lib)**: For in-memory integration with other Go tools (e.g., `ouvrage-lutrin`).
   - **CLI**: For developers and CI/CD automation.
   - **Server**: A REST API microservice for centralizing configuration transformation.

Since Go is the lingua franca of SRE and Platform Engineering, the project will be developed in Go.

## Decision
We will structure the project using Hexagonal Architecture principles to keep the core mutation logic decoupled from the transport/delivery layers (CLI, Server, Library).

### 1. Technology Choices
* **AST Parser**: We will use `gopkg.in/yaml.v3`. Its `yaml.Node` representation acts as a concrete AST that retains document layout, styling, line numbers, and comments. Since JSON is a subset of YAML, it can parse and format JSON files seamlessly.
* **CLI Framework**: We will use `github.com/spf13/cobra` to build the command-line interface.
* **Metrics**: We will use the official Prometheus Go client library (`github.com/prometheus/client_golang`) to register and expose metrics.

### 2. Codebase Layout
The repository will be organized as follows:
```text
.
├── architecture/
│   └── adrs/                    # Architecture Decision Records (ADRs)
│   └── design-meeting-notes.md  # Unified specifications and meeting logs
├── internal/
│   ├── core/                    # Pure domain logic (AST parsing, mutation engine, sandbox)
│   │   ├── model/               # Playbook definitions, engine context, and condition nodes
│   │   ├── parser/              # Playbook, condition expression, and path parser
│   │   ├── evaluator/           # Sandboxed expression & function evaluator, token expansion
│   │   └── mutator/             # Mutation & loop execution runner
│   ├── audit/                   # Decoupled audit event routing and sink management
│   └── adapters/
│       ├── cli/                 # Cobra command-line adapter
│       ├── server/              # Stateless HTTP/2 streaming server (OSP protocol)
│       └── mcp/                 # Model Context Protocol (MCP) server adapter
├── pkg/
│   └── sdk/                     # Go Library wrapper for integration with external tools
├── cmd/
│   └── ast-mutate/              # Single entrypoint binary supporting CLI, Server, and MCP modes
├── go.mod
└── go.sum
```

### 3. Traceability (Audit Trail)
Every mutation operation will generate structured logs written to an `audit-mutations.jsonl` file in JSON Lines format. This file tracks the exact execution trace (timestamp, target file, rule applied, targeted path, old value, new value, and outcome status).

### 4. Observability
Every execution mode (specifically CLI runs and the HTTP Server daemon) will register core metrics using a shared Prometheus registry. The HTTP Server will expose a `/metrics` endpoint.

## Consequences
* **Pros**:
  - High testability: Core logic can be unit-tested without mocking HTTP servers or CLI contexts.
  - Safe mutations: Leverages the proven AST parser of `yaml.v3` to preserve comments and formatting.
  - Versatility: Single binary supports SDK imports, CLI pipelines, and web service deployments.
* **Cons**:
  - Increased boilerplate due to hexagonal separation.
  - Relying on `yaml.v3` means parsing performance is bound to the library's throughput, but for configuration mutation (which is not in the hot path of request routing), safety and comment preservation are much higher priorities than microsecond performance.
