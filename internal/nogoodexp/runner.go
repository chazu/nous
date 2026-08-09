package nogoodexp

import (
	"fmt"
	"slices"

	"github.com/chazu/nous/internal/nogoodbaseline"
	"github.com/chazu/nous/internal/nogoodfixture"
	"github.com/chazu/nous/internal/nogoodoracle"
)

var RequiredPolicies = []string{
	"chronological", "forward-checking", "mac-cbj", "mac-cbj-empty",
	"no-artifact", "reset", "concrete-memo", "nous-generalized",
	"recomputed", "corrupted", "wrong-family", "random", "match-only",
}

type TaskOutcome struct {
	Ordinal     int       `json:"ordinal"`
	Cohort      string    `json:"cohort"`
	Satisfied   bool      `json:"satisfied"`
	Witness     []int     `json:"witness,omitempty"`
	Disposition string    `json:"disposition,omitempty"`
	Work        int64     `json:"work"`
	Vector      [12]int64 `json:"vector"`
	PruneSound  bool      `json:"prune_sound"`
}

type PolicyExecution struct {
	Policy     string           `json:"policy"`
	Tasks      []TaskOutcome    `json:"tasks"`
	Transcript TranscriptBundle `json:"-"`
}

type PanelExecution struct {
	Role              string            `json:"role"`
	Panel             string            `json:"panel"`
	AcquisitionWork   int64             `json:"acquisition_work"`
	AcquisitionVector [12]int64         `json:"acquisition_vector"`
	Policies          []PolicyExecution `json:"policies"`
}

func RunDevelopmentExecution(domainsDir, role string) (PanelExecution, error) {
	tasks, err := nogoodfixture.Panel("development")
	if err != nil {
		return PanelExecution{}, err
	}
	return runPanelExecution(domainsDir, role, "development", tasks)
}

func runPanelExecution(domainsDir, role, panel string, tasks []nogoodfixture.Task) (PanelExecution, error) {
	training, err := RunTraining(domainsDir)
	if err != nil {
		return PanelExecution{}, err
	}
	artifact, _, authority, err := FreezeArtifact(training)
	if err != nil {
		return PanelExecution{}, err
	}
	corrupted := artifact
	corrupted.Mask = 5
	corrupted.Digest = artifactDigest(corrupted)
	random := artifact
	random.Mask = 3
	random.Digest = artifactDigest(random)
	wrongFamily := artifact
	wrongFamily.SchemaVersion = "blocked-pair-three-color/v1"
	wrongFamily.Digest = artifactDigest(wrongFamily)

	bridges := map[string]*BridgeExecution{}
	bridgeInputs := map[string]struct {
		artifact  *FrozenArtifact
		authority *ArtifactAuthority
	}{
		"mac-cbj-empty":    {},
		"no-artifact":      {},
		"reset":            {},
		"concrete-memo":    {},
		"nous-generalized": {&artifact, &authority},
		"corrupted":        {&corrupted, &authority},
		"wrong-family":     {&wrongFamily, &authority},
		"random":           {&random, &authority},
		"match-only":       {&artifact, &authority},
	}
	for policy, input := range bridgeInputs {
		bridge, err := NewBridgeExecution(domainsDir, input.artifact, input.authority)
		if err != nil {
			return PanelExecution{}, err
		}
		bridges[policy] = bridge
	}
	acquisition := acquisitionTranscript(training, bridges["nous-generalized"].preflight)
	if work := transcriptWork(acquisition); work > 2000 {
		return PanelExecution{}, fmt.Errorf("acquisition work %d exceeds cap", work)
	}

	execution := PanelExecution{Role: role, Panel: panel, AcquisitionWork: transcriptWork(acquisition), AcquisitionVector: transcriptVector(acquisition)}
	for _, policy := range RequiredPolicies {
		policyExecution := PolicyExecution{Policy: policy}
		var policyEvents []TranscriptEvent
		if policy == "nous-generalized" {
			policyEvents = appendEvents(policyEvents, acquisition)
		}
		for _, task := range tasks {
			outcome, events, err := runPolicyTask(domainsDir, policy, task, artifact, authority, bridges[policy])
			if err != nil {
				return PanelExecution{}, fmt.Errorf("%s task %d: %w", policy, task.Ordinal, err)
			}
			policyExecution.Tasks = append(policyExecution.Tasks, outcome)
			policyEvents = appendEvents(policyEvents, events)
		}
		bundle, err := EncodeTranscript(policyEvents)
		if err != nil {
			return PanelExecution{}, fmt.Errorf("encode %s transcript: %w", policy, err)
		}
		decoded, err := DecodeTranscript(bundle.Raw)
		if err != nil || decoded.Vector != bundle.Vector {
			return PanelExecution{}, fmt.Errorf("%s transcript conservation failed: %v", policy, err)
		}
		policyExecution.Transcript = bundle
		execution.Policies = append(execution.Policies, policyExecution)
	}
	return execution, nil
}

func runPolicyTask(domainsDir, policy string, task nogoodfixture.Task, artifact FrozenArtifact, authority ArtifactAuthority, bridge *BridgeExecution) (TaskOutcome, []TranscriptEvent, error) {
	decision := nogoodbaseline.Literal{Variable: task.Decision.Variable, Color: task.Decision.Color}
	oracle, err := nogoodoracle.Enumerate(task.ProblemJSON, nogoodoracle.Literal(decision))
	if err != nil {
		return TaskOutcome{}, nil, err
	}
	var result nogoodbaseline.Result
	var disposition string
	var events []TranscriptEvent
	switch policy {
	case "chronological":
		result, err = nogoodbaseline.Chronological(task.ProblemJSON, decision)
	case "forward-checking":
		result, err = nogoodbaseline.ForwardChecking(task.ProblemJSON, decision)
	case "mac-cbj":
		result, err = nogoodbaseline.MACCBJ(task.ProblemJSON, decision)
	case "recomputed":
		training, trainErr := RunTraining(domainsDir)
		if trainErr != nil {
			return TaskOutcome{}, nil, trainErr
		}
		freshArtifact, _, freshAuthority, freezeErr := FreezeArtifact(training)
		if freezeErr != nil {
			return TaskOutcome{}, nil, freezeErr
		}
		freshBridge, bridgeErr := NewBridgeExecution(domainsDir, &freshArtifact, &freshAuthority)
		if bridgeErr != nil {
			return TaskOutcome{}, nil, bridgeErr
		}
		d, bridgeErr := freshBridge.Consider(task.ProblemJSON, task.Decision)
		if bridgeErr != nil {
			return TaskOutcome{}, nil, bridgeErr
		}
		disposition = d.Status
		localAcquisition := acquisitionTranscript(training, freshBridge.preflight)
		for index := range localAcquisition {
			localAcquisition[index].TaskOrdinal = uint32(task.Ordinal)
		}
		events = appendEvents(events, localAcquisition)
		events = appendEvents(events, bridgeTranscript(uint32(task.Ordinal), d))
		if d.Status == "propose-prune" {
			result = nogoodbaseline.Result{Satisfied: false}
		} else {
			result, err = nogoodbaseline.MACCBJ(task.ProblemJSON, decision)
		}
	default:
		if bridge == nil {
			return TaskOutcome{}, nil, fmt.Errorf("missing bridge execution")
		}
		d, bridgeErr := bridge.Consider(task.ProblemJSON, task.Decision)
		if bridgeErr != nil {
			return TaskOutcome{}, nil, bridgeErr
		}
		disposition = d.Status
		events = appendEvents(events, bridgeTranscript(uint32(task.Ordinal), d))
		prune := d.Status == "propose-prune" && policy == "nous-generalized"
		if prune {
			result = nogoodbaseline.Result{Satisfied: false}
		} else {
			result, err = nogoodbaseline.MACCBJ(task.ProblemJSON, decision)
		}
	}
	if err != nil {
		return TaskOutcome{}, nil, err
	}
	if policy == "chronological" || policy == "forward-checking" || policy == "mac-cbj" {
		events, err = baselineTranscript(uint32(task.Ordinal), result)
	} else if result.Events != nil {
		baselineEvents, mapErr := baselineTranscript(uint32(task.Ordinal), result)
		if mapErr != nil {
			return TaskOutcome{}, nil, mapErr
		}
		events = appendEvents(events, baselineEvents)
	}
	if err != nil {
		return TaskOutcome{}, nil, err
	}
	if result.Satisfied != oracle.Satisfiable || result.Satisfied && !containsOracleWitness(oracle.Solutions, result.Witness) {
		return TaskOutcome{}, nil, fmt.Errorf("oracle parity failed")
	}
	pruneSound := disposition != "propose-prune" || !oracle.Satisfiable
	if !pruneSound {
		return TaskOutcome{}, nil, fmt.Errorf("unsound omitted branch")
	}
	vector := transcriptVector(events)
	work := transcriptWork(events)
	if disposition == "propose-prune" && policy == "nous-generalized" && work > 128 {
		return TaskOutcome{}, nil, fmt.Errorf("learned prune work %d exceeds hard cap", work)
	}
	return TaskOutcome{Ordinal: task.Ordinal, Cohort: string(task.Cohort), Satisfied: result.Satisfied, Witness: slices.Clone(result.Witness), Disposition: disposition, Work: work, Vector: vector, PruneSound: pruneSound}, events, nil
}

func containsOracleWitness(solutions [][]int, witness []int) bool {
	return slices.ContainsFunc(solutions, func(solution []int) bool { return slices.Equal(solution, witness) })
}
