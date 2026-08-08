package kuberepair

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
)

func seedBundle() Bundle {
	bundle := Bundle{
		Namespace: "delta",
		Deployment: Deployment{
			Name:     "orbit",
			Selector: []Label{{Key: "app", Value: "api"}},
			Template: Template{
				Labels: []Label{{Key: "app", Value: "broken"}},
				Containers: []Container{{
					Name:      "server",
					Ports:     []NamedPort{{Name: "health", Number: 9090}, {Name: "web", Number: 8080}},
					Readiness: &Probe{Path: "/ready", Port: PortRef{Kind: "name", Name: "web"}},
				}},
			},
		},
		Service: Service{
			Name:     "gateway",
			Selector: []Label{{Key: "app", Value: "api"}},
			Port:     ServicePort{Name: "https", Port: 443, TargetPort: PortRef{Kind: "name", Name: "stale"}},
		},
		Pods: []Pod{{
			Name:       "other",
			Labels:     []Label{{Key: "app", Value: "decoy"}},
			Containers: []Container{{Name: "other", Ports: []NamedPort{{Name: "web", Number: 7070}}}},
		}},
	}
	bundle.Protected = []string{
		pathID(Path{Kind: "declared-port", Resource: "orbit", Container: "server", Port: "health"}),
		pathID(Path{Kind: "declared-port", Resource: "orbit", Container: "server", Port: "web"}),
		pathID(Path{Kind: "deployment-label", Resource: "orbit", Key: "app"}),
		pathID(Path{Kind: "readiness-port", Resource: "orbit", Container: "server"}),
		pathID(Path{Kind: "service-label", Resource: "gateway", Key: "app"}),
	}
	canonicalizeBundle(&bundle)
	return bundle
}

func TestAtomicEnumerationAndOpaqueIntent(t *testing.T) {
	bundle := seedBundle()
	encoded, err := EncodeBundle(bundle)
	if err != nil {
		t.Fatal(err)
	}
	edits, err := EnumerateEdits(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if len(edits) != 8 {
		for _, edit := range edits {
			t.Log(edit)
		}
		t.Fatalf("edit count = %d, want 8", len(edits))
	}
	var labelEdit, targetEdit string
	for _, encodedEdit := range edits {
		edit, err := DecodeEdit(encodedEdit)
		if err != nil {
			t.Fatal(err)
		}
		if edit.Kind == "put-label" && edit.Source != nil && edit.Source.Kind == "deployment-label" {
			labelEdit = encodedEdit
		}
		if edit.Kind == "set-port-name" && edit.Destination.Kind == "service-target" && edit.Source != nil && edit.Source.Port == "web" {
			targetEdit = encodedEdit
		}
	}
	if labelEdit == "" || targetEdit == "" {
		t.Fatalf("missing intended atomic edits")
	}
	afterLabel, err := Apply(encoded, labelEdit)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Apply(afterLabel, labelEdit); err == nil {
		t.Fatal("duplicate destination write succeeded")
	}
	repaired, err := Apply(afterLabel, targetEdit)
	if err != nil {
		t.Fatal(err)
	}

	hash := sha256.Sum256([]byte("intent-alpha"))
	handle := hex.EncodeToString(hash[:])
	intent := Intent{
		DesiredPods:     []string{"deployment/orbit"},
		BackendPort:     8080,
		ReadinessPorts:  map[string]int{"server": 8080},
		ProtectedDigest: ProtectedDigest(bundle),
	}
	cleanup, err := RegisterIntent(handle, intent)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	handleValue, err := EncodeHandle(handle)
	if err != nil {
		t.Fatal(err)
	}
	if EqualOrSatisfies(encoded, handleValue) {
		t.Fatal("unrepaired bundle satisfied private intent")
	}
	if !EqualOrSatisfies(repaired, handleValue) {
		t.Fatal("atomic repair did not satisfy private intent")
	}
	if !EqualOrSatisfies(repaired, repaired) || EqualOrSatisfies(handleValue, repaired) {
		t.Fatal("asymmetric comparator contract failed")
	}
}

func TestStrictCanonicalAndFeatureIdentity(t *testing.T) {
	encoded, err := EncodeBundle(seedBundle())
	if err != nil {
		t.Fatal(err)
	}
	if !ValidBundle(encoded) || ValidBundle(encoded+" ") || ValidValue(`{"version":"unknown"}`) {
		t.Fatal("strict value validation failed")
	}
	edits, err := EnumerateEdits(encoded)
	if err != nil {
		t.Fatal(err)
	}
	var chosen Edit
	for _, value := range edits {
		edit, _ := DecodeEdit(value)
		if edit.Kind == "set-port-number" {
			chosen = edit
			break
		}
	}
	first, relation, err := FeatureKey(mustEdit(t, chosen))
	if err != nil {
		t.Fatal(err)
	}
	chosen.Destination.Resource = "renamed"
	chosen.Source.Resource = "renamed"
	chosen.Source.Container = "worker"
	chosen.Source.Port = "backend"
	second, secondRelation, err := FeatureKey(mustEdit(t, chosen))
	if err != nil {
		t.Fatal(err)
	}
	if first != second || relation != secondRelation || !strings.Contains(first, "service-target") {
		t.Fatalf("feature keys leaked aliases: %q/%q versus %q/%q", first, relation, second, secondRelation)
	}
}

func mustEdit(t *testing.T, edit Edit) string {
	t.Helper()
	value, err := EncodeEdit(edit)
	if err != nil {
		t.Fatal(err)
	}
	return value
}
