units: [
	{
		name:    "EmptySet"
		worth:   400
		isA: ["Set", "Structure", "MathObj", "Anything"]
		english: "The set with no elements"
		data: []
	},
	{
		name:    "SetOfNumbers"
		worth:   600
		isA: ["Set", "Structure", "MathObj", "Anything"]
		english: "The integers from 1 to 20"
		data: [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20]
		specializations: ["SetOfPrimes", "SetOfEvens", "SetOfOdds"]
	},
	{
		name:    "SetOfPrimes"
		worth:   600
		isA: ["Set", "SetOfNumbers", "Structure", "MathObj", "Anything"]
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
		isA: ["Set", "SetOfNumbers", "Structure", "MathObj", "Anything"]
		english: "Even numbers up to 20"
		data: [2, 4, 6, 8, 10, 12, 14, 16, 18, 20]
		generalizations: ["SetOfNumbers"]
	},
	{
		name:    "SetOfOdds"
		worth:   500
		isA: ["Set", "SetOfNumbers", "Structure", "MathObj", "Anything"]
		english: "Odd numbers up to 20"
		data: [1, 3, 5, 7, 9, 11, 13, 15, 17, 19]
		generalizations: ["SetOfNumbers"]
	},
	{
		name:  "SortedList"
		worth: 400
		isA: ["List", "Structure", "MathObj", "Anything"]
	},
	{
		name:    "OSetOfNumbers"
		worth:   500
		isA: ["OSet", "Set", "Structure", "MathObj", "Anything"]
		english: "The integers from 1 to 20 in ascending order"
		data: [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20]
	},
	{
		name:    "OSetOfPrimesDesc"
		worth:   500
		isA: ["OSet", "Set", "Structure", "MathObj", "Anything"]
		english: "Primes under 20 in descending order"
		data: [19, 17, 13, 11, 7, 5, 3, 2]
	},
]
