# EURISKO Parity: Phased Implementation

**Date:** 2026-04-15
**Invariant:** Each phase produces a working system. Nothing subtracts from existing capabilities.

---

## Phase 1: Slot Ontology

**Why first:** H3/H5/H6/H17/H18 can't be implemented without it. HindSight H12-H14 need to know which slot was changed. This is the foundation everything else builds on.

### Issues

**1.1: Slot definition units**
Create ~25 slot-definition units as first-class units in the store with isA: ["Slot"]. Each gets: DataType, SuperSlots, SubSlots, SibSlots, CriterialSlot/NonCriterialSlot classification, DontCopy flag, ElimSlots.

Slots to define: Worth, IsA, Examples, NonExamples, Domain, Range, Defn, FastDefn, Alg, FastAlg, Arity, Generalizations, Specializations, Creditors, Applics, IntApplics, English, Abbrev, Interestingness, Rarity, Inverse, OverallRecord, Restrictions, Extensions, InDomainOf, IsRangeOf, Conjectures, ConjectureAbout, Generator, Format.

**1.2: CriterialSlot vs NonCriterialSlot classification**
Define CriterialSlot and NonCriterialSlot as units. Criterial slots are the ones that matter for identity/equivalence (Arity, Domain, Range, Examples, Defn). Non-criterial slots are metadata (Abbrev, English, Creditors, Worth). This classification drives H5Criterial and H19Criterial.

**1.3: Inverse slot maintenance**
When setting a slot with a known inverse (e.g., setting Range on an op), automatically maintain the inverse (IsRangeOf on the target type). Implementation: a lookup table of known inverse pairs, checked in `unit.Set()` or in a store-level hook.

Pairs: Generalizations/Specializations, Domain/InDomainOf, Range/IsRangeOf, Extensions/Restrictions, Inverse/Inverse, MoreInteresting/LessInteresting, SuperSlots/SubSlots.

**1.4: DSL builtins for slot reasoning**
New builtins: `criterial-slots` (push list of criterial slot names for a unit), `non-criterial-slots`, `sib-slots` (push sibling slots), `super-slots`, `sub-slots`, `inverse-slot` (push the inverse slot name), `slot-type` (push the DataType of a slot), `all-slots` (push list of all populated slot names for a unit).

**1.5: Load slot definitions in seed**
New file `internal/seed/slots.go` that creates all slot definition units. Called from both math and observation domain loaders.

---

## Phase 2: Generalization/Specialization Pipeline

**Why second:** The biggest functional gap. EURISKO's power came from being able to systematically explore the space of concept variations. Requires slot ontology from Phase 1.

### Issues

**2.1: H3 -- "Randomly choose a slot to specialize"**
IfWorkingOnTask: Specializations task without SlotToChange set. ThenCompute: randomly pick a slot from the unit's populated slots (or criterial slots). ThenAddToAgenda: add task with SlotToChange in supplementary info.

Low worth (101) because H5Criterial and H5Good are better.

**2.2: H5/H5Criterial/H5Good -- "Choose specific slots to specialize"**
Three variants of slot selection. H5: random subset. H5Criterial: only criterial slots. H5Good: worth-weighted selection (GoodSubset). Each adds multiple tasks, one per selected slot.

New DSL builtins: `random-subset` (randomly select N items from a list), `good-subset` (select by worth weighting), `best-subset` (select highest worth).

**2.3: H6 -- "Specialize a given slot of a given unit"**
The workhorse specialization heuristic. Given a unit and a SlotToChange, applies SpecializeDataType to narrow the slot value. Creates new unit with specialized slot.

Needs: `specialize-value` DSL builtin or Go-side helper that can narrow a type reference, add a constraint, or restrict a domain.

**2.4: Enhance H-Specialize to match H6**
Current H-Specialize only narrows domain type. Extend it to accept SlotToChange from supplementary task info and specialize any slot, not just domain.

**2.5: H16 -- "If sometimes useful, try generalizing"**
The generalization trigger (counterpart to H1). IfTrulyRelevant: checks good fraction > 0.1 in applics. ThenConjecture: creates ProtoConjec about generalizations. ThenAddToAgenda: adds task to find generalizations.

**2.6: H17 -- "Choose slots to generalize"**
Counterpart to H5. IfWorkingOnTask: Generalizations task without SlotToChange. ThenCompute: RandomSubset of slot names.

**2.7: H18 -- "Generalize a given slot"**
Counterpart to H6. Applies GeneralizeDataType to widen the slot value. Creates new unit with generalized slot.

Needs: `generalize-value` DSL builtin or Go-side helper that can widen a type reference, remove a constraint, or broaden a domain.

**2.8: Supplementary task info**
EURISKO tasks carried supplementary information (SlotToChange, CurSup). The agenda Task struct needs a `Supplementary map[string]any` field (or similar) to pass slot-selection context between heuristics.

---

## Phase 3: Rich HindSight

**Why third:** Requires slot ontology (Phase 1) and understanding of slot changes from the specialization/generalization pipeline (Phase 2).

### Issues

**3.1: Track slot changes in mutations/specializations**
When H6/H18/mutation creates a new unit by changing a slot, record: which slot was changed (CSlot), what it was changed from (CFrom), what it was changed to (CTo). Store on the new unit as provenance metadata.

**3.2: H12 -- "Prevent the slot type from being changed"**
When unit dies, extract CSlot from its creation provenance. Create HAvoid rule that prevents changing objects of that slot's type (GSlot) via sibling slots (CSlotSibs).

**3.3: H13 -- "Prevent changing CFrom into anything"**
When unit dies, extract CFrom. Create HAvoid2 rule that blocks changing CFrom in the relevant slot or its siblings.

**3.4: H14 -- "Prevent any transformation into CTo"**
When unit dies, extract CTo. Create HAvoid3 rule that blocks any transformation resulting in CTo in the relevant slot or its siblings.

**3.5: Replace createAvoidanceRule with H12/H13/H14**
The current `createAvoidanceRule` becomes three separate HindSight heuristic firings. Each generates a different HAvoid variant. Keep the existing avoidance mechanism as a fallback for units without slot-change provenance.

**3.6: HAvoidIfWorking -- probabilistic gate**
"If generalizing IfWorkingOnTask, abort 90% of the time." A learned safety heuristic that prevents the most dangerous mutation target.

---

## Phase 4: Remaining Heuristics

**Why fourth:** These are independent heuristics that don't require new infrastructure beyond what Phases 1-3 provide.

### Issues

**4.1: H1 -- "Specialize operations with >4/5 bad results"**
Full implementation with ProtoConjec creation and targeted specialization proposals. Extends beyond H-PenalizeTrivial.

**4.2: H2 -- "Kill prolific-but-mediocre creators"**
More nuanced than H-KillWorthless. Specifically targets heuristics that create many units with mediocre worth (the "spewing garbage" pattern).

**4.3: H4 -- "Gather empirical data about new concepts"**
Post-creation task scheduling. When new units are created, add tasks to find their instances, examples, and applics. Extends H-ExploreSlots.

**4.4: H8 -- "Find applics in generalizations' applics"**
Search up the isA tree for application records that might apply to the current unit.

**4.5: H10/H15 -- "Get examples from operations whose range is this type"**
Uses IsRangeOf (from Phase 1 inverse maintenance) to find operations that produce this type, then extracts examples from their applics.

**4.6: H19/H19Criterial -- "Eliminate duplicate new units"**
Post-creation sweep that compares new units' slots (or just criterial slots) against existing units and kills duplicates.

**4.7: H20 -- "Run f on args used for other ops"**
Cross-pollination: when an operation shares domain types with other operations, run it on their arguments too.

**4.8: H21 extension -- structured conjecture creation**
Enhance H-Conjecture to create ProtoConjec units with provenance metadata, not just print output.

**4.9: H22/H23 -- Interestingness evaluation**
Require the Interestingness calculus (IntExamples, IsAInt, WhyInt). Check instances/examples against interestingness predicates.

**4.10: H24 -- "Do all examples satisfy the same rare predicate?"**
Requires Rarity tracking on predicates. Tests all examples of a category against rare predicates to find shared properties.

---

## Phase 5: Expanded Type Hierarchy and Operations

**Why fifth:** Enriches the concept space for exploration. Each addition is independent and can be done incrementally.

### Issues

**5.1: OSet type and operations**
Ordered set (no duplicates, ordered). OSetInsert, OSetDelete, OSetIntersect, OSetUnion, OSetEqual.

**5.2: OPair and Pair types**
Ordered and unordered pair types. ReverseOPair operation. Enables Relation (set of ordered pairs).

**5.3: Projection operations**
Proj1, Proj2, FirstEle, SecondEle, ThirdEle, LastEle, AllButFirst, AllButSecond, AllButThird, AllButLast. Most map directly to existing DSL builtins (first, rest, last) but need to exist as unit concepts.

**5.4: Structure type classification**
OrdStruc/UnOrdStruc, MultEleStruc/NoMultEleStruc, EmptyStruc/NonEmptyStruc, SetOfSets, StructureOfStructures. These are type-level classifications that drive H29 and per-type operation applicability.

**5.5: Per-type operations**
ListInsert/Delete/Intersect/Union/Difference, BagInsert/Delete/Intersect/Union/Difference. Each as a unit with defn and domain/range.

**5.6: Meta-operations with algorithms**
Give Compose and Restrict executable defn slots. Implement InvertOp. Implement Transpose (swap binary op argument order).

**5.7: Choice operations**
RandomChoose, RandomSubset, GoodChoose, GoodSubset, BestChoose, BestSubset as unit concepts (they exist as behaviors in H3/H5 but need to be first-class units that other heuristics can discover and reason about).

**5.8: Logical operations as units**
AND, OR, NOT, Implies, TheFirstOf, TheSecondOf as operation units with defn and domain/range.

**5.9: Numeric operations as units**
Add, Multiply, Successor, Square as operation units (currently exist only as DSL builtins, not as units in the store).

**5.10: Additional predicates**
Equality predicates per structure type. Numeric comparison predicates with Transpose. Constant predicates (AlwaysT, AlwaysNIL). UndefinedPred.

**5.11: H25-H28 -- Predicate set analysis**
Once predicate units exist, implement the satisfying/failing set heuristics.

**5.12: H29 -- Multiplicity mutation**
Once MultEleStruc exists, implement element multiplicity mutation for generating new examples.

---

## Phase 6: Interestingness and Rarity

**Why sixth:** Requires a critical mass of predicates and operations (Phase 5) to be useful. The interestingness calculus is an evaluation layer that sits on top of the concept space.

### Issues

**6.1: Interestingness slot and predicate**
Define Interestingness as a slot holding a computable predicate for each unit. Default: based on worth, rarity of type, structural novelty.

**6.2: IntExamples and IsAInt**
IntExamples (interesting examples) as a sub-slot of Examples. IsAInt as its inverse. H23 populates IntExamples.

**6.3: Rarity tracking**
Rarity slot on predicates: (frequency-True, number-T, number-F). Updated each time a predicate is evaluated. Used by H24 to find rare predicates that all examples of a category satisfy.

**6.4: WhyInt explanations**
When a concept is judged interesting, record WHY in a WhyInt slot. Enables meta-reasoning about interestingness.

**6.5: MoreInteresting/LessInteresting ordering**
Relative interestingness between concepts. Maintained as an inverse pair.

---

## Phase 7: Definition Representations and Generators

**Why last:** These are refinements that improve efficiency and enable deeper exploration but aren't required for the core heuristic repertoire to function.

### Issues

**7.1: Multiple definition types**
FastDefn, FastAlg, CompiledDefn, UnitizedDefn, IterativeDefn, RecursiveDefn. Allow heuristics to switch between representations and compile slow definitions into fast ones.

**7.2: Generator slot**
Enable concepts to specify how to generate new instances systematically. NNumber's Generator `((0) (ADD1) (old))` means: start at 0, apply ADD1 to get the next, reuse previous results.

**7.3: Applics enrichment**
Record full input/output pairs in applics (not just target + success boolean). Per-ThenPart Record/FailedRecord tracking. IntApplics (interesting applications) and IndirectApplics.

**7.4: IfAboutToWorkOnTask slot**
Add the pre-execution condition slot to the firing sequence. HAvoid variants that need to abort before any ThenParts run use this instead of ifPotentiallyRelevant.

**7.5: Structured conjecture system**
ProtoConjec as a proper unit type with ConjectureAbout, provenance, and status tracking. H1 and H16 create ProtoConjec units instead of printing text.

---

## Summary

| Phase | Focus | Issues | Dependencies |
|---|---|---|---|
| 1 | Slot ontology | 5 | None |
| 2 | Generalization/specialization | 8 | Phase 1 |
| 3 | Rich HindSight | 6 | Phase 1, 2 |
| 4 | Remaining heuristics | 10 | Phase 1, 2, 3 |
| 5 | Type hierarchy + operations | 12 | Phase 1 |
| 6 | Interestingness + rarity | 5 | Phase 4, 5 |
| 7 | Definition representations | 5 | Phase 1 |

Phases 4 and 5 can be parallelized. Phase 7 can start after Phase 1.

Total: 51 issues across 7 phases.
