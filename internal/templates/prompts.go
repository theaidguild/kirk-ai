package templates

import (
	"fmt"
	"strings"
)

// PromptTemplate represents a structured prompt for specific tasks
type PromptTemplate struct {
	Name        string
	Description string
	Template    string
	Variables   []string
}

// GetPromptTemplates returns templates optimized for different model capabilities
func GetPromptTemplates() map[string]PromptTemplate {
	return map[string]PromptTemplate{
		"code_generation": {
			Name:        "Code Generation",
			Description: "Generate clean, well-documented code",
			Template: `You are a skilled software engineer. Please generate clean, efficient, and well-documented code for the following request:

**Task**: {{.prompt}}

**Guidelines**:
- Write clear, readable code
- Add meaningful comments
- Follow language best practices
- Include a brief explanation

Please provide your solution:`,
			Variables: []string{"prompt"},
		},
		"code_review": {
			Name:        "Code Review",
			Description: "Review and suggest improvements for code",
			Template: `You are a senior software engineer reviewing code. Please analyze the following code and provide constructive feedback:

**Code to Review**:
{{.prompt}}

**Please provide**:
1. Overall assessment
2. Specific issues or improvements
3. Security considerations (if applicable)
4. Performance considerations
5. Suggested improvements with examples

**Review**:`,
			Variables: []string{"prompt"},
		},
		"debugging": {
			Name:        "Debugging",
			Description: "Help debug code issues",
			Template: `You are a debugging expert. Help identify and fix the issue in the following code/problem:

**Problem Description**: {{.prompt}}

**Please provide**:
1. Likely cause of the issue
2. Step-by-step debugging approach
3. Suggested fix with explanation
4. Prevention strategies

**Debugging Analysis**:`,
			Variables: []string{"prompt"},
		},
		"translation": {
			Name:        "Translation",
			Description: "Translate text between languages",
			Template: `You are a professional translator. Translate the following text accurately while preserving meaning and context:

**Source Text**: {{.prompt}}
**Target Language**: {{.target_language}}

**Translation**:`,
			Variables: []string{"prompt", "target_language"},
		},
		"reasoning": {
			Name:        "Reasoning",
			Description: "Step-by-step logical reasoning",
			Template: `You are a logical reasoning expert. Break down the following problem into clear, logical steps:

**Problem**: {{.prompt}}

**Please provide**:
1. Problem understanding
2. Step-by-step reasoning
3. Final conclusion
4. Confidence level in the answer

**Reasoning**:`,
			Variables: []string{"prompt"},
		},
		"explanation": {
			Name:        "Explanation",
			Description: "Explain complex concepts clearly",
			Template: `You are an expert teacher. Explain the following concept in a clear, structured way:

**Topic**: {{.prompt}}

**Please provide**:
1. Simple definition
2. Key concepts and components
3. Real-world examples
4. Common misconceptions (if any)

**Explanation**:`,
			Variables: []string{"prompt"},
		},
		"optimization": {
			Name:        "Optimization",
			Description: "Optimize code or processes",
			Template: `You are a performance optimization expert. Analyze and suggest optimizations for:

**Subject**: {{.prompt}}

**Please provide**:
1. Current analysis
2. Performance bottlenecks
3. Specific optimization strategies
4. Expected improvements
5. Trade-offs to consider

**Optimization Recommendations**:`,
			Variables: []string{"prompt"},
		},
	}
}

// ApplyTemplate applies a template with given variables
func ApplyTemplate(templateName string, variables map[string]string) (string, error) {
	templates := GetPromptTemplates()

	template, exists := templates[templateName]
	if !exists {
		return "", fmt.Errorf("template '%s' not found", templateName)
	}

	// Simple template variable replacement
	result := template.Template
	for key, value := range variables {
		placeholder := fmt.Sprintf("{{.%s}}", key)
		result = strings.ReplaceAll(result, placeholder, value)
	}

	// Check for unreplaced variables
	if strings.Contains(result, "{{.") {
		return "", fmt.Errorf("template has unreplaced variables")
	}

	return result, nil
}

// GetOptimalTemplate suggests the best template for a given prompt
func GetOptimalTemplate(prompt string) string {
	promptLower := strings.ToLower(prompt)

	// Code-related keywords
	codeKeywords := []string{"function", "class", "method", "algorithm", "code", "program", "script", "implement", "write"}
	for _, keyword := range codeKeywords {
		if strings.Contains(promptLower, keyword) {
			return "code_generation"
		}
	}

	// Debug-related keywords
	debugKeywords := []string{"bug", "error", "debug", "fix", "broken", "issue", "problem"}
	for _, keyword := range debugKeywords {
		if strings.Contains(promptLower, keyword) {
			return "debugging"
		}
	}

	// Translation keywords
	translationKeywords := []string{"translate", "translation", "convert to", "in chinese", "in spanish", "in french"}
	for _, keyword := range translationKeywords {
		if strings.Contains(promptLower, keyword) {
			return "translation"
		}
	}

	// Reasoning keywords
	reasoningKeywords := []string{"analyze", "reasoning", "logic", "solve", "calculate", "prove", "deduce"}
	for _, keyword := range reasoningKeywords {
		if strings.Contains(promptLower, keyword) {
			return "reasoning"
		}
	}

	// Explanation keywords
	explanationKeywords := []string{"explain", "what is", "how does", "definition", "concept"}
	for _, keyword := range explanationKeywords {
		if strings.Contains(promptLower, keyword) {
			return "explanation"
		}
	}

	// Optimization keywords
	optimizationKeywords := []string{"optimize", "performance", "faster", "efficient", "improve"}
	for _, keyword := range optimizationKeywords {
		if strings.Contains(promptLower, keyword) {
			return "optimization"
		}
	}

	// Default to no template (regular chat)
	return ""
}

// ListTemplates returns all available template names and descriptions
func ListTemplates() map[string]string {
	templates := GetPromptTemplates()
	result := make(map[string]string)

	for name, template := range templates {
		result[name] = template.Description
	}

	return result
}
