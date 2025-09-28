# Commands

This project uses Cobra for CLI command structure. Below are examples and common flags.

Global flags (available to all commands):
- `--url` — Ollama server URL (default: `http://localhost:11434`)
- `--model` — explicitly choose a model (by default the CLI auto-selects a suitable model)
- `-v, --verbose` — enable verbose output (prints metadata and progress)
- `-s, --stream` — enable streaming mode where supported (prints partial model output as it arrives)


## chat

Use the `chat` command to send prompts to a selected model.

Basic usage:

```bash
./kirk-ai chat "Write a short poem about the sea"
```

Advanced examples:
- Stream a response and show metadata:

```bash
./kirk-ai chat "Explain polymorphism in simple terms" --stream --verbose
```

- Force a specific model (useful when you want a known configuration):

```bash
./kirk-ai chat "Generate unit test examples for a Go function" --model gemma3:4b
```

- Feed a long prompt from a file (shell substitution — safe for arbitrary text):

```bash
./kirk-ai chat "$(cat my_long_prompt.txt)" --verbose
```

Notes:
- `chat` requires at least one argument (the prompt). Use shell substitution to include multi-line prompts from files.
- When `--stream` is enabled the CLI prints chunks as they arrive and then a final newline; `--verbose` prints model/latency metadata.


## embed

Generate embeddings for text snippets. The `embed` command supports both single-text embeddings and embedding batches from an embeddings-ready JSON file.

Single text example (prints embedding vector to stdout):

```bash
./kirk-ai embed "Document text to embed"
```

Embed chunks from a prepared JSON file (recommended when you have many documents/chunks):
- Embed the first chunk (default behavior) from a file:

```bash
./kirk-ai embed --file tpusa_crawl/embeddings/tpusa_embeddings_ready.json
```

- Embed all chunks from a file and write full embedding objects to disk:

```bash
./kirk-ai embed --file tpusa_crawl/embeddings/tpusa_embeddings_ready.json --all --out out/embeddings_with_vectors.json
```

- Embed a specific chunk index from a file (0-based index):

```bash
./kirk-ai embed --file tpusa_crawl/embeddings/tpusa_embeddings_ready.json --chunk 42 --out out/chunk_42_embedding.json
```

- Tune performance and API usage:

```bash
./kirk-ai embed --file embeddings.json --all --concurrency 8 --batch-size 20 --rate 10.0 --out embeddings-out.json
```
  - `--concurrency` controls how many worker goroutines run in parallel
  - `--batch-size` controls how many chunks each worker collects before sending API calls
  - `--rate` sets a global requests-per-second limit (set to `0` to disable rate limiting)

Scripting tips:
- To embed many separate short texts from a file line-by-line you can combine shell tools with `xargs` or a loop:

```bash
cat texts.txt | while IFS= read -r line; do ./kirk-ai embed "${line}"; done
```
- Use `--out` when embedding from files to get a JSON with `id`, `chunk_index`, `content`, `metadata`, and `embedding` fields which is ideal for building a vector store.


## models

List models available from the Ollama server.

```bash
./kirk-ai models
```

Notes and examples:
- The command prints detected capabilities (e.g., embedding, code) and a recommended model for coding and embeddings.
- If no models are present the CLI will instruct you to `ollama pull <model-name>`.


## search

Search through an embeddings file using semantic similarity.

Basic usage:

```bash
./kirk-ai search "How do I configure CORS?" --embeddings out/embeddings_with_vectors.json
```

Advanced options:
- Return more results and lower the similarity threshold:

```bash
./kirk-ai search "privacy policy jurisdiction" --embeddings embeddings.json --top-k 10 --threshold 0.55
```

Notes:
- `--embeddings` is required and should point to a JSON file produced by `embed --out` (or otherwise containing `embedding` vectors).
- `--top-k` and `--threshold` allow you to tune recall vs precision for your semantic search.


## rag

Retrieval-augmented generation (RAG) — answer questions using embeddings as context.

Minimal example (requires embeddings file):

```bash
./kirk-ai rag "What is the organization's refund policy?" --embeddings out/embeddings_with_vectors.json
```

Useful flags and examples:
- Control how many chunks are combined as context:

```bash
./kirk-ai rag "Summarize the key benefits" --embeddings embeddings.json --context-size 5
```

- Make RAG more strict or permissive in choosing context by similarity threshold:

```bash
./kirk-ai rag "Who is the target audience?" --embeddings embeddings.json --similarity-threshold 0.65
```

- Progressive loading for large contexts (reduces startup latency):

```bash
./kirk-ai rag "Explain the onboarding process" --embeddings embeddings.json --context-size 60 --progressive
```

- Prefer faster (smaller) models when latency matters or override the model explicitly for RAG using `--rag-model`:

```bash
./kirk-ai rag "Provide a short answer" --embeddings embeddings.json --prefer-fast --rag-model gemma3:4b
```

Notes:
- `--rag-model` explicitly sets the chat model used for the RAG generation step and overrides the CLI's automatic RAG model selection. The global `--model` flag is a general-purpose flag for some commands, but `--rag-model` is the recommended way to choose the chat model for `rag` to ensure the behavior you expect.


## benchmark

Benchmark model performance across a small set of standardized prompts.

Basic usage (tests a chosen default or recommended coding model):

```bash
./kirk-ai benchmark
```

Advanced usage:
- Test all available models:

```bash
./kirk-ai benchmark --all
```

- Test a specific model (substring matching supported):

```bash
./kirk-ai benchmark --model gemma3:4b
```

- Run a quicker benchmark set:

```bash
./kirk-ai benchmark --quick
```

Notes:
- Benchmark prints response times and tokens/sec metrics and summarizes model reliability and speed when multiple models are tested.


## Tips & troubleshooting
- If you see "No models found" errors, install a model with Ollama: `ollama pull <model-name>` and re-run `./kirk-ai models`.
- Use `--verbose` to get timing and progress information that helps tune concurrency, batch sizes, and rate limits.
- For automation, prefer embedding a whole dataset (`--file` + `--all`) and writing `--out` once; then run `search` or `rag` against that single canonical embeddings file.
- The default Ollama URL is `http://localhost:11434`. Set `--url` to target a remote Ollama server if needed.

For more command-specific details, run the command with `--help` (e.g., `./kirk-ai embed --help`).
