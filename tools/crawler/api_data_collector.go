package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/mmcdole/gofeed"
)

func runAPIDataCollector() {
	ensureDir("tpusa_crawl/raw_html")
	endpoints := []string{
		"https://tpusa.com/wp-json/wp/v2/posts",
		"https://tpusa.com/wp-json/wp/v2/pages",
		"https://tpusa.com/feed/",
		"https://tpusa.com/sitemap.xml",
		"https://tpusa.com/robots.txt",
	}

	client := &http.Client{}
	available := []map[string]interface{}{}
	for _, ep := range endpoints {
		resp, err := client.Head(ep)
		if err != nil {
			fmt.Println("✗ Error accessing:", ep)
			continue
		}
		if resp.StatusCode == http.StatusOK {
			available = append(available, map[string]interface{}{
				"url":          ep,
				"content_type": resp.Header.Get("Content-Type"),
				"size":         resp.ContentLength,
			})
			fmt.Println("✓ Available:", ep)
		} else {
			fmt.Println("✗ Not available:", ep)
		}
	}

	if len(available) > 0 {
		b, _ := json.MarshalIndent(available, "", "  ")
		os.WriteFile("tpusa_crawl/api_endpoints.json", b, 0o644)
	}

	// Parse RSS feed with gofeed
	fp := gofeed.NewParser()
	feed, err := fp.ParseURL("https://tpusa.com/feed/")
	if err == nil && feed != nil {
		b, _ := json.MarshalIndent(feed.Items, "", "  ")
		os.WriteFile("tpusa_crawl/feed_items.json", b, 0o644)
		log.Printf("saved %d feed items", len(feed.Items))
	} else if err != nil {
		log.Printf("could not parse feed: %v", err)
	}
}
