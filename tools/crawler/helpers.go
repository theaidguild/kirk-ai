package main

import (
	"io/ioutil"
	"log"
	"os"
	"strings"
)

// ensureDir is shared across crawler tools to avoid duplicate definitions
func ensureDir(p string) {
	if err := os.MkdirAll(p, 0o755); err != nil {
		log.Fatalf("mkdir %s: %v", p, err)
	}
}

// readURLsFromFile returns non-empty trimmed lines from a file or an error.
func readURLsFromFile(path string) ([]string, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	lines := []string{}
	for _, l := range strings.Split(string(b), "\n") {
		if s := strings.TrimSpace(l); s != "" {
			lines = append(lines, s)
		}
	}
	return lines, nil
}
