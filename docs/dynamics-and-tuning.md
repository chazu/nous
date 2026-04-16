# Dynamics and Tuning

Empirical study of nous's control loop on the math domain over 300-cycle
runs. Documents the observed dynamics of specialization/generalization
chains, worth decay, the credit system, and the immune-system heuristics.
Captures three bugs found, the fixes, and the resulting steady-state
behavior.

## Reproduction

All measurements below are from:

```sh
go build -o nous ./cmd/nous
./nous run -domain math -cycles 300 -v 2 > /tmp/run.log 2>&1
```

The math domain seeds at 111 units (of which 20 are heuristics). Runs use
deterministic RNG (seed 42) so numbers are reproducible.

## Bugs found and fixed

Three distinct bugs conspired to distort dynamics. All three have been
fixed; commits are `941e4d2`, `bcd7acd`, `70335ae`.

### 1. Generalization pipeline blocked (`criterial-slots` case mismatch)

Slot keys stored on units are lowercase-first (`"domain"`, `"range"`,
`"defn"`). Slot definition units are PascalCase (`"Domain"`, `"Range"`).
The `criterial-slots` builtin checked `store.IsA(slotKey, "CriterialSlot")`
with the raw lowercase key — a unit named `"domain"` does not exist, so
the check always returned false. H17-ChooseGenSlots, H3-RandomSlot, and
H5-Criterial all iterate `criterial-slots` to schedule work, so none of
them produced any tasks.

Specialization still worked because `math/H-Specialize` bypasses
`criterial-slots` and iterates the `domain` slot directly, but
generalization had no such escape hatch.

**Fix**: capitalize the first letter of the slot key before the IsA
lookup (`internal/dsl/builtins.go:slotDefName`).

**Effect**: 300-cycle run produced 16 generalizations (was 0).

### 2. Worth clamping bypassed by DSL `set-slot`

`Unit.SetWorth` clamped to `[0, 1000]`, but the generic `Unit.Set` did
not. DSL `set-slot` writes through `Store.SetSlot` → `Unit.Set`, so
heuristics like H-Conjecture that subtract 200 from worth accumulated
unbounded negative values. Observed worths reached -3700.

**Fix**: clamp worth in `Unit.Set` (`internal/unit/unit.go`).

**Effect**: all worths stay within `[0, 1000]`.

### 3. Task-only heuristics firing in unit-focus

`fireUnitRule` evaluated `ifPotentiallyRelevant` and `ifTrulyRelevant` but
silently ran `thenCompute` for any heuristic that had neither. H17, H18,
H3, H5, H6, and H-FindExamples have only `ifWorkingOnTask` — they are
slot-triggered by design. `fireUnitRule` was executing their `thenCompute`
unconditionally on every unit-focus.

Concrete failure mode: on unit-focus of `Set`, H17 saw `criterial-slots`
return `["examples"]` (because H-FindExamples had just populated
`Set.examples`), picked a random example, found its generalizations, and
queued a generalization task on Set — a non-Op type. That produced
spurious units like `Set-gen-Set`, `List-gen-List`, `Number-gen-Number`.

**Fix**: `fireUnitRule` skips heuristics with no unit-phase condition
(`internal/engine/fire.go`), and resets `CurSlot` on unit-focus so stale
bindings from the prior task don't leak.

**Effect**: no `X-gen-X` noise on non-Op types. Specialization chain count
dropped from 64 to 18 (removing spurious firings).

### 4a. Discovered results never examined (H-RunOnExamples)

The flagship EURISKO discovery `Primes \ Odds = {2}` is surfaced by
running `SetIntersect` on `(SetOfPrimes, SetOfEvens)` and having
`H-CheckExtremes` detect the singleton. But `H-RunOnExamples` only
created the result unit at worth 500 and moved on — nothing scheduled
any heuristic to inspect the data. Results sat unexamined because
worth-500 derived units never won unit-focus against heuristics
(600-800) and seed math units (600-700).

**Fix**: `H-RunOnExamples` now emits a priority-300 examine task on each
newly created result, so `H-CheckExtremes`, `H-Conjecture`,
`H-BoostInteresting`, and `H-PenalizeTrivial` all fire via task
dispatch. Priority 300 is low enough not to crowd out H-Specialize's
spec tasks at 1000/600.

**Effect**: `{2}` flagship now surfaces again (and from multiple
operator paths). Conjectures per 300 cycles climbed to 1869 (all
unique). Kills rose from 3 to 11 — derived units that reduce to existing
sets now get penalized and culled.

### 4b. Orphan tasks on killed units caused agenda loops

When the examination path actually started penalizing and killing
derived units, a latent bug surfaced: `Agenda` had no
`PurgeUnit`. Tasks referencing a killed unit stayed in the queue.
Popping such a task fires every heuristic with `ArgU = killed-unit-name`;
`get-slot` on a missing unit returns nil for every slot. Any
`slot=nil` guard (e.g. `H-ExploreSlots`'s `explored=nil`) matches,
`H-ExploreSlots` re-queues its own trigger task, and the engine spins
forever on a dead unit.

Observed: one killed unit's `.examples` task fired H-ExploreSlots 966
times across ~970 cycles, crowding out all other work.

**Fix**: `Agenda.PurgeUnit(unitName)` drops every pending task for the
named unit; `HandleDeletedUnit` calls it as part of the kill bookkeeping.

### 4. Immune-system heuristics locked out of unit-focus

`highestWorthUnfocused` skipped every unit with `Heuristic` in its isA
chain. That included heuristic _instances_, not just the `Heuristic`
meta-unit. Consequently H2-KillGarbageCreator and H-AnalyzeApplics — the
two heuristics whose job is to evaluate other heuristics — never ran.
The pruning layer was architecturally dormant.

**Fix**: skip only the `Heuristic` meta-unit itself
(`internal/engine/engine.go:highestWorthUnfocused`).

**Effect**: H2 now evaluates on every heuristic focus; H-AnalyzeApplics
fires 2× per 300-cycle run. Conjecture count jumped from 2103 to 4784
(heuristic focus runs first by worth, giving Sets more data-accumulation
time before H-Conjecture fires).

## Observed steady-state dynamics (post-fix)

### Unit growth

300-cycle math run, mutation disabled for clarity:

| Metric              | Count |
|---------------------|-------|
| Seed units          |   111 |
| Final units         |   539 |
| Specializations     |    15 |
| Generalizations     |    10 |
| Operations applied  |   ~465 |
| Conjectures (total) |  4784 |
| Conjectures (unique)|  ~1420 |
| Kills               |     3 |

### Chain depth distribution

Derived Op units carry `-on-X` (specialization) or `-gen-X`
(generalization) suffixes. The measured depth distribution is bounded and
Zipfian-ish:

| Depth | Count |
|-------|------:|
|   1   |    89 |
|   2   |    91 |
|   3   |    69 |
|   4   |    38 |
|   5   |     6 |

No chains beyond depth 5. The combinatorial surface area does grow at
each step, but H19-EliminateDuplicates and lower worths of machine-created
chain links naturally throttle exploration. No per-operation depth cap is
needed.

### Worth distribution (all units)

| Band           | Count | Share |
|----------------|------:|------:|
| low (<100)     |     3 |   0.6% |
| mediocre (100–400) | 28 |   5.5% |
| good (400–600) |   433 |  84.6% |
| high (≥600)    |    48 |   9.4% |

The "good" band dominates because most derived units enter at 500–600 and
few penalty paths exist today. Credit flows from `H-Conjecture` (−200
for redundant sets), `H-PenalizeTrivial` (−200 for empty data), and
downstream via `creditors` halving when a child is killed.

### Heuristic firing counts

Post-fix, 300-cycle run, mutation enabled:

| Heuristic                   | Fires |
|-----------------------------|------:|
| H-ExploreSlots              |   184 |
| H-FindExamples              |   179 |
| H-RunOnExamples             |   102 |
| H17-ChooseGenSlots          |    26 |
| H16-Generalize              |    26 |
| H-CheckDomain               |    22 |
| H-Conjecture                |    17 |
| H-Specialize                |    15 |
| H6-Specialize               |    15 |
| H-CheckExtremes             |    13 |
| H-BoostInteresting          |    12 |
| H18-Generalize              |    11 |
| H-PenalizeTrivial           |     9 |
| H19-EliminateDuplicates     |     7 |
| H-KillWorthless             |     3 |
| H-AnalyzeApplics            |     2 |
| H2-KillGarbageCreator       |     0 |

`H3-RandomSlot` and `H5-Criterial` fire zero times. They require a task
with `SlotName=specializations` and no `SlotToChange` extra — no
scheduler in the current ruleset produces such tasks. This is EURISKO
parity debt, not a bug: EURISKO's pipeline had additional unit-focus
triggers that queued these. Tracked under Phase 4 (remaining heuristics).

### Credit/worth decay trajectories

Representative heuristics over 300 cycles (all mutations disabled):

| Heuristic          | Initial | Final | Δ   |
|--------------------|--------:|------:|----:|
| H-RunOnExamples    |     750 |   400 | −350 |
| H3-RandomSlot      |     101 |   101 |    0 |
| H-Specialize       |     650 |   650 |    0 |
| H-ExploreSlots     |     500 |   500 |    0 |
| H2-KillGarbageCreator | 700  |   700 |    0 |

H-RunOnExamples loses worth because many of its results are duplicates
or trivial — H19 and H-PenalizeTrivial punish its creditors. This is the
credit system functioning correctly.

## Why H2-KillGarbageCreator stays dormant

H2's `ifTrulyRelevant` requires:
- a target heuristic to have ≥5 children, AND
- ≥80% of those children to be worth<400.

Current dynamics produce only 5.5% mediocre units overall, far below the
80% bar. H2 would start firing if:
- a mutant heuristic produced many bad children (mutations are rare now),
- chain exploration deepened (per-step worth decay would accumulate),
- penalty paths broadened (more heuristics push worth below 400).

H2's threshold is not tuned down because the current behavior is a true
negative, not a false negative. The system is not producing garbage
prolifically enough to need intervention.

## Tuning decisions

| Knob                    | Current  | Kept | Rationale |
|-------------------------|---------:|------|-----------|
| H2 child count          |        5 | yes  | EURISKO default; no prolific offender observed |
| H2 mediocre fraction    |      80% | yes  | High precision > high recall for kills |
| H-KillWorthless threshold |    100 | yes  | Three actual <100 units were caught this run |
| H-PenalizeTrivial delta |     -200 | yes  | Moves empty-result units out of the "good" band |
| H-Conjecture redundancy |     -200 | yes  | Same |
| H6/H-Specialize one-shot| present  | yes  | `specTaskAdded` flag keeps exploration from re-seeding |
| H16 one-shot            | present  | yes  | `genTaskAdded` flag same |

No thresholds were changed. The four fixes above were sufficient to
restore intended dynamics.

## Open questions for future runs

- Long-run behavior (>1000 cycles): does the mediocre fraction grow as
  exploration deepens, eventually tripping H2?
- Phase 3 (Rich HindSight) will add slot-change provenance that enables
  finer-grained kills — may allow a lower H-KillWorthless threshold.
- H3-RandomSlot / H5-Criterial activation (Phase 4): need a unit-focus
  trigger that queues SlotToChange=nil tasks.

## Reproduction commands

```sh
# Build and run
go build -o nous ./cmd/nous
./nous run -domain math -cycles 300 -v 1 > run.log 2>&1

# Unit/conjecture counts
grep "^units=" run.log
grep -c "^CONJECTURE:" run.log
grep -c "^Created generalized" run.log
grep -c "^Created specialized" run.log
grep -c "^Killing" run.log

# Heuristic firing table
grep -oE "Heuristic [A-Za-z0-9-]+ fired" run.log | sort | uniq -c | sort -rn

# Chain depth distribution
grep -oE '[A-Z][A-Za-z]+(-on-[A-Z][A-Za-z]+|-gen-[A-Z][A-Za-z]+)+' run.log \
  | sort -u \
  | awk '{
      n=gsub("-on-","",$0); m=gsub("-gen-","",$0);
      counts[n+m]++
    } END { for(k in counts) printf "depth=%d count=%d\n", k, counts[k] | "sort -n" }'

# Worth distribution
./nous run -domain math -cycles 300 -v 0 2>&1 \
  | grep -oE "^ +[0-9-]+ +[A-Za-z][A-Za-z0-9_-]+" \
  | awk '{ w=$1;
      if (w<100) b="low"; else if (w<400) b="med";
      else if (w<600) b="good"; else b="high";
      count[b]++
    } END { for (k in count) printf "%s=%d\n", k, count[k] }'
```
