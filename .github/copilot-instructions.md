# Copilot Instructions for Kirk-AI

## Repository Overview

Kirk-AI is a Go-based command-line interface for interacting with Ollama AI models. It provides chat interactions, text embeddings, code generation, translation, and model benchmarking capabilities. The project uses a clean architecture with modular design and comprehensive error handling.

**Repository Details:**
- **Language**: Go 1.25.1
- **Framework**: Cobra CLI framework for command structure
- **Size**: ~20 files, lightweight codebase
- **Target Runtime**: Cross-platform CLI tool
- **Dependencies**: Minimal - only `github.com/spf13/cobra v1.10.1`
- **Architecture**: Modular design with `cmd/`, `internal/` package structure

## Build and Validation

### Prerequisites
- Go 1.19 or higher (tested with Go 1.25.1)
- Ollama server running at `http://localhost:11434` (for functional testing only)
- At least one Ollama model installed (for functional testing only)

### Build Process

**Always run these commands in sequence:**

1. **Download dependencies** (always required before building):
   ```bash
   go mod download && go mod tidy
   ```

2. **Build the application**:
   ```bash
   go build -v .
   ```
   - Builds binary as `kirk-ai` (or `kirk-ai.exe` on Windows)
   - Build time: ~10-20 seconds
   - Alternative: `go build -o custom-name .` for custom binary name

3. **Alternative build with go run** (for development):
   ```bash
   go run main.go [command] [args]
   ```

**Makefile shortcut:**

The repository now includes a `Makefile` that mirrors the workflow above. Prefer the Make targets when you need a quick rebuild or want consistent formatting:

```bash
make deps build
```

Run `make help` to discover additional targets.

### Validation Commands

**Always run in this order:**

1. **Format code** (required before committing):
   ```bash
   go fmt ./...
   # or: make fmt
   ```

2. **Vet code** (required for CI):
   ```bash
   go vet ./...
   # or: make vet
   ```

3. **Run tests** (no test files currently exist):
   ```bash
   go test -v ./...
   # or: make test
   # Output: [no test files] for all packages
   ```

4. **Clean build artifacts**:
   ```bash
   go clean
   # or: make clean
   ```

### Functional Testing

**Note**: These commands require Ollama server running and will fail with connection errors if unavailable. This is expected behavior, not a build issue.

```bash
# Test CLI structure (always works)
./kirk-ai --help

# Quick checks via Make targets
make run
make build
make models
make check-ollama
# Test commands (require Ollama)
./kirk-ai models
./kirk-ai chat "hello" --verbose
./kirk-ai code "write a hello world function"
```

`make models` (and other binary-driven targets like `make benchmark`) expect the compiled binary in `build/kirk-ai`, so run `make build` beforehand.

**Expected errors without Ollama:**
- `dial tcp [::1]:11434: connect: connection refused`
- This is normal and not a build failure

### Build Validation Issues and Workarounds

1. **Import path issues**: The module name is `kirk-ai` - always use this in import statements
2. **Timeout behavior**: HTTP client has 120-second timeout for model operations
3. **Clean environment**: Run `go clean` between builds if changing build flags
4. **Format sensitivity**: Always run `go fmt ./...` before building to avoid style issues

## Project Layout and Architecture

### Directory Structure
```
kirk-ai/
├── main.go                    # Application entry point (9 lines)
├── go.mod                     # Go module definition
├── go.sum                     # Dependency checksums
├── .gitignore                 # Excludes binaries, build dirs, IDE files
├── cmd/                       # CLI commands (Cobra architecture)
│   ├── root.go               # Root command and global flags
│   ├── chat.go               # Chat with AI models
│   ├── code.go               # Code generation (optimized for gemma3:4b)
│   ├── translate.go          # Translation (optimized for gemma3:4b)
│   ├── embed.go              # Text embeddings
│   ├── models.go             # List available models
│   └── benchmark.go          # Performance benchmarking
└── internal/                  # Internal packages (no external API)
    ├── client/               # Ollama API client
    │   └── ollama.go        # HTTP client with 120s timeout
    ├── models/               # Data structures
    │   └── ollama.go        # Request/Response types
    ├── config/               # Model configuration
    │   └── models.go        # Model capabilities and priorities
    ├── templates/            # Prompt templates
    │   └── prompts.go       # Code generation, debugging templates
    └── errors/               # Custom error types
        └── errors.go        # APIError, NetworkError, ValidationError
```

### Key Architectural Elements

**CLI Command Structure:**
- Built with Cobra framework
- Global flags: `--url`, `--model`, `--verbose`, `--stream`
- Commands auto-select appropriate models when none specified
- Prefer `gemma3:4b` for coding tasks, embedding models for embeddings

**Model Selection Logic:**
- Priority-based model selection (see `internal/config/models.go`)
- `gemma3:4b` has priority 95 for coding tasks
- Automatic fallback to available models
- Embedding models filtered for chat tasks

**Error Handling:**
- Custom error types: `APIError`, `NetworkError`, `ValidationError`
- Comprehensive error messages with context
- Connection failures are graceful (expected when Ollama unavailable)

**HTTP Client Configuration:**
- Base URL: `http://localhost:11434` (Ollama default)
- Timeout: 120 seconds (for model loading/processing)
- No streaming implementation (placeholder for future)

### Configuration Files

**go.mod**: Defines module `kirk-ai` and Go 1.25.1 requirement
**go.sum**: Dependency checksums (only Cobra framework)
**.gitignore**: Excludes binaries (`kirk-ai`, `kirk-ai.exe`), build artifacts, IDE files

### Dependencies
- **Direct**: `github.com/spf13/cobra v1.10.1` (CLI framework)
- **Indirect**: `github.com/inconshreveable/mousetrap`, `github.com/spf13/pflag`
- **No external HTTP libraries**: Uses standard library `net/http`

## Validation Pipeline

**Current Status**: No CI/CD pipeline exists. Validation is manual only.

### Manual Validation Steps

1. **Code Quality**:
   ```bash
   go fmt ./...  # Must pass without changes
   go vet ./...  # Must pass without errors
   ```

2. **Build Verification**:
   ```bash
   go build -v .  # Must complete successfully
   # or: make build
   ./kirk-ai --help  # Must show help without errors
   # or: make run
   ```

3. **Binary Testing**:
   ```bash
   # Test command structure
   ./kirk-ai models --help
   ./kirk-ai chat --help
   ./kirk-ai code --help
   # or: make build && make models
   ```

### Adding Tests

**Current state**: No test files exist. When adding tests:
- Place test files as `*_test.go` alongside source files
- Use standard Go testing package
- Test error handling with mock Ollama responses
- Test model selection logic with mock model lists

## Key Implementation Notes

### Model Preferences
- **Code Generation**: Prefers `gemma3:4b` > other models > excludes embedding models
- **Chat**: Auto-selects first available non-embedding model
- **Embeddings**: Prefers models with "embed" in name
- **Translation**: Optimized for `gemma3:4b`

### Common Patterns
- All commands follow: validate input → select model → call API → format output
- Error handling is consistent across commands
- Verbose mode adds model selection and timing information
- Commands join multiple arguments with spaces for prompts

### Template System
Located in `internal/templates/prompts.go`:
- Code generation templates
- Debugging templates
- Optimization templates
- Template auto-selection based on keywords

### Build Artifacts to Exclude
Always add to `.gitignore`:
- `kirk-ai` (Linux/Mac binary)
- `kirk-ai.exe` (Windows binary)
- Any custom build names you create
- `/build/`, `/dist/` directories
- Coverage reports (`*.out`, `coverage.*`)

## Troubleshooting Guide

### Common Build Issues

1. **Module path errors**: Always use `kirk-ai/internal/...` in imports
2. **Dependency issues**: Run `go mod tidy` before building
3. **Format issues**: Run `go fmt ./...` before `go vet`

### Runtime Errors (Expected)

1. **Connection refused**: Normal when Ollama not running
2. **No models found**: Normal when no Ollama models installed
3. **Timeout errors**: Normal for very large models (120s limit)

### Development Workflow

1. Make code changes
2. Run `go fmt ./...` (or `make fmt`)
3. Run `go vet ./...` (or `make vet`)
4. Run `go build -v .` (or `make deps build`)
5. Test with `./kirk-ai --help` (or `make run`)
6. For functional testing, ensure Ollama running with models

## Agent Guidelines

**Trust these instructions** - they are validated and current. Only search/explore if:
- Instructions seem incomplete for your specific task
- You find contradictory information in the codebase
- Build/validation steps fail unexpectedly

**Key time savers:**
- Build process is straightforward: `go mod tidy && go build .` or `make deps build`
- No complex build scripts or configuration required
- Connection errors to Ollama are normal in development
- Focus on Go code quality (fmt, vet) rather than functional testing without Ollama