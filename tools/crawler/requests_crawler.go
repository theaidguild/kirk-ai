package main

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"os"
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

// main was renamed to runRequestsCrawler so this file can be part of a multi-tool package
func runRequestsCrawler() {
	start := []string{"https://tpusa.com/", "https://tpusa.com/about/"}
	visited := map[string]struct{}{}
	queue := make([]string, 0)
	queue = append(queue, start...)

	var data []map[string]interface{}

	for len(queue) > 0 && len(visited) < 500 {
		u := queue[0]
		queue = queue[1:]
		if _, ok := visited[u]; ok {
			continue
		}

		doc, err := fetchAndParse(u)
		if err != nil {
			log.Println("error fetching", u, err)
			continue
		}

		visited[u] = struct{}{}

		page := map[string]interface{}{
			"url":   u,
			"title": strings.TrimSpace(doc.Find("title").Text()),
		}
		// Extract main content paragraph text
		main := doc.Find("main").First()
		if main.Length() == 0 {
			main = doc.Find("body")
		}
		paras := []string{}
		main.Find("p").Each(func(i int, s *goquery.Selection) {
			if t := strings.TrimSpace(s.Text()); t != "" {
				paras = append(paras, t)
			}
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
	out := "tpusa_crawl/requests_results.json"
	if err := os.WriteFile(out, b, 0o644); err != nil {
		log.Fatalf("write: %v", err)
	}
	log.Printf("requests crawler: saved %d pages to %s", len(data), out)
}
