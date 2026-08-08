package ruleinductionexp

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestManifestRoundTripsWithoutOmittedFields(t *testing.T) {
	manifest := PreregisteredManifest()
	encoded, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	var decoded Manifest
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(decoded, manifest) {
		t.Fatal("manifest did not round trip")
	}
}
