package cmd

import (
	"fmt"
	"os"

	"kirk-ai/internal/client"

	"github.com/spf13/cobra"
)

var (
	// Global flags
	baseURL      string
	model        string
	verbose      bool
	stream       bool
	ollamaClient *client.OllamaClient
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "kirk-ai",
	Short: "A CLI tool for interacting with Ollama AI models",
	Long: `Kirk-AI is a command-line interface for interacting with Ollama AI models.
It supports both chat interactions and text embeddings using various models.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		ollamaClient = client.NewOllamaClient(baseURL)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVar(&baseURL, "url", "http://localhost:11434", "Ollama server URL")
	rootCmd.PersistentFlags().StringVar(&model, "model", "", "Model to use (auto-detect if not specified)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
	rootCmd.PersistentFlags().BoolVarP(&stream, "stream", "s", false, "Enable streaming output (real-time response)")
}
