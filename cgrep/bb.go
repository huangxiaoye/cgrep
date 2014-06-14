package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	files := os.Args[1:]

	for _, name := range files {
		matches, _ := filepath.Glob(name)
		fmt.Println(matches)

	}
}
