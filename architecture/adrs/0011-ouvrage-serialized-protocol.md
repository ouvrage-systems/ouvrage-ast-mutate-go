# ADR 0011: Ouvrage Serialized Protocol over HTTP/2 YAML Stream

## Status
Accepted

## Context
When running `ast-mutate` in Server Mode (REST API), the engine must receive several inputs to perform a transformation:
1. The playbook containing the mutation instructions.
2. Input variables (`vars.*`).
3. External reference files (`source.*` assets).
4. The target configuration stream to mutate (`$`).

Using a multi-step API (e.g. creating a context, uploading assets, executing, and retrieving outputs) forces the server to become **stateful**. This introduces session management, cache cleanup complexity, and security risks (disk-filling DDoS).
Conversely, sending everything in a single JSON body requires loading the entire payload into memory, defeating the $O(1)$ streaming memory guarantee.

## Decision
We will validate and implement the **Ouvrage Serialized Protocol (OSP)**. The entire execution payload is packaged into a single, unified, stateless YAML stream sent via a single HTTP POST request.

### 1. Protocol Stream Structure
The request body (`Content-Type: application/x-yaml`) consists of sequential YAML documents separated by standard `---` markers:

1. **Document 1 (Header/Envelope)**: A `MutationPackage` object describing the metadata, input variables, and the format of the incoming payload.
2. **Document 2 (Playbook)**: The `MutationPlaybook` resource containing the rules.
3. **Documents 3 to N (Sources)**: Eager-loaded static assets (JSON/YAML mappings) mapped to their namespaces.
4. **Delimiter**: A final `---` marks the end of the metadata block.
5. **Stream Payload ($)**: The actual target documents to mutate.

### 2. Format Transition
The envelope header specifies the format of the target payload (`spec.payload_format`):
* **YAML mode**: The engine continues decoding YAML documents separated by `---`.
* **JSONL mode**: The engine switches to a line-by-line reader, parsing each line as an independent JSON object.

```text
HTTP POST /v1/mutate
Content-Type: application/x-yaml
Transfer-Encoding: chunked

---
apiVersion: ouvrage.io/v1alpha1
kind: MutationPackage
spec:
  payload_format: "jsonl"
  vars: { env: "prod" }
---
apiVersion: ouvrage.io/v1alpha1
kind: MutationPlaybook
...
---
# First data document (JSONL line)
{"title": "D1", "panels": []}
{"title": "D2", "panels": []}
```

### 3. HTTP/2 and Streaming Compliance
The protocol leverages standard **HTTP/1.1 Chunked Transfer Encoding** (via RFC 9112) and **HTTP/2 DATA frames** (via RFC 7540).
* Since Go's `net/http` abstracts the request body as an `io.Reader`, the engine decodes documents incrementally from the network socket as they arrive.
* This maintains a strict $O(1)$ memory consumption and avoids temporary file storage on the server.

## Consequences
* **Pros**:
  - Stateless Server: The server does not persist sessions or file caches.
  - Performance: Zero-copy, direct pipeline streaming from the network socket.
  - UNIX pipeline friendly: Easily crafted in shell scripts by concatenating files and sending them using `curl --data-binary @-`.
* **Cons**:
  - Unconventional for standard REST API consumers (requires client tools that understand YAML stream payloads).
