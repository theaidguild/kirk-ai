package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly/v2"
)

func sanitizeFilename(u string) string {
	// keep only safe chars
	r := regexp.MustCompile(`[^a-zA-Z0-9._-]`)
	return r.ReplaceAllString(u, "_")
}

func runCollyCrawler() {
	var urlFile string
	var parallel int
	flag.StringVar(&urlFile, "urls", "tpusa_crawl/discovered_urls.txt", "file with URLs to fetch")
	flag.IntVar(&parallel, "parallel", 4, "colly parallelism per process")
	flag.Parse()

	outDir := "tpusa_crawl/raw_html"
	ensureDir(outDir)
	jsonOut := "tpusa_crawl/colly_results.json"

	c := colly.NewCollector(
		colly.AllowedDomains("tpusa.com"),
		colly.MaxDepth(3),
		colly.Async(true),
	)

	c.Limit(&colly.LimitRule{DomainGlob: "*tpusa.*", Parallelism: parallel, Delay: 500 * time.Millisecond})

	var results []map[string]interface{}
	c.OnHTML("html", func(e *colly.HTMLElement) {
		sel := e.DOM
		page := map[string]interface{}{}
		page["url"] = e.Request.URL.String()
		page["title"] = strings.TrimSpace(sel.Find("title").Text())
		page["meta_description"] = strings.TrimSpace(sel.Find("meta[name=description]").AttrOr("content", ""))
		// collect paragraphs
		paras := []string{}
		sel.Find("p").Each(func(i int, s *goquery.Selection) {
			if t := strings.TrimSpace(s.Text()); t != "" {
				paras = append(paras, t)
			}
		})
		page["content"] = strings.Join(paras, " ")

		// Save raw HTML snapshot
		u := e.Request.URL.String()
		fname := sanitizeFilename(u)
		htmlStr, err := e.DOM.Html()
		if err != nil {
			log.Printf("warning: could not obtain html for %s: %v", u, err)
		} else {
			htmlPath := filepath.Join(outDir, fname+".html")
			if err := os.WriteFile(htmlPath, []byte(htmlStr), 0o644); err != nil {
				log.Printf("warning: could not write html snapshot for %s: %v", u, err)
			}
		}

		results = append(results, page)
	})

	c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		href := e.Attr("href")
		// resolve and visit
		if u, err := e.Request.URL.Parse(href); err == nil {
			// only follow tpusa domain
			if strings.Contains(u.Hostname(), "tpusa") {
				e.Request.Visit(u.String())
			}
		}
	})

	c.OnRequest(func(r *colly.Request) { log.Println("visiting", r.URL.String()) })
	c.OnError(func(r *colly.Response, err error) { log.Printf("error %s: %v", r.Request.URL.String(), err) })

	start := "https://tpusa.com/"
	// seed sitemap discovery alongside crawler
	u, _ := url.Parse(start)
	sitemapURL := fmt.Sprintf("%s://%s/sitemap.xml", u.Scheme, u.Host)
	log.Println("seeding with sitemap", sitemapURL)
	c.Visit(sitemapURL)

	// If a urls file is provided, use it as seeds (overrides default start)
	if _, err := os.Stat(urlFile); err == nil {
		if urls, err := readURLsFromFile(urlFile); err == nil && len(urls) > 0 {
			for _, u := range urls {
				c.Visit(u)
			}
		}
	}

	if err := c.Visit(start); err != nil {
		log.Fatalf("visit start: %v", err)
	}
	c.Wait()

	// write JSON
	jb, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		log.Fatalf("json marshal: %v", err)
	}
	if err := os.WriteFile(jsonOut, jb, 0o644); err != nil {
		log.Fatalf("write results: %v", err)
	}
	log.Printf("colly: written %d pages to %s", len(results), jsonOut)
}
