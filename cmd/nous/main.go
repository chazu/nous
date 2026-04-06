// nous is a EURISKO-style discovery engine.
//
// Usage:
//
//	nous run [-v N] [-cycles N] [-domain NAME]
//	nous run -domain observations -pudl ~/.pudl
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"

	"github.com/chazu/nous/internal/agenda"
	"github.com/chazu/nous/internal/engine"
	"github.com/chazu/nous/internal/pudlbridge"
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
	pudlDir := fs.String("pudl", "", "pudl config directory (enables Mode 2, reads from pudl fact store)")
	fs.Parse(args)

	// Build the system
	store := unit.NewStore()
	ag := agenda.New()

	// Load seed knowledge
	if err := seed.LoadDomain(store, *domain); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	// Load heuristics — use observation heuristics for Mode 2
	if *domain == "observations" {
		seed.LoadObservationHeuristics(store)
	} else {
		seed.LoadHeuristics(store)
	}

	// Mode 2: load facts from pudl
	var bridge *pudlbridge.Bridge
	if *pudlDir != "" {
		var err error
		bridge, err = pudlbridge.New(*pudlDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error opening pudl: %v\n", err)
			os.Exit(1)
		}
		defer bridge.Close()

		obsCount, err := bridge.LoadObservations(store)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error loading observations: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("nous: loaded %d observations from pudl\n", obsCount)
	}

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

	// Write discoveries back to pudl
	if bridge != nil {
		writeDiscoveries(bridge, store)
	}

	// Print final state
	fmt.Printf("\n%s\n", eng.Stats())
	eng.DumpWorths()
}

// writeDiscoveries records new units created during the run back to pudl.
func writeDiscoveries(bridge *pudlbridge.Bridge, store *unit.Store) {
	for _, name := range store.All() {
		u := store.Get(name)
		if u == nil {
			continue
		}
		isA := u.GetStrings("isA")

		// Write conjectures back as observations
		for _, t := range isA {
			if t == "Conjecture" {
				desc := u.GetString("english")
				if desc == "" {
					desc = name
				}
				kind := u.GetString("kind")
				if kind == "" {
					kind = "pattern"
				}
				_, err := bridge.WriteFact("observation", map[string]interface{}{
					"kind":        kind,
					"description": desc,
					"source":      "nous",
					"status":      "raw",
					"worth":       float64(u.GetInt("worth")) / 1000.0,
				}, "nous")
				if err != nil {
					// Dedup conflict is fine — means we already wrote this
					continue
				}
				fmt.Printf("nous → pudl: %s\n", desc)
				break
			}
		}

		// Write repo hotspots back
		for _, t := range isA {
			if t == "RepoHotspot" {
				repo := u.GetString("repo")
				count := u.GetInt("observation_count")
				if repo != "" {
					_, err := bridge.WriteFact("repo_hotspot", map[string]interface{}{
						"repo":              repo,
						"observation_count": count,
					}, "nous")
					if err != nil {
						continue
					}
					fmt.Printf("nous → pudl: hotspot %s (%d observations)\n", repo, count)
				}
				break
			}
		}
	}
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
  -no-mutate    Disable heuristic mutation
  -pudl DIR     pudl config directory (Mode 2: reason over pudl facts)
`)
	os.Exit(1)
}
