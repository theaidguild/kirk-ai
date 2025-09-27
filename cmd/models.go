package cmd

import (
	"fmt"
	"os"

	"kirk-ai/internal/config"

	"github.com/spf13/cobra"
)

// modelsCmd represents the models command
var modelsCmd = &cobra.Command{
	Use:   "models",
	Short: "List available models",
	Long:  `List all models available in your Ollama installation.`,
	Run:   runModelsCommand,
}

func runModelsCommand(cmd *cobra.Command, args []string) {
	models, err := ollamaClient.ListModels()
	if err != nil {
		fmt.Printf("Error getting models: %v\n", err)
		os.Exit(1)
	}

	if len(models) == 0 {
		fmt.Println("No models found. Please install a model first using 'ollama pull <model-name>'")
		return
	}

	fmt.Println("Available models:")
	fmt.Println("=================")

	for _, modelName := range models {
		modelInfo, hasInfo := config.GetModelInfo(modelName)
		if hasInfo {
			fmt.Printf("\n📦 %s\n", modelName)
			fmt.Printf("   Description: %s\n", modelInfo.Description)
			fmt.Printf("   Priority: %d\n", modelInfo.Priority)
			fmt.Printf("   Capabilities: ")
			for i, cap := range modelInfo.Capabilities {
				if i > 0 {
					fmt.Print(", ")
				}
				fmt.Printf("%s", cap)
			}
			fmt.Println()
		} else {
			fmt.Printf("\n📦 %s\n", modelName)
			fmt.Printf("   Description: Unknown model\n")
		}
	}

	fmt.Printf("\n\nRecommended for coding tasks: ")
	bestCoding := config.SelectBestModel(models, config.CapabilityCode)
	if bestCoding != "" {
		fmt.Printf("%s ✨\n", bestCoding)
	} else {
		fmt.Println("None detected")
	}

	fmt.Printf("Recommended for embeddings: ")
	bestEmbedding := config.SelectBestModel(models, config.CapabilityEmbedding)
	if bestEmbedding != "" {
		fmt.Printf("%s ✨\n", bestEmbedding)
	} else {
		fmt.Println("None detected")
	}
}

func init() {
	rootCmd.AddCommand(modelsCmd)
}
