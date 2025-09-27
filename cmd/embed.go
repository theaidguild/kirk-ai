package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/spf13/cobra"
)

var (
	// new flags
	embedFile    string
	embedChunk   int
	embedAll     bool
	embedOut     string
	embedBatch   int     // number of chunks a worker will try to collect/process at once
	embedConc    int     // number of concurrent workers
	embedRateRps float64 // requests per second global rate limit
)

// Named types (single source of truth) so both the command and worker functions share the same types.
type crawledChunk struct {
	ChunkIndex  int                    `json:"chunk_index"`
	Content     string                 `json:"content"`
	ID          string                 `json:"id"`
	Metadata    map[string]interface{} `json:"metadata"`
	TotalChunks int                    `json:"total_chunks"`
}

type outItem struct {
	ID         string                 `json:"id"`
	ChunkIndex int                    `json:"chunk_index"`
	Content    string                 `json:"content,omitempty"`  // Store original content
	Metadata   map[string]interface{} `json:"metadata,omitempty"` // Store metadata
	Embedding  []float64              `json:"embedding,omitempty"`
	Error      string                 `json:"error,omitempty"`
}

// embedCmd represents the embed command
var embedCmd = &cobra.Command{
	Use:   "embed [text]",
	Short: "Generate embeddings for the given text (or from a file of chunks)",
	Long:  `Generate vector embeddings for the provided text or for chunks contained in an embeddings-ready JSON file.`,
	Args:  cobra.ArbitraryArgs, // allow zero args when using --file
	Run:   runEmbedCommand,
}

func runEmbedCommand(cmd *cobra.Command, args []string) {
	// If no file and no text was provided, show usage
	if embedFile == "" && len(args) == 0 {
		fmt.Println("Please provide text to embed or --file <path> to embed chunks from a file")
		_ = cmd.Usage()
		os.Exit(1)
	}

	// FILE PATH FLOW
	if embedFile != "" {
		b, err := os.ReadFile(embedFile)
		if err != nil {
			fmt.Printf("Error reading file '%s': %v\n", embedFile, err)
			os.Exit(1)
		}

		var chunks []crawledChunk
		if err := json.Unmarshal(b, &chunks); err != nil {
			fmt.Printf("Error parsing JSON from '%s': %v\n", embedFile, err)
			os.Exit(1)
		}

		if len(chunks) == 0 {
			fmt.Println("No chunks found in file")
			os.Exit(1)
		}

		// Deduplicate chunks by ID to avoid processing duplicates
		seenIDs := make(map[string]bool)
		dedupedChunks := make([]crawledChunk, 0, len(chunks))
		duplicateCount := 0

		for _, chunk := range chunks {
			if !seenIDs[chunk.ID] {
				seenIDs[chunk.ID] = true
				dedupedChunks = append(dedupedChunks, chunk)
			} else {
				duplicateCount++
			}
		}

		chunks = dedupedChunks
		if verbose && duplicateCount > 0 {
			fmt.Printf("Removed %d duplicate chunks, %d unique chunks remaining\n", duplicateCount, len(chunks))
		}

		// Model selection (reuse existing logic)
		selectedModel := model
		if selectedModel == "" {
			models, err := ollamaClient.ListModels()
			if err != nil {
				fmt.Printf("Error getting models: %v\n", err)
				os.Exit(1)
			}
			if len(models) == 0 {
				fmt.Println("No models found. Please install a model first using 'ollama pull <model-name>'")
				os.Exit(1)
			}
			selectedModel = ollamaClient.SelectEmbeddingModel(models)
			if selectedModel == "" {
				fmt.Println("No suitable embedding model found")
				os.Exit(1)
			}
		}

		// Choose which chunks to embed
		toEmbed := make([]crawledChunk, 0)
		if embedAll {
			toEmbed = chunks
		} else if embedChunk >= 0 {
			found := false
			for _, c := range chunks {
				if c.ChunkIndex == embedChunk {
					toEmbed = append(toEmbed, c)
					found = true
					break
				}
			}
			if !found {
				fmt.Printf("Chunk index %d not found in file\n", embedChunk)
				os.Exit(1)
			}
		} else {
			// default: first chunk
			toEmbed = append(toEmbed, chunks[0])
		}

		// Prepare concurrency / rate limiting / batching
		if embedBatch <= 0 {
			embedBatch = 1
		}
		if embedConc <= 0 {
			embedConc = 4
		}
		rateEnabled := embedRateRps > 0.0
		var rateTicker *time.Ticker
		var rateCh <-chan time.Time
		if rateEnabled {
			interval := time.Duration(float64(time.Second) / embedRateRps)
			if interval <= 0 {
				interval = time.Millisecond // fallback minimal interval
			}
			rateTicker = time.NewTicker(interval)
			rateCh = rateTicker.C
			defer rateTicker.Stop()
		}

		// Output collection
		var outMu sync.Mutex
		var out []outItem

		// Jobs channel
		jobs := make(chan crawledChunk, len(toEmbed))
		for _, c := range toEmbed {
			jobs <- c
		}
		close(jobs)

		var wg sync.WaitGroup
		wg.Add(embedConc)

		// Shared progress counter (use atomic to avoid data race)
		var processed int64
		total := len(toEmbed)

		// Worker function - simplified to avoid duplicate processing
		worker := func(id int) {
			defer wg.Done()

			for {
				batch := make([]crawledChunk, 0, embedBatch)

				// Collect up to embedBatch jobs from the channel
				for len(batch) < embedBatch {
					c, ok := <-jobs
					if !ok {
						// Channel closed - process any remaining batch and exit
						if len(batch) > 0 {
							processBatch(batch, selectedModel, rateCh, rateEnabled, &outMu, &out)
							atomic.AddInt64(&processed, int64(len(batch)))
							if verbose {
								cur := atomic.LoadInt64(&processed)
								fmt.Printf("worker-%d processed batch size %d (progress %d/%d)\n", id, len(batch), cur, total)
							}
						}
						return
					}
					batch = append(batch, c)
				}

				// Process the collected batch
				processBatch(batch, selectedModel, rateCh, rateEnabled, &outMu, &out)

				// Progress reporting
				atomic.AddInt64(&processed, int64(len(batch)))
				if verbose {
					cur := atomic.LoadInt64(&processed)
					fmt.Printf("worker-%d processed batch size %d (progress %d/%d)\n", id, len(batch), cur, total)
				}
			}
		}

		// start workers
		for i := 0; i < embedConc; i++ {
			go worker(i)
		}

		// wait for all workers to finish
		wg.Wait()

		// Optionally write full embeddings to a JSON file
		if embedOut != "" {
			ob, _ := json.MarshalIndent(out, "", "  ")
			if err := os.WriteFile(embedOut, ob, 0644); err != nil {
				fmt.Printf("Error writing output to '%s': %v\n", embedOut, err)
				os.Exit(1)
			}
			fmt.Printf("Embeddings written to %s\n", embedOut)
		}
		return
	}

	// TEXT FLOW (single text embedding)
	text := strings.Join(args, " ")

	selectedModel := model
	if selectedModel == "" {
		// Auto-select an embedding model
		models, err := ollamaClient.ListModels()
		if err != nil {
			fmt.Printf("Error getting models: %v\n", err)
			os.Exit(1)
		}
		if len(models) == 0 {
			fmt.Println("No models found. Please install a model first using 'ollama pull <model-name>'")
			os.Exit(1)
		}
		selectedModel = ollamaClient.SelectEmbeddingModel(models)
		if selectedModel == "" {
			fmt.Println("No suitable embedding model found")
			os.Exit(1)
		}
	}

	if verbose {
		fmt.Printf("Using model: %s\n", selectedModel)
		fmt.Printf("Generating embeddings for: %s\n", text)
		fmt.Println("---")
	}

	response, err := ollamaClient.Embedding(selectedModel, text)
	if err != nil {
		fmt.Printf("Error generating embeddings: %v\n", err)
		os.Exit(1)
	}

	if verbose {
		fmt.Printf("Embedding vector (dimension: %d):\n", len(response.Embedding))
	}

	// Print embeddings in a readable format
	fmt.Print("[")
	for i, val := range response.Embedding {
		if i > 0 {
			fmt.Print(", ")
		}
		fmt.Printf("%.6f", val)
	}
	fmt.Println("]")
}

// processBatch processes provided chunks sequentially, respecting the provided rate channel.
// rateCh is nil when rate limiting is disabled.
func processBatch(batch []crawledChunk, selectedModel string, rateCh <-chan time.Time, rateEnabled bool, outMu *sync.Mutex, out *[]outItem) {
	for _, c := range batch {
		// wait for rate token if enabled
		if rateEnabled {
			<-rateCh
		}

		if verbose {
			fmt.Printf("Embedding chunk %d (id=%s)...\n", c.ChunkIndex, c.ID)
		}
		resp, err := ollamaClient.Embedding(selectedModel, c.Content)
		outMu.Lock()
		if err != nil {
			fmt.Printf("Error embedding chunk %d: %v\n", c.ChunkIndex, err)
			*out = append(*out, outItem{
				ID:         c.ID,
				ChunkIndex: c.ChunkIndex,
				Content:    c.Content,  // Store content even on error
				Metadata:   c.Metadata, // Store metadata even on error
				Error:      err.Error(),
			})
			outMu.Unlock()
			continue
		}
		// Print a concise representation to stdout
		fmt.Printf("Chunk %d (id=%s) embedding dimension=%d\n", c.ChunkIndex, c.ID, len(resp.Embedding))
		previewN := 8
		if len(resp.Embedding) < previewN {
			previewN = len(resp.Embedding)
		}
		fmt.Print("[")
		for i := 0; i < previewN; i++ {
			if i > 0 {
				fmt.Print(", ")
			}
			fmt.Printf("%.6f", resp.Embedding[i])
		}
		if previewN < len(resp.Embedding) {
			fmt.Print(", ...")
		}
		fmt.Println("]")

		*out = append(*out, outItem{
			ID:         c.ID,
			ChunkIndex: c.ChunkIndex,
			Content:    c.Content,  // Store content for search/RAG
			Metadata:   c.Metadata, // Store metadata for additional context
			Embedding:  resp.Embedding,
		})
		outMu.Unlock()
	}
}

func init() {
	rootCmd.AddCommand(embedCmd)

	// Register new flags
	embedCmd.Flags().StringVar(&embedFile, "file", "", "Path to embeddings-ready JSON file (e.g. tpusa_crawl/embeddings/tpusa_embeddings_ready.json)")
	embedCmd.Flags().BoolVar(&embedAll, "all", false, "Embed all chunks contained in --file")
	embedCmd.Flags().IntVar(&embedChunk, "chunk", -1, "Embed a specific chunk index from --file (0-based)")
	embedCmd.Flags().StringVar(&embedOut, "out", "", "Optional path to write embeddings JSON output")

	// Batching / rate limiting flags
	embedCmd.Flags().IntVar(&embedBatch, "batch-size", 10, "Number of chunks a worker will collect and process at once (internal batching)")
	embedCmd.Flags().IntVar(&embedConc, "concurrency", 4, "Number of concurrent workers embedding chunks")
	embedCmd.Flags().Float64Var(&embedRateRps, "rate", 5.0, "Global embedding requests per second (set to 0 to disable rate limiting)")
}
