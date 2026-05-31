# ADR 0013: Unified Documentation System and Repository Layout

## Status
Accepted

## Context
In the Ouvrage ecosystem, projects will span multiple programming languages (Go, Python, Rust). We need a unified documentation portal with a consistent design, global search, and cross-project accessibility.
We must also prevent "configuration file soup" at the root of the repository, ensuring that Go-related files (`go.mod`), Python-related files, and documentation-related config files (`mkdocs.yml`) do not clutter the workspace root.

## Decision
We will adopt **Material for MkDocs** as the unified documentation framework for the repository.
To maintain an ultra-clean repository root, we will isolate the documentation build system inside a dedicated `docs/` directory using the **Fully Isolated Docs Folder** pattern.

### 1. Repository Layout Specification

The repository will be structured as follows, keeping the root free of documentation configuration files:

```text
.
├── architecture/            # Internal architecture records (ADRs, meeting notes)
├── docs/                    # Isolated documentation portal
│   ├── mkdocs.yml           # MkDocs configuration (defines docs_dir: src)
│   ├── requirements.txt     # Python documentation dependencies (mkdocs-material, etc.)
│   └── src/                 # Source Markdown files for the website
│       ├── index.md         # Documentation landing page
│       ├── adrs/            # Symlinked or copied ADRs for publishing
│       └── api/             # Generated Go/Python API docs (via gomarkdoc, etc.)
├── LICENSE
├── README.md                # Simple landing page pointing to docs/ or build steps
└── go.mod                   # (When initialized) Go module at the root
```

### 2. Key Rules
* **No `mkdocs.yml` or `requirements.txt` at the root**: All documentation-related tools, requirements, and configurations must live strictly inside `docs/`.
* **Isolated `docs_dir`**: The `mkdocs.yml` file will configure the source directory to point to `src/` (e.g. `docs_dir: src`).
* **Source Generation**: Code API documentations (e.g. generated via `gomarkdoc` for Go, or `mkdocstrings` for Python) will be outputted directly into `docs/src/api/` during the build step.

## Consequences
* **Pros**:
  - **Pristine Root**: The repository root remains clean and focused solely on the project code (`go.mod`, code folders) and basic Git files.
  - **Language Isolation**: Python dependencies for MkDocs are isolated in `docs/requirements.txt`, avoiding conflict with root configurations.
  - **Standardized CI/CD**: The documentation website is built by running `mkdocs build -f docs/mkdocs.yml`, keeping the build command decoupled.
* **Cons**:
  - Requires explicit `-f docs/mkdocs.yml` flags when running MkDocs commands locally.
