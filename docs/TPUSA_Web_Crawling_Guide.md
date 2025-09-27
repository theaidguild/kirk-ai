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

## 2. Python-Based Crawling Solutions

### Using Scrapy Framework

```python
# scrapy_tpusa_crawler.py
import scrapy
from urllib.parse import urljoin, urlparse
import json
import time

class TPUSACrawler(scrapy.Spider):
    name = 'tpusa_crawler'
    allowed_domains = ['tpusa.com']
    start_urls = [
        'https://tpusa.com/',
        'https://tpusa.com/about/',
        'https://tpusa.com/team/',
        'https://tpusa.com/news/',
        'https://tpusa.com/events/',
        'https://tpusa.com/programs/'
    ]
    
    custom_settings = {
        'DOWNLOAD_DELAY': 2,
        'RANDOMIZE_DOWNLOAD_DELAY': True,
        'CONCURRENT_REQUESTS': 1,
        'ROBOTSTXT_OBEY': True,
        'USER_AGENT': 'Mozilla/5.0 (compatible; Research Bot)'
    }
    
    def parse(self, response):
        # Extract main content
        content_data = {
            'url': response.url,
            'title': response.css('title::text').get(),
            'meta_description': response.css('meta[name="description"]::attr(content)').get(),
            'headings': {
                'h1': response.css('h1::text').getall(),
                'h2': response.css('h2::text').getall(),
                'h3': response.css('h3::text').getall()
            },
            'paragraphs': response.css('p::text').getall(),
            'links': response.css('a::attr(href)').getall(),
            'timestamp': time.strftime('%Y-%m-%d %H:%M:%S')
        }
        
        # Clean and structure content
        content_data['clean_text'] = self.extract_clean_text(response)
        
        yield content_data
        
        # Follow links to other pages on the same domain
        for link in response.css('a::attr(href)').getall():
            if link:
                absolute_url = urljoin(response.url, link)
                if self.is_valid_url(absolute_url):
                    yield response.follow(link, self.parse)
    
    def extract_clean_text(self, response):
        # Remove script and style elements
        text_content = response.css('body *:not(script):not(style)::text').getall()
        clean_text = ' '.join([text.strip() for text in text_content if text.strip()])
        return clean_text
    
    def is_valid_url(self, url):
        parsed = urlparse(url)
        # Skip certain file types and external domains
        skip_extensions = ['.pdf', '.jpg', '.png', '.gif', '.css', '.js', '.zip']
        return (parsed.netloc in self.allowed_domains and 
                not any(url.endswith(ext) for ext in skip_extensions))

# Run with: scrapy crawl tpusa_crawler -o tpusa_data.json
```

### Using Requests + BeautifulSoup

```python
# requests_crawler.py
import requests
from bs4 import BeautifulSoup
import json
import time
from urllib.parse import urljoin, urlparse
from collections import deque
import re

class TPUSARequestsCrawler:
    def __init__(self):
        self.session = requests.Session()
        self.session.headers.update({
            'User-Agent': 'Mozilla/5.0 (compatible; Research Bot)'
        })
        self.visited_urls = set()
        self.data = []
        
    def crawl(self, start_urls, max_pages=200, delay=2):
        url_queue = deque(start_urls)
        
        while url_queue and len(self.visited_urls) < max_pages:
            url = url_queue.popleft()
            
            if url in self.visited_urls:
                continue
                
            try:
                print(f"Crawling: {url}")
                response = self.session.get(url, timeout=10)
                response.raise_for_status()
                
                self.visited_urls.add(url)
                
                # Parse content
                soup = BeautifulSoup(response.content, 'html.parser')
                page_data = self.extract_page_data(soup, url)
                self.data.append(page_data)
                
                # Find new URLs to crawl
                new_urls = self.find_links(soup, url)
                url_queue.extend(new_urls)
                
                time.sleep(delay)
                
            except Exception as e:
                print(f"Error crawling {url}: {e}")
                continue
    
    def extract_page_data(self, soup, url):
        # Remove unwanted elements
        for element in soup(['script', 'style', 'nav', 'header', 'footer']):
            element.decompose()
        
        return {
            'url': url,
            'title': soup.find('title').get_text() if soup.find('title') else '',
            'meta_description': self.get_meta_description(soup),
            'headings': self.extract_headings(soup),
            'content': self.extract_content(soup),
            'breadcrumbs': self.extract_breadcrumbs(soup),
            'publication_date': self.extract_date(soup),
            'author': self.extract_author(soup)
        }
    
    def get_meta_description(self, soup):
        meta = soup.find('meta', attrs={'name': 'description'})
        return meta.get('content', '') if meta else ''
    
    def extract_headings(self, soup):
        headings = {}
        for level in range(1, 7):
            headings[f'h{level}'] = [h.get_text().strip() 
                                   for h in soup.find_all(f'h{level}')]
        return headings
    
    def extract_content(self, soup):
        # Focus on main content areas
        main_content = soup.find('main') or soup.find('article') or soup.find('body')
        
        if main_content:
            paragraphs = [p.get_text().strip() 
                         for p in main_content.find_all('p') 
                         if p.get_text().strip()]
            return ' '.join(paragraphs)
        return ''
    
    def extract_breadcrumbs(self, soup):
        breadcrumb_selectors = [
            '.breadcrumb',
            '.breadcrumbs', 
            '[itemtype*="BreadcrumbList"]'
        ]
        
        for selector in breadcrumb_selectors:
            breadcrumb = soup.select_one(selector)
            if breadcrumb:
                return [a.get_text().strip() for a in breadcrumb.find_all('a')]
        return []
    
    def extract_date(self, soup):
        # Look for various date formats
        date_selectors = [
            'time[datetime]',
            '.published-date',
            '.post-date',
            '[itemtype*="Article"] time'
        ]
        
        for selector in date_selectors:
            date_elem = soup.select_one(selector)
            if date_elem:
                return date_elem.get('datetime') or date_elem.get_text().strip()
        return None
    
    def extract_author(self, soup):
        author_selectors = [
            '.author',
            '.byline',
            '[rel="author"]',
            '[itemtype*="Person"]'
        ]
        
        for selector in author_selectors:
            author_elem = soup.select_one(selector)
            if author_elem:
                return author_elem.get_text().strip()
        return None
    
    def find_links(self, soup, current_url):
        links = []
        base_domain = urlparse(current_url).netloc
        
        for link in soup.find_all('a', href=True):
            href = link['href']
            absolute_url = urljoin(current_url, href)
            parsed_url = urlparse(absolute_url)
            
            # Only crawl same domain links
            if (parsed_url.netloc == base_domain and 
                absolute_url not in self.visited_urls and
                self.is_crawlable_url(absolute_url)):
                links.append(absolute_url)
        
        return list(set(links))  # Remove duplicates
    
    def is_crawlable_url(self, url):
        skip_patterns = [
            r'\.pdf$', r'\.jpg$', r'\.png$', r'\.gif$', r'\.css$', r'\.js$',
            r'/wp-admin/', r'/wp-content/', r'/feed/', r'#', r'mailto:',
            r'/search/', r'/tag/', r'/category/'
        ]
        
        return not any(re.search(pattern, url, re.I) for pattern in skip_patterns)
    
    def save_data(self, filename):
        with open(filename, 'w', encoding='utf-8') as f:
            json.dump(self.data, f, indent=2, ensure_ascii=False)
        print(f"Data saved to {filename}")

# Usage example
if __name__ == "__main__":
    crawler = TPUSARequestsCrawler()
    start_urls = [
        'https://tpusa.com/',
        'https://tpusa.com/about/',
        'https://tpusa.com/team/',
        'https://tpusa.com/news/',
        'https://tpusa.com/events/'
    ]
    
    crawler.crawl(start_urls, max_pages=500, delay=2)
    crawler.save_data('tpusa_crawled_data.json')
```

## 3. Advanced Crawling Techniques

### JavaScript-Rendered Content (Selenium)

```python
# selenium_crawler.py
from selenium import webdriver
from selenium.webdriver.chrome.options import Options
from selenium.webdriver.common.by import By
from selenium.webdriver.support.ui import WebDriverWait
from selenium.webdriver.support import expected_conditions as EC
import json
import time

class TPUSASeleniumCrawler:
    def __init__(self):
        self.setup_driver()
        self.data = []
    
    def setup_driver(self):
        chrome_options = Options()
        chrome_options.add_argument('--headless')
        chrome_options.add_argument('--no-sandbox')
        chrome_options.add_argument('--disable-dev-shm-usage')
        chrome_options.add_argument('--user-agent=Mozilla/5.0 (compatible; Research Bot)')
        
        self.driver = webdriver.Chrome(options=chrome_options)
        self.driver.set_page_load_timeout(30)
    
    def crawl_dynamic_content(self, urls):
        for url in urls:
            try:
                print(f"Loading: {url}")
                self.driver.get(url)
                
                # Wait for dynamic content to load
                WebDriverWait(self.driver, 10).until(
                    EC.presence_of_element_located((By.TAG_NAME, "body"))
                )
                
                # Scroll to load lazy content
                self.driver.execute_script("window.scrollTo(0, document.body.scrollHeight);")
                time.sleep(3)
                
                # Extract data
                page_data = {
                    'url': url,
                    'title': self.driver.title,
                    'content': self.driver.find_element(By.TAG_NAME, "body").text,
                    'links': [elem.get_attribute('href') 
                             for elem in self.driver.find_elements(By.TAG_NAME, "a") 
                             if elem.get_attribute('href')]
                }
                
                self.data.append(page_data)
                time.sleep(2)
                
            except Exception as e:
                print(f"Error with {url}: {e}")
                continue
    
    def close(self):
        self.driver.quit()
```

### API-Based Data Collection

```python
# api_data_collector.py
import requests
import json

class TPUSAAPICollector:
    def __init__(self):
        self.session = requests.Session()
        self.session.headers.update({
            'User-Agent': 'Mozilla/5.0 (compatible; Research Bot)'
        })
    
    def check_for_apis(self):
        # Common WordPress/CMS API endpoints
        api_endpoints = [
            'https://tpusa.com/wp-json/wp/v2/posts',
            'https://tpusa.com/wp-json/wp/v2/pages',
            'https://tpusa.com/api/posts',
            'https://tpusa.com/feed/',
            'https://tpusa.com/sitemap.xml',
            'https://tpusa.com/robots.txt'
        ]
        
        available_endpoints = []
        for endpoint in api_endpoints:
            try:
                response = self.session.get(endpoint, timeout=10)
                if response.status_code == 200:
                    available_endpoints.append({
                        'url': endpoint,
                        'content_type': response.headers.get('content-type', ''),
                        'size': len(response.content)
                    })
                    print(f"✓ Available: {endpoint}")
                else:
                    print(f"✗ Not available: {endpoint}")
            except:
                print(f"✗ Error accessing: {endpoint}")
        
        return available_endpoints
    
    def collect_rss_data(self, rss_url):
        import feedparser
        
        feed = feedparser.parse(rss_url)
        posts = []
        
        for entry in feed.entries:
            post_data = {
                'title': getattr(entry, 'title', ''),
                'link': getattr(entry, 'link', ''),
                'summary': getattr(entry, 'summary', ''),
                'published': getattr(entry, 'published', ''),
                'author': getattr(entry, 'author', ''),
                'tags': [tag.term for tag in getattr(entry, 'tags', [])]
            }
            posts.append(post_data)
        
        return posts
```

## 4. Content Processing and Cleaning

### Text Extraction and Cleaning

```python
# content_processor.py
import re
from bs4 import BeautifulSoup
import html

class ContentProcessor:
    def __init__(self):
        self.unwanted_patterns = [
            r'Share this:.*',
            r'Like this:.*',
            r'Related posts:.*',
            r'Tags:.*',
            r'Categories:.*',
            r'Copyright.*',
            r'All rights reserved.*'
        ]
    
    def clean_html_content(self, html_content):
        soup = BeautifulSoup(html_content, 'html.parser')
        
        # Remove unwanted elements
        for element in soup(['script', 'style', 'nav', 'header', 'footer', 
                           'aside', 'form', 'iframe', 'noscript']):
            element.decompose()
        
        # Remove elements with specific classes (common WordPress widgets)
        unwanted_classes = ['sidebar', 'widget', 'advertisement', 'social-share']
        for class_name in unwanted_classes:
            for element in soup.find_all(class_=class_name):
                element.decompose()
        
        # Extract clean text
        text = soup.get_text()
        return self.clean_text(text)
    
    def clean_text(self, text):
        # Decode HTML entities
        text = html.unescape(text)
        
        # Remove unwanted patterns
        for pattern in self.unwanted_patterns:
            text = re.sub(pattern, '', text, flags=re.IGNORECASE)
        
        # Normalize whitespace
        text = re.sub(r'\s+', ' ', text)
        text = text.strip()
        
        return text
    
    def extract_structured_data(self, soup):
        structured_data = {}
        
        # Extract JSON-LD structured data
        json_ld_scripts = soup.find_all('script', type='application/ld+json')
        for script in json_ld_scripts:
            try:
                data = json.loads(script.string)
                structured_data['json_ld'] = data
            except:
                continue
        
        # Extract Open Graph data
        og_data = {}
        og_tags = soup.find_all('meta', property=lambda x: x and x.startswith('og:'))
        for tag in og_tags:
            property_name = tag.get('property', '').replace('og:', '')
            content = tag.get('content', '')
            og_data[property_name] = content
        
        if og_data:
            structured_data['open_graph'] = og_data
        
        return structured_data
```

## 5. Crawling Strategy and Implementation

### Complete Crawling Pipeline

```bash
#!/bin/bash
# crawl_tpusa.sh

# Create directory structure
mkdir -p tpusa_crawl/{raw_html,processed_data,logs}

# Step 1: Discover URLs via sitemap
echo "Discovering URLs..."
curl -s https://tpusa.com/sitemap.xml | \
grep -oP '(?<=<loc>)[^<]+' > tpusa_crawl/discovered_urls.txt

# Step 2: Run Python crawler
echo "Starting comprehensive crawl..."
python3 requests_crawler.py

# Step 3: Process and clean data
echo "Processing crawled data..."
python3 content_processor.py

# Step 4: Generate embeddings-ready data
echo "Preparing data for embeddings..."
python3 prepare_embeddings_data.py

echo "Crawl complete. Data ready in tpusa_crawl/ directory"
```

### Embeddings Data Preparation

```python
# prepare_embeddings_data.py
import json
import re
from datetime import datetime

class EmbeddingsDataPreparation:
    def __init__(self, crawled_data_file):
        with open(crawled_data_file, 'r', encoding='utf-8') as f:
            self.raw_data = json.load(f)
        
        self.processed_chunks = []
    
    def chunk_content(self, text, max_tokens=500):
        # Simple sentence-based chunking
        sentences = re.split(r'[.!?]+', text)
        chunks = []
        current_chunk = ""
        
        for sentence in sentences:
            sentence = sentence.strip()
            if not sentence:
                continue
                
            # Rough token estimation (words * 1.3)
            estimated_tokens = len((current_chunk + " " + sentence).split()) * 1.3
            
            if estimated_tokens > max_tokens and current_chunk:
                chunks.append(current_chunk.strip())
                current_chunk = sentence
            else:
                current_chunk += " " + sentence
        
        if current_chunk.strip():
            chunks.append(current_chunk.strip())
        
        return chunks
    
    def process_for_embeddings(self):
        for page in self.raw_data:
            if not page.get('content'):
                continue
            
            # Chunk the content
            content_chunks = self.chunk_content(page['content'])
            
            for i, chunk in enumerate(content_chunks):
                embedding_doc = {
                    'id': f"{page['url']}#chunk_{i}",
                    'source_url': page['url'],
                    'title': page.get('title', ''),
                    'content': chunk,
                    'chunk_index': i,
                    'total_chunks': len(content_chunks),
                    'metadata': {
                        'page_title': page.get('title', ''),
                        'meta_description': page.get('meta_description', ''),
                        'headings': page.get('headings', {}),
                        'publication_date': page.get('publication_date'),
                        'author': page.get('author'),
                        'breadcrumbs': page.get('breadcrumbs', []),
                        'crawled_at': datetime.now().isoformat()
                    }
                }
                
                # Add content categories
                embedding_doc['metadata']['content_category'] = self.categorize_content(page)
                embedding_doc['metadata']['keywords'] = self.extract_keywords(chunk)
                
                self.processed_chunks.append(embedding_doc)
    
    def categorize_content(self, page):
        url = page['url'].lower()
        title = page.get('title', '').lower()
        
        if 'about' in url or 'mission' in url:
            return 'organizational_info'
        elif 'team' in url or 'staff' in url or 'leadership' in url:
            return 'leadership_bio'
        elif 'news' in url or 'press' in url:
            return 'news_article'
        elif 'event' in url or 'tour' in url:
            return 'event_info'
        elif 'program' in url or 'initiative' in url:
            return 'program_info'
        else:
            return 'general_content'
    
    def extract_keywords(self, text):
        # Simple keyword extraction
        common_tpusa_terms = [
            'turning point', 'tpusa', 'charlie kirk', 'conservative', 'student',
            'campus', 'activism', 'freedom', 'liberty', 'free market',
            'limited government', 'fiscal responsibility', 'american exceptionalism'
        ]
        
        found_keywords = []
        text_lower = text.lower()
        
        for term in common_tpusa_terms:
            if term in text_lower:
                found_keywords.append(term)
        
        return found_keywords
    
    def save_embeddings_data(self, output_file):
        with open(output_file, 'w', encoding='utf-8') as f:
            json.dump(self.processed_chunks, f, indent=2, ensure_ascii=False)
        
        print(f"Processed {len(self.processed_chunks)} chunks for embeddings")
        print(f"Data saved to {output_file}")

# Usage
if __name__ == "__main__":
    processor = EmbeddingsDataPreparation('tpusa_crawled_data.json')
    processor.process_for_embeddings()
    processor.save_embeddings_data('tpusa_embeddings_ready.json')
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

## 7. Installation and Setup Commands

```bash
# Install required Python packages
pip install scrapy requests beautifulsoup4 selenium lxml feedparser

# Install Scrapy (if not already installed)
pip install scrapy

# Install Chrome WebDriver for Selenium
# macOS with Homebrew:
brew install chromedriver

# Or download manually from: https://chromedriver.chromium.org/

# Create project structure
mkdir tpusa_crawler
cd tpusa_crawler
scrapy startproject tpusa_spider
```

This comprehensive guide provides multiple approaches to crawling TPUSA's website, from simple command-line tools to sophisticated frameworks. Choose the approach that best fits your technical requirements and the scale of data you need to collect.