package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

var unwantedPatterns = []string{
	`Share this:.*`, `Like this:.*`, `Related posts:.*`, `Tags:.*`,
	`Categories:.*`, `Copyright.*`, `All rights reserved.*`,
}

func cleanHTMLContent(htmlStr string) string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlStr))
	if err != nil {
		return ""
	}

	// Remove unwanted nodes
	doc.Find("script, style, nav, header, footer, aside, form, iframe, noscript").Each(func(i int, s *goquery.Selection) {
		s.Remove()
	})

	unwantedClasses := []string{"sidebar", "widget", "advertisement", "social-share"}
	for _, cls := range unwantedClasses {
		doc.Find("." + cls).Each(func(i int, s *goquery.Selection) { s.Remove() })
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
	if err != nil {
		return res
	}

	// Extract JSON-LD
	jsonLD := []interface{}{}
	doc.Find("script[type='application/ld+json']").Each(func(i int, s *goquery.Selection) {
		if t := strings.TrimSpace(s.Text()); t != "" {
			var v interface{}
			if err := json.Unmarshal([]byte(t), &v); err == nil {
				jsonLD = append(jsonLD, v)
			}
		}
	})
	if len(jsonLD) > 0 {
		res["json_ld"] = jsonLD
	}

	// Open Graph
	og := map[string]string{}
	doc.Find("meta").Each(func(i int, s *goquery.Selection) {
		if p, _ := s.Attr("property"); strings.HasPrefix(p, "og:") {
			og[strings.TrimPrefix(p, "og:")] = s.AttrOr("content", "")
		}
	})
	if len(og) > 0 {
		res["open_graph"] = og
	}
	return res
}

func processRawHTMLDir(rawDir, outFile string) {
	files, err := ioutil.ReadDir(rawDir)
	if err != nil {
		log.Fatalf("read dir: %v", err)
	}
	out := []map[string]interface{}{}
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		if !strings.HasSuffix(f.Name(), ".html") {
			continue
		}
		b, err := ioutil.ReadFile(filepath.Join(rawDir, f.Name()))
		if err != nil {
			log.Printf("read %s: %v", f.Name(), err)
			continue
		}
		h := string(b)
		clean := cleanHTMLContent(h)
		meta := extractStructuredData(h)
		out = append(out, map[string]interface{}{"file": f.Name(), "content": clean, "meta": meta})
	}
	jb, _ := json.MarshalIndent(out, "", "  ")
	ioutil.WriteFile(outFile, jb, 0o644)
	fmt.Printf("processed %d files -> %s\n", len(out), outFile)
}

func runContentProcessor() {
	processRawHTMLDir("tpusa_crawl/raw_html", "tpusa_crawl/processed_data/processed_pages.json")
}
