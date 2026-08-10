// nous is a EURISKO-style discovery engine.
//
// Usage:
//
//	nous run [-v N] [-cycles N] [-domain NAME]
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"

	"github.com/chazu/nous/internal/actionrelationrun"
	"github.com/chazu/nous/internal/actionrelationscore"
	"github.com/chazu/nous/internal/agenda"
	"github.com/chazu/nous/internal/causaldpproof"
	"github.com/chazu/nous/internal/causalexpv2"
	"github.com/chazu/nous/internal/causalrun"
	"github.com/chazu/nous/internal/causalv2"
	"github.com/chazu/nous/internal/configrepairexp"
	"github.com/chazu/nous/internal/engine"
	"github.com/chazu/nous/internal/gameexp"
	"github.com/chazu/nous/internal/kuberepairexp"
	"github.com/chazu/nous/internal/nogoodexp"
	"github.com/chazu/nous/internal/rewriteexp"
	"github.com/chazu/nous/internal/ruleinductionexp"
	"github.com/chazu/nous/internal/seed"
	"github.com/chazu/nous/internal/transformexp"
	"github.com/chazu/nous/internal/unit"
	causal "github.com/chazu/nous/internal/vocab/causal"
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
	case "kuberepair-trials":
		kubeRepairTrialsCmd(os.Args[2:])
	case "game-trials":
		gameTrialsCmd(os.Args[2:])
	case "nogood-trials":
		nogoodTrialsCmd(os.Args[2:])
	case "ruleinduction-trials":
		ruleInductionTrialsCmd(os.Args[2:])
	case "transform-schema-trials":
		transformSchemaTrialsCmd(os.Args[2:])
	case "causal-trials":
		causalTrialsCmd(os.Args[2:])
	case "actionrelation-trials":
		actionRelationTrialsCmd(os.Args[2:])
	case "help", "-h", "--help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		usage()
	}
}

func actionRelationTrialsCmd(args []string) {
	fs := flag.NewFlagSet("actionrelation-trials", flag.ExitOnError)
	repoRoot := fs.String("repo-root", ".", "canonical repository root")
	stage := fs.String("stage", "prepare", "prepare, competence, claim, or execute")
	panel := fs.String("panel", "development", "development, validation, or locked")
	_ = fs.String("unlock-token", "", "locked execute token actionrelations/v1:<exact-clean-HEAD>")
	fs.Parse(args)
	root, err := filepath.Abs(*repoRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	var canonical []byte
	switch *stage {
	case "prepare":
		if *panel == "development" {
			value, runErr := actionrelationrun.PreparePrerequisites(context.Background(), root)
			err, canonical = runErr, value.Canonical
		} else {
			value, runErr := actionrelationrun.PrepareProtected(context.Background(), root, *panel, os.Args)
			err, canonical = runErr, value.Canonical
		}
	case "competence":
		value, runErr := actionrelationrun.ExecuteCompetence(root, os.Args)
		err, canonical = runErr, value.Canonical
	case "claim":
		value, runErr := actionrelationrun.ClaimProtected(context.Background(), root, *panel, os.Args)
		err, canonical = runErr, value.Canonical
	case "execute":
		var value actionrelationscore.Report
		var runErr error
		if *panel == "development" {
			value, runErr = actionrelationrun.ExecuteDevelopment(context.Background(), root, os.Args)
		} else {
			value, runErr = actionrelationrun.ExecuteProtected(context.Background(), root, *panel, os.Args)
		}
		err, canonical = runErr, value.Canonical
	default:
		err = fmt.Errorf("unknown actionrelation stage %q", *stage)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(canonical))
}

func transformSchemaTrialsCmd(args []string) {
	fs := flag.NewFlagSet("transform-schema-trials", flag.ExitOnError)
	repoRoot := fs.String("repo-root", ".", "canonical repository root")
	domainsDir := fs.String("domains-dir", "domains", "canonical repository domains/ path")
	panel := fs.String("panel", "development", "development, validation, or locked")
	unlockToken := fs.String("unlock-token", "", "locked token transform-schema/v1:<exact-clean-HEAD>")
	fs.Parse(args)
	root, err := filepath.Abs(*repoRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	domains := *domainsDir
	if !filepath.IsAbs(domains) {
		domains = filepath.Join(root, domains)
	}
	var encoded []byte
	switch *panel {
	case "development":
		report, runErr := transformexp.ExecuteDevelopment(root, domains)
		err = runErr
		if err == nil {
			encoded, err = report.JSON()
		}
	case "validation":
		report, runErr := transformexp.ExecuteValidation(root, domains)
		err = runErr
		if err == nil {
			encoded, err = report.JSON()
		}
	case "locked":
		report, runErr := transformexp.ExecuteLocked(root, domains, *unlockToken)
		err = runErr
		if err == nil {
			encoded, err = report.JSON()
		}
	default:
		err = fmt.Errorf("unknown transform-schema panel %q", *panel)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(encoded))
}

func nogoodTrialsCmd(args []string) {
	fs := flag.NewFlagSet("nogood-trials", flag.ExitOnError)
	repoRoot := fs.String("repo-root", ".", "canonical repository root")
	domainsDir := fs.String("domains-dir", "domains", "canonical repository domains/ path")
	panel := fs.String("panel", "development", "development, validation, or locked")
	unlockToken := fs.String("unlock-token", "", "locked token nogoods/v2:<exact-clean-HEAD>")
	fs.Parse(args)
	root, err := filepath.Abs(*repoRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	domains := *domainsDir
	if !filepath.IsAbs(domains) {
		domains = filepath.Join(root, domains)
	}
	var report nogoodexp.Report
	switch *panel {
	case "development":
		report, err = nogoodexp.ExecuteDevelopment(root, domains)
	case "validation":
		report, err = nogoodexp.ExecuteValidation(root, domains)
	case "locked":
		report, err = nogoodexp.ExecuteLocked(root, domains, *unlockToken)
	default:
		err = fmt.Errorf("unknown nogood panel %q", *panel)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	encoded, err := json.Marshal(report)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: encode report: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(encoded))
}

func kubeRepairTrialsCmd(args []string) {
	fs := flag.NewFlagSet("kuberepair-trials", flag.ExitOnError)
	domainsDir := fs.String("domains-dir", "domains", "filesystem path to domains/ directory")
	panel := fs.String("panel", "development", "development, validation, or locked")
	fs.Parse(args)
	report, err := kuberepairexp.Run(*domainsDir, *panel)
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

func causalTrialsCmd(args []string) {
	fs := flag.NewFlagSet("causal-trials", flag.ExitOnError)
	repoRoot := fs.String("repo-root", ".", "repository root")
	panel := fs.String("panel", "development", "proof, training, replay, development, validation, or locked")
	fs.Parse(args)
	root, err := filepath.Abs(*repoRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: resolve repository root: %v\n", err)
		os.Exit(1)
	}
	ctx := context.Background()
	switch *panel {
	case "proof":
		proof, runErr := causaldpproof.Run()
		if runErr != nil {
			err = runErr
			break
		}
		encoded, encodeErr := causalv2.CanonicalJSON(proof)
		if encodeErr != nil {
			err = encodeErr
			break
		}
		fmt.Println(string(encoded))
		return
	case "development":
		diagnostics, runErr := causalexpv2.RunDevelopment(ctx, root)
		if runErr != nil {
			err = runErr
			break
		}
		encoded, encodeErr := causalv2.CanonicalJSON(diagnostics)
		if encodeErr != nil {
			err = encodeErr
			break
		}
		fmt.Println(string(encoded))
		return
	case "training":
		err = causalexpv2.ExecuteProtectedPanel(ctx, root, causalexpv2.PanelTraining)
	case "replay":
		err = causalexpv2.ExecuteReplay(ctx, root)
		if err == nil {
			fmt.Println(`{"status":"replay-succeeded"}`)
			return
		}
	case "validation":
		err = causalexpv2.ExecuteProtectedPanel(ctx, root, causalexpv2.PanelValidation)
	case "locked":
		err = causalexpv2.ExecuteProtectedPanel(ctx, root, causalexpv2.PanelLocked)
	default:
		err = fmt.Errorf("unknown causal panel %q", *panel)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	reportPath, err := causalReportPath(root, *panel)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: locate report: %v\n", err)
		os.Exit(1)
	}
	report, err := os.ReadFile(reportPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: read report: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(report))
}

func causalReportPath(repoRoot, panel string) (string, error) {
	if panel == "training" {
		return filepath.Join(repoRoot, causalexpv2.TrainingEvidenceDirectory, causalexpv2.TrainingReportName), nil
	}
	command := exec.Command("git", "-C", repoRoot, "rev-parse", "--git-common-dir")
	output, err := command.Output()
	if err != nil {
		return "", err
	}
	common := strings.TrimSpace(string(output))
	if !filepath.IsAbs(common) {
		common = filepath.Join(repoRoot, common)
	}
	return filepath.Join(filepath.Clean(common), causalexpv2.ResultsDirectoryName,
		"active-causal-diagnosis-v3-"+panel+".json"), nil
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
	if *domain == "causal" {
		runCausalDevelopmentBoundary()
		return
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

type externalCausalTeacher struct{}

func (externalCausalTeacher) Respond(_, _ string) (string, error) {
	return "", fmt.Errorf("standalone causal run stops before the external teacher call")
}

func runCausalDevelopmentBoundary() {
	manifest := causalv2.PreregisteredManifest()
	fixture, err := causalexpv2.NewDiagnosticDevelopmentCapability().GenerateDevelopment(manifest.DevelopmentSeeds.Start, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: generate causal development fixture: %v\n", err)
		os.Exit(1)
	}
	publicBytes, err := causalv2.CanonicalJSON(fixture.PublicFixture)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: encode causal fixture: %v\n", err)
		os.Exit(1)
	}
	rules := causal.Rules()
	if len(rules) == 0 {
		fmt.Fprintln(os.Stderr, "error: causal acquisition grammar is empty")
		os.Exit(1)
	}
	profile := causalv2.Profile{
		ProfileVersion:  causalv2.ProfileDomain,
		Manifest:        manifest,
		Panel:           "development",
		Seed:            fixture.PublicFixture.Seed,
		AcquisitionCode: rules[0].Code(),
		FixtureDigest:   fixture.PublicFixture.FixtureDigest,
	}
	if err := causalv2.SignProfile(&profile); err != nil {
		fmt.Fprintf(os.Stderr, "error: sign causal profile: %v\n", err)
		os.Exit(1)
	}
	profileBytes, err := causalv2.CanonicalJSON(profile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: encode causal profile: %v\n", err)
		os.Exit(1)
	}
	runner, err := causalrun.NewEpisode(publicBytes, profileBytes, externalCausalTeacher{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: start causal episode: %v\n", err)
		os.Exit(1)
	}
	defer runner.Close()
	boundary, err := runner.AdvanceToTeacher(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: reach causal teacher boundary: %v\n", err)
		os.Exit(1)
	}
	encoded, err := causalv2.CanonicalJSON(boundary)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: encode causal boundary: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("nous: causal development seed %d reached verified external-teacher boundary\n%s\n", fixture.PublicFixture.Seed, encoded)
}

func usage() {
	fmt.Fprintf(os.Stderr, `nous: a EURISKO-style discovery engine

Usage:
  nous run [-v N] [-cycles N] [-domain NAME]    Run the discovery engine
  nous rewrite-trials [flags]                   Run rewrite experiments
  nous configrepair-trials [flags]              Run Kubernetes/Terraform repair trials
  nous kuberepair-trials -panel NAME            Run atomic Kubernetes repair ordering trials
  nous game-trials [flags]                      Run iterated-game strategy trials
  nous nogood-trials -panel NAME                Run guarded nogood development/validation/locked panel
  nous ruleinduction-trials [flags]             Run relational rule-induction development trials
  nous transform-schema-trials -panel NAME      Run guarded transformation-schema panels
  nous causal-trials -panel NAME                Run v2 causal development/training/replay/validation/locked panel
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
