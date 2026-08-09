# Constraint and nogood learning vocabulary plan

## Status and authority

Status: proposed Part 3 lane-specific implementation plan, revision 2.

Revision 1 was committed at
`10eb2deafd4d8a203257026d0c7925a4f6eaba86` and blocked independently by all
three reviewers. Revision 2 distinguishes training and target completion
universes, closes role normalization, freezes the engine/search bridge and
MAC-CBJ algorithm, makes every generator/statistical stream executable, maps
the complete lifecycle ledger, defines one-shot reruns, removes the exhaustive
solution-output confound, and adds a pre-panel attainability gate.

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

The sole learned artifact is a `blocked-pair/v1` schema. It describes a
three-role graph-coloring motif in which assigning a two-valued anchor its color
shared with two mutually unequal pair variables leaves both pair variables with
the same sole remaining color.
The sole causal use is pruning that anchor decision before the ordinary search
policy explores its continuation. A prune is legal only after an
instance-specific, fully materialized one-completion certificate has been
checked.

A positive result would demonstrate bounded negative-knowledge construction,
alpha-invariant reuse, and a lifecycle work advantage over a fixed strong
conventional CSP policy. It would not demonstrate general clause learning,
arbitrary CSP solving, SAT/CDCL, scalable graph coloring, scheduling insight,
or automatic invention of the feature language. V1 deliberately supplies the
three-role/domain guard language and asks Nous to discover the necessary
topology within it.

The hypothesis is:

> Over the fixed held-out stream, the frozen Nous-learned blocked-pair
> nogood reduces total charged lifecycle work by at least 5% relative to
> maintaining arc consistency with conflict-directed backjumping, while both
> policies return the correct satisfiability result and every learned prune
> preserves the independent oracle's complete solution set.

Semantic competence and marginal utility are separate. Exact construction and
sound use of the schema can pass even when the lifecycle hypothesis is null.

## Preregistered manifest

The implementation exposes this exact JSON as a source constant and reproduces
it byte-for-byte in every report:

```json
{
  "experiment_version": "nogoods/v1",
  "seed_authority": "part3/nogoods/v1",
  "generator_version": "blocked-pair-csp/v1",
  "grammar_version": "three-role-three-edge-mask/v1",
  "semantics_version": "finite-neq-csp/v1",
  "oracle_version": "independent-exhaustive-coloring/v1",
  "baseline_version": "mac-cbj-mrv-degree/v1",
  "cost_version": "nogood-lifecycle-events/v1",
  "statistics_version": "paired-stratified-bootstrap/v1",
  "report_version": "nogood-trials/v1",
  "integrity_contract": "budgeted-transcript",
  "training_seeds": {"start": 831001, "count": 4, "step": 1},
  "competence_development_seeds": {"start": 831101, "count": 8, "step": 1},
  "competence_validation_seeds": {"start": 831201, "count": 16, "step": 1},
  "development_seeds": {"start": 832001, "count": 384, "step": 1},
  "validation_seeds": {"start": 833001, "count": 768, "step": 1},
  "locked_tasks": 1536,
  "value_count": 4,
  "minimum_variables": 3,
  "maximum_variables": 8,
  "maximum_edges": 18,
  "schema_roles": ["anchor", "pair-0", "pair-1"],
  "candidate_edge_masks": 8,
  "training_examples": 4,
  "maximum_training_completions": 2,
  "target_certificate_completions": 1,
  "training_work_cap": 40000,
  "policy_work_cap": 2000000,
  "engine_cycle_cap": 2000,
  "attributed_unit_cap": 200000,
  "report_byte_cap": 16777216,
  "minimum_primary_reduction": 0.05,
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
- between three and eight opaque variables, each with a nonempty explicit
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

### Search outcome and solution-set preservation

Every evaluated utility policy decides satisfiability. `satisfied` contains the
first witness reached under the frozen search order; `no-solution` means the
search space was exhausted. Policies do not enumerate a large common solution
set merely to establish the work endpoint.

The independent oracle enumerates the complete Cartesian product in descriptor
order without importing production, fixture, heuristic, engine, or experiment
code. It is consulted only after policy termination. It verifies the witness or
empty result and also returns the full sorted solution-set digest for the prune
audit. For every omitted branch, the oracle independently enumerates the set of
solutions under that prefix and requires it to be empty. Thus the learned
policy preserves the exact full solution set even though the task terminal is a
satisfiability decision. A false witness, false `no-solution`, nonempty omitted
branch, or oracle disagreement is mechanical invalidity.

## The blocked-pair schema language

### Fixed guard and learned topology

Every schema has three variable roles: anchor `a` and symmetric pair roles `x`
and `y`. A proposed binding satisfies the fixed guard only when:

1. `domain(a)` has exactly two colors `{blocked, escape}`;
2. `domain(x) = domain(y)` has exactly two colors `{blocked, only}`;
3. `blocked`, `escape`, and `only` are pairwise distinct;
4. the current decision literal is exactly `a = blocked`; and
5. `x` has the lower target variable descriptor position and `y` the higher.

The grammar's only learned field is a three-bit required-edge mask:

```text
bit 0: a--x
bit 1: a--y
bit 2: x--y
```

A mask matches a target binding when every required edge is present. Extra
edges are permitted. Mask identity is its unsigned integer `0..7`; role and
color aliases do not enter the key. All eight masks have equal representational
rights. Refinement adds one previously absent bit. The root is mask zero;
duplicate paths converge on one canonical candidate unit.

Mask 7 is sound: the anchor decision removes `blocked` from both pair variables,
leaving both equal to `only`, which violates their required inequality. Each
mask with one of the three edges absent has a concrete satisfying
counterexample in the declared language. That fact is oracle truth, not a
production shortcut; Nous must arrive at the mask through evidence.

### What counts as learning

The training corpus consists of one failed full-mask branch and three
counterexample branches, one for each single missing edge. Display aliases and
color descriptor positions differ on every seed. Before a candidate is scored,
the ordinary heuristic path must create:

- a candidate unit reached through explicit one-bit refinements;
- a binding/match record for each public training example;
- the complete residual pair Cartesian product for every matched example;
- one result record per completion;
- one agreement or counterexample record per example; and
- a selection record retaining every exact tie.

A match claims that the branch has no completion; a nonmatch abstains. A
candidate is training-exact when it matches the full-mask failed branch, every
matched branch is observed to have no completion, and it matches none of the
three satisfiable single-edge counterexamples. Any mask below 7 omits at least
one bit and therefore matches the satisfiable example missing that bit; only
mask 7 is exact. Selection occurs only after the complete eight-candidate
evidence barrier. Selecting mask 7 is therefore an empirical consequence of
the four examples, not a hard-coded return value.

Training completion universes are derived from actual example edges after the
anchor decision. A pair variable retains `blocked` exactly when its anchor edge
is absent. The full example and missing-pair-edge example contain the sole
completion `x=only,y=only`; a missing-anchor-edge example contains two residual
completions. The evidence barrier stores the exact sorted Cartesian key set.
Enumerating only `only` in a missing-anchor-edge example is invalid because it
would omit the satisfying completion `x=blocked,y=only` or its symmetric form.

After selection, promotion performs a separate proof across all 24 injective
substitutions of `blocked,escape,only` into the four colors. For each it
materializes `x=only,y=only`, evaluates all three required edges, and records
the conflict. The promoted artifact contains mask, guard version, evidence
boundary digest, completion digest, provenance, and schema semantic key. Any
satisfying completion prevents promotion.

The promoted schema's explicit quantified scope is only the guard above,
arbitrary opaque aliases, all 24 role-respecting color substitutions, target
bindings whose required edges exist, and the sole pair completion after both
anchor edges remove `blocked`. The one-or-two residual universe is a training
example rule, not part of a promoted target certificate. The schema makes no
claim about other domain shapes, constraint kinds, partial assignments, or
omitted variables.

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
a blocked pair; choose a role mapping; learn or promote a schema; backjump; enforce
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
instance certificate binding `a,x,y,blocked,escape,only`, then create and
evaluate the sole pair completion. A complete-evidence barrier checks its
canonical key, required edge/domain facts, conflict result,
each completion, artifact hash, target digest, and current decision digest.
Only then may it omit the represented branch. Certificate and match work are
policy-visible and charged. Certificates are binding- and decision-specific;
they cannot be reused for a different target binding or decision.

### Frozen search/engine bridge

All confirmatory policies use the same synchronous bridge at the same point in
the frozen MAC-CBJ loop. After search proposes a value, verifies its public
domain membership and already-assigned incident edges, binds the literal, and
records removal of the variable's other values—but before it creates or
enqueues any AC-3 arc for that decision—the driver pauses search. Guards always
read immutable public domains and edges, never MAC-reduced domains.

The driver inserts exactly one `NogoodRequest` unit containing policy-profile
hash, target digest, monotonically increasing request number, current literal,
sorted current assignment, immutable-domain digest, current reduced-domain
digest, exact-conflict-store digest, and artifact-store digest. It adds one
`ngConsiderPrune` task and runs the ordinary engine to agenda quiescence. It may
not insert a role, binding, match result, completion, certificate, or decision.

CUE heuristics must end the request with exactly one sealed
`NogoodDisposition`:

- `resume` records that no complete applicable proof was found;
- `propose-prune` names one frozen artifact, normalized binding, one completion
  record, one certificate, and one prune proposal; or
- `bridge-invalid` identifies an internal duplicate, stale, corrupt, ambiguous,
  over-cap, or incomplete bridge state.

Every disposition includes the request digest and authoritative digest of every
referenced unit. The Go adapter may check one already materialized disposition
for structural completeness and digest agreement. It may apply a prune only for
`propose-prune` whose completion record independently passes the production
one-record checks and complete-evidence barrier; it may not search for a
substitute or repair a record. A stale request number, different
assignment/domain digest, multiple dispositions, residual agenda task, or
`bridge-invalid` makes the policy and panel mechanically invalid.

`resume` continues with the exact AC-3 initialization that would have followed
the pause. `propose-prune` restores the pre-proposal domain snapshot, records
the omitted prefix, and returns failure with conflict set `{a}` to the same CBJ
caller; no AC-3 event for that decision occurs. `match-only` validates the same
proposal but deliberately executes the `resume` path. The primary
`mac-cbj-empty`, no-artifact, learned, match-only, corrupted, wrong-family, and
random policies all cross this bridge on every locally legal decision. An
explicit empty artifact store therefore pays the same request, agenda,
disposition, and adapter path as the learned store. Standalone MAC-CBJ without
the bridge is diagnostic only and cannot become the primary comparator.

Certificates are occurrence-specific: even the same binding/literal under a
different current assignment or reduced-domain digest must be rematerialized.
No cross-prefix, cross-request, or cross-policy certificate cache exists. This
is conservative and makes target certification cost part of every causal use.

Wrong-family, corrupted-mask, random-mask, reset, and no-artifact controls
receive byte-identical public cases and legal operations. Corrupted mask 3
removes edge `x--y`; wrong-family uses a domain-separated valid mask-7 artifact
whose pair domains have three colors instead of two; random chooses one
mask uniformly from `0..6` once at the training freeze and reuses that one
frozen artifact for the entire panel. None may silently fall back to the correct
artifact.

## Training fixtures

### Deterministic streams and aliases

Every deterministic stream begins with canonical compact JSON containing only
UTF-8 strings and integers:

```text
["part3/nogoods/v1", panel, root, ordinal, purpose]
```

The exact panel tokens and stream scopes are:

| Use | `panel` | `root` | `ordinal` | `purpose` |
| --- | --- | --- | ---: | --- |
| training fixture | `training` | decimal training seed | `0` | each fixture purpose below |
| development competence | `competence-development` | decimal competence seed | `0` | each fixture purpose |
| validation competence | `competence-validation` | decimal competence seed | `0` | each fixture purpose |
| development utility | `development` | decimal task seed | `0` | each fixture purpose |
| validation utility | `validation` | decimal task seed | `0` | each fixture purpose |
| locked utility | `locked` | 64-character lowercase private root | global task ordinal | each fixture purpose |
| random control | `training` | `831001` | `0` | `random-control` |
| confirmatory bootstrap | panel token | first public panel seed or private root | replicate index | `bootstrap/nous-vs-mac` |
| confirmatory randomization | panel token | first public panel seed or private root | replicate index | `randomization/nous-vs-mac` |
| power outer sample | `power` | `832001` | outer index | `panel` |
| power inner bootstrap | `power` | `832001` | outer index | `bootstrap` |
| power inner randomization | `power` | `832001` | outer index | `randomization` |

Fixture purposes are exactly `variable-aliases`, `color-aliases`,
`variable-positions`, and `color-positions`. The implementation hashes the JSON
bytes with SHA-256, interprets bytes `[0:8]` and `[8:16]` as unsigned big-endian
integers, and passes them in that order to `math/rand/v2.NewPCG`. Every purpose
constructs a fresh PCG. There is no shared-state draw order between purposes.

Within a permutation purpose, Fisher-Yates consumes exactly one `Uint64N(i+1)`
for `i=n-1..1`. A fixture with `n` variables permutes source positions
`0..n-1`: training has three, the external-variable competence case and utility
have eight. Color positions always permute `0..3` independently of
color display aliases, so semantic color descriptor roles genuinely change
between tasks. Given color-position permutation `p`, `blocked=min(p0,p1)`,
`escape=max(p0,p1)`, `only=p2`, and `spare=p3`. This keeps the
causal value first under the frozen value order without fixing its descriptor
identity. No value is drawn for fixed cohort, distractor-domain, edge, or
missing-bit templates: those are functions only of ordinal as specified below.

Training display aliases come from `ta0,ta1,ta2,ta3` and
`tc0,tc1,tc2,tc3`. Every competence and utility panel uses the disjoint sources
`ha0,ha1,ha2,ha3,ha4,ha5,ha6,ha7` and `hc0,hc1,hc2,hc3`.
Alias and descriptor-position permutations are independent. This proves that a
training concrete key cannot equal a held-out concrete key. Semantic roles,
cohort names, and seeds never appear in a public problem object or Nous unit.

The random control makes exactly one draw `Uint64N(7)`, yielding a frozen mask
in `0..6`; it is not redrawn per task or panel. A statistics stream draws all
indices for one replicate in fixed cohort order and then increasing sampled
position. Power `panel` draws locked-size paired indices in that same order;
its `bootstrap` and `randomization` streams then draw all inner replicates in
increasing replicate order from their independent PCGs. Ties never consume
randomness.

Unit tests freeze the canonical JSON bytes, SHA-256 digest, two PCG seeds, first
four `Uint64` outputs, and resulting permutation for training seed 831001,
development seed 832001, validation seed 833001, and a fixed all-zero 64-hex
locked test root. A mismatch is mechanical invalidity.

The `variable-positions` golden vectors are:

| Panel/root | SHA-256 | PCG seeds | First four `Uint64` | Permutation |
| --- | --- | --- | --- | --- |
| `training` / `831001` | `ac5eb4be74b192caba90677858e7ca28c346c1329bfe16347b864600b1669f0d` | `12420563552428987082,13443358654286252584` | `16196928853732314818,13964589984287698613,14828555489593947842,3864106448264698449` | `0,1,2` |
| `development` / `832001` | `7ee119fb7276549805e9faaa614a7049671b62ab1466d6a66417e274b279e2a6` | `9142617286286660760,426147249446875209` | `1213905441346112375,1699159744660973649,1395889987714557996,13902222507069901990` | `5,2,1,4,3,6,0,7` |
| `validation` / `833001` | `c6ae755044795c25e589d32a073dc98744347a48470364294121d4445b8aca9e` | `14316509253064023077,16539983283958434183` | `4803874459439516066,14275222390261356384,10176726368503672072,10929098418971052156` | `6,0,1,4,7,3,5,2` |
| `locked` / 64 zeroes | `d1eae2c49127eaedc67e0983f6395fd5ece5fd712a4b44a8598253a172644af1` | `15126151632354011885,14302879928951594965` | `14781793926225946851,10566838600647845555,7622181505994308792,3324590857268697477` | `5,1,6,7,0,2,4,3` |

Training seeds `831001..831004` map directly, without rejection:

- `831001` is the full three-edge motif and has no completion after the blocked
  anchor decision; and
- `831002..831004` remove bits `0..2` respectively and supply the canonical
  satisfying completion for that missing edge.

Each fixture first permutes the three variable and four color descriptor
positions, then normalizes `x,y` in that target basis, and only then removes
the ordinal's edge bit. A pre-training audit requires normalized missing masks
to be exactly `{1,2,4}` once each. Its semantic edge mask is fixed by
ordinal, not sampled. Each contains only the three schema roles, so unrelated
context cannot leak the label. The observed failure/success is produced by
materialized completion evaluation, never stored as a fixture label.

The training terminal is one of `promoted`, `no-promotable-artifact`,
`ambiguous`, `budget-exhausted`, or `mechanical-invalid`. `ambiguous` retains
all exact masks and selects the canonical first only for diagnostics; V1's
promotion gate requires the unique mask 7. Budget exhaustion is genuine only
when the reconciled ledger reaches its exact cap. A false terminal class is
mechanical invalidity.

## Competence panels

Competence tests representation, construction, and sound application without
making a work-advantage claim. Development has eight cases and validation has
16:

- full motifs under unseen vertex and color aliases;
- full motifs with extra irrelevant edges;
- each of the three single-edge near misses;
- two-edge near misses;
- wrong anchor decisions;
- nonmatching domain shapes; and
- corrupted and missing completion records.

Development ordinals `0..7` are, in order: full motif, full motif with external
irrelevant variables, missing bits 0, 1, and 2, wrong decision `a=escape`, pair
domain size three, and one duplicate completion. Validation ordinals `0..15`
repeat those eight semantic cases under independent aliases, followed by
two-edge misses `{0,1}`, `{0,2}`, and `{1,2}`, anchor-domain size three,
unequal pair domains, one cross-decision completion, one stale-target
completion, and one color-position permutation audit. Corruption cases are expected rejections,
not fixture failures. The direct ordinal mapping is a source constant; cases
are never rejected or replaced. For every case the independent oracle checks
all relevant role bindings and all four-color completions. Competence passes
only if:

- Nous constructs and promotes exactly mask 7 from a fresh store;
- semantic keys and selection are invariant under aliases and pair-role
  permutation;
- every legal full-motif application is proposed;
- no near miss or wrong decision is pruned;
- every certificate contains exactly the one required completion;
- deleting, duplicating, corrupting, or cross-binding one record prevents the
  prune; and
- the store, action transcript, attribution ledger, and terminal record
  reconcile exactly.

Oracle competence work is reported separately and never credited as policy
work or a marginal advantage.

## Utility task stream

Every utility task has eight variables and four colors. Three variables form the
anchor/pair scope and five are distractors. Opaque aliases, descriptor
positions come from independent domain-separated streams; distractor domains
and non-scope edges are fixed ordinal templates. The semantic cohort is fixed
by ordinal before any stream is consumed.

The semantic utility template is constructed before alias and descriptor
permutation. Roles `a,x,y,d0,d1,d2,d3,d4` are assigned to the independently
permuted descriptor positions. Colors `blocked,escape,only,spare` are assigned to
independently permuted color descriptor positions and independently permuted
display aliases. In reusable and near-miss cases,
`domain(a)={blocked,escape}` and both pair domains are `{blocked,only}`.
Distractor domains cycle by task ordinal through these four fixed tuples:

```text
0: {all4}, {all4}, {all4}, {all4}, {all4}
1: {blocked,escape,spare}, {all4}, {escape,only,spare}, {all4}, {blocked,only,spare}
2: {all4}, {blocked,escape,only}, {all4}, {escape,only,spare}, {all4}
3: {blocked,escape,spare}, {escape,only,spare}, {all4}, {blocked,only,spare}, {all4}
```

Every reusable/near-miss template contains the five fan edges `a--d0` through
`a--d4`. Its distractor edges, selected by ordinal modulo four, are:

```text
0: d0--d1, d1--d2, d2--d3, d3--d4, d4--d0
1: template 0 plus d0--d2, d1--d3
2: template 1 plus d2--d4
3: every pair among d0..d4 except d3--d4
```

No distractor edge touches `x` or `y`. A `reusable` case adds all three schema
edges. A `near-miss` case removes bit `(cohortOrdinal mod 3)` from those three.
An `irrelevant` case gives every variable all four colors and uses its fan and
distractor edges plus `a--x,x--y`. An `independent-unsat` case gives every
variable all four colors except `d0` and `d1`, whose domains are the singleton
`{blocked}`, and adds `d0--d1` plus the selected distractor template; duplicate
edges are canonicalized before serialization. These rules cannot cross the
18-edge cap, and an independent exhaustive check requires every reusable escape
branch to be satisfiable.

Development's 384 tasks contain:

- 288 `reusable` cases with one full motif, a blocked anchor branch, and at
  least one solution through the escape color;
- 32 `near-miss` cases with one required edge absent;
- 32 `irrelevant` satisfiable cases with no guard-respecting binding; and
- 32 `independent-unsat` cases whose contradiction does not match the schema.

Validation doubles each count. Locked uses 1,536 tasks with exactly 1,152, 128,
128, and 128 cases respectively. Within every panel, contiguous ordinals map to
cohorts in the order listed; `cohortOrdinal` starts at zero within each cohort.
The task seed is the matching manifest arithmetic-sequence member. Locked uses
its root and global ordinal. There is no semantic rejection loop. Generation
then verifies the stated cohort, stated normalized-binding cardinality, bounds,
and oracle solution status; disagreement invalidates the panel instead of
regenerating it.

`blocked` has a lower color descriptor position than `escape`, so the causal
branch is visited first under the frozen value order, but neither display name
reveals its role. Descriptor positions and distractor
placement vary. Every reusable and near-miss task contains one normalized guard
binding; only reusable binds all three required mask edges. Irrelevant and
independent-unsat tasks have no guard binding. No policy may use the cohort
label.

## Policies and causal controls

All policies receive canonical byte-identical problems and emit the same
solution representation and ledger events.

### Conventional policies

- `chronological` performs descriptor-order depth-first search to the first
  witness with only locally assigned-edge checks.
- `forward-checking` removes inconsistent neighbor values after each decision,
  chooses minimum remaining domain, then maximum degree, then descriptor order.
- `mac-cbj-empty` is the primary conventional comparator. It maintains AC-3
  after every decision, uses the frozen bridge with an empty artifact store,
  records exact instance conflicts, and performs conflict-directed
  backjumping. Its concrete records are task-local and discarded after each
  task.
- `concrete-memo` stores the exact tuple `(training target digest, sorted
  variable display aliases, sorted color display aliases, complete concrete
  decision literal set)` for the failed training branch. It requests a prune
  only on byte-equal tuple match through the same bridge. Held-out namespaces
  are disjoint, so the pre-panel audit requires zero lookup matches and its
  search projection must equal `mac-cbj-empty`; lookup overhead remains charged.

The baseline implementations may be complete conventional algorithms, but
they receive no learned schema, cohort identity, oracle result, or hidden
generator field. The complete primary algorithm is frozen below rather than
deferred to source constants.

### Frozen MAC-CBJ algorithm

`mac-cbj-empty` and every learned/control policy execute this identical
recursive procedure. Current domains begin as immutable public domains.
Variable choice minimizes current domain cardinality, then maximizes static
degree in the original public graph, then minimizes variable descriptor
position. Values are tried by increasing color descriptor position.

Before selecting a variable, scan task-local exact nogoods by increasing
semantic key. A record matches only when every concrete literal is present in
the current assignment. A match returns the record's variable set as failure.
If all variables are assigned, independently check every domain and edge in
canonical order and return the witness.

For each value of selected variable `v`:

1. snapshot current domains, assignment, and deletion explanations;
2. propose and bind `v=value`; remove each other current value of `v` in color
   order with explanation `{v}`; reject immediately on a public-domain or
   already-assigned-edge violation;
3. invoke the frozen bridge before AC-3; a legal prune supplies failure set
   `{v}`, while `resume` continues;
4. initialize a FIFO AC-3 queue with `(neighbor,v)` for every neighbor in
   increasing descriptor position, suppressing duplicate queued pairs;
5. pop FIFO and revise `Xi` against `Xj`. For each current `Xi` value in color
   order, scan current `Xj` values in color order until inequality support is
   found. If none exists, delete it. Its explanation is the sorted union of the
   deletion explanations of every original `Xj` value unequal to it. If `Xi`
   changed but is nonempty, append `(Xk,Xi)` for every neighbor `Xk != Xj` in
   descriptor order unless already queued;
6. a domain wipeout returns the sorted union of explanations for every original
   value of the wiped variable. Otherwise recurse;
7. restore the exact snapshot after any failed value. Materialize the concrete
   nogood consisting of the current literal for each variable in the returned
   failure set and retain it by semantic key;
8. if the returned set omits `v`, emit one backjump and return it immediately;
   otherwise union `failure - {v}` into `conflicts(v)` and try the next value;
   and
9. after all values fail, return `conflicts(v) union {v}` and record that exact
   concrete nogood. The first witness stops the task. Root failure is
   `no-solution`.

An absent deletion explanation is the empty set. Union iterates source sets and
members in descriptor order, charges each attempted member, and deduplicates by
variable descriptor. Direct assigned-edge failure contains the two endpoint
decision variables; public-domain failure contains `{v}`. Queue membership is a
Boolean matrix restored with the search snapshot. No learned schema changes
variable/value order, explanation construction, exact-nogood retention, or
post-failure resumption.

Committed golden microtraces are required before any panel. They fix the
complete request sequence, decisions, FIFO contents after every transition,
revisions, deletion explanations, concrete nogoods, conflict unions,
backjumps, terminal/witness, and 12-category ledger vector for: a satisfiable
edge, an unsatisfiable equal-singleton edge, the blocked pair, a
nonchronological backjump, a task-local exact-nogood hit, and one bridge resume.
The baseline and learned adapter with an empty store must be bisimilar on every
golden trace.

### Learned and ablated policies

- `nous-generalized` runs the same `mac-cbj-empty` search plus frozen schema matching,
  full target certificate construction, and legal pruning.
- `no-artifact` runs the identical adapter path with an explicit empty artifact
  store and must be byte-equal to `mac-cbj-empty`, including adapter records;
  it is a second execution identity used to detect policy branching.
- `recomputed` discards the frozen schema and reruns the complete eight-candidate
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
12. semantic-key/cache/store read or write, agenda enqueue/dequeue, adapter
    request/disposition check, evidence-boundary check, or record write.

An event has one canonical tuple payload and appears once in an append-only
policy transcript. Appending the envelope that stores an already charged event
does not recursively create another event. The raw 12-category vector, scalar
sum, event count, and transcript SHA-256 are reported. A conservation test
independently reconstructs the scalar from the transcript. Every Nous heuristic
action is mapped to these events from its stored artifacts; every conventional
algorithm is directly instrumented with the same event definitions. Wall time,
Go allocations, CUE interpreter instructions, and post-termination oracle
audit are diagnostics and do not enter the primary scalar.

The following transition table is normative. “One” means one event with the
category above; loops emit once per visited member, including a duplicate or
failed member.

| Transition | Required events |
| --- | --- |
| schedule/pop one agenda task | one category 12 enqueue and one category 12 dequeue |
| evaluate heuristic antecedent | one category 2 per requested public/store atom; one category 12 disposition even when rejected |
| propose one candidate or one-bit edge | one category 1; one category 12 semantic-key read |
| duplicate refinement path | the same proposal/key-read events; no candidate write |
| novel candidate/refinement unit | proposal/key-read plus one category 12 unit write |
| propose a role/binding tuple | one category 1 per tuple and one category 2 per guard fact examined |
| schema mask match | one category 8 artifact read, three category 8 bit checks, and one category 2 per required target edge read |
| construct/evaluate a completion | one category 9 construction, one category 2 per domain read, one category 4 per inequality visited in canonical order, one category 12 result-record write |
| agreement or counterexample | one category 12 read of every referenced result and one category 12 record write |
| evidence barrier | one category 10 check per expected key, one per actual key, one per digest/count predicate, then one category 12 barrier-record write |
| score/select candidate | one category 8 evidence read per candidate, one category 12 selection comparison, and one category 12 tie/selection record write |
| promote/freeze artifact | one category 8 artifact write, one category 12 provenance write, and one category 12 boundary check/write |
| bridge request | one category 12 request write, one agenda enqueue/dequeue pair, and one category 12 request-digest check |
| bridge disposition | one category 12 disposition write and one category 12 check for request, assignment, domain, artifact, and authoritative digests individually |
| adapter validates proposal | one category 10 reference check per named unit and one category 10 check per barrier predicate; rejected proposals retain all checks |
| assignment decision | one category 3 proposal, bind, and eventual unbind/restore individually; one category 2 domain read and one category 4 per assigned incident edge examined |
| AC-3 transition | one category 6 per enqueue, dequeue, and revision attempt; one category 4 per support comparison; one category 5 per deletion, restoration, and empty check |
| exact nogood/CBJ | one category 12 per store lookup/read/write; one category 7 per attempted conflict-set insertion, union member, and backjump |
| cache operation | one category 12 for every hit, miss, read, or write; a hit never suppresses its lookup charge |
| witness/terminal | one category 11 per witness check or terminal classification and one category 12 terminal-record write |

The store/action audit reconstructs this table from authoritative units rather
than trusting driver counters. Conversely, the baseline transcript is replayed
through an independent reducer. Bisimulation tests require
`mac-cbj-empty`, `no-artifact`, and learned-with-empty-store to have identical
transition payloads and vectors; adapter-only records are therefore never free
or silently excluded. The report retains proposed duplicates, rejected
antecedents/actions, failed matches, cache misses, and terminal events, not only
successful artifacts.

Training candidate construction, evidence, artifact validation, promotion,
storage, and serialization are charged once to `nous-generalized`. At each
panel size that acquisition total is allocated evenly as the fixed fraction
`training_work / task_count` for paired estimates; the aggregate primary is
exactly training work plus all target work. Artifact load, matching,
certification, and application are charged per target. Conventional baselines
have zero cross-task acquisition but pay all task-local propagation and
conflict-record work. `recomputed` pays full acquisition per target.

The experiment aborts a policy only immediately before the event that would
exceed 2,000,000 units. A reconciled `budget-exhausted` record is an internally
well-formed policy terminal but makes the whole panel `invalid`, because the
satisfiability decision is unproved. It can never be relabelled a valid null.
Training similarly becomes invalid if its separately frozen 40,000-unit cap is
crossed. The attributed-unit and report-byte caps are separate mechanical
limits; crossing either makes the panel invalid.

## Evidence-free attainability bound

The 5% threshold is fixed only because the frozen grammar and ledger admit it
without consulting a development, validation, or locked seed. The constructive
non-panel witness uses utility template 3, semantic descriptor order
`a,d0,d1,d2,d3,d4,x,y`, color positions
`blocked,escape,only,spare`, and the full mask. It is not a member of any panel
seed sequence and can never enter inference.

A mechanical expansion of the transition table gives these conservative bounds
after cancelling the byte-identical bridge prefix:

| Witness path | Frozen bound and reason |
| --- | --- |
| empty-artifact blocked branch | `240 <= M <= 320`; FIFO must process the five fan neighbors before `x,y`, their dense-graph requeues, both pair reductions, and the final singleton inequality before CBJ returns |
| generalized blocked branch | `L <= 144`; one normalized pair scan, three mask atoms, one completion, its three edges, complete barrier, disposition, and prune return |
| nonreusable learned overhead | `L-M <= 24` on the witness near-miss/immediate-terminal controls |
| one acquisition | `A <= 40000`, enforced before any panel |

The bounds are obtained by substituting the exact variable domains and edge
queue into the frozen pseudocode and counting each table row; no probability,
wall time, or empirical fixture is used. The repository plan test must encode
this algebra as literal transition multiplicities, not call production,
baseline, fixture, or experiment code. Reviewers compare that test to the
document before implementation authority.

Repeating the witness in the locked 1,152/128/128/128 cohort proportions gives
the conservative best attainable numerator and denominator bounds:

```text
numerator <= 40000 + 1152*(144-240) + 384*24 = -61376
denominator <= 1536*320 = 491520
ratio <= -61376/491520 = -0.124869...
```

Thus a 5% lifecycle effect is attainable under the ledger even after maximum
acquisition and nonreusable overhead. This is only a feasibility bound, not a
forecast and not evidence for V1. If architecture review rejects any counted
transition or the literal algebra test disagrees, the plan must be revised
before implementation; development cannot be used to discover attainability.

## Statistical endpoint and classification

Mechanical gates are evaluated first. They require accepted-plan source and
dependency hashes, fresh-store schema reconstruction, competence success,
deterministic byte-equal rerun, exact transcript conservation, no forbidden
input, correct satisfiability terminals for every required policy/task, exact
full-solution-set preservation by every prune, and independent oracle
agreement.

The required utility matrix on development, validation, and locked is exactly
`chronological`, `forward-checking`, standalone `mac-cbj`, `mac-cbj-empty`,
`no-artifact`, `concrete-memo`, `nous-generalized`, `recomputed`, `corrupted`,
`wrong-family`, `random`, and `match-only` on every task. Every one is subject
to witness/no-solution parity, work-cap, deterministic-transcript, and terminal
gates. Every policy that proposes a prune is additionally subject to the full
omitted-solution audit. Only `nous-generalized` versus `mac-cbj-empty` enters
confirmatory inference; the other ten are frozen diagnostics and causal
controls, never a post-hoc replacement comparator.

Let `A` be the one observed training-acquisition total, `L_i` learned target
work, `M_i` `mac-cbj-empty` target work, `d_i=L_i-M_i`, and `N` the fixed task
count. The one primary point estimate is:

```text
R = (A + sum_i d_i) / sum_i M_i
```

For bootstrap replicate `r`, sample paired indices with replacement separately
inside each cohort, preserving all cohort counts. In cohort order reusable,
near-miss, irrelevant, independent-unsat and sampled-position order, draw
`Uint64N(cohortCount)` from that replicate's fresh PCG. Recompute both numerator
and denominator:

```text
R_r = (A + sum_j d_[I_rj]) / sum_j M_[I_rj]
```

`A` occurs once per replicate and is never resampled or duplicated as `A/N`.
Sort 10,000 `R_r` values as exact rational cross-products, breaking exact ties
by replicate index only for report order; interval endpoints are values at
zero-based indices 249 and 9749.

The paired randomization treats acquisition as one paired lifecycle block, not
`N` pseudo-observations. In replicate `r`, draw one swap bit for `A`, followed
by one bit per task in the fixed cohort/task order. Draw zero means sign `+1`;
one means `-1`:

```text
T_observed = A + sum_i d_i
T_r = sign_acquisition*A + sum_i sign_i*d_i
p = (1 + count(|T_r| >= |T_observed|)) / 10001
```

All arithmetic is signed 64-bit after a checked overflow preflight. This is a
symmetric paired test of the complete lifecycle difference; the ratio is the
effect estimate. Exact satisfiability and prune soundness are gates, not terms
that trade against work.

V1 is `valid-positive` only if all mechanical gates pass, the primary point
estimate is at most `-0.05`, the interval upper bound is below zero, the paired
randomization p-value is below 0.05, and mean target-only overhead on the pooled
near-miss/irrelevant cohort is no more than 10% above `mac-cbj-empty`. It is
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
the locked 1,152/128/128/128 counts. The outer `panel` PCG draws paired
development indices with replacement within each cohort in fixed cohort and
sample-position order. `A` is copied once into the synthetic panel. Each uses
2,000 inner stratified bootstrap
replicates, 2,000 independently seeded within-task label-swap replicates, and
the same equations, recomputed denominators, point, interval, p-value, and
nonreusable-harm gates. Inner bootstrap endpoints are sorted indices 49 and
1949. Inner randomization draws the one acquisition sign first and has Monte
Carlo denominator 2,001. Power streams use canonical JSON
`["part3/nogoods/v1","power",832001,outerOrdinal,purpose]`, where purpose is
exactly `panel`, `bootstrap`, or `randomization`; bootstrap and randomization
derive a fresh PCG from that tuple and never share state. Power is the passing
fraction and must be at least 0.80 before validation or locked execution is
reachable.

Validation may be invoked once after a committed power-positive development
report. Its guard exclusively creates
`.nous/nogoods-v1-validation-receipt.json`, generates and serializes all public
fixture bytes once, and executes two fresh stores/policy sets internally from
those same immutable bytes. The two canonical reports must be byte-equal; only
the first execution supplies empirical work, while the second is separately
reported integrity audit work. The guard then exclusively writes
`.nous/nogoods-v1-validation-report.json` and finalizes the receipt. A crash or
any existing receipt consumes/refuses the attempt. Validation must pass
competence and all mechanical gates; its empirical result is evidence, not a
tuning gate. After the committed result, all implementation and plan paths are
frozen.

Locked execution follows the repository's one-shot pattern. One guarded API
used by the CLI and package callers requires an unlock token naming the exact
clean `HEAD`, a committed prerequisite manifest, accepted implementation
reviews, accepted development/validation reports, power at least 0.80, no
`go.work` or module replacement, and canonical repository/domain paths. The
manifest hashes every protected input and its committed bytes. The guard
rejects existing receipt/report paths or symlinks, exclusively creates and
fsyncs a `claimed` receipt, reads a fresh 32-byte root from `crypto/rand`, and
records `started` before deriving a fixture. A crash consumes the attempt.

Inside that one locked invocation, every generated canonical fixture byte slice
is retained in memory and supplied independently to two fresh complete policy
runs. Their canonical reports, transcript hashes, terminals, witnesses, and
work vectors must be byte-equal before inference. As in validation, only the
first run contributes empirical work and the second is labelled integrity
audit. This is the deterministic rerun; the private root is never regenerated
and no second locked invocation exists.

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
- all eight masks and every one-bit refinement edge;
- the unique exact selection of mask 7 from the four fixed structures;
- all 24 color substitutions and the sole promoted completion;
- exhaustive three-role agreement with the independent oracle;
- alpha-renaming, semantic color-position, and pair-role permutation invariance;
- rejection of missing, duplicate, corrupted, cross-target, cross-decision,
  and stale certificate records;
- no prune for every single-edge near miss and wrong anchor decision;
- a successful prune before ordinary continuation evaluation;
- byte-equal no-artifact and baseline solution/transcript projections;
- concrete-memo failure under complete alias renaming;
- corrupted, wrong-family, random, recomputed, and match-only behavior;
- AC-3 and CBJ microcases independent of production semantics;
- satisfiability parity and complete omitted-branch solution-set audit for every
  development fixture;
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
