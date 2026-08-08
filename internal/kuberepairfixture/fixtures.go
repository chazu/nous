// Package kuberepairfixture generates deterministic synthetic tasks. It owns
// public fixtures and private intents but does not rank or solve repairs.
package kuberepairfixture

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"

	kuberepair "github.com/chazu/nous/internal/vocab/kuberepair"
)

type Case struct {
	ID     string
	Public string
	Handle string
	Intent kuberepair.Intent
	Edits  []string
}

const (
	FaultTemplate = 1 << iota
	FaultService
	FaultExtraSelector
)

// Training returns three tasks whose unique minimum edits expose the component
// features later used in recomposition.
func Training(seed int64) ([]Case, error) {
	return buildCases(seed, []int{FaultTemplate, FaultService, FaultExtraSelector})
}

// Recomposition returns a deterministic task with the requested fault mask.
func Recomposition(seed int64, mask int) (Case, error) {
	cases, err := buildCases(seed, []int{mask})
	if err != nil {
		return Case{}, err
	}
	return cases[0], nil
}

// CrossRole requires a label-copy relation at an uncredited destination role.
func CrossRole(seed int64) (Case, error) {
	suffix := fmt.Sprintf("x%d", seed%100000)
	key := "app" + suffix
	value := "api" + suffix
	deployment := "work" + suffix
	service := "front" + suffix
	container := "main" + suffix
	backend := 9000 + int(seed%500)
	bundle := kuberepair.Bundle{Namespace: "ns" + suffix, Deployment: kuberepair.Deployment{Name: deployment, Selector: []kuberepair.Label{{Key: key, Value: value}}, Template: kuberepair.Template{Labels: []kuberepair.Label{{Key: key, Value: value}}, Containers: []kuberepair.Container{{Name: container, Ports: []kuberepair.NamedPort{{Name: "http" + suffix, Number: backend}}}}}}, Service: kuberepair.Service{Name: service, Selector: []kuberepair.Label{{Key: key, Value: "bad" + suffix}}, Port: kuberepair.ServicePort{Name: "web" + suffix, Port: backend, TargetPort: kuberepair.PortRef{Kind: "number", Number: backend}}}, Pods: []kuberepair.Pod{{Name: "other" + suffix, Labels: []kuberepair.Label{{Key: key, Value: "decoy" + suffix}}, Containers: []kuberepair.Container{{Name: "other" + suffix, Ports: []kuberepair.NamedPort{{Name: "http" + suffix, Number: backend + 1}}}}}}}
	bundle.Protected = []string{path(kuberepair.Path{Kind: "declared-port", Resource: deployment, Container: container, Port: "http" + suffix}), path(kuberepair.Path{Kind: "service-target", Resource: service}), path(kuberepair.Path{Kind: "template-label", Resource: deployment, Key: key})}
	sort.Strings(bundle.Protected)
	return finishCase(fmt.Sprintf("cross-%d", seed), bundle, []string{"deployment/" + deployment}, backend)
}

// Unrelated requires a reference feature absent from the label-only training
// curriculum. When noSolution is true, private intent names no reachable port.
func Unrelated(seed int64, noSolution bool) (Case, error) {
	suffix := fmt.Sprintf("x%d", seed%100000)
	key := "app" + suffix
	value := "api" + suffix
	deployment := "work" + suffix
	service := "front" + suffix
	container := "main" + suffix
	backend := 10000 + int(seed%500)
	bundle := kuberepair.Bundle{Namespace: "ns" + suffix, Deployment: kuberepair.Deployment{Name: deployment, Selector: []kuberepair.Label{{Key: key, Value: value}}, Template: kuberepair.Template{Labels: []kuberepair.Label{{Key: key, Value: value}}, Containers: []kuberepair.Container{{Name: container, Ports: []kuberepair.NamedPort{{Name: "health" + suffix, Number: backend + 1}, {Name: "web" + suffix, Number: backend}}}}}}, Service: kuberepair.Service{Name: service, Selector: []kuberepair.Label{{Key: key, Value: value}}, Port: kuberepair.ServicePort{Name: "https" + suffix, Port: 443, TargetPort: kuberepair.PortRef{Kind: "name", Name: "stale" + suffix}}}}
	bundle.Protected = []string{path(kuberepair.Path{Kind: "deployment-label", Resource: deployment, Key: key}), path(kuberepair.Path{Kind: "service-label", Resource: service, Key: key}), path(kuberepair.Path{Kind: "template-label", Resource: deployment, Key: key})}
	sort.Strings(bundle.Protected)
	want := backend
	if noSolution {
		want = 65535
	}
	label := "unrelated"
	if noSolution {
		label = "no-solution"
	}
	return finishCase(fmt.Sprintf("%s-%d", label, seed), bundle, []string{"deployment/" + deployment}, want)
}

func finishCase(id string, bundle kuberepair.Bundle, desired []string, backend int) (Case, error) {
	public, err := kuberepair.EncodeBundle(bundle)
	if err != nil {
		return Case{}, err
	}
	edits, err := kuberepair.EnumerateEdits(public)
	if err != nil {
		return Case{}, err
	}
	if len(edits) > 8 {
		return Case{}, fmt.Errorf("case %s edit count = %d", id, len(edits))
	}
	sum := sha256.Sum256([]byte("kuberepair-intent-v1|" + id))
	handle := hex.EncodeToString(sum[:])
	intent := kuberepair.Intent{DesiredPods: desired, BackendPort: backend, ReadinessPorts: map[string]int{}, ProtectedDigest: kuberepair.ProtectedDigest(bundle)}
	return Case{ID: id, Public: public, Handle: handle, Intent: intent, Edits: edits}, nil
}

func buildCases(seed int64, masks []int) ([]Case, error) {
	out := make([]Case, 0, len(masks))
	for index, mask := range masks {
		caseData, err := labelCase(seed+int64(index), mask)
		if err != nil {
			return nil, err
		}
		out = append(out, caseData)
	}
	return out, nil
}

func labelCase(seed int64, mask int) (Case, error) {
	suffix := fmt.Sprintf("x%d", seed%100000)
	appKey, tierKey, zoneKey := "app"+suffix, "tier"+suffix, "zone"+suffix
	appValue, tierValue := "api"+suffix, "front"+suffix
	templateApp := appValue
	if mask&FaultTemplate != 0 {
		templateApp = "badapp" + suffix
	}
	serviceTier := tierValue
	if mask&FaultService != 0 {
		serviceTier = "badtier" + suffix
	}
	deploymentName, serviceName, containerName := "work"+suffix, "front"+suffix, "main"+suffix
	labels := []kuberepair.Label{{Key: appKey, Value: templateApp}, {Key: tierKey, Value: tierValue}}
	serviceLabels := []kuberepair.Label{{Key: tierKey, Value: serviceTier}}
	if mask&FaultExtraSelector != 0 {
		serviceLabels = append(serviceLabels, kuberepair.Label{Key: zoneKey, Value: "badzone" + suffix})
	}
	backend := 8000 + int(seed%1000)
	bundle := kuberepair.Bundle{
		Namespace:  "ns" + suffix,
		Deployment: kuberepair.Deployment{Name: deploymentName, Selector: []kuberepair.Label{{Key: appKey, Value: appValue}}, Template: kuberepair.Template{Labels: labels, Containers: []kuberepair.Container{{Name: containerName, Ports: []kuberepair.NamedPort{{Name: "http" + suffix, Number: backend}}}}}},
		Service:    kuberepair.Service{Name: serviceName, Selector: serviceLabels, Port: kuberepair.ServicePort{Name: "web" + suffix, Port: backend, TargetPort: kuberepair.PortRef{Kind: "number", Number: backend}}},
		Pods:       []kuberepair.Pod{{Name: "other" + suffix, Labels: []kuberepair.Label{{Key: appKey, Value: "decoy" + suffix}, {Key: tierKey, Value: "other" + suffix}, {Key: zoneKey, Value: "otherzone" + suffix}}, Containers: []kuberepair.Container{{Name: "other" + suffix, Ports: []kuberepair.NamedPort{{Name: "http" + suffix, Number: backend + 1}}}}}},
	}
	bundle.Protected = []string{
		path(kuberepair.Path{Kind: "declared-port", Resource: deploymentName, Container: containerName, Port: "http" + suffix}),
		path(kuberepair.Path{Kind: "service-label", Resource: serviceName, Key: appKey}),
		path(kuberepair.Path{Kind: "service-target", Resource: serviceName}),
		path(kuberepair.Path{Kind: "template-label", Resource: deploymentName, Key: tierKey}),
	}
	if mask&FaultExtraSelector != 0 {
		// Protect the absent template zone leaf so the one-edit curriculum has a
		// unique removal repair rather than an equivalent selector-expansion edit.
		bundle.Protected = append(bundle.Protected,
			path(kuberepair.Path{Kind: "template-label", Resource: deploymentName, Key: zoneKey}))
	}
	if mask == FaultService {
		bundle.Protected = append(bundle.Protected,
			path(kuberepair.Path{Kind: "template-label", Resource: deploymentName, Key: appKey}))
	}
	sort.Strings(bundle.Protected)
	public, err := kuberepair.EncodeBundle(bundle)
	if err != nil {
		return Case{}, err
	}
	edits, err := kuberepair.EnumerateEdits(public)
	if err != nil {
		return Case{}, err
	}
	if len(edits) < 3 || len(edits) > 8 {
		return Case{}, fmt.Errorf("case %d edit count = %d", mask, len(edits))
	}
	id := fmt.Sprintf("label-%d-%d", seed, mask)
	sum := sha256.Sum256([]byte("kuberepair-intent-v1|" + id))
	handle := hex.EncodeToString(sum[:])
	intent := kuberepair.Intent{DesiredPods: []string{"deployment/" + deploymentName}, BackendPort: backend, ReadinessPorts: map[string]int{}, ProtectedDigest: kuberepair.ProtectedDigest(bundle)}
	return Case{ID: id, Public: public, Handle: handle, Intent: intent, Edits: edits}, nil
}

func Seed() (Case, error) {
	bundle := kuberepair.Bundle{
		Namespace: "delta",
		Deployment: kuberepair.Deployment{
			Name:     "orbit",
			Selector: []kuberepair.Label{{Key: "app", Value: "api"}},
			Template: kuberepair.Template{
				Labels: []kuberepair.Label{{Key: "app", Value: "broken"}},
				Containers: []kuberepair.Container{{
					Name:      "server",
					Ports:     []kuberepair.NamedPort{{Name: "health", Number: 9090}, {Name: "web", Number: 8080}},
					Readiness: &kuberepair.Probe{Path: "/ready", Port: kuberepair.PortRef{Kind: "name", Name: "web"}},
				}},
			},
		},
		Service: kuberepair.Service{
			Name:     "gateway",
			Selector: []kuberepair.Label{{Key: "app", Value: "api"}},
			Port:     kuberepair.ServicePort{Name: "https", Port: 443, TargetPort: kuberepair.PortRef{Kind: "name", Name: "stale"}},
		},
		Pods: []kuberepair.Pod{{
			Name:       "other",
			Labels:     []kuberepair.Label{{Key: "app", Value: "decoy"}},
			Containers: []kuberepair.Container{{Name: "other", Ports: []kuberepair.NamedPort{{Name: "web", Number: 7070}}}},
		}},
	}
	bundle.Protected = []string{
		path(kuberepair.Path{Kind: "declared-port", Resource: "orbit", Container: "server", Port: "health"}),
		path(kuberepair.Path{Kind: "declared-port", Resource: "orbit", Container: "server", Port: "web"}),
		path(kuberepair.Path{Kind: "deployment-label", Resource: "orbit", Key: "app"}),
		path(kuberepair.Path{Kind: "readiness-port", Resource: "orbit", Container: "server"}),
		path(kuberepair.Path{Kind: "service-label", Resource: "gateway", Key: "app"}),
	}
	sort.Strings(bundle.Protected)
	public, err := kuberepair.EncodeBundle(bundle)
	if err != nil {
		return Case{}, err
	}
	edits, err := kuberepair.EnumerateEdits(public)
	if err != nil {
		return Case{}, err
	}
	if len(edits) != 8 {
		return Case{}, fmt.Errorf("seed edit count = %d, want 8", len(edits))
	}
	sum := sha256.Sum256([]byte("kuberepair-seed-intent-v1"))
	handle := hex.EncodeToString(sum[:])
	intent := kuberepair.Intent{
		DesiredPods:     []string{"deployment/orbit"},
		BackendPort:     8080,
		ReadinessPorts:  map[string]int{"server": 8080},
		ProtectedDigest: kuberepair.ProtectedDigest(bundle),
	}
	return Case{ID: "seed", Public: public, Handle: handle, Intent: intent, Edits: edits}, nil
}

func path(value kuberepair.Path) string {
	encoded, _ := json.Marshal(value)
	return string(encoded)
}
