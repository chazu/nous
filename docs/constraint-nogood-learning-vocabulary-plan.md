# Constraint and nogood learning vocabulary plan

## Status and authority

Status: proposed Part 3 lane-specific implementation plan, revision 9.

Revision 1 was committed at
`10eb2deafd4d8a203257026d0c7925a4f6eaba86` and blocked independently by all
three reviewers. Revision 2 distinguishes training and target completion
universes, closes role normalization, freezes the engine/search bridge and
MAC-CBJ algorithm, makes every generator/statistical stream executable, maps
the complete lifecycle ledger, defines one-shot reruns, removes the exhaustive
solution-output confound, and adds a pre-panel attainability gate.
Revision 2 was committed at
`0285755538c7837d614d44883e0a4d451f9d3743` and blocked on the conventional
comparator, full-task attainability math, one distractor topology, the actual
engine task seam, CBJ failure-set lifetime, acquisition randomization, evidence
persistence, and a duplicated random control. Revision 3 makes the utility
object a fixed branch-completion query, restores standalone MAC-CBJ as primary,
and closes those mechanical contracts. Revision 3 was committed at
`740bd7944b0e0cb2583961fa21c32a6ba8938b05` and blocked on one stale comparator
claim, the observable engine boundary, the fixed-decision root frame, artifact
authorization, evidence persistence and hashing, and literal attainability
counts. Revision 4 was committed at
`778816b77a7ff7b37ebad9c106eefd06602e0d42` and resolved those semantics but
was blocked on the umbrella's engine freeze, development rerun retention,
engine bookkeeping/profile charges, per-template bounds, aggregate adapter
validation, and an incomplete binary grammar. Revision 5 was committed at
`c5119a6776e91b372a1d1c042c9a055dd76761ab`; it was blocked on the common
`Heuristic` category dispatch, remaining task/certificate/prune/terminal
charges, power-cell support, and acquisition transcript ownership. Revision 6
resolves that complete review union. Revision 6 was committed at
`89efcc25f67fdc601bbdff122196894a497300ba`; architecture accepted it, while
theory found one missing agreement-result read and experimental review required
at least two development realizations in every semantic cell. Revision 7 made
those final mechanical corrections and was accepted at commit
`549f123eb4413b27e926f8e565e3af376030b3e8`. During pre-panel implementation,
the literal no-match cap test exposed an arithmetic contradiction: the
normative ledger requires three target-edge reads which the no-match subtotal
omitted. V1 is therefore withdrawn unexecuted; it has no competence,
development, validation, or locked evidence. Revision 8 gives the corrected
executable protocol V2 identity, reallocates those three reads between the
matcher and completion rows without changing the 125-event prune maximum,
sets the 81-event no-match maximum and 83-event hard cap, and corrects the
evidence-free attainability proof. No protected panel was observed.

Revision 8 was committed at `b13d9f23cf23ea389fb4745fd35159ac4f59e3be`
and blocked independently on the umbrella-frozen seed identity, the exact
scope of the no-match cap, the missing canonical development-report path,
nonreplayable locked fixture evidence, and an ambiguous cross-panel terminal
classification. Revision 9 resolves that complete review union while retaining
the already accepted ledger arithmetic. No panel was observed between
revisions.

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

V2 asks whether ordinary Nous heuristics can convert failed finite-CSP branches
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
or automatic invention of the feature language. V2 deliberately supplies the
three-role/domain guard language and asks Nous to discover the necessary
topology within it.

The hypothesis is:

> Over the fixed held-out stream, the frozen Nous-learned blocked-pair
> nogood reduces total charged lifecycle work relative to
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
  "experiment_version": "nogoods/v2",
  "seed_authority": "part3/nogoods/v1",
  "generator_version": "blocked-pair-csp/v1",
  "grammar_version": "three-role-three-edge-mask/v1",
  "semantics_version": "finite-neq-csp/v1",
  "oracle_version": "independent-exhaustive-coloring/v1",
  "baseline_version": "mac-cbj-mrv-degree/v1",
  "cost_version": "nogood-lifecycle-events/v2",
  "statistics_version": "paired-stratified-bootstrap/v1",
  "report_version": "nogood-trials/v2",
  "integrity_contract": "budgeted-transcript",
  "training_seeds": {"start": 831001, "count": 4, "step": 1},
  "competence_development_seeds": {"start": 831101, "count": 8, "step": 1},
  "competence_validation_seeds": {"start": 831201, "count": 16, "step": 1},
  "development_seeds": {"start": 832001, "count": 96, "step": 1},
  "validation_seeds": {"start": 833001, "count": 192, "step": 1},
  "locked_tasks": 384,
  "value_count": 4,
  "minimum_variables": 3,
  "maximum_variables": 8,
  "maximum_edges": 18,
  "schema_roles": ["anchor", "pair-0", "pair-1"],
  "candidate_edge_masks": 8,
  "training_examples": 4,
  "maximum_training_completions": 2,
  "target_certificate_completions": 1,
  "training_work_cap": 2000,
  "target_prune_work_cap": 128,
  "no_match_bridge_overhead_cap": 83,
  "policy_work_cap_per_task": 2000000,
  "bridge_task_pop_cap": 2000,
  "attributed_unit_cap": 200000,
  "report_byte_cap": 16777216,
  "fixture_bundle_byte_cap": 16777216,
  "transcript_event_cap_per_execution": 8000000,
  "transcript_event_cap_per_bundle": 16000000,
  "transcript_raw_byte_cap_per_execution": 1073741824,
  "transcript_raw_byte_cap_per_bundle": 2147483648,
  "transcript_gzip_byte_cap_per_execution": 1074000000,
  "transcript_gzip_byte_cap_per_bundle": 2148000000,
  "minimum_primary_reduction": 0.00,
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
policy, panel mapping, thresholds, or statistics creates the next experiment
version and preserves all earlier evidence. Revision 8 created experiment V2
because it changed a V1 threshold, but it did not have authority to rename the
umbrella-frozen `part3/nogoods/v1` seed namespace. Revision 9 restores that
namespace and all of its frozen streams while retaining V2 experiment, cost,
report, evidence-path, and unlock identities. Experiment V1 remains withdrawn
and unexecuted.

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

Every utility case is a branch-completion query. It supplies a locally legal
base prefix and one proposed decision literal; reusable cases supply
`a=blocked`. Every policy first validates and binds that same literal, then asks
whether this fixed branch has any completion. It never explores the anchor's
escape branch or a sibling of the supplied decision. `satisfied` contains the
first completion reached under the frozen search order; `no-solution` means the
fixed branch was exhausted. This is exactly the branch the learned artifact may
prune, so alternate-branch work cannot dilute or manufacture the endpoint.

The independent oracle enumerates the complete Cartesian product in descriptor
order without importing production, fixture, heuristic, engine, or experiment
code. It is consulted only after policy termination. It verifies the witness or
empty result and also returns the full sorted solution-set digest for the fixed-branch prune
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

The learned and bridge-control policies use one synchronous bridge only for the
supplied top-level branch decision. After the policy verifies its public domain
membership and already-assigned incident edges, binds the literal, and records
removal of the variable's other values—but before it creates or enqueues any
AC-3 arc—the driver pauses. Recursive decisions after `resume` use standalone
MAC-CBJ without another bridge. Guards always read immutable public domains and
edges, never MAC-reduced domains.

The driver inserts exactly one `NogoodRequest` unit containing policy-profile
hash, target digest, monotonically increasing request number, current literal,
sorted current assignment, immutable-domain digest, current reduced-domain
digest, exact-conflict-store digest, and artifact-store digest. It adds one
`ngConsiderPrune` task. It may not insert a role, binding, match result,
completion, certificate, or decision.

The bridge store is a normal `domains/common` plus `domains/nogoods` load. Its
exact `Examples("Heuristic")` result is
`["Heuristic","NG-H-ConsiderPrune"]`: the first entry is the common category
unit itself and the second is the sole lane heuristic. There is no other common-
pack or other-domain heuristic instance. The category unit has no program; the
lane heuristic has exactly one `ifWorkingOnTask` program and one ThenPart action
program. A source-structure test compares the exact sorted
`Examples("Heuristic")` set, nonempty program slots, and program hashes with the
committed bridge profile once against each execution's canonical serialized
bridge-seed bytes, before any task store is decoded or any request exists. The
preflight performs one example query and two identity reads, reads all five
IfPart and six ThenPart slots on both units (22 slot reads), compares all 22 with
their required empty/nonempty state, hashes the two nonempty programs (two
source reads and two hashes), and compares the store/profile/source digests
(three comparisons): exactly 54 category-12 events. It is recorded in that
execution's acquisition transcript; primary inference charges it once inside
`A`, while the audit execution reports its independent copy outside empirical
work. Consequently an unrelated heuristic cannot create hidden work or change
a disposition. No per-request profile audit occurs.

The adapter checks `VM.InitError()` once, requires the agenda to be empty before
that task is inserted, then repeatedly calls `Agenda.Pop()` followed by
`Engine.WorkOnTask(task)` until `Agenda.Len()==0`. It never calls `Engine.Run`
or `Engine.WorkOnUnit`, so unit-focus, mutation, worth-growth, and periodic
engine behavior cannot occur. At most 2,000 tasks may be popped. Before each
`WorkOnTask`, the adapter requires the task's target unit to carry the current
request digest; a task for any other request is `bridge-invalid`. Bridge
heuristics may not delete units, and the adapter requires `VM.DeletedUnits` to
remain empty after every task, so the engine's unexported deletion-bookkeeping
path is unnecessary. `TaskNum`, agenda enqueue/dequeue, the fixed engine
bookkeeping surcharge below, and every materialized bridge heuristic fact/action
are charged by the ledger. Every policy/task decodes a fresh store from the
immutable audited bridge-seed bytes and creates a fresh agenda, VM, and engine,
so no focused state or task number crosses a query. The already-counted request
digest check binds that fresh store to the audited seed digest; it does not
rescan program slots.

The adapter does not infer private antecedent execution. `NG-H-ConsiderPrune`
has one total task predicate which is true exactly for the public slot/profile
and one action program which must seal a disposition. A false predicate, abort,
or VM error leaves no sealed disposition and is therefore `bridge-invalid`.
Every successful phase materializes its individual fact/action units; the
adapter audits those units and the final disposition through public store state.
No log parsing or private-engine observation is part of correctness or work.

Current unchanged engine dispatch is charged as a fixed 22-event category-12
surcharge per request: one `TaskNum` increment; nine events for the empty
`Heuristic` category dispatch (two `Store.Count` reads, two `Agenda.Len` reads,
and five `trackApplics` reads/writes); and 12 for `NG-H-ConsiderPrune` (the same
nine plus the three `trackThenPartRecord` heuristic/record reads/write). A
source-structure test pins these operations; any engine change that alters them
requires a new experiment version. The post-store audit checks both application
entries and the lane action counter, so the surcharge cannot be claimed without
the bookkeeping effects.

Mechanical integrity reads are outside the semantic-work ledger for every
policy: `VM.InitError`, the pre/post `VM.DeletedUnits` checks, `Agenda.Len` guard
checks performed by the adapter rather than by engine dispatch, residual-agenda
and output-path checks, and post-termination oracle checks. They are counted in
a separate integrity vector and any failure invalidates the panel, but they do
not construct, inspect, or apply a candidate/domain artifact. Mutex operations,
VM environment assignment, type-dispatch traversal, and logging are likewise
control mechanics. In contrast, the request write, agenda enqueue/dequeue,
request-digest check, `TaskNum`, and all persistent engine bookkeeping are
semantic-work events as frozen above.

CUE heuristics must end the request with exactly one sealed
`NogoodDisposition`:

- `resume` records that no complete applicable proof was found;
- `propose-prune` names one frozen artifact, normalized binding, one completion
  record, one certificate, and one prune proposal; or
- `bridge-invalid` identifies an internal duplicate, stale, corrupt, ambiguous,
  over-cap, or incomplete bridge state.

Target matching itself is frozen. The current literal supplies `a` and
`blocked`. The action program's first phase scans the immutable anchor domain in color order and
materializes its sole other color as `escape`, or resumes if the domain is not
size two. It then visits every other variable exactly once in descriptor order,
materializing a role candidate only when that immutable domain has size two,
contains `blocked`, and has one derived `only` color. Its second phase proposes
every unordered pair of retained candidates in descriptor order, checks equal
`only`, explicitly checks `only != escape`, normalizes the lower variable as
`x`, and creates one binding. Its third phase
visits every frozen artifact in semantic-key order and checks its three bits and
target edges. Zero applicable bindings yields `resume`; exactly one may build
the completion/certificate; more than one complete applicable binding is
`bridge-invalid`. No index, driver prefilter, early no-match exit, or Go role
enumerator exists. The maximum is seven role candidates and 21 pair proposals,
and all rejected candidates/pairs, including a rejected `only == escape`, remain
charged. Exhaustive matcher/oracle tests include all two-color anchor/pair
domain triples and require agreement with all five guard conditions.

Every disposition includes the request digest and authoritative digest of every
referenced unit. The Go adapter may check one already materialized disposition
for structural completeness and digest agreement. The CUE complete-evidence
barrier owns its 18 predicate checks and seals one aggregate barrier record
whose digest commits the ordered predicate/result keys. The adapter does not
repeat those predicates or validate every referenced unit separately. It makes
exactly six category-10 checks: disposition envelope, aggregate barrier seal,
authoritative artifact, completion record, request/target/decision digest tuple,
and referenced-unit-set digest. It may apply a prune only when all six pass; it
may not search for a substitute or repair a record. The independent transcript
audit later reconstructs all 18 barrier predicates from the retained units. A
stale request number, different
assignment/domain digest, multiple dispositions, residual agenda task, or
`bridge-invalid` makes the policy and panel mechanically invalid.

`resume` continues with the exact AC-3 initialization that would have followed
the pause. `propose-prune` restores the pre-proposal domain snapshot, records
the omitted fixed branch, and returns `no-solution`; no AC-3 event for that
decision occurs. `match-only` validates the same
proposal but deliberately executes the `resume` path. The bridge controls
`mac-cbj-empty`, no-artifact, reset, learned, match-only, corrupted, wrong-family,
and random cross this bridge only on the supplied branch decision. An
explicit empty artifact store therefore pays the same request, agenda,
disposition, and adapter path as the learned store. Standalone `mac-cbj`, which
does not cross the bridge, is the only primary comparator.

Certificates are occurrence-specific: even the same binding/literal under a
different current assignment or reduced-domain digest must be rematerialized.
No cross-prefix, cross-request, or cross-policy certificate cache exists. This
is conservative and makes target certification cost part of every causal use.

Wrong-family, corrupted-mask, random-mask, reset, and no-artifact controls
receive byte-identical public cases and legal operations. Corrupted mask 5
removes edge `a--y`; wrong-family uses a domain-separated valid mask-7 artifact
whose pair domains have three colors instead of two; random chooses one
mask uniformly from `0..6` once at the training freeze and reuses that one
frozen artifact for the entire panel. Only an artifact whose mask, promotion
substitution set, evidence-boundary digest, and provenance digest equal the
accepted training freeze is authoritative. A nonauthoritative artifact may be
matched and diagnosed but can only seal `resume`; the adapter rejects any prune
proposal naming it. None may silently fall back to the correct artifact.

Expected control dispositions are frozen:

| Policy | reusable | near-miss | irrelevant | independent-unsat |
| --- | --- | --- | --- | --- |
| `nous-generalized` | `propose-prune` | `resume` | `resume` | `resume` |
| `match-only` | valid proposal, then adapter `resume` | `resume` | `resume` | `resume` |
| `corrupted`, `random` | diagnostic match allowed, always `resume` | diagnostic match allowed, always `resume` | `resume` | `resume` |
| `wrong-family`, `reset`, `no-artifact`, `mac-cbj-empty` | `resume` | `resume` | `resume` | `resume` |

The exhaustive control test crosses every policy with every cohort, all three
near-miss masks, and every descriptor/color permutation. A corrupted or random
artifact is never authorization evidence, even when its weaker mask happens to
match and its sampled completion happens to conflict.

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
indices for one replicate in the fixed 24-stratum order and then increasing sampled
position. Power `panel` draws locked-size paired indices in that same order;
its `bootstrap` and `randomization` streams then draw all inner replicates in
increasing replicate order from their independent PCGs. Ties never consume
randomness.

The canonical random-control JSON is
`["part3/nogoods/v1","training",831001,0,"random-control"]`; its SHA-256 is
`287e2fe3f24afab3f17a07eef3a485774c68135bdcc19933e358437c00777e5f`,
its PCG seeds are `2917822264651741875,17400228833170589047`, its first raw
`Uint64` is `10321276913399045564`, and a fresh PCG's first `Uint64N(7)` is
mask 3. The separately fixed corrupted mask is 5, so the two controls are
distinct.

Unit tests freeze the canonical JSON bytes, SHA-256 digest, two PCG seeds, first
four `Uint64` outputs, and resulting permutation for training seed 831001,
development seed 832001, validation seed 833001, and a fixed all-zero 64-hex
locked test root. A mismatch is mechanical invalidity.

The `variable-positions` golden vectors are:

| Panel/root | SHA-256 | PCG seeds | First four `Uint64` | Permutation |
| --- | --- | --- | --- | --- |
| `training` / `831001` | `ac5eb4be74b192caba90677858e7ca28c346c1329bfe16347b864600b1669f0d` | `12420563552428987082,13443358654286252584` | `16196928853732314818,13964589984287698613,14828555489593947842,3864106448264698449` | `0,1,2` |
| `development` / `832001` | `7ee119fb7276549805e9faaa614a7049671b62ab1466d6a66417e274b279e2a6` | `9142617286286660760,426147249446875209` | `1213905441346112375,1699159744660973649,1395889987714557996,13902222507069901990` | `5,2,1,4,3,6,0,7` |
| `validation` / `833001` | `c6ae755044795c25e589d32a073dc98744347a48470364294121d4445b8aca9e` | `14316509253064011885,16539983283958434183` | `4803874459439516066,14275222390261356384,10176726368503672072,10929098418971052156` | `6,0,1,4,7,3,5,2` |
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
all exact masks and selects the canonical first only for diagnostics; V2's
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
`a--d4`. Its distractor graph is a fixed complete tripartite `K(2,2,1)` selected
with the domain template by ordinal modulo four:

```text
0: d0--d1,d0--d2,d0--d3,d0--d4,d1--d3,d1--d4,d2--d3,d2--d4
1: d0--d1,d0--d2,d0--d3,d1--d2,d1--d3,d1--d4,d2--d4,d3--d4
2: d0--d1,d0--d2,d0--d3,d0--d4,d1--d3,d1--d4,d2--d3,d2--d4
3: d0--d1,d0--d2,d0--d4,d1--d2,d1--d3,d2--d3,d2--d4,d3--d4
```

Each graph is three-colorable and every distractor has public static degree at
least four after its fan edge. The partitions maximize the frozen requeue lower
bound subject to the domain template; they are source constants rather than a
generator search. Committed witnesses after `a=blocked` are respectively
`(escape,only,only,spare,spare)`,
`(spare,escape,only,only,spare)`,
`(escape,only,only,spare,spare)`, and
`(spare,escape,only,spare,escape)`. No distractor edge touches `x` or `y`.

A `reusable` case adds all three schema
edges. A `near-miss` case removes bit `(cohortOrdinal mod 3)` from those three.
An `irrelevant` case gives every variable all four colors and uses its fan and
distractor edges plus `a--x,x--y`. An `independent-unsat` case gives every
variable all four colors except `d0` and `d1`, whose domains are the singleton
`{blocked}`, and adds `d0--d1` plus the selected distractor template; duplicate
edges are canonicalized before serialization. These rules cannot cross the
18-edge cap. Every utility query has the explicit empty base partial assignment
and supplies decision `a=blocked`. An independent
exhaustive check requires the fixed branch to be empty for reusable and
independent-unsat cases and satisfiable for near-miss and irrelevant cases.

Development's 96 tasks contain:

- 56 `reusable` cases with one full motif and no completion under the supplied
  blocked anchor decision;
- 24 `near-miss` cases with one required edge absent and a branch completion;
- 8 `irrelevant` satisfiable branches with no guard-respecting binding; and
- 8 `independent-unsat` branches whose contradiction does not match the schema.

Validation doubles each count. Locked uses 384 tasks with exactly 312, 48, 12,
and 12 cases respectively. Within every panel, contiguous ordinals map to
cohorts in the order listed; `cohortOrdinal` starts at zero within each cohort.
The task seed is the matching manifest arithmetic-sequence member. Locked uses
its root and global ordinal. There is no semantic rejection loop. Generation
then verifies the stated cohort, stated normalized-binding cardinality, bounds,
and oracle solution status; disagreement invalidates the panel instead of
regenerating it.

Inference and power use 24 frozen semantic strata, ordered as: four reusable
template IDs; 12 near-miss `(template ID, missing-bit)` cells in template-major,
then bit-major order; four irrelevant template IDs; and four independent-unsat
template IDs. Development has respectively 14, 2, 2, and 2 tasks per such cell;
validation has 28, 4, 4, and 4; locked has 78, 4, 3, and 3. Thus every locked
cell is represented in development, including every anchor-edge and pair-edge
near miss. Within a cell task order is global task ordinal. Broad cohort totals
remain reporting groups only; bootstrap and power never pool distinct cells.

`blocked` has a lower color descriptor position than `escape`, but the supplied
decision—not search order—selects the branch and neither display name reveals
its role. Descriptor positions and distractor
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
- `mac-cbj` is the primary conventional comparator. It starts directly from the
  supplied bound decision, maintains AC-3 after every decision, records exact
  instance conflicts, and performs conflict-directed backjumping without any
  Nous bridge, artifact, agenda, or adapter work.
- `mac-cbj-empty` executes the top-level bridge with an empty artifact store,
  then resumes the byte-identical standalone algorithm. It is an overhead
  control and cannot replace standalone `mac-cbj` in inference.
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

Standalone `mac-cbj` and every resumed learned/control policy execute this
identical procedure. Current domains begin as immutable public domains and the
base partial assignment is empty. The supplied decision is a distinguished
one-value root activation for `a`: snapshot the empty prefix, save the literal
`a=blocked`, validate/propose/bind it, remove `escape` with explanation `{a}`,
and initialize AC-3 exactly as step 3 below. Bridge policies pause once between
that bind and the initial AC-3 queue; standalone `mac-cbj` does not. A sound
top-level prune records the saved literal, restores the root snapshot, and ends
the fixed-branch query. A `resume` enters the same initial AC-3 and recursive
state byte-for-byte.

Failure in the root activation has one wrapper rule independent of where it
arises. Immediate public-domain or assigned-edge failure, initial AC-3 wipeout,
and failure returned by recursion are normalized to a set containing only
variables assigned in the root activation's caller prefix plus `a`. While the
root assignment is still present, seal the exact saved-literal nogood; project
the failure through `failure - {a}`; restore domains, explanations, and the empty
base prefix; and require the projected set to be empty. Empty means
`no-solution`; nonempty is an internal baseline error because the base prefix is
empty. A resumed bridge uses this same wrapper. Tests cover all three failure
origins and a learned `resume` separately.

Variable choice minimizes current domain cardinality, then maximizes static
degree in the original public graph, then minimizes variable descriptor
position. Values are tried by increasing color descriptor position.

Before selecting a variable, scan task-local exact nogoods by increasing
semantic key. A record matches only when every concrete literal is present in
the current assignment. A match returns the record's variable set as failure.
If all variables are assigned, independently check every domain and edge in
canonical order and return the witness.

Each recursive activation selects `v`, creates one fresh empty local
`conflicts(v)`, and then performs, for each value:

1. snapshot current domains, assignment, and deletion explanations;
2. propose and bind `v=value`; remove each other current value of `v` in color
   order with explanation `{v}`; reject immediately on a public-domain or
   already-assigned-edge violation;
3. initialize a FIFO AC-3 queue with `(neighbor,v)` for every neighbor ordered by
   decreasing static degree in the original public graph, then increasing
   descriptor position, suppressing duplicate queued pairs;
4. pop FIFO and revise `Xi` against `Xj`. For each current `Xi` value in color
   order, scan current `Xj` values in color order until inequality support is
   found. If none exists, delete it. Its explanation is the sorted union of the
   deletion explanations of every original `Xj` value unequal to it. If `Xi`
   changed but is nonempty, append `(Xk,Xi)` for every neighbor `Xk != Xj` in
   decreasing static degree and then descriptor order unless already queued;
5. a domain wipeout returns the sorted union of explanations for every original
   value of the wiped variable. Otherwise recurse;
6. while the failed branch assignment is still present, materialize any exact
   nogood from the saved literal of each variable in the returned failure set;
   only then restore the snapshot;
7. if the returned set omits `v`, emit one backjump and return it immediately;
   otherwise union `failure - {v}` into `conflicts(v)` and try the next value;
   and
8. after all values fail, materialize the exhausted-variable consequence from
   the surviving caller-prefix literals named by `conflicts(v)`, then return
   `conflicts(v)`, never `v`. The root wrapper above performs the only terminal
   failure projection. The first witness stops the task.

An absent deletion explanation is the empty set. Union iterates source sets and
members in descriptor order, charges each attempted member, and deduplicates by
variable descriptor. Direct assigned-edge failure contains the two endpoint
decision variables; public-domain failure contains `{v}`. Queue membership is a
Boolean matrix restored with the search snapshot. `conflicts(v)` is activation-
local, is neither snapshotted nor restored, and is discarded on return. A
returned failure set contains only variables assigned in the caller's surviving
prefix; the saved failed-branch literal map exists only until its nogood record
is sealed. No learned schema changes
variable/value order, explanation construction, exact-nogood retention, or
post-failure resumption.

Committed golden microtraces are required before any panel. They fix the
complete request sequence, decisions, FIFO contents after every transition,
revisions, deletion explanations, concrete nogoods, conflict unions,
backjumps, terminal/witness, and 12-category ledger vector for: a satisfiable
edge, an unsatisfiable equal-singleton edge, the blocked pair, a
nonchronological backjump, a task-local exact-nogood hit, and one bridge resume.
One further trace revisits the same selected variable in two sibling branches
and proves that no activation-local conflict leaks. The standalone continuation
and learned adapter after `resume` must be bisimilar on every golden trace.

### Learned and ablated policies

- `nous-generalized` runs the frozen top-level bridge and, on `resume`, the same
  standalone `mac-cbj` continuation; on a sound proposal it returns
  `no-solution` for the supplied branch without entering AC-3, after full target
  certificate construction and legal pruning.
- `no-artifact` runs the identical adapter path with an explicit empty artifact
  store and must be byte-equal to `mac-cbj-empty`, including adapter records;
  it is a second execution identity used to detect policy branching.
- `reset` loads a fresh seed-worth/profile store that contains no promoted
  schema and must have the same disposition/search projection as no-artifact;
  its distinct store reads remain charged.
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
| successful unchanged-engine task dispatch | the exact 22 category-12 dispatch surcharge specified at the bridge boundary |
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
| adapter validates proposal | exactly six category 10 checks of the sealed aggregate fields specified above; predicate-level checks belong to the CUE barrier and are not repeated |
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

### Transcript persistence and cap scopes

Complete transcripts are evidence-bundle artifacts, not embedded in the JSON
report. For panel `<panel>` the canonical directory is
`.nous/nogoods-v2-<panel>-transcripts/`; it contains `primary/` and `audit/`,
each with one deterministic `<policy>.ngt.gz` for all 13 required policies and
an `execution-manifest.json`, plus one root `fixtures.json` and one root
`manifest.json`. Development,
validation, and locked all execute two fresh stores against the same immutable
fixture bytes and retain all 26 chunks. Validation and locked guards
exclusively create the root directory alongside their receipt/report and refuse
any existing path or symlink. Development requires an absent output directory
at command start. No panel may discard, replace, or alias a chunk from the other
execution. Only the primary execution contributes empirical work; audit work and
its independent 54-event profile preflight are reported as integrity evidence.

`fixtures.json` is the canonical compact JSON array, in task-ordinal order, of
records containing `ordinal`, the exact canonical problem byte slice encoded as
unpadded base64url, and the supplied decision's numeric `variable` and `color`.
It contains no seed, cohort, template, private root, oracle result, or outcome.
Its decoded problem bytes must themselves pass canonical problem decoding.
Both executions consume the same decoded records from this file; neither may
regenerate a task. The root manifest records its byte length and SHA-256, and
the receipt finalization hashes it transitively through the root manifest. Its
uncompressed size is capped by `fixture_bundle_byte_cap=16777216`; crossing the
cap before either execution makes the attempt invalid. This retained public
input is sufficient for an independent reviewer to rerun the baseline, oracle,
and complete omitted-solution audit without the private locked root.

Acquisition has no separate chunk. In both execution roles, all profile,
training, selection, promotion, and freeze events belong to
`nous-generalized.ngt.gz` before its utility events. Event sequence is one
zero-based, gap-free counter per policy chunk across all phases. The unsigned
32-bit task ordinal encodes phase without collision:

| Task ordinal | Meaning |
| --- | --- |
| `0xffffffff` | bridge-profile preflight |
| `0x80000000..0x80000003` | training examples 0..3 |
| `0x80000004` | cross-example refinement/evidence/selection |
| `0x90000000..0x90000017` | promotion substitutions 0..23 |
| `0x90000018` | artifact freeze/serialization |
| `0..task_count-1` | utility task ordinal |

No other high-bit ordinal is legal. `A` is exactly the sum of categories for
all high-bit events in the primary `nous-generalized` chunk; the audit copy is
reconstructed but excluded from inference. `recomputed` target-local
acquisition uses that utility task's ordinary ordinal and an operation ID that
names the recomputation phase. All other policy chunks begin at sequence zero
with utility work. Execution manifests additionally report the high-bit event
count/vector and reconstructed `A`, which the independent reducer checks against
the chunk.

Uncompressed `ngt/v1` begins with a 4 KiB header, followed by a canonical UTF-8
dictionary capped at 1 MiB per policy. Header bytes `0:4` are ASCII
`NGT1`; `4:6` are unsigned big-endian header version 1; `6:8` are unsigned
big-endian record width 96; `8:16` are the unsigned big-endian dictionary byte
length; `16:24` are the unsigned big-endian event count; `24:56` are the
dictionary SHA-256; and `56:4096` are zero.

The dictionary grammar is exact. It starts with an unsigned big-endian 32-bit
entry count. Each entry is an unsigned big-endian 16-bit byte length followed by
that many UTF-8 bytes, with no terminator or padding. Entries are nonempty,
unique, valid shortest-form UTF-8, contain no NUL, and are at most 128 bytes.
They are sorted by `(SHA-256(bytes), bytes)` ascending. Header dictionary length
counts the four-byte count, every two-byte length prefix, and every payload byte;
that entire number must be at most 1,048,576. No trailing dictionary bytes are
legal.

Every event record is exactly 96 bytes, with all integers big-endian:

| Offset | Width | Field |
| ---: | ---: | --- |
| 0 | 1 | record format, fixed `1` |
| 1 | 1 | category `1..12` |
| 2 | 1 | transition code from the closed enum below |
| 3 | 1 | flags, fixed zero in `ngt/v1` |
| 4 | 4 | unsigned task ordinal |
| 8 | 8 | unsigned event sequence, starting at zero |
| 16 | 32 | eight signed 32-bit operands `o0..o7` |
| 48 | 32 | `SHA-256("ngt-tuple/v1\0" || bytes[0:48])` |
| 80 | 16 | zero |

Dictionary ID zero means absent; positive IDs are one-based. In the table, `id`
means a required positive dictionary ID, `id?` permits zero or a positive ID,
and `n` means a signed numeric literal. All omitted operands must be zero. This
is the complete transition enum; there are no extension or
implementation-defined codes:

| Code | Name | Category | `o0..o7` schema |
| ---: | --- | ---: | --- |
| 1 | candidate-refinement | 1 | candidate id, parent id?, mask n, added-bit n, operation id, outcome id |
| 2 | fact-read | 2 | subject id, predicate-or-slot id, object id, outcome id |
| 3 | assignment | 3 | variable id, color id, depth n, operation id, outcome id |
| 4 | inequality | 4 | left-variable id, left-color id, right-variable id, right-color id, outcome id |
| 5 | domain | 5 | variable id, color id, operation id, explanation id?, outcome id |
| 6 | ac3 | 6 | variable id, peer-variable id, color id?, queue-position n, operation id, outcome id |
| 7 | conflict-nogood | 7 | conflict id, variable id?, literal id?, target-depth n, operation id, outcome id |
| 8 | binding-schema | 8 | artifact id?, binding id?, atom-or-bit n, subject id?, object id?, operation id, outcome id |
| 9 | completion | 9 | binding id, completion id, constraint id?, left-value id?, right-value id?, operation id, outcome id |
| 10 | certificate-barrier | 10 | certificate id, record id?, predicate id, expected id?, actual id?, operation id, outcome id |
| 11 | witness-terminal | 11 | terminal-or-witness id, outcome n |
| 12 | semantic-store | 12 | key id, unit id?, slot id?, operation id, outcome id |
| 13 | agenda | 12 | task id, unit id, slot id, priority n, operation id, outcome id |
| 14 | bridge-request | 12 | request id, target id, decision id, digest id, operation id, outcome id |
| 15 | bridge-disposition | 12 | request id, disposition id, artifact id?, barrier id?, operation id, outcome id |
| 16 | control-bookkeeping | 12 | heuristic id, slot id?, counter n, operation id, outcome id |
| 17 | cache | 12 | key id, value id?, operation id, outcome id |
| 18 | record-write | 12 | record id, subject id?, sequence n, operation id, outcome id |

The header dictionary digest and whole-file raw digest bind ID meanings; the
per-record digest binds its exact fixed-width tuple. The independent reducer
rejects a code/category mismatch, a zero required ID, a nonzero omitted operand,
an out-of-range ID, bad order/sequence, or a digest mismatch. Gzip uses DEFLATE
level 9, empty name/comment, zero modification time, OS byte 255, and exactly one
member.

The complete acquisition-bearing golden stream contains one representative
profile-preflight event at task ordinal `0xffffffff`; its counter operand is 54,
the last operation in the full preflight. Its sorted dictionary maps IDs
`1=ok`, `2=profile-preflight`, `3=NG-H-ConsiderPrune` and has hex
`0000000300026f6b001170726f66696c652d707265666c6967687400124e472d482d436f6e73696465725072756e65`,
SHA-256 `0acd1aa361e1264b0d44a4d4bd575b8acae7c3dc3ab1d5ca6aa478e1248f36f1`,
length 47, and event count 1. Its header is the fields above followed by zero
through byte 4095. Its record hex is:

```text
010c1000ffffffff00000000000000000000000300000000000000360000000200000001000000000000000000000000e204c485551c6120b3757589a8ea9c9920114284c61ca8067613c0a271a0094300000000000000000000000000000000
```

The raw stream is 4,239 bytes with SHA-256
`4ce62bdfec29e813e3b7aaffef74acd2416722d82f114a9a51efb5291c56d7bd`.
Its complete 194-byte gzip member is this base64, with SHA-256
`2d44c34dcbd9cb17136ebbc71478b70d0766d0251bcb27268040c912c91f85b9`:

```text
H4sIAAAAAAAC//NzDzFkYGRIYIAAfSjNyHVWanHiQzVvXpclV/aGR3eden74jtXGq6eyllQ8VOk3+8gwCkbBKBgFo2AUjIJRMApGwSgYBaNgFIyCoQSYGZjysxkEC4ry0zJzUnULilLTcjLTM0oYhPzcdT10nfPzijNTUosCikrzUhl5BBj+AwGKdggwA2Im0NABstmPWI60hsokKmwuLe1c8WrOTAVBp5ZjMivYyoQPLCpcwOmM7hYAEloDNo8QAAA=
```

The golden single-policy conformance manifest is the following canonical JSON
(production manifests require all 13 entries), with SHA-256
`b185448f2aa4df97eb5dafe231a7d704c7dd74c137c62e426ecec80296e61d0a`:

```json
{"acquisition_event_count":1,"acquisition_vector":[0,0,0,0,0,0,0,0,0,0,0,1],"acquisition_work":1,"execution_role":"primary","policies":[{"event_count":1,"first_sequence":0,"gzip_sha256":"2d44c34dcbd9cb17136ebbc71478b70d0766d0251bcb27268040c912c91f85b9","gzip_size":194,"last_sequence":0,"policy":"nous-generalized","raw_sha256":"4ce62bdfec29e813e3b7aaffef74acd2416722d82f114a9a51efb5291c56d7bd","raw_size":4239}]}
```

Each execution manifest records execution role, policy, raw/compressed sizes,
event count, raw SHA-256, gzip SHA-256, and first/last sequence for its 13
chunks. The root manifest records both execution-manifest digests and the
fixture-bundle size/digest and report-payload digest. The report payload excludes evidence digests and is
hashed first. The report then retains that payload, its digest, and only the
root-manifest digest—not manifest contents. The root manifest never contains a
hash of the final report, so the graph is acyclic:

```text
chunks -> execution manifests --+
fixtures.json ------------------> root manifest <- report payload
                                      |
                                      v
                              final report reference
```

Receipt finalization hashes the final report, root manifest, fixture bundle,
both execution manifests, and all chunks. The primary and audit canonical semantic report
payloads exclude execution role and evidence paths and must be byte-equal;
their transcript hashes are compared positionally before the wrapper report is
formed.

Scopes and abort boundaries are exact:

- `training_work_cap=2000` covers the one complete acquisition transcript,
  including the execution's 54-event bridge-profile preflight;
- `target_prune_work_cap=128` covers one learned request from supplied-decision
  validation through a prune terminal;
- `no_match_bridge_overhead_cap=83` covers exactly one bridge request that ends
  in `resume` before constructing any completion: from that request's write
  through its disposition/adapter records, after cancelling the byte-identical
  standalone continuation. It applies to every bridge-backed policy when that
  zero-completion condition holds. A reset profile preflight, recomputed
  target-local acquisition, and `match-only`'s validated proposal/completion/
  certificate segment are outside this scalar scope; they remain charged under
  their component ledgers and `policy_work_cap_per_task`. Transcript operation
  IDs mechanically delimit the excluded phases. A bridge-backed policy may not
  evade the cap by changing its policy name;
- `policy_work_cap_per_task=2000000` covers one policy on one utility task and
  aborts before the exceeding event;
- `bridge_task_pop_cap=2000` covers one request and aborts before pop 2,001;
- `attributed_unit_cap=200000` covers one fresh training or utility store;
- `report_byte_cap=16777216` covers each uncompressed canonical JSON report;
- `fixture_bundle_byte_cap=16777216` covers the one uncompressed canonical
  `fixtures.json` shared by both executions of a panel;
- transcript per-execution caps cover the sum across its 13 policies and bundle
  caps cover both protected executions, including dictionary/header bytes. They
  are checked before an event, before raw close, after deterministic compression,
  and again before root-manifest finalization respectively.

At eight million events, 96-byte records plus thirteen maximum dictionaries and
headers are below 0.75 GiB and therefore below the 1 GiB raw cap; the exact
integer preflight is a test vector. Across 13 chunks, the DEFLATE stored-block
worst case adds at most five bytes per 32,768 raw bytes, one partial-block
allowance and 18-byte header/trailer per chunk. A 1 GiB aggregate is therefore
below 1,073,906,000 bytes and below the 1,074,000,000-byte gzip cap even without
compression. Two executions are therefore below the exact doubled bundle caps.
The report is separately bounded at 16 MiB and retains aggregate vectors and the
root-manifest digest; individual events remain in the 13 or 26 hashed chunks.
No transcript can be discarded after a positive, null, invalid, or crashed
protected attempt.

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
Training similarly becomes invalid if its separately frozen 2,000-unit cap is
crossed. The attributed-unit and report-byte caps are separate mechanical
limits; crossing either makes the panel invalid.

## Evidence-free attainability bound

V2 requires a strictly negative lifecycle difference but sets no arbitrary
minimum magnitude. That endpoint is attainable without consulting a
development, validation, or locked seed. A literal expansion of the frozen
full fixed-branch query—not a cancelled branch fragment—establishes:

| Complete path | Frozen bound over every allowed template and descriptor/color permutation |
| --- | ---: |
| standalone MAC-CBJ on a reusable branch | at least 150 events |
| generalized matching/certificate/prune on a reusable branch | at most 125 events; hard cap 128 |
| no-match bridge overhead before the identical standalone continuation | at most 81 events; hard cap 83 |
| one complete acquisition | at most 2,000 events |

Every distractor has static degree strictly above the degree-two `x,y`, so the
frozen initial queue processes all five distractor anchor arcs before either
pair anchor arc under every descriptor permutation. All distractor requeues
therefore precede pair requeues. The four template-specific literal lower-bound
vectors, including root wrapper, first pair wipeout, conflict projection,
nogood, and terminal, are:

| Template | initial comparisons | changed neighbors | distractor requeues | requeue comparisons | c1 | c2 | c3 | c4 | c5 | c6 | c7 | c8 | c9 | c10 | c11 | c12 | total |
| ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| 0 | 24 | 7 | 16 | 48 | 0 | 1 | 3 | 73 | 19 | 73 | 4 | 0 | 0 | 0 | 1 | 3 | 177 |
| 1 | 21 | 6 | 13 | 35 | 0 | 1 | 3 | 57 | 17 | 64 | 4 | 0 | 0 | 0 | 1 | 3 | **150** |
| 2 | 22 | 6 | 13 | 37 | 0 | 1 | 3 | 60 | 17 | 64 | 4 | 0 | 0 | 0 | 1 | 3 | 153 |
| 3 | 21 | 6 | 13 | 35 | 0 | 1 | 3 | 57 | 17 | 64 | 4 | 0 | 0 | 0 | 1 | 3 | **150** |

For every template, category 4 is initial comparisons plus requeue comparisons
plus the final pair comparison. Category 5 is three root delete/check/restore
events, two per changed neighbor, and two for final pair deletion/wipeout.
Category 6 is 21 initial enqueue/dequeue/revision events, one enqueue per
distractor and pair requeue, two dequeue/revision events per distractor requeue,
and two for the final pair requeue. Requeue comparison lower bounds sum the
post-anchor target-domain cardinality for each directed arc; a scan visits at
least one peer value for every target value. The four conflict events are the
final deletion explanation member, two wipeout union attempts, and root removal
of `a`. The three final store events are exact-nogood lookup, exact-nogood write,
and terminal-record write. Color permutations can add comparisons but cannot
reduce these domain-cardinality sums.

The reusable learned path is fixture-specific: exactly `x,y`, not seven
variables, pass the size-two/contains-`blocked` role guard, so exactly one pair
is proposed. Its exact maximum vector is:

| Path component | c1 | c2 | c3 | c4 | c5 | c6 | c7 | c8 | c9 | c10 | c11 | c12 | total |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| root pre-bridge | 0 | 1 | 2 | 0 | 2 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 5 |
| request, agenda, and both engine dispatches | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 26 | 26 |
| anchor, seven visits, two candidates, one pair, one artifact | 3 | 20 | 0 | 0 | 0 | 0 | 0 | 10 | 0 | 0 | 0 | 12 | 45 |
| one completion and complete barrier | 0 | 2 | 0 | 3 | 0 | 0 | 0 | 0 | 1 | 19 | 0 | 7 | 32 |
| disposition, adapter, omitted-prefix record, root restore, terminal | 0 | 0 | 1 | 0 | 1 | 0 | 0 | 0 | 0 | 6 | 1 | 8 | 17 |
| **maximum** | **3** | **23** | **3** | **3** | **3** | **0** | **0** | **10** | **1** | **25** | **1** | **53** | **125** |

The guard row includes the explicit `only != escape` read/check. The matcher
row's 12 category-12 events are seven visit records, two candidate writes, one
binding write, and artifact match read/result. The completion row's seven are
the completion-result write, agreement result-read, agreement-record write,
expected-key-set write, actual-key-set write, sealed-barrier write, and
certificate-index write; its 19 category-10 events are 18 barrier
predicates plus certificate-record creation. The final eight category-12 events
are disposition write plus five digest checks, omitted-prefix/prune record, and
terminal-record write.

The request row is four request/agenda/digest events plus the exact 22-event
unchanged-engine dispatch surcharge. The once-per-execution 54-event profile preflight is
inside acquisition `A`, not this request row. Any extra task makes the
conformance test fail before a panel. The three-event difference between 125 and
the hard prune cap is abort headroom, not
uncharged work and not part of this feasibility proof.

For a zero-completion `resume` target the shared root and resumed MAC-CBJ events
are byte-identical and cancel. The maximum capped bridge-only extra vector is the 26-event
request/engine row plus the 45-event matcher row plus ten disposition/adapter
events, exactly 81. The matcher subtotal includes all three required target-edge
reads; revision 7 incorrectly parked them in the completion row, which preserved
the 125-event prune total but omitted them from the no-match path. Near misses
have the same two candidates and one pair; irrelevant and independent-unsat
cases retain fewer candidates. Nonauthoritative controls stop after provenance
diagnostics and never construct a completion, so they do not exceed this bound.
The 83-event hard cap preserves two events of abort headroom. The complete
acquisition has its separately enforced 2,000-event cap.

These are deliberately conservative integer bounds, not mean estimates. The
plan-conformance test encodes each transition-table multiplicity as a literal
constant and proves the inequalities without importing or invoking production,
fixtures, baseline, engine, or experiment code. Architecture and experimental
review accept that literal derivation before implementation; a disagreement
requires another plan revision, not development observation.

At the locked 312/48/12/12 proportions, every reusable template occurs 78
times. Using the corrected hard caps and the already frozen per-template
standalone lower bounds, the worst attainable total difference is negative:

```text
A + sum(L-M) <= 2000
                  + 78*((128-177)+(128-150)+(128-153)+(128-150))
                  + 72*83
                = -1228
```

The standalone denominator is strictly positive, so the corresponding primary
ratio is below zero. This proves only that V2 can answer its hypothesis; it does
not forecast the fixed panel or supply evidence for a positive result. A panel
crossing the acquisition/prune/no-match cap is mechanically invalid rather than
used to revise the bound.

## Statistical endpoint and classification

Mechanical gates are evaluated first. They require accepted-plan source and
dependency hashes, fresh-store schema reconstruction, competence success,
deterministic byte-equal rerun, exact transcript conservation, no forbidden
input, correct satisfiability terminals for every required policy/task, exact
full-solution-set preservation by every prune, and independent oracle
agreement.

The required utility matrix on development, validation, and locked is exactly
`chronological`, `forward-checking`, standalone `mac-cbj`, `mac-cbj-empty`,
`no-artifact`, `reset`, `concrete-memo`, `nous-generalized`, `recomputed`, `corrupted`,
`wrong-family`, `random`, and `match-only` on every task. Every one is subject
to witness/no-solution parity, work-cap, deterministic-transcript, and terminal
gates. Every policy that proposes a prune is additionally subject to the full
omitted-solution audit. Only `nous-generalized` versus standalone `mac-cbj` enters
confirmatory inference; the other eleven are frozen diagnostics and causal
controls, never a post-hoc replacement comparator.

Let `A` be the one observed training-acquisition total, `L_i` learned target
work, `M_i` standalone `mac-cbj` target work, `d_i=L_i-M_i`, and `N` the fixed task
count. The one primary point estimate is:

```text
R = (A + sum_i d_i) / sum_i M_i
```

For bootstrap replicate `r`, sample paired indices with replacement separately
inside each of the 24 semantic strata, preserving all stratum counts. In the
frozen stratum order above and sampled-position order, draw
`Uint64N(stratumCount)` from that replicate's fresh PCG. Recompute both numerator
and denominator:

```text
R_r = (A + sum_j d_[I_rj]) / sum_j M_[I_rj]
```

`A` occurs once per replicate and is never resampled or duplicated as `A/N`.
Sort 10,000 `R_r` values as exact rational cross-products, breaking exact ties
by replicate index only for report order; interval endpoints are values at
zero-based indices 249 and 9749.

For randomization, acquisition is deterministically amortized over the same
task sampling unit without division. Define `e_i = N*d_i + A`; then
`sum(e_i) = N*(A + sum(d_i))`, so its sign exactly matches total lifecycle
advantage. Under the sharp paired null, each task's two complete lifecycle
outcomes are exchangeable. Replicate `r` draws exactly one `Uint64N(2)` per task
from a fresh PCG in fixed stratum/task order. Zero means sign `+1`; one means
`-1`:

```text
T_observed = sum_i e_i
T_r = sum_i sign_i*e_i
p = (1 + count(|T_r| >= |T_observed|)) / 10001
```

This conditions on the one observed `A` and makes no acquisition-block
exchangeability claim. The inferential population is the fixed panel of paired
deterministic tasks; the bootstrap describes sensitivity to task composition,
while randomization assumes within-task label exchangeability under the sharp
null. All arithmetic is signed 64-bit after a checked overflow preflight. The
development replicate-zero tuple has SHA-256
`4c310ed3e073097cc03302dcdc59eeb3c8eca9b31b23ded25c70e58481abca34`,
PCG seeds `5490185723907869052,13849416426707349171`, and first eight
`Uint64N(2)` draws `0,1,1,0,1,0,0,0`. Exact satisfiability and prune soundness
are gates, not terms that trade against work.

For one panel, its empirical gate passes only if all mechanical gates pass, the
primary point estimate is strictly below zero, the interval upper bound is below zero, the paired
randomization p-value is below 0.05, and target-only overhead on the pooled
near-miss/irrelevant cohort satisfies the exact ratio-of-sums gate
`H = sum(L_i-M_i)/sum(M_i) <= 0.10` against standalone `mac-cbj`. When its
mechanical gates pass but any empirical gate fails, that panel has an empirical
miss. A correctness, soundness, leakage, accounting, provenance, determinism,
boundary, or frozen-protocol failure is mechanical invalidity.

The lane has exactly one terminal classification, fixed before any panel:

| Furthest reached state | Required condition | Lane classification and next action |
| --- | --- | --- |
| development | any development mechanical failure | `invalid`; stop |
| development power | power below 0.80 after mechanically valid development | `valid-null` at development stage; validation and locked remain unopened |
| development power | power at least 0.80 | development's empirical pass/miss is diagnostic; run validation |
| validation | any validation mechanical failure | `invalid`; stop |
| validation | mechanically valid report | validation's empirical pass/miss is diagnostic and never gates continuation; commit it, then authorize locked execution |
| locked | any locked mechanical failure | `invalid`; stop and consume the attempt |
| locked | mechanically valid and its own empirical gate passes | `valid-positive` |
| locked | mechanically valid and its own empirical gate misses | `valid-null` |

No pooled, best-panel, majority, or post-hoc statistic exists. The locked panel
alone supplies the confirmatory point, interval, randomization p-value, and `H`
once the preregistered power path reaches it. Development supplies
effect/variance data only for the frozen power gate; validation is a public
integrity/generalization diagnostic.

Post-freeze target work, certification work, pruned branches, concrete-conflict
count, solution count, artifact precision, generalization distance, storage,
amortization crossover, and all ablation contrasts are secondary diagnostics.
They cannot create a positive label.

## Development, validation, and locked sequence

Development is run only after implementation-candidate review. Its exact report
path is `.nous/nogoods-v2-development-report.json` and its exact transcript
root is `.nous/nogoods-v2-development-transcripts/`. Its command
generates each public fixture byte slice once, writes the bounded
`fixtures.json`, and runs two fresh complete stores/policy sets as `primary`
and `audit` by independently decoding that file. It refuses an
existing report or transcript directory, retains both 13-policy chunks and both
execution manifests, and requires byte-equal semantic payloads and positional
transcript hashes. Only primary work enters empirical estimates; audit work is
reported separately. A mismatch is invalid and both executions remain in the
bundle.

Development supplies the only effect/variance population used for the frozen
power simulation. Exactly 2,000 synthetic locked panels are built by resampling
the 24 development strata to the locked 312/48/12/12 broad-cohort counts and
the exact per-cell counts frozen above. The outer `panel` PCG draws paired
development indices with replacement within each semantic cell in fixed
stratum/sample-position order. No locked semantic cell is estimated from a
different template or missing bit. `A` is copied once into the synthetic panel. Each uses
2,000 inner stratified bootstrap
replicates, 2,000 independently seeded within-task label-swap replicates, and
the same equations, recomputed denominators, point, interval, p-value, and
nonreusable-harm gates. Inner bootstrap endpoints are sorted indices 49 and
1949. Inner randomization uses `e_i = 384*d_i + A`, draws exactly one
`Uint64N(2)` per synthetic task and has Monte Carlo denominator 2,001. Power
streams use canonical JSON
`["part3/nogoods/v1","power",832001,outerOrdinal,purpose]`, where purpose is
exactly `panel`, `bootstrap`, or `randomization`; bootstrap and randomization
derive a fresh PCG from that tuple and never share state. Power is the passing
fraction and must be at least 0.80 before validation or locked execution is
reachable.

Validation may be invoked once after a committed power-positive development
report. Its guard exclusively creates
`.nous/nogoods-v2-validation-receipt.json`, generates and serializes all public
fixture bytes once into the bounded `fixtures.json`, and executes two fresh
stores/policy sets internally by independently decoding those same immutable
bytes. The two canonical reports must be byte-equal; only
the first execution supplies empirical work, while the second is separately
reported integrity audit work. The guard then exclusively writes
`.nous/nogoods-v2-validation-report.json` and finalizes the receipt. A crash or
any existing receipt consumes/refuses the attempt. Validation must pass
competence and all mechanical gates; its empirical result is evidence, not a
tuning gate. After the committed result, all implementation and plan paths are
frozen.

Locked execution follows the repository's one-shot pattern. One guarded API
used by the CLI and package callers requires the exact unlock token
`nogoods/v2:<40-lowercase-hex-clean-HEAD>`, a committed prerequisite manifest, accepted implementation
reviews, accepted development/validation reports, power at least 0.80, no
`go.work` or module replacement, and canonical repository/domain paths. The
manifest hashes every protected input and its committed bytes. The guard
rejects existing receipt/report/transcript paths or symlinks, exclusively creates and
fsyncs a `claimed` receipt, reads a fresh 32-byte root from `crypto/rand`, and
records `started` before deriving a fixture. A crash consumes the attempt.

Inside that one locked invocation, every generated canonical fixture byte slice
is encoded once into the bounded `fixtures.json` after the receipt reaches
`started`; the private root is then erased. Both fresh complete policy runs
decode that retained file independently. Their canonical reports, transcript hashes, terminals, witnesses, and
work vectors must be byte-equal before inference. As in validation, only the
first run contributes empirical work and the second is labelled integrity
audit. This is the deterministic rerun; the private root is never regenerated
and no second locked invocation exists.

The only locked evidence paths are
`.nous/nogoods-v2-locked-receipt.json`,
`.nous/nogoods-v2-locked-report.json`, and the canonical locked transcript
directory, whose root contains the public `fixtures.json`. No other API accepts `locked`, returns a
locked fixture, or exposes the root. An integrity-clean empirical miss is
`valid-null`; mechanical failure is `invalid`; either finalizes the receipt.

If development power is below 0.80, V2 ends as a development-stage valid null,
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
- exhaustive matcher agreement for `only != escape` and every other complete
  guard atom;
- a successful prune before ordinary continuation evaluation;
- standalone-primary proof that no bridge/agenda/adapter event enters
  `mac-cbj`, plus byte-equal continuation after every bridge `resume`;
- exact `Agenda.Pop`/`Engine.WorkOnTask` bridge execution, missing-disposition
  invalidity for false/abort/error outcomes, task-pop overflow,
  cross-request task rejection, and proof that `Engine.Run`, unit-focus, and
  deletion bookkeeping are unreachable;
- exact bridge heuristic-set/program-hash profile with no common-pack or
  other-domain heuristic instance beyond the common category unit, its
  once-per-execution 54-event charge, and the unchanged engine dispatch's
  22-event per-request surcharge;
- byte-equal no-artifact and `mac-cbj-empty` solution/transcript projections;
- concrete-memo failure under complete alias renaming;
- corrupted, wrong-family, random, recomputed, and match-only behavior crossed
  with every cohort, with nonauthoritative artifacts unable to prune;
- AC-3 and CBJ microcases independent of production semantics;
- failed-branch literal capture before restoration, root projection from
  immediate edge failure, initial propagation failure, recursive failure and
  resumed execution, and activation-local conflict reset when a variable is
  revisited;
- literal full-task attainability multiplicities and acquisition/prune/no-match
  cap enforcement without importing experiment code, including mechanical
  exclusion of reset preflight, recomputed acquisition, and match-only
  completion segments from the zero-completion resume cap;
- exact 24-cell development/validation/locked support, stratum draw order, and
  power resampling with no template or missing-bit substitution;
- satisfiability parity and complete omitted-branch solution-set audit for every
  development fixture;
- exact work conservation under success, no-solution, and budget exhaustion;
- production/oracle/baseline import and source-separation checks;
- absence of Part 2 paths, seeds, receipts, reports, and `.git/nous-attempts`
  reads in source constants, runtime open traces, dependencies, stores, and
  reports;
- deterministic stream/randomization vectors, reruns, statistics, report
  encoding, exact 96-byte record layout/operand reconstruction, transcript
  reducer parity, acquisition ordinal/chunk ownership and golden reconstruction
  of `A`, all 13/26 chunk hashes, acyclic payload/manifest/report hashes,
  raw/gzip worst-case bounds, and every per-execution/bundle cap scope;
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
same principle in propositional search. V2 adopts their central causal test—can
recorded failure prevent repeated work?—but intentionally does not implement
their general algorithms.

The exhaustive target certificate is stricter and less scalable than normal
clause learning. That is deliberate: in this first bounded trial, a work win
counts only after the learned abstraction has paid for an independently
inspectable proof that its prune is sound. A valid null would therefore be
useful evidence that this proof regime overwhelms the small search saving and
should be redesigned before broader nogood grammars are attempted.
