units: [
	{
		name:  "Number"
		worth: 600
		isA: ["MathObj", "Anything"]
		specializations: ["EvenNum", "OddNum", "PrimeNum", "PerfectNum", "SquareNum"]
	},
	{
		name:  "EvenNum"
		worth: 400
		isA: ["Number", "MathObj", "Anything"]
		defn: #"""
			even?
			"""#
		examples: [2, 4, 6, 8, 10]
	},
	{
		name:  "OddNum"
		worth: 400
		isA: ["Number", "MathObj", "Anything"]
		defn: #"""
			odd?
			"""#
		examples: [1, 3, 5, 7, 9]
	},
	{
		name:    "PrimeNum"
		worth:   600
		isA: ["Number", "MathObj", "Anything"]
		english: "A number with no divisors other than 1 and itself"
		defn:    #"""
			prime?
			"""#
		examples: [2, 3, 5, 7, 11, 13, 17, 19, 23, 29]
	},
	{
		name:    "PerfectNum"
		worth:   500
		isA: ["Number", "MathObj", "Anything"]
		english: "A number equal to the sum of its proper divisors"
		examples: [6, 28, 496]
	},
	{
		name:  "SquareNum"
		worth: 400
		isA: ["Number", "MathObj", "Anything"]
		examples: [1, 4, 9, 16, 25, 36]
	},
	{
		name:  "TruthValue"
		worth: 400
		isA: ["MathObj", "Anything"]
	},
]
