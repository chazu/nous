package nogoodexp

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/chazu/nous/internal/unit"
	"github.com/chazu/nous/internal/vocab/nogoods"
)

const nogoodfixtureAuthority = "part3/nogoods/v1"

type FrozenResult struct {
	Key      string `json:"key"`
	XColor   int    `json:"x_color"`
	YColor   int    `json:"y_color"`
	Conflict bool   `json:"conflict"`
}

type FrozenEvidence struct {
	Example       string          `json:"example"`
	ProblemDigest string          `json:"problem_digest"`
	Binding       nogoods.Binding `json:"binding"`
	Matches       bool            `json:"matches"`
	AllConflict   bool            `json:"all_conflict"`
	ExpectedKeys  []string        `json:"expected_keys"`
	ActualKeys    []string        `json:"actual_keys"`
	Results       []FrozenResult  `json:"results"`
	Barrier       string          `json:"barrier"`
}

type FrozenPromotionProof struct {
	Proof         string             `json:"proof"`
	Case          string             `json:"case"`
	ProblemDigest string             `json:"problem_digest"`
	Binding       nogoods.Binding    `json:"binding"`
	Completion    nogoods.Completion `json:"completion"`
	Conflict      bool               `json:"conflict"`
}

type FrozenCandidate struct {
	Name             string           `json:"name"`
	Mask             int              `json:"mask"`
	TrainingExact    bool             `json:"training_exact"`
	EvidenceComplete bool             `json:"evidence_complete"`
	Evidence         []FrozenEvidence `json:"evidence"`
}

type FrozenArtifact struct {
	SchemaVersion          string                 `json:"schema_version"`
	GuardVersion           string                 `json:"guard_version"`
	Name                   string                 `json:"name"`
	Mask                   int                    `json:"mask"`
	SchemaSemanticKey      string                 `json:"schema_semantic_key"`
	Selection              string                 `json:"selection"`
	SelectionBarrier       string                 `json:"selection_barrier"`
	Candidates             []FrozenCandidate      `json:"candidates"`
	ExactTies              []string               `json:"exact_ties"`
	ExpectedMasks          []int                  `json:"expected_masks"`
	ActualMasks            []int                  `json:"actual_masks"`
	EvidenceBoundaryDigest string                 `json:"evidence_boundary_digest"`
	CompletionDigest       string                 `json:"completion_digest"`
	PromotionBarrier       string                 `json:"promotion_barrier"`
	PromotionProofs        []FrozenPromotionProof `json:"promotion_proofs"`
	PromotionDigest        string                 `json:"promotion_digest"`
	Provenance             string                 `json:"provenance"`
	ProvenanceDigest       string                 `json:"provenance_digest"`
	Digest                 string                 `json:"digest"`
}

// ArtifactAuthority is an opaque capability produced only by a successful
// fresh training freeze. Parsing structurally valid bytes does not grant it.
type ArtifactAuthority struct {
	digest           string
	evidenceBoundary string
	promotionDigest  string
	provenanceDigest string
}

func FreezeArtifact(run TrainingRun) (FrozenArtifact, []byte, ArtifactAuthority, error) {
	if run.Terminal != "promoted" || run.Artifact == "" {
		return FrozenArtifact{}, nil, ArtifactAuthority{}, fmt.Errorf("training did not promote an artifact")
	}
	u := run.Store.Get(run.Artifact)
	if u == nil {
		return FrozenArtifact{}, nil, ArtifactAuthority{}, fmt.Errorf("artifact unit is missing")
	}
	selection := run.Store.Get(u.GetString("selection"))
	if selection == nil {
		return FrozenArtifact{}, nil, ArtifactAuthority{}, fmt.Errorf("artifact selection is missing")
	}
	selectionBarrier := run.Store.Get(selection.GetString("barrier"))
	candidate := run.Store.Get(selection.GetString("selectedCandidate"))
	if selectionBarrier == nil || !selectionBarrier.GetBool("sealed") || candidate == nil {
		return FrozenArtifact{}, nil, ArtifactAuthority{}, fmt.Errorf("selection barrier is not sealed")
	}
	artifact := FrozenArtifact{
		SchemaVersion: u.GetString("schemaVersion"), GuardVersion: u.GetString("guardVersion"), Name: u.Name,
		Mask: u.GetInt("mask"), Selection: selection.Name, SelectionBarrier: selectionBarrier.Name,
		PromotionBarrier: u.GetString("promotionBarrier"), Provenance: u.GetString("provenance"),
	}
	artifact.ExactTies = slices.Clone(selection.GetStrings("ties"))
	artifact.ExpectedMasks = intSlot(selectionBarrier, "expectedKeys")
	artifact.ActualMasks = intSlot(selectionBarrier, "actualKeys")
	artifact.SchemaSemanticKey = digestJSON(struct {
		Schema string `json:"schema"`
		Guard  string `json:"guard"`
		Mask   int    `json:"mask"`
	}{artifact.SchemaVersion, artifact.GuardVersion, artifact.Mask})
	for _, candidateName := range run.Store.Examples("NogoodCandidate") {
		if candidateName == "NogoodCandidate" {
			continue
		}
		candidateUnit := run.Store.Get(candidateName)
		frozenCandidate := FrozenCandidate{Name: candidateName, Mask: candidateUnit.GetInt("mask"), TrainingExact: candidateUnit.GetBool("trainingExact"), EvidenceComplete: candidateUnit.GetBool("evidenceComplete")}
		for _, evidenceName := range candidateUnit.GetStrings("evidenceUnits") {
			evidence := run.Store.Get(evidenceName)
			if evidence == nil {
				return FrozenArtifact{}, nil, ArtifactAuthority{}, fmt.Errorf("missing training evidence %q", evidenceName)
			}
			example := run.Store.Get(evidence.GetString("example"))
			bindingUnit := run.Store.Get(evidence.GetString("binding"))
			barrier := run.Store.Get(evidence.GetString("barrier"))
			if example == nil || bindingUnit == nil || barrier == nil || !barrier.GetBool("sealed") {
				return FrozenArtifact{}, nil, ArtifactAuthority{}, fmt.Errorf("incomplete training evidence %q", evidenceName)
			}
			record := FrozenEvidence{Example: example.Name, ProblemDigest: digestBytes([]byte(example.GetString("problem"))), Binding: nogoods.Binding{Anchor: bindingUnit.GetInt("anchor"), X: bindingUnit.GetInt("x"), Y: bindingUnit.GetInt("y"), Blocked: bindingUnit.GetInt("blocked"), Escape: bindingUnit.GetInt("escape"), Only: bindingUnit.GetInt("only")}, Matches: evidence.GetBool("matches"), AllConflict: evidence.GetBool("allConflict"), ExpectedKeys: slices.Clone(barrier.GetStrings("expectedKeys")), ActualKeys: slices.Clone(barrier.GetStrings("actualKeys")), Barrier: barrier.Name}
			for _, resultName := range evidence.GetStrings("results") {
				result := run.Store.Get(resultName)
				if result == nil {
					return FrozenArtifact{}, nil, ArtifactAuthority{}, fmt.Errorf("missing completion result")
				}
				record.Results = append(record.Results, FrozenResult{Key: result.Name, XColor: result.GetInt("xColor"), YColor: result.GetInt("yColor"), Conflict: result.GetBool("conflict")})
			}
			frozenCandidate.Evidence = append(frozenCandidate.Evidence, record)
		}
		artifact.Candidates = append(artifact.Candidates, frozenCandidate)
	}
	sort.Slice(artifact.Candidates, func(i, j int) bool { return artifact.Candidates[i].Mask < artifact.Candidates[j].Mask })
	artifact.EvidenceBoundaryDigest = digestJSON(struct {
		Selection, Barrier string
		Ties               []string
		Expected, Actual   []int
		Candidates         []FrozenCandidate
	}{artifact.Selection, artifact.SelectionBarrier, artifact.ExactTies, artifact.ExpectedMasks, artifact.ActualMasks, artifact.Candidates})
	artifact.CompletionDigest = digestJSON(artifact.Candidates)
	promotionBarrier := run.Store.Get(artifact.PromotionBarrier)
	if promotionBarrier == nil || !promotionBarrier.GetBool("sealed") {
		return FrozenArtifact{}, nil, ArtifactAuthority{}, fmt.Errorf("promotion barrier is not sealed")
	}
	for _, proofName := range u.GetStrings("promotionProofs") {
		proof := run.Store.Get(proofName)
		if proof == nil {
			return FrozenArtifact{}, nil, ArtifactAuthority{}, fmt.Errorf("missing promotion proof")
		}
		promotionCase := run.Store.Get(proof.GetString("case"))
		if promotionCase == nil {
			return FrozenArtifact{}, nil, ArtifactAuthority{}, fmt.Errorf("missing promotion case")
		}
		artifact.PromotionProofs = append(artifact.PromotionProofs, FrozenPromotionProof{
			Proof: proof.Name, Case: promotionCase.Name, ProblemDigest: digestBytes([]byte(promotionCase.GetString("problem"))),
			Binding:    nogoods.Binding{Anchor: promotionCase.GetInt("anchor"), X: promotionCase.GetInt("x"), Y: promotionCase.GetInt("y"), Blocked: promotionCase.GetInt("blocked"), Escape: promotionCase.GetInt("escape"), Only: promotionCase.GetInt("only")},
			Completion: nogoods.Completion{XColor: promotionCase.GetInt("xColor"), YColor: promotionCase.GetInt("yColor")}, Conflict: proof.GetBool("conflict"),
		})
	}
	artifact.PromotionDigest = digestJSON(struct {
		Barrier string                 `json:"barrier"`
		Proofs  []FrozenPromotionProof `json:"proofs"`
	}{artifact.PromotionBarrier, artifact.PromotionProofs})
	artifact.ProvenanceDigest = digestJSON(struct {
		Authority string `json:"authority"`
		Evidence  string `json:"evidence"`
		Promotion string `json:"promotion"`
	}{artifact.Provenance, artifact.EvidenceBoundaryDigest, artifact.PromotionDigest})
	artifact.Digest = artifactDigest(artifact)
	encoded, _ := json.Marshal(artifact)
	if err := artifact.Validate(); err != nil {
		return FrozenArtifact{}, nil, ArtifactAuthority{}, err
	}
	authority := ArtifactAuthority{digest: artifact.Digest, evidenceBoundary: artifact.EvidenceBoundaryDigest, promotionDigest: artifact.PromotionDigest, provenanceDigest: artifact.ProvenanceDigest}
	return artifact, encoded, authority, nil
}

func ParseFrozenArtifact(data []byte) (FrozenArtifact, error) {
	var artifact FrozenArtifact
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&artifact); err != nil {
		return FrozenArtifact{}, err
	}
	canonical, _ := json.Marshal(artifact)
	if !slices.Equal(canonical, data) {
		return FrozenArtifact{}, fmt.Errorf("artifact encoding is not canonical")
	}
	return artifact, artifact.Validate()
}

func (artifact FrozenArtifact) Validate() error {
	if artifact.Name == "" || artifact.SchemaVersion == "" || artifact.GuardVersion == "" || artifact.Mask < 0 || artifact.Mask > 7 || artifact.Selection == "" || artifact.SelectionBarrier == "" || artifact.PromotionBarrier == "" || artifact.Provenance == "" {
		return fmt.Errorf("invalid frozen nogood artifact envelope")
	}
	for _, digest := range []string{artifact.SchemaSemanticKey, artifact.EvidenceBoundaryDigest, artifact.CompletionDigest, artifact.PromotionDigest, artifact.ProvenanceDigest, artifact.Digest} {
		if !validDigest(digest) {
			return fmt.Errorf("invalid frozen nogood artifact digest")
		}
	}
	if len(artifact.Candidates) != 8 || len(artifact.ExactTies) != 1 || !slices.Equal(artifact.ExpectedMasks, []int{0, 1, 2, 3, 4, 5, 6, 7}) || !slices.Equal(artifact.ActualMasks, artifact.ExpectedMasks) || len(artifact.PromotionProofs) != 24 || artifact.Digest != artifactDigest(artifact) {
		return fmt.Errorf("invalid frozen nogood artifact contents")
	}
	for mask, candidate := range artifact.Candidates {
		if candidate.Mask != mask || !candidate.EvidenceComplete || len(candidate.Evidence) != 4 || candidate.TrainingExact != (mask == 7) {
			return fmt.Errorf("invalid candidate evidence boundary")
		}
		for _, evidence := range candidate.Evidence {
			if evidence.Example == "" || evidence.Barrier == "" || !validDigest(evidence.ProblemDigest) || !slices.Equal(evidence.ExpectedKeys, evidence.ActualKeys) || len(evidence.Results) != len(evidence.ActualKeys) {
				return fmt.Errorf("invalid training evidence set")
			}
		}
	}
	proofs, cases := map[string]bool{}, map[string]bool{}
	roleTriples := map[[3]int]bool{}
	for _, proof := range artifact.PromotionProofs {
		triple := [3]int{proof.Binding.Blocked, proof.Binding.Escape, proof.Binding.Only}
		if proof.Proof == "" || proof.Case == "" || proofs[proof.Proof] || cases[proof.Case] || roleTriples[triple] || !proof.Conflict || !validDigest(proof.ProblemDigest) || triple[0] == triple[1] || triple[0] == triple[2] || triple[1] == triple[2] || proof.Completion.XColor != triple[2] || proof.Completion.YColor != triple[2] {
			return fmt.Errorf("invalid promotion proof set")
		}
		proofs[proof.Proof], cases[proof.Case], roleTriples[triple] = true, true, true
	}
	return nil
}

func (authority ArtifactAuthority) accepts(artifact FrozenArtifact) bool {
	return authority.digest != "" && artifact.Digest == authority.digest && artifact.EvidenceBoundaryDigest == authority.evidenceBoundary && artifact.PromotionDigest == authority.promotionDigest && artifact.ProvenanceDigest == authority.provenanceDigest
}

func installArtifact(store *unit.Store, artifact FrozenArtifact, authority *ArtifactAuthority) error {
	if err := artifact.Validate(); err != nil {
		return err
	}
	authoritative := authority != nil && authority.accepts(artifact)
	u := unit.New(artifact.Name)
	u.Set("isA", []string{"NogoodArtifact", "Anything"})
	u.Set("schemaVersion", artifact.SchemaVersion)
	u.Set("guardVersion", artifact.GuardVersion)
	u.Set("mask", artifact.Mask)
	u.Set("schemaSemanticKey", artifact.SchemaSemanticKey)
	u.Set("evidenceBoundaryDigest", artifact.EvidenceBoundaryDigest)
	u.Set("completionDigest", artifact.CompletionDigest)
	u.Set("promotionDigest", artifact.PromotionDigest)
	u.Set("provenance", artifact.Provenance)
	u.Set("provenanceDigest", artifact.ProvenanceDigest)
	u.Set("artifactDigest", artifact.Digest)
	u.Set("frozen", true)
	u.Set("authoritative", authoritative)
	store.Put(u)
	return nil
}

func artifactDigest(artifact FrozenArtifact) string {
	artifact.Digest = ""
	return digestJSON(artifact)
}

func digestJSON(value any) string {
	encoded, _ := json.Marshal(value)
	return digestBytes(encoded)
}

func digestBytes(value []byte) string {
	digest := sha256.Sum256(value)
	return hex.EncodeToString(digest[:])
}

func validDigest(value string) bool {
	if len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil && value == strings.ToLower(value)
}

func intSlot(u *unit.Unit, slot string) []int {
	values, _ := u.Get(slot).([]int)
	return slices.Clone(values)
}
