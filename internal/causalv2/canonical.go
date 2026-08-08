// Package causalv2 implements the canonical proof primitives for the accepted
// active-causal-diagnosis/v2 experiment. It deliberately contains no seed
// generation, online runner, or evidence publication capability.
package causalv2

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

const (
	SHA256HexLength = 64
	ZeroDigest      = "0000000000000000000000000000000000000000000000000000000000000000"
)

// CanonicalJSON encodes compact JSON with HTML escaping disabled and no
// trailing newline.
func CanonicalJSON(value any) ([]byte, error) {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		return nil, err
	}
	encoded := buf.Bytes()
	if len(encoded) == 0 || encoded[len(encoded)-1] != '\n' {
		return nil, errors.New("canonical encoder omitted terminal newline")
	}
	return append([]byte(nil), encoded[:len(encoded)-1]...), nil
}

// StrictDecode rejects unknown fields, trailing values, and every encoding
// which does not byte-for-byte equal the canonical re-encoding.
func StrictDecode[T any](data []byte) (T, error) {
	var value T
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value); err != nil {
		return value, err
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return value, errors.New("trailing JSON value")
		}
		return value, fmt.Errorf("trailing JSON: %w", err)
	}
	canonical, err := CanonicalJSON(value)
	if err != nil {
		return value, err
	}
	if !bytes.Equal(data, canonical) {
		return value, errors.New("noncanonical JSON encoding")
	}
	return value, nil
}

// Digest returns SHA-256(domain || NUL || canonical-json(value)).
func Digest(domain string, value any) (string, error) {
	if domain == "" {
		return "", errors.New("empty digest domain")
	}
	encoded, err := CanonicalJSON(value)
	if err != nil {
		return "", err
	}
	h := sha256.New()
	_, _ = h.Write([]byte(domain))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write(encoded)
	return hex.EncodeToString(h.Sum(nil)), nil
}

func validDigest(value string, allowEmpty bool) bool {
	if allowEmpty && value == "" {
		return true
	}
	if len(value) != SHA256HexLength {
		return false
	}
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == sha256.Size && value == hex.EncodeToString(decoded)
}

func requireDigest(field, value string, allowEmpty bool) error {
	if !validDigest(value, allowEmpty) {
		return fmt.Errorf("%s is not canonical lowercase SHA-256 hex", field)
	}
	return nil
}

func requireCanonicalRaw(raw json.RawMessage) error {
	if len(raw) == 0 {
		return errors.New("empty canonical payload")
	}
	var value any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return errors.New("payload has trailing JSON")
	}
	encoded, err := CanonicalJSON(value)
	if err != nil {
		return err
	}
	if !bytes.Equal(encoded, raw) {
		return errors.New("noncanonical payload JSON")
	}
	return nil
}

func equalCanonical[T comparable](got, want T, field string) error {
	if got != want {
		return fmt.Errorf("%s=%v, want %v", field, got, want)
	}
	return nil
}
