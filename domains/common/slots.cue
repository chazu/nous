units: [{
	name:    "CriterialSlot"
	worth:   500
	isA:     ["Slot", "Anything"]
	english: "Slots that determine identity and equivalence"
}, {
	name:    "NonCriterialSlot"
	worth:   500
	isA:     ["Slot", "Anything"]
	english: "Metadata slots that don't affect identity"
},

// === Criterial Slots ===

{
	name:     "Arity"
	worth:    600
	isA:      ["Slot", "CriterialSlot", "Anything"]
	dataType: "Integer"
	english:  "Number of arguments an operation takes"
}, {
	name:     "Domain"
	worth:    600
	isA:      ["Slot", "CriterialSlot", "Anything"]
	dataType: "Unit"
	inverse:  "InDomainOf"
	sibSlots: ["Range"]
	elimSlots: ["Applics"]
	english:  "The input type of an operation"
}, {
	name:     "Range"
	worth:    600
	isA:      ["Slot", "CriterialSlot", "Anything"]
	dataType: "Unit"
	inverse:  "IsRangeOf"
	sibSlots: ["Domain"]
	elimSlots: ["Applics"]
	english:  "The output type of an operation"
}, {
	name:     "Examples"
	worth:    600
	isA:      ["Slot", "CriterialSlot", "Anything"]
	dataType: "Unit"
	inverse:  "IsA"
	subSlots: ["IntExamples"]
	sibSlots: ["NonExamples"]
	english:  "Known instances of a concept"
}, {
	name:     "NonExamples"
	worth:    600
	isA:      ["Slot", "CriterialSlot", "Anything"]
	dataType: "Unit"
	sibSlots: ["Examples"]
	english:  "Known non-instances of a concept"
}, {
	name:     "Defn"
	worth:    600
	isA:      ["Slot", "CriterialSlot", "Anything"]
	dataType: "LispPred"
	elimSlots: ["Applics"]
	english:  "Predicate definition of a concept"
}, {
	name:     "FastDefn"
	worth:    600
	isA:      ["Slot", "CriterialSlot", "Anything"]
	dataType: "LispPred"
	superSlots: ["Defn"]
	english:  "Optimized predicate definition"
}, {
	name:     "Alg"
	worth:    600
	isA:      ["Slot", "CriterialSlot", "Anything"]
	dataType: "LispFn"
	elimSlots: ["Applics"]
	english:  "Algorithm for computing a function"
}, {
	name:     "FastAlg"
	worth:    600
	isA:      ["Slot", "CriterialSlot", "Anything"]
	dataType: "LispFn"
	superSlots: ["Alg"]
	english:  "Optimized algorithm for computing a function"
}, {
	name:     "CompiledDefn"
	worth:    600
	isA:      ["Slot", "CriterialSlot", "Anything"]
	dataType: "LispPred"
	superSlots: ["Defn"]
	english:  "Machine-compiled predicate definition"
}, {
	name:     "UnitizedDefn"
	worth:    600
	isA:      ["Slot", "CriterialSlot", "Anything"]
	dataType: "LispPred"
	superSlots: ["Defn"]
	english:  "Definition expressed in terms of other units"
}, {
	name:     "IterativeDefn"
	worth:    600
	isA:      ["Slot", "CriterialSlot", "Anything"]
	dataType: "LispPred"
	superSlots: ["Defn"]
	english:  "Iterative predicate definition"
}, {
	name:     "RecursiveDefn"
	worth:    600
	isA:      ["Slot", "CriterialSlot", "Anything"]
	dataType: "LispPred"
	superSlots: ["Defn"]
	english:  "Recursive predicate definition"
}, {
	name:     "NecDefn"
	worth:    600
	isA:      ["Slot", "CriterialSlot", "Anything"]
	dataType: "LispPred"
	superSlots: ["Defn"]
	english:  "Necessary condition definition"
}, {
	name:     "SufDefn"
	worth:    600
	isA:      ["Slot", "CriterialSlot", "Anything"]
	dataType: "LispPred"
	superSlots: ["Defn"]
	english:  "Sufficient condition definition"
},

// === Non-Criterial Slots ===

{
	name:     "Worth"
	worth:    600
	isA:      ["Slot", "NonCriterialSlot", "Anything"]
	dataType: "Integer"
	english:  "Estimated importance of a unit"
}, {
	name:     "IsA"
	worth:    300
	isA:      ["Slot", "NonCriterialSlot", "Anything"]
	dataType: "Unit"
	inverse:  "Examples"
	english:  "Categories this unit belongs to"
}, {
	name:     "English"
	worth:    300
	isA:      ["Slot", "NonCriterialSlot", "Anything"]
	dataType: "Text"
	english:  "Human-readable description"
}, {
	name:     "Abbrev"
	worth:    300
	isA:      ["Slot", "NonCriterialSlot", "Anything"]
	dataType: "Text"
	english:  "Abbreviated name"
}, {
	name:     "Creditors"
	worth:    600
	isA:      ["Slot", "NonCriterialSlot", "Anything"]
	dataType: "Unit"
	dontCopy: true
	english:  "Heuristics that contributed to this unit"
}, {
	name:     "Applics"
	worth:    500
	isA:      ["Slot", "NonCriterialSlot", "Anything"]
	dataType: "IOPair"
	subSlots: ["IntApplics", "DirectApplics"]
	dontCopy: true
	english:  "Recorded input-output applications"
}, {
	name:     "IntApplics"
	worth:    500
	isA:      ["Slot", "NonCriterialSlot", "Anything"]
	dataType: "Unit"
	superSlots: ["Applics"]
	dontCopy: true
	english:  "Interesting applications"
}, {
	name:     "DirectApplics"
	worth:    500
	isA:      ["Slot", "NonCriterialSlot", "Anything"]
	dataType: "Unit"
	superSlots: ["Applics"]
	dontCopy: true
	english:  "Direct applications"
}, {
	name:     "IndirectApplics"
	worth:    300
	isA:      ["Slot", "NonCriterialSlot", "Anything"]
	dataType: "Unit"
	superSlots: ["Applics"]
	dontCopy: true
	english:  "Indirect applications"
}, {
	name:     "Generalizations"
	worth:    300
	isA:      ["Slot", "NonCriterialSlot", "Anything"]
	dataType: "Unit"
	inverse:  "Specializations"
	english:  "More general concepts"
}, {
	name:     "Specializations"
	worth:    300
	isA:      ["Slot", "NonCriterialSlot", "Anything"]
	dataType: "Unit"
	inverse:  "Generalizations"
	english:  "More specific concepts"
}, {
	name:     "Interestingness"
	worth:    300
	isA:      ["Slot", "NonCriterialSlot", "Anything"]
	dataType: "LispPred"
	english:  "Predicate for judging interestingness"
}, {
	name:     "Rarity"
	worth:    500
	isA:      ["Slot", "NonCriterialSlot", "Anything"]
	dataType: "Number"
	dontCopy: true
	english:  "How uncommon this concept is"
}, {
	name:     "OverallRecord"
	worth:    300
	isA:      ["Slot", "NonCriterialSlot", "Anything"]
	dataType: "IOPair"
	dontCopy: true
	english:  "Aggregate success/failure record"
}, {
	name:     "Inverse"
	worth:    600
	isA:      ["Slot", "NonCriterialSlot", "Anything"]
	dataType: "Slot"
	inverse:  "Inverse"
	english:  "The inverse relation of a slot"
}, {
	name:     "Restrictions"
	worth:    300
	isA:      ["Slot", "NonCriterialSlot", "Anything"]
	dataType: "Unit"
	inverse:  "Extensions"
	english:  "Narrower versions of this concept"
}, {
	name:     "Extensions"
	worth:    300
	isA:      ["Slot", "NonCriterialSlot", "Anything"]
	dataType: "Unit"
	inverse:  "Restrictions"
	english:  "Broader versions of this concept"
}, {
	name:     "InDomainOf"
	worth:    300
	isA:      ["Slot", "NonCriterialSlot", "Anything"]
	dataType: "Unit"
	inverse:  "Domain"
	sibSlots: ["IsRangeOf"]
	dontCopy: true
	english:  "Operations that take this as domain"
}, {
	name:     "IsRangeOf"
	worth:    300
	isA:      ["Slot", "NonCriterialSlot", "Anything"]
	dataType: "Unit"
	inverse:  "Range"
	sibSlots: ["InDomainOf"]
	dontCopy: true
	english:  "Operations that produce this as range"
}, {
	name:     "Conjectures"
	worth:    300
	isA:      ["Slot", "NonCriterialSlot", "Anything"]
	dataType: "Unit"
	inverse:  "ConjectureAbout"
	english:  "Conjectures involving this unit"
}, {
	name:     "ConjectureAbout"
	worth:    300
	isA:      ["Slot", "NonCriterialSlot", "Anything"]
	dataType: "Unit"
	inverse:  "Conjectures"
	english:  "Units this conjecture is about"
}, {
	name:     "Generator"
	worth:    600
	isA:      ["Slot", "NonCriterialSlot", "Anything"]
	dataType: "Generator"
	english:  "Procedure that produces instances"
}, {
	name:     "Format"
	worth:    600
	isA:      ["Slot", "NonCriterialSlot", "Anything"]
	dataType: "Text"
	english:  "Display format specification"
}, {
	name:     "WhyInt"
	worth:    300
	isA:      ["Slot", "NonCriterialSlot", "Anything"]
	dataType: "Text"
	english:  "Explanation of why this is interesting"
}, {
	name:     "MoreInteresting"
	worth:    300
	isA:      ["Slot", "NonCriterialSlot", "Anything"]
	dataType: "Unit"
	inverse:  "LessInteresting"
	english:  "Units more interesting than this one"
}, {
	name:     "LessInteresting"
	worth:    300
	isA:      ["Slot", "NonCriterialSlot", "Anything"]
	dataType: "Unit"
	inverse:  "MoreInteresting"
	english:  "Units less interesting than this one"
}, {
	name:     "SuperSlots"
	worth:    300
	isA:      ["Slot", "NonCriterialSlot", "Anything"]
	dataType: "Slot"
	inverse:  "SubSlots"
	sibSlots: ["SibSlots"]
	english:  "Parent slots in the slot hierarchy"
}, {
	name:     "SubSlots"
	worth:    300
	isA:      ["Slot", "NonCriterialSlot", "Anything"]
	dataType: "Slot"
	inverse:  "SuperSlots"
	sibSlots: ["SibSlots"]
	english:  "Child slots in the slot hierarchy"
}, {
	name:     "SibSlots"
	worth:    300
	isA:      ["Slot", "NonCriterialSlot", "Anything"]
	dataType: "Slot"
	sibSlots: ["SuperSlots", "SubSlots"]
	english:  "Sibling slots in the slot hierarchy"
}, {
	name:     "DataType"
	worth:    600
	isA:      ["Slot", "NonCriterialSlot", "Anything"]
	dataType: "Unit"
	english:  "Type of values stored in a slot"
}, {
	name:     "DontCopy"
	worth:    300
	isA:      ["Slot", "NonCriterialSlot", "Anything"]
	dataType: "Boolean"
	english:  "Whether to skip this slot when copying a unit"
}, {
	name:     "ElimSlots"
	worth:    300
	isA:      ["Slot", "NonCriterialSlot", "Anything"]
	dataType: "Slot"
	english:  "Slots to clear when this slot changes"
}, {
	name:     "DoubleCheck"
	worth:    300
	isA:      ["Slot", "NonCriterialSlot", "Anything"]
	dataType: "Boolean"
	english:  "Whether to verify values before accepting"
}]
