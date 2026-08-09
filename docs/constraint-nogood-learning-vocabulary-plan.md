# Constraint and nogood learning vocabulary plan

## Status and authority

Status: proposed Part 3 lane-specific implementation plan, revision 1.

This document narrows Vocabulary 1 of the accepted
[Part 3 vocabulary research program](vocabulary-research-program-v3.md). It is
not yet implementation authority. The exact committed revision must receive
independent architecture, constraint-semantics, and experimental-validity
acceptance before production or experiment code is added.

The lane identity is fixed by the umbrella:

- domain pack: `domains/nogoods`;
- seed authority: `part3/nogoods/v1`;
- production package: `internal/vocab/nogoods`;
- fixture package: `internal/nogoodfixture`;
- experiment package: `internal/nogoodexp`; and
- CLI: `nous nogood-trials`.

This lane is independent of the terminated Part 2 program and of every prior
Nous vocabulary. It does not repair, resume, or reinterpret a Part 2 phase. It
does not use Mu, PUDL, Kubernetes objects, scheduling fixtures, failure
minimization, hidden teachers, or protected receipts from another experiment.

## Research question and claim boundary

V1 asks whether ordinary Nous heuristics can convert failed finite-CSP branches
into one sound, alpha-invariant generalized nogood and then use that artifact to
prune the corresponding branch in later renamed problems while preserving the
complete solution set.

The sole learned artifact is a `blocked-triangle/v1` schema. It describes a
four-role graph-coloring motif in which assigning a two-valued anchor its color
shared with a three-color inequality triangle makes the triangle impossible.
The sole causal use is pruning that anchor decision before the ordinary search
policy explores its continuation. A prune is legal only after an
instance-specific, fully materialized eight-completion certificate has been
checked.

A positive result would demonstrate bounded negative-knowledge construction,
alpha-invariant reuse, and a lifecycle work advantage over a fixed strong
conventional CSP policy. It would not demonstrate general clause learning,
arbitrary CSP solving, SAT/CDCL, scalable graph coloring, scheduling insight,
or automatic invention of the feature language. V1 deliberately supplies the
four-role/domain guard language and asks Nous to discover the necessary
topology within it.

The hypothesis is:

> Over the fixed held-out stream, the frozen Nous-learned blocked-triangle
> nogood reduces total charged lifecycle work by at least 10% relative to
> maintaining arc consistency with conflict-directed backjumping, while both
> policies enumerate exactly the independent oracle's complete solution set.

Semantic competence and marginal utility are separate. Exact construction and
sound use of the schema can pass even when the lifecycle hypothesis is null.

## Preregistered manifest

The implementation exposes this exact JSON as a source constant and reproduces
it byte-for-byte in every report:

```json
{
  "experiment_version": "nogoods/v1",
  "seed_authority": "part3/nogoods/v1",
  "generator_version": "blocked-triangle-csp/v1",
  "grammar_version": "four-role-six-edge-mask/v1",
  "semantics_version": "finite-neq-csp/v1",
  "oracle_version": "independent-exhaustive-coloring/v1",
  "baseline_version": "mac-cbj-mrv-degree/v1",
  "cost_version": "nogood-lifecycle-events/v1",
  "statistics_version": "paired-stratified-bootstrap/v1",
  "report_version": "nogood-trials/v1",
  "integrity_contract": "budgeted-transcript",
  "training_seeds": {"start": 831001, "count": 7, "step": 1},
  "competence_development_seeds": {"start": 831101, "count": 8, "step": 1},
  "competence_validation_seeds": {"start": 831201, "count": 16, "step": 1},
  "development_seeds": {"start": 832001, "count": 24, "step": 1},
  "validation_seeds": {"start": 833001, "count": 48, "step": 1},
  "locked_tasks": 96,
  "value_count": 4,
  "minimum_variables": 4,
  "maximum_variables": 8,
  "maximum_edges": 18,
  "schema_roles": ["anchor", "triangle-0", "triangle-1", "triangle-2"],
  "candidate_edge_masks": 64,
  "training_examples": 7,
  "certificate_completions": 8,
  "policy_work_cap": 2000000,
  "engine_cycle_cap": 2000,
  "attributed_unit_cap": 200000,
  "report_byte_cap": 16777216,
  "minimum_primary_reduction": 0.10,
  "maximum_nonreusable_harm": 0.10,
  "alpha": 0.05,
  "confidence_interval": "paired-stratified-bootstrap-two-sided-95",
  "paired_test": "paired-stratified-randomization-two-sided",
  "bootstrap_replicates": 10000,
  "randomization_replicates": 10000,
  "bootstrap_indices_zero_based": [249, 9749],
  "development_power_outer_replicates": 2000,
  "development_power_inner_replicates": 2000,
  "development_power_randomization_replicates": 2000,
  "power_bootstrap_indices_zero_based": [49, 1949],
  "minimum_locked_power": 0.80,
  "tie_policy": "canonical-mask-then-semantic-key",
  "mutation_enabled": false
}
```

The seed notation means exactly `start + i*step` for zero-based
`0 <= i < count`. These values and every stream derived from them are disjoint
from prior Nous experiments. Validation is unreachable unless the development
power gate passes. Locked task roots are generated only after the one-shot
locked guard is claimed; no locked root or fixture is present in source.

Any post-review change to the grammar, generator, semantics, work events,
policy, panel mapping, thresholds, or statistics creates `nogoods/v2` and
preserves V1 evidence.

## Finite CSP semantics

### Public problem object

A `finite-neq-csp/v1` problem contains:

- four opaque color atoms, represented canonically only by descriptor ordinal;
- between four and eight opaque variables, each with a nonempty explicit
  subset of those colors;
- an undirected, duplicate-free set of at most 18 binary inequality edges;
- a duplicate-free ordered partial assignment; and
- a canonical variable order independent of display names.

The only primitive constraint is inequality. A complete assignment is a
solution exactly when each variable receives an allowed color and the endpoints
of every edge receive different colors. A partial assignment is locally
consistent when every assigned value is allowed and every fully assigned edge
is satisfied. Absence of an edge means absence of a constraint; absence of an
assignment is not a wildcard value.

Variable names and color strings are presentation aliases. Semantic identity
uses descriptor positions, domain bitsets, canonical edge pairs, and assignment
pairs. Duplicate variables, duplicate assignments, self edges, unknown colors,
empty domains, reordered descriptor positions, and bounds violations are
invalid rather than normalized away.

### Search outcome

Every evaluated utility policy enumerates solutions, not merely the first
solution. Its successful terminal contains the sorted complete solution set,
count, and SHA-256 digest. `no-solution` is successful only when the set is
empty. Search order may change work but cannot change this object.

The independent oracle enumerates the complete Cartesian product in descriptor
order without importing production, fixture, heuristic, engine, or experiment
code. It is consulted only after policy termination. A solution-set mismatch,
false `no-solution`, false success, or oracle disagreement is mechanical
invalidity.

## The blocked-triangle schema language

### Fixed guard and learned topology

Every schema has four variable roles: anchor `a` and symmetric triangle roles
`x`, `y`, and `z`. A proposed binding satisfies the fixed guard only when:

1. `domain(a)` has exactly two colors `{blocked, escape}`;
2. `domain(x) = domain(y) = domain(z)` has exactly three colors
   `{low, high, blocked}`;
3. `blocked`, `escape`, `low`, and `high` are pairwise distinct;
4. the current decision literal is exactly `a = blocked`; and
5. the triangle roles are normalized by their target descriptor positions.

The grammar's only learned field is a six-bit required-edge mask over canonical
pairs:

```text
bit 0: a--x       bit 3: x--y
bit 1: a--y       bit 4: x--z
bit 2: a--z       bit 5: y--z
```

A mask matches a target binding when every required edge is present. Extra
edges are permitted. Mask identity is its unsigned integer `0..63`; role and
color aliases do not enter the key. All 64 masks have equal representational
rights. Refinement adds one previously absent bit. The root is mask zero;
duplicate paths converge on one canonical candidate unit.

Mask 63 is sound: the anchor decision removes `blocked` from all three triangle
variables and the remaining pairwise-inequality triangle cannot be colored with
only `low` and `high`. Each mask with one of the six edges absent has a concrete
satisfying counterexample in the declared language. That fact is oracle truth,
not a production shortcut; Nous must arrive at the mask through evidence.

### What counts as learning

The training corpus consists of one failed full-mask branch and six
counterexample branches, one for each single missing edge. Display aliases and
color permutations differ on every seed. Before a candidate is scored, the
ordinary heuristic path must create:

- a candidate unit reached through explicit one-bit refinements;
- a binding/match record for each public training example;
- eight explicit triangle completions for every matched example;
- one result record per completion;
- one agreement or counterexample record per example; and
- a selection record retaining every exact tie.

A candidate agrees with an example only when its prediction from the fully
materialized completions equals that example's observed branch outcome. A
match is a claim that the branch has no completion; a nonmatch abstains. A
candidate is training-exact when it matches the full-mask failed branch, every
matched branch is observed to have no completion, and it matches none of the
six satisfiable single-edge counterexamples. A candidate with no positive match
is ineligible. Any mask below 63 omits at least one bit and therefore matches
the satisfiable example missing that bit; only mask 63 is exact. Selection
chooses the first training-exact candidate in ascending canonical mask order
only after the complete 64-candidate evidence barrier. Therefore selecting
mask 63 is an empirical consequence of the seven examples, not a hard-coded
return value.

After selection, promotion performs a separate competence proof across all 24
permutations of the four color roles. For each permutation, it materializes all
eight assignments of `low` or `high` to `x,y,z`, evaluates all six required
edges, and records the conflict. The promoted artifact contains mask, guard
version, evidence boundary digest, completion digest, provenance, and schema
semantic key. Any satisfying completion prevents promotion.

The schema's explicit quantified scope is only the guard above, arbitrary
opaque aliases, all 24 role-respecting color substitutions, target bindings
whose required edges exist, and the eight triangle completions after the anchor
decision. It makes no claim about other domain shapes, constraint kinds,
partial assignments, or omitted variables.

## Production/heuristic boundary

Pure Go in `internal/vocab/nogoods` may perform only bounded one-object or
one-step operations:

- parse and canonicalize one explicit CSP, assignment, mask, or binding;
- report one requested domain-membership or edge-membership fact;
- extend one explicit partial assignment by one decision;
- evaluate one explicit partial or complete assignment;
- refine one mask by one bit;
- test one explicit schema atom against one explicit binding;
- evaluate one explicit three-role completion; and
- validate the internal consistency of one already materialized certificate
  record.

No production function may enumerate variables, bindings, masks, refinements,
completion universes, search trees, conflicts, solutions, or certificates; find
a triangle; choose a role mapping; learn or promote a schema; backjump; enforce
arc consistency; or return whether a set of records is exhaustive.

Ordinary CUE heuristics in `domains/nogoods` own candidate population,
one-bit refinement tasks, binding proposals, completion-unit construction,
evidence aggregation, counterexample retention, complete-evidence barriers,
selection, promotion, target matching, certificate construction, and prune
proposals. Mutation is disabled. The existing engine, agenda, VM, common pack,
credit mechanics, and all other vocabularies remain byte-unchanged.

An experiment adapter may serialize public fixtures, run the engine, and audit
the resulting store/transcript. It may not insert the winning mask, create a
missing heuristic artifact, choose a production refinement, repair provenance,
or reinterpret an incomplete run as success.

## Independent layers

The implementation has four acyclic layers:

1. `internal/vocab/nogoods` contains only pure bounded semantics and may import
   the standard library;
2. `internal/nogoodfixture` constructs public cases and may import only the
   standard library plus production data types needed for serialization;
3. `internal/nogoodoracle` independently decodes canonical public JSON and
   performs exhaustive solution enumeration without importing production,
   fixture, experiment, engine, seed, DSL, or CUE packages; and
4. `internal/nogoodexp` owns policies, meters, panel guards, statistics, and
   reports.

The conventional baseline is implemented in `internal/nogoodbaseline` from
the canonical public JSON and cannot import production or Nous heuristic code.
Source and dependency tests enforce these boundaries. The fixture, baseline,
and oracle may duplicate small semantic definitions; sharing production
evaluation code would defeat independence.

## Information boundary and freeze

Before training starts, all policies may see the grammar, public training CSPs,
partial assignments, and the outcome of each charged explicit completion they
request. They may not see the selected mask, a precomputed conflict core, an
oracle solution set, a minimal counterexample explanation, or another policy's
transcript.

Training ends at one immutable evidence boundary. The selected schema and its
promotion certificate are serialized, hashed, and loaded afresh for every
later policy. Development, validation, and locked cases do not exist in the
store at this boundary. Held-out target truth is unavailable until a policy
terminates.

At utility time every policy sees the complete public CSP. The learned policy
may match only its frozen artifact. Before any prune it must materialize an
instance certificate binding `a,x,y,z,blocked,escape,low,high`, then create and
evaluate all eight triangle completions. A complete-evidence barrier checks the
eight distinct canonical keys, required edge/domain facts, conflict result for
each completion, artifact hash, target digest, and current decision digest.
Only then may it omit the represented branch. Certificate and match work are
policy-visible and charged. Certificates are binding- and decision-specific;
they cannot be reused for a different target binding or decision.

Wrong-family, corrupted-mask, random-mask, reset, and no-artifact controls
receive byte-identical public cases and legal operations. Corrupted mask 31
removes edge `y--z`; wrong-family uses a domain-separated valid mask-63 artifact
whose fixed guard has two triangle colors instead of three; random chooses one
mask uniformly from `0..63` using its frozen stream. None may silently fall
back to the correct artifact.

## Training fixtures

### Deterministic streams and aliases

Every deterministic stream begins with canonical compact JSON containing only
UTF-8 strings and integers:

```text
["part3/nogoods/v1", panel, root, ordinal, stream]
```

For public panels, `root` is the decimal task seed from the manifest and
`ordinal` is zero. For locked tasks, `root` is the 64-character lowercase root
hex and `ordinal` is the zero-based task ordinal. The implementation hashes the
JSON bytes with SHA-256, interprets bytes `[0:8]` and `[8:16]` as unsigned
big-endian integers, and passes them in that order to
`math/rand/v2.NewPCG`. Stream names are exactly `variable-aliases`,
`color-aliases`, `role-positions`, and `control-mask`. Each alias stream applies
Fisher-Yates to a fixed source list; the role-position stream independently
permutes the eight descriptor positions. The control-mask stream draws one
unbiased integer from `[0,64)` by `Uint64N(64)`. No stream is reused across
panels or purposes.

The fixed source aliases are `vax,vby,vcz,vdw,vex,vfy,vgz,vhw` and
`amber,blue,coral,denim`. They carry no semantic role before permutation.
Training and competence four-role fixtures use the first four variable aliases
after permutation. Utility fixtures use all eight. Source lists, semantic
roles, cohort names, and seeds never appear in a public problem object or Nous
unit.

Training seeds `831001..831007` map directly, without rejection:

- `831001` is the full six-edge motif and has no completion after the blocked
  anchor decision; and
- `831002..831007` remove bits `0..5` respectively and supply the canonical
  satisfying completion for that missing edge.

Each fixture permutes color aliases, variable aliases, and triangle display
order from independent streams. Its semantic edge mask is fixed by ordinal,
not sampled. Each contains only the four schema roles, so unrelated context
cannot leak the label. The observed failure/success is produced by materialized
completion evaluation, never stored as a fixture label.

The training terminal is one of `promoted`, `no-promotable-artifact`,
`ambiguous`, `budget-exhausted`, or `mechanical-invalid`. `ambiguous` retains
all exact masks and selects the canonical first only for diagnostics; V1's
promotion gate requires the unique mask 63. Budget exhaustion is genuine only
when the reconciled ledger reaches its exact cap. A false terminal class is
mechanical invalidity.

## Competence panels

Competence tests representation, construction, and sound application without
making a work-advantage claim. Development has eight cases and validation has
16:

- full motifs under unseen vertex and color aliases;
- full motifs with extra irrelevant edges;
- each of the six single-edge near misses;
- two-edge near misses;
- wrong anchor decisions;
- nonmatching domain shapes; and
- corrupted and missing completion records.

Development ordinals `0..7` are, in order: full motif, full motif with external
irrelevant variables, missing bits 0, 1, 2, 3, 4, and 5. Validation ordinals
`0..15` repeat those eight semantic cases under independent aliases, followed
by two-edge misses `{0,3}`, `{1,4}`, and `{2,5}`, wrong decision `a=escape`,
anchor-domain size three, triangle-domain size two, one duplicate completion,
and one cross-decision completion. Corruption cases are expected rejections,
not fixture failures. The direct ordinal mapping is a source constant; cases
are never rejected or replaced. For every case the independent oracle checks
all relevant role bindings and all four-color completions. Competence passes
only if:

- Nous constructs and promotes exactly mask 63 from a fresh store;
- semantic keys and selection are invariant under aliases and triangle-role
  permutation;
- every legal full-motif application is proposed;
- no near miss or wrong decision is pruned;
- every certificate contains exactly the eight required completions;
- deleting, duplicating, corrupting, or cross-binding one record prevents the
  prune; and
- the store, action transcript, attribution ledger, and terminal record
  reconcile exactly.

Oracle competence work is reported separately and never credited as policy
work or a marginal advantage.

## Utility task stream

Every utility task has eight variables and four colors. Four variables form the
anchor/triangle scope and four are distractors. Opaque aliases, descriptor
positions, distractor domains, and non-scope edges come from independent
domain-separated streams. The semantic cohort is fixed by ordinal before those
streams are consumed.

The semantic utility template is constructed before alias and descriptor
permutation. Roles `a,x,y,z,d0,d1,d2,d3` are assigned to the independently
permuted descriptor positions. Colors `blocked,escape,low,high` are assigned to
the independently permuted color aliases. In reusable and near-miss cases,
`domain(a)={blocked,escape}` and each triangle domain is
`{blocked,low,high}`. Distractor domains cycle by task ordinal through these
four fixed tuples:

```text
0: {all4}, {all4}, {all4}, {all4}
1: {all4}, {blocked,escape,low}, {all4}, {escape,low,high}
2: {blocked,escape,high}, {all4}, {blocked,low,high}, {all4}
3: {all4}, {escape,low}, {all4}, {blocked,high}
```

The distractor edge templates, selected by the same ordinal modulo four, are:

```text
0: d0--d1, d1--d2, d2--d3
1: d0--d1, d1--d2, d2--d3, d3--d0
2: d0--d1, d0--d2, d0--d3
3: d0--d1, d2--d3, d1--d2
```

No template edge connects a distractor to a schema role. A `reusable` case adds
all six schema edges. A `near-miss` case removes bit
`(cohortOrdinal mod 6)` from those six. An `irrelevant` case gives every
variable all four colors and uses only its distractor edge template plus the
path `a--x,x--y,y--z`. An `independent-unsat` case gives every variable all
four colors except `d0` and `d1`, whose domains are both the singleton
`{blocked}`, and adds `d0--d1` plus its distractor template; duplicate edges
are canonicalized by template construction before serialization. These rules
cannot cross the 18-edge cap.

Development's 24 tasks contain:

- 12 `reusable` cases with one full motif, a blocked anchor branch, and at
  least one solution through the escape color;
- 4 `near-miss` cases with one required edge absent;
- 4 `irrelevant` satisfiable cases with no guard-respecting binding; and
- 4 `independent-unsat` cases whose contradiction does not match the schema.

Validation doubles each count. Locked uses 96 tasks with exactly 48, 16, 16,
and 16 cases respectively. Within every panel, contiguous ordinals map to
cohorts in the order listed; `cohortOrdinal` starts at zero within each cohort.
The task seed is the matching manifest arithmetic-sequence member. Locked uses
its root and global ordinal. There is no semantic rejection loop. Generation
then verifies the stated cohort, unique binding, bounds, and oracle solution
status; disagreement invalidates the panel instead of regenerating it.

The blocked anchor value is first in the schema binding's domain but display
names do not reveal it. Descriptor positions and distractor placement vary.
Every reusable task contains a unique guard-respecting binding. Near-miss and
irrelevant tasks contain none. Independent-unsat tasks may contain local
conflicts but cannot match the schema. No policy may use the cohort label.

## Policies and causal controls

All policies receive canonical byte-identical problems and emit the same
solution representation and ledger events.

### Conventional policies

- `chronological` performs descriptor-order depth-first enumeration with only
  locally assigned-edge checks.
- `forward-checking` removes inconsistent neighbor values after each decision,
  chooses minimum remaining domain, then maximum degree, then descriptor order.
- `mac-cbj` is the primary conventional comparator. It maintains AC-3 after
  every decision, uses the same MRV/degree tie order, records exact instance
  conflicts, and performs conflict-directed backjumping. Its concrete records
  are task-local and discarded after each task.
- `concrete-memo` replays exact training variable/color decision keys. Held-out
  aliases should make it behaviorally identical to `mac-cbj` apart from charged
  lookup overhead.

The baseline implementations may be complete conventional algorithms, but
they receive no learned schema, cohort identity, oracle result, or hidden
generator field. AC-3 queue order, CBJ conflict-set union, solution enumeration,
and tie-breaking are specified by source constants and tested against textbook
microcases.

### Learned and ablated policies

- `nous-generalized` runs the same `mac-cbj` policy plus frozen schema matching,
  full target certificate construction, and legal pruning.
- `no-artifact` runs the identical adapter path with an explicit empty artifact
  store and must be byte-equal to `mac-cbj` except for declared adapter records.
- `recomputed` discards the frozen schema and reruns the complete 64-candidate
  acquisition and promotion lifecycle independently for every target before it
  may prune.
- `corrupted`, `wrong-family`, and `random` use the controls fixed above.
- `match-only` constructs matches and certificates but does not prune, isolating
  causal use from artifact possession.

Every skipped refinement is recorded as `(target, decision, artifact,
certificate, omitted-prefix)`. Post-termination audit enumerates the skipped
prefix independently and confirms that it contains no solution. One false
prune, missing solution, or unrecorded skip invalidates the complete lane.

## Common lifecycle-work ledger

The primary endpoint is the sum of fixed unit-cost semantic events. Each event
below costs exactly one; no post-observation weights exist:

1. candidate or one-bit refinement construction;
2. public domain-membership or edge-membership read;
3. assignment proposal, bind, or unbind;
4. inequality predicate evaluation;
5. domain deletion, restoration, or empty-domain check;
6. AC-3 arc enqueue, dequeue, or revision attempt;
7. conflict-set insertion, union member, or backjump;
8. schema storage write, load/read, guard atom check, or mask-bit check;
9. completion construction or completion predicate evaluation;
10. certificate record creation or barrier predicate check;
11. solution insertion, duplicate check, or terminal classification;
12. cache read, cache write, evidence-boundary check, or terminal-record write.

An event has one canonical tuple payload and appears once in an append-only
policy transcript. Appending the envelope that stores an already charged event
does not recursively create another event. The raw 12-category vector, scalar
sum, event count, and transcript SHA-256 are reported. A conservation test
independently reconstructs the scalar from the transcript. Every Nous heuristic
action is mapped to these events from its stored artifacts; every conventional
algorithm is directly instrumented with the same event definitions. Wall time,
Go allocations, CUE interpreter instructions, and post-termination oracle
audit are diagnostics and do not enter the primary scalar.

Training candidate construction, evidence, artifact validation, promotion,
storage, and serialization are charged once to `nous-generalized`. At each
panel size that acquisition total is allocated evenly as the fixed fraction
`training_work / task_count` for paired estimates; the aggregate primary is
exactly training work plus all target work. Artifact load, matching,
certification, and application are charged per target. Conventional baselines
have zero cross-task acquisition but pay all task-local propagation and
conflict-record work. `recomputed` pays full acquisition per target.

The experiment aborts a policy only immediately before the event that would
exceed 2,000,000 units. A reconciled `budget-exhausted` terminal is mechanically
valid but cannot satisfy solution equality and therefore cannot support a
positive marginal claim. The attributed-unit and report-byte caps are separate
mechanical limits; crossing either makes the panel invalid.

## Statistical endpoint and classification

Mechanical gates are evaluated first. They require accepted-plan source and
dependency hashes, fresh-store schema reconstruction, competence success,
deterministic byte-equal rerun, exact transcript conservation, no forbidden
input, exact solution-set equality for every confirmatory policy/task, no false
prune, and independent oracle agreement.

The one primary empirical endpoint is total lifecycle work for
`nous-generalized` relative to `mac-cbj` over the complete panel. For each task,
the paired effect uses target work plus the fixed acquisition fraction. The
point estimate is:

```text
(mean(nous target + acquisition/N) - mean(mac-cbj target))
----------------------------------------------------------------
                     mean(mac-cbj target)
```

Ten thousand stratified paired bootstrap replicates resample paired task
indices with replacement within the four fixed cohorts, retaining cohort
counts. The two-sided 95% interval uses sorted zero-based indices 249 and 9749.
The paired stratified randomization test swaps policy labels within task and
uses an independent domain-separated stream. Its two-sided Monte Carlo p-value
is `(1 + count(|replicate effect| >= |observed effect|)) / 10001`. Exact
solution equality is a gate, not a term that can trade against work.

V1 is `valid-positive` only if all mechanical gates pass, the primary point
estimate is at most `-0.10`, the interval upper bound is below zero, the paired
randomization p-value is below 0.05, and mean target-only overhead on the pooled
near-miss/irrelevant cohort is no more than 10% above `mac-cbj`. It is
`valid-null` when mechanical gates pass but any empirical gate fails. It is
`invalid` when a correctness, soundness, leakage, accounting, provenance,
determinism, boundary, or frozen-protocol gate fails.

Post-freeze target work, certification work, pruned branches, concrete-conflict
count, solution count, artifact precision, generalization distance, storage,
amortization crossover, and all ablation contrasts are secondary diagnostics.
They cannot create a positive label.

## Development, validation, and locked sequence

Development is run only after implementation-candidate review. It supplies the
only effect/variance population used for the frozen power simulation. Exactly
2,000 synthetic locked panels are built by resampling the development strata to
the locked 48/16/16/16 counts. Each uses 2,000 inner stratified bootstrap
replicates, 2,000 independently seeded within-task label-swap replicates, and
the same point, interval, p-value, and nonreusable-harm gates. The inner Monte
Carlo p-value denominator is 2,001. Power streams use canonical JSON
`["part3/nogoods/v1","power",832001,outerOrdinal,purpose]`, where purpose is
exactly `panel`, `bootstrap`, or `randomization`; bootstrap and randomization
derive a fresh PCG from that tuple and never share state. Power is the passing
fraction and must be at least 0.80 before validation or locked execution is
reachable.

Validation may run once after a committed power-positive development report.
It must pass competence, all mechanical gates, and a deterministic byte-equal
rerun. Its empirical result is evidence, not a tuning gate. After the committed
validation result, all implementation and plan paths are frozen.

Locked execution follows the repository's one-shot pattern. One guarded API
used by the CLI and package callers requires an unlock token naming the exact
clean `HEAD`, a committed prerequisite manifest, accepted implementation
reviews, accepted development/validation reports, power at least 0.80, no
`go.work` or module replacement, and canonical repository/domain paths. The
manifest hashes every protected input and its committed bytes. The guard
rejects existing receipt/report paths or symlinks, exclusively creates and
fsyncs a `claimed` receipt, reads a fresh 32-byte root from `crypto/rand`, and
records `started` before deriving a fixture. A crash consumes the attempt.

The only locked receipt and report are
`.nous/nogoods-v1-locked-receipt.json` and
`.nous/nogoods-v1-locked-report.json`. No other API accepts `locked`, returns a
locked fixture, or exposes the root. An integrity-clean empirical miss is
`valid-null`; mechanical failure is `invalid`; either finalizes the receipt.

If development power is below 0.80, V1 ends as a development-stage valid null,
validation and locked stay unopened, and the negative feasibility result is
documented. The panel is not enlarged and thresholds are not changed.

## Required tests and audits

Before any empirical run, tests must cover:

- every valid and invalid CSP encoding and canonical identity rule;
- all 64 masks and every one-bit refinement edge;
- the unique exact selection of mask 63 from the seven fixed structures;
- all 24 color substitutions and all eight promoted completions;
- exhaustive four-role agreement with the independent oracle;
- alpha-renaming and triangle-role permutation invariance;
- rejection of missing, duplicate, corrupted, cross-target, cross-decision,
  and stale certificate records;
- no prune for every single-edge near miss and wrong anchor decision;
- a successful prune before ordinary continuation evaluation;
- byte-equal no-artifact and baseline solution/transcript projections;
- concrete-memo failure under complete alias renaming;
- corrupted, wrong-family, random, recomputed, and match-only behavior;
- AC-3 and CBJ microcases independent of production semantics;
- complete solution-set parity for every development fixture;
- exact work conservation under success, no-solution, and budget exhaustion;
- production/oracle/baseline import and source-separation checks;
- absence of Part 2 paths, seeds, receipts, reports, and `.git/nous-attempts`
  reads in source constants, runtime open traces, dependencies, stores, and
  reports;
- deterministic stream vectors, reruns, statistics, and report encoding;
- validation and locked refusal before their gates; and
- one-shot locked refusal after an existing or partial receipt.

Implementation review must inspect the actual source boundary, generated
population, evidence barrier, transcript conservation, conventional baseline,
oracle independence, panel guard, and command path. Assertions in a report are
not substitutes for adversarial tests or independently recomputed evidence.

## Implementation and delivery sequence

1. Commit this plan without implementation.
2. Obtain independent architecture, constraint-semantics, and experimental
   reviews of that exact commit.
3. Revise and recommit until all three reviewers return unqualified acceptance.
4. Implement the pure production semantics and exhaustive unit tests.
5. Implement the CUE population/refinement/evidence/promotion loop and prove the
   store contains the intended first-class artifacts.
6. Implement independent fixture, baseline, oracle, meter, statistics, report,
   CLI, and guards.
7. Commit the implementation candidate and obtain the same three independent
   adversarial reviews of code plus protocol conformance.
8. Correct blockers without observing a protected panel; recommit and repeat
   review until accepted.
9. Run and commit the competence/development evidence and result document.
10. Run validation only if power authorizes it; run locked only if every prior
    guard authorizes it.
11. Record the lane as valid-positive, valid-null, or invalid in the Part 3
    capability matrix, commit, and push before beginning Vocabulary 2.

## Research relationship

Dechter's work on
[backjumping and constraint learning](https://doi.org/10.1016/0004-3702%2890%2990046-3)
frames learned constraints as reusable explanations of failed search. GRASP's
[conflict analysis and recording](https://doi.org/10.1109/12.769433) shows the
same principle in propositional search. V1 adopts their central causal test—can
recorded failure prevent repeated work?—but intentionally does not implement
their general algorithms.

The exhaustive target certificate is stricter and less scalable than normal
clause learning. That is deliberate: in this first bounded trial, a work win
counts only after the learned abstraction has paid for an independently
inspectable proof that its prune is sound. A valid null would therefore be
useful evidence that this proof regime overwhelms the small search saving and
should be redesigned before broader nogood grammars are attempted.
