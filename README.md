# Kirk-AI CLI

A command-line interface for interacting with Ollama AI models, supporting both chat interactions and text embeddings.

## Prerequisites

- Go 1.19 or higher
- Ollama installed and running locally
- At least one Ollama model installed

## Installation

1. Clone or navigate to this project directory
2. Ensure Ollama is running (default: `http://localhost:11434`)
3. Make sure you have at least one model installed in Ollama

## Quick Start

### Install a model (if you haven't already)
```bash
# Install a lightweight model
ollama pull llama2:7b

# Or install other models like:
# ollama pull codellama:7b
# ollama pull mistral:7b
```

### Run the application
```bash
go run main.go
```

## Project Structure

```
kirk-ai/
├── main.go                    # Application entry point
├── go.mod                     # Go module file
├── go.sum                     # Go module dependencies
├── cmd/                       # CLI commands (Cobra)
│   ├── root.go               # Root command and global flags
│   ├── chat.go               # Chat command
│   ├── embed.go              # Embedding command
│   └── models.go             # Models listing command
├── internal/                  # Internal packages
│   ├── client/               # Ollama API client
│   │   └── ollama.go        # HTTP client implementation
│   ├── models/               # Data structures
│   │   └── ollama.go        # Request/Response models
│   └── errors/               # Custom error types
│       └── errors.go        # Error definitions
└── README.md                 # This file
```

## Features

- **Chat**: Send text prompts to AI models and receive responses
- **Embeddings**: Generate vector embeddings for text using embedding models (e.g., EmbeddingGemma)
- **Model Management**: List available models in your Ollama installation
- **Auto Model Selection**: Automatically chooses appropriate models for different tasks
- **Verbose Output**: Optional detailed metadata about responses
- **Clean Architecture**: Well-organized code with separation of concerns
- **Error Handling**: Comprehensive custom error types and handling

## Usage Examples

### Basic Commands

```bash
# List available models
./kirk-ai models

# Chat with AI (auto-selects a chat model)
./kirk-ai chat "Hello! Tell me a joke"

# Generate embeddings (auto-selects an embedding model)
./kirk-ai embed "This is some text to embed"

# Use a specific model
./kirk-ai chat "Explain quantum physics" --model llama3.1:8b

# Enable verbose output
./kirk-ai chat "Hello" --verbose
```

### Chat Examples

```bash
# Simple chat
./kirk-ai chat "What is machine learning?"

# Chat with specific model and verbose output
./kirk-ai chat "Explain neural networks" --model llama3.2:3b --verbose

# Multi-word prompts (quotes recommended)
./kirk-ai chat "Tell me about the history of artificial intelligence"
```

### Embedding Examples

```bash
# Generate embeddings for text
./kirk-ai embed "Natural language processing"

# Use specific embedding model
./kirk-ai embed "Machine learning concepts" --model embeddinggemma:latest --verbose
```

## API Reference

### OllamaClient

#### Methods

- `NewOllamaClient(baseURL string) *OllamaClient`: Creates a new client instance
- `Chat(model, prompt string) (*ChatResponse, error)`: Sends a chat request
- `ListModels() ([]string, error)`: Gets available models

#### Structures

- `ChatRequest`: Request structure for chat API
- `ChatResponse`: Response structure from chat API  
- `Message`: Individual message structure

## Configuration

### Default Settings

- **Ollama URL**: `http://localhost:11434`
- **HTTP Timeout**: 30 seconds
- **Streaming**: Disabled (for simplicity)

### Environment Variables

You can modify the code to support environment variables:

```go
baseURL := os.Getenv("OLLAMA_URL")
if baseURL == "" {
    baseURL = "http://localhost:11434"
}
```

## Troubleshooting

### Common Issues

1. **"Connection refused"**: Make sure Ollama is running (`ollama serve`)
2. **"No models found"**: Install a model using `ollama pull <model-name>`
3. **Timeout errors**: Increase the HTTP client timeout for large models

### Debugging

Enable verbose output by adding debug logging:

```go
import "log"

// Add before API calls
log.Printf("Sending request to: %s", c.BaseURL+"/api/chat")
```

## Next Steps

### Suggested Enhancements

1. **Streaming Support**: Implement streaming responses for real-time output
2. **Configuration File**: Add YAML/JSON config for settings
3. **CLI Interface**: Add command-line arguments for interactive use
4. **Conversation History**: Implement conversation persistence
5. **Model Management**: Add functions to pull/remove models
6. **Web Interface**: Create a simple web UI

### Example: Adding Streaming

```go
func (c *OllamaClient) ChatStream(model, prompt string, callback func(string)) error {
    request := ChatRequest{
        Model: model,
        Messages: []Message{{Role: "user", Content: prompt}},
        Stream: true,
    }
    
    // Implementation for streaming...
}
```

## GitHub Pages

This repository includes a simple static site served from the `docs/` folder and an automated GitHub Actions workflow that deploys it on pushes to the `main` branch (`.github/workflows/pages.yml`). After the first successful deployment the site will be available at:

https://theaidguild.github.io/kirk-ai/

If the site does not appear automatically, open the repository Settings → Pages and ensure the source is set to "Deploy from a GitHub Action" or to the `docs/` folder on the `main` branch.

To update the site, edit files under `docs/` (for example `docs/index.html`) and push to `main`.

## Contributing

Feel free to extend this basic client with additional features like:
- Better error handling
- Configuration management
- Multiple conversation threads
- Model fine-tuning support

## License

This is a basic example project. Use and modify as needed for your projects.