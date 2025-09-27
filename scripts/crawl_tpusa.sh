#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

# Create directory structure
mkdir -p tpusa_crawl/{raw_html,processed_data,embeddings,logs}

echo "Discovering URLs via sitemap..."
# Use sed instead of grep -P for portability (macOS grep doesn't support -P by default)
curl -s https://tpusa.com/sitemap.xml | sed -n 's:.*<loc>\(.*\)</loc>.*:\1:p' > tpusa_crawl/discovered_urls.txt || true

# Run Colly crawler (fast, respects robots if configured in collector)
if command -v go >/dev/null 2>&1; then
  echo "Running Colly crawler..."
  go run ./tools/crawler colly 2>&1 | tee tpusa_crawl/logs/colly.log || true
else
  echo "Go not found, skipping Colly crawler"
fi

# Run requests-style crawler
if command -v go >/dev/null 2>&1; then
  echo "Running requests-style crawler..."
  go run ./tools/crawler requests 2>&1 | tee tpusa_crawl/logs/requests.log || true
fi

# Optionally run chromedp for JS-rendered pages (may require Chrome installed)
if command -v go >/dev/null 2>&1 && (command -v google-chrome >/dev/null 2>&1 || [ -x "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome" ]); then
  echo "Running chromedp (JS-rendered fetch)..."
  go run ./tools/crawler chromedp -urls tpusa_crawl/discovered_urls.txt 2>&1 | tee tpusa_crawl/logs/chromedp.log || true
else
  echo "Skipping chromedp (chrome not found or go missing)"
fi

# Run API/RSS collector
if command -v go >/dev/null 2>&1; then
  echo "Running API/RSS collector..."
  go run ./tools/crawler api 2>&1 | tee tpusa_crawl/logs/api_collector.log || true
fi

# Process raw HTML into cleaned structured JSON
if command -v go >/dev/null 2>&1; then
  echo "Processing raw HTML..."
  go run ./tools/processor content 2>&1 | tee tpusa_crawl/logs/processor.log || true
fi

# Prepare embeddings data
if command -v go >/dev/null 2>&1; then
  echo "Preparing embeddings data..."
  go run ./tools/processor embedprep 2>&1 | tee tpusa_crawl/logs/prepare_embeddings.log || true
fi

echo "Pipeline complete. Outputs are under tpusa_crawl/"
