package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var (
	benchmarkAll     bool
	benchmarkModel   string
	benchmarkQuick   bool
)

// benchmarkCmd represents the benchmark command
var benchmarkCmd = &cobra.Command{
	Use:   "benchmark",
	Short: "Benchmark model performance",
	Long: `Benchmark the performance of available models with standardized tests.
This helps you understand which models work best for different tasks.`,
	Run: runBenchmarkCommand,
}

func runBenchmarkCommand(cmd *cobra.Command, args []string) {
	models, err := ollamaClient.ListModels()
	if err != nil {
		fmt.Printf("Error getting models: %v\n", err)
		os.Exit(1)
	}
	
	if len(models) == 0 {
		fmt.Println("No models found. Please install a model first using 'ollama pull <model-name>'")
		os.Exit(1)
	}

	var modelsToTest []string
	
	if benchmarkModel != "" {
		// Test specific model
		found := false
		for _, m := range models {
			if strings.Contains(strings.ToLower(m), strings.ToLower(benchmarkModel)) {
				modelsToTest = append(modelsToTest, m)
				found = true
			}
		}
		if !found {
			fmt.Printf("Model '%s' not found\n", benchmarkModel)
			os.Exit(1)
		}
	} else if benchmarkAll {
		// Test all models
		modelsToTest = models
	} else {
		// Test gemma3:4b if available, otherwise first non-embedding model
		for _, m := range models {
			if strings.Contains(strings.ToLower(m), "gemma3") {
				modelsToTest = append(modelsToTest, m)
				break
			}
		}
		if len(modelsToTest) == 0 {
			// Fallback to first non-embedding model
			for _, m := range models {
				if !strings.Contains(strings.ToLower(m), "embed") {
					modelsToTest = append(modelsToTest, m)
					break
				}
			}
		}
	}

	if len(modelsToTest) == 0 {
		fmt.Println("No suitable models found for benchmarking")
		os.Exit(1)
	}

	fmt.Printf("Benchmarking %d model(s)...\n\n", len(modelsToTest))

	// Define benchmark tests
	tests := getBenchmarkTests(benchmarkQuick)
	
	// Results storage
	results := make(map[string][]BenchmarkResult)

	for _, modelName := range modelsToTest {
		fmt.Printf("Testing model: %s\n", modelName)
		fmt.Println(strings.Repeat("-", 50))
		
		modelResults := make([]BenchmarkResult, 0, len(tests))
		
		for i, test := range tests {
			fmt.Printf("[%d/%d] %s... ", i+1, len(tests), test.Name)
			
			start := time.Now()
			response, err := ollamaClient.Chat(modelName, test.Prompt)
			duration := time.Since(start)
			
			if err != nil {
				fmt.Printf("FAILED (%v)\n", err)
				modelResults = append(modelResults, BenchmarkResult{
					TestName: test.Name,
					Success:  false,
					Duration: duration,
					Error:    err.Error(),
				})
				continue
			}
			
			tokensPerSecond := 0.0
			if response.EvalCount > 0 && response.EvalDuration > 0 {
				tokensPerSecond = float64(response.EvalCount) / (float64(response.EvalDuration) / 1e9)
			}
			
			fmt.Printf("OK (%.2fs, %.1f tokens/s)\n", duration.Seconds(), tokensPerSecond)
			
			modelResults = append(modelResults, BenchmarkResult{
				TestName:        test.Name,
				Success:         true,
				Duration:        duration,
				TokensPerSecond: tokensPerSecond,
				ResponseLength:  len(response.Message.Content),
				TotalTokens:     response.EvalCount,
			})
		}
		
		results[modelName] = modelResults
		fmt.Println()
	}

	// Print summary
	printBenchmarkSummary(results)
}

type BenchmarkTest struct {
	Name   string
	Prompt string
}

type BenchmarkResult struct {
	TestName        string
	Success         bool
	Duration        time.Duration
	TokensPerSecond float64
	ResponseLength  int
	TotalTokens     int
	Error           string
}

func getBenchmarkTests(quick bool) []BenchmarkTest {
	tests := []BenchmarkTest{
		{
			Name:   "Simple Chat",
			Prompt: "Hello! How are you today?",
		},
		{
			Name:   "Code Generation",
			Prompt: "Write a simple Python function to calculate the factorial of a number.",
		},
		{
			Name:   "Reasoning",
			Prompt: "If it takes 5 machines 5 minutes to make 5 widgets, how long would it take 100 machines to make 100 widgets?",
		},
	}

	if !quick {
		tests = append(tests, []BenchmarkTest{
			{
				Name:   "Translation",
				Prompt: "Translate 'Hello, how are you?' to Spanish, French, and German.",
			},
			{
				Name:   "Creative Writing",
				Prompt: "Write a short story about a robot learning to paint, in exactly 100 words.",
			},
			{
				Name:   "Complex Reasoning",
				Prompt: "Explain the concept of recursion in programming with a practical example.",
			},
		}...)
	}

	return tests
}

func printBenchmarkSummary(results map[string][]BenchmarkResult) {
	fmt.Println("BENCHMARK SUMMARY")
	fmt.Println(strings.Repeat("=", 60))
	
	for modelName, modelResults := range results {
		fmt.Printf("\nModel: %s\n", modelName)
		fmt.Println(strings.Repeat("-", 30))
		
		successCount := 0
		totalDuration := time.Duration(0)
		totalTokensPerSecond := 0.0
		validTokenTests := 0
		
		for _, result := range modelResults {
			if result.Success {
				successCount++
				totalDuration += result.Duration
				if result.TokensPerSecond > 0 {
					totalTokensPerSecond += result.TokensPerSecond
					validTokenTests++
				}
			}
		}
		
		fmt.Printf("Tests passed: %d/%d\n", successCount, len(modelResults))
		if successCount > 0 {
			avgDuration := totalDuration / time.Duration(successCount)
			fmt.Printf("Average response time: %.2fs\n", avgDuration.Seconds())
			
			if validTokenTests > 0 {
				avgTokensPerSecond := totalTokensPerSecond / float64(validTokenTests)
				fmt.Printf("Average tokens/sec: %.1f\n", avgTokensPerSecond)
			}
		}
		
		// Show failed tests
		for _, result := range modelResults {
			if !result.Success {
				fmt.Printf("FAILED - %s: %s\n", result.TestName, result.Error)
			}
		}
	}

	// Model comparison if multiple models tested
	if len(results) > 1 {
		fmt.Println("\nMODEL COMPARISON")
		fmt.Println(strings.Repeat("=", 60))
		
		bestSpeed := ""
		bestSpeedValue := 0.0
		bestReliability := ""
		bestReliabilityRate := 0.0
		
		for modelName, modelResults := range results {
			successCount := 0
			totalTokensPerSecond := 0.0
			validTokenTests := 0
			
			for _, result := range modelResults {
				if result.Success {
					successCount++
					if result.TokensPerSecond > 0 {
						totalTokensPerSecond += result.TokensPerSecond
						validTokenTests++
					}
				}
			}
			
			reliabilityRate := float64(successCount) / float64(len(modelResults))
			if reliabilityRate > bestReliabilityRate {
				bestReliability = modelName
				bestReliabilityRate = reliabilityRate
			}
			
			if validTokenTests > 0 {
				avgTokensPerSecond := totalTokensPerSecond / float64(validTokenTests)
				if avgTokensPerSecond > bestSpeedValue {
					bestSpeed = modelName
					bestSpeedValue = avgTokensPerSecond
				}
			}
		}
		
		if bestSpeed != "" {
			fmt.Printf("Fastest model: %s (%.1f tokens/sec)\n", bestSpeed, bestSpeedValue)
		}
		if bestReliability != "" {
			fmt.Printf("Most reliable: %s (%.1f%% success rate)\n", bestReliability, bestReliabilityRate*100)
		}
		
		// Recommend gemma3:4b if it performed well
		if gemmaResults, exists := results["gemma3:4b"]; exists {
			successCount := 0
			for _, result := range gemmaResults {
				if result.Success {
					successCount++
				}
			}
			reliabilityRate := float64(successCount) / float64(len(gemmaResults))
			
			if reliabilityRate >= 0.8 { // 80% success rate
				fmt.Printf("\n✅ gemma3:4b shows strong performance (%.1f%% success rate)\n", reliabilityRate*100)
				fmt.Println("   Recommended for coding, reasoning, and creative tasks.")
			}
		}
	}
}

func init() {
	rootCmd.AddCommand(benchmarkCmd)
	
	benchmarkCmd.Flags().BoolVarP(&benchmarkAll, "all", "a", false, "Test all available models")
	benchmarkCmd.Flags().StringVarP(&benchmarkModel, "model", "m", "", "Test specific model")
	benchmarkCmd.Flags().BoolVarP(&benchmarkQuick, "quick", "q", false, "Run quick benchmark (fewer tests)")
}