package dsl

import (
	"testing"

	"github.com/chazu/nous/internal/agenda"
	"github.com/chazu/nous/internal/unit"
	kuberepair "github.com/chazu/nous/internal/vocab/kuberepair"
)

func kubeRepairTestStore() *unit.Store {
	store := protocolTestStore("")
	marker := unit.New("KubeRepairVocabulary")
	marker.Set("isA", []string{"Vocabulary", "Anything"})
	marker.Set("dslExtension", "kuberepair")
	store.Put(marker)
	return store
}

func TestKubeRepairWordsAreScopedAndStrict(t *testing.T) {
	base := NewVM(protocolTestStore(""), agenda.New(), nil)
	if _, err := base.Execute(`"x" kube-bundle-valid?`); err == nil {
		t.Fatal("kuberepair word leaked into base VM")
	}
	bundle := kuberepair.Bundle{
		Namespace:  "alpha",
		Deployment: kuberepair.Deployment{Name: "work", Selector: []kuberepair.Label{{Key: "app", Value: "a"}}, Template: kuberepair.Template{Labels: []kuberepair.Label{{Key: "app", Value: "b"}}, Containers: []kuberepair.Container{{Name: "main", Ports: []kuberepair.NamedPort{{Name: "http", Number: 8080}}}}}},
		Service:    kuberepair.Service{Name: "front", Selector: []kuberepair.Label{{Key: "app", Value: "a"}}, Port: kuberepair.ServicePort{Name: "web", Port: 80, TargetPort: kuberepair.PortRef{Kind: "name", Name: "stale"}}},
	}
	encoded, err := kuberepair.EncodeBundle(bundle)
	if err != nil {
		t.Fatal(err)
	}
	if !kuberepair.ValidBundle(encoded) {
		t.Fatalf("encoded fixture is not valid: %s", encoded)
	}
	vm := NewVM(kubeRepairTestStore(), agenda.New(), nil)
	vm.SetEnv("bundle", StringVal(encoded))
	value, err := vm.Execute(`"bundle" @ kube-bundle-valid?`)
	if err != nil || value.Kind() != VBool || !value.AsBool() {
		t.Fatalf("bundle validation = (%v,%v)", value, err)
	}
	list, err := vm.Execute(`"bundle" @ kube-enumerate-edits`)
	if err != nil || list.Kind() != VList || len(list.AsList()) == 0 {
		t.Fatalf("edit enumeration = (%v,%v)", list, err)
	}
	edit := list.AsList()[0].AsString()
	vm.SetEnv("edit", StringVal(edit))
	result, err := vm.Execute(`"bundle" @ "edit" @ kube-apply-edit`)
	if err != nil || result.Kind() != VString || !kuberepair.ValidBundle(result.AsString()) {
		t.Fatalf("edit application = (%v,%v)", result, err)
	}
	if value, err := vm.Execute(`1 kube-repair-value-valid?`); err != nil || value.Kind() != VBool || value.AsBool() {
		t.Fatalf("wrong-kind validation = (%v,%v)", value, err)
	}
}
