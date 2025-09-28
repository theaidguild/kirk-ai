#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

# Configurable knobs (export to override)
CHROMEDP_CONC="${CHROMEDP_CONC:-3}"   # number of parallel chromedp workers
SKIP_CHROMEDP="${SKIP_CHROMEDP:-0}"   # set to 1 to skip chromedp step
BUILD_DIR="./build/tools"

# Crawler concurrency knobs
CRAWLER_PROCS="${CRAWLER_PROCS:-6}"         # total number of crawler processes to use when splitting
COLLY_PROCS="${COLLY_PROCS:-3}"             # how many colly processes to spawn
REQUESTS_PROCS="${REQUESTS_PROCS:-2}"       # how many requests processes to spawn
COLLY_PARALLEL="${COLLY_PARALLEL:-4}"       # per-colly-process parallelism (colly Limit Parallelism)
REQUESTS_WORKERS="${REQUESTS_WORKERS:-4}"   # per-requests-process worker count

mkdir -p tpusa_crawl/{raw_html,processed_data,embeddings,logs}
mkdir -p "$BUILD_DIR"

# Pre-download deps to reduce unexpected network time during builds
if command -v make >/dev/null 2>&1; then
  echo "Ensuring deps are downloaded (make deps)..."
  make deps >/dev/null || go mod download >/dev/null || true
else
  echo "Downloading go modules..."
  go mod download >/dev/null || true
fi

# Build tools once to avoid repeated compilation cost of 'go run'
echo "Building crawler and processor tools (one-time)..."
go build -o "$BUILD_DIR/crawler" ./tools/crawler
go build -o "$BUILD_DIR/processor" ./tools/processor

CRAWLER_BIN="$BUILD_DIR/crawler"
PROCESSOR_BIN="$BUILD_DIR/processor"

# Kill children on exit
pids=()
INTERRUPTED=0
SIGINT_COUNT=0
on_sigint() {
  SIGINT_COUNT=$((SIGINT_COUNT+1))
  if [ "$SIGINT_COUNT" -ge 2 ]; then
    echo "\nSecond SIGINT received — exiting immediately."
    exit 130
  fi
  echo "\nSIGINT/SIGTERM received — aborting crawl stage and proceeding to processing of collected data..."
  INTERRUPTED=1
  # Try to terminate running background workers gracefully
  for pid in "${pids[@]:-}"; do
    if kill -0 "$pid" 2>/dev/null; then
      echo "Terminating crawler pid $pid..."
      kill "$pid" 2>/dev/null || true
    fi
  done
}
trap 'on_sigint' INT TERM

cleanup() {
  echo "Cleaning up..."
  # If we were interrupted and already attempted to stop crawlers, don't double-kill
  if [ "${INTERRUPTED:-0}" -eq 1 ]; then
    echo "Interrupted run: not killing background workers (they were already signalled)."
    return
  fi
  for pid in "${pids[@]:-}"; do
    if kill -0 "$pid" 2>/dev/null; then
      kill "$pid" || true
    fi
  done
}
trap cleanup EXIT

echo "Discovering URLs via sitemap..."
# Use sed instead of grep -P for portability (macOS grep doesn't support -P by default)
curl -s https://tpusa.com/sitemap.xml | sed -n 's:.*<loc>\(.*\)</loc>.*:\1:p' > tpusa_crawl/discovered_urls.txt || true

# Fallback if sitemap fetch failed or returned no entries
if [ ! -s tpusa_crawl/discovered_urls.txt ]; then
  echo "Warning: sitemap fetch returned no URLs — seeding with start URL"
  echo "https://tpusa.com/" > tpusa_crawl/discovered_urls.txt
fi

# Debugging / verbosity control
VERBOSE="${VERBOSE:-0}"
if [ "$VERBOSE" = "1" ]; then
  echo "Initial discovered URLs count: $(wc -l < tpusa_crawl/discovered_urls.txt || echo 0)"
  echo "Sample URLs:"
  head -n 10 tpusa_crawl/discovered_urls.txt
fi

# --- START: URL filtering to avoid crawling unnecessary locations ---
# Tunables (export to override)
INCLUDE_HOST_REGEX="${INCLUDE_HOST_REGEX:-^https?://(www\.)?tpusa\.com}"
EXCLUDE_HOST_REGEX="${EXCLUDE_HOST_REGEX:-rumble\.com}"
EXCLUDE_EXTS="${EXCLUDE_EXTS:-pdf|jpg|jpeg|png|gif|zip|gz|tar|mp4|mp3|woff2?|svg|ico|css|js|json|rss|xml}"
EXCLUDE_PATH_REGEX="${EXCLUDE_PATH_REGEX:-/search|/tag/|/author/|/calendar|/feed|/print|/amp/|/amp$}"
HEAD_CONC="${HEAD_CONC:-10}"

echo "Filtering discovered URLs (host whitelist, host blacklist, ext blacklist, path excludes, dedupe)..."
cat tpusa_crawl/discovered_urls.txt \
  | sed -E 's/^[[:space:]]+//;s/[[:space:]]+$//' \
  | sed -E 's/([?&])(utm_[^&]+|gclid|fbclid)=[^&]*(&|$)/\1/g' \
  | sed -E 's/[?&]$//' \
  | grep -E "$INCLUDE_HOST_REGEX" \
  | grep -v -E "$EXCLUDE_HOST_REGEX" \
  | grep -v -E "\.($EXCLUDE_EXTS)(\?|$)" \
  | grep -v -E "$EXCLUDE_PATH_REGEX" \
  | sed -E 's:/$::' \
  | sort -u > tpusa_crawl/discovered_urls.filtered.txt

# Log what hosts we are explicitly excluding for debugging
if [ -n "${EXCLUDE_HOST_REGEX}" ]; then
  echo "Excluding hosts matching: $EXCLUDE_HOST_REGEX"
fi

# Respect robots.txt Disallow rules (simple fixed-path exclusion)
robots_file="$(mktemp)"
if curl -sSfL --max-time 5 https://tpusa.com/robots.txt > "$robots_file" 2>/dev/null; then
  awk '/^[Dd]isallow:/ {print $2}' "$robots_file" | sed -E 's:^/:https://tpusa.com/:' > tpusa_crawl/robots_exclude.txt || true
  if [ -s tpusa_crawl/robots_exclude.txt ]; then
    grep -v -F -f tpusa_crawl/robots_exclude.txt tpusa_crawl/discovered_urls.filtered.txt > tpusa_crawl/discovered_urls.tmp && mv tpusa_crawl/discovered_urls.tmp tpusa_crawl/discovered_urls.filtered.txt || true
  fi
fi
rm -f "$robots_file"

# Optionally verify via fast HEAD requests and keep only HTML 200 responses (parallelized)
echo "Verifying content-type via HEAD (parallel=${HEAD_CONC})..."
cat tpusa_crawl/discovered_urls.filtered.txt \
  | xargs -n1 -P "$HEAD_CONC" -I{} sh -c 'curl -s -I -L -m 10 -o /dev/null -w "%{http_code} %{content_type} %{url_effective}\n" "{}"' \
  | awk '$1==200 && $2 ~ /text\/html/ {print $3}' \
  > tpusa_crawl/discovered_urls.checked.txt || true

# Finalize list used by the pipeline (fallback to filtered if HEAD check failed)
if [ -s tpusa_crawl/discovered_urls.checked.txt ]; then
  mv tpusa_crawl/discovered_urls.checked.txt tpusa_crawl/discovered_urls.txt
else
  mv tpusa_crawl/discovered_urls.filtered.txt tpusa_crawl/discovered_urls.txt
fi

# Ensure we have at least one URL to use downstream — conservative fallback
if [ ! -s tpusa_crawl/discovered_urls.txt ]; then
  echo "Warning: no URLs remained after filtering/HEAD checks — seeding with start URL"
  echo "https://tpusa.com/" > tpusa_crawl/discovered_urls.txt
fi

if [ "$VERBOSE" = "1" ]; then
  echo "Final discovered URLs count: $(wc -l < tpusa_crawl/discovered_urls.txt || echo 0)"
  echo "Sample final URLs:"
  head -n 20 tpusa_crawl/discovered_urls.txt
fi

rm -f tpusa_crawl/discovered_urls.filtered.txt tpusa_crawl/robots_exclude.txt || true
# --- END: URL filtering ---

# Run network-bound crawlers concurrently (colly, requests, api)
# Make this stage tolerant to SIGINT so we can continue to processing on Ctrl+C.
set +e
echo "Starting crawlers in parallel (colly + requests + api)..."

NUM_URLS=$(wc -l < tpusa_crawl/discovered_urls.txt || echo 0)

# Helper to split and launch workers for a given mode and worker count
launch_workers() {
  local mode="$1"; shift
  local procs="$1"; shift
  local extra_args="$@"
  if [ "$NUM_URLS" -le 0 ]; then
    echo "No discovered URLs to feed to $mode — skipping"
    return
  fi
  chunk_size=$(( (NUM_URLS + procs - 1) / procs ))
  split_prefix="tpusa_crawl/discovered_urls.${mode}.part."
  rm -f ${split_prefix}*
  split -l "$chunk_size" tpusa_crawl/discovered_urls.txt "$split_prefix"
  for part in ${split_prefix}*; do
    [ -s "$part" ] || continue
    log="tpusa_crawl/logs/${mode}.$(basename "$part").log"
    echo "Launching $mode worker for $(wc -l < "$part") URLs -> $log"
    "$CRAWLER_BIN" "$mode" -urls "$part" $extra_args 2>&1 | tee "$log" &
    pids+=($!)
  done
}

# Launch colly workers
launch_workers colly "$COLLY_PROCS" "-parallel ${COLLY_PARALLEL}"
# Launch requests workers
launch_workers requests "$REQUESTS_PROCS" "-workers ${REQUESTS_WORKERS}"
# Run API collector once (lightweight)
"$CRAWLER_BIN" api 2>&1 | tee tpusa_crawl/logs/api_collector.log &
pids+=($!)

# Wait for crawlers to finish (but allow Ctrl+C to abort waiting and proceed)
echo "Waiting for crawlers to complete..."
for pid in "${pids[@]}"; do
  if [ "${INTERRUPTED:-0}" -eq 1 ]; then
    echo "Interrupt received — skipping wait for remaining crawlers"
    break
  fi
  # wait may be interrupted by SIGINT — don't exit script because of set -e
  wait "$pid" || true
done
pids=()  # reset for next stage
set -e

# --- START: Build dynamic chromedp list (heuristics) ---
# Heuristics: if a colly/raw_html snapshot exists but is small (< threshold) or contains typical JS app markers,
# or if there is no snapshot and the URL path is shallow, then queue for chromedp.
DYNAMIC_FILE="tpusa_crawl/dynamic_urls.txt"
rm -f "$DYNAMIC_FILE"
DYNAMIC_THRESHOLD_BYTES="${DYNAMIC_THRESHOLD_BYTES:-2000}"

echo "Building dynamic URLs list for chromedp (heuristics: small or JS-marked snapshots, missing shallow snapshots)..."
while IFS= read -r url; do
  [ -z "${url// /}" ] && continue
  # sanitize into filename similar to colly's sanitizer: non-alnum/.-_ => _
  fname=$(echo "$url" | sed -E 's@https?://@@' | sed -E 's@/$@@' | sed -E 's@[^A-Za-z0-9._-]@_@g')
  html_path="tpusa_crawl/raw_html/${fname}.html"

  if [ -f "$html_path" ]; then
    size=$(wc -c < "$html_path" 2>/dev/null || echo 0)
    if [ "$size" -lt "$DYNAMIC_THRESHOLD_BYTES" ]; then
      echo "$url" >> "$DYNAMIC_FILE"
      continue
    fi
    if grep -qiE 'data-reactroot|window\.__INITIAL_STATE__|id="app"|id="root"|class="app-root"' "$html_path" 2>/dev/null; then
      echo "$url" >> "$DYNAMIC_FILE"
      continue
    fi
  else
    # No snapshot from fast crawlers — include only shallow paths to avoid over-including.
    path_only=$(echo "$url" | sed -E 's@https?://[^/]+@@')
    # count non-empty segments
    depth=$(echo "$path_only" | awk -F'/' '{n=0; for(i=1;i<=NF;i++) if($i!="") n++; print n}')
    if [ "$depth" -le 2 ]; then
      echo "$url" >> "$DYNAMIC_FILE"
      continue
    fi
  fi
done < tpusa_crawl/discovered_urls.txt

# Dedupe and finalize
if [ -s "$DYNAMIC_FILE" ]; then
  sort -u "$DYNAMIC_FILE" -o "$DYNAMIC_FILE"
  echo "chromedp: will run on $(wc -l < \"$DYNAMIC_FILE\") pages (dynamic list)"
  CHROMEDP_URLS_FILE="$DYNAMIC_FILE"
else
  echo "chromedp: no dynamic pages detected; chromedp will be skipped unless forced by SKIP_CHROMEDP=0 and you set a fallback."
  unset CHROMEDP_URLS_FILE
fi
# --- END: Build dynamic chromedp list (heuristics) ---

# Optionally run chromedp in parallel by splitting discovered URLs into chunks
if [ "${SKIP_CHROMEDP}" != "1" ] && [ -s tpusa_crawl/discovered_urls.txt ]; then
  if [ -n "${CHROMEDP_URLS_FILE:-}" ] && [ -s "$CHROMEDP_URLS_FILE" ]; then
    echo "Running chromedp workers (concurrency=${CHROMEDP_CONC}) on dynamic URL list..."
    num_urls=$(wc -l < "$CHROMEDP_URLS_FILE" || echo 0)
    if [ "$num_urls" -gt 0 ]; then
      # Compute chunk size (ceil)
      chunk_size=$(( (num_urls + CHROMEDP_CONC - 1) / CHROMEDP_CONC ))
      split_prefix="tpusa_crawl/discovered_urls.part."
      rm -f ${split_prefix}*
      split -l "$chunk_size" "$CHROMEDP_URLS_FILE" "$split_prefix"
      for part in ${split_prefix}*; do
        [ -s "$part" ] || continue
        log="tpusa_crawl/logs/chromedp.$(basename "$part").log"
        "$CRAWLER_BIN" chromedp -urls "$part" 2>&1 | tee "$log" &
        pids+=($!)
      done

      echo "Waiting for chromedp workers..."
      set +e
      for pid in "${pids[@]}"; do
        if [ "${INTERRUPTED:-0}" -eq 1 ]; then
          echo "Interrupt received — skipping wait for remaining chromedp workers"
          break
        fi
        wait "$pid" || true
      done
      set -e
      pids=()
      rm -f ${split_prefix}*
    fi
  else
    echo "Skipping chromedp (no dynamic URLs to render)"
  fi
else
  echo "Skipping chromedp (SKIP_CHROMEDP=${SKIP_CHROMEDP})"
fi

# Process raw HTML -> processed JSON
echo "Processing raw HTML into cleaned JSON..."
"$PROCESSOR_BIN" content 2>&1 | tee tpusa_crawl/logs/processor.log

# Prepare embeddings-ready JSON
echo "Preparing embeddings data (chunking + dedupe + metadata)..."
"$PROCESSOR_BIN" embedprep 2>&1 | tee tpusa_crawl/logs/prepare_embeddings.log

echo "Pipeline complete. Outputs are under tpusa_crawl/"