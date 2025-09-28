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
	codeLanguage string
	codeTemplate string
	codeOptimize bool
)

// codeCmd represents the code command
var codeCmd = &cobra.Command{
	Use:   "code [description]",
	Short: "Generate code using AI (optimized for gemma3:4b)",
	Long: `Generate clean, well-documented code using AI models.
This command is optimized to use gemma3:4b when available for superior coding capabilities.`,
	Args: cobra.MinimumNArgs(1),
	Run:  runCodeCommand,
}

func runCodeCommand(cmd *cobra.Command, args []string) {
	description := strings.Join(args, " ")

	// Auto-select qwen3:4b for code generation if available
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

		// Prefer gemma3:4b for coding tasks
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
			fmt.Println("No suitable model found for code generation")
			os.Exit(1)
		}
	}

	// Enhance the prompt based on flags
	enhancedPrompt := description
	if codeLanguage != "" {
		enhancedPrompt = fmt.Sprintf("Write %s code for: %s", codeLanguage, description)
	}

	// Apply template if specified
	var finalPrompt string
	if codeTemplate != "" {
		variables := map[string]string{"prompt": enhancedPrompt}
		templatedPrompt, err := templates.ApplyTemplate(codeTemplate, variables)
		if err != nil {
			fmt.Printf("Template error: %v\n", err)
			os.Exit(1)
		}
		finalPrompt = templatedPrompt
	} else {
		// Use auto-detected template or code_generation template
		detectedTemplate := templates.GetOptimalTemplate(enhancedPrompt)
		if detectedTemplate == "" || detectedTemplate == "code_generation" {
			variables := map[string]string{"prompt": enhancedPrompt}
			templatedPrompt, err := templates.ApplyTemplate("code_generation", variables)
			if err == nil {
				finalPrompt = templatedPrompt
			} else {
				finalPrompt = enhancedPrompt
			}
		} else {
			finalPrompt = enhancedPrompt
		}
	}

	if verbose {
		fmt.Printf("Using model: %s\n", selectedModel)
		fmt.Printf("Language: %s\n", codeLanguage)
		if codeTemplate != "" {
			fmt.Printf("Template: %s\n", codeTemplate)
		}
		if stream {
			fmt.Printf("Streaming: enabled\n")
		}
		fmt.Println("---")
	}

	var response *models.ChatResponse
	var err error

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
		fmt.Printf("Error generating code: %v\n", err)
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
	rootCmd.AddCommand(codeCmd)

	codeCmd.Flags().StringVarP(&codeLanguage, "language", "l", "", "Programming language (e.g., python, javascript, go)")
	codeCmd.Flags().StringVarP(&codeTemplate, "template", "t", "", "Prompt template to use (code_generation, debugging, optimization)")
	codeCmd.Flags().BoolVar(&codeOptimize, "optimize", false, "Focus on code optimization")
}
