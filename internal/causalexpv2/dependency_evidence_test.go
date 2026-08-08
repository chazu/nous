package causalexpv2

import (
	"reflect"
	"testing"
)

func TestSourceDependencyMetadataUsesQualifiedGoTypes(t *testing.T) {
	t.Parallel()

	sources := map[string][]byte{
		"internal/causalrun/runner.go": []byte(`package causalrun
import (
	"context"
	"net/url"
)
type Runner struct {
	Public context.Context
	hiddenValue *url.URL
}
func (*Runner) Execute(ctx context.Context) error { return nil }
type Teacher interface {
	Respond(token string) (string, error)
}
`),
		"internal/example/example.go": []byte(`package example
import (
	"context"
	"net/url"
)
type Local struct{}
func Exported(values map[string]*url.URL, callback func(context.Context) error) {}
func (*Local) Method(value *Local) {}
`),
	}

	evidence, err := sourceDependencyMetadata(sources)
	if err != nil {
		t.Fatal(err)
	}
	got := evidence.Files["internal/example/example.go"].Parameters
	wantFunctions := []string{
		"github.com/chazu/nous/internal/example.(*github.com/chazu/nous/internal/example.Local).Method",
		"github.com/chazu/nous/internal/example.Exported",
		"github.com/chazu/nous/internal/example.Exported",
	}
	wantTypes := []string{
		"*github.com/chazu/nous/internal/example.Local",
		"map[string]*net/url.URL",
		"func(context.Context) error",
	}
	if len(got) != len(wantTypes) {
		t.Fatalf("parameters = %#v", got)
	}
	for index := range got {
		if got[index].Function != wantFunctions[index] || got[index].Type != wantTypes[index] {
			t.Fatalf("parameter %d = %#v, want function %q type %q", index, got[index], wantFunctions[index], wantTypes[index])
		}
	}
	if !reflect.DeepEqual(evidence.RunnerMethods, []string{"Executefunc(ctx context.Context) error"}) {
		t.Fatalf("runner methods = %#v", evidence.RunnerMethods)
	}
	if !reflect.DeepEqual(evidence.TeacherMethods, []string{"Respondfunc(token string) (string, error)"}) {
		t.Fatalf("teacher methods = %#v", evidence.TeacherMethods)
	}
	if len(evidence.RunnerFields) != 2 || evidence.RunnerFields[0].Name != "Public" || evidence.RunnerFields[0].Type != "context.Context" || evidence.RunnerFields[0].HiddenBearing || evidence.RunnerFields[1].Name != "hiddenValue" || evidence.RunnerFields[1].Type != "*net/url.URL" || !evidence.RunnerFields[1].HiddenBearing {
		t.Fatalf("runner fields = %#v", evidence.RunnerFields)
	}
}
