package main

import (
	"context"
	"fmt"
	"os"

	"github.com/chazu/nous/internal/causalexpv2"
)

func main() {
	result := "diagnostic-failure"
	var err error
	if len(os.Args) != 1 {
		err = fmt.Errorf("arguments are not accepted")
	} else {
		var root string
		root, err = os.Getwd()
		if err == nil {
			result, err = causalexpv2.ExecuteCacheDiagnostic(context.Background(), root)
		}
	}
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "diagnostic: diagnostic-failure")
		os.Exit(1)
	}
	_, _ = fmt.Fprintf(os.Stdout, "diagnostic: %s\n", result)
}
