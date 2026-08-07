// A deliberately small second vocabulary. Build graphs are represented as
// sets of canonical "consumer>dependency" edge strings.
units: [
	{
		name:  "BuildConcept"
		worth: 500
		isA: ["Anything"]
	},
	{
		name:  "BuildGraph"
		worth: 700
		isA: ["BuildConcept", "Anything"]
		english: "A set of directed consumer>dependency edges"
		specializations: ["ApplicationGraph", "LibraryGraph"]
	},
	{
		name:  "ApplicationGraph"
		worth: 600
		isA: ["BuildGraph", "BuildConcept", "Anything"]
	},
	{
		name:  "LibraryGraph"
		worth: 600
		isA: ["BuildGraph", "BuildConcept", "Anything"]
	},
	{
		name:  "GraphOp"
		worth: 600
		isA: ["Op", "BuildConcept", "Anything"]
	},
	{
		name:  "GraphPred"
		worth: 600
		isA: ["Pred", "BuildConcept", "Anything"]
	},
	{
		name:  "FrontendGraph"
		worth: 650
		isA: ["ApplicationGraph", "BuildGraph", "BuildConcept", "Anything"]
		english: "Frontend service and its direct build dependencies"
		data: ["web>api", "web>ui", "ui>design-system"]
	},
	{
		name:  "BackendGraph"
		worth: 650
		isA: ["ApplicationGraph", "BuildGraph", "BuildConcept", "Anything"]
		english: "Backend service and its direct build dependencies"
		data: ["api>core", "api>db", "db>schema"]
	},
	{
		name:  "SharedLibraryGraph"
		worth: 600
		isA: ["LibraryGraph", "BuildGraph", "BuildConcept", "Anything"]
		english: "Dependencies shared by application graphs"
		data: ["api>core", "ui>design-system"]
	},
]
