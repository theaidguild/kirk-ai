package main

import (
	"context"
	"flag"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
)

func runChromedpCrawler() {
	var urlFile string
	flag.StringVar(&urlFile, "urls", "tpusa_crawl/discovered_urls.txt", "file with URLs to fetch")
	flag.Parse()

	ensureDir("tpusa_crawl/raw_html")

	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	urls := []string{"https://tpusa.com/"}
	if _, err := os.Stat(urlFile); err == nil {
		if u, err := readURLsFromFile(urlFile); err == nil && len(u) > 0 {
			urls = u
		}
	}

	for _, u := range urls {
		ctx2, cancel := context.WithTimeout(ctx, 30*time.Second)
		var html string
		err := chromedp.Run(ctx2,
			chromedp.Navigate(u),
			chromedp.WaitReady("body", chromedp.ByQuery),
			chromedp.OuterHTML("html", &html, chromedp.ByQuery),
		)
		cancel()
		if err != nil {
			log.Printf("chromedp error for %s: %v", u, err)
			continue
		}
		fname := strings.ReplaceAll(strings.ReplaceAll(u, ":", ""), "/", "_")
		path := filepath.Join("tpusa_crawl/raw_html", fname+".html")
		if err := os.WriteFile(path, []byte(html), 0o644); err != nil {
			log.Printf("write html %s: %v", path, err)
		}
		log.Printf("chromedp: saved %s", path)
	}
}
