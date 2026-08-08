package causalv2

import (
	"strings"
	"testing"
)

func validDependencyProofForTest() DependencyProof {
	return DependencyProof{
		AuditedCommit: strings.Repeat("a", 40),
		AuditedRoots:  []string{"."},
		Files: []DependencyFile{{
			Path:                       "example.go",
			SourceSHA256:               strings.Repeat("b", 64),
			Imports:                    []string{},
			ExportedFunctionParameters: []DependencyParameter{},
		}},
		RunnerMethods:  []string{},
		RunnerFields:   []RunnerField{},
		TeacherMethods: []string{},
		Lookups:        1,
		Forbidden:      []string{},
	}
}

func TestVerifyDependencyProofUsesExistingPredicate(t *testing.T) {
	proof := validDependencyProofForTest()
	if err := VerifyDependencyProof(proof); err != nil {
		t.Fatalf("valid dependency proof: %v", err)
	}
	proof.Forbidden = nil
	if got := VerifyDependencyProof(proof); got == nil || got.Error() != "dependency array forbidden is nil at index 0" {
		t.Fatalf("nil forbidden diagnostic = %v", got)
	}
	proof = validDependencyProofForTest()
	proof.RunnerMethods = []string{"z", "a"}
	if got := VerifyDependencyProof(proof); got == nil || got.Error() != `dependency array runner_methods is not sorted at index 1: "z" > "a"` {
		t.Fatalf("unsorted runner-method diagnostic = %v", got)
	}
}
