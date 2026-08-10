package actionrelationutility

import (
	"testing"

	"github.com/chazu/nous/internal/actionrelationsearch"
)

func TestSearchRunVerifierRequiresResolvedCertificateAttempt(t *testing.T) {
	run, err := ExecutePolicy("../../domains", independentUtilityWorld(), actionrelationsearch.DynamicSleep, "development", "authority", 10, 0, 8192, "verify-attempt")
	if err != nil {
		t.Fatal(err)
	}
	attempts := run.Store.Examples("ActionRelationCertificateAttempt")
	if len(attempts) == 0 {
		t.Fatal("certified run did not retain an attempt")
	}
	run.Store.Delete(attempts[0])
	if err := VerifySearchRun(run); err == nil {
		t.Fatal("accepted a cache row whose certificate attempt does not resolve")
	}
}

func TestSearchRunVerifierRejectsForgedOperationRange(t *testing.T) {
	run, err := ExecutePolicy("../../domains", independentUtilityWorld(), actionrelationsearch.StaticSleep, "development", "authority", 11, 0, 4096, "verify-root")
	if err != nil {
		t.Fatal(err)
	}
	if len(run.ProofRoots) == 0 {
		t.Fatal("certified run did not retain a proof range")
	}
	run.ProofRoots[0].Canonical = append([]byte(nil), run.ProofRoots[0].Canonical...)
	run.ProofRoots[0].Canonical[len(run.ProofRoots[0].Canonical)-2] ^= 1
	if err := VerifySearchRun(run); err == nil {
		t.Fatal("accepted a forged certificate operation range")
	}
}
