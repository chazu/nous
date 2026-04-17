# EURISKO Parity: Phased Implementation

**Date:** 2026-04-15 (updated 2026-04-16)
**Invariant:** Each phase produces a working system. Nothing subtracts from existing capabilities.

**Companion docs:** [docs/dynamics-and-tuning.md](dynamics-and-tuning.md) records observed runtime behavior, four control-loop bugs found post-Phase-2, their fixes, and the resulting steady-state numbers.

---

## Phase 0: CUE Data Layer -- COMPLETE

Domain definitions moved from hardcoded Go to CUE files in `domains/`. CUE loader at `internal/cueload/`. `-domains-dir` flag for development without recompilation. Old Go seed files deleted.

**Completed:** 0.1-0.7

---

## Phase 1: Slot Ontology -- COMPLETE

49 slot definition units in `domains/common/slots.cue`. Store-level inverse maintenance (`RegisterInverse`, `SetSlot`). 8 DSL builtins for slot reasoning. Inverse index computed after domain load.

**Completed:** 1.1-1.5

**Known bug (1.6):** RESOLVED (commit `941e4d2`). Root cause turned out to be in the `criterial-slots` DSL builtin, not in inverse maintenance: slot keys are lowercase-first (`"domain"`) but slot definition units are PascalCase (`"Domain"`), so `store.IsA("domain", "CriterialSlot")` always returned false and H17 iterated an empty list. Fix capitalizes the slot key before the IsA lookup. See `docs/dynamics-and-tuning.md`.

---

## Phase 2: Generalization/Specialization Pipeline -- COMPLETE

Multi-step pipeline implemented. H-Specialize evolved to emit tasks. H6-Specialize creates units. H3/H5-Criterial select slots. H16/H17/H18 mirror for generalization. Task.Extra carries supplementary info. New builtins: `add-spec-task`, `add-gen-task`, `replace-slot-value`, `get-task-extra`, `set-task-extra`, `random-choice`, `random-subset`, `starts-with?`.

**Completed:** 2.1-2.8

**Stabilization fixes applied during integration testing:**
- H6/H18 nil guard (task extras can be nil when heuristic fires during unit-focus)
- H-RunOnExamples restricted to seed data only (prevents combinatorial explosion from result sets feeding back as inputs)
- H-Specialize one-shot guard (`specTaskAdded` flag prevents repeated task emission)
- H16 one-shot guard (`genTaskAdded` flag prevents agenda flooding)
- H6 uses `restrictedTo` slot instead of modifying domain (so H-RunOnExamples can still find data via parent types)

**Self-modification loop (pre-Phase work) also complete:**
- No-op firing detection
- Deferred failure on unit death
- Performance-based mutation trigger
- Worth-growth reward
- H-AnalyzeApplics meta-heuristic
- HindSight validation with promotion/demotion
- H2-KillGarbageCreator
- H19-EliminateDuplicates

---

## Phase 3: Rich HindSight

**Why next:** The current `createAvoidanceRule` generates a single crude HAvoid template that blocks by isA type. EURISKO had three distinct strategies that analyzed which slot was changed and what values were involved. Requires slot-change provenance from the specialization/generalization pipeline (Phase 2).

### Issues

**3.1: Track slot changes in mutations/specializations** -- COMPLETE (H6/H18)
When H6/H18/mutation creates a new unit by changing a slot, record: which slot was changed (cSlot), what it was changed from (cFrom), what it was changed to (cTo). Store on the new unit as provenance metadata.

Implemented via DSL builtin `record-slot-change (unitName slot from to --)` in `internal/dsl/builtins.go`. H6-Specialize and H18-Generalize in `domains/common/heuristics.cue` each call it after their existing slot write. Mutation path (`internal/engine/mutation.go`) is deferred until H12-14 land — code-edit mutation conflates with data-slot change, so we defer until we know what H12-14 actually need to read.

Direction convention: `cFrom` is pre-change, `cTo` is post-change. For specialization `cFrom` is the wider type and `cTo` the narrower; for generalization the reverse. H13 (block CFrom) and H14 (block CTo) read the correct slot accordingly.

**3.2: H12 -- "Prevent the slot type from being changed"** -- COMPLETE
When unit dies, extract cSlot/gSlot from its creation provenance. Create HAvoid-N rule that vetoes any future task whose CurSlot == gSlot and whose SlotToChange ∈ (cSlot ∪ siblings). HAvoid uses `ifAboutToWorkOnTask`, aborts the task before H6/H18 fires.

Implementation: `createH12Rule` in `internal/engine/credit.go`, called from `HandleDeletedUnit` when the graveyard snapshot carries both `cSlot` and `gSlot`. Legacy `createAvoidanceRule` remains as fallback for units without provenance (H-RunOnExamples output, seed units).

New infrastructure:
- `ifAboutToWorkOnTask` wired as first gate in `fireTaskRule` (previously the slot existed in IfPartSlots and programSlots lists but was never executed). Pulled forward from Phase 7.4 — H13/H14 don't need it; only H12 and HAvoidIfWorking do.
- `record-slot-change` DSL builtin now also captures the task's CurSlot as `gSlot` on the created unit.
- HAvoid naming: `HAvoid-N` gensym (EURISKO does no up-front dedup; duplicates resolved later by H19).
- Starting worth 700 (EURISKO value), isA includes HAvoidRule and HindSightRule.

Siblings computed at HAvoid-creation time via `siblingSlots` — reads the slot unit's pre-computed `sibSlots` list. EURISKO's 50-sibling cap preserved; overflow falls back to cSlot alone.

**3.3: H13 -- "Prevent changing CFrom into anything"** -- COMPLETE
When unit dies, extract cFrom. Create HAvoid2-N rule that post-hoc kills newly-created units whose cFrom matches.

Implementation: `createH13Rule` in `internal/engine/credit.go`, called from `HandleDeletedUnit` alongside H12 when provenance is present. Unlike H12 (pre-abort), H13 uses `ifFinishedWorkingOnTask`: the task is allowed to run, then the HAvoid2 iterates new-units and kills any whose stored cFrom matches the dying unit's cFrom.

New infrastructure:
- `ifFinishedWorkingOnTask` firing phase added to `WorkOnTask` (previously declared but never executed). Runs after all ThenParts of all heuristics complete, with VM.NewUnits populated from the task's creations.
- `VM.NewUnits` now cleared per-task at task start so the post-task phase sees only this task's creations, not cumulative output.
- `new-units` DSL builtin pushes the current NewUnits list as a string list.
- HAvoid2-N naming; starting worth 700; isA includes HAvoidRule and HindSightRule.

**3.4: H14 -- "Prevent any transformation into CTo"** -- COMPLETE
Mirror of H13 but inverted: HAvoid3-N kills newly-created units whose cTo matches the dying unit's cTo. Same `ifFinishedWorkingOnTask` mechanism as H13. Implementation: `createH14Rule` in `internal/engine/credit.go`.

**3.5: Replace createAvoidanceRule with H12/H13/H14** -- COMPLETE
`HandleDeletedUnit` in `internal/engine/credit.go` dispatches: if the grave snapshot carries `cSlot` and `gSlot` (i.e. the unit was created via H6/H18), fire H12+H13+H14 in sequence; otherwise fall back to the legacy `createAvoidanceRule`. All three variants always fire together when provenance is present — each looks at a different aspect of the slot change.

**3.6: HAvoidIfWorking -- probabilistic gate** -- COMPLETE
Seeded domain heuristic (not HindSight-generated) in `domains/common/heuristics.cue`. Uses `ifAboutToWorkOnTask`: if CurSlot=="generalizations" AND SlotToChange=="ifWorkingOnTask", calls abort 9 times out of 10. Requires new `random-int` DSL builtin. Defensive guard against the system generalizing its own heuristics' trigger conditions — rarely fires in math-domain runs but protects against meta-level self-destruction.

---

## Phase 4: Remaining Heuristics

**Why fourth:** These are independent heuristics that don't require new infrastructure beyond what Phases 1-3 provide.

### Issues

**4.1: H1 -- "Specialize operations with >4/5 bad results"**
Full implementation with ProtoConjec creation and targeted specialization proposals. Extends beyond H-PenalizeTrivial. Should use applics data to identify which operations have high failure rates and propose specific specializations based on the failure patterns.

**~~4.2: H2 -- "Kill prolific-but-mediocre creators"~~** COMPLETE
Implemented as H2-KillGarbageCreator during engine stabilization. Scans children of heuristics, punishes those with 5+ children and 80%+ mediocre worth.

**4.3: H4 -- "Gather empirical data about new concepts"** -- COMPLETE
Implemented as a CUE heuristic in `domains/common/heuristics.cue`. Uses `ifFinishedWorkingOnTask` + the `new-units` builtin (both landed in Phase 3.3) to schedule an `examples` task for each newly-created unit that doesn't yet have examples populated. Fires reliably — a 100-cycle math run leaves ~13 pending "After synthesis, seek instances" tasks on the agenda.

**4.4: H8 -- "Find applics in generalizations' applics"** -- BLOCKED on type predicates
Needs `DomainTests` — running each domain type's `defn` as a predicate against a candidate arg to check type applicability. We don't have type defn predicates on our type units (Set, Number, etc.). Deferred until Phase 5.6 (meta-ops with executable defns) or hand-seeded domain-type predicates.
Workable shape when unblocked: walk `generalizations` chain, read `applics-args`, filter by domain-type predicates, apply this unit's alg, record.

**4.5: H10/H15 -- "Get examples from operations whose range is this type"** -- COMPLETE
Both in `domains/common/heuristics.cue`. H10 picks one random op from `isRangeOf` via `random-choice`; H15 iterates them all. Both use the Phase 7.3 `applics-outputs` builtin and `add-to-slot` (inverse-maintained) to append each recorded output as an Example. Dormant unless the target unit is in some op's range — works for Set/Number in the math domain once applics accumulate.

**~~4.6: H19/H19Criterial -- "Eliminate duplicate new units"~~** COMPLETE
H19-EliminateDuplicates implemented during engine stabilization. H19Criterial added as a CUE heuristic in Phase 4a — `ifFinishedWorkingOnTask` iterates new-units, compares all criterial slots against peers in its isA category, kills structurally-identical duplicates. Skips H-Specialize/H18-Generalize-created units (our H6 stores the restriction in `restrictedTo` rather than modifying criterial slots, so specs would false-positive against their parents).

**4.7: H20 -- "Run f on args used for other ops"** -- COMPLETE
Unit-focus heuristic: for each sibling op (same first-isA category) with recorded applics, take its arg tuples and apply CurUnit's alg to them, recording the new applic and creating a result unit. Uses new `apply-op-args (argList opName -- result)` builtin that resolves arg unit names to their `data` slot before running the defn. Caps at 3 cross-applications per firing to avoid flooding. 300-cycle math run produced ~115 cross-applications (e.g. SetDifference run on GCD's numeric args).

New DSL builtins: `apply-op-args`, `list-join`.

**4.8: H21 extension -- structured conjecture creation**
Enhance H-Conjecture to create ProtoConjec units with provenance metadata, not just print output.

**4.9: H22/H23 -- Interestingness evaluation** -- COMPLETE (dormant until seeded)
Both heuristics live in `domains/common/heuristics.cue`.
- **H22**: `ifFinishedWorkingOnTask` — when an examples-task finishes and the unit has an Interestingness predicate, schedule an intExamples-task.
- **H23**: `ifWorkingOnTask` on CurSlot==intExamples — iterates the unit's examples, runs the Interestingness predicate against each via new `is-interesting?` builtin, appends passers to intExamples via `add-to-slot` (inverse-maintained so isAInt auto-wires).

New DSL builtins: `is-interesting? (unit cand -- bool)` runs the unit's interestingness slot as a DSL program with env `candidate` bound; `add-to-slot (value unit slot --)` appends unique through Store.SetSlot so inverses fire.

Both dormant in current math-domain runs — no seeded unit has an Interestingness predicate. Will activate when H1 or Phase 5.10 introduces predicates, or when we hand-seed a few. Tested directly: `TestH23FillsIntExamples` and `TestH22SchedulesIntExamplesTask`.

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

## Phase 6: Interestingness and Rarity -- COMPLETE (scaffolding)

Scaffolding only. The heuristics that actually populate these slots (H22/H23/H24) live in Phase 4. Phase 6 installs the slot ontology so Phase 4b can drop in without further infrastructure.

### Issues

**6.1: Interestingness slot and predicate** -- COMPLETE (slot defined; no default compute)
Interestingness slot present in `domains/common/slots.cue` with dataType LispPred. Per-unit predicate; default compute deferred — in EURISKO, units without an Interestingness predicate simply don't trigger H23. That's acceptable behaviour for now.

**6.2: IntExamples and IsAInt** -- COMPLETE
Both slots defined with proper inverse wiring (IntExamples ↔ IsAInt) and `IntExamples.superSlots = [Examples]` so IntExamples ⊆ Examples. Inverse maintenance verified by test (setting intExamples on a unit automatically writes isAInt on the target).

**6.3: Rarity tracking** -- COMPLETE (slot shape only; population deferred)
Rarity slot defined with dataType `List` and format `[frequency-True, num-True, num-False]` matching EURISKO's tuple. Population hook deferred until Phase 5.10 (predicates as first-class units) — there's nothing to track until predicates are callable-by-name and we can wrap the call site.

**6.4: WhyInt explanations** -- COMPLETE
WhyInt slot defined (Text dataType).

**6.5: MoreInteresting/LessInteresting ordering** -- COMPLETE
Both slots defined with inverse pair wiring. Verified by test.

---

## Phase 7: Definition Representations and Generators

**Why last:** These are refinements that improve efficiency and enable deeper exploration but aren't required for the core heuristic repertoire to function.

### Issues

**7.1: Multiple definition types**
FastDefn, FastAlg, CompiledDefn, UnitizedDefn, IterativeDefn, RecursiveDefn. Allow heuristics to switch between representations and compile slow definitions into fast ones.

**7.2: Generator slot**
Enable concepts to specify how to generate new instances systematically. NNumber's Generator `((0) (ADD1) (old))` means: start at 0, apply ADD1 to get the next, reuse previous results.

**7.3: Applics enrichment** -- COMPLETE
Applic entries now carry `args []string`, `output string`, and `direct bool` alongside the pre-existing `taskNum/target/result`. Pre-7.3 entries keep working — extras are optional.

- `record-applic (opName argList output --)` DSL builtin appends rich applic entries. H-RunOnExamples now calls it after each successful op application (binary and unary branches), so e.g. SetUnion gets entries like `{args:[SetOfNumbers SetOfPrimes], output:SetUnion-on-SetOfNumbers-SetOfPrimes, direct:true}`.
- Accessors: `applics-outputs`, `applics-args`, `applics-direct` — give H8/H10/H15/H20 the I/O data they need without exposing raw map internals to DSL.
- `list-of (v1..vn n -- list)` builtin for constructing argument tuples in DSL.
- Per-ThenPart records: `executeThenParts` now calls `trackThenPartRecord` after each slot, writing `<slot>Record = {successes, failures}` on the heuristic. Enables finer-grained credit assignment than the single overallRecord.
- IntApplics, DirectApplics, IndirectApplics slot units already existed in slots.cue; added IndirectApplics to Applics.subSlots.

Deferred: actual IntApplics/IndirectApplics population — nothing writes them yet, awaits H22 variants or H23 extension. DirectApplics is implicit (default direct=true).

**~~7.4: IfAboutToWorkOnTask slot~~** COMPLETE
Pulled forward during Phase 3.2. Wired as the first gate in `fireTaskRule` and included in IfPartSlots/programSlots.

**7.5: Structured conjecture system**
ProtoConjec as a proper unit type with ConjectureAbout, provenance, and status tracking. H1 and H16 create ProtoConjec units instead of printing text.

---

## Summary

| Phase | Focus | Issues | Status |
|---|---|---|---|
| 0 | CUE data layer | 7 | COMPLETE |
| 1 | Slot ontology | 5 + 1 bug | COMPLETE (bug 1.6 open) |
| 2 | Generalization/specialization | 8 | COMPLETE (+ stabilization fixes) |
| 3 | Rich HindSight | 6 | COMPLETE |
| 4 | Remaining heuristics | 10 (9 done, H8 blocked) | H1, H24 pending; H8 needs type predicates |
| 5 | Type hierarchy + operations | 12 | Not started |
| 6 | Interestingness + rarity | 5 | COMPLETE (scaffolding; population in 4b/5.10) |
| 7 | Definition representations | 5 | Not started |

**Dependencies:** Phases 4 and 5 can be parallelized. Phase 7 can start after Phase 1. Phase 6 requires Phases 4 and 5.

**Current system state (300-cycle math domain):**
- 111 units loaded (20 heuristics), grows to ~222 units
- 90 operations applied, 5 specializations, 1,620 conjectures
- 32 kills, 32 HindSight events, 32 credit halvings
- 496 duplicate detections, 10 mutations
- Stable and bounded growth
- H-RunOnExamples worth drops to 66 (correctly identified as prolific garbage creator)
- 0 generalizations (blocked by bug 1.6)

**Remaining issues: 36 across 5 phases + 1 bug.**
