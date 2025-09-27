package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var (
	ragEmbeddingsFile string
	ragContextSize    int
)

var ragCmd = &cobra.Command{
	Use:   "rag [question]",
	Short: "Answer questions using retrieval-augmented generation",
	Long:  `Use semantic search to find relevant context from embeddings and generate informed answers using RAG (Retrieval-Augmented Generation).`,
	Args:  cobra.MinimumNArgs(1),
	Run:   runRAGCommand,
}

func runRAGCommand(cmd *cobra.Command, args []string) {
	question := strings.Join(args, " ")

	if ragEmbeddingsFile == "" {
		fmt.Println("Please specify embeddings file with --embeddings flag")
		os.Exit(1)
	}

	// Load embeddings with content
	embeddings, err := loadEmbeddings(ragEmbeddingsFile)
	if err != nil {
		fmt.Printf("Error loading embeddings: %v\n", err)
		os.Exit(1)
	}

	if verbose {
		fmt.Printf("Loaded %d embeddings for RAG\n", len(embeddings))
	}

	// Generate embedding for question
	queryEmbedding, err := generateQueryEmbedding(question)
	if err != nil {
		fmt.Printf("Error generating query embedding: %v\n", err)
		os.Exit(1)
	}

	// Search for relevant context with lower threshold for RAG
	results := searchSimilar(queryEmbedding, embeddings, ragContextSize, 0.3)

	if len(results) == 0 {
		fmt.Printf("No relevant context found for question: %s\n", question)
		fmt.Println("Try lowering the similarity threshold or asking a different question.")
		return
	}

	// Build context from search results
	var contextParts []string
	for _, result := range results {
		content := getContentFromEmbedding(result.Item)
		if content != "" {
			contextParts = append(contextParts, content)
		}
	}

	if len(contextParts) == 0 {
		fmt.Println("Found similar embeddings but no content available for context.")
		fmt.Println("Make sure your embeddings file includes content data.")
		return
	}

	context := strings.Join(contextParts, "\n\n")

	// Generate answer using context
	answer, err := generateRAGAnswer(question, context)
	if err != nil {
		fmt.Printf("Error generating answer: %v\n", err)
		os.Exit(1)
	}

	// Display results
	fmt.Printf("Question: %s\n", question)
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("Answer: %s\n", answer)

	if verbose {
		fmt.Printf("\nContext used:\n")
		fmt.Printf("- Used %d context chunks\n", len(results))
		for i, result := range results {
			fmt.Printf("  [%d] Chunk %d (similarity: %.3f)\n",
				i+1, result.Item.ChunkIndex, result.Similarity)
		}
		fmt.Printf("- Context length: %d characters\n", len(context))
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
	// Select chat model
	models, err := ollamaClient.ListModels()
	if err != nil {
		return "", err
	}

	selectedModel := selectChatModel(models)
	if selectedModel == "" {
		return "", fmt.Errorf("no suitable chat model found")
	}

	if verbose {
		fmt.Printf("Using model for answer generation: %s\n", selectedModel)
	}

	// Build RAG prompt
	prompt := fmt.Sprintf(`Based on the following context, please answer the question. Be concise and accurate. If the answer is not clearly available in the context, say so.

Context:
%s

Question: %s

Answer:`, context, question)

	response, err := ollamaClient.Chat(selectedModel, prompt)
	if err != nil {
		return "", err
	}

	return response.Message.Content, nil
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

	ragCmd.MarkFlagRequired("embeddings")
}
