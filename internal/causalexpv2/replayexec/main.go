package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/chazu/nous/internal/causalexpv2"
)

func main() {
	err := errors.New("causal-v2 replay worker accepts no arguments")
	if len(os.Args) == 1 {
		err = causalexpv2.ReplayRegenerate(context.Background())
	}
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
