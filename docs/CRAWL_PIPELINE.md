TPUSA Crawl Pipeline

Overview

This repository contains a small Go-based crawling pipeline (helper tools) that discovers pages from tpusa.com, fetches HTML (static and JS-rendered), collects RSS/API endpoints, cleans content, and prepares embeddings-ready chunks.

Prerequisites

- Go 1.19+ installed (use `go env` to confirm).
- Chrome or Chromium installed for `chromedp` (optional).
- Add required Go modules used by helper tools:

  go get github.com/gocolly/colly/v2
  go get github.com/PuerkitoBio/goquery
  go get github.com/chromedp/chromedp
  go get github.com/mmcdole/gofeed

Usage

1. Make sure you are in the repository root.
2. Make the script executable:

   chmod +x scripts/crawl_tpusa.sh

3. Run the pipeline (this will create `tpusa_crawl/`):

   ./scripts/crawl_tpusa.sh

Notes

- The helper Go programs live under `tools/` and are marked with a `//go:build tools` build tag so they don't affect normal builds.
- The pipeline writes raw HTML to `tpusa_crawl/raw_html`, processed JSON to `tpusa_crawl/processed_data`, and embeddings-ready JSON to `tpusa_crawl/embeddings`.
- For large crawls, consider running colly or chromedp with appropriate rate limits and respecting `robots.txt`.
