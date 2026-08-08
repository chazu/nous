// Package kuberepairoracle is a source-independent exhaustive evaluator for the
// bounded Kubernetes repair experiment. It intentionally imports no production
// vocabulary, DSL, engine, seed, credit, fixture, or trial package.
package kuberepairoracle

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"
)

const (
	bundleVersion = "kubernetes-bundle/v1"
	editVersion   = "kubernetes-edit/v1"
)

var namePattern = regexp.MustCompile(`^[a-z](?:[a-z0-9-]{0,61}[a-z0-9])?$`)

type Label struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}
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
	States        []string
}

type Analysis struct {
	Edits  []string
	Result Result
}

// Analyze independently decodes, enumerates, applies, and solves a task. It
// intentionally accepts no production-generated edit universe.
func Analyze(public string, intent Intent, maxLength int) (Analysis, error) {
	wrapper, err := decodeBundle(public)
	if err != nil {
		return Analysis{}, err
	}
	edits := enumerate(wrapper.Bundle)
	encodedEdits := make([]string, len(edits))
	for i, edit := range edits {
		encoded, marshalErr := json.Marshal(edit)
		if marshalErr != nil {
			return Analysis{}, marshalErr
		}
		encodedEdits[i] = string(encoded)
	}
	sort.Strings(encodedEdits)
	for i, encoded := range encodedEdits {
		if err := json.Unmarshal([]byte(encoded), &edits[i]); err != nil {
			return Analysis{}, err
		}
	}
	result := solve(wrapper.Bundle, edits, intent, maxLength)
	return Analysis{Edits: encodedEdits, Result: result}, nil
}

// EnumerateEdits independently returns the complete canonical public edit set
// without evaluating private intent.
func EnumerateEdits(public string) ([]string, error) {
	wrapper, err := decodeBundle(public)
	if err != nil {
		return nil, err
	}
	edits := enumerate(wrapper.Bundle)
	encoded := make([]string, len(edits))
	for index, edit := range edits {
		value, marshalErr := json.Marshal(edit)
		if marshalErr != nil {
			return nil, marshalErr
		}
		encoded[index] = string(value)
	}
	sort.Strings(encoded)
	return encoded, nil
}

func Satisfies(public string, intent Intent) (bool, error) {
	wrapper, err := decodeBundle(public)
	if err != nil {
		return false, err
	}
	return satisfies(wrapper.Bundle, intent), nil
}

func decodeBundle(public string) (envelope, error) {
	var wrapper envelope
	decoder := json.NewDecoder(strings.NewReader(public))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&wrapper); err != nil || wrapper.Version != bundleVersion {
		return envelope{}, fmt.Errorf("decode public bundle")
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return envelope{}, fmt.Errorf("trailing public data")
	}
	canonical, _ := json.Marshal(wrapper)
	if string(canonical) != public || !validBundle(wrapper.Bundle) {
		return envelope{}, fmt.Errorf("noncanonical or invalid public bundle")
	}
	return wrapper, nil
}

func solve(initial Bundle, edits []Edit, intent Intent, maxLength int) Result {
	if satisfies(initial, intent) {
		return Result{Terminal: "already-correct", MinimumLength: 0, Plans: [][]int{{}}, States: []string{semanticState(initial)}}
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
					result.States = nil
				}
				if len(candidate) == result.MinimumLength {
					result.Plans = append(result.Plans, candidate)
					result.States = append(result.States, semanticState(next))
				}
				continue
			}
			if len(candidate) < result.MinimumLength {
				walk(next, candidate, remaining-1)
			}
		}
	}
	walk(initial, nil, maxLength)
	if len(result.Plans) > 0 {
		result.Terminal = "solution"
		sort.Slice(result.Plans, func(i, j int) bool { return sequenceKey(result.Plans[i]) < sequenceKey(result.Plans[j]) })
		sort.Strings(result.States)
		result.States = uniqueStrings(result.States)
	}
	return result
}

func semanticState(bundle Bundle) string {
	bundle.Writes = nil
	encoded, _ := json.Marshal(envelope{Version: bundleVersion, Bundle: bundle})
	return string(encoded)
}

func uniqueStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := values[:1]
	for _, value := range values[1:] {
		if value != out[len(out)-1] {
			out = append(out, value)
		}
	}
	return out
}

func enumerate(bundle Bundle) []Edit {
	var edits []Edit
	sources := labelPaths(bundle)
	keys := map[string]bool{}
	for _, source := range sources {
		keys[source.Key] = true
	}
	var destinations []Path
	for key := range keys {
		destinations = append(destinations,
			Path{Kind: "template-label", Resource: bundle.Deployment.Name, Key: key},
			Path{Kind: "service-label", Resource: bundle.Service.Name, Key: key})
	}
	sort.Slice(destinations, func(i, j int) bool { return pathID(destinations[i]) < pathID(destinations[j]) })
	for _, destination := range destinations {
		for _, source := range sources {
			if destination.Key == source.Key && pathID(destination) != pathID(source) {
				addIfChanges(bundle, Edit{Version: editVersion, Kind: "put-label", Destination: destination, Source: copyPath(source)}, &edits)
			}
		}
	}
	for _, label := range bundle.Service.Selector {
		addIfChanges(bundle, Edit{Version: editVersion, Kind: "remove-label", Destination: Path{Kind: "service-label", Resource: bundle.Service.Name, Key: label.Key}}, &edits)
	}
	references := []Path{{Kind: "service-target", Resource: bundle.Service.Name}}
	for _, container := range bundle.Deployment.Template.Containers {
		if container.Readiness != nil {
			references = append(references, Path{Kind: "readiness-port", Resource: bundle.Deployment.Name, Container: container.Name})
		}
	}
	ports := declaredPorts(bundle)
	for _, destination := range references {
		for _, source := range ports {
			if destination.Kind == "readiness-port" && destination.Container != source.Container {
				continue
			}
			addIfChanges(bundle, Edit{Version: editVersion, Kind: "set-port-name", Destination: destination, Source: copyPath(source)}, &edits)
			addIfChanges(bundle, Edit{Version: editVersion, Kind: "set-port-number", Destination: destination, Source: copyPath(source)}, &edits)
		}
	}
	addIfChanges(bundle, Edit{Version: editVersion, Kind: "unset-service-target", Destination: Path{Kind: "service-target", Resource: bundle.Service.Name}}, &edits)
	seen := map[string]bool{}
	var out []Edit
	for _, edit := range edits {
		encoded, _ := json.Marshal(edit)
		if !seen[string(encoded)] {
			seen[string(encoded)] = true
			out = append(out, edit)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		left, _ := json.Marshal(out[i])
		right, _ := json.Marshal(out[j])
		return string(left) < string(right)
	})
	return out
}

func copyPath(path Path) *Path { value := path; return &value }

func addIfChanges(bundle Bundle, edit Edit, edits *[]Edit) {
	if contains(bundle.Protected, pathID(edit.Destination)) || contains(bundle.Writes, pathID(edit.Destination)) {
		return
	}
	if _, ok := apply(bundle, edit); ok {
		*edits = append(*edits, edit)
	}
}

func labelPaths(bundle Bundle) []Path {
	var out []Path
	for _, label := range bundle.Deployment.Selector {
		out = append(out, Path{Kind: "deployment-label", Resource: bundle.Deployment.Name, Key: label.Key})
	}
	for _, label := range bundle.Deployment.Template.Labels {
		out = append(out, Path{Kind: "template-label", Resource: bundle.Deployment.Name, Key: label.Key})
	}
	for _, label := range bundle.Service.Selector {
		out = append(out, Path{Kind: "service-label", Resource: bundle.Service.Name, Key: label.Key})
	}
	for _, pod := range bundle.Pods {
		for _, label := range pod.Labels {
			out = append(out, Path{Kind: "pod-label", Resource: pod.Name, Key: label.Key})
		}
	}
	return out
}

func declaredPorts(bundle Bundle) []Path {
	var out []Path
	for _, container := range bundle.Deployment.Template.Containers {
		for _, port := range container.Ports {
			out = append(out, Path{Kind: "declared-port", Resource: bundle.Deployment.Name, Container: container.Name, Port: port.Name})
		}
	}
	return out
}

func validBundle(bundle Bundle) bool {
	if !namePattern.MatchString(bundle.Namespace) || !namePattern.MatchString(bundle.Deployment.Name) || !namePattern.MatchString(bundle.Service.Name) ||
		len(bundle.Deployment.Selector) == 0 || len(bundle.Pods) > 2 ||
		len(bundle.Deployment.Template.Containers) == 0 || len(bundle.Deployment.Template.Containers) > 2 ||
		!validLabels(bundle.Deployment.Selector) || !validLabels(bundle.Deployment.Template.Labels) || !validLabels(bundle.Service.Selector) ||
		!validContainers(bundle.Deployment.Template.Containers) || !namePattern.MatchString(bundle.Service.Port.Name) || !validPort(bundle.Service.Port.Port) || !validRef(bundle.Service.Port.TargetPort, true) ||
		!sortedUnique(bundle.Protected) || !sortedUnique(bundle.Writes) {
		return false
	}
	previousPod := ""
	for _, pod := range bundle.Pods {
		if !namePattern.MatchString(pod.Name) || pod.Name <= previousPod || !validLabels(pod.Labels) || !validContainers(pod.Containers) {
			return false
		}
		previousPod = pod.Name
	}
	for _, encoded := range append(append([]string(nil), bundle.Protected...), bundle.Writes...) {
		var path Path
		if json.Unmarshal([]byte(encoded), &path) != nil || pathID(path) != encoded || !validPathShape(path) || !pathExists(bundle, path) {
			return false
		}
	}
	for _, encoded := range bundle.Writes {
		var path Path
		_ = json.Unmarshal([]byte(encoded), &path)
		if contains(bundle.Protected, encoded) || !writablePath(path) {
			return false
		}
	}
	return true
}

func validPathShape(path Path) bool {
	if !namePattern.MatchString(path.Resource) {
		return false
	}
	switch path.Kind {
	case "deployment-label", "template-label", "service-label", "pod-label":
		return namePattern.MatchString(path.Key) && path.Container == "" && path.Port == ""
	case "declared-port":
		return namePattern.MatchString(path.Container) && namePattern.MatchString(path.Port) && path.Key == ""
	case "readiness-port", "readiness-path", "readiness-presence":
		return namePattern.MatchString(path.Container) && path.Port == "" && path.Key == ""
	case "service-target", "service-port":
		return path.Container == "" && path.Port == "" && path.Key == ""
	}
	return false
}

func writablePath(path Path) bool {
	switch path.Kind {
	case "template-label", "service-label", "service-target", "readiness-port":
		return true
	default:
		return false
	}
}

func validLabels(labels []Label) bool {
	if len(labels) > 4 {
		return false
	}
	previous := ""
	for _, label := range labels {
		if !namePattern.MatchString(label.Key) || !namePattern.MatchString(label.Value) || label.Key <= previous {
			return false
		}
		previous = label.Key
	}
	return true
}

func validContainers(containers []Container) bool {
	if len(containers) == 0 || len(containers) > 2 {
		return false
	}
	previous := ""
	ports := map[string]bool{}
	for _, container := range containers {
		if !namePattern.MatchString(container.Name) || container.Name <= previous || len(container.Ports) > 4 {
			return false
		}
		previous = container.Name
		previousPort := ""
		for _, port := range container.Ports {
			if !namePattern.MatchString(port.Name) || port.Name <= previousPort || !validPort(port.Number) || ports[port.Name] {
				return false
			}
			ports[port.Name] = true
			previousPort = port.Name
		}
		if container.Readiness != nil && (container.Readiness.Path == "" || len(container.Readiness.Path) > 256 || !validRef(container.Readiness.Port, false)) {
			return false
		}
	}
	return true
}

func validPort(value int) bool { return value >= 1 && value <= 65535 }
func validRef(ref PortRef, allowDefault bool) bool {
	switch ref.Kind {
	case "default":
		return allowDefault && ref.Name == "" && ref.Number == 0
	case "name":
		return namePattern.MatchString(ref.Name) && ref.Number == 0
	case "number":
		return ref.Name == "" && validPort(ref.Number)
	default:
		return false
	}
}
func sortedUnique(values []string) bool {
	for index := range values {
		if index > 0 && values[index] <= values[index-1] {
			return false
		}
	}
	return true
}

func pathExists(bundle Bundle, path Path) bool {
	switch path.Kind {
	case "deployment-label":
		_, ok := label(bundle.Deployment.Selector, path.Key)
		return path.Resource == bundle.Deployment.Name && ok
	case "template-label":
		return path.Resource == bundle.Deployment.Name
	case "service-label":
		return path.Resource == bundle.Service.Name
	case "pod-label":
		for _, pod := range bundle.Pods {
			if pod.Name == path.Resource {
				_, ok := label(pod.Labels, path.Key)
				return ok
			}
		}
	case "declared-port":
		_, ok := getPort(bundle, path)
		return ok
	case "service-target", "service-port":
		return path.Resource == bundle.Service.Name
	case "readiness-port", "readiness-path", "readiness-presence":
		if path.Resource == bundle.Deployment.Name {
			for _, container := range bundle.Deployment.Template.Containers {
				if container.Name == path.Container {
					return container.Readiness != nil
				}
			}
		}
	}
	return false
}

func apply(bundle Bundle, edit Edit) (Bundle, bool) {
	bundle = clone(bundle)
	destination := pathID(edit.Destination)
	if contains(bundle.Protected, destination) || contains(bundle.Writes, destination) || !pathExists(bundle, edit.Destination) || edit.Source != nil && !pathExists(bundle, *edit.Source) {
		return Bundle{}, false
	}
	if edit.Destination.Kind == "readiness-port" && edit.Source != nil && edit.Destination.Container != edit.Source.Container {
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
		if path.Resource != bundle.Deployment.Name {
			return "", false
		}
		return label(bundle.Deployment.Selector, path.Key)
	case "template-label":
		if path.Resource != bundle.Deployment.Name {
			return "", false
		}
		return label(bundle.Deployment.Template.Labels, path.Key)
	case "service-label":
		if path.Resource != bundle.Service.Name {
			return "", false
		}
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
		if path.Resource != bundle.Deployment.Name {
			return false
		}
		labels = &bundle.Deployment.Template.Labels
	} else if path.Kind == "service-label" {
		if path.Resource != bundle.Service.Name {
			return false
		}
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
	if path.Resource != bundle.Deployment.Name {
		return NamedPort{}, false
	}
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
		if path.Resource != bundle.Service.Name {
			return false
		}
		if bundle.Service.Port.TargetPort == ref {
			return false
		}
		bundle.Service.Port.TargetPort = ref
		return true
	}
	if path.Kind == "readiness-port" {
		if path.Resource != bundle.Deployment.Name {
			return false
		}
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
