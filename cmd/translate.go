package cmd

import (
	"fmt"
	"os"
	"strings"

	"kirk-ai/internal/models"
	"kirk-ai/internal/templates"

	"github.com/spf13/cobra"
)

var (
	targetLang string
	sourceLang string
)

// translateCmd represents the translate command
var translateCmd = &cobra.Command{
	Use:   "translate [text]",
	Short: "Translate text between languages (optimized for gemma3:4b)",
	Long: `Translate text between languages using AI models.
This command is optimized to use gemma3:4b when available for superior language capabilities.

Examples:
  kirk-ai translate "Hello world" --to spanish
  kirk-ai translate "Bonjour le monde" --from french --to english
  kirk-ai translate "Hola mundo" --to german`,
	Args: cobra.MinimumNArgs(1),
	Run:  runTranslateCommand,
}

func runTranslateCommand(cmd *cobra.Command, args []string) {
	text := strings.Join(args, " ")

	var response *models.ChatResponse
	var err error

	if targetLang == "" {
		fmt.Println("Target language is required. Use --to flag")
		os.Exit(1)
	}

	// Auto-select qwen3:4b for translation if available
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

		// Prefer gemma3:4b for translation tasks
		for _, modelName := range models {
			if strings.Contains(strings.ToLower(modelName), "gemma3") {
				selectedModel = modelName
				break
			}
		}

		// Fallback to first non-embedding model
		if selectedModel == "" {
			selectedModel = ollamaClient.SelectChatModel(models)
		}

		if selectedModel == "" {
			fmt.Println("No suitable model found for translation")
			os.Exit(1)
		}
	}

	// Build the translation prompt
	variables := map[string]string{
		"prompt":          text,
		"target_language": targetLang,
	}

	if sourceLang != "" {
		// Custom template for source->target translation
		customTemplate := fmt.Sprintf(`You are a professional translator. Translate the following text from %s to %s accurately while preserving meaning and context:

**Source Text (%s)**: {{.prompt}}
**Target Language**: %s

**Translation**:`, sourceLang, targetLang, sourceLang, targetLang)

		finalPrompt := customTemplate
		for key, value := range variables {
			placeholder := fmt.Sprintf("{{.%s}}", key)
			finalPrompt = strings.ReplaceAll(finalPrompt, placeholder, value)
		}

		if verbose {
			fmt.Printf("Using model: %s\n", selectedModel)
			fmt.Printf("From: %s\n", sourceLang)
			fmt.Printf("To: %s\n", targetLang)
			if stream {
				fmt.Printf("Streaming: enabled\n")
			}
			fmt.Println("---")
		}

		if stream {
			// Use streaming mode
			response, err = ollamaClient.ChatStream(selectedModel, finalPrompt, func(chunk *models.StreamingChatResponse) error {
				// Print each chunk as it arrives
				fmt.Print(chunk.Message.Content)
				return nil
			})
			fmt.Println() // Add newline after streaming
		} else {
			// Use non-streaming mode
			response, err = ollamaClient.Chat(selectedModel, finalPrompt)
			if err == nil {
				fmt.Printf("%s\n", response.Message.Content)
			}
		}

		if err != nil {
			fmt.Printf("Error in translation: %v\n", err)
			os.Exit(1)
		}

	} else {
		// Use template for auto-detected source language
		finalPrompt, promptErr := templates.ApplyTemplate("translation", variables)
		if promptErr != nil {
			// Fallback to simple prompt
			finalPrompt = fmt.Sprintf("Translate the following text to %s: %s", targetLang, text)
		}

		if verbose {
			fmt.Printf("Using model: %s\n", selectedModel)
			fmt.Printf("To: %s\n", targetLang)
			if stream {
				fmt.Printf("Streaming: enabled\n")
			}
			fmt.Println("---")
		}

		if stream {
			// Use streaming mode
			response, err = ollamaClient.ChatStream(selectedModel, finalPrompt, func(chunk *models.StreamingChatResponse) error {
				// Print each chunk as it arrives
				fmt.Print(chunk.Message.Content)
				return nil
			})
			fmt.Println() // Add newline after streaming
		} else {
			// Use non-streaming mode
			response, err = ollamaClient.Chat(selectedModel, finalPrompt)
			if err == nil {
				fmt.Printf("%s\n", response.Message.Content)
			}
		}

		if err != nil {
			fmt.Printf("Error in translation: %v\n", err)
			os.Exit(1)
		}
	}

	if verbose {
		if response != nil {
			fmt.Printf("\n--- Response metadata ---\n")
			fmt.Printf("Model: %s\n", response.Model)
			fmt.Printf("Total duration: %d ns\n", response.TotalDuration)
			fmt.Printf("Tokens evaluated: %d\n", response.EvalCount)
			if response.EvalCount > 0 {
				tokensPerSecond := float64(response.EvalCount) / (float64(response.EvalDuration) / 1e9)
				fmt.Printf("Tokens per second: %.2f\n", tokensPerSecond)
			}
		}
		fmt.Printf("\n--- Translation completed ---\n")
	}
}

func init() {
	rootCmd.AddCommand(translateCmd)

	translateCmd.Flags().StringVar(&targetLang, "to", "", "Target language (required)")
	translateCmd.Flags().StringVar(&sourceLang, "from", "", "Source language (optional, auto-detect if not specified)")

	translateCmd.MarkFlagRequired("to")
}
