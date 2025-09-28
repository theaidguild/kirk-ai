# Commands

This project uses the Cobra CLI framework. The main commands are:

- `chat` — interact with chat models
- `embed` — generate vector embeddings
- `models` — list available models
- `benchmark` — run model benchmarks
- `code` — generate code using the recommended coding model

Each command supports `--model`, `--verbose`, and `--stream` flags where appropriate.

Example:

```bash
./kirk-ai chat "Explain recursion" --verbose
```
