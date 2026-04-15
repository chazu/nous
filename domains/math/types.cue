units: [
	{
		name:  "Structure"
		worth: 600
		isA: ["MathObj", "Anything"]
		specializations: ["Set", "List", "Bag"]
	},
	{
		name:    "Set"
		worth:   700
		isA: ["Structure", "MathObj", "Anything"]
		english: "An unordered collection with no duplicate elements"
		specializations: ["EmptySet", "SetOfNumbers", "SetOfPrimes", "SetOfEvens"]
	},
	{
		name:    "List"
		worth:   600
		isA: ["Structure", "MathObj", "Anything"]
		english: "An ordered collection that may contain duplicates"
		specializations: ["SortedList"]
	},
	{
		name:  "Bag"
		worth: 500
		isA: ["Structure", "MathObj", "Anything"]
	},
]
