package main

import (
	"embed"
	"fmt"
	"os"
	"sort"
	"strings"
)

//go:embed guides/*.txt
var guides embed.FS

var guideNames = map[string]string{
	"observations": "How to record observations (start here)",
	"query":        "How to check existing data before observing",
	"scope":        "Scope format reference",
	"overview":     "What nous is and how agents participate",
}

func guideCmd(args []string) {
	if len(args) == 0 {
		listGuides()
		return
	}

	name := args[0]
	data, err := guides.ReadFile("guides/" + name + ".txt")
	if err != nil {
		fmt.Fprintf(os.Stderr, "unknown guide: %s\n\n", name)
		listGuides()
		os.Exit(1)
	}

	fmt.Print(string(data))
}

func listGuides() {
	fmt.Fprintf(os.Stderr, "Available guides:\n\n")
	names := make([]string, 0, len(guideNames))
	for n := range guideNames {
		names = append(names, n)
	}
	sort.Strings(names)

	maxLen := 0
	for _, n := range names {
		if len(n) > maxLen {
			maxLen = len(n)
		}
	}

	for _, n := range names {
		padding := strings.Repeat(" ", maxLen-len(n)+2)
		fmt.Fprintf(os.Stderr, "  nous guide %s%s%s\n", n, padding, guideNames[n])
	}
	fmt.Fprintln(os.Stderr)
}
