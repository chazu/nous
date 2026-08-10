package actionrelations

import (
	"bytes"
	"encoding/json"
	"slices"
)

type Relation struct {
	Pattern              Pattern
	Guard                Guard
	PositiveObservations []string
	NegativeObservations []string
}

func ParseRelation(data []byte) (Relation, error) {
	v, err := decodeOne(data)
	if err != nil {
		return Relation{}, err
	}
	row, ok := v.([]any)
	if !ok || len(row) != 6 || row[0] != RelationVersion || row[1] != "commutes" {
		return Relation{}, ErrInvalid
	}
	patternBytes, _ := json.Marshal(row[2])
	pattern, err := ParsePattern(patternBytes)
	if err != nil {
		return Relation{}, err
	}
	guardBytes, _ := json.Marshal(row[3])
	guard, err := ParseGuard(guardBytes)
	if err != nil {
		return Relation{}, err
	}
	positive, ok := stringList(row[4])
	if !ok {
		return Relation{}, ErrInvalid
	}
	negative, ok := stringList(row[5])
	if !ok {
		return Relation{}, ErrInvalid
	}
	relation := Relation{Pattern: pattern, Guard: guard, PositiveObservations: positive, NegativeObservations: negative}
	if err := relation.Validate(); err != nil {
		return Relation{}, err
	}
	canonical, _ := relation.CanonicalJSON()
	if !bytes.Equal(canonical, data) {
		return Relation{}, ErrInvalid
	}
	return relation, nil
}

func (r Relation) Validate() error {
	if err := r.Pattern.Validate(); err != nil {
		return err
	}
	if err := r.Guard.Validate(); err != nil {
		return err
	}
	if !validDigestSet(r.PositiveObservations) || !validDigestSet(r.NegativeObservations) {
		return ErrInvalid
	}
	return nil
}

func (r Relation) CanonicalJSON() ([]byte, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal([]any{RelationVersion, "commutes", r.Pattern.wire(), r.Guard.wire(), nonNilStrings(r.PositiveObservations), nonNilStrings(r.NegativeObservations)})
}

func (r Relation) Digest() (string, error) { return digestCanonical(r.CanonicalJSON()) }

func (p Pattern) wire() []any { return []any{PatternVersion, p.Kinds, p.Roles} }

func (g Guard) wire() []any {
	literals := make([]any, len(g.Literals))
	for index, literal := range g.Literals {
		literals[index] = []any{literal.Atom, literal.Polarity}
	}
	return []any{GuardVersion, literals}
}

type Artifact struct {
	RelationDigests      []string
	SemanticTrainingRoot string
}

func NewArtifact(relations []Relation, semanticTrainingRoot string) (Artifact, error) {
	if len(relations) < 1 || len(relations) > 451 || !validDigest(semanticTrainingRoot) {
		return Artifact{}, ErrInvalid
	}
	type item struct {
		bytes  []byte
		digest string
	}
	items := make([]item, len(relations))
	for index, relation := range relations {
		data, err := relation.CanonicalJSON()
		if err != nil {
			return Artifact{}, err
		}
		digest, _ := relation.Digest()
		items[index] = item{bytes: data, digest: digest}
	}
	slices.SortFunc(items, func(a, b item) int { return bytes.Compare(a.bytes, b.bytes) })
	digests := make([]string, len(items))
	for index, item := range items {
		if index > 0 && bytes.Equal(items[index-1].bytes, item.bytes) {
			return Artifact{}, ErrInvalid
		}
		digests[index] = item.digest
	}
	return Artifact{RelationDigests: digests, SemanticTrainingRoot: semanticTrainingRoot}, nil
}

func ParseArtifact(data []byte) (Artifact, error) {
	v, err := decodeOne(data)
	if err != nil {
		return Artifact{}, err
	}
	row, ok := v.([]any)
	if !ok || len(row) != 3 || row[0] != ArtifactVersion {
		return Artifact{}, ErrInvalid
	}
	digests, ok := stringList(row[1])
	if !ok {
		return Artifact{}, ErrInvalid
	}
	root, ok := row[2].(string)
	if !ok {
		return Artifact{}, ErrInvalid
	}
	artifact := Artifact{RelationDigests: digests, SemanticTrainingRoot: root}
	if err := artifact.ValidateShape(); err != nil {
		return Artifact{}, err
	}
	canonical, _ := artifact.CanonicalJSON()
	if !bytes.Equal(canonical, data) {
		return Artifact{}, ErrInvalid
	}
	return artifact, nil
}

func (a Artifact) ValidateShape() error {
	if len(a.RelationDigests) < 1 || len(a.RelationDigests) > 451 || !validDigest(a.SemanticTrainingRoot) {
		return ErrInvalid
	}
	seen := map[string]bool{}
	for _, digest := range a.RelationDigests {
		if !validDigest(digest) || seen[digest] {
			return ErrInvalid
		}
		seen[digest] = true
	}
	return nil
}

// ValidateResolved verifies canonical relation-byte order and every digest.
func (a Artifact) ValidateResolved(relations map[string]Relation) error {
	if err := a.ValidateShape(); err != nil {
		return err
	}
	var previous []byte
	for index, digest := range a.RelationDigests {
		relation, ok := relations[digest]
		if !ok {
			return ErrInvalid
		}
		data, err := relation.CanonicalJSON()
		if err != nil {
			return err
		}
		actual, _ := relation.Digest()
		if actual != digest || index > 0 && bytes.Compare(previous, data) >= 0 {
			return ErrInvalid
		}
		previous = data
	}
	return nil
}

func (a Artifact) CanonicalJSON() ([]byte, error) {
	if err := a.ValidateShape(); err != nil {
		return nil, err
	}
	return json.Marshal([]any{ArtifactVersion, a.RelationDigests, a.SemanticTrainingRoot})
}

func (a Artifact) Digest() (string, error) { return digestCanonical(a.CanonicalJSON()) }

func validDigestSet(values []string) bool {
	for index, value := range values {
		if !validDigest(value) || index > 0 && value <= values[index-1] {
			return false
		}
	}
	return true
}

func stringList(value any) ([]string, bool) {
	raw, ok := value.([]any)
	if !ok {
		return nil, false
	}
	result := make([]string, len(raw))
	for index, item := range raw {
		result[index], ok = item.(string)
		if !ok {
			return nil, false
		}
	}
	return result, true
}

func nonNilStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}
