units: [
	{
		name:    "EmptySet"
		worth:   400
		isA: ["EmptyStruc", "Set", "Structure", "MathObj", "Anything"]
		english: "The set with no elements"
		data: []
	},
	{
		name:    "SetOfNumbers"
		worth:   600
		isA: ["UnOrdStruc", "NoMultEleStruc", "NonEmptyStruc", "Set", "Structure", "MathObj", "Anything"]
		english: "The integers from 1 to 20"
		data: [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20]
		specializations: ["SetOfPrimes", "SetOfEvens", "SetOfOdds"]
	},
	{
		name:    "SetOfPrimes"
		worth:   600
		isA: ["UnOrdStruc", "NoMultEleStruc", "NonEmptyStruc", "Set", "SetOfNumbers", "Structure", "MathObj", "Anything"]
		english: "Prime numbers up to 20"
		data: [2, 3, 5, 7, 11, 13, 17, 19]
		defn: #"""
			# Filter: keep only primes
			each it prime? if it then end make-set
			"""#
		generalizations: ["SetOfNumbers"]
	},
	{
		name:  "SetOfEvens"
		worth: 600
		isA: ["UnOrdStruc", "NoMultEleStruc", "NonEmptyStruc", "Set", "SetOfNumbers", "Structure", "MathObj", "Anything"]
		english: "Even numbers up to 20"
		data: [2, 4, 6, 8, 10, 12, 14, 16, 18, 20]
		generalizations: ["SetOfNumbers"]
	},
	{
		name:    "SetOfOdds"
		worth:   500
		isA: ["UnOrdStruc", "NoMultEleStruc", "NonEmptyStruc", "Set", "SetOfNumbers", "Structure", "MathObj", "Anything"]
		english: "Odd numbers up to 20"
		data: [1, 3, 5, 7, 9, 11, 13, 15, 17, 19]
		generalizations: ["SetOfNumbers"]
	},
	{
		name:  "SortedList"
		worth: 400
		isA: ["OrdStruc", "MultEleStruc", "List", "Structure", "MathObj", "Anything"]
	},
	{
		name:    "OSetOfNumbers"
		worth:   500
		isA: ["OrdStruc", "NoMultEleStruc", "NonEmptyStruc", "OSet", "Set", "Structure", "MathObj", "Anything"]
		english: "The integers from 1 to 20 in ascending order"
		data: [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20]
	},
	{
		name:    "OSetOfPrimesDesc"
		worth:   500
		isA: ["OrdStruc", "NoMultEleStruc", "NonEmptyStruc", "OSet", "Set", "Structure", "MathObj", "Anything"]
		english: "Primes under 20 in descending order"
		data: [19, 17, 13, 11, 7, 5, 3, 2]
	},
	{
		name:    "BagOfTallies"
		worth:   500
		isA: ["MultEleStruc", "UnOrdStruc", "NonEmptyStruc", "Bag", "Structure", "MathObj", "Anything"]
		english: "A bag with duplicate elements — a canonical multiset for mutation experiments"
		data: [1, 1, 2, 2, 2, 3, 5]
		examples: ["Bag-ex-tally-a", "Bag-ex-tally-b", "Bag-ex-tally-c"]
		initialTasks: [{
			priority: 700
			slot: "examples"
			reason: "Bootstrap H29 on a pre-populated MultEleStruc"
		}]
	},
	{
		name:    "Bag-ex-tally-a"
		worth:   300
		isA: ["Bag", "Structure", "MathObj", "Anything"]
		data: [1, 1, 2, 3]
	},
	{
		name:    "Bag-ex-tally-b"
		worth:   300
		isA: ["Bag", "Structure", "MathObj", "Anything"]
		data: [2, 2, 4, 5, 5]
	},
	{
		name:    "Bag-ex-tally-c"
		worth:   300
		isA: ["Bag", "Structure", "MathObj", "Anything"]
		data: [3, 3, 3, 7]
	},
]
