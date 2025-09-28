package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"time"
)

// isLowQualityChunk checks if a chunk contains mostly navigation/footer content
func isLowQualityChunk(content string) bool {
	content = strings.TrimSpace(content)

	// Check for minimum word count
	words := strings.Fields(content)
	if len(words) < 10 {
		return true
	}

	// Check for footer-only content patterns
	footerPatterns := []string{
		"Charlie Kirk Benny Johnson Jack Posobiec Alex Clark Stephen Davis View all contributors",
		"TPUSA Contributors TPUSA curates some of the country's top conservative influencers",
		"View all contributors",
		"Sort by: Most recent Most popular OP-EDS",
	}

	for _, pattern := range footerPatterns {
		if strings.Contains(content, pattern) && len(words) < 50 {
			return true
		}
	}

	// Check if content is mostly navigation/menu items (lots of "Read more", "Article", etc.)
	navWords := []string{"Read more", "Article", "Show details", "View all"}
	navCount := 0
	for _, navWord := range navWords {
		navCount += strings.Count(content, navWord)
	}

	// If more than 30% of the content appears to be navigation
	if float64(navCount*2)/float64(len(words)) > 0.3 {
		return true
	}

	return false
}

// cleanContent removes common navigation and footer elements
func cleanContent(text string) string {
	// Remove common footer patterns
	footerPatterns := []string{
		"TPUSA Contributors TPUSA curates some of the country's top conservative influencers—covering a spectrum of topics ranging from politics to pop culture Charlie Kirk Benny Johnson Jack Posobiec Alex Clark Stephen Davis View all contributors",
		"Charlie Kirk Benny Johnson Jack Posobiec Alex Clark Stephen Davis View all contributors",
	}

	for _, pattern := range footerPatterns {
		text = strings.ReplaceAll(text, pattern, "")
	}

	return strings.TrimSpace(text)
}

func chunkContent(text string, maxTokens int) []string {
	// Clean the content first
	text = cleanContent(text)

	if strings.TrimSpace(text) == "" {
		return []string{}
	}

	// Split by sentences, but also consider paragraph breaks
	sentences := regexp.MustCompile(`[.!?]+\s*`).Split(text, -1)
	chunks := []string{}
	current := ""

	for _, s := range sentences {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}

		// Estimate token count (rough approximation)
		est := int(float64(len(strings.Fields(current+" "+s))) * 1.3)

		if est > maxTokens && current != "" {
			// Before adding the chunk, check if it's high quality
			chunkCandidate := strings.TrimSpace(current)
			if !isLowQualityChunk(chunkCandidate) {
				chunks = append(chunks, chunkCandidate)
			}
			current = s
		} else {
			if current == "" {
				current = s
			} else {
				current += " " + s
			}
		}
	}

	// Add the final chunk if it's high quality
	if strings.TrimSpace(current) != "" {
		finalChunk := strings.TrimSpace(current)
		if !isLowQualityChunk(finalChunk) {
			chunks = append(chunks, finalChunk)
		}
	}

	return chunks
}

func processForEmbeddings(inputFile, outputFile string) {
	b, err := os.ReadFile(inputFile)
	if err != nil {
		log.Fatal(err)
	}
	var pages []map[string]interface{}
	if err := json.Unmarshal(b, &pages); err != nil {
		log.Fatal(err)
	}

	out := []map[string]interface{}{}
	seenContent := make(map[string]bool) // For deduplication

	for pageIndex, page := range pages {
		content, _ := page["content"].(string)
		if content == "" {
			continue
		}

		// Get URL or generate a fallback identifier
		var baseID string
		if url, ok := page["url"].(string); ok && url != "" {
			baseID = url
		} else {
			// Generate a unique identifier for pages without URLs
			baseID = fmt.Sprintf("page_%d", pageIndex)
		}

		chunks := chunkContent(content, 500)

		// Skip pages that produce no valid chunks
		if len(chunks) == 0 {
			continue
		}

		for i, c := range chunks {
			// Deduplicate similar content
			contentKey := strings.ToLower(strings.TrimSpace(c))
			if len(contentKey) < 50 { // For short content, be more strict about duplicates
				if seenContent[contentKey] {
					continue
				}
				seenContent[contentKey] = true
			} else {
				// For longer content, check first 100 characters to avoid near-duplicates
				keyPrefix := contentKey
				if len(keyPrefix) > 100 {
					keyPrefix = keyPrefix[:100]
				}
				if seenContent[keyPrefix] {
					continue
				}
				seenContent[keyPrefix] = true
			}

			id := fmt.Sprintf("%s#chunk_%d", baseID, i)
			doc := map[string]interface{}{
				"id":           id,
				"source_url":   page["url"],
				"title":        page["title"],
				"content":      c,
				"chunk_index":  i,
				"total_chunks": len(chunks),
				"metadata": map[string]interface{}{
					"crawled_at": time.Now().Format(time.RFC3339),
					"word_count": len(strings.Fields(c)),
					"char_count": len(c),
				},
			}
			out = append(out, doc)
		}
	}
	ob, _ := json.MarshalIndent(out, "", "  ")
	if err := os.WriteFile(outputFile, ob, 0o644); err != nil {
		log.Fatalf("write output: %v", err)
	}
	log.Printf("Processed %d chunks for embeddings", len(out))
}

func runPrepareEmbeddings() {
	ensureDir("tpusa_crawl/embeddings")
	processForEmbeddings("tpusa_crawl/processed_data/processed_pages.json", "tpusa_crawl/embeddings/tpusa_embeddings_ready.json")
}
