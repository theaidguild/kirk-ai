package main

import (
	"flag"
	"fmt"
	"os"
)

func printUsage() {
	fmt.Println("Usage: processor <tool>")
	fmt.Println("Available tools:")
	fmt.Println("  content   - process raw HTML into cleaned JSON")
	fmt.Println("  embedprep - prepare processed pages into embedding-ready chunks")
}

func main() {
	flag.Parse()
	if flag.NArg() == 0 {
		printUsage()
		os.Exit(1)
	}
	tool := flag.Arg(0)
	switch tool {
	case "content":
		runContentProcessor()
	case "embedprep":
		runPrepareEmbeddings()
	default:
		fmt.Fprintf(os.Stderr, "unknown tool: %s\n", tool)
		printUsage()
		os.Exit(2)
	}
}
