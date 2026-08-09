package nogoodexp

import (
	"bytes"
	"encoding/json"
	"slices"
	"testing"

	"github.com/chazu/nous/internal/nogoodfixture"
)

func TestFixtureBundleRoundTripsExactPublicInputs(t *testing.T) {
	tasks, err := nogoodfixture.DevelopmentPanel()
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := encodeFixtureBundle("development", tasks)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := decodeFixtureBundle("development", encoded)
	if err != nil {
		t.Fatal(err)
	}
	if len(decoded) != DevelopmentTaskCount {
		t.Fatalf("decoded tasks = %d", len(decoded))
	}
	for index := range tasks {
		if decoded[index].Ordinal != index || decoded[index].Cohort != tasks[index].Cohort || decoded[index].Template != tasks[index].Template || decoded[index].MissingBit != tasks[index].MissingBit || !bytes.Equal(decoded[index].ProblemJSON, tasks[index].ProblemJSON) || decoded[index].Decision != tasks[index].Decision {
			t.Fatalf("fixture %d did not round trip", index)
		}
	}
	reencoded, err := encodeFixtureBundle("development", decoded)
	if err != nil || !bytes.Equal(reencoded, encoded) {
		t.Fatalf("fixture bundle did not byte-round-trip: %v", err)
	}
}

func TestFixtureBundleRejectsNoncanonicalWireForms(t *testing.T) {
	tasks, err := nogoodfixture.DevelopmentPanel()
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := encodeFixtureBundle("development", tasks)
	if err != nil {
		t.Fatal(err)
	}
	var records []fixtureRecord
	if err := json.Unmarshal(encoded, &records); err != nil {
		t.Fatal(err)
	}
	padded := slices.Clone(records)
	padded[0].Problem += "="
	paddedBytes, _ := json.Marshal(padded)
	tests := map[string][]byte{
		"whitespace":    append([]byte(" "), encoded...),
		"duplicate-key": bytes.Replace(encoded, []byte(`{"ordinal":0`), []byte(`{"ordinal":0,"ordinal":0`), 1),
		"unknown-key":   bytes.Replace(encoded, []byte(`,"problem":`), []byte(`,"unknown":0,"problem":`), 1),
		"padded-base64": paddedBytes,
		"wrong-length":  []byte("[]"),
	}
	for name, input := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := decodeFixtureBundle("development", input); err == nil {
				t.Fatal("accepted noncanonical fixture bundle")
			}
		})
	}
}
