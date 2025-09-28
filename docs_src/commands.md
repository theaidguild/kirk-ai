# Commands

This project uses Cobra for CLI command structure. Below are examples and common flags.

## chat

Use the `chat` command to send prompts to a selected model.

```bash
./kirk-ai chat "Write a short poem about the sea"
```

Flags:
- `--model` — specify model name
- `--verbose` — show metadata

## embed

Generate embeddings for text snippets.

```bash
./kirk-ai embed "Document text to embed"
```

## models

List models available from the Ollama server.

```bash
./kirk-ai models
```

## code

Generate code snippets or helper functions using the recommended coding model.

```bash
./kirk-ai code "Create a function that reverses a string"
```
