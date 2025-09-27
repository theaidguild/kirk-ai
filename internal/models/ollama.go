package models

import "time"

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest represents the request structure for Ollama chat API
type ChatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream"`
}

// ChatResponse represents the response from Ollama chat API
type ChatResponse struct {
	Model              string    `json:"model"`
	CreatedAt          time.Time `json:"created_at"`
	Message            Message   `json:"message"`
	Done               bool      `json:"done"`
	TotalDuration      int64     `json:"total_duration"`
	LoadDuration       int64     `json:"load_duration"`
	PromptEvalCount    int       `json:"prompt_eval_count"`
	PromptEvalDuration int64     `json:"prompt_eval_duration"`
	EvalCount          int       `json:"eval_count"`
	EvalDuration       int64     `json:"eval_duration"`
}

// EmbeddingRequest represents the request structure for Ollama embedding API
type EmbeddingRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

// EmbeddingResponse represents the response from Ollama embedding API
type EmbeddingResponse struct {
	Embedding []float64 `json:"embedding"`
}

// ModelsResponse represents the response from Ollama models API
type ModelsResponse struct {
	Models []Model `json:"models"`
}

// Model represents a single model in the models response
type Model struct {
	Name string `json:"name"`
}

// StreamingChatResponse represents a single chunk in a streaming response
type StreamingChatResponse struct {
	Model              string    `json:"model"`
	CreatedAt          time.Time `json:"created_at"`
	Message            Message   `json:"message"`
	Done               bool      `json:"done"`
	TotalDuration      int64     `json:"total_duration,omitempty"`
	LoadDuration       int64     `json:"load_duration,omitempty"`
	PromptEvalCount    int       `json:"prompt_eval_count,omitempty"`
	PromptEvalDuration int64     `json:"prompt_eval_duration,omitempty"`
	EvalCount          int       `json:"eval_count,omitempty"`
	EvalDuration       int64     `json:"eval_duration,omitempty"`
}
