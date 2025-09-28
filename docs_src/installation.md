# Installation

Requirements:

- Go 1.19 or higher
- Ollama (optional, for functional testing)

Install from source:

```bash
# download dependencies
go mod download && go mod tidy

# build
go build -v .
```

Run directly for development:

```bash
go run main.go
```

If you want to serve docs locally during writing:

```bash
pip install mkdocs
mkdocs serve
```
