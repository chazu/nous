package nogoodexp

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math/rand/v2"

	"github.com/chazu/nous/internal/nogoodfixture"
	"github.com/chazu/nous/internal/nogoodfixturecore"
)

func validationPanel() ([]nogoodfixture.Task, error) {
	counts := [4]int{112, 48, 16, 16}
	cohorts := protectedCohorts()
	tasks := make([]nogoodfixture.Task, 0, ValidationTaskCount)
	global := 0
	for cohortIndex, cohort := range cohorts {
		for local := 0; local < counts[cohortIndex]; local++ {
			seed := 833001 + global
			constructed, err := nogoodfixturecore.Construct(global, cohort.core, local, func(purpose string) *rand.Rand {
				return protectedFixtureStream("validation", seed, 0, purpose)
			})
			if err != nil {
				return nil, err
			}
			tasks = append(tasks, nogoodfixture.Task{
				Panel: "validation", Ordinal: global, Seed: seed, Cohort: cohort.fixture, CohortOrdinal: local,
				Template: constructed.Template, MissingBit: constructed.MissingBit,
				ProblemJSON: constructed.ProblemJSON, Decision: constructed.Decision,
			})
			global++
		}
	}
	return tasks, nil
}

// lockedPanel is deliberately unexported and is called only by the one-shot
// guard after its durable claim. The generic constructor has no panel/root API.
func lockedPanel(root string) ([]nogoodfixture.Task, error) {
	if len(root) != 64 {
		return nil, fmt.Errorf("locked root must be 64 lowercase hex characters")
	}
	for _, c := range root {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return nil, fmt.Errorf("locked root must be 64 lowercase hex characters")
		}
	}
	counts := [4]int{312, 48, 12, 12}
	cohorts := protectedCohorts()
	tasks := make([]nogoodfixture.Task, 0, LockedTaskCount)
	global := 0
	for cohortIndex, cohort := range cohorts {
		for local := 0; local < counts[cohortIndex]; local++ {
			ordinal := global
			constructed, err := nogoodfixturecore.Construct(global, cohort.core, local, func(purpose string) *rand.Rand {
				return lockedFixtureStream(root, ordinal, purpose)
			})
			if err != nil {
				return nil, err
			}
			tasks = append(tasks, nogoodfixture.Task{
				Panel: "locked", Ordinal: global, Cohort: cohort.fixture, CohortOrdinal: local,
				Template: constructed.Template, MissingBit: constructed.MissingBit,
				ProblemJSON: constructed.ProblemJSON, Decision: constructed.Decision,
			})
			global++
		}
	}
	return tasks, nil
}

func lockedFixtureStream(root string, ordinal int, purpose string) *rand.Rand {
	return protectedFixtureStream("locked", root, ordinal, purpose)
}

func protectedFixtureStream(panel string, root any, ordinal int, purpose string) *rand.Rand {
	encoded, err := json.Marshal([]any{nogoodfixture.SeedAuthority, panel, root, ordinal, purpose})
	if err != nil {
		panic(err)
	}
	digest := sha256.Sum256(encoded)
	return rand.New(rand.NewPCG(binary.BigEndian.Uint64(digest[:8]), binary.BigEndian.Uint64(digest[8:16])))
}

func protectedCohorts() [4]struct {
	fixture nogoodfixture.Cohort
	core    nogoodfixturecore.Cohort
} {
	return [4]struct {
		fixture nogoodfixture.Cohort
		core    nogoodfixturecore.Cohort
	}{
		{nogoodfixture.Reusable, nogoodfixturecore.Reusable},
		{nogoodfixture.NearMiss, nogoodfixturecore.NearMiss},
		{nogoodfixture.Irrelevant, nogoodfixturecore.Irrelevant},
		{nogoodfixture.IndependentUnsat, nogoodfixturecore.IndependentUnsat},
	}
}
