package causalrun

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/chazu/nous/internal/causalv2"
)

type replayResponse struct {
	action  string
	outcome string
}

type replayTeacher struct {
	token     string
	responses []replayResponse
	next      int
}

func (t *replayTeacher) Respond(token, action string) (string, error) {
	if token != t.token {
		return "", errors.New("replay teacher received wrong token")
	}
	if t.next >= len(t.responses) {
		return "", errors.New("production requested an unrecorded teacher response")
	}
	response := t.responses[t.next]
	if response.action != action {
		return "", fmt.Errorf("replay action=%q, production selected %q", response.action, action)
	}
	t.next++
	return response.outcome, nil
}

// VerifyEpisode replays canonical public evidence in a fresh isolated runner.
// It never mutates the supplied bytes and requires every regenerated artifact
// byte, charge index, authorization, counter-derived budget snapshot, and
// transcript link to match exactly.
func VerifyEpisode(publicFixtureBytes, profileBytes []byte, artifacts [][]byte) (EpisodeResult, error) {
	profile, err := causalv2.VerifyProfile(profileBytes)
	if err != nil {
		return EpisodeResult{}, err
	}
	fixture, err := causalv2.VerifyPublicFixtureForPanel(publicFixtureBytes, profile.Panel)
	if err != nil {
		return EpisodeResult{}, err
	}
	if profile.FixtureDigest != fixture.FixtureDigest || profile.Seed != fixture.Seed {
		return EpisodeResult{}, errors.New("replay profile/fixture context mismatch")
	}
	var responses []replayResponse
	for index, encoded := range artifacts {
		artifact, err := causalv2.VerifyArtifact(encoded)
		if err != nil {
			return EpisodeResult{}, fmt.Errorf("artifact %d: %w", index, err)
		}
		if artifact.ProfileDigest != profile.ProfileDigest || artifact.Scope != profile.ProfileDigest || artifact.ChargeIndex != index {
			return EpisodeResult{}, fmt.Errorf("artifact %d context or charge order mismatch", index)
		}
		if artifact.Kind == "result" {
			payload, err := causalv2.StrictDecode[causalv2.ResultPayload](artifact.Payload)
			if err != nil {
				return EpisodeResult{}, err
			}
			responses = append(responses, replayResponse{action: payload.Action, outcome: payload.Outcome})
		}
	}
	teacher := &replayTeacher{token: fixture.OpaqueToken, responses: responses}
	runner, err := NewEpisode(publicFixtureBytes, profileBytes, teacher)
	if err != nil {
		return EpisodeResult{}, err
	}
	defer runner.Close()
	result, err := runner.Run(context.Background())
	if err != nil {
		return EpisodeResult{}, err
	}
	if teacher.next != len(responses) {
		return EpisodeResult{}, errors.New("evidence contains unused result artifacts")
	}
	regenerated := runner.ArtifactBytes()
	if len(regenerated) != len(artifacts) {
		return EpisodeResult{}, fmt.Errorf("artifact count=%d, regenerated %d", len(artifacts), len(regenerated))
	}
	for index := range artifacts {
		if !bytes.Equal(artifacts[index], regenerated[index]) {
			return EpisodeResult{}, fmt.Errorf("artifact %d differs from fresh replay", index)
		}
	}
	return result, nil
}
