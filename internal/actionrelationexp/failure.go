package actionrelationexp

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// InvalidAuthority is a canonical append-only placeholder for an authority
// document that could not be constructed before an invalid terminal receipt.
// It is legal only as a target of that invalid receipt, never in a publication.
type InvalidAuthority struct {
	Panel             string
	Kind              string
	SourceRoot        string
	AttemptCommitment string
	Reason            string
	Canonical         []byte
	Digest            string
}

func BuildInvalidAuthority(value InvalidAuthority) (InvalidAuthority, error) {
	value.Canonical, value.Digest = nil, ""
	canonical, err := invalidAuthorityCanonical(value)
	if err != nil {
		return InvalidAuthority{}, err
	}
	value.Canonical, value.Digest = canonical, shaHex(canonical)
	return value, VerifyInvalidAuthority(value)
}

func ParseInvalidAuthority(data []byte) (InvalidAuthority, error) {
	var fields []json.RawMessage
	var version, terminal string
	value := InvalidAuthority{Canonical: bytes.Clone(data), Digest: shaHex(data)}
	if len(data) > 8192 || json.Unmarshal(data, &fields) != nil || len(fields) != 7 || json.Unmarshal(fields[0], &version) != nil || version != "actionrelation-invalid-authority/v1" || json.Unmarshal(fields[1], &value.Panel) != nil || json.Unmarshal(fields[2], &value.Kind) != nil || json.Unmarshal(fields[3], &value.SourceRoot) != nil || json.Unmarshal(fields[4], &value.AttemptCommitment) != nil || json.Unmarshal(fields[5], &value.Reason) != nil || json.Unmarshal(fields[6], &terminal) != nil || terminal != "invalid" || VerifyInvalidAuthority(value) != nil {
		return InvalidAuthority{}, fmt.Errorf("failure authority does not reconstruct")
	}
	return value, nil
}

func VerifyInvalidAuthority(value InvalidAuthority) error {
	canonical, err := invalidAuthorityCanonical(value)
	if err != nil || !bytes.Equal(canonical, value.Canonical) || value.Digest != shaHex(value.Canonical) {
		return fmt.Errorf("invalid failure authority")
	}
	return nil
}

func invalidAuthorityCanonical(value InvalidAuthority) ([]byte, error) {
	if !panelNames[value.Panel] || value.Kind != "fixture-root" && value.Kind != "evidence-payload" && value.Kind != "report" || !digestText(value.SourceRoot) || !digestText(value.AttemptCommitment) || !boundedPrintableASCII(value.Reason, 1024) {
		return nil, fmt.Errorf("invalid failure authority fields")
	}
	if value.Panel == "development" && value.AttemptCommitment != zeroAuthorityDigest {
		return nil, fmt.Errorf("development failure has protected attempt authority")
	}
	return json.Marshal([]any{"actionrelation-invalid-authority/v1", value.Panel, value.Kind, value.SourceRoot, value.AttemptCommitment, value.Reason, "invalid"})
}
