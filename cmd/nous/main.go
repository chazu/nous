// nous is a EURISKO-style discovery engine.
//
// Usage:
//
//	nous run [-v N] [-cycles N] [-domain NAME]
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"

	"github.com/chazu/nous/internal/agenda"
	"github.com/chazu/nous/internal/configrepairexp"
	"github.com/chazu/nous/internal/engine"
	"github.com/chazu/nous/internal/gameexp"
	"github.com/chazu/nous/internal/rewriteexp"
	"github.com/chazu/nous/internal/ruleinductionexp"
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
	case "rewrite-trials":
		rewriteTrialsCmd(os.Args[2:])
	case "configrepair-trials":
		configurationRepairTrialsCmd(os.Args[2:])
	case "game-trials":
		gameTrialsCmd(os.Args[2:])
	case "ruleinduction-trials":
		ruleInductionTrialsCmd(os.Args[2:])
	case "help", "-h", "--help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		usage()
	}
}

func ruleInductionTrialsCmd(args []string) {
	fs := flag.NewFlagSet("ruleinduction-trials", flag.ExitOnError)
	domainsDir := fs.String("domains-dir", "domains", "filesystem path to domains/ directory")
	panel := fs.String("panel", "development", "development, training, validation, or locked")
	implementationCommit := fs.String("implementation-commit", "", "immutable implementation commit (required for locked panel)")
	fs.Parse(args)
	if *panel == "locked" && *implementationCommit == "" {
		fmt.Fprintln(os.Stderr, "error: -implementation-commit is required for the locked panel")
		os.Exit(2)
	}
	var report ruleinductionexp.Report
	var err error
	if *panel == "locked" {
		report, err = ruleinductionexp.RunLockedPanel(*domainsDir, *implementationCommit)
	} else {
		report, err = ruleinductionexp.RunPanel(*domainsDir, *panel)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	if *panel != "locked" {
		report.ImplementationCommit = *implementationCommit
	}
	encoded, err := report.JSON()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: encode report: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(encoded))
}

func gameTrialsCmd(args []string) {
	fs := flag.NewFlagSet("game-trials", flag.ExitOnError)
	domainsDir := fs.String("domains-dir", "domains", "filesystem path to domains/ directory")
	fs.Parse(args)

	report, err := gameexp.Run(*domainsDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	encoded, err := report.JSON()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: encode report: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(encoded))
}

func configurationRepairTrialsCmd(args []string) {
	fs := flag.NewFlagSet("configrepair-trials", flag.ExitOnError)
	domainsDir := fs.String("domains-dir", "domains", "filesystem path to domains/ directory")
	fs.Parse(args)

	report, err := configrepairexp.Run(*domainsDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	encoded, err := report.JSON()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: encode report: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(encoded))
}

func rewriteTrialsCmd(args []string) {
	fs := flag.NewFlagSet("rewrite-trials", flag.ExitOnError)
	domainsDir := fs.String("domains-dir", "domains", "filesystem path to domains/ directory")
	trialSeed := fs.Int64("seed", 4242, "deterministic experiment seed")
	problems := fs.Int("problems", 40, "generated robustness problems")
	curricula := fs.Int("curricula", 60, "two-stage credit curricula")
	budget := fs.Int("budget", 4, "phase-two candidate evaluation budget")
	fs.Parse(args)

	report, err := rewriteexp.Run(*domainsDir, *trialSeed, *problems, *curricula, *budget)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	encoded, err := report.JSON()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: encode report: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(encoded))
}

func runCmd(args []string) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	verbosity := fs.Int("v", 1, "verbosity level (0=quiet, 1=normal, 2=detailed, 3=debug)")
	maxCycles := fs.Int("cycles", 100, "maximum number of cycles")
	domain := fs.String("domain", "math", "seed domain to load ("+seed.Available()+")")
	noMutate := fs.Bool("no-mutate", false, "disable heuristic mutation")
	storeJSON := fs.Bool("store-json", false, "print canonical final-store JSON")
	domainsDir := fs.String("domains-dir", "", "filesystem path to domains/ directory")
	fs.Parse(args)

	// Set domains directory
	if *domainsDir != "" {
		seed.DomainsDir = *domainsDir
	} else {
		// Try ./domains/ (running from project root)
		if _, err := os.Stat("domains"); err == nil {
			seed.DomainsDir = "domains"
		} else {
			fmt.Fprintf(os.Stderr, "error: cannot find domains/ directory. Use -domains-dir flag.\n")
			os.Exit(1)
		}
	}

	// Build the system
	store := unit.NewStore()
	ag := agenda.New()

	// Load seed knowledge
	if err := seed.LoadDomain(store, *domain); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
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

	// Seed the agenda with one task per Op (option B for breadth coverage;
	// see engine.SeedInitialAgenda docstring).
	eng.SeedInitialAgenda()

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
	if *storeJSON {
		snapshot, err := store.CanonicalJSON()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: snapshot store: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("\n--- Canonical Store JSON ---\n%s\n", snapshot)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `nous: a EURISKO-style discovery engine

Usage:
  nous run [-v N] [-cycles N] [-domain NAME]    Run the discovery engine
  nous rewrite-trials [flags]                   Run rewrite experiments
  nous configrepair-trials [flags]              Run Kubernetes/Terraform repair trials
  nous game-trials [flags]                      Run iterated-game strategy trials
  nous ruleinduction-trials [flags]             Run relational rule-induction development trials
  nous help                                     Show this help

Flags:
  -v N            Verbosity (0=quiet, 1=normal, 2=detailed, 3=debug)
  -cycles N       Maximum cycles (default 100)
  -domain NAME    Seed domain to load (default: math)
  -domains-dir D  Filesystem path to domains/ directory
  -no-mutate      Disable heuristic mutation
  -store-json     Print canonical final-store JSON
`)
	os.Exit(1)
}
