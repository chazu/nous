package transformoracle_test

import (
	"bytes"
	"testing"

	"github.com/chazu/nous/internal/transformoracle"
	"github.com/chazu/nous/internal/vocab/transformschema"
)

func TestOracleAgreesOnAllSchemas(t *testing.T) {
	f := transformschema.Forest{Nodes: []transformschema.Node{
		{ID: 0, Kind: "group", Parent: -1, Target: -1},
		{ID: 1, Kind: "definition", Parent: 0, Key: "service", Value: "old", Target: -1},
		{ID: 2, Kind: "request", Parent: 0, Key: "change", From: "old", To: "new", Target: 1},
		{ID: 3, Kind: "reference", Parent: 0, Key: "client", Value: "old", Target: 1},
	}}
	fb, _ := f.CanonicalJSON()
	for _, schema := range transformschema.Schemas() {
		sb, _ := schema.CanonicalJSON()
		production, productionErr := schema.Apply(f)
		oracle, oracleErr := transformoracle.Apply(fb, sb)
		if (productionErr != nil) != (oracleErr != nil) || production.Terminal != oracle.Terminal {
			t.Fatalf("schema=%s production=%s/%v oracle=%s/%v", sb, production.Terminal, productionErr, oracle.Terminal, oracleErr)
		}
		if production.Output != nil {
			pb, _ := production.Output.CanonicalJSON()
			if !bytes.Equal(pb, oracle.Output) {
				t.Fatalf("schema=%s output differs", sb)
			}
		}
	}
}
