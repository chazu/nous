units: [
	{
		name:  "Conjecture"
		worth: 500
		isA: ["MathConcept", "Anything"]
		specializations: ["GoldbachConjecture"]
	},
	{
		name:    "GoldbachConjecture"
		worth:   400
		isA: ["Conjecture", "MathConcept", "Anything"]
		english: "Every even number greater than 2 is the sum of two primes"
		status:  "unverified"
	},
]
