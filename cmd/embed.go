package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// embedCmd represents the embed command
var embedCmd = &cobra.Command{
	Use:   "embed [text]",
	Short: "Generate embeddings for the given text",
	Long:  `Generate vector embeddings for the provided text using the specified model.`,
	Args:  cobra.MinimumNArgs(1),
	Run:   runEmbedCommand,
}

func runEmbedCommand(cmd *cobra.Command, args []string) {
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

func init() {
	rootCmd.AddCommand(embedCmd)
}