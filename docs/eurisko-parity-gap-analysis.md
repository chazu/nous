# EURISKO Parity Gap Analysis

**Date:** 2026-04-15
**Goal:** Faithful reimplementation of EURISKO's math domain as nous's foundation. Everything added on top extends rather than subtracts the original capabilities.

**Source:** Analysis of EURUNITS file from the original EURISKO Interlisp source (~/dev/interlisp/EURISKO/EURUNITS), compared against the current nous codebase.

---

## 1. Scale Comparison

| Dimension | EURISKO | nous | Gap |
|---|---|---|---|
| Total units | 304 | 53 seed + ~12 dynamic | ~240 units |
| Heuristics | 29 (H1-H29) | 11 (+ H-AnalyzeApplics) | 18 heuristics |
| HAvoid variants | 6 | 1 template | 5 variants |
| HindSight rules | 3 (H12-H14) | 1 template | 2 strategies |
| Slot definitions | ~30 (as first-class units) | 0 (opaque strings) | 30 units |
| Structure types | 12 (Set, List, Bag, OSet, OPair, Pair, ...) | 3 (Set, List, Bag) | 9 types |
| Operations | ~50 (set ops, list ops, projections, logic, numeric, meta) | 8 (set ops, GCD, DivisorsOf, Compose, Restrict) | ~42 operations |
| Predicates | ~20 (equality, comparison, membership, constant) | 3 (MemberOf, SubsetOf, SetEqual) | ~17 predicates |
| Definition types | 7 (Defn, FastDefn, FastAlg, CompiledDefn, UnitizedDefn, IterativeDefn, RecursiveDefn) | 1 (defn) | 6 types |

---

## 2. Heuristic-by-Heuristic Comparison

### Heuristics nous has

| nous heuristic | EURISKO equivalent | Match quality | Notes |
|---|---|---|---|
| H-FindExamples (700) | H7 + H9 | Good | H7: "If no instances, find some." H9: "Look in generalizations' examples." nous combines both into one. |
| H-RunOnExamples (750) | H11 | Good | H11: "Find applics by running Alg on Domain members." Most complex heuristic in both systems. |
| H-CheckExtremes (600) | (none) | Novel | No EURISKO equivalent. Examines empty/singleton sets. Keep. |
| H-Specialize (650) | H6 (partial) | Shallow | H6: "Specialize a given slot of a given unit." nous only narrows domain type; H6 can specialize any slot. |
| H-CheckDomain (550) | H20 (partial) | Different | H20: "Run f on args used for other ops." nous creates self-compositions when domain/range overlap. |
| H-Conjecture (700) | H21 | Good | H21: "If op u duplicates u2's results, conjecture extension." nous checks set equality and subset. |
| H-ExploreSlots (500) | H4 + H10 (partial) | Rough | H4: "Gather empirical data about new units." H10: "If instances found, check interestingness." |
| H-KillWorthless (800) | H2 (partial) | Simplified | H2 is more nuanced: checks if a creator "spews garbage" (many mediocre units). nous uses simple worth threshold. |
| H-PenalizeTrivial (600) | Part of H1 | Partial | H1 checks 4/5 bad ratio across all applics. nous only penalizes empty-data units. |
| H-BoostInteresting (650) | H22 + H23 (partial) | Shallow | H22/H23: full Interestingness calculus with rare predicates. nous checks empty/singleton data. |
| H-AnalyzeApplics (600) | (novel) | Novel | Extends H1's concept. No EURISKO equivalent. Keep. |

### Heuristics EURISKO has that nous lacks

**H1: "If >4/5 bad results, specialize" (worth 724)**
- IfTrulyRelevant: Checks that some Applics have high Worth but most have low Worth
- ThenConjecture: Creates ProtoConjec describing specialization opportunity
- ThenAddToAgenda: Adds tasks to find specializations
- nous has H-PenalizeTrivial which penalizes trivial results, but H1 is broader: it analyzes the full applics record, conjectures WHY the operation is failing, and proposes targeted specializations

**H2: "Kill concept that leads to garbage" (worth 700)**
- IfFinishedWorkingOnTask: Checks if a creator produced many units with mediocre worth
- ThenDeleteOldConcepts: Kills units with Worth <= 175
- ThenCompute: Severely punishes the creator
- nous has H-KillWorthless but it's simpler: kills anything below worth 100 with creditors. H2 specifically targets prolific-but-mediocre creators

**H3: "Randomly choose a slot to specialize" (worth 101)**
- IfWorkingOnTask: Checks for Specializations task without SlotToChange
- ThenCompute: Randomly selects a slot
- Requires slot ontology (knowing what slots a unit has that can be specialized)

**H4: "Gather empirical data about new concepts" (worth 703)**
- IfFinishedWorkingOnTask: Checks for NewUnits in TaskResults
- ThenAddToAgenda: Adds tasks to find instances for each new unit
- nous's H-ExploreSlots partially covers this

**H5/H5Criterial/H5Good: "Choose specific slots to specialize" (worth 151/700/700)**
- Three variants: random selection, criterial-slots-only, high-worth-slots-only
- H5Criterial uses Examples of CriterialSlot to pick which slots
- H5Good uses GoodSubset instead of RandomSubset
- Subsumption: H5Criterial subsumes H3 and H5
- Requires slot ontology with CriterialSlot classification

**H6: "Specialize a given slot of a given unit" (worth 700)**
- IfWorkingOnTask: Checks for SlotToChange in supplementary info
- ThenCompute: Applies SpecializeDataType function to the slot value
- ThenDefineNewConcepts: Creates new unit with the specialized slot
- nous's H-Specialize only narrows domain type. H6 can specialize ANY slot

**H8: "Find applics in generalizations' applics" (worth 700)**
- IfWorkingOnTask: Checks Applics slot and Generalizations
- ThenCompute: Maps over generalization applics, tests domain membership
- Requires Generalizations slot and IsA traversal

**H10: "If unit is range of op, get examples from op's applics" (worth 700)**
- IfWorkingOnTask: Checks Examples slot and IsRangeOf
- ThenCompute: Extracts outputs from operation's application pairs
- Requires IsRangeOf inverse slot (inverse of Range)

**H15: "Like H10 but for multiple operations" (worth 700)**
- Same as H10 but handles case where unit is range of multiple operations
- Aggregates from all operations' applics

**H16: "If sometimes useful, generalize" (worth 600)**
- IfTrulyRelevant: Checks good fraction > 0.1
- ThenConjecture: Creates ProtoConjec about generalizations
- ThenAddToAgenda: Adds task to find generalizations
- The generalization counterpart to H1. nous has no generalization capability

**H17: "Choose slots to generalize" (worth 600)**
- IfWorkingOnTask: Checks Generalizations task without SlotToChange
- ThenCompute: RandomSubset of slot names
- The generalization counterpart to H5. Requires slot ontology

**H18: "Generalize a given slot" (worth 704)**
- IfWorkingOnTask: Checks Generalizations task with SlotToChange
- ThenCompute: Applies GeneralizeDataType function
- ThenDefineNewConcepts: Creates new generalized unit
- The generalization counterpart to H6

**H19/H19Criterial: "Eliminate duplicate new units" (worth 150/700)**
- IfFinishedWorkingOnTask: Checks NewUnits for slot-by-slot equivalence
- ThenDeleteOldConcepts: Kills duplicates
- H19Criterial only checks criterial slots (more efficient)
- nous's H-Conjecture detects equality but only for sets with data

**H22: "Check instances for interesting ones" (worth 500)**
- IfFinishedWorkingOnTask: Checks Instances slot and Interestingness predicate
- ThenAddToAgenda: Adds task to evaluate instances
- Requires Interestingness as a computable predicate, not just worth

**H23: "Some examples may be interesting" (worth 700)**
- IfWorkingOnTask: Checks IntExamples slot and Interestingness
- ThenCompute: Tests each example against Interestingness predicate
- Requires IntExamples slot (interesting examples subset)

**H24: "Do all examples satisfy the same rare predicate?" (worth 500)**
- IfWorkingOnTask: Filters UnaryPred by worth/interestingness and Rarity
- ThenCompute: Tests all examples against predicates
- Requires Rarity tracking and predicate units

**H25-H28: "Study satisfying/failing sets of predicates" (worth 500 each)**
- H25: For interesting predicates, define the set of tuples where it holds
- H26: For interesting predicates, define the set where it fails
- H27: Same as H25 but for unary predicates specifically
- H28: Same as H26 but for unary predicates
- These create new Set units derived from predicate analysis

**H29: "Mutate examples by changing multiplicities" (worth 500)**
- IfWorkingOnTask: Checks MultEleStruc and Examples slot
- ThenCompute: Randomly inserts/deletes elements from examples
- Requires MultEleStruc type hierarchy

### HindSight Rules (H12-H14)

EURISKO had three distinct HindSight strategies. nous has one template.

**H12: "Prevent the slot type from being changed" (worth 700)**
- When unit C dies, extract SlotToChange and GSlot
- Create HAvoid rule that prevents changing objects of that type in that slot
- Uses: CSlotSibs (sibling slots of the changed slot)
- Generates: HAvoid (blocks the slot type)

**H13: "Prevent changing CFrom into anything" (worth 700)**
- When unit C dies, extract CFrom (the original value) and CTo (what it was changed to)
- Create HAvoid2 rule that blocks changing CFrom in sibling slots
- Generates: HAvoid2 (blocks the source value), HAvoid2AND (learned instance)

**H14: "Prevent transforming anything INTO CTo" (worth 700)**
- When unit C dies, extract CTo
- Create HAvoid3 rule that blocks any transformation resulting in CTo in sibling slots
- Generates: HAvoid3 (blocks the target value), HAvoid3First (learned instance)

**nous's current implementation:** `createAvoidanceRule` generates a single crude HAvoid template that blocks a creditor from creating units of the same isA type. It doesn't know which slot was changed, what the original value was, or what it was changed to.

### HAvoid Variants

| EURISKO | What it avoids | Generated by | nous equivalent |
|---|---|---|---|
| HAvoid | Changing GSlot via CSlotSibs | H12 | Partial (our template) |
| HAvoid2 | Changing CFrom in CSlot or siblings | H13 | Missing |
| HAvoid2AND | Changing AND in IfWorkingOnTask during generalization | H13 (learned) | Missing |
| HAvoid3 | Transforming anything into CTo in CSlot | H14 | Missing |
| HAvoid3First | Transforming anything into TheFirstOf in IfWorkingOnTask | H14 (learned) | Missing |
| HAvoidIfWorking | Generalizing IfWorkingOnTask (10% random abort) | Manual/learned | Missing |

---

## 3. Structural Gaps

### 3.1 Slot Ontology

EURISKO treats slots as units with properties:

```
Slot (worth: 600)
  isA: (ReprConcept Anything Category)
  SubSlots, SuperSlots, SibSlots
  DataType
  Inverse
  DontCopy
  ElimSlots
  Format
  CriterialSlot vs NonCriterialSlot

CriterialSlot examples: Arity, Domain, Range, Examples, NonExamples, FastDefn, Defn, ...
NonCriterialSlot examples: Abbrev, Applics, Creditors, English, Generalizations, Worth, ...
```

Key slot relationships used by heuristics:
- `SuperSlots(IntExamples) = (Examples)` -- IntExamples is a sub-slot of Examples
- `Inverse(Generalizations) = (Specializations)` -- auto-maintained bidirectional links
- `Inverse(Domain) = (InDomainOf)` -- "what ops have me as domain?"
- `Inverse(Range) = (IsRangeOf)` -- "what ops produce me?" (used by H10, H15)
- `SibSlots` -- slots at the same level (used by HindSight for avoidance scope)

nous has none of this. Slots are string keys. No relationships between slots. No metadata about what type a slot holds.

### 3.2 Inverse Slot Maintenance

EURISKO automatically maintains inverse relationships. When you set `Range(SetUnion) = Set`, the system also sets `IsRangeOf(Set) = SetUnion`. This is critical for H10 and H15 which need to find "what operations produce this type?"

nous doesn't maintain inverses. Adding `IsRangeOf` requires either:
- Automatic inverse maintenance in `unit.Set()` (when setting a slot with a known inverse, also set the inverse on the target)
- Or a query-time scan (slower but simpler)

### 3.3 Missing Type Hierarchy

EURISKO's structure types that nous lacks:

```
Structure
  OrdStruc (ordered structures)
    List
    OSet (ordered set -- no duplicates, ordered)
  UnOrdStruc (unordered structures)
    Set
    Bag
  MultEleStruc (allows duplicate elements)
    Bag
    List
  NoMultEleStruc (no duplicates)
    Set
    OSet
  EmptyStruc
  NonEmptyStruc
  SetOfSets
  StructureOfStructures
  SetOfOPairs
  Relation (set of ordered pairs)
```

Pair types:
```
OPair (ordered pair)
  ReverseOPair
Pair (unordered pair)
```

This matters because H29 operates specifically on MultEleStruc (mutate element multiplicities), and the structure classification drives which operations are applicable.

### 3.4 Missing Operations

**Meta-operations (operate on operations):**
- `Compose` -- exists as a unit but has no executable defn/algorithm
- `Restrict` -- exists as a unit but has no executable defn/algorithm
- `InvertOp` -- does not exist. Creates the inverse of an operation
- `Transpose` -- swaps argument order of a binary operation

**Projection operations:**
- `Proj1`, `Proj2` -- extract first/second element of a pair
- `Proj1of3`, `Proj2of3`, `Proj3of3` -- for ternary operations
- `FirstEle`, `SecondEle`, `ThirdEle`, `LastEle` -- list element access
- `AllButFirst`, `AllButSecond`, `AllButThird`, `AllButLast` -- list slicing
- `MEMB`, `MEMBER` -- membership predicates (slightly different semantics)

**Structural operations per type:**
- `ListInsert`, `ListDelete`, `ListDelete1`, `ListIntersect`, `ListUnion`, `ListDifference`
- `BagInsert`, `BagDelete`, `BagDelete1`, `BagIntersect`, `BagUnion`, `BagDifference`
- `OSetInsert`, `OSetDelete`, `OSetIntersect`, `OSetUnion`
- `StrucInsert`, `StrucDelete`, `StrucIntersect`, `StrucUnion`, `StrucDifference`
- `MultEleStrucInsert`, `MultEleStrucDelete1`

**Numeric operations:**
- `Add` (PLUS), `Multiply` (TIMES), `Successor` (ADD1) -- exist as DSL builtins but not as unit concepts
- `Square` -- exists as a number type but not as an operation unit

**Choice operations:**
- `RandomChoose`, `RandomSubset` -- used by H3/H5 for slot selection
- `GoodChoose`, `GoodSubset`, `BestChoose`, `BestSubset` -- worth-weighted variants

**Logical operations as units:**
- `AND`, `OR`, `NOT`, `Implies`, `TheFirstOf`, `TheSecondOf` -- exist in Interlisp but not as nous units

**Parallel/repetition operations:**
- `ParallelReplace`, `ParallelReplace2` -- apply op to each element
- `ParallelJoin`, `ParallelJoin2` -- combine results
- `Coalesce` -- merge structure levels
- `Repeat`, `Repeat2` -- iterate operation

### 3.5 Missing Predicate Units

EURISKO predicates that nous lacks:

**Equality predicates (per structure type):**
- `StrucEqual`, `ListEqual`, `BagEqual`, `OSetEqual`, `OrdStrucEqual`
- `EQUAL` (deep), `EQ` (identity)

**Numeric comparison predicates:**
- `IEQP`, `ILEQ`, `IGEQ`, `ILESSP`, `IGREATERP`
- Each has a `Transpose` (inverse comparison)

**Constant predicates:**
- `AlwaysT`, `AlwaysNIL`, `AlwaysT2`, `AlwaysNIL2`
- `ConstantPred`, `ConstantUnaryPred`, `ConstantBinaryPred`
- `UndefinedPred`

### 3.6 Interestingness Calculus

EURISKO had a multi-faceted interestingness system:

- `Interestingness` -- a slot holding a predicate that determines if a concept is interesting
- `IntExamples` -- interesting examples (a sub-slot of Examples)
- `IsAInt` -- inverse of IntExamples
- `MoreInteresting` / `LessInteresting` -- relative interestingness ordering between concepts
- `WhyInt` -- explanation of why something is interesting
- `Rarity` -- tracking how often predicates succeed (format: frequency-True, number-T, number-F)

nous has flat `worth` as the sole measure. H-BoostInteresting checks for empty/singleton data, which is a tiny fraction of what EURISKO's interestingness system did.

### 3.7 Applics Structure

EURISKO's applics are richer than nous's:

```
Applics: ((args) (results)) pairs -- full input/output records
IntApplics: interesting application instances (sub-slot)
IndirectApplics: ((situation resultant-units directness) ...)
DirectApplics: direct application records
Record: successful execution records per ThenPart
RecordSlot: which slot was involved
FailedRecord: failed execution records per ThenPart
FailedRecordFor: per-unit failed records
OverallRecord: (call-count . success-count)
```

nous's applics is a rolling window of `{target, result}` pairs. No input/output recording, no per-ThenPart records, no distinction between direct and indirect applications.

### 3.8 Definition/Algorithm Representations

EURISKO had multiple representation types for the same concept:

```
Defn -- declarative definition (predicate)
FastDefn -- compiled/efficient definition
Alg -- algorithm (function)
FastAlg -- compiled/efficient algorithm
CompiledDefn -- machine-compiled definition
UnitizedDefn -- definition in terms of other units
UnitizedAlg -- algorithm in terms of other units
IterativeDefn -- iterative definition
IterativeAlg -- iterative algorithm
RecursiveDefn -- recursive definition
RecursiveAlg -- recursive algorithm
```

Heuristics could switch between representations, compile definitions for speed, or compare representations for equivalence. nous has a single `defn` string.

### 3.9 Conjecture System

EURISKO had a structured conjecture system:

```
Conjecture -- base type
ProtoConjec -- proto-conjecture (unverified, worth 802)
ConjectureAbout -- what the conjecture is about
Conjectures -- slot holding active conjectures
```

H1 and H16 create ProtoConjec units with explanatory metadata. H21 creates equivalence conjectures. nous's H-Conjecture prints conjecture text but doesn't create structured conjecture units with provenance.

### 3.10 IfAboutToWorkOnTask

EURISKO had an `IfAboutToWorkOnTask` condition slot that fires BEFORE ThenParts execute. This is used by HAvoid to abort tasks before work begins. nous doesn't have this -- HAvoid rules use `ifPotentiallyRelevant` with `abort`, which is functionally similar but structurally different. The original IfAboutToWorkOnTask was a separate phase in the firing sequence, between task selection and ThenPart execution.

### 3.11 Generator Slot

EURISKO concepts could have a `Generator` slot specifying how to produce new instances:

```
NNumber Generator: ((0) (ADD1) (old))
  -- start at 0, apply ADD1 to previous, use old results
```

This enabled systematic enumeration of concept instances. nous has no generator mechanism.

---

## 4. What nous Has That EURISKO Didn't

These are extensions that should be preserved:

1. **H-AnalyzeApplics** -- meta-heuristic that inspects applics for type-skewed patterns. Extends H1's concept.
2. **H-CheckExtremes** -- examines edge cases of sets. No EURISKO equivalent.
3. **Performance-based mutation trigger** -- applics-driven, not random. Better than EURISKO's approach.
4. **Worth-growth reward** -- positive credit flow to creators. EURISKO had punishment but less systematic reward.
5. **Bitemporal fact store integration** (pudl bridge) -- external knowledge layer.
6. **Three-loop architecture** (Datalog / nous / human) -- separation EURISKO lacked.
7. **Mutation validation** -- tokenizer dry-run before accepting mutants. EURISKO accepted all.
8. **Mode 2: Observations domain** -- reasoning over accumulated real-world data.
9. **Token-level mutation system** -- 7 mutation types with validation. More systematic than EURISKO's code-level mutation.

---

## 5. What Doesn't Need to Change

The following nous components are compatible with EURISKO parity:

- **Engine core** (engine.go, fire.go) -- two-level control loop matches EURISKO's Interp2
- **Unit/Store model** (unit.go, store.go) -- direct equivalent of Interlisp property lists
- **Agenda** (agenda.go) -- priority queue with duplicate merging matches EURISKO's agenda
- **DSL/VM** (vm.go, token.go, value.go) -- stack-based interpreter is a valid replacement for Interlisp lambdas
- **Credit assignment** (credit.go) -- punishCreators, rewardCreators, trackApplics, HandleDeletedUnit
- **Mutation system** (mutate.go, mutation.go) -- token-level mutation with performance-based trigger
- **HindSight mechanism** -- Graveyard, createAvoidanceRule (will be extended, not replaced)
- **Self-modification loop** -- all four feedback loops just wired

---

## 6. Compatibility with CUE-as-RLL

All gap-closing work is compatible with the eventual CUE migration:

- Slot ontology units map directly to CUE definitions with metadata fields
- Inverse relationships become CUE constraints (bidirectional)
- Heuristic programs remain as stack DSL strings until CUE-as-RLL replaces them as a batch
- New type hierarchy units are standard units -- they'll become CUE schemas
- The additional heuristics are more DSL programs -- same migration path as existing ones
- ProtoConjec and structured conjectures map to CUE values naturally
- Generator slots could become CUE comprehensions in the long term

The principle: build EURISKO parity in the current representation, then migrate the representation. The structure and semantics transfer; only the syntax changes.
