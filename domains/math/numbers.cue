// Number types and concrete integer instances.
// EURISKO represented small integers as individual concept units so operations
// like DivisorsOf and GCD could iterate them via store.Examples("Number").
// The N-<i> units below are those instances — each isA the Number subtypes
// it belongs to (EvenNum/OddNum/PrimeNum/PerfectNum/SquareNum) plus Number
// itself, with data = the integer.
units: [
	{
		name:  "Number"
		worth: 600
		isA: ["MathObj", "Anything"]
		specializations: ["EvenNum", "OddNum", "PrimeNum", "PerfectNum", "SquareNum"]
		defn: #"""
			is-int?
			"""#
		generator: {
			initial: [0]
			step:    "1 +"
		}
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

	// Concrete integer instances.
	// isA lists every Number subtype the integer satisfies; store.IsA walks
	// isA chains so Number is reachable transitively through any subtype.
	{name: "N-1",  worth: 400, isA: ["OddNum", "SquareNum"],          data: 1},
	{name: "N-2",  worth: 400, isA: ["EvenNum", "PrimeNum"],           data: 2},
	{name: "N-3",  worth: 400, isA: ["OddNum", "PrimeNum"],            data: 3},
	{name: "N-4",  worth: 400, isA: ["EvenNum", "SquareNum"],          data: 4},
	{name: "N-5",  worth: 400, isA: ["OddNum", "PrimeNum"],            data: 5},
	{name: "N-6",  worth: 400, isA: ["EvenNum", "PerfectNum"],         data: 6},
	{name: "N-7",  worth: 400, isA: ["OddNum", "PrimeNum"],            data: 7},
	{name: "N-8",  worth: 400, isA: ["EvenNum"],                       data: 8},
	{name: "N-9",  worth: 400, isA: ["OddNum", "SquareNum"],           data: 9},
	{name: "N-10", worth: 400, isA: ["EvenNum"],                       data: 10},
	{name: "N-11", worth: 400, isA: ["OddNum", "PrimeNum"],            data: 11},
	{name: "N-12", worth: 400, isA: ["EvenNum"],                       data: 12},
	{name: "N-13", worth: 400, isA: ["OddNum", "PrimeNum"],            data: 13},
	{name: "N-14", worth: 400, isA: ["EvenNum"],                       data: 14},
	{name: "N-15", worth: 400, isA: ["OddNum"],                        data: 15},
	{name: "N-16", worth: 400, isA: ["EvenNum", "SquareNum"],          data: 16},
	{name: "N-17", worth: 400, isA: ["OddNum", "PrimeNum"],            data: 17},
	{name: "N-18", worth: 400, isA: ["EvenNum"],                       data: 18},
	{name: "N-19", worth: 400, isA: ["OddNum", "PrimeNum"],            data: 19},
	{name: "N-20", worth: 400, isA: ["EvenNum"],                       data: 20},
]
