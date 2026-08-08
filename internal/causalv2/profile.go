package causalv2

import (
	"errors"
	"fmt"
	"slices"

	causal "github.com/chazu/nous/internal/vocab/causal"
)

const (
	ProfileDomain        = "causal-profile/v2"
	CentralProfileDomain = "causal-central-profile/v2"
)

type Profile struct {
	ProfileVersion  string   `json:"profile_version"`
	Manifest        Manifest `json:"manifest"`
	Panel           string   `json:"panel"`
	Seed            int64    `json:"seed"`
	AcquisitionCode string   `json:"acquisition_code"`
	FixtureDigest   string   `json:"fixture_digest"`
	ProfileDigest   string   `json:"profile_digest"`
}

func validAcquisition(code string) bool {
	if _, err := causal.ParseRule(code); err == nil {
		return true
	}
	return slices.Contains([]string{"lexical-fixed", "uniform-random", "passive-only", "dynamic-optimal"}, code)
}

func validateProfile(profile Profile) error {
	if profile.ProfileVersion != ProfileDomain {
		return fmt.Errorf("profile_version=%q, want %q", profile.ProfileVersion, ProfileDomain)
	}
	if err := ValidateManifest(profile.Manifest); err != nil {
		return err
	}
	if !panels[profile.Panel] {
		return fmt.Errorf("invalid panel %q", profile.Panel)
	}
	if !validAcquisition(profile.AcquisitionCode) {
		return fmt.Errorf("invalid acquisition code %q", profile.AcquisitionCode)
	}
	return requireDigest("fixture_digest", profile.FixtureDigest, false)
}

func profileDigest(profile Profile) (string, error) {
	profile.ProfileDigest = ""
	return Digest(ProfileDomain, profile)
}

func SignProfile(profile *Profile) error {
	if profile == nil {
		return errors.New("nil profile")
	}
	if err := validateProfile(*profile); err != nil {
		return err
	}
	digest, err := profileDigest(*profile)
	if err != nil {
		return err
	}
	profile.ProfileDigest = digest
	encoded, err := CanonicalJSON(profile)
	if err != nil {
		return err
	}
	return CheckByteCap(encoded, PreregisteredManifest().DescriptorByteCap)
}

func VerifyProfile(data []byte) (Profile, error) {
	profile, err := StrictDecode[Profile](data)
	if err != nil {
		return profile, err
	}
	if err := validateProfile(profile); err != nil {
		return profile, err
	}
	if err := requireDigest("profile_digest", profile.ProfileDigest, false); err != nil {
		return profile, err
	}
	want, err := profileDigest(profile)
	if err != nil {
		return profile, err
	}
	if profile.ProfileDigest != want {
		return profile, errors.New("profile digest mismatch")
	}
	if err := CheckByteCap(data, PreregisteredManifest().DescriptorByteCap); err != nil {
		return profile, err
	}
	return profile, nil
}

func VerifyProfileForFixture(data []byte, fixtureDigest string) (Profile, error) {
	profile, err := VerifyProfile(data)
	if err != nil {
		return profile, err
	}
	if profile.FixtureDigest != fixtureDigest {
		return profile, errors.New("profile fixture digest mismatch")
	}
	return profile, nil
}

type CentralProfile struct {
	CentralProfileVersion string   `json:"central_profile_version"`
	Manifest              Manifest `json:"manifest"`
	PlanCommit            string   `json:"plan_commit"`
	PretrainingCommit     string   `json:"pretraining_commit"`
	CreditEnabled         bool     `json:"credit_enabled"`
	TrainingKey           string   `json:"training_key"`
	ProfileDigest         string   `json:"profile_digest"`
}

func validCommit(commit string) bool {
	return len(commit) == 40 && validHex(commit)
}

func validHex(value string) bool {
	for _, digit := range value {
		if !(digit >= '0' && digit <= '9') && !(digit >= 'a' && digit <= 'f') {
			return false
		}
	}
	return true
}

func validateCentralProfile(profile CentralProfile) error {
	if profile.CentralProfileVersion != CentralProfileDomain {
		return errors.New("invalid central profile version")
	}
	if err := ValidateManifest(profile.Manifest); err != nil {
		return err
	}
	if !validCommit(profile.PlanCommit) || !validCommit(profile.PretrainingCommit) {
		return errors.New("central profile commits must be lowercase 40-hex")
	}
	wantKey := hashParts(CentralProfileDomain, profile.PlanCommit, profile.PretrainingCommit)
	if profile.TrainingKey != wantKey {
		return errors.New("central training key mismatch")
	}
	return nil
}

func centralProfileDigest(profile CentralProfile) (string, error) {
	profile.ProfileDigest = ""
	return Digest(CentralProfileDomain, profile)
}

func SignCentralProfile(profile *CentralProfile) error {
	if profile == nil {
		return errors.New("nil central profile")
	}
	profile.TrainingKey = hashParts(CentralProfileDomain, profile.PlanCommit, profile.PretrainingCommit)
	if err := validateCentralProfile(*profile); err != nil {
		return err
	}
	digest, err := centralProfileDigest(*profile)
	if err != nil {
		return err
	}
	profile.ProfileDigest = digest
	return nil
}

func VerifyCentralProfile(data []byte) (CentralProfile, error) {
	profile, err := StrictDecode[CentralProfile](data)
	if err != nil {
		return profile, err
	}
	if err := validateCentralProfile(profile); err != nil {
		return profile, err
	}
	if err := requireDigest("profile_digest", profile.ProfileDigest, false); err != nil {
		return profile, err
	}
	want, err := centralProfileDigest(profile)
	if err != nil {
		return profile, err
	}
	if profile.ProfileDigest != want {
		return profile, errors.New("central profile digest mismatch")
	}
	return profile, nil
}
