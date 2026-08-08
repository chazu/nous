package seed

import (
	"context"
	"io"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/chazu/nous/internal/agenda"
	"github.com/chazu/nous/internal/credit"
	"github.com/chazu/nous/internal/engine"
	"github.com/chazu/nous/internal/kuberepairfixture"
	"github.com/chazu/nous/internal/kuberepairoracle"
	"github.com/chazu/nous/internal/unit"
	kuberepair "github.com/chazu/nous/internal/vocab/kuberepair"
)

func loadKubeRepair(t *testing.T) (*unit.Store, func()) {
	t.Helper()
	caseData, err := kuberepairfixture.Seed()
	if err != nil {
		t.Fatal(err)
	}
	cleanup, err := kuberepair.RegisterIntent(caseData.Handle, caseData.Intent)
	if err != nil {
		t.Fatal(err)
	}
	store := unit.NewStore()
	DomainsDir = "../../domains"
	if err := LoadDomain(store, "kuberepair"); err != nil {
		cleanup()
		t.Fatal(err)
	}
	return store, cleanup
}

func runKubeRepair(t *testing.T, store *unit.Store, simplify bool) *engine.Engine {
	t.Helper()
	experiment := store.Get("KubeRepairExperiment")
	eng := engine.New(store, agenda.New())
	eng.Out = io.Discard
	eng.VM.Out = io.Discard
	eng.MutConfig.Enabled = false
	eng.WorkOnTask(&agenda.Task{Priority: 800, UnitName: experiment.Name, SlotName: experiment.GetString("synthesisTaskSlot")})
	for _, candidate := range experiment.GetStrings("candidateUnits") {
		eng.WorkOnTask(&agenda.Task{Priority: 700, UnitName: candidate, SlotName: experiment.GetString("evaluationTaskSlot")})
	}
	eng.WorkOnTask(&agenda.Task{Priority: 600, UnitName: experiment.Name, SlotName: experiment.GetString("finalizationTaskSlot")})
	if simplify {
		eng.WorkOnTask(&agenda.Task{Priority: 550, UnitName: experiment.Name, SlotName: experiment.GetString("simplificationTaskSlot")})
	}
	return eng
}

func kubeMembers(store *unit.Store, category string) []*unit.Unit {
	var out []*unit.Unit
	for _, name := range store.Examples(category) {
		if name != category {
			out = append(out, store.Get(name))
		}
	}
	return out
}

func TestKubeRepairSeedDiscoversCompleteMinimumSet(t *testing.T) {
	store, cleanup := loadKubeRepair(t)
	defer cleanup()
	eng := runKubeRepair(t, store, true)
	experiment := store.Get("KubeRepairExperiment")
	if !experiment.GetBool("generationComplete") || !experiment.GetBool("finalizationComplete") || !experiment.GetBool("simplificationComplete") {
		t.Fatalf("incomplete experiment: %#v", experiment.Slots)
	}
	if got := len(experiment.GetStrings("candidateUnits")); got != 584 {
		t.Fatalf("candidate count = %d, want 584", got)
	}
	selected := experiment.GetStrings("selectedPrograms")
	if experiment.GetString("selectionStatus") != "co-minimal" || experiment.GetInt("minimumLength") != 2 || len(selected) != 8 {
		t.Fatalf("selection = status %q length %d programs %d", experiment.GetString("selectionStatus"), experiment.GetInt("minimumLength"), len(selected))
	}
	allowedLabels := map[string]bool{"bound-edit-00": true, "bound-edit-02": true}
	allowedTargets := map[string]bool{"bound-edit-04": true, "bound-edit-06": true}
	for _, name := range selected {
		sequence := store.Get(name).GetStrings("semanticSequence")
		if len(sequence) != 2 || !(allowedLabels[sequence[0]] && allowedTargets[sequence[1]] || allowedTargets[sequence[0]] && allowedLabels[sequence[1]]) {
			t.Fatalf("unsafe or incomplete selected sequence %v", sequence)
		}
	}
	caseData, _ := kuberepairfixture.Seed()
	oracleResult, err := kuberepairoracle.Solve(caseData.Public, caseData.Edits, kuberepairoracle.Intent{
		DesiredPods: caseData.Intent.DesiredPods, BackendPort: caseData.Intent.BackendPort,
		ReadinessPorts: caseData.Intent.ReadinessPorts, ProtectedDigest: caseData.Intent.ProtectedDigest,
	}, 3)
	if err != nil {
		t.Fatal(err)
	}
	var oracleSequences []string
	for _, plan := range oracleResult.Plans {
		parts := make([]string, len(plan))
		for index, editIndex := range plan {
			parts[index] = "bound-edit-" + string(rune('0'+editIndex/10)) + string(rune('0'+editIndex%10))
		}
		oracleSequences = append(oracleSequences, strings.Join(parts, "/"))
	}
	var nousSequences []string
	for _, name := range selected {
		nousSequences = append(nousSequences, strings.Join(store.Get(name).GetStrings("semanticSequence"), "/"))
	}
	sort.Strings(oracleSequences)
	sort.Strings(nousSequences)
	if oracleResult.Terminal != "solution" || oracleResult.MinimumLength != 2 || !reflect.DeepEqual(nousSequences, oracleSequences) {
		t.Fatalf("oracle disagreement: terminal=%s minimum=%d nous=%v oracle=%v", oracleResult.Terminal, oracleResult.MinimumLength, nousSequences, oracleSequences)
	}
	handleValue, _ := kuberepair.EncodeHandle(caseData.Handle)
	for _, name := range selected {
		result, err := eng.VM.Execute(`"KubeRepairSeedExample" "input" get-slot "` + name + `" apply-op`)
		if err != nil || result.IsNil() || !kuberepair.EqualOrSatisfies(result.AsString(), handleValue) {
			t.Fatalf("selected program %s failed replay: (%v,%v)", name, result, err)
		}
	}
	for _, primitive := range kubeMembers(store, "KubeAtomicEdit") {
		store.Delete(primitive.Name)
	}
	name := selected[0]
	result, err := eng.VM.Execute(`"KubeRepairSeedExample" "input" get-slot "` + name + `" apply-op`)
	if err != nil || result.IsNil() || !kuberepair.EqualOrSatisfies(result.AsString(), handleValue) {
		t.Fatalf("materialized winner failed after primitive deletion: (%v,%v)", result, err)
	}
}

func TestKubeRepairGenericHeuristicsAreVerbatim(t *testing.T) {
	stackSource, err := os.ReadFile("../../domains/tinystack/heuristics.cue")
	if err != nil {
		t.Fatal(err)
	}
	kubeSource, err := os.ReadFile("../../domains/kuberepair/heuristics.cue")
	if err != nil {
		t.Fatal(err)
	}
	marker := []byte(`name:  "H-CompareProgramSimplifications"`)
	stackPrefix := stackSource[:strings.Index(string(stackSource), string(marker))]
	kubePrefix := kubeSource[:strings.Index(string(kubeSource), string(marker))]
	if !reflect.DeepEqual(stackPrefix, kubePrefix) {
		t.Fatal("generic enumerate/evaluate/select heuristic source differs")
	}
	if strings.Contains(string(kubeSource), "StackProgramSimplifiesToPrimitive") || !strings.Contains(string(kubeSource), "KubeRepairPlanSimplifiesToAtomicEdit") {
		t.Fatal("Kubernetes simplification conjecture contract is not domain-specific")
	}
}

func TestKubeRepairSelectedPlansEmitStructuralCredit(t *testing.T) {
	store, cleanup := loadKubeRepair(t)
	defer cleanup()
	runKubeRepair(t, store, false)
	creditEngine := engine.New(store, agenda.New())
	creditEngine.Out = io.Discard
	creditEngine.VM.Out = io.Discard
	creditEngine.MutConfig.Enabled = false
	creditEngine.MaxCycles = 16
	if err := creditEngine.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	contextName := kuberepair.CreditContext
	for _, subject := range []string{"KubeFeaturePutTemplateFromDeployment", "KubeFeaturePutTemplateFromService", "KubeFeatureServiceTargetName", "KubeFeatureServiceTargetNumber"} {
		if got := credit.RewardTotal(store, credit.Tuple{Context: contextName, Subject: subject, Role: "component"}); got == 0 {
			t.Fatalf("missing structural component credit for %s", subject)
		}
	}
	var structural []string
	for _, record := range kubeMembers(store, credit.Category) {
		if record.GetString("creditRole") == "decision" && strings.HasPrefix(record.GetString("creditSubject"), "sha256:structural:v1:") {
			structural = append(structural, record.GetString("creditSubject"))
		}
	}
	sort.Strings(structural)
	if len(structural) != 8 {
		t.Fatalf("structural decision records = %d, want 8", len(structural))
	}
}
