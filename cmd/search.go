package cmd

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

var (
	searchEmbeddingsFile string
	searchTopK           int
	searchThreshold      float64
)

type embeddingItem struct {
	ID         string                 `json:"id"`
	ChunkIndex int                    `json:"chunk_index"`
	Content    string                 `json:"content,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	Embedding  []float64              `json:"embedding,omitempty"`
	Error      string                 `json:"error,omitempty"`
}

type searchResult struct {
	Item       embeddingItem
	Similarity float64
}

var searchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search through embeddings using semantic similarity",
	Long:  `Search for semantically similar content in your embeddings database using cosine similarity.`,
	Args:  cobra.MinimumNArgs(1),
	Run:   runSearchCommand,
}

func runSearchCommand(cmd *cobra.Command, args []string) {
	query := strings.Join(args, " ")

	if searchEmbeddingsFile == "" {
		fmt.Println("Please specify embeddings file with --embeddings flag")
		os.Exit(1)
	}

	// Load embeddings
	embeddings, err := loadEmbeddings(searchEmbeddingsFile)
	if err != nil {
		fmt.Printf("Error loading embeddings: %v\n", err)
		os.Exit(1)
	}

	if verbose {
		fmt.Printf("Loaded %d embeddings\n", len(embeddings))
	}

	// Generate embedding for query
	queryEmbedding, err := generateQueryEmbedding(query)
	if err != nil {
		fmt.Printf("Error generating query embedding: %v\n", err)
		os.Exit(1)
	}

	// Search for similar embeddings
	results := searchSimilar(queryEmbedding, embeddings, searchTopK, searchThreshold)

	// Display results
	displaySearchResults(query, results)
}

func loadEmbeddings(filename string) ([]embeddingItem, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var embeddings []embeddingItem
	if err := json.Unmarshal(data, &embeddings); err != nil {
		return nil, err
	}

	// Filter out items with errors or missing embeddings
	validEmbeddings := make([]embeddingItem, 0, len(embeddings))
	for _, item := range embeddings {
		if item.Error == "" && len(item.Embedding) > 0 {
			validEmbeddings = append(validEmbeddings, item)
		}
	}

	return validEmbeddings, nil
}

func generateQueryEmbedding(query string) ([]float64, error) {
	// Auto-select embedding model
	models, err := ollamaClient.ListModels()
	if err != nil {
		return nil, err
	}

	selectedModel := ollamaClient.SelectEmbeddingModel(models)
	if selectedModel == "" {
		return nil, fmt.Errorf("no suitable embedding model found")
	}

	if verbose {
		fmt.Printf("Using model for query: %s\n", selectedModel)
	}

	response, err := ollamaClient.Embedding(selectedModel, query)
	if err != nil {
		return nil, err
	}

	return response.Embedding, nil
}

func cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

func searchSimilar(queryEmbedding []float64, embeddings []embeddingItem, topK int, threshold float64) []searchResult {
	candidates := []searchResult{}

	for _, item := range embeddings {
		if len(item.Embedding) == 0 {
			continue
		}

		similarity := cosineSimilarity(queryEmbedding, item.Embedding)
		if similarity >= threshold {
			candidates = append(candidates, searchResult{Item: item, Similarity: similarity})
		}
	}

	// Sort by similarity (descending)
	sort.SliceStable(candidates, func(i, j int) bool {
		return candidates[i].Similarity > candidates[j].Similarity
	})

	// Deduplicate by ID or content prefix and limit to topK
	seen := map[string]bool{}
	out := make([]searchResult, 0, len(candidates))
	for _, c := range candidates {
		if topK > 0 && len(out) >= topK {
			break
		}

		key := c.Item.ID
		if key == "" {
			// Fallback to content prefix for deduplication; include chunk index if content missing
			key = c.Item.Content
			if key == "" {
				key = fmt.Sprintf("chunk_%d", c.Item.ChunkIndex)
			}
			// Limit key length to avoid excessive map keys
			if len(key) > 200 {
				key = key[:200]
			}
		}

		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, c)
	}

	return out
}

func displaySearchResults(query string, results []searchResult) {
	fmt.Printf("Search results for: \"%s\"\n", query)
	fmt.Println(strings.Repeat("=", 50))

	if len(results) == 0 {
		fmt.Printf("No results found above similarity threshold %.3f\n", searchThreshold)
		return
	}

	for i, result := range results {
		fmt.Printf("\n[%d] Chunk %d (Similarity: %.4f)\n",
			i+1, result.Item.ChunkIndex, result.Similarity)
		fmt.Printf("ID: %s\n", result.Item.ID)

		// Display content if available
		if result.Item.Content != "" {
			content := result.Item.Content
			if len(content) > 200 {
				content = content[:200] + "..."
			}
			fmt.Printf("Content: %s\n", content)
		}

		// Display metadata if available
		if len(result.Item.Metadata) > 0 {
			fmt.Printf("Metadata: %v\n", result.Item.Metadata)
		}

		fmt.Println(strings.Repeat("-", 30))
	}

	if verbose {
		fmt.Printf("\nFound %d results above threshold %.3f\n",
			len(results), searchThreshold)
	}
}

func init() {
	rootCmd.AddCommand(searchCmd)

	searchCmd.Flags().StringVar(&searchEmbeddingsFile, "embeddings", "",
		"Path to embeddings JSON file (required)")
	searchCmd.Flags().IntVar(&searchTopK, "top-k", 5,
		"Number of top results to return")
	searchCmd.Flags().Float64Var(&searchThreshold, "threshold", 0.7,
		"Minimum similarity threshold (0.0-1.0)")

	searchCmd.MarkFlagRequired("embeddings")
}
