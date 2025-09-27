package main

import (
	"log"
	"os"
)

// ensureDir is shared across crawler tools to avoid duplicate definitions
func ensureDir(p string) {
	if err := os.MkdirAll(p, 0o755); err != nil {
		log.Fatalf("mkdir %s: %v", p, err)
	}
}
