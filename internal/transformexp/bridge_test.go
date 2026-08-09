package transformexp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/chazu/nous/internal/unit"
	transformschema "github.com/chazu/nous/internal/vocab/transformschema"
)

func TestOrdinaryHeuristicsAcquireAndAllocate(t *testing.T) {
	c, err := makeCurriculum(0, 8, 841001)
	if err != nil {
		t.Fatal(err)
	}
	run, err := runAcquisition("../../domains", c.Training, "safe-test")
	if err != nil {
		t.Fatal(err)
	}
	if run.Terminal != "completed" || len(run.Programs) != 4 || len(run.Candidates) != 13 || run.Artifact == "" {
		for _, name := range run.Candidates {
			t.Logf("candidate %s stage=%s value=%s status=%s parent=%s partial=%s", name, run.Store.Get(name).GetString("stage"), run.Store.Get(name).GetString("value"), run.Store.Get(name).GetString("status"), run.Store.Get(name).GetString("parentCandidate"), run.Store.Get(name).GetString("partial"))
		}
		t.Fatalf("terminal=%s programs=%d candidates=%d tasks=%d", run.Terminal, len(run.Programs), len(run.Candidates), run.TasksPopped)
	}
	assertRefinementProvenance(t, run)
	if got := []byte(run.Store.Get(run.Artifact).GetString("schema")); !bytes.Equal(got, c.Latent) {
		t.Fatalf("artifact schema=%s latent=%s", got, c.Latent)
	}
	if len(run.MeterRecords) != 1737 {
		t.Fatalf("meter records=%d", len(run.MeterRecords))
	}
	closures, frozen := 0, 0
	for i, record := range run.MeterRecords {
		if record.Phase == "" || len(record.Inputs) == 0 {
			t.Fatalf("meter record %d lacks semantic preimage: %+v", i, record)
		}
		if record.Operation == "verify" && record.Phase == "freeze" && len(record.Inputs) == 1 {
			if objectVersion(record.Inputs[0], "transform-closure/v1") {
				closures++
			}
			if objectVersion(record.Inputs[0], "transform-schema/v1") {
				frozen++
			}
		}
	}
	if closures != 5 || frozen != 1 {
		t.Fatalf("authenticated closures=%d frozen artifacts=%d", closures, frozen)
	}
	survivors := map[string][]string{}
	for _, name := range run.Candidates {
		u := run.Store.Get(name)
		if u.GetString("status") == "survivor" {
			stage := u.GetString("stage")
			survivors[stage] = append(survivors[stage], u.GetString("value"))
		}
	}
	if got := survivors["target"]; len(got) != 1 || got[0] != "definition+references" {
		t.Fatalf("target survivors=%v", got)
	}
	if got := survivors["anchor"]; len(got) != 1 || got[0] != "request-target" {
		t.Fatalf("anchor survivors=%v", got)
	}
	if got := survivors["scope"]; len(got) != 1 || got[0] != "global" {
		t.Fatalf("scope survivors=%v", got)
	}
	if got := survivors["old-guard"]; len(got) != 1 || got[0] != "any" {
		t.Fatalf("old-guard survivors=%v", got)
	}
	if got := survivors["locality"]; len(got) != 1 || got[0] != "required" {
		t.Fatalf("locality survivors=%v", got)
	}
}

func TestReducerRejectsForgedStageClosure(t *testing.T) {
	c, err := makeCurriculum(0, 8, 841001)
	if err != nil {
		t.Fatal(err)
	}
	run, err := runAcquisition("../../domains", c.Training, "closure-forgery")
	if err != nil {
		t.Fatal(err)
	}
	bundle, err := transcriptFromAcquisition(run, 0, NousRefine, "0123456789abcdef", digestBytes([]byte("closure manifest")))
	if err != nil {
		t.Fatal(err)
	}
	state := newTransformLifecycleState(string(NousRefine))
	scanner := bufio.NewScanner(bytes.NewReader(bundle.Raw))
	for scanner.Scan() {
		event, _ := parseTransformEvent(scanner.Bytes())
		operation, _ := parseTransformOperation(bundle.Objects[event.Object])
		if operation.Operation != "verify" || !objectVersion(bundle.Objects[operation.Inputs[0]], "transform-closure/v1") {
			if err := state.observe(operation, bundle.Objects); err != nil {
				t.Fatal(err)
			}
			continue
		}
		var closure []any
		if json.Unmarshal(bundle.Objects[operation.Inputs[0]], &closure) != nil {
			t.Fatal("closure did not decode")
		}
		closure[4] = strings.Repeat("0", 64)
		forged := mustJSON(closure)
		objects := map[string][]byte{}
		for digest, value := range bundle.Objects {
			objects[digest] = value
		}
		forgedDigest := digestBytes(forged)
		objects[forgedDigest] = forged
		operation.Inputs[0] = forgedDigest
		if err := state.observe(operation, objects); err == nil {
			t.Fatal("forged closure survivor was accepted")
		}
		return
	}
	t.Fatal("transcript contained no closure")
}

func assertRefinementProvenance(t *testing.T, run acquisitionRun) {
	t.Helper()
	if run.Root == "" || len(run.Edges) != 12 {
		t.Fatalf("root=%q edges=%d", run.Root, len(run.Edges))
	}
	root, err := transformschema.ParsePartial([]byte(run.Store.Get(run.Root).GetString("partial")))
	if err != nil || root.Stage != 0 {
		t.Fatalf("root partial=%+v err=%v", root, err)
	}
	seenChildren := map[string]bool{}
	for _, edgeName := range run.Edges {
		edge := run.Store.Get(edgeName)
		parentName, childName := edge.GetString("parentCandidate"), edge.GetString("childCandidate")
		parent, err := transformschema.ParsePartial([]byte(run.Store.Get(parentName).GetString("partial")))
		if err != nil {
			t.Fatalf("edge %s parent %s: %v", edgeName, parentName, err)
		}
		want, err := parent.Refine(edge.GetString("value"))
		if err != nil {
			t.Fatalf("edge %s refine: %v", edgeName, err)
		}
		got, err := transformschema.ParsePartial([]byte(run.Store.Get(childName).GetString("partial")))
		if err != nil || got != want {
			t.Fatalf("edge %s child=%+v want=%+v err=%v", edgeName, got, want, err)
		}
		if run.Store.Get(childName).GetString("refinementEdge") != edgeName {
			t.Fatalf("child %s does not link edge %s", childName, edgeName)
		}
		seenChildren[childName] = true
	}
	if len(seenChildren) != 12 {
		t.Fatalf("unique refined children=%d", len(seenChildren))
	}
	for _, name := range run.Candidates {
		if name != run.Root && run.Store.Get(name).GetString("evidenceUnit") == "" {
			t.Fatalf("candidate %s lacks factor evidence", name)
		}
	}
	artifact := run.Store.Get(run.Artifact)
	if got := len(artifact.GetStrings("evidenceBarriers")); got != 5 {
		t.Fatalf("artifact evidence barriers=%d", got)
	}
}

func TestAcquisitionRequiresOrdinaryCausalHeuristics(t *testing.T) {
	c, err := makeCurriculum(0, 8, 841001)
	if err != nil {
		t.Fatal(err)
	}
	for _, heuristic := range []string{
		"H-TransformAcquireConcretePrograms",
		"H-TransformRefineSchemaFactors",
		"H-TransformEvaluateFactor",
		"H-TransformCloseEvidenceBarriers",
	} {
		run, err := runAcquisitionConfigured("../../domains", c.Training, "without-"+heuristic, func(store *unit.Store) {
			store.Delete(heuristic)
		})
		if err != nil {
			t.Fatalf("delete %s: %v", heuristic, err)
		}
		if run.Terminal == "completed" || run.Artifact != "" {
			t.Fatalf("delete %s still completed with artifact %s", heuristic, run.Artifact)
		}
	}
}

func TestAcquisitionOnlyStopsBeforeFactorSearch(t *testing.T) {
	c, err := makeCurriculum(0, 8, 841001)
	if err != nil {
		t.Fatal(err)
	}
	run, err := runAcquisitionConfigured("../../domains", c.Training, "acquisition-only", func(store *unit.Store) {
		store.Get("H-TransformAcquireConcretePrograms").Set("acquisitionOnly", true)
	})
	if err != nil {
		t.Fatal(err)
	}
	if run.Terminal != "" || len(run.Programs) != 4 || len(run.Candidates) != 0 || run.TasksPopped != 1 {
		t.Fatalf("acquisition-only terminal=%q programs=%d candidates=%d tasks=%d", run.Terminal, len(run.Programs), len(run.Candidates), run.TasksPopped)
	}
	operations := map[string]int{}
	for _, record := range run.MeterRecords {
		operations[record.Operation]++
	}
	if operations["edit-validate"] == 0 || operations["edit-apply"] == 0 || operations["schema-application"] != 0 || operations["refine"] != 0 {
		t.Fatalf("acquisition-only operations=%v", operations)
	}
}

func TestOrdinaryHeuristicsRecoverEverySemanticFamily(t *testing.T) {
	for family := range familySchemas {
		c, err := makeCurriculum(family, family, 841100+uint64(family))
		if err != nil {
			t.Fatalf("family %d curriculum: %v", family, err)
		}
		run, err := runAcquisition("../../domains", c.Training, "family-"+string(rune('0'+family)))
		if err != nil {
			t.Fatalf("family %d acquisition: %v", family, err)
		}
		if run.Terminal != "completed" || run.Artifact == "" {
			t.Fatalf("family %d terminal=%s artifact=%q", family, run.Terminal, run.Artifact)
		}
		got := []byte(run.Store.Get(run.Artifact).GetString("schema"))
		if !bytes.Equal(got, c.Latent) {
			t.Fatalf("family %d artifact=%s latent=%s", family, got, c.Latent)
		}
	}
}
