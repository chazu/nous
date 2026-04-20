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
		isA: ["UnOrdStruc", "NoMultEleStruc", "Structure", "MathObj", "Anything"]
		english: "An unordered collection with no duplicate elements"
		specializations: ["EmptySet", "SetOfNumbers", "SetOfPrimes", "SetOfEvens", "OSet"]
		defn: #"""
			is-list?
			"""#
	},
	{
		name:    "List"
		worth:   600
		isA: ["OrdStruc", "MultEleStruc", "Structure", "MathObj", "Anything"]
		english: "An ordered collection that may contain duplicates"
		specializations: ["SortedList"]
		defn: #"""
			is-list?
			"""#
	},
	{
		name:  "Bag"
		worth: 500
		isA: ["UnOrdStruc", "MultEleStruc", "Structure", "MathObj", "Anything"]
		defn: #"""
			is-list?
			"""#
	},
	{
		name:    "OSet"
		worth:   600
		isA: ["OrdStruc", "NoMultEleStruc", "Set", "Structure", "MathObj", "Anything"]
		english: "An ordered collection with no duplicate elements"
		specializations: ["OSetOfNumbers", "OSetOfPrimesDesc"]
		defn: #"""
			is-list?
			"""#
	},
	{
		name:    "OrdStruc"
		worth:   500
		isA: ["Structure", "MathObj", "Anything"]
		english: "A structure whose elements have a definite order"
		specializations: ["OSet", "List"]
	},
	{
		name:    "UnOrdStruc"
		worth:   500
		isA: ["Structure", "MathObj", "Anything"]
		english: "A structure whose elements have no definite order"
		specializations: ["Set", "Bag"]
	},
	{
		name:    "MultEleStruc"
		worth:   500
		isA: ["Structure", "MathObj", "Anything"]
		english: "A structure that may contain duplicate elements"
		specializations: ["List", "Bag"]
	},
	{
		name:    "NoMultEleStruc"
		worth:   500
		isA: ["Structure", "MathObj", "Anything"]
		english: "A structure that rejects duplicate elements"
		specializations: ["Set", "OSet"]
	},
	{
		name:    "EmptyStruc"
		worth:   400
		isA: ["Structure", "MathObj", "Anything"]
		english: "A structure containing no elements"
		specializations: ["EmptySet"]
	},
	{
		name:    "NonEmptyStruc"
		worth:   400
		isA: ["Structure", "MathObj", "Anything"]
		english: "A structure containing at least one element"
		specializations: ["SetOfNumbers", "SetOfPrimes", "SetOfEvens", "SetOfOdds", "OSetOfNumbers", "OSetOfPrimesDesc"]
	},
]
