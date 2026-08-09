package nogoodexp

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/chazu/nous/internal/unit"
	"github.com/chazu/nous/internal/vocab/nogoods"
)

type FrozenArtifact struct {
	SchemaVersion          string   `json:"schema_version"`
	GuardVersion           string   `json:"guard_version"`
	Name                   string   `json:"name"`
	Mask                   int      `json:"mask"`
	PromotionProofCount    int      `json:"promotion_proof_count"`
	PromotionProofs        []string `json:"promotion_proofs"`
	EvidenceBoundaryDigest string   `json:"evidence_boundary_digest"`
	Provenance             string   `json:"provenance"`
	Digest                 string   `json:"digest"`
}

func FreezeArtifact(run TrainingRun) (FrozenArtifact, []byte, error) {
	if run.Terminal != "promoted" || run.Artifact == "" {
		return FrozenArtifact{}, nil, fmt.Errorf("training did not promote an artifact")
	}
	u := run.Store.Get(run.Artifact)
	if u == nil {
		return FrozenArtifact{}, nil, fmt.Errorf("promoted artifact unit is missing")
	}
	selection := run.Store.Get(u.GetString("selection"))
	if selection == nil {
		return FrozenArtifact{}, nil, fmt.Errorf("artifact selection is missing")
	}
	candidate := run.Store.Get(selection.GetString("selectedCandidate"))
	if candidate == nil {
		return FrozenArtifact{}, nil, fmt.Errorf("selected candidate is missing")
	}
	boundaryMaterial := struct {
		Selection string   `json:"selection"`
		Candidate string   `json:"candidate"`
		Evidence  []string `json:"evidence"`
	}{Selection: selection.Name, Candidate: candidate.Name, Evidence: slices.Clone(candidate.GetStrings("evidenceUnits"))}
	boundaryBytes, _ := json.Marshal(boundaryMaterial)
	boundaryDigest := sha256.Sum256(boundaryBytes)
	artifact := FrozenArtifact{
		SchemaVersion:          u.GetString("schemaVersion"),
		GuardVersion:           u.GetString("guardVersion"),
		Name:                   u.Name,
		Mask:                   u.GetInt("mask"),
		PromotionProofCount:    u.GetInt("promotionProofCount"),
		PromotionProofs:        slices.Clone(u.GetStrings("promotionProofs")),
		EvidenceBoundaryDigest: hex.EncodeToString(boundaryDigest[:]),
		Provenance:             u.GetString("provenance"),
	}
	unsigned, _ := json.Marshal(artifact)
	digest := sha256.Sum256(unsigned)
	artifact.Digest = hex.EncodeToString(digest[:])
	encoded, _ := json.Marshal(artifact)
	if err := artifact.Validate(); err != nil {
		return FrozenArtifact{}, nil, err
	}
	return artifact, encoded, nil
}

func ParseFrozenArtifact(data []byte) (FrozenArtifact, error) {
	var artifact FrozenArtifact
	if err := json.Unmarshal(data, &artifact); err != nil {
		return FrozenArtifact{}, err
	}
	canonical, _ := json.Marshal(artifact)
	if !slices.Equal(canonical, data) {
		return FrozenArtifact{}, fmt.Errorf("artifact encoding is not canonical")
	}
	if err := artifact.Validate(); err != nil {
		return FrozenArtifact{}, err
	}
	return artifact, nil
}

func (artifact FrozenArtifact) Validate() error {
	digest := artifact.Digest
	artifact.Digest = ""
	unsigned, _ := json.Marshal(artifact)
	want := sha256.Sum256(unsigned)
	if artifact.SchemaVersion != nogoods.SchemaVersion || artifact.GuardVersion != nogoods.GuardVersion || artifact.Mask != int(nogoods.FullMask) || artifact.PromotionProofCount != 24 || len(artifact.PromotionProofs) != 24 || artifact.Provenance != nogoodfixtureAuthority || len(artifact.EvidenceBoundaryDigest) != 64 || digest != hex.EncodeToString(want[:]) {
		return fmt.Errorf("invalid frozen nogood artifact")
	}
	return nil
}

const nogoodfixtureAuthority = "part3/nogoods/v1"

func installArtifact(store *unit.Store, artifact FrozenArtifact) error {
	if err := artifact.Validate(); err != nil {
		return err
	}
	u := unit.New(artifact.Name)
	u.Set("isA", []string{"NogoodArtifact", "Anything"})
	u.Set("schemaVersion", artifact.SchemaVersion)
	u.Set("guardVersion", artifact.GuardVersion)
	u.Set("mask", artifact.Mask)
	u.Set("promotionProofCount", artifact.PromotionProofCount)
	u.Set("promotionProofs", slices.Clone(artifact.PromotionProofs))
	u.Set("evidenceBoundaryDigest", artifact.EvidenceBoundaryDigest)
	u.Set("provenance", artifact.Provenance)
	u.Set("artifactDigest", artifact.Digest)
	u.Set("frozen", true)
	u.Set("authoritative", true)
	store.Put(u)
	return nil
}
