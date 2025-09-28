package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/temoto/robotstxt"
)

var excludeHostRE = regexp.MustCompile(`(?i)rumble\.com`)
var excludePathRE = regexp.MustCompile(`(?i)/c/turningpointusa`) // skip Rumble channel path used by TPUSA

// shared http client with timeout and connection reuse
var httpClient = &http.Client{
	Timeout: 20 * time.Second,
	Transport: &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:        100,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
	},
}

// normalizeURL removes fragments and normalizes path
func normalizeURL(raw string) string {
	r := strings.TrimSpace(raw)
	if r == "" {
		return ""
	}
	u, err := url.Parse(r)
	if err != nil {
		return ""
	}
	// Ensure scheme and host exist for relative inputs
	if !u.IsAbs() {
		return ""
	}
	u.Fragment = ""
	// collapse duplicate slashes at end
	u.Path = strings.TrimRight(u.Path, "/")
	if u.Path == "" {
		u.Path = "/"
	}
	return u.String()
}

// isHTMLResponse checks content-type header
func isHTMLResponse(resp *http.Response) bool {
	ct := resp.Header.Get("Content-Type")
	return strings.Contains(ct, "text/html")
}

// simple error type to avoid fmt import
type errorString string

func (e errorString) Error() string { return string(e) }

// fetchAndParse now accepts a context and does retries + content-type check
func fetchAndParse(ctx context.Context, u string) (*goquery.Document, error) {
	var lastErr error
	backoff := 500 * time.Millisecond
	for attempt := 0; attempt < 3; attempt++ {
		req, _ := http.NewRequestWithContext(ctx, "GET", u, nil)
		req.Header.Set("User-Agent", "kirk-ai-crawler/1.0 (+https://github.com/theaidguild/kirk-ai)")
		resp, err := httpClient.Do(req)
		if err != nil {
			lastErr = err
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
			}
			time.Sleep(backoff)
			backoff *= 2
			continue
		}

		// ensure body closed and skip non-HTML/status
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			resp.Body.Close()
			return nil, &url.Error{Op: "GET", URL: u, Err: errorString("status non-2xx")}
		}
		if !isHTMLResponse(resp) {
			resp.Body.Close()
			return nil, &url.Error{Op: "GET", URL: u, Err: errorString("non-html content")}
		}

		doc, err := goquery.NewDocumentFromReader(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, err
		}
		return doc, nil
	}
	return nil, lastErr
}

// robots cache and mutex
var robotsCache = make(map[string]*robotstxt.RobotsData)
var robotsMu sync.Mutex

// isAllowedByRobots checks robots.txt for the URL's host and returns whether the given path is allowed
func isAllowedByRobots(ctx context.Context, raw string) bool {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" {
		return false
	}
	host := parsed.Scheme + "://" + parsed.Host

	robotsMu.Lock()
	data, ok := robotsCache[host]
	robotsMu.Unlock()
	if !ok {
		// fetch robots.txt
		robotsURL := host + "/robots.txt"
		req, _ := http.NewRequestWithContext(ctx, "GET", robotsURL, nil)
		req.Header.Set("User-Agent", "kirk-ai-crawler/1.0")
		resp, err := httpClient.Do(req)
		if err != nil || resp == nil {
			// If robots.txt can't be fetched, default to allowing (fail-open) but log
			if err != nil {
				log.Printf("requests crawler: could not fetch robots.txt for %s: %v", host, err)
			}
			return true
		}
		defer resp.Body.Close()
		rdata, err := robotstxt.FromResponse(resp)
		if err != nil {
			log.Printf("requests crawler: could not parse robots.txt for %s: %v", host, err)
			return true
		}
		robotsMu.Lock()
		robotsCache[host] = rdata
		robotsMu.Unlock()
		data = rdata
	}
	group := data.FindGroup("kirk-ai-crawler")
	if group == nil {
		group = data.FindGroup("*")
	}
	return group.Test(parsed.Path)
}

// isCrawlable returns false for assets, external hosts we want to avoid, and other known non-HTML patterns.
var skipCrawlRE = regexp.MustCompile(`(?i)\.(pdf|jpg|jpeg|png|gif|css|js|ico|svg|woff2?|zip)$|/wp-admin/|/wp-content/|/feed/|mailto:|/rss/|\#`)

func isCrawlable(raw string) bool {
	parsed, err := url.Parse(raw)
	if err != nil {
		return false
	}
	// exclude known hosts
	if excludeHostRE.MatchString(parsed.Host) {
		return false
	}
	// exclude specific paths
	if excludePathRE.MatchString(parsed.Path) {
		return false
	}
	// skip common static asset patterns and other unwanted paths
	if skipCrawlRE.MatchString(raw) {
		return false
	}
	return true
}

// main was renamed to runRequestsCrawler so this file can be part of a multi-tool package
func runRequestsCrawler() {
	var urlFile string
	var workers int
	var verbose bool
	flag.StringVar(&urlFile, "urls", "", "file with URLs to fetch (each URL fetched once)")
	flag.IntVar(&workers, "workers", 4, "number of parallel fetch workers for requests crawler when -urls is used")
	flag.BoolVar(&verbose, "v", false, "verbose logging")
	flag.Parse()

	// context with cancellation on SIGINT/SIGTERM
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sigch := make(chan os.Signal, 1)
	signal.Notify(sigch, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigch
		log.Println("requests crawler: interrupt received, shutting down...")
		cancel()
	}()

	// results aggregator channel (reduce mutex usage)
	results := make(chan map[string]interface{}, 256)
	var wgResults sync.WaitGroup
	var collected []map[string]interface{}
	wgResults.Add(1)
	go func() {
		defer wgResults.Done()
		for r := range results {
			collected = append(collected, r)
		}
	}()

	// helper to push a result respecting context
	pushResult := func(r map[string]interface{}) {
		select {
		case results <- r:
		case <-ctx.Done():
		}
	}

	// Buffered jobs + rate limiter (global)
	jobs := make(chan string, 1024)
	limiter := time.Tick(200 * time.Millisecond) // 5 req/sec global rate limit; adjust as needed

	// worker function using fetchAndParse
	worker := func(wg *sync.WaitGroup) {
		defer wg.Done()
		for u := range jobs {
			select {
			case <-ctx.Done():
				return
			default:
			}
			<-limiter
			u = normalizeURL(u)
			if u == "" {
				continue
			}
			if !isCrawlable(u) {
				if verbose {
					log.Println("requests crawler: skipping excluded URL:", u)
				}
				continue
			}
			if !isAllowedByRobots(ctx, u) {
				if verbose {
					log.Println("requests crawler: disallowed by robots.txt:", u)
				}
				continue
			}
			doc, err := fetchAndParse(ctx, u)
			if err != nil {
				if verbose {
					log.Println("error fetching", u, err)
				}
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
			// remove scripts/styles from selection
			main.Find("script, style, noscript").Remove()
			paras := []string{}
			main.Find("p").Each(func(i int, s *goquery.Selection) {
				if t := strings.TrimSpace(s.Text()); t != "" {
					paras = append(paras, t)
				}
			})
			content := strings.Join(paras, " ")
			if len(content) > 50_000 {
				content = content[:50_000]
			}
			page["content"] = content
			pushResult(page)
		}
	}

	// start workers when urls file provided
	if urlFile != "" {
		urls, err := readURLsFromFile(urlFile)
		if err != nil {
			log.Fatalf("could not read urls file: %v", err)
		}
		var wg sync.WaitGroup
		if workers < 1 {
			workers = 1
		}
		for i := 0; i < workers; i++ {
			wg.Add(1)
			go worker(&wg)
		}
		// deduplicate as we push, avoid enqueuing same URL twice
		seen := make(map[string]struct{})
		breakEnqueue := false
		for _, u := range urls {
			u = normalizeURL(u)
			if u == "" {
				continue
			}
			if _, ok := seen[u]; ok {
				continue
			}
			seen[u] = struct{}{}
			if !isCrawlable(u) {
				if verbose {
					log.Println("requests crawler: skipping excluded URL from input list:", u)
				}
				continue
			}
			if !isAllowedByRobots(ctx, u) {
				if verbose {
					log.Println("requests crawler: disallowed by robots.txt from input list:", u)
				}
				continue
			}
			select {
			case jobs <- u:
			case <-ctx.Done():
				breakEnqueue = true
			}
			if breakEnqueue {
				break
			}
		}
		close(jobs)
		wg.Wait()
		close(results)
		wgResults.Wait()
		b, _ := json.MarshalIndent(collected, "", "  ")
		out := "tpusa_crawl/requests_results.json"
		_ = os.MkdirAll("tpusa_crawl", 0o755)
		if err := os.WriteFile(out, b, 0o644); err != nil {
			log.Fatalf("write: %v", err)
		}
		log.Printf("requests crawler: saved %d pages to %s", len(collected), out)
		return
	}

	// Fallback: improved BFS single-process crawler with dedup-on-enqueue and normalization
	start := []string{"https://tpusa.com/", "https://tpusa.com/about/"}
	visited := map[string]struct{}{}
	enqueued := map[string]struct{}{}
	queue := make([]string, 0)
	for _, s := range start {
		n := normalizeURL(s)
		if n != "" {
			queue = append(queue, n)
			enqueued[n] = struct{}{}
		}
	}
	var data []map[string]interface{}

	for len(queue) > 0 && len(visited) < 500 {
		if ctx.Err() != nil {
			break
		}
		u := queue[0]
		queue = queue[1:]
		if _, ok := visited[u]; ok {
			continue
		}
		doc, err := fetchAndParse(ctx, u)
		if err != nil {
			if verbose {
				log.Println("error fetching", u, err)
			}
			continue
		}
		visited[u] = struct{}{}
		page := map[string]interface{}{
			"url":   u,
			"title": strings.TrimSpace(doc.Find("title").Text()),
		}
		main := doc.Find("main").First()
		if main.Length() == 0 {
			main = doc.Find("body")
		}
		main.Find("script, style, noscript").Remove()
		paras := []string{}
		main.Find("p").Each(func(i int, s *goquery.Selection) {
			if t := strings.TrimSpace(s.Text()); t != "" {
				paras = append(paras, t)
			}
		})
		content := strings.Join(paras, " ")
		if len(content) > 50_000 {
			content = content[:50_000]
		}
		page["content"] = content
		data = append(data, page)

		// Enqueue links (normalize, check robots, and dedupe on enqueue)
		doc.Find("a[href]").Each(func(i int, s *goquery.Selection) {
			href, _ := s.Attr("href")
			abs := href
			if parsed, err := url.Parse(href); err == nil && !parsed.IsAbs() {
				base, _ := url.Parse(u)
				abs = base.ResolveReference(parsed).String()
			}
			abs = normalizeURL(abs)
			if abs == "" || !isCrawlable(abs) {
				return
			}
			if !isAllowedByRobots(ctx, abs) {
				return
			}
			if _, seen := visited[abs]; !seen {
				if _, enq := enqueued[abs]; !enq {
					enqueued[abs] = struct{}{}
					queue = append(queue, abs)
				}
			}
		})
	}

	collected = append(collected, data...)
	close(results)
	wgResults.Wait()

	b, _ := json.MarshalIndent(collected, "", "  ")
	out := "tpusa_crawl/requests_results.json"
	_ = os.MkdirAll("tpusa_crawl", 0o755)
	if err := os.WriteFile(out, b, 0o644); err != nil {
		log.Fatalf("write: %v", err)
	}
	log.Printf("requests crawler: saved %d pages to %s", len(collected), out)
}
