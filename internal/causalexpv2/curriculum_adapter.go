package causalexpv2

import (
	"context"

	"github.com/chazu/nous/internal/causalcurriculum"
)

func init() {
	centralCurriculumAdapter = func(ctx context.Context, centralProfileBytes []byte, episodeBytes, certificateBytes [][]byte) (curriculumOutcome, error) {
		result, err := causalcurriculum.Run(ctx, centralProfileBytes, episodeBytes, certificateBytes)
		if err != nil {
			return curriculumOutcome{}, err
		}
		if err := causalcurriculum.Verify(centralProfileBytes, episodeBytes, certificateBytes, result); err != nil {
			return curriculumOutcome{}, err
		}
		return curriculumOutcome{Applications: result.Applications, Aggregates: result.Aggregates, WinnerTies: result.WinnerTies, SelectedRule: result.SelectedRule, Unresolved: result.Unresolved, Counts: result.Counts, TaskMeterItems: result.TaskMeterItems, TerminalTranscriptDigest: result.TerminalTranscriptDigest}, nil
	}
}
