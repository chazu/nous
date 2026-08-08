package kuberepairoracle

import (
	"os"
	"strings"
	"testing"
)

func TestOracleSourceHasNoProductionDependency(t *testing.T) {
	data, err := os.ReadFile("oracle.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"internal/vocab/kuberepair", "internal/dsl", "internal/engine", "internal/seed", "internal/credit", "internal/kuberepairexp", "internal/kuberepairfixture"} {
		if strings.Contains(string(data), forbidden) {
			t.Fatalf("oracle imports forbidden package %q", forbidden)
		}
	}
}
