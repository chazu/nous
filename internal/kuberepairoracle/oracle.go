// Package kuberepairoracle is a source-independent exhaustive evaluator for the
// bounded Kubernetes repair experiment. It intentionally imports no production
// vocabulary, DSL, engine, seed, credit, fixture, or trial package.
package kuberepairoracle

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

type Label struct{ Key, Value string }
type PortRef struct {
	Kind   string `json:"kind"`
	Name   string `json:"name,omitempty"`
	Number int    `json:"number,omitempty"`
}
type NamedPort struct {
	Name   string `json:"name"`
	Number int    `json:"number"`
}
type Probe struct {
	Path string  `json:"path"`
	Port PortRef `json:"port"`
}
type Container struct {
	Name      string      `json:"name"`
	Ports     []NamedPort `json:"ports"`
	Readiness *Probe      `json:"readiness,omitempty"`
}
type Template struct {
	Labels     []Label     `json:"labels"`
	Containers []Container `json:"containers"`
}
type Deployment struct {
	Name     string   `json:"name"`
	Selector []Label  `json:"selector"`
	Template Template `json:"template"`
}
type ServicePort struct {
	Name       string  `json:"name"`
	Port       int     `json:"port"`
	TargetPort PortRef `json:"targetPort"`
}
type Service struct {
	Name     string      `json:"name"`
	Selector []Label     `json:"selector"`
	Port     ServicePort `json:"servicePort"`
}
type Pod struct {
	Name       string      `json:"name"`
	Labels     []Label     `json:"labels"`
	Containers []Container `json:"containers"`
}
type Bundle struct {
	Namespace  string     `json:"namespace"`
	Deployment Deployment `json:"deployment"`
	Service    Service    `json:"service"`
	Pods       []Pod      `json:"pods"`
	Protected  []string   `json:"protected"`
	Writes     []string   `json:"writes,omitempty"`
}
type envelope struct {
	Version string `json:"version"`
	Bundle  Bundle `json:"bundle"`
}
type Path struct {
	Kind      string `json:"kind"`
	Resource  string `json:"resource"`
	Container string `json:"container,omitempty"`
	Port      string `json:"port,omitempty"`
	Key       string `json:"key,omitempty"`
}
type Edit struct {
	Version     string `json:"version"`
	Kind        string `json:"kind"`
	Destination Path   `json:"destination"`
	Source      *Path  `json:"source,omitempty"`
}

type Intent struct {
	DesiredPods     []string
	BackendPort     int
	ReadinessPorts  map[string]int
	ProtectedDigest string
}

type Result struct {
	Terminal      string
	MinimumLength int
	Plans         [][]int
}

func Solve(public string, encodedEdits []string, intent Intent, maxLength int) (Result, error) {
	var wrapper envelope
	if err := json.Unmarshal([]byte(public), &wrapper); err != nil || wrapper.Version != "kubernetes-bundle/v1" {
		return Result{}, fmt.Errorf("decode public bundle")
	}
	edits := make([]Edit, len(encodedEdits))
	for i, value := range encodedEdits {
		if json.Unmarshal([]byte(value), &edits[i]) != nil {
			return Result{}, fmt.Errorf("decode edit %d", i)
		}
	}
	if satisfies(wrapper.Bundle, intent) {
		return Result{Terminal: "already-correct", MinimumLength: 0, Plans: [][]int{{}}}, nil
	}
	result := Result{Terminal: "no-solution", MinimumLength: maxLength + 1}
	var walk func(Bundle, []int, int)
	walk = func(state Bundle, sequence []int, remaining int) {
		if remaining == 0 {
			return
		}
		for index, edit := range edits {
			next, ok := apply(state, edit)
			if !ok {
				continue
			}
			candidate := append(append([]int(nil), sequence...), index)
			if satisfies(next, intent) {
				if len(candidate) < result.MinimumLength {
					result.MinimumLength = len(candidate)
					result.Plans = nil
				}
				if len(candidate) == result.MinimumLength {
					result.Plans = append(result.Plans, candidate)
				}
				continue
			}
			if len(candidate) < result.MinimumLength {
				walk(next, candidate, remaining-1)
			}
		}
	}
	walk(wrapper.Bundle, nil, maxLength)
	if len(result.Plans) > 0 {
		result.Terminal = "solution"
		sort.Slice(result.Plans, func(i, j int) bool { return sequenceKey(result.Plans[i]) < sequenceKey(result.Plans[j]) })
	}
	return result, nil
}

func apply(bundle Bundle, edit Edit) (Bundle, bool) {
	bundle = clone(bundle)
	destination := pathID(edit.Destination)
	if contains(bundle.Protected, destination) || contains(bundle.Writes, destination) {
		return Bundle{}, false
	}
	changed := false
	switch edit.Kind {
	case "put-label":
		value, ok := getLabel(bundle, *edit.Source)
		if !ok {
			return Bundle{}, false
		}
		changed = setLabel(&bundle, edit.Destination, value)
	case "remove-label":
		changed = removeLabel(&bundle.Service.Selector, edit.Destination.Key)
	case "set-port-name", "set-port-number":
		port, ok := getPort(bundle, *edit.Source)
		if !ok {
			return Bundle{}, false
		}
		ref := PortRef{Kind: "name", Name: port.Name}
		if edit.Kind == "set-port-number" {
			ref = PortRef{Kind: "number", Number: port.Number}
		}
		changed = setRef(&bundle, edit.Destination, ref)
	case "unset-service-target":
		changed = setRef(&bundle, edit.Destination, PortRef{Kind: "default"})
	default:
		return Bundle{}, false
	}
	if !changed {
		return Bundle{}, false
	}
	bundle.Writes = append(bundle.Writes, destination)
	sort.Strings(bundle.Writes)
	return bundle, true
}

func satisfies(bundle Bundle, intent Intent) bool {
	if digest(bundle) != intent.ProtectedDigest || !labelsMatch(bundle.Deployment.Selector, bundle.Deployment.Template.Labels) {
		return false
	}
	selected := selected(bundle)
	if !equalStrings(selected, intent.DesiredPods) {
		return false
	}
	backend, ok := backend(bundle, selected)
	if !ok || backend != intent.BackendPort {
		return false
	}
	seen := 0
	for _, container := range bundle.Deployment.Template.Containers {
		if container.Readiness != nil {
			seen++
			number, ok := resolveContainer(container, container.Readiness.Port)
			if !ok || intent.ReadinessPorts[container.Name] != number {
				return false
			}
		}
	}
	return seen == len(intent.ReadinessPorts)
}

func labelsMatch(selector, labels []Label) bool {
	for _, want := range selector {
		got, ok := label(labels, want.Key)
		if !ok || got != want.Value {
			return false
		}
	}
	return true
}
func selected(bundle Bundle) []string {
	var out []string
	if labelsMatch(bundle.Service.Selector, bundle.Deployment.Template.Labels) {
		out = append(out, "deployment/"+bundle.Deployment.Name)
	}
	for _, pod := range bundle.Pods {
		if labelsMatch(bundle.Service.Selector, pod.Labels) {
			out = append(out, "pod/"+pod.Name)
		}
	}
	sort.Strings(out)
	return out
}
func backend(bundle Bundle, selected []string) (int, bool) {
	number := 0
	for _, id := range selected {
		containers := bundle.Deployment.Template.Containers
		if strings.HasPrefix(id, "pod/") {
			containers = nil
			for _, pod := range bundle.Pods {
				if pod.Name == strings.TrimPrefix(id, "pod/") {
					containers = pod.Containers
				}
			}
		}
		value, ok := resolvePod(containers, bundle.Service.Port.TargetPort, bundle.Service.Port.Port)
		if !ok || number != 0 && number != value {
			return 0, false
		}
		number = value
	}
	return number, number != 0
}
func resolvePod(containers []Container, ref PortRef, service int) (int, bool) {
	if ref.Kind == "default" {
		return service, true
	}
	if ref.Kind == "number" {
		return ref.Number, true
	}
	found, number := 0, 0
	for _, container := range containers {
		for _, port := range container.Ports {
			if port.Name == ref.Name {
				found++
				number = port.Number
			}
		}
	}
	return number, found == 1
}
func resolveContainer(container Container, ref PortRef) (int, bool) {
	if ref.Kind == "number" {
		return ref.Number, true
	}
	if ref.Kind != "name" {
		return 0, false
	}
	found, number := 0, 0
	for _, port := range container.Ports {
		if port.Name == ref.Name {
			found++
			number = port.Number
		}
	}
	return number, found == 1
}

func digest(bundle Bundle) string {
	values := make([]string, 0, len(bundle.Protected))
	for _, id := range bundle.Protected {
		values = append(values, id+"="+protected(bundle, id))
	}
	encoded, _ := json.Marshal(values)
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:])
}
func protected(bundle Bundle, id string) string {
	var path Path
	if json.Unmarshal([]byte(id), &path) != nil {
		return "<invalid>"
	}
	switch path.Kind {
	case "deployment-label", "template-label", "service-label", "pod-label":
		if value, ok := getLabel(bundle, path); ok {
			return value
		}
		return "<absent>"
	case "declared-port":
		if port, ok := getPort(bundle, path); ok {
			return fmt.Sprintf("%s:%d", port.Name, port.Number)
		}
		return "<absent>"
	case "service-target":
		encoded, _ := json.Marshal(bundle.Service.Port.TargetPort)
		return string(encoded)
	case "readiness-port":
		for _, container := range bundle.Deployment.Template.Containers {
			if container.Name == path.Container && container.Readiness != nil {
				encoded, _ := json.Marshal(container.Readiness.Port)
				return string(encoded)
			}
		}
	}
	return "<invalid>"
}

func getLabel(bundle Bundle, path Path) (string, bool) {
	switch path.Kind {
	case "deployment-label":
		return label(bundle.Deployment.Selector, path.Key)
	case "template-label":
		return label(bundle.Deployment.Template.Labels, path.Key)
	case "service-label":
		return label(bundle.Service.Selector, path.Key)
	case "pod-label":
		for _, pod := range bundle.Pods {
			if pod.Name == path.Resource {
				return label(pod.Labels, path.Key)
			}
		}
	}
	return "", false
}
func label(labels []Label, key string) (string, bool) {
	for _, value := range labels {
		if value.Key == key {
			return value.Value, true
		}
	}
	return "", false
}
func setLabel(bundle *Bundle, path Path, value string) bool {
	var labels *[]Label
	if path.Kind == "template-label" {
		labels = &bundle.Deployment.Template.Labels
	} else if path.Kind == "service-label" {
		labels = &bundle.Service.Selector
	} else {
		return false
	}
	for index := range *labels {
		if (*labels)[index].Key == path.Key {
			if (*labels)[index].Value == value {
				return false
			}
			(*labels)[index].Value = value
			return true
		}
	}
	*labels = append(*labels, Label{Key: path.Key, Value: value})
	sort.Slice(*labels, func(i, j int) bool { return (*labels)[i].Key < (*labels)[j].Key })
	return true
}
func removeLabel(labels *[]Label, key string) bool {
	for i, value := range *labels {
		if value.Key == key {
			*labels = append((*labels)[:i], (*labels)[i+1:]...)
			return true
		}
	}
	return false
}
func getPort(bundle Bundle, path Path) (NamedPort, bool) {
	for _, container := range bundle.Deployment.Template.Containers {
		if container.Name == path.Container {
			for _, port := range container.Ports {
				if port.Name == path.Port {
					return port, true
				}
			}
		}
	}
	return NamedPort{}, false
}
func setRef(bundle *Bundle, path Path, ref PortRef) bool {
	if path.Kind == "service-target" {
		if bundle.Service.Port.TargetPort == ref {
			return false
		}
		bundle.Service.Port.TargetPort = ref
		return true
	}
	if path.Kind == "readiness-port" {
		for i := range bundle.Deployment.Template.Containers {
			container := &bundle.Deployment.Template.Containers[i]
			if container.Name == path.Container && container.Readiness != nil {
				if container.Readiness.Port == ref {
					return false
				}
				container.Readiness.Port = ref
				return true
			}
		}
	}
	return false
}

func clone(bundle Bundle) Bundle {
	encoded, _ := json.Marshal(bundle)
	var out Bundle
	_ = json.Unmarshal(encoded, &out)
	return out
}
func pathID(path Path) string { encoded, _ := json.Marshal(path); return string(encoded) }
func contains(values []string, want string) bool {
	index := sort.SearchStrings(values, want)
	return index < len(values) && values[index] == want
}
func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
func sequenceKey(values []int) string {
	parts := make([]string, len(values))
	for i, value := range values {
		parts[i] = fmt.Sprintf("%03d", value)
	}
	return strings.Join(parts, "/")
}
