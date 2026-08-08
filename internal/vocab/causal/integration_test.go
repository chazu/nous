package causal_test

import (
	"context"
	"io"
	"testing"

	"github.com/chazu/nous/internal/agenda"
	"github.com/chazu/nous/internal/causaloracle"
	"github.com/chazu/nous/internal/engine"
	"github.com/chazu/nous/internal/seed"
	"github.com/chazu/nous/internal/unit"
	causal "github.com/chazu/nous/internal/vocab/causal"
)

func TestCUEProposalAndTeacherResume(t *testing.T) {
	store := unit.NewStore()
	seed.DomainsDir = "../../../domains"
	if err := seed.LoadDomain(store, "causal"); err != nil {
		t.Fatal(err)
	}
	universe := causal.Enumerate()
	var posterior []string
	for _, h := range universe[:12] {
		code, _ := causal.Code(h)
		posterior = append(posterior, code)
	}
	d := unit.New("Causal.Test")
	d.Set("isA", []string{"CausalExperiment", "Anything"})
	d.Set("profileVersion", causal.ProfileVersion)
	d.Set("experimentVersion", causal.ExperimentV1)
	d.Set("profileDigest", "test-profile")
	d.Set("posterior", posterior)
	d.Set("initialPosterior", posterior)
	d.Set("acquisitionCode", "P=E;M=raw;S=W")
	d.Set("cost0", 5)
	d.Set("cost1", 20)
	d.Set("cost2", 40)
	d.Set("state", "ready")
	d.Set("proposeTaskSlot", "causalPropose")
	d.Set("updateTaskSlot", "causalUpdate")
	d.Set("actionCount", 0)
	d.Set("step", 0)
	d.Set("totalCost", 0)
	d.Set("consumedActions", []string{})
	d.Set("transcriptDigest", "")
	store.Put(d)
	ag := agenda.New()
	ag.Push(&agenda.Task{Priority: 900, UnitName: d.Name, SlotName: "causalPropose"})
	eng := engine.New(store, ag)
	eng.MaxCycles = 500
	eng.Verbosity = 0
	eng.Out = io.Discard
	eng.MutConfig.Enabled = false
	if err := eng.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := d.GetString("state"); got != "awaiting-teacher" {
		t.Fatalf("state=%q selected=%q", got, d.GetString("selectedAction"))
	}
	if len(store.Examples("CausalProposal"))-1 != 6 {
		t.Fatalf("proposals=%d", len(store.Examples("CausalProposal"))-1)
	}
	hidden := posterior[0]
	teacher, err := causaloracle.NewTeacher("token", hidden)
	if err != nil {
		t.Fatal(err)
	}
	outcome, err := teacher.Respond("token", d.GetString("selectedAction"))
	if err != nil {
		t.Fatal(err)
	}
	response := unit.New("Causal.Response.Test")
	response.Set("isA", []string{"CausalTeacherResult", "Anything"})
	response.Set("outcome", outcome)
	response.Set("action", d.GetString("selectedAction"))
	store.Put(response)
	d.Set("responseUnit", response.Name)
	d.Set("state", "response-present")
	ag.Push(&agenda.Task{Priority: 900, UnitName: d.Name, SlotName: "causalUpdate"})
	if err := eng.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if d.GetInt("actionCount") != 1 {
		t.Fatalf("actions=%d state=%s", d.GetInt("actionCount"), d.GetString("state"))
	}
	if len(d.GetStrings("posterior")) >= len(posterior) {
		t.Fatalf("posterior did not shrink: %d", len(d.GetStrings("posterior")))
	}
}
