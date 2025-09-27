# TPUSA Website Crawling Techniques for AI Knowledge Base

## Overview

This guide provides comprehensive techniques and tools for crawling the TPUSA website to build a robust knowledge base for AI embeddings. The approaches range from simple command-line tools to sophisticated crawling frameworks.

## 1. Command-Line Based Crawling

### Using wget (Recursive Download)

```bash
# Basic recursive crawl with depth limit
wget --recursive \
     --level=3 \
     --no-clobber \
     --page-requisites \
     --html-extension \
     --convert-links \
     --restrict-file-names=windows \
     --domains tpusa.com \
     --no-parent \
     --wait=1 \
     --random-wait \
     --user-agent="Mozilla/5.0 (compatible; Research Bot)" \
     https://tpusa.com/

# More targeted approach for specific sections
wget --recursive \
     --level=2 \
     --accept="*.html,*.htm" \
     --reject="*.pdf,*.jpg,*.png,*.gif,*.css,*.js" \
     --domains tpusa.com \
     --wait=2 \
     --random-wait \
     --user-agent="Mozilla/5.0 (compatible; Research Bot)" \
     https://tpusa.com/about/ \
     https://tpusa.com/news/ \
     https://tpusa.com/events/ \
     https://tpusa.com/team/
```

### Using curl with sitemap parsing

```bash
# Download and parse sitemap
curl -s https://tpusa.com/sitemap.xml | \
grep -oP '(?<=<loc>)[^<]+' | \
head -100 | \
while read url; do
    echo "Downloading: $url"
    curl -s -A "Mozilla/5.0 (compatible; Research Bot)" \
         -o "$(basename "$url").html" \
         "$url"
    sleep 2
done
```

## 2. Go-Based Crawling Solutions

### Using Colly (recommended for simple, fast crawls)

```go
// colly_crawler.go
package main

import (
	"encoding/json"
	"log"
	"strings"

	"github.com/gocolly/colly/v2"
	"github.com/PuerkitoBio/goquery"
)

func main() {
	c := colly.NewCollector(
		colly.AllowedDomains("tpusa.com"),
		colly.MaxDepth(3),
	)

	var results []map[string]interface{}

	c.OnHTML("html", func(e *colly.HTMLElement) {
		doc := make(map[string]interface{})
		sel := e.DOM

		// Title
		doc["url"] = e.Request.URL.String()
		doc["title"] = strings.TrimSpace(sel.Find("title").Text())

		// Meta description
		doc["meta_description"] = strings.TrimSpace(sel.Find("meta[name=description]").AttrOr("content", ""))

		// Headings
		headings := map[string][]string{}
		for i := 1; i <= 3; i++ {
			h := make([]string, 0)
			sel.Find("h" + string('0'+i)).Each(func(_ int, s *goquery.Selection) {
				text := strings.TrimSpace(s.Text())
				if text != "" {
					h = append(h, text)
				}
			})
			headings["h"+string('0'+i)] = h
		}
		doc["headings"] = headings

		// Paragraphs
		paras := make([]string, 0)
		sel.Find("p").Each(func(_ int, s *goquery.Selection) {
			text := strings.TrimSpace(s.Text())
			if text != "" {
				paras = append(paras, text)
			}
		})
		doc["paragraphs"] = paras

		// Links
		links := make([]string, 0)
		sel.Find("a[href]").Each(func(_ int, s *goquery.Selection) {
			href, _ := s.Attr("href")
			links = append(links, href)
		})
		doc["links"] = links

		results = append(results, doc)
	})

	// Follow internal links
	c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		href := e.Attr("href")
		// Let Colly resolve and follow internal links
		e.Request.Visit(href)
	})

	c.OnRequest(func(r *colly.Request) {
		log.Println("Visiting", r.URL.String())
	})

	if err := c.Visit("https://tpusa.com/"); err != nil {
		log.Fatal(err)
	}

	// Save results as JSON
	out, _ := json.MarshalIndent(results, "", "  ")
	log.Printf("Collected %d pages\n", len(results))
	_ = out // write to file as needed
}
```

### Using net/http + goquery (explicit queue & more control)

```go
// requests_crawler.go
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

func fetchAndParse(u string) (*goquery.Document, error) {
	resp, err := http.Get(u)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return goquery.NewDocumentFromReader(resp.Body)
}

func isCrawlable(u string) bool {
	skip := regexp.MustCompile(`(?i)\.(pdf|jpg|png|gif|css|js)$|/wp-admin/|/wp-content/|/feed/|#|mailto:`)
	return !skip.MatchString(u)
}

func main() {
	start := []string{"https://tpusa.com/", "https://tpusa.com/about/"}
	visited := map[string]struct{}{}
	queue := make([]string, 0)
	queue = append(queue, start...)

	var data []map[string]interface{}

	for len(queue) > 0 && len(visited) < 500 {
		u := queue[0]
		queue = queue[1:]
		if _, ok := visited[u]; ok { continue }

		doc, err := fetchAndParse(u)
		if err != nil { log.Println("error fetching", u, err); continue }

		visited[u] = struct{}{}

		page := map[string]interface{}{
			"url": u,
			"title": strings.TrimSpace(doc.Find("title").Text()),
		}
		// Extract main content paragraph text
		main := doc.Find("main").First()
		if main.Length() == 0 { main = doc.Find("body") }
		paras := []string{}
		main.Find("p").Each(func(i int, s *goquery.Selection) {
			if t := strings.TrimSpace(s.Text()); t != "" { paras = append(paras, t) }
		})
		page["content"] = strings.Join(paras, " ")
		data = append(data, page)

		// Enqueue links
		doc.Find("a[href]").Each(func(i int, s *goquery.Selection) {
			href, _ := s.Attr("href")
			abs := href
			if parsed, err := url.Parse(href); err == nil && !parsed.IsAbs() {
				base, _ := url.Parse(u)
				abs = base.ResolveReference(parsed).String()
			}
			if isCrawlable(abs) {
				if _, seen := visited[abs]; !seen {
					queue = append(queue, abs)
				}
			}
		})
	}

	b, _ := json.MarshalIndent(data, "", "  ")
	log.Printf("Saved %d pages\n", len(data))
	_ = b // write to file if desired
}
```

### JavaScript-Rendered Content (chromedp)

```go
// chromedp_crawler.go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/chromedp/chromedp"
)

func main() {
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var body string
	url := "https://tpusa.com/"
	if err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.WaitReady("body", chromedp.ByQuery),
		chromedp.OuterHTML("html", &body, chromedp.ByQuery),
	); err != nil {
		log.Fatal(err)
	}

	fmt.Println("Page length:", len(body))
	// Parse `body` with goquery if you need structured extraction
}
```

### API-Based Data Collection (HTTP checks + RSS with gofeed)

```go
// api_data_collector.go
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/mmcdole/gofeed"
)

func main() {
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
				"url": ep,
				"content_type": resp.Header.Get("Content-Type"),
				"size": resp.ContentLength,
			})
			fmt.Println("✓ Available:", ep)
		} else {
			fmt.Println("✗ Not available:", ep)
		}
	}

	// Parse RSS feed with gofeed
	fp := gofeed.NewParser()
	feed, err := fp.ParseURL("https://tpusa.com/feed/")
	if err == nil && feed != nil {
		b, _ := json.MarshalIndent(feed.Items, "", "  ")
		_ = b // save feed items as needed
	}

	_ = available
}
```

## 4. Content Processing and Cleaning (Go)

```go
// content_processor.go
package main

import (
	"regexp"
	"strings"
	"encoding/json"
	"fmt"

	"github.com/PuerkitoBio/goquery"
}

var unwantedPatterns = []string{
	`Share this:.*`, `Like this:.*`, `Related posts:.*`, `Tags:.*`,
	`Categories:.*`, `Copyright.*`, `All rights reserved.*`,
}

func cleanHTMLContent(htmlStr string) string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlStr))
	if err != nil { return "" }

	// Remove unwanted nodes
	doc.Find("script, style, nav, header, footer, aside, form, iframe, noscript").Each(func(i int, s *goquery.Selection){
		s.Remove()
	})

	unwantedClasses := []string{"sidebar", "widget", "advertisement", "social-share"}
	for _, cls := range unwantedClasses {
		doc.Find("."+cls).Each(func(i int, s *goquery.Selection){ s.Remove() })
	}

	text := strings.TrimSpace(doc.Text())
	return cleanText(text)
}

func cleanText(text string) string {
	// Remove unwanted patterns
	for _, p := range unwantedPatterns {
		r := regexp.MustCompile(`(?i)` + p)
		text = r.ReplaceAllString(text, "")
	}
	// Normalize whitespace
	rws := regexp.MustCompile(`\s+`)
	text = rws.ReplaceAllString(text, " ")
	return strings.TrimSpace(text)
}

func extractStructuredData(htmlStr string) map[string]interface{} {
	res := map[string]interface{}{}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlStr))
	if err != nil { return res }

	// Extract JSON-LD
	jsonLD := []interface{}{}
	doc.Find("script[type='application/ld+json']").Each(func(i int, s *goquery.Selection){
		if t := strings.TrimSpace(s.Text()); t != "" {
			var v interface{}
			if err := json.Unmarshal([]byte(t), &v); err == nil { jsonLD = append(jsonLD, v) }
		}
	})
	if len(jsonLD) > 0 { res["json_ld"] = jsonLD }

	// Open Graph
	og := map[string]string{}
	doc.Find("meta").Each(func(i int, s *goquery.Selection){
		if p, _ := s.Attr("property"); strings.HasPrefix(p, "og:") {
			og[strings.TrimPrefix(p, "og:")] = s.AttrOr("content", "")
		}
	})
	if len(og) > 0 { res["open_graph"] = og }
	return res
}

func main() {
	fmt.Println("content processor helpers defined")
}
```

## 5. Crawling Strategy and Implementation

### Complete Crawling Pipeline (bash wrapper updated for Go tools)

```bash
#!/bin/bash
# crawl_tpusa.sh (Go-based primitives)

# Create directory structure
mkdir -p tpusa_crawl/{raw_html,processed_data,logs}

# Step 1: Discover URLs via sitemap
curl -s https://tpusa.com/sitemap.xml | grep -oP '(?<=<loc>)[^<]+' > tpusa_crawl/discovered_urls.txt

# Step 2: Run Go crawler (example using Colly)
# go run colly_crawler.go

# Step 3: Process and clean data
# go run content_processor.go

# Step 4: Generate embeddings-ready data
# go run prepare_embeddings_data.go

echo "Crawl complete. Data ready in tpusa_crawl/ directory"
```

### Embeddings Data Preparation (Go)

```go
// prepare_embeddings_data.go
package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
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
		if s == "" { continue }
		est := int(float64(len(strings.Fields(current+" "+s))) * 1.3)
		if est > maxTokens && current != "" {
			chunks = append(chunks, strings.TrimSpace(current))
			current = s
		} else {
			if current == "" { current = s } else { current += " "+s }
		}
	}
	if strings.TrimSpace(current) != "" { chunks = append(chunks, strings.TrimSpace(current)) }
	return chunks
}

func processForEmbeddings(inputFile, outputFile string) {
	b, err := ioutil.ReadFile(inputFile)
	if err != nil { log.Fatal(err) }
	var pages []map[string]interface{}
	if err := json.Unmarshal(b, &pages); err != nil { log.Fatal(err) }

	out := []map[string]interface{}{}
	for _, page := range pages {
		content, _ := page["content"].(string)
		if content == "" { continue }
		chunks := chunkContent(content, 500)
		for i, c := range chunks {
			doc := map[string]interface{}{
				"id": page["url"].(string) + "#chunk_" + string(i),
				"source_url": page["url"],
				"title": page["title"],
				"content": c,
				"chunk_index": i,
				"total_chunks": len(chunks),
				"metadata": map[string]interface{}{
					"crawled_at": time.Now().Format(time.RFC3339),
				},
			}
			out = append(out, doc)
		}
	}
	ob, _ := json.MarshalIndent(out, "", "  ")
	ioutil.WriteFile(outputFile, ob, 0644)
	log.Printf("Processed %d chunks for embeddings", len(out))
}

func main() {
	processForEmbeddings("tpusa_crawled_data.json", "tpusa_embeddings_ready.json")
}
```

## 6. Best Practices and Considerations

### Ethical Crawling Guidelines
- Always check and respect `robots.txt`
- Implement reasonable delays between requests (1-3 seconds)
- Use appropriate User-Agent strings
- Monitor server response codes and back off on errors
- Don't overwhelm the server with concurrent requests

### Legal and Compliance
- Ensure compliance with website terms of service
- Consider copyright implications for content usage
- Implement data retention and deletion policies
- Document data sources and collection methods

### Technical Optimization
- Use session management for cookie persistence
- Implement retry logic with exponential backoff
- Cache responses to avoid redundant requests
- Monitor and log crawling activities
- Implement duplicate detection and filtering

## 7. Installation and Setup Commands (Go)

```bash
# Install required Go packages
# Modules used in examples: colly, goquery, chromedp, gofeed
go get github.com/gocolly/colly/v2
go get github.com/PuerkitoBio/goquery
go get github.com/chromedp/chromedp
go get github.com/mmcdole/gofeed

# Build/run examples
go run colly_crawler.go
go run requests_crawler.go
go run chromedp_crawler.go

# Optional: Install Chrome (for chromedp) on macOS
brew install --cask google-chrome
```

This comprehensive guide provides multiple approaches to crawling TPUSA's website, from simple command-line tools to sophisticated frameworks. Choose the approach that best fits your technical requirements and the scale of data you need to collect.