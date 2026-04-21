// Per-type Op category units (Phase 5.5).
// Mirrors EURISKO's StrucOp / SetOp / BagOp / ListOp / OSetOp / MultEleStrucOp /
// OrdStrucOp hierarchy. Each is a Category unit whose `generalizations` list
// chains toward the root (Op, MathOp, MathConcept, Anything). Concrete ops
// gain the appropriate per-type category in their `isA` list; store.IsA walks
// both isA and generalizations, so isA("SetUnion", "StrucOp") resolves via
// SetUnion.isA ⊇ SetOp → SetOp.generalizations ⊇ StrucOp.
//
// EURISKO excerpts (EURUNITS):
//   StrucOp ⊃ SetOp | BagOp | ListOp | OSetOp | MultEleStrucOp | OrdStrucOp | UnitOp
//   MultEleStrucOp ⊃ ListOp | BagOp
//   OrdStrucOp ⊃ ListOp | OSetOp
units: [
	{
		name:    "StrucOp"
		worth:   500
		isA: ["Category", "MathConcept", "MathObj", "Anything"]
		english: "Category: operations on Structures"
		generalizations: ["MathConcept", "Op", "MathOp", "Anything"]
	},
	{
		name:    "SetOp"
		worth:   500
		isA: ["Category", "MathConcept", "MathObj", "Anything"]
		english: "Category: operations on Sets"
		generalizations: ["StrucOp", "MathConcept", "Op", "MathOp", "Anything"]
	},
	{
		name:    "BagOp"
		worth:   500
		isA: ["Category", "MathConcept", "MathObj", "Anything"]
		english: "Category: operations on Bags"
		generalizations: ["MultEleStrucOp", "StrucOp", "MathConcept", "Op", "MathOp", "Anything"]
	},
	{
		name:    "ListOp"
		worth:   500
		isA: ["Category", "MathConcept", "MathObj", "Anything"]
		english: "Category: operations on Lists"
		generalizations: ["MultEleStrucOp", "OrdStrucOp", "StrucOp", "MathConcept", "Op", "MathOp", "Anything"]
	},
	{
		name:    "OSetOp"
		worth:   500
		isA: ["Category", "MathConcept", "MathObj", "Anything"]
		english: "Category: operations on OSets (ordered sets)"
		generalizations: ["OrdStrucOp", "StrucOp", "MathConcept", "Op", "MathOp", "Anything"]
	},
	{
		name:    "MultEleStrucOp"
		worth:   500
		isA: ["Category", "MathConcept", "MathObj", "Anything"]
		english: "Category: operations on MultEleStruc (structures allowing duplicates)"
		generalizations: ["StrucOp", "MathConcept", "Op", "MathOp", "Anything"]
	},
	{
		name:    "OrdStrucOp"
		worth:   500
		isA: ["Category", "MathConcept", "MathObj", "Anything"]
		english: "Category: operations on OrdStruc (structures with a defined order)"
		generalizations: ["StrucOp", "MathConcept", "Op", "MathOp", "Anything"]
	},
	{
		name:    "UnitOp"
		worth:   500
		isA: ["Category", "MathConcept", "MathObj", "Anything"]
		english: "Category: operations on Units (UnaryUnitOp etc.)"
		generalizations: ["StrucOp", "MathConcept", "Op", "MathOp", "Anything"]
	},
]
