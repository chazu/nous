// Package kuberepair implements a deliberately bounded Kubernetes selector
// and reference model. It has no dependency on Nous units, the DSL, or the
// engine.
package kuberepair

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"
	"sync"
)

const (
	BundleVersion = "kubernetes-bundle/v1"
	HandleVersion = "kubernetes-intent-handle/v1"
	EditVersion   = "kubernetes-edit/v1"
	CreditContext = "kubernetes-selector-reference/atomic-edits/v1"
	MaxValueBytes = 32768
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

type Envelope struct {
	Version string  `json:"version"`
	Bundle  *Bundle `json:"bundle,omitempty"`
	Handle  string  `json:"handle,omitempty"`
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
	DesiredPods     []string       `json:"desiredPods"`
	BackendPort     int            `json:"backendPort"`
	ReadinessPorts  map[string]int `json:"readinessPorts"`
	ProtectedDigest string         `json:"protectedDigest"`
}

type intentRecord struct {
	intent Intent
	calls  []string
}

var intentTable = struct {
	sync.RWMutex
	values map[string]*intentRecord
}{values: map[string]*intentRecord{}}

// RegisterIntent installs a driver-owned opaque capability for evaluation and
// returns a cleanup function. Handles must be unique and 64 lowercase hex
// digits, so store-visible data cannot reveal intent equality.
func RegisterIntent(handle string, intent Intent) (func(), error) {
	if !validHandle(handle) || !validIntent(intent) {
		return nil, errors.New("invalid intent registration")
	}
	intentTable.Lock()
	defer intentTable.Unlock()
	if _, exists := intentTable.values[handle]; exists {
		return nil, errors.New("intent handle already registered")
	}
	intentTable.values[handle] = &intentRecord{intent: cloneIntent(intent)}
	return func() {
		intentTable.Lock()
		delete(intentTable.values, handle)
		intentTable.Unlock()
	}, nil
}

func cloneIntent(in Intent) Intent {
	out := in
	out.DesiredPods = append([]string(nil), in.DesiredPods...)
	out.ReadinessPorts = make(map[string]int, len(in.ReadinessPorts))
	for key, value := range in.ReadinessPorts {
		out.ReadinessPorts[key] = value
	}
	return out
}

// EvaluationLog returns the canonical state digests for Boolean terminal
// invocations made through one registered intent capability. It is driver-only
// instrumentation and is not exposed as a DSL word.
func EvaluationLog(handle string) []string {
	intentTable.RLock()
	defer intentTable.RUnlock()
	record := intentTable.values[handle]
	if record == nil {
		return nil
	}
	return append([]string(nil), record.calls...)
}

func lookupAndRecordIntent(handle, encodedBundle string) (Intent, bool) {
	intentTable.Lock()
	defer intentTable.Unlock()
	record, ok := intentTable.values[handle]
	if !ok {
		return Intent{}, false
	}
	digest := sha256.Sum256([]byte(encodedBundle))
	record.calls = append(record.calls, hex.EncodeToString(digest[:]))
	return cloneIntent(record.intent), true
}

func validHandle(value string) bool {
	if len(value) != 64 {
		return false
	}
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == 32 && value == strings.ToLower(value)
}

func ValidBundle(encoded string) bool {
	_, err := DecodeBundle(encoded)
	return err == nil
}

func ValidValue(encoded string) bool {
	_, kind, err := decodeValue(encoded)
	return err == nil && (kind == BundleVersion || kind == HandleVersion)
}

func EncodeBundle(bundle Bundle) (string, error) {
	canonicalizeBundle(&bundle)
	if err := validateBundle(bundle); err != nil {
		return "", err
	}
	return marshalCanonical(Envelope{Version: BundleVersion, Bundle: &bundle})
}

func EncodeHandle(handle string) (string, error) {
	if !validHandle(handle) {
		return "", errors.New("invalid intent handle")
	}
	return marshalCanonical(Envelope{Version: HandleVersion, Handle: handle})
}

func EncodeEdit(edit Edit) (string, error) {
	if err := validateEdit(edit); err != nil {
		return "", err
	}
	return marshalCanonical(edit)
}

func DecodeBundle(encoded string) (Bundle, error) {
	envelope, kind, err := decodeValue(encoded)
	if err != nil || kind != BundleVersion || envelope.Bundle == nil {
		if err == nil {
			err = errors.New("not a bundle")
		}
		return Bundle{}, err
	}
	return *envelope.Bundle, nil
}

func DecodeEdit(encoded string) (Edit, error) {
	if len(encoded) == 0 || len(encoded) > MaxValueBytes {
		return Edit{}, errors.New("invalid edit size")
	}
	var edit Edit
	if err := strictDecode(encoded, &edit); err != nil {
		return Edit{}, err
	}
	if err := validateEdit(edit); err != nil {
		return Edit{}, err
	}
	canonical, _ := marshalCanonical(edit)
	if canonical != encoded {
		return Edit{}, errors.New("noncanonical edit")
	}
	return edit, nil
}

func decodeValue(encoded string) (Envelope, string, error) {
	if len(encoded) == 0 || len(encoded) > MaxValueBytes {
		return Envelope{}, "", errors.New("invalid value size")
	}
	var envelope Envelope
	if err := strictDecode(encoded, &envelope); err != nil {
		return Envelope{}, "", err
	}
	switch envelope.Version {
	case BundleVersion:
		if envelope.Bundle == nil || envelope.Handle != "" {
			return Envelope{}, "", errors.New("malformed bundle envelope")
		}
		canonicalizeBundle(envelope.Bundle)
		if err := validateBundle(*envelope.Bundle); err != nil {
			return Envelope{}, "", err
		}
	case HandleVersion:
		if envelope.Bundle != nil || !validHandle(envelope.Handle) {
			return Envelope{}, "", errors.New("malformed handle envelope")
		}
	default:
		return Envelope{}, "", errors.New("unknown value version")
	}
	canonical, _ := marshalCanonical(envelope)
	if canonical != encoded {
		return Envelope{}, "", errors.New("noncanonical value")
	}
	return envelope, envelope.Version, nil
}

func strictDecode(encoded string, target any) error {
	decoder := json.NewDecoder(strings.NewReader(encoded))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return errors.New("trailing JSON")
	}
	return nil
}

func marshalCanonical(value any) (string, error) {
	encoded, err := json.Marshal(value)
	return string(encoded), err
}

func canonicalizeBundle(bundle *Bundle) {
	sortLabels(bundle.Deployment.Selector)
	sortLabels(bundle.Deployment.Template.Labels)
	canonicalizeContainers(bundle.Deployment.Template.Containers)
	sortLabels(bundle.Service.Selector)
	for index := range bundle.Pods {
		sortLabels(bundle.Pods[index].Labels)
		canonicalizeContainers(bundle.Pods[index].Containers)
	}
	sort.Slice(bundle.Pods, func(i, j int) bool { return bundle.Pods[i].Name < bundle.Pods[j].Name })
	sort.Strings(bundle.Protected)
	sort.Strings(bundle.Writes)
}

func canonicalizeContainers(containers []Container) {
	for index := range containers {
		sort.Slice(containers[index].Ports, func(i, j int) bool { return containers[index].Ports[i].Name < containers[index].Ports[j].Name })
	}
	sort.Slice(containers, func(i, j int) bool { return containers[i].Name < containers[j].Name })
}

func sortLabels(labels []Label) {
	sort.Slice(labels, func(i, j int) bool { return labels[i].Key < labels[j].Key })
}

func validateBundle(bundle Bundle) error {
	if !validName(bundle.Namespace) || !validName(bundle.Deployment.Name) || !validName(bundle.Service.Name) ||
		len(bundle.Pods) > 2 || len(bundle.Deployment.Template.Containers) < 1 || len(bundle.Deployment.Template.Containers) > 2 {
		return errors.New("invalid bundle bounds or identity")
	}
	if err := validateLabels(bundle.Deployment.Selector, true); err != nil {
		return fmt.Errorf("deployment selector: %w", err)
	}
	if err := validateLabels(bundle.Deployment.Template.Labels, false); err != nil {
		return fmt.Errorf("template labels: %w", err)
	}
	if err := validateLabels(bundle.Service.Selector, false); err != nil {
		return fmt.Errorf("service selector: %w", err)
	}
	if err := validateContainers(bundle.Deployment.Template.Containers); err != nil {
		return err
	}
	if !validName(bundle.Service.Port.Name) || !validPort(bundle.Service.Port.Port) || !validPortRef(bundle.Service.Port.TargetPort) {
		return errors.New("invalid service port")
	}
	seenPods := map[string]bool{}
	for _, pod := range bundle.Pods {
		if !validName(pod.Name) || seenPods[pod.Name] {
			return errors.New("invalid or duplicate pod")
		}
		seenPods[pod.Name] = true
		if err := validateLabels(pod.Labels, false); err != nil {
			return err
		}
		if err := validateContainers(pod.Containers); err != nil {
			return err
		}
	}
	if !sortedUnique(bundle.Protected) || !sortedUnique(bundle.Writes) {
		return errors.New("invalid path sets")
	}
	for _, encodedPath := range append(append([]string(nil), bundle.Protected...), bundle.Writes...) {
		if encodedPath == "" || len(encodedPath) > 512 {
			return errors.New("invalid path identity")
		}
		var path Path
		if strictDecode(encodedPath, &path) != nil || !validPath(path, false) || pathID(path) != encodedPath || !pathExists(bundle, path) {
			return errors.New("unresolved path identity")
		}
	}
	for _, encodedPath := range bundle.Writes {
		var path Path
		_ = json.Unmarshal([]byte(encodedPath), &path)
		if !writablePath(path) || contains(bundle.Protected, encodedPath) {
			return errors.New("invalid write history")
		}
	}
	return nil
}

func validateLabels(labels []Label, nonempty bool) error {
	if (nonempty && len(labels) == 0) || len(labels) > 4 {
		return errors.New("invalid label count")
	}
	previous := ""
	for _, label := range labels {
		if !validAtom(label.Key) || !validAtom(label.Value) || label.Key <= previous {
			return errors.New("invalid labels")
		}
		previous = label.Key
	}
	return nil
}

func validateContainers(containers []Container) error {
	if len(containers) == 0 || len(containers) > 2 {
		return errors.New("invalid container count")
	}
	previous := ""
	seenPortNames := map[string]bool{}
	for _, container := range containers {
		if !validName(container.Name) || container.Name <= previous || len(container.Ports) > 4 {
			return errors.New("invalid containers")
		}
		previous = container.Name
		portPrevious := ""
		for _, port := range container.Ports {
			if !validName(port.Name) || port.Name <= portPrevious || !validPort(port.Number) || seenPortNames[port.Name] {
				return errors.New("invalid ports")
			}
			seenPortNames[port.Name] = true
			portPrevious = port.Name
		}
		if container.Readiness != nil && (container.Readiness.Path == "" || len(container.Readiness.Path) > 256 || !validPortRef(container.Readiness.Port) || container.Readiness.Port.Kind == "default") {
			return errors.New("invalid readiness")
		}
	}
	return nil
}

func validName(value string) bool { return namePattern.MatchString(value) }
func validAtom(value string) bool { return validName(value) }
func validPort(value int) bool    { return value >= 1 && value <= 65535 }

func validPortRef(ref PortRef) bool {
	switch ref.Kind {
	case "default":
		return ref.Name == "" && ref.Number == 0
	case "name":
		return validName(ref.Name) && ref.Number == 0
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

func validIntent(intent Intent) bool {
	if !validPort(intent.BackendPort) || len(intent.DesiredPods) == 0 || !sortedUnique(intent.DesiredPods) || len(intent.ProtectedDigest) != 64 {
		return false
	}
	if _, err := hex.DecodeString(intent.ProtectedDigest); err != nil {
		return false
	}
	for key, value := range intent.ReadinessPorts {
		if !validName(key) || !validPort(value) {
			return false
		}
	}
	return true
}

func validateEdit(edit Edit) error {
	if edit.Version != EditVersion || !validPath(edit.Destination, true) {
		return errors.New("invalid edit")
	}
	switch edit.Kind {
	case "put-label":
		if edit.Source == nil || !validPath(*edit.Source, false) || edit.Destination.Kind != "template-label" && edit.Destination.Kind != "service-label" || !strings.HasSuffix(edit.Source.Kind, "label") || edit.Source.Key != edit.Destination.Key {
			return errors.New("invalid put-label")
		}
	case "remove-label":
		if edit.Source != nil || edit.Destination.Kind != "service-label" {
			return errors.New("invalid remove-label")
		}
	case "set-port-name", "set-port-number":
		if edit.Source == nil || edit.Source.Kind != "declared-port" || edit.Destination.Kind != "service-target" && edit.Destination.Kind != "readiness-port" || !validPath(*edit.Source, false) {
			return errors.New("invalid port edit")
		}
	case "unset-service-target":
		if edit.Source != nil || edit.Destination.Kind != "service-target" {
			return errors.New("invalid unset")
		}
	default:
		return errors.New("unknown edit kind")
	}
	return nil
}

func validPath(path Path, _ bool) bool {
	if !validName(path.Resource) {
		return false
	}
	switch path.Kind {
	case "deployment-label", "template-label", "service-label", "pod-label":
		return validAtom(path.Key) && path.Container == "" && path.Port == ""
	case "declared-port":
		return validName(path.Container) && validName(path.Port) && path.Key == ""
	case "readiness-port":
		return validName(path.Container) && path.Port == "" && path.Key == ""
	case "service-target":
		return path.Container == "" && path.Port == "" && path.Key == ""
	case "service-port":
		return path.Container == "" && path.Port == "" && path.Key == ""
	case "readiness-path", "readiness-presence":
		return validName(path.Container) && path.Port == "" && path.Key == ""
	default:
		return false
	}
}

func writablePath(path Path) bool {
	switch path.Kind {
	case "template-label", "service-label", "service-target", "readiness-port":
		return true
	default:
		return false
	}
}

func pathExists(bundle Bundle, path Path) bool {
	switch path.Kind {
	case "deployment-label":
		_, ok := labelValue(bundle.Deployment.Selector, path.Key)
		return path.Resource == bundle.Deployment.Name && ok
	case "template-label":
		// Missing label leaves are valid destinations when their key exists at a
		// public source; source reads still require presence.
		return path.Resource == bundle.Deployment.Name
	case "service-label":
		return path.Resource == bundle.Service.Name
	case "pod-label":
		for _, pod := range bundle.Pods {
			if pod.Name == path.Resource {
				_, ok := labelValue(pod.Labels, path.Key)
				return ok
			}
		}
		return false
	case "declared-port":
		_, ok := getDeclaredPort(bundle, path)
		return ok
	case "service-target", "service-port":
		return path.Resource == bundle.Service.Name
	case "readiness-port", "readiness-path", "readiness-presence":
		if path.Resource != bundle.Deployment.Name {
			return false
		}
		for _, container := range bundle.Deployment.Template.Containers {
			if container.Name == path.Container {
				return container.Readiness != nil
			}
		}
	}
	return false
}

func validateBoundEdit(bundle Bundle, edit Edit) error {
	if !pathExists(bundle, edit.Destination) || !writablePath(edit.Destination) {
		return errors.New("unresolved edit destination")
	}
	if edit.Source != nil && !pathExists(bundle, *edit.Source) {
		return errors.New("unresolved edit source")
	}
	if edit.Destination.Kind == "readiness-port" && edit.Source != nil && edit.Source.Container != edit.Destination.Container {
		return errors.New("readiness source is not owned by destination container")
	}
	return nil
}

func pathID(path Path) string {
	encoded, _ := json.Marshal(path)
	return string(encoded)
}

func EnumerateEdits(encoded string) ([]string, error) {
	bundle, err := DecodeBundle(encoded)
	if err != nil {
		return nil, err
	}
	var edits []Edit
	labelSources := labelPaths(bundle)
	keys := map[string]bool{}
	for _, source := range labelSources {
		keys[source.Key] = true
	}
	var labelDestinations []Path
	for key := range keys {
		labelDestinations = append(labelDestinations,
			Path{Kind: "template-label", Resource: bundle.Deployment.Name, Key: key},
			Path{Kind: "service-label", Resource: bundle.Service.Name, Key: key})
	}
	sort.Slice(labelDestinations, func(i, j int) bool { return pathID(labelDestinations[i]) < pathID(labelDestinations[j]) })
	for _, destination := range labelDestinations {
		for _, source := range labelSources {
			if source.Key != destination.Key || pathID(source) == pathID(destination) {
				continue
			}
			addIfChanges(bundle, Edit{Version: EditVersion, Kind: "put-label", Destination: destination, Source: &source}, &edits)
		}
	}
	for _, destination := range serviceLabelPaths(bundle) {
		addIfChanges(bundle, Edit{Version: EditVersion, Kind: "remove-label", Destination: destination}, &edits)
	}
	var references []Path
	references = append(references, Path{Kind: "service-target", Resource: bundle.Service.Name})
	for _, container := range bundle.Deployment.Template.Containers {
		if container.Readiness != nil {
			references = append(references, Path{Kind: "readiness-port", Resource: bundle.Deployment.Name, Container: container.Name})
		}
	}
	ports := declaredPortPaths(bundle.Deployment.Name, bundle.Deployment.Template.Containers)
	for _, destination := range references {
		for _, source := range ports {
			sourceCopy := source
			addIfChanges(bundle, Edit{Version: EditVersion, Kind: "set-port-name", Destination: destination, Source: &sourceCopy}, &edits)
			addIfChanges(bundle, Edit{Version: EditVersion, Kind: "set-port-number", Destination: destination, Source: &sourceCopy}, &edits)
		}
	}
	addIfChanges(bundle, Edit{Version: EditVersion, Kind: "unset-service-target", Destination: Path{Kind: "service-target", Resource: bundle.Service.Name}}, &edits)
	encodedEdits := make([]string, 0, len(edits))
	seen := map[string]bool{}
	for _, edit := range edits {
		value, _ := EncodeEdit(edit)
		if !seen[value] {
			seen[value] = true
			encodedEdits = append(encodedEdits, value)
		}
	}
	sort.Strings(encodedEdits)
	return encodedEdits, nil
}

func addIfChanges(bundle Bundle, edit Edit, edits *[]Edit) {
	if contains(bundle.Protected, pathID(edit.Destination)) || contains(bundle.Writes, pathID(edit.Destination)) {
		return
	}
	if validateBoundEdit(bundle, edit) != nil {
		return
	}
	copyBundle := cloneBundle(bundle)
	if _, changed, err := applyDecoded(&copyBundle, edit); err == nil && changed {
		*edits = append(*edits, edit)
	}
}

func cloneBundle(bundle Bundle) Bundle {
	encoded, _ := json.Marshal(bundle)
	var out Bundle
	_ = json.Unmarshal(encoded, &out)
	return out
}

func Apply(encodedBundle, encodedEdit string) (string, error) {
	bundle, err := DecodeBundle(encodedBundle)
	if err != nil {
		return "", err
	}
	edit, err := DecodeEdit(encodedEdit)
	if err != nil {
		return "", err
	}
	if contains(bundle.Protected, pathID(edit.Destination)) || contains(bundle.Writes, pathID(edit.Destination)) {
		return "", errors.New("illegal destination")
	}
	if err := validateBoundEdit(bundle, edit); err != nil {
		return "", err
	}
	if _, changed, err := applyDecoded(&bundle, edit); err != nil || !changed {
		if err == nil {
			err = errors.New("no-op edit")
		}
		return "", err
	}
	bundle.Writes = append(bundle.Writes, pathID(edit.Destination))
	return EncodeBundle(bundle)
}

func applyDecoded(bundle *Bundle, edit Edit) (any, bool, error) {
	switch edit.Kind {
	case "put-label":
		value, ok := getLabel(*bundle, *edit.Source)
		if !ok {
			return nil, false, errors.New("missing label source")
		}
		return setLabel(bundle, edit.Destination, value)
	case "remove-label":
		changed := removeLabel(&bundle.Service.Selector, edit.Destination.Key)
		return nil, changed, nil
	case "set-port-name", "set-port-number":
		port, ok := getDeclaredPort(*bundle, *edit.Source)
		if !ok {
			return nil, false, errors.New("missing port source")
		}
		ref := PortRef{Kind: "name", Name: port.Name}
		if edit.Kind == "set-port-number" {
			ref = PortRef{Kind: "number", Number: port.Number}
		}
		return setPortRef(bundle, edit.Destination, ref)
	case "unset-service-target":
		ref := PortRef{Kind: "default"}
		return setPortRef(bundle, edit.Destination, ref)
	default:
		return nil, false, errors.New("unknown edit")
	}
}

func labelPaths(bundle Bundle) []Path {
	paths := []Path{}
	for _, label := range bundle.Deployment.Selector {
		paths = append(paths, Path{Kind: "deployment-label", Resource: bundle.Deployment.Name, Key: label.Key})
	}
	paths = append(paths, templateLabelPaths(bundle)...)
	paths = append(paths, serviceLabelPaths(bundle)...)
	for _, pod := range bundle.Pods {
		for _, label := range pod.Labels {
			paths = append(paths, Path{Kind: "pod-label", Resource: pod.Name, Key: label.Key})
		}
	}
	return paths
}

func templateLabelPaths(bundle Bundle) []Path {
	var out []Path
	for _, l := range bundle.Deployment.Template.Labels {
		out = append(out, Path{Kind: "template-label", Resource: bundle.Deployment.Name, Key: l.Key})
	}
	for _, l := range bundle.Deployment.Selector {
		if _, ok := labelValue(bundle.Deployment.Template.Labels, l.Key); !ok {
			out = append(out, Path{Kind: "template-label", Resource: bundle.Deployment.Name, Key: l.Key})
		}
	}
	return uniquePaths(out)
}
func serviceLabelPaths(bundle Bundle) []Path {
	var out []Path
	for _, l := range bundle.Service.Selector {
		out = append(out, Path{Kind: "service-label", Resource: bundle.Service.Name, Key: l.Key})
	}
	for _, l := range bundle.Deployment.Selector {
		if _, ok := labelValue(bundle.Service.Selector, l.Key); !ok {
			out = append(out, Path{Kind: "service-label", Resource: bundle.Service.Name, Key: l.Key})
		}
	}
	return uniquePaths(out)
}
func uniquePaths(in []Path) []Path {
	seen := map[string]bool{}
	var out []Path
	for _, p := range in {
		id := pathID(p)
		if !seen[id] {
			seen[id] = true
			out = append(out, p)
		}
	}
	return out
}

func declaredPortPaths(resource string, containers []Container) []Path {
	var out []Path
	for _, c := range containers {
		for _, p := range c.Ports {
			out = append(out, Path{Kind: "declared-port", Resource: resource, Container: c.Name, Port: p.Name})
		}
	}
	return out
}

func getLabel(bundle Bundle, path Path) (string, bool) {
	switch path.Kind {
	case "deployment-label":
		if path.Resource != bundle.Deployment.Name {
			return "", false
		}
		return labelValue(bundle.Deployment.Selector, path.Key)
	case "template-label":
		if path.Resource != bundle.Deployment.Name {
			return "", false
		}
		return labelValue(bundle.Deployment.Template.Labels, path.Key)
	case "service-label":
		if path.Resource != bundle.Service.Name {
			return "", false
		}
		return labelValue(bundle.Service.Selector, path.Key)
	case "pod-label":
		for _, pod := range bundle.Pods {
			if pod.Name == path.Resource {
				return labelValue(pod.Labels, path.Key)
			}
		}
	}
	return "", false
}

func labelValue(labels []Label, key string) (string, bool) {
	index := sort.Search(len(labels), func(i int) bool { return labels[i].Key >= key })
	if index < len(labels) && labels[index].Key == key {
		return labels[index].Value, true
	}
	return "", false
}
func setLabel(bundle *Bundle, path Path, value string) (any, bool, error) {
	var labels *[]Label
	switch path.Kind {
	case "template-label":
		if path.Resource != bundle.Deployment.Name {
			return nil, false, errors.New("wrong deployment")
		}
		labels = &bundle.Deployment.Template.Labels
	case "service-label":
		if path.Resource != bundle.Service.Name {
			return nil, false, errors.New("wrong service")
		}
		labels = &bundle.Service.Selector
	default:
		return nil, false, errors.New("illegal label destination")
	}
	old, ok := labelValue(*labels, path.Key)
	if ok && old == value {
		return value, false, nil
	}
	if ok {
		for i := range *labels {
			if (*labels)[i].Key == path.Key {
				(*labels)[i].Value = value
			}
		}
	} else {
		*labels = append(*labels, Label{Key: path.Key, Value: value})
	}
	sortLabels(*labels)
	return value, true, nil
}
func removeLabel(labels *[]Label, key string) bool {
	for i, l := range *labels {
		if l.Key == key {
			*labels = append((*labels)[:i], (*labels)[i+1:]...)
			return true
		}
	}
	return false
}

func getDeclaredPort(bundle Bundle, path Path) (NamedPort, bool) {
	if path.Resource != bundle.Deployment.Name {
		return NamedPort{}, false
	}
	containers := bundle.Deployment.Template.Containers
	for _, c := range containers {
		if c.Name == path.Container {
			for _, p := range c.Ports {
				if p.Name == path.Port {
					return p, true
				}
			}
		}
	}
	return NamedPort{}, false
}
func setPortRef(bundle *Bundle, path Path, ref PortRef) (any, bool, error) {
	switch path.Kind {
	case "service-target":
		if path.Resource != bundle.Service.Name {
			return nil, false, errors.New("wrong service")
		}
		old := bundle.Service.Port.TargetPort
		if old == ref {
			return ref, false, nil
		}
		bundle.Service.Port.TargetPort = ref
		return ref, true, nil
	case "readiness-port":
		if path.Resource != bundle.Deployment.Name {
			return nil, false, errors.New("wrong deployment")
		}
		for i := range bundle.Deployment.Template.Containers {
			c := &bundle.Deployment.Template.Containers[i]
			if c.Name == path.Container && c.Readiness != nil {
				old := c.Readiness.Port
				if old == ref {
					return ref, false, nil
				}
				c.Readiness.Port = ref
				return ref, true, nil
			}
		}
		return nil, false, errors.New("missing readiness target")
	default:
		return nil, false, errors.New("illegal reference destination")
	}
}

func EqualOrSatisfies(left, right string) bool {
	leftEnvelope, leftKind, leftErr := decodeValue(left)
	rightEnvelope, rightKind, rightErr := decodeValue(right)
	if leftErr != nil || rightErr != nil {
		return false
	}
	if leftKind == rightKind {
		return left == right
	}
	if leftKind != BundleVersion || rightKind != HandleVersion {
		return false
	}
	intent, ok := lookupAndRecordIntent(rightEnvelope.Handle, left)
	return ok && satisfies(*leftEnvelope.Bundle, intent)
}

// PublicViolationVector is the fixed, intent-blind signal used by the matched
// conventional baseline. Smaller vectors are better.
func PublicViolationVector(encoded string) ([]int, error) {
	bundle, err := DecodeBundle(encoded)
	if err != nil {
		return nil, err
	}
	deploymentMismatches := 0
	for _, selector := range bundle.Deployment.Selector {
		value, ok := labelValue(bundle.Deployment.Template.Labels, selector.Key)
		if !ok || value != selector.Value {
			deploymentMismatches++
		}
	}
	emptyService := 0
	if len(bundle.Service.Selector) == 0 {
		emptyService = 1
	}
	selected := selectedPods(bundle)
	noSelected := 0
	serviceUnresolved := 0
	if len(selected) == 0 {
		noSelected = 1
	} else if _, ok := serviceBackend(bundle, selected); !ok {
		serviceUnresolved = 1
	}
	readinessUnresolved := 0
	for _, container := range bundle.Deployment.Template.Containers {
		if container.Readiness != nil {
			if _, ok := resolveInContainer(container, container.Readiness.Port); !ok {
				readinessUnresolved++
			}
		}
	}
	return []int{deploymentMismatches, emptyService, noSelected, serviceUnresolved, readinessUnresolved, len(bundle.Writes)}, nil
}

func satisfies(bundle Bundle, intent Intent) bool {
	if ProtectedDigest(bundle) != intent.ProtectedDigest || !labelsMatch(bundle.Deployment.Selector, bundle.Deployment.Template.Labels) {
		return false
	}
	selected := selectedPods(bundle)
	if !sameStrings(selected, intent.DesiredPods) {
		return false
	}
	backend, ok := serviceBackend(bundle, selected)
	if !ok || backend != intent.BackendPort {
		return false
	}
	if len(intent.ReadinessPorts) != readinessCount(bundle.Deployment.Template.Containers) {
		return false
	}
	for _, c := range bundle.Deployment.Template.Containers {
		if c.Readiness != nil {
			number, ok := resolveInContainer(c, c.Readiness.Port)
			if !ok || intent.ReadinessPorts[c.Name] != number {
				return false
			}
		}
	}
	return true
}

func labelsMatch(selector, labels []Label) bool {
	for _, want := range selector {
		got, ok := labelValue(labels, want.Key)
		if !ok || got != want.Value {
			return false
		}
	}
	return true
}
func selectedPods(bundle Bundle) []string {
	var out []string
	if labelsMatch(bundle.Service.Selector, bundle.Deployment.Template.Labels) {
		out = append(out, "deployment/"+bundle.Deployment.Name)
	}
	for _, p := range bundle.Pods {
		if labelsMatch(bundle.Service.Selector, p.Labels) {
			out = append(out, "pod/"+p.Name)
		}
	}
	sort.Strings(out)
	return out
}
func serviceBackend(bundle Bundle, selected []string) (int, bool) {
	var number int
	for _, id := range selected {
		var containers []Container
		if strings.HasPrefix(id, "deployment/") {
			containers = bundle.Deployment.Template.Containers
		} else {
			name := strings.TrimPrefix(id, "pod/")
			for _, p := range bundle.Pods {
				if p.Name == name {
					containers = p.Containers
				}
			}
		}
		resolved, ok := resolvePod(containers, bundle.Service.Port.TargetPort, bundle.Service.Port.Port)
		if !ok || (number != 0 && number != resolved) {
			return 0, false
		}
		number = resolved
	}
	return number, number != 0
}
func resolvePod(containers []Container, ref PortRef, servicePort int) (int, bool) {
	if ref.Kind == "default" {
		return servicePort, true
	}
	if ref.Kind == "number" {
		return ref.Number, true
	}
	found := 0
	number := 0
	for _, c := range containers {
		for _, p := range c.Ports {
			if p.Name == ref.Name {
				found++
				number = p.Number
			}
		}
	}
	return number, found == 1
}
func resolveInContainer(container Container, ref PortRef) (int, bool) {
	if ref.Kind == "number" {
		return ref.Number, true
	}
	if ref.Kind != "name" {
		return 0, false
	}
	found := 0
	number := 0
	for _, p := range container.Ports {
		if p.Name == ref.Name {
			found++
			number = p.Number
		}
	}
	return number, found == 1
}
func readinessCount(containers []Container) int {
	n := 0
	for _, c := range containers {
		if c.Readiness != nil {
			n++
		}
	}
	return n
}
func sameStrings(a, b []string) bool {
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

func ProtectedDigest(bundle Bundle) string {
	values := make([]string, 0, len(bundle.Protected))
	for _, id := range bundle.Protected {
		values = append(values, id+"="+protectedValue(bundle, id))
	}
	encoded, _ := json.Marshal(values)
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:])
}

func protectedValue(bundle Bundle, id string) string {
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
		if p, ok := getDeclaredPort(bundle, path); ok {
			return fmt.Sprintf("%s:%d", p.Name, p.Number)
		}
		return "<absent>"
	case "service-target":
		if path.Resource != bundle.Service.Name {
			return "<invalid>"
		}
		encoded, _ := json.Marshal(bundle.Service.Port.TargetPort)
		return string(encoded)
	case "service-port":
		if path.Resource != bundle.Service.Name {
			return "<invalid>"
		}
		return fmt.Sprintf("%d", bundle.Service.Port.Port)
	case "readiness-port":
		if path.Resource != bundle.Deployment.Name {
			return "<invalid>"
		}
		for _, c := range bundle.Deployment.Template.Containers {
			if c.Name == path.Container && c.Readiness != nil {
				encoded, _ := json.Marshal(c.Readiness.Port)
				return string(encoded)
			}
		}
	case "readiness-path", "readiness-presence":
		if path.Resource != bundle.Deployment.Name {
			return "<invalid>"
		}
		for _, c := range bundle.Deployment.Template.Containers {
			if c.Name == path.Container {
				if c.Readiness == nil {
					return "absent"
				}
				if path.Kind == "readiness-presence" {
					return "present"
				}
				return c.Readiness.Path
			}
		}
	}
	return "<invalid>"
}
func contains(values []string, want string) bool {
	index := sort.SearchStrings(values, want)
	return index < len(values) && values[index] == want
}

func FeatureKey(encodedEdit string) (string, string, error) {
	edit, err := DecodeEdit(encodedEdit)
	if err != nil {
		return "", "", err
	}
	sourceRole := ""
	if edit.Source != nil {
		sourceRole = edit.Source.Kind
	}
	component := strings.Join([]string{"kube-feature/v1", edit.Kind, edit.Destination.Kind, sourceRole}, "|")
	representation := ""
	if edit.Kind == "set-port-name" {
		representation = "name"
	}
	if edit.Kind == "set-port-number" {
		representation = "number"
	}
	if edit.Kind == "unset-service-target" {
		representation = "default"
	}
	family := "label-copy"
	if edit.Kind == "remove-label" {
		family = "label-remove"
	}
	if strings.HasPrefix(edit.Kind, "set-port") || edit.Kind == "unset-service-target" {
		family = "reference-to-declared-port"
	}
	relation := strings.Join([]string{"kube-relation/v1", family, representation}, "|")
	return component, relation, nil
}
