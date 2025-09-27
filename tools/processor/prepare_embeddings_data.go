package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strings"
	"time"
)

func chunkContent(text string, maxTokens int) []string {
	sentences := regexp.MustCompile(`[.!?]+\s*`).Split(text, -1)
	chunks := []string{}
	current := ""
	for _, s := range sentences {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		est := int(float64(len(strings.Fields(current+" "+s))) * 1.3)
		if est > maxTokens && current != "" {
			chunks = append(chunks, strings.TrimSpace(current))
			current = s
		} else {
			if current == "" {
				current = s
			} else {
				current += " " + s
			}
		}
	}
	if strings.TrimSpace(current) != "" {
		chunks = append(chunks, strings.TrimSpace(current))
	}
	return chunks
}

func processForEmbeddings(inputFile, outputFile string) {
	b, err := ioutil.ReadFile(inputFile)
	if err != nil {
		log.Fatal(err)
	}
	var pages []map[string]interface{}
	if err := json.Unmarshal(b, &pages); err != nil {
		log.Fatal(err)
	}

	out := []map[string]interface{}{}
	for _, page := range pages {
		content, _ := page["content"].(string)
		if content == "" {
			continue
		}
		chunks := chunkContent(content, 500)
		for i, c := range chunks {
			id := fmt.Sprintf("%v#chunk_%d", page["url"], i)
			doc := map[string]interface{}{
				"id":           id,
				"source_url":   page["url"],
				"title":        page["title"],
				"content":      c,
				"chunk_index":  i,
				"total_chunks": len(chunks),
				"metadata": map[string]interface{}{
					"crawled_at": time.Now().Format(time.RFC3339),
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
