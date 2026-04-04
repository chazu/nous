// nous is a EURISKO-style discovery engine.
//
// Usage:
//
//	nous run [-v N] [-cycles N]
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"

	"github.com/chazu/nous/internal/agenda"
	"github.com/chazu/nous/internal/engine"
	"github.com/chazu/nous/internal/seed"
	"github.com/chazu/nous/internal/unit"
)

func main() {
	if len(os.Args) < 2 {
		usage()
	}

	switch os.Args[1] {
	case "run":
		runCmd(os.Args[2:])
	case "help", "-h", "--help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		usage()
	}
}

func runCmd(args []string) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	verbosity := fs.Int("v", 1, "verbosity level (0=quiet, 1=normal, 2=detailed, 3=debug)")
	maxCycles := fs.Int("cycles", 100, "maximum number of cycles")
	domain := fs.String("domain", "math", "seed domain to load ("+seed.Available()+")")
	noMutate := fs.Bool("no-mutate", false, "disable heuristic mutation")
	fs.Parse(args)

	// Build the system
	store := unit.NewStore()
	ag := agenda.New()

	// Load seed knowledge
	if err := seed.LoadDomain(store, *domain); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	seed.LoadHeuristics(store)

	fmt.Printf("nous: loaded %d units (%d heuristics)\n",
		store.Count(), len(store.Examples("Heuristic")))

	// Create and configure engine
	eng := engine.New(store, ag)
	eng.MaxCycles = *maxCycles
	eng.Verbosity = *verbosity
	if *noMutate {
		eng.MutConfig.Enabled = false
	}

	// Run with interrupt handling
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	if err := eng.Run(ctx); err != nil && err != context.Canceled {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	// Print final state
	fmt.Printf("\n%s\n", eng.Stats())
	eng.DumpWorths()
}

func usage() {
	fmt.Fprintf(os.Stderr, `nous: a EURISKO-style discovery engine

Usage:
  nous run [-v N] [-cycles N] [-domain NAME]    Run the discovery engine
  nous help                                     Show this help

Flags:
  -v N          Verbosity (0=quiet, 1=normal, 2=detailed, 3=debug)
  -cycles N     Maximum cycles (default 100)
  -domain NAME  Seed domain to load (default: math)
`)
	os.Exit(1)
}
