package client

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"kirk-ai/internal/errors"
	"kirk-ai/internal/models"
)

// OllamaClient represents a client for interacting with Ollama API
type OllamaClient struct {
	BaseURL string
	Client  *http.Client
}

// NewOllamaClient creates a new Ollama client
func NewOllamaClient(baseURL string) *OllamaClient {
	return &OllamaClient{
		BaseURL: baseURL,
		Client: &http.Client{
			Timeout: 120 * time.Second, // Increased for model loading
		},
	}
}

// NewOllamaClientWithTimeout creates a new Ollama client with custom timeout
func NewOllamaClientWithTimeout(baseURL string, timeout time.Duration) *OllamaClient {
	return &OllamaClient{
		BaseURL: baseURL,
		Client: &http.Client{
			Timeout: timeout,
		},
	}
}

// Chat sends a chat request to Ollama and returns the response
func (c *OllamaClient) Chat(model, prompt string) (*models.ChatResponse, error) {
	if model == "" {
		return nil, errors.NewValidationError("model", "model cannot be empty")
	}
	if prompt == "" {
		return nil, errors.NewValidationError("prompt", "prompt cannot be empty")
	}

	request := models.ChatRequest{
		Model: model,
		Messages: []models.Message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Stream: false,
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, errors.NewNetworkError("marshal request", err)
	}

	resp, err := c.Client.Post(c.BaseURL+"/api/chat", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, errors.NewNetworkError("send request", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.NewNetworkError("read response", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, errors.NewAPIError(resp.StatusCode, string(body))
	}

	var chatResponse models.ChatResponse
	if err := json.Unmarshal(body, &chatResponse); err != nil {
		return nil, errors.NewNetworkError("unmarshal response", err)
	}

	return &chatResponse, nil
}

// Embedding generates embeddings for the given text using the specified model
func (c *OllamaClient) Embedding(model, text string) (*models.EmbeddingResponse, error) {
	if model == "" {
		return nil, errors.NewValidationError("model", "model cannot be empty")
	}
	if text == "" {
		return nil, errors.NewValidationError("text", "text cannot be empty")
	}

	request := models.EmbeddingRequest{
		Model:  model,
		Prompt: text,
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, errors.NewNetworkError("marshal request", err)
	}

	resp, err := c.Client.Post(c.BaseURL+"/api/embeddings", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, errors.NewNetworkError("send request", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.NewNetworkError("read response", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, errors.NewAPIError(resp.StatusCode, string(body))
	}

	var embeddingResponse models.EmbeddingResponse
	if err := json.Unmarshal(body, &embeddingResponse); err != nil {
		return nil, errors.NewNetworkError("unmarshal response", err)
	}

	return &embeddingResponse, nil
}

// ListModels gets the list of available models from Ollama
func (c *OllamaClient) ListModels() ([]string, error) {
	resp, err := c.Client.Get(c.BaseURL + "/api/tags")
	if err != nil {
		return nil, errors.NewNetworkError("send request", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.NewNetworkError("read response", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, errors.NewAPIError(resp.StatusCode, string(body))
	}

	var response models.ModelsResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, errors.NewNetworkError("unmarshal response", err)
	}

	modelNames := make([]string, len(response.Models))
	for i, model := range response.Models {
		modelNames[i] = model.Name
	}

	return modelNames, nil
}

// SelectChatModel automatically selects a suitable model for chat
// Deprecated: Use SelectModelByCapability instead
func (c *OllamaClient) SelectChatModel(models []string) string {
	return c.SelectModelByCapability(models, "chat")
}

// SelectEmbeddingModel automatically selects a suitable model for embeddings
// Deprecated: Use SelectModelByCapability instead
func (c *OllamaClient) SelectEmbeddingModel(models []string) string {
	return c.SelectModelByCapability(models, "embedding")
}

// SelectModelByCapability selects the best model for a given capability
func (c *OllamaClient) SelectModelByCapability(models []string, capability string) string {
	// This will be implemented using the config package
	// For now, maintain backward compatibility
	if capability == "embedding" {
		for _, model := range models {
			if strings.Contains(strings.ToLower(model), "embed") {
				return model
			}
		}
	} else if capability == "rag" {
		// For RAG, prefer faster, smaller models for better performance
		fastModels := []string{"llama3.2:1b", "gemma2:2b", "qwen2.5:1.5b", "llama3.2:3b"}
		for _, fast := range fastModels {
			for _, model := range models {
				if strings.Contains(strings.ToLower(model), fast) {
					return model
				}
			}
		}
		// Fallback to regular chat model selection
		capability = "chat"
	}

	if capability == "chat" {
		// Prefer gemma3:4b for chat and other tasks
		for _, model := range models {
			if strings.Contains(strings.ToLower(model), "gemma3") {
				return model
			}
		}
		// Fallback to non-embedding models
		for _, model := range models {
			if !strings.Contains(strings.ToLower(model), "embed") {
				return model
			}
		}
	}
	if len(models) > 0 {
		return models[0]
	}
	return ""
}

// ChatStream sends a streaming chat request to Ollama and calls the callback for each chunk
func (c *OllamaClient) ChatStream(model, prompt string, callback func(chunk *models.StreamingChatResponse) error) (*models.ChatResponse, error) {
	if model == "" {
		return nil, errors.NewValidationError("model", "model cannot be empty")
	}
	if prompt == "" {
		return nil, errors.NewValidationError("prompt", "prompt cannot be empty")
	}

	request := models.ChatRequest{
		Model: model,
		Messages: []models.Message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Stream: true, // Enable streaming
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, errors.NewNetworkError("marshal request", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", c.BaseURL+"/api/chat", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, errors.NewNetworkError("create request", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, errors.NewNetworkError("send request", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, errors.NewAPIError(resp.StatusCode, string(body))
	}

	scanner := bufio.NewScanner(resp.Body)
	var finalResponse *models.ChatResponse
	fullContent := ""

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var chunk models.StreamingChatResponse
		if err := json.Unmarshal([]byte(line), &chunk); err != nil {
			// Skip malformed chunks but don't fail
			continue
		}

		// Call the callback with the chunk
		if callback != nil {
			if err := callback(&chunk); err != nil {
				return nil, fmt.Errorf("callback error: %w", err)
			}
		}

		// Accumulate content
		fullContent += chunk.Message.Content

		// If this is the final chunk, save the metadata
		if chunk.Done {
			finalResponse = &models.ChatResponse{
				Model:              chunk.Model,
				CreatedAt:          chunk.CreatedAt,
				Message:            models.Message{Role: "assistant", Content: fullContent},
				Done:               true,
				TotalDuration:      chunk.TotalDuration,
				LoadDuration:       chunk.LoadDuration,
				PromptEvalCount:    chunk.PromptEvalCount,
				PromptEvalDuration: chunk.PromptEvalDuration,
				EvalCount:          chunk.EvalCount,
				EvalDuration:       chunk.EvalDuration,
			}
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, errors.NewNetworkError("read stream", err)
	}

	if finalResponse == nil {
		return nil, errors.NewNetworkError("incomplete response", fmt.Errorf("no final chunk received"))
	}

	return finalResponse, nil
}
