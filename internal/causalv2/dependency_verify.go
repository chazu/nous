package causalv2

import (
	"fmt"
	"sort"
)

// VerifyDependencyProof exposes the existing dependency-proof acceptance
// predicate with field-specific canonicality diagnostics. It deliberately
// delegates final acceptance to validateDependencyProof.
func VerifyDependencyProof(proof DependencyProof) error {
	if err := diagnoseDependencyArrays(proof); err != nil {
		return err
	}
	return validateDependencyProof(proof)
}

func diagnoseDependencyArrays(proof DependencyProof) error {
	if err := diagnoseStrings("audited_roots", proof.AuditedRoots); err != nil {
		return err
	}
	if proof.Files == nil {
		return fmt.Errorf("dependency array files is nil at index 0")
	}
	for index, file := range proof.Files {
		if index > 0 && proof.Files[index-1].Path >= file.Path {
			return fmt.Errorf("dependency array files is not strictly sorted at index %d: %q >= %q", index, proof.Files[index-1].Path, file.Path)
		}
		if err := diagnoseStrings(fmt.Sprintf("files[%d].imports", index), file.Imports); err != nil {
			return err
		}
		if file.ExportedFunctionParameters == nil {
			return fmt.Errorf("dependency array files[%d].exported_function_parameters is nil at index 0", index)
		}
		for parameterIndex := 1; parameterIndex < len(file.ExportedFunctionParameters); parameterIndex++ {
			previous := file.ExportedFunctionParameters[parameterIndex-1]
			current := file.ExportedFunctionParameters[parameterIndex]
			if previous.Function > current.Function || previous.Function == current.Function && previous.ParameterIndex >= current.ParameterIndex {
				return fmt.Errorf("dependency array files[%d].exported_function_parameters is not strictly sorted at index %d: %q[%d] >= %q[%d]", index, parameterIndex, previous.Function, previous.ParameterIndex, current.Function, current.ParameterIndex)
			}
		}
	}
	if err := diagnoseStrings("runner_methods", proof.RunnerMethods); err != nil {
		return err
	}
	if proof.RunnerFields == nil {
		return fmt.Errorf("dependency array runner_fields is nil at index 0")
	}
	for index := 1; index < len(proof.RunnerFields); index++ {
		if proof.RunnerFields[index-1].Name >= proof.RunnerFields[index].Name {
			return fmt.Errorf("dependency array runner_fields is not strictly sorted at index %d: %q >= %q", index, proof.RunnerFields[index-1].Name, proof.RunnerFields[index].Name)
		}
	}
	if err := diagnoseStrings("teacher_methods", proof.TeacherMethods); err != nil {
		return err
	}
	return diagnoseStrings("forbidden", proof.Forbidden)
}

func diagnoseStrings(name string, values []string) error {
	if values == nil {
		return fmt.Errorf("dependency array %s is nil at index 0", name)
	}
	if index := firstUnsortedString(values); index >= 0 {
		return fmt.Errorf("dependency array %s is not sorted at index %d: %q > %q", name, index, values[index-1], values[index])
	}
	return nil
}

func firstUnsortedString(values []string) int {
	if sort.StringsAreSorted(values) {
		return -1
	}
	for index := 1; index < len(values); index++ {
		if values[index-1] > values[index] {
			return index
		}
	}
	return -1
}
