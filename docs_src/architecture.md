# Architecture

kirk-ai follows a simple modular architecture:

- `cmd/` — Cobra commands that implement CLI entry points
- `internal/client` — Ollama HTTP client
- `internal/models` — request/response structs
- `internal/templates` — prompt templates used for code generation

The codebase intentionally keeps dependencies minimal and relies on the Go standard library for HTTP operations.

See the `internal/` folder for implementation details and the `cmd/` folder for how commands are wired together.
