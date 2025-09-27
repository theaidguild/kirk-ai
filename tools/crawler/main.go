package main

import (
	"flag"
	"fmt"
	"os"
)

func printUsage() {
	fmt.Println("Usage: crawler <tool>")
	fmt.Println("Available tools:")
	fmt.Println("  api    - run API endpoint checker and feed fetcher")
	fmt.Println("  colly  - run colly-based crawler")
	fmt.Println("  chromedp - run chromedp-based crawler")
	fmt.Println("  requests - run simple requests-based crawler")
}

func main() {
	flag.Parse()
	if flag.NArg() == 0 {
		printUsage()
		os.Exit(1)
	}
	tool := flag.Arg(0)
	switch tool {
	case "api":
		runAPIDataCollector()
	case "colly":
		runCollyCrawler()
	case "chromedp":
		runChromedpCrawler()
	case "requests":
		runRequestsCrawler()
	default:
		fmt.Fprintf(os.Stderr, "unknown tool: %s\n", tool)
		printUsage()
		os.Exit(2)
	}
}
