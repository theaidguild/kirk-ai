# Architecture

kirk-ai is organized to keep responsibilities separated and the codebase easy to reason about.

- `cmd/` — CLI command definitions and wiring (Cobra)
- `internal/client` — HTTP client for Ollama interactions
- `internal/templates` — Prompt templates used for code generation tasks
- `internal/models` — Request/response structs

The CLI follows a simple flow: parse flags → select model → call client → format output.

## Extending the CLI

Add a new command under `cmd/` and register it in `root.go`. Use existing helpers for model selection and error handling to keep behavior consistent.
