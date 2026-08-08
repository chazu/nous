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

func TestApplyRejectsForgedResourceAliases(t *testing.T) {
	bundle := seedBundle()
	public, err := EncodeBundle(bundle)
	if err != nil {
		t.Fatal(err)
	}
	tests := []Edit{
		{Version: EditVersion, Kind: "put-label", Destination: Path{Kind: "template-label", Resource: "forged", Key: "app"}, Source: &Path{Kind: "deployment-label", Resource: bundle.Deployment.Name, Key: "app"}},
		{Version: EditVersion, Kind: "put-label", Destination: Path{Kind: "service-label", Resource: "forged", Key: "app"}, Source: &Path{Kind: "deployment-label", Resource: bundle.Deployment.Name, Key: "app"}},
		{Version: EditVersion, Kind: "set-port-number", Destination: Path{Kind: "service-target", Resource: "forged"}, Source: &Path{Kind: "declared-port", Resource: bundle.Deployment.Name, Container: "server", Port: "web"}},
	}
	for _, edit := range tests {
		encoded := mustEdit(t, edit)
		if _, err := Apply(public, encoded); err == nil {
			t.Fatalf("forged edit was accepted: %s", encoded)
		}
	}
}

func TestProtectedAliasCannotBeRewritten(t *testing.T) {
	bundle := seedBundle()
	bundle.Protected = []string{pathID(Path{Kind: "template-label", Resource: bundle.Deployment.Name, Key: "app"})}
	public, err := EncodeBundle(bundle)
	if err != nil {
		t.Fatal(err)
	}
	edit := Edit{Version: EditVersion, Kind: "put-label", Destination: Path{Kind: "template-label", Resource: "forged", Key: "app"}, Source: &Path{Kind: "deployment-label", Resource: bundle.Deployment.Name, Key: "app"}}
	if _, err := Apply(public, mustEdit(t, edit)); err == nil {
		t.Fatal("forged alias bypassed protected destination")
	}
}

func TestPinnedSubsetRejectsInvalidNamesAndPodWideDuplicatePorts(t *testing.T) {
	bundle := seedBundle()
	bundle.Deployment.Template.Labels[0].Value = "not valid"
	if _, err := EncodeBundle(bundle); err == nil {
		t.Fatal("invalid synthetic label atom was accepted")
	}
	bundle = seedBundle()
	bundle.Deployment.Template.Containers = append(bundle.Deployment.Template.Containers, Container{Name: "sidecar", Ports: []NamedPort{{Name: "web", Number: 8081}}})
	if _, err := EncodeBundle(bundle); err == nil {
		t.Fatal("pod-wide duplicate port name was accepted")
	}
}

func TestLabelEnumerationCoversEveryPublicSourceKey(t *testing.T) {
	bundle := seedBundle()
	bundle.Pods[0].Labels = append(bundle.Pods[0].Labels, Label{Key: "zone", Value: "east"})
	encoded, err := EncodeBundle(bundle)
	if err != nil {
		t.Fatal(err)
	}
	edits, err := EnumerateEdits(encoded)
	if err != nil {
		t.Fatal(err)
	}
	foundTemplate, foundService := false, false
	for _, value := range edits {
		edit, _ := DecodeEdit(value)
		if edit.Kind == "put-label" && edit.Source != nil && edit.Source.Kind == "pod-label" && edit.Source.Key == "zone" {
			foundTemplate = foundTemplate || edit.Destination.Kind == "template-label"
			foundService = foundService || edit.Destination.Kind == "service-label"
		}
	}
	if !foundTemplate || !foundService {
		t.Fatalf("source-only key destinations missing: template=%v service=%v", foundTemplate, foundService)
	}
}

func TestTerminalEvaluationLogCountsActualCapabilityCalls(t *testing.T) {
	bundle := seedBundle()
	public, _ := EncodeBundle(bundle)
	hash := sha256.Sum256([]byte("logged-intent"))
	handle := hex.EncodeToString(hash[:])
	cleanup, err := RegisterIntent(handle, Intent{DesiredPods: []string{"deployment/orbit"}, BackendPort: 8080, ReadinessPorts: map[string]int{"server": 8080}, ProtectedDigest: ProtectedDigest(bundle)})
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	handleValue, _ := EncodeHandle(handle)
	EqualOrSatisfies(public, handleValue)
	EqualOrSatisfies(public, handleValue)
	log := EvaluationLog(handle)
	if len(log) != 2 || log[0] != log[1] {
		t.Fatalf("evaluation log = %#v", log)
	}
}

func TestRemovingLastServiceSelectorRemainsStructurallyValid(t *testing.T) {
	bundle := seedBundle()
	bundle.Protected = nil
	public, _ := EncodeBundle(bundle)
	edit := mustEdit(t, Edit{Version: EditVersion, Kind: "remove-label", Destination: Path{Kind: "service-label", Resource: bundle.Service.Name, Key: "app"}})
	result, err := Apply(public, edit)
	if err != nil || !ValidBundle(result) {
		t.Fatalf("last-selector removal = (%q, %v)", result, err)
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
