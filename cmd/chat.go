package cmd

import (
	"fmt"
	"os"
	"strings"

	"kirk-ai/internal/models"

	"github.com/spf13/cobra"
)

// chatCmd represents the chat command
var chatCmd = &cobra.Command{
	Use:   "chat [text]",
	Short: "Send a chat message to the AI model",
	Long:  `Send a text prompt to the specified AI model and receive a response.`,
	Args:  cobra.MinimumNArgs(1),
	Run:   runChatCommand,
}

func runChatCommand(cmd *cobra.Command, args []string) {
	prompt := strings.Join(args, " ")

	selectedModel := model
	if selectedModel == "" {
		// Auto-select the first available chat model
		models, err := ollamaClient.ListModels()
		if err != nil {
			fmt.Printf("Error getting models: %v\n", err)
			os.Exit(1)
		}
		if len(models) == 0 {
			fmt.Println("No models found. Please install a model first using 'ollama pull <model-name>'")
			os.Exit(1)
		}
		selectedModel = ollamaClient.SelectChatModel(models)
		if selectedModel == "" {
			fmt.Println("No suitable chat model found")
			os.Exit(1)
		}
	}

	if verbose {
		fmt.Printf("Using model: %s\n", selectedModel)
		fmt.Printf("Sending prompt: %s\n", prompt)
		if stream {
			fmt.Printf("Streaming: enabled\n")
		}
		fmt.Println("---")
	}

	var response *models.ChatResponse
	var err error

	if stream {
		// Use streaming mode
		response, err = ollamaClient.ChatStream(selectedModel, prompt, func(chunk *models.StreamingChatResponse) error {
			// Print each chunk as it arrives
			fmt.Print(chunk.Message.Content)
			return nil
		})
		fmt.Println() // Add newline after streaming
	} else {
		// Use non-streaming mode
		response, err = ollamaClient.Chat(selectedModel, prompt)
		if err == nil {
			fmt.Printf("%s\n", response.Message.Content)
		}
	}

	if err != nil {
		fmt.Printf("Error in chat: %v\n", err)
		os.Exit(1)
	}

	if verbose {
		fmt.Printf("\n--- Response metadata ---\n")
		fmt.Printf("Model: %s\n", response.Model)
		fmt.Printf("Total duration: %d ns\n", response.TotalDuration)
		fmt.Printf("Tokens evaluated: %d\n", response.EvalCount)
		if response.EvalCount > 0 {
			tokensPerSecond := float64(response.EvalCount) / (float64(response.EvalDuration) / 1e9)
			fmt.Printf("Tokens per second: %.2f\n", tokensPerSecond)
		}
	}
}

func init() {
	rootCmd.AddCommand(chatCmd)
}
