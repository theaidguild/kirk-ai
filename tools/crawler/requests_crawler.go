package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
)

var excludeHostRE = regexp.MustCompile(`(?i)rumble\.com`)
var excludePathRE = regexp.MustCompile(`(?i)/c/turningpointusa`) // skip Rumble channel path used by TPUSA

func fetchAndParse(u string) (*goquery.Document, error) {
	resp, err := http.Get(u)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return goquery.NewDocumentFromReader(resp.Body)
}

func isCrawlable(u string) bool {
	// quick parse to examine host/path
	parsed, err := url.Parse(u)
	if err == nil {
		// exclude known external hosts
		if excludeHostRE.MatchString(parsed.Host) {
			return false
		}
		// exclude specific problematic paths (e.g. Rumble channel pages)
		if excludePathRE.MatchString(parsed.Path) {
			return false
		}
	}

	skip := regexp.MustCompile(`(?i)\.(pdf|jpg|png|gif|css|js)$|/wp-admin/|/wp-content/|/feed/|#|mailto:`)
	return !skip.MatchString(u)
}

// main was renamed to runRequestsCrawler so this file can be part of a multi-tool package
func runRequestsCrawler() {
	var urlFile string
	var workers int
	flag.StringVar(&urlFile, "urls", "", "file with URLs to fetch (each URL fetched once)")
	flag.IntVar(&workers, "workers", 4, "number of parallel fetch workers for requests crawler when -urls is used")
	flag.Parse()

	if urlFile != "" {
		// Fast parallel fetch of provided URL list
		urls, err := readURLsFromFile(urlFile)
		if err != nil {
			log.Fatalf("could not read urls file: %v", err)
		}

		var mu sync.Mutex
		var data []map[string]interface{}
		jobs := make(chan string)
		wg := sync.WaitGroup{}

		// Spawn workers
		for i := 0; i < workers; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for u := range jobs {
					if !isCrawlable(u) {
						log.Println("requests crawler: skipping excluded URL:", u)
						continue
					}
					doc, err := fetchAndParse(u)
					if err != nil {
						log.Println("error fetching", u, err)
						continue
					}
					page := map[string]interface{}{
						"url":   u,
						"title": strings.TrimSpace(doc.Find("title").Text()),
					}
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

					mu.Lock()
					data = append(data, page)
					mu.Unlock()
				}
			}()
		}

		// Push jobs
		for _, u := range urls {
			if !isCrawlable(u) {
				log.Println("requests crawler: skipping excluded URL from input list:", u)
				continue
			}
			jobs <- u
		}
		close(jobs)
		wg.Wait()

		b, _ := json.MarshalIndent(data, "", "  ")
		out := "tpusa_crawl/requests_results.json"
		if err := os.WriteFile(out, b, 0o644); err != nil {
			log.Fatalf("write: %v", err)
		}
		log.Printf("requests crawler: saved %d pages to %s", len(data), out)
		return
	}

	// Fallback: existing BFS single-process crawler
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
