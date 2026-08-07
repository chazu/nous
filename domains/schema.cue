package domains

#Unit: {
	name:  string
	worth: int & >=0 & <=1000

	isA: [...string]

	english?:         string
	abbrev?:          string
	domain?:          [...string] | string
	range?:           [...string] | string
	arity?:           int
	defn?:            string
	data?:            string | [...int] | [...string]
	examples?:        _
	nonExamples?:     _
	generalizations?: [...string]
	specializations?: [...string]
	creditors?:       [...string]
	inverse?:         string
	status?:          string
	overallRecord?: {successes: int, failures: int}

	ifPotentiallyRelevant?:   string
	ifTrulyRelevant?:         string
	ifWorkingOnTask?:         string
	ifFinishedWorkingOnTask?: string
	thenCompute?:             string
	thenAddToAgenda?:         string
	thenDefineNewConcepts?:   string
	thenDeleteOldConcepts?:   string
	thenPrintToUser?:         string
	thenConjecture?:          string
	thenModifySlots?:         string

	...
}
