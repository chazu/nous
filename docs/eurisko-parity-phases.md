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

**4.1: H1 -- "Specialize operations with >4/5 bad results"** -- COMPLETE
CUE heuristic in `domains/common/heuristics.cue`. `ifPotentiallyRelevant` gate: ArgU isA Op AND `applics-bad?` (≥5 applics AND ≥80% failures, per EURISKO's >4/5 rule). On fire, creates a `Conjec-HighFailureRate-<op>` ProtoConjec via `make-protoconjec` (kind=HighFailureRate, creditor=H1) and enqueues a priority-600 task on `<op>.specializations` — the H6-Specialize pipeline picks it up downstream.

New DSL builtin: `applics-bad? (unitName minTotal -- bool)` — threshold hardcoded at 80% (per EURISKO); `minTotal` is parameterized.

Test: `TestH1FlagsBadOp` — op with 1/4 success/failure fires; op with <5 total does not.

**~~4.2: H2 -- "Kill prolific-but-mediocre creators"~~** COMPLETE
Implemented as H2-KillGarbageCreator during engine stabilization. Scans children of heuristics, punishes those with 5+ children and 80%+ mediocre worth.

**4.3: H4 -- "Gather empirical data about new concepts"** -- COMPLETE
Implemented as a CUE heuristic in `domains/common/heuristics.cue`. Uses `ifFinishedWorkingOnTask` + the `new-units` builtin (both landed in Phase 3.3) to schedule an `examples` task for each newly-created unit that doesn't yet have examples populated. Fires reliably — a 100-cycle math run leaves ~13 pending "After synthesis, seek instances" tasks on the agenda.

**4.4: H8 -- "Find applics in generalizations' applics"** -- COMPLETE (via Phase 5.6 C.2)
Type-predicate defns landed on Set/List/Bag (`is-list?`) and Number (`is-int?`) — see 5.6 C.1 below. H8 itself in `domains/common/heuristics.cue` walks `generalizations`, filters each parent's `applics-args` by positional domain-type match via `apply-pred` on the dereferenced arg's data, and when the tuple passes it applies this op's defn and records the new applic. One-shot per op (`h8Ran` flag), bounded by configurable `h8Cap` (default 3).
H6-Specialize now links new specs to parent via `specializations` (inverse-wires `generalizations` on the spec) so H8's walk finds something.
300-cycle math run: 8 H8 firings, 2-3 propagated applics per firing across SetIntersect/SetUnion/GCD/DivisorsOf/SetDifference specializations.

**4.5: H10/H15 -- "Get examples from operations whose range is this type"** -- COMPLETE
Both in `domains/common/heuristics.cue`. H10 picks one random op from `isRangeOf` via `random-choice`; H15 iterates them all. Both use the Phase 7.3 `applics-outputs` builtin and `add-to-slot` (inverse-maintained) to append each recorded output as an Example. Dormant unless the target unit is in some op's range — works for Set/Number in the math domain once applics accumulate.

**~~4.6: H19/H19Criterial -- "Eliminate duplicate new units"~~** COMPLETE
H19-EliminateDuplicates implemented during engine stabilization. H19Criterial added as a CUE heuristic in Phase 4a — `ifFinishedWorkingOnTask` iterates new-units, compares all criterial slots against peers in its isA category, kills structurally-identical duplicates. Skips H-Specialize/H18-Generalize-created units (our H6 stores the restriction in `restrictedTo` rather than modifying criterial slots, so specs would false-positive against their parents).

**4.7: H20 -- "Run f on args used for other ops"** -- COMPLETE
Unit-focus heuristic: for each sibling op (same first-isA category) with recorded applics, take its arg tuples and apply CurUnit's alg to them, recording the new applic and creating a result unit. Uses new `apply-op-args (argList opName -- result)` builtin that resolves arg unit names to their `data` slot before running the defn. Caps at 3 cross-applications per firing to avoid flooding. 300-cycle math run produced ~115 cross-applications (e.g. SetDifference run on GCD's numeric args).

New DSL builtins: `apply-op-args`, `list-join`.

**4.8: H21 extension -- structured conjecture creation** -- COMPLETE (via Phase 7.5)
H-Conjecture now calls `make-protoconjec` alongside the existing print in both SetEqual and SubsetOf branches. Prints retained for observability.

**4.9: H22/H23 -- Interestingness evaluation** -- COMPLETE (dormant until seeded)
Both heuristics live in `domains/common/heuristics.cue`.
- **H22**: `ifFinishedWorkingOnTask` — when an examples-task finishes and the unit has an Interestingness predicate, schedule an intExamples-task.
- **H23**: `ifWorkingOnTask` on CurSlot==intExamples — iterates the unit's examples, runs the Interestingness predicate against each via new `is-interesting?` builtin, appends passers to intExamples via `add-to-slot` (inverse-maintained so isAInt auto-wires).

New DSL builtins: `is-interesting? (unit cand -- bool)` runs the unit's interestingness slot as a DSL program with env `candidate` bound; `add-to-slot (value unit slot --)` appends unique through Store.SetSlot so inverses fire.

Both dormant in current math-domain runs — no seeded unit has an Interestingness predicate. Will activate when H1 or Phase 5.10 introduces predicates, or when we hand-seed a few. Tested directly: `TestH23FillsIntExamples` and `TestH22SchedulesIntExamplesTask`.

**4.10: H24 -- "Do all examples satisfy the same rare predicate?"** -- COMPLETE
Dual-mode heuristic: ifPotentiallyRelevant (unit-focus) AND ifWorkingOnTask (task-focus on whyInt). For each unary predicate whose rarity is ≤0.3 (or yet-unknown) AND whose domain matches the examples' type, tests whether every data-bearing example with matching type returns true. If ≥4 examples passed, appends the predicate to the category's whyInt slot.

Companion heuristic **H24-Seeder** schedules whyInt tasks when an examples task finishes on a category with ≥4 examples (guarded against re-scheduling via `whyIntScheduled` flag).

**Bootstrap**: `SeedInitialAgenda` now also seeds whyInt tasks for categories with ≥4 examples pre-populated from CUE (priority 700), so categories like Set/Number/PrimeNum get H24 runs even when their examples never require an H-FindExamples pass.

**Bugs fixed along the way** (both surfaced by H24):
- `anyToValue` now handles `[]any` recursively — the Rarity `[freq, numT, numF]` tuple was stringifying so `first` never returned a useful number. Filter was permanently accepting everything.
- `<`, `>`, `<=`, `>=` now compare as floats when either operand is a float. Fractional rarity thresholds (0.3) were truncating to int and misfiring.

**Honest state**: H24 plumbing works end-to-end. Real discovery density is low with our current 4-predicate set (only AlwaysT matches broadly, gets filtered after first firing; IsEmpty/IsSingleton are Set-typed so don't cross into Number-valued categories). Richer predicate seeding or learned predicates from H1 will give it more to work with.

---

## Phase 5: Expanded Type Hierarchy and Operations

**Why fifth:** Enriches the concept space for exploration. Each addition is independent and can be done incrementally.

### Issues

**5.1: OSet type and operations** -- COMPLETE (2026-04-20)

`OSet` added as `Set` specialization in `domains/math/types.cue`. Five op units (OSetUnion, OSetIntersect, OSetInsert, OSetDelete, OSetEqual) with corresponding order-preserving DSL builtins (`oset-union`, `oset-intersect`, `oset-insert`, `oset-delete`, `oset-equal?`) that linear-scan inputs without sorting. Seed instances `OSetOfNumbers` (ascending) and `OSetOfPrimesDesc` (descending); the descending seed makes order preservation observable to heuristics. Engine smoke test (`TestOSetUnionPreservesOrderViaEngine`) guards against silent regression to canonicalizing set-*. OSetDifference and ReverseOSet deferred.

**5.2: OPair and Pair types** -- PARTIAL (OPair + H25/H26 complete; Pair and ReverseOPair deferred)

Added `OPair` category in `domains/math/pairs.cue`. Instances carry `data=[a, b]` and are materialized on-demand by H25/H26 with deterministic names `OPair-<a>-<b>` + `unit-exists?` dedupe.

H25 and H26 heuristics in `domains/common/heuristics.cue` mirror H27/H28 for binary predicates. Same three-limb interest gate (`worth >= 600 OR isAInt OR rarity[0] < 0.3`). On fire, iterate `pred.domain[0].examples × pred.domain[1].examples` (Cartesian) capped by the configurable `pairCap` slot (default 50). Each satisfying pair (H25) or failing pair (H26) becomes an OPair instance; the resulting unit names populate the new `SatisfyingSetFor<pred>` / `FailingSetFor<pred>` category's examples. >=4 examples seeds a downstream whyInt task so H24 can discover further interesting predicates on the pair set.

Bumped `SetEqual` and `SubsetOf` to worth 700 in `domains/math/predicates.cue` — H-Conjecture uses the `set-equal?` / `set-subset?` DSL builtins directly rather than `apply-pred`, so predicate units never accrue rarity from conjecture generation. Worth-bumping is the simplest path to passing the H25/H26 gate on startup.

H-ExercisePreds extended to schedule a one-shot whyInt task on each BinaryPred (via a new `predFocusScheduled` flag) so that H25/H26 actually get a chance to see ArgU=pred in task-focus mode.

300-cycle math-domain shakeout produced: H25/H26 each firing twice (on SetEqual + SubsetOf), H27/H28 firing once each, 115 OPair instances, 6 new pair/element categories, specialization pipeline still producing units.

Deferred: unordered `Pair` type (separate abstraction — binary-pred analysis only needs OPair), `ReverseOPair` operation (future mutation target), `Relation` (emergent from H25 output — needs no new machinery).

**5.3: Projection operations** -- PARTIAL (2026-04-20)

Six of ten projection ops landed in `domains/math/operations.cue`: Proj1, Proj2 (OPair domain), FirstEle, LastEle, AllButFirst, AllButLast (OrdStruc domain via 5.4 classification). New DSL builtin `but-last` added. SecondEle, ThirdEle, AllButSecond, AllButThird deferred — redundant with `rest`-chain composition; add when a heuristic demands them. Engine smoke test `TestFirstEleAppliedToOSetOfPrimesDesc` guards OrdStruc dispatch.

**5.4: Structure type classification** -- PARTIAL (2026-04-20)

Six classification marker categories added in `domains/math/types.cue`: OrdStruc, UnOrdStruc, MultEleStruc, NoMultEleStruc, EmptyStruc, NonEmptyStruc. Each has no `defn` — pure marker categories queryable via `store.IsA` chain walks. **Instance-level tagging** (not type-level): the Ord/UnOrd/Mult/NoMult tags live on concrete instance units (SetOfNumbers, OSetOfPrimesDesc, SortedList, etc.), not on the abstract Set/List/Bag/OSet types. This avoids transitive `IsA` contradictions — since `OSet isA Set`, tagging Set with UnOrdStruc would make OSet transitively UnOrdStruc, contradicting its OrdStruc tag. The Ord/Mult classification category units therefore carry no `specializations` slot (the inverse wiring would re-create the contradiction via `generalizations`); EmptyStruc/NonEmptyStruc keep their instance-level specializations since there's no such conflict. SetOfSets and StructureOfStructures higher-order categories deferred — no concrete instances yet. Unblocks Phase 5.12 H29 and Phase 5.6 D Restrict.

**5.5: Per-type operations**
ListInsert/Delete/Intersect/Union/Difference, BagInsert/Delete/Intersect/Union/Difference. Each as a unit with defn and domain/range.

**5.6: Meta-operations with algorithms** -- PARTIAL (A, B, C.1, C.2 complete; D deferred)

Phase 5.6 was sliced into four independent pieces. Slices C.1 and C.2 shipped together and unblock H8 (Phase 4.4).

- **C.1 Type predicates** -- COMPLETE. New `is-int?`/`is-list?`/`is-string?` DSL builtins introspecting Value kind. `defn` slots added to `Number` (`is-int?`) and `Set`/`List`/`Bag` (`is-list?` — finer discrimination is semantic work, not type-kind work). `apply-pred` on a type unit now acts as a type test.
- **C.2 H8** -- COMPLETE (see 4.4 above).
- **A Transpose** -- COMPLETE (2026-04-19). `transpose-op` builtin + H-Transpose heuristic create `Transpose-<op>` for any BinaryOp; domain reversed, defn prefixed with `swap`. Commutativity handled reactively by H19-EliminateDuplicates. Plan: `docs/superpowers/plans/2026-04-19-transpose-and-compose.md`. Followup (2026-04-19): commutativity sampling added to `transpose-op` — op.defn is run on (a,b) and (b,a) for up to 3 sample pairs drawn from the domain type's data-bearing examples. If all pairs agree, no Transpose is created. Plus new `H-SemanticDup` heuristic kills Transpose/Compose units whose observed applics are fully reproduced by any generalization. Spec: `docs/superpowers/specs/2026-04-19-semantic-duplicate-ops-design.md`. Closes the "H19 doesn't prune commutative Transposes" observation from the Phase 5.6 A+B smoke run.
- **B Compose** -- COMPLETE (2026-04-19). `compose-ops` builtin creates `Compose-<f>-<g>` when range(f) == domain(g) as ordered string slices. Composed defn chains apply-op on f then g. H-Compose iterates Op.examples capped at 3/firing. H-CheckDomain deleted (its SelfCompose branch produced shell units without defns; falls out naturally from H-Compose with f=g).
- **D Restrict + InvertOp** -- deferred. Restrict partially exists via `restrictedTo` (H6-Specialize); InvertOp is genuinely complex and low priority.

**Followup TODO — H-SemanticDup creditor gate.** Current gate is `creditors contains H-Transpose or H-Compose` (narrowed from the original plan's `creditors != nil` to avoid killing H-Specialize outputs, which by design reproduce parent applics on a restricted domain). This list must be extended whenever a new meta-op heuristic lands that produces units whose correctness criterion is "behaviorally distinct from parent". Candidates that would need adding: `H-Invert` (Phase 5.6 D), `H-Curry`, `H-Distribute`, any future meta-op heuristic. Better shape for the future: mark meta-op output units with an explicit `isMetaOp: true` slot or a shared creditor category (e.g. `MetaOpHeuristic` in isA), and gate H-SemanticDup on that category — removes the "edit this CUE file every time" burden. See `internal/engine/engine_test.go:TestH19CriterialSparesSpecializations` for the failure mode this gate protects against.

**5.7: Choice operations**
RandomChoose, RandomSubset, GoodChoose, GoodSubset, BestChoose, BestSubset as unit concepts (they exist as behaviors in H3/H5 but need to be first-class units that other heuristics can discover and reason about).

**5.8: Logical operations as units**
AND, OR, NOT, Implies, TheFirstOf, TheSecondOf as operation units with defn and domain/range.

**5.9: Numeric operations as units** -- COMPLETE
Add, Multiply, Successor, Square landed as seed units in `domains/math/operations.cue` (Phase 5.9 + 5.11 plan, 2026-04-19). Each has worth 500, BinaryOp/UnaryOp isA as appropriate, [Number]→[Number] signatures, and seeded raw-literal examples matching the GCD/DivisorsOf precedent. H-RunOnExamples picks them up via the existing pipeline; no engine changes.

**5.10: Additional predicates** -- COMPLETE (core set + Rarity hook)
Predicates as first-class units were already usable (MemberOf/SubsetOf/SetEqual in `domains/math/predicates.cue`). This phase added:
- `IsEmpty`, `IsSingleton` unary set predicates
- `AlwaysT`, `AlwaysNIL` constant predicates (the key ConstantPred subcategory members)
- Rarity tracking hook in `apply-op`: any call to a unit whose isA includes "Pred" now increments the Rarity tuple `[freqTrue, numT, numF]` on the predicate unit. Unblocks H24 and the Phase 6.3 population gap.

Deferred: numeric comparison predicates (IEQP, IGEQ, IGREATERP, ILESSP) with Transpose — natural to add alongside 5.6 meta-ops, not needed yet.

**5.11: H25-H28 -- Predicate set analysis** -- PARTIAL (H27/H28 COMPLETE; H25/H26 deferred)

Unary satisfying/failing set heuristics in `domains/common/heuristics.cue`. When a unit-focus lands on an interesting UnaryPred, H27 creates `SatisfyingSetFor<pred>` and H28 creates `FailingSetFor<pred>` — new categories whose examples are the source domain's elements that respectively satisfy or fail the predicate.

Gate (nous extension of EURISKO's `HasHighWorth OR IsAInt`): `worth >= 600 OR isAInt != nil OR rarity[0] < 0.3`. The rarity limb lets H24's rare-predicate flags feed H27/H28 directly.

Each new category inherits `isA` from the source category, sets `generalizations=[source]`, `defn=<pred>` (for recomputability), and `creditors=[H27]`/`[H28]`. When the filtered set has >=4 members, a `whyInt` task is seeded on the new category so H24 can discover further interesting predicates — closing the pred→category→pred loop.

Uses the existing `apply-pred` builtin; no new Go primitive needed. One-shot dedupe via `unit-exists?`.

H25 / H26 (n-ary satisfying/failing) deferred — they need tuple evaluation that waits on Pair/Tuple types (Phase 5.2).

**Numeric comparison predicates** -- COMPLETE (2026-04-19)
IEQP, IGEQ, IGREATERP, ILESSP landed in `domains/math/predicates.cue` as BinaryPred units with [Number, Number]→TruthValue. Phase 5.10 Rarity hook populates rarity on invocation. Transpose variants (EURISKO pairs IGEQ↔IGREATERP and ILESSP) deferred to Phase 5.6A.

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

**6.3: Rarity tracking** -- COMPLETE
Rarity slot defined with format `[frequency-True, num-True, num-False]`. Population hook landed in Phase 5.10: `apply-op` checks if the target op isA Pred and, if so, updates Rarity on the predicate unit after each call. H24 can now read rarity values to find rare predicates.

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

**7.2: Generator slot** -- COMPLETE

Generator format: `{initial: [values], step: "<dsl program>"}`. The step program takes the previous value on the stack and leaves the next on top — iterating from the initial seeds to produce as many values as asked. `run-generator (unitName count -- list)` builtin in `internal/dsl/builtins_math.go` does the iteration.

`H-Generate` heuristic in `domains/common/heuristics.cue` fires one-shot (`generated` flag) on any unit with a generator. It produces `generateCount` values (default 10, configurable per-heuristic slot), materializes each as a fresh `<Unit>-gen-<i>` instance unit with isA=[Unit], data=value, and appends them to the source unit's examples slot.

Seeded on `Number` in `domains/math/numbers.cue` with counting generator `{initial: [0], step: "1 +"}`. EURISKO's `(old)` reuse hint is unimplemented — our step program sees only the last value. Revisit if a future generator needs the full history.

300-cycle math run: 10 Number-gen-* instances created on Number's first focus.

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

**7.5: Structured conjecture system** -- COMPLETE
`ProtoConjec` category added to `domains/common/types.cue`. New slots `ConjecKind`, `Status`, `Evidence`, `Statement`, `SupportCount` in `domains/common/slots.cue`. `ConjectureAbout`/`Conjectures` inverse pair already existed pre-phase.

New DSL builtin `make-protoconjec (kind aboutList statement creditor -- unitName)`: creates a ProtoConjec unit with readable name `Conjec-<kind>-<sorted-about>`. On re-derivation of the same (kind, sorted-about), returns the existing name and bumps `supportCount`. Sets `conjectureAbout` via `Store.SetSlot` so the inverse `Conjectures` auto-wires on each target.

H-Conjecture retrofit: SetEqual and SubsetOf branches now call `make-protoconjec` alongside the existing prints. H1 (Phase 4.1) creates `HighFailureRate` ProtoConjecs. H16 retrofit deferred — H16 does not yet fire with unit-creating output in current runs.

Tests: `TestMakeProtoConjec` (builtin round-trip, inverse, dedupe), `TestHConjectureCreatesProtoConjec` (engine end-to-end), `TestH1FlagsBadOp` (gate + conjec + spec task).

---

## Summary

| Phase | Focus | Issues | Status |
|---|---|---|---|
| 0 | CUE data layer | 7 | COMPLETE |
| 1 | Slot ontology | 5 + 1 bug | COMPLETE (bug 1.6 open) |
| 2 | Generalization/specialization | 8 | COMPLETE (+ stabilization fixes) |
| 3 | Rich HindSight | 6 | COMPLETE |
| 4 | Remaining heuristics | 10 | COMPLETE |
| 5 | Type hierarchy + operations | 12 | PARTIAL (5.1, 5.2, 5.3 partial, 5.4 partial, 5.6 A/B/C.1/C.2, 5.9, 5.10, 5.11 numeric-preds + H27/H28) |
| 6 | Interestingness + rarity | 5 | COMPLETE (scaffolding; population in 4b/5.10) |
| 7 | Definition representations | 5 | PARTIAL (7.2, 7.3, 7.4, 7.5 complete; 7.1 not started) |

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
