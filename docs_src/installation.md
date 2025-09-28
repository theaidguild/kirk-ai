# Installation

## Requirements

- Go 1.19 or higher
- Ollama (optional) for model-based features

## Quickstart

1. Clone the repository

```bash
git clone https://github.com/theaidguild/kirk-ai.git
cd kirk-ai
```

2. Download dependencies and build

```bash
go mod download && go mod tidy

go build -v .
```

3. Run locally

```bash
./kirk-ai --help
```

For documentation preview:

```bash
pip install -r docs_requirements.txt
mkdocs serve -f mkdocs.yml
```
