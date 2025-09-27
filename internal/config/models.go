package config

import (
	"strings"
)

// ModelCapability represents what a model is good at
type ModelCapability string

const (
	CapabilityChat        ModelCapability = "chat"
	CapabilityCode        ModelCapability = "code"
	CapabilityEmbedding   ModelCapability = "embedding"
	CapabilityReasoning   ModelCapability = "reasoning"
	CapabilityTranslation ModelCapability = "translation"
	CapabilityCreative    ModelCapability = "creative"
)

// ModelConfig defines model capabilities and preferences
type ModelConfig struct {
	Name         string
	Capabilities []ModelCapability
	Priority     int // Higher number = higher priority
	Description  string
}

// GetModelConfigs returns predefined model configurations
func GetModelConfigs() map[string]ModelConfig {
	return map[string]ModelConfig{
		"gemma3:4b": {
			Name:         "gemma3:4b",
			Capabilities: []ModelCapability{CapabilityChat, CapabilityCode, CapabilityReasoning, CapabilityCreative},
			Priority:     95,
			Description:  "Gemma 3 4B - Excellent for coding, reasoning, and creative tasks",
		},
		"llama3.1:8b": {
			Name:         "llama3.1:8b",
			Capabilities: []ModelCapability{CapabilityChat, CapabilityCreative, CapabilityReasoning},
			Priority:     80,
			Description:  "Llama 3.1 8B - Strong general-purpose model",
		},
		"llama3.2:3b": {
			Name:         "llama3.2:3b",
			Capabilities: []ModelCapability{CapabilityChat, CapabilityCreative},
			Priority:     70,
			Description:  "Llama 3.2 3B - Lightweight general-purpose model",
		},
		"embeddinggemma:latest": {
			Name:         "embeddinggemma:latest",
			Capabilities: []ModelCapability{CapabilityEmbedding},
			Priority:     90,
			Description:  "Gemma embedding model - Optimized for text embeddings",
		},
	}
}

// SelectBestModel selects the best available model for a given capability
func SelectBestModel(availableModels []string, capability ModelCapability) string {
	configs := GetModelConfigs()
	bestModel := ""
	bestPriority := -1

	for _, modelName := range availableModels {
		// Try exact match first
		if config, exists := configs[modelName]; exists {
			if hasCapability(config.Capabilities, capability) && config.Priority > bestPriority {
				bestModel = modelName
				bestPriority = config.Priority
			}
			continue
		}

		// Try partial match for model variants
		for configName, config := range configs {
			if strings.Contains(strings.ToLower(modelName), strings.ToLower(configName)) ||
				strings.Contains(strings.ToLower(configName), strings.ToLower(modelName)) {
				if hasCapability(config.Capabilities, capability) && config.Priority > bestPriority {
					bestModel = modelName
					bestPriority = config.Priority
				}
			}
		}
	}

	// Fallback: if no configured model found, use legacy logic
	if bestModel == "" && len(availableModels) > 0 {
		if capability == CapabilityEmbedding {
			for _, model := range availableModels {
				if strings.Contains(strings.ToLower(model), "embed") {
					return model
				}
			}
		} else {
			// For non-embedding tasks, prefer gemma3:4b if available
			for _, model := range availableModels {
				if strings.Contains(strings.ToLower(model), "gemma3") {
					return model
				}
			}
			// Otherwise, avoid embedding models
			for _, model := range availableModels {
				if !strings.Contains(strings.ToLower(model), "embed") {
					return model
				}
			}
		}
		return availableModels[0]
	}

	return bestModel
}

// hasCapability checks if a model has a specific capability
func hasCapability(capabilities []ModelCapability, target ModelCapability) bool {
	for _, cap := range capabilities {
		if cap == target {
			return true
		}
	}
	return false
}

// GetModelInfo returns information about a specific model
func GetModelInfo(modelName string) (ModelConfig, bool) {
	configs := GetModelConfigs()

	// Try exact match first
	if config, exists := configs[modelName]; exists {
		return config, true
	}

	// Try partial match
	for configName, config := range configs {
		if strings.Contains(strings.ToLower(modelName), strings.ToLower(configName)) {
			return config, true
		}
	}

	return ModelConfig{}, false
}
