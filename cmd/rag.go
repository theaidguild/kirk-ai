package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"kirk-ai/internal/client"

	"github.com/spf13/cobra"
)

var (
	ragEmbeddingsFile      string
	ragContextSize         int
	ragSimilarityThreshold float64
	ragMaxContextLength    int
	ragProgressive         bool
	ragTimeout             int
)

var ragCmd = &cobra.Command{
	Use:   "rag [question]",
	Short: "Answer questions using retrieval-augmented generation",
	Long:  `Use semantic search to find relevant context from embeddings and generate informed answers using RAG (Retrieval-Augmented Generation).`,
	Args:  cobra.MinimumNArgs(1),
	Run:   runRAGCommand,
}

func runRAGCommand(cmd *cobra.Command, args []string) {
	start := time.Now()
	question := strings.Join(args, " ")

	if ragEmbeddingsFile == "" {
		fmt.Println("Please specify embeddings file with --embeddings flag")
		os.Exit(1)
	}

	// Load embeddings with content
	loadStart := time.Now()
	embeddings, err := loadEmbeddings(ragEmbeddingsFile)
	if err != nil {
		fmt.Printf("Error loading embeddings: %v\n", err)
		os.Exit(1)
	}

	if verbose {
		fmt.Printf("Loaded %d embeddings for RAG in %v\n", len(embeddings), time.Since(loadStart))
	}

	// Generate embedding for question
	embedStart := time.Now()
	queryEmbedding, err := generateQueryEmbedding(question)
	if err != nil {
		fmt.Printf("Error generating query embedding: %v\n", err)
		os.Exit(1)
	}

	if verbose {
		fmt.Printf("Generated query embedding in %v\n", time.Since(embedStart))
	}

	// Determine context size and similarity threshold based on configuration
	contextSize := ragContextSize
	similarityThreshold := ragSimilarityThreshold

	// Progressive loading: start with smaller context for large requests
	if ragProgressive && ragContextSize > 10 {
		contextSize = ragContextSize / 3
		if contextSize < 5 {
			contextSize = 5
		}
		similarityThreshold = 0.5 // More aggressive filtering for progressive loading
		if verbose {
			fmt.Printf("Using progressive context loading: starting with %d chunks (threshold: %.2f)\n", contextSize, similarityThreshold)
		}
	}

	// Dynamic similarity threshold based on context size
	if similarityThreshold == 0.0 {
		if ragContextSize > 20 {
			similarityThreshold = 0.5 // More aggressive for large contexts
		} else {
			similarityThreshold = 0.3 // Default threshold
		}
	}

	// Search for relevant context
	searchStart := time.Now()
	results := searchSimilar(queryEmbedding, embeddings, contextSize, similarityThreshold)

	if verbose {
		fmt.Printf("Search completed in %v (found %d results with threshold %.2f)\n",
			time.Since(searchStart), len(results), similarityThreshold)
	}

	if len(results) == 0 {
		fmt.Printf("No relevant context found for question: %s\n", question)
		fmt.Printf("Try lowering the similarity threshold (current: %.2f) or asking a different question.\n", similarityThreshold)
		return
	}

	// Build context with length limit
	contextStart := time.Now()
	var contextParts []string
	totalLength := 0
	maxLength := ragMaxContextLength
	if maxLength == 0 {
		maxLength = 8000 // Default max context length
	}

	for _, result := range results {
		content := getContentFromEmbedding(result.Item)
		if content != "" {
			if totalLength+len(content) > maxLength {
				// Truncate content to fit within limit
				remaining := maxLength - totalLength
				if remaining > 100 { // Only add if meaningful
					content = content[:remaining] + "..."
					contextParts = append(contextParts, content)
					totalLength += len(content)
				}
				break
			}
			contextParts = append(contextParts, content)
			totalLength += len(content)
		}
	}

	if len(contextParts) == 0 {
		fmt.Println("Found similar embeddings but no content available for context.")
		fmt.Println("Make sure your embeddings file includes content data.")
		return
	}

	context := strings.Join(contextParts, "\n\n")

	if verbose {
		fmt.Printf("Context built in %v (%d characters, %d chunks)\n",
			time.Since(contextStart), len(context), len(contextParts))
	}

	// Generate answer using context with custom timeout if specified
	answerStart := time.Now()
	answer, err := generateRAGAnswerWithTimeout(question, context, time.Duration(ragTimeout)*time.Second)
	if err != nil {
		fmt.Printf("Error generating answer: %v\n", err)
		os.Exit(1)
	}

	if verbose {
		fmt.Printf("Answer generated in %v\n", time.Since(answerStart))
	}

	// Display results
	fmt.Printf("Question: %s\n", question)
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("Answer: %s\n", answer)

	if verbose {
		fmt.Printf("\nPerformance Summary:\n")
		fmt.Printf("- Total time: %v\n", time.Since(start))
		fmt.Printf("- Context used: %d chunks (%.2f similarity threshold)\n", len(results), similarityThreshold)
		for i, result := range results {
			fmt.Printf("  [%d] Chunk %d (similarity: %.3f)\n",
				i+1, result.Item.ChunkIndex, result.Similarity)
		}
		fmt.Printf("- Context length: %d characters (max: %d)\n", len(context), maxLength)
	}
}

func getContentFromEmbedding(item embeddingItem) string {
	// First try direct content field
	if item.Content != "" {
		return item.Content
	}

	// Try to extract content from metadata
	if item.Metadata != nil {
		if content, ok := item.Metadata["content"].(string); ok && content != "" {
			return content
		}
	}

	return ""
}

func generateRAGAnswer(question, context string) (string, error) {
	return generateRAGAnswerWithTimeout(question, context, 0)
}

func generateRAGAnswerWithTimeout(question, context string, timeout time.Duration) (string, error) {
	// Select chat model optimized for RAG
	models, err := ollamaClient.ListModels()
	if err != nil {
		return "", err
	}

	// Use RAG-optimized model selection
	selectedModel := ollamaClient.SelectModelByCapability(models, "rag")
	if selectedModel == "" {
		// Fallback to regular chat model
		selectedModel = selectChatModel(models)
	}
	if selectedModel == "" {
		return "", fmt.Errorf("no suitable chat model found")
	}

	if verbose {
		fmt.Printf("Using RAG-optimized model: %s\n", selectedModel)
	}

	// Build RAG prompt
	prompt := fmt.Sprintf(`Based on the following context, please answer the question. Be concise and accurate. If the answer is not clearly available in the context, say so.

Context:
%s

Question: %s

Answer:`, context, question)

	// Use custom client with timeout if specified
	if timeout > 0 {
		// Create client with custom timeout
		customClient := client.NewOllamaClientWithTimeout(baseURL, timeout)
		chatResponse, err := customClient.Chat(selectedModel, prompt)
		if err != nil {
			return "", err
		}
		return chatResponse.Message.Content, nil
	} else {
		// Use default client
		chatResponse, err := ollamaClient.Chat(selectedModel, prompt)
		if err != nil {
			return "", err
		}
		return chatResponse.Message.Content, nil
	}
}

// Helper function to select a chat model (non-embedding model)
func selectChatModel(models []string) string {
	// Prefer specific models known to work well for chat
	preferredModels := []string{"llama3.2", "llama3.1", "gemma2", "qwen2.5", "mistral"}

	// First try preferred models
	for _, preferred := range preferredModels {
		for _, model := range models {
			if strings.Contains(strings.ToLower(model), preferred) {
				return model
			}
		}
	}

	// Then exclude embedding models and take the first available
	for _, model := range models {
		modelLower := strings.ToLower(model)
		// Skip embedding models
		if strings.Contains(modelLower, "embed") || strings.Contains(modelLower, "bge") {
			continue
		}
		return model
	}

	return ""
}

func init() {
	rootCmd.AddCommand(ragCmd)

	ragCmd.Flags().StringVar(&ragEmbeddingsFile, "embeddings", "",
		"Path to embeddings JSON file (required)")
	ragCmd.Flags().IntVar(&ragContextSize, "context-size", 3,
		"Number of context chunks to use for answer generation")
	ragCmd.Flags().Float64Var(&ragSimilarityThreshold, "similarity-threshold", 0.0,
		"Similarity threshold for filtering context (0.0 = auto, higher = more strict)")
	ragCmd.Flags().IntVar(&ragMaxContextLength, "max-context-length", 8000,
		"Maximum total character length for context to prevent timeouts")
	ragCmd.Flags().BoolVar(&ragProgressive, "progressive", false,
		"Use progressive context loading for large context sizes")
	ragCmd.Flags().IntVar(&ragTimeout, "timeout", 0,
		"Custom timeout in seconds for answer generation (0 = use default)")

	ragCmd.MarkFlagRequired("embeddings")
}
