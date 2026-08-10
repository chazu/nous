# Guarded action-relations vocabulary plan

## Status and authority

Status: provisional Part 3 Vocabulary 3 plan, revision 2.

Revision 1 was committed at
`971aad8b223e98d5e4d56f8e395c8de96543663e` and unanimously rejected by
architecture, action-semantics, and experimental-validity review. Revision 2
closes the reported blockers by freezing oriented sleep-proof chains,
baseline-specific eligibility witnesses, total action/observation/guard
semantics, ordinary-CUE utility tasks, policy-blind fixture attempts and
strata, exact baseline/cache/statistical algorithms, bounded ordered evidence
journals, canonical report wires, and committed review/build/attempt authority.

This plan narrows the accepted
[Part 3 vocabulary research program](vocabulary-research-program-v3.md) without
amending it. Its exact lane identity is:

- domain: `domains/actionrelations`;
- production semantics: `internal/vocab/actionrelations`;
- independent fixtures: `internal/actionrelationfixture`;
- independent oracle: `internal/actionrelationoracle`;
- experiment and evidence guard: `internal/actionrelationexp`;
- command: `nous actionrelation-trials`;
- seed authority: `part3/actionrelations/v1`; and
- plan path: `docs/guarded-action-relations-vocabulary-plan.md`.

No implementation or protected panel is authorized until independent
architecture, action-semantics, and experimental-validity reviewers accept the
same exact plan commit. After implementation, those three scopes must accept
the same exact implementation commit and a canonical review manifest must bind
the protected source surface before development can run.

## Research question and claim boundary

Can ordinary Nous heuristics learn an alpha-invariant, state-guarded
commutativity relation from explicit action executions, materialize its
counterevidence, and use it to reduce later interleaving exploration under a
common lifecycle-work ledger without losing any terminal behavior?

V1 makes exactly one learned-relation claim: guarded commutativity. `Enables`,
`disables`, and `conflicts` are retained observation labels and negative
evidence, not promoted capabilities. The learned relation is not itself
permission to prune. It may only propose a pair for a charged local diamond
certificate. A search omission is legal only when that certificate succeeds at
the exact public state where the sleep-set rule uses it.

The distinguishing artifact is an inspectable set of tied guarded relation
schemas. Its required causal use is to avoid failed local-certificate attempts
and to admit successful sleep-set propagation in renamed later worlds. Merely
sorting actions, comparing final states, caching an exact state/action pair, or
running a conventional partial-order reducer does not satisfy the claim.

The claim is restricted to deterministic finite action systems, explicit full
state observations, finite action-occurrence multisets, and reachability of
terminal behaviors. It is not a concurrency-memory-model, linearizability,
liveness, fairness, intermediate temporal-logic, external-system, or real
software-scheduler claim.

## Historical and cross-vocabulary isolation

This lane does not import or read the existing protocol vocabulary, the Part 2
active-causal packages or artifacts, transformation-schema evidence, or any
other learned Store. It differs from `domains/protocols`: protocols evaluate a
supplied unary transform against a supplied machine relation; action relations
learn a conditional binary relation between state-changing actions and use
locally certified instances to compress a later search.

The complete production dependency graph may contain only the Go standard
library and the lane's own pure package. The scoped DSL adapter may additionally
depend on the base DSL and unit packages. Fixture, oracle, and experiment
packages remain separate. Source-boundary tests reject imports, constants,
fixture paths, reports, seeds, receipts, and runtime file reads from the closest
earlier packs.

## Frozen manifest

The implementation must reproduce this semantic object exactly before any
panel constructor is callable:

```json
{
  "experiment_version": "actionrelations/v1",
  "seed_authority": "part3/actionrelations/v1",
  "state_version": "finite-action-state/v1",
  "action_version": "finite-action/v1",
  "observation_version": "action-pair-observation/v1",
  "guard_version": "action-guard/v1",
  "relation_version": "guarded-action-relation/v1",
  "certificate_version": "local-diamond-certificate/v1",
  "search_version": "certified-sleep-search/v1",
  "cost_version": "actionrelation-lifecycle/v1",
  "statistics_version": "paired-stratified-ratio/v1",
  "report_version": "actionrelation-trials/v1",
  "evidence_version": "actionrelation-packed-evidence/v1",
  "maximum_cells": 3,
  "minimum_cell_value": 0,
  "maximum_cell_value": 3,
  "maximum_event_count": 8,
  "maximum_actions": 8,
  "maximum_reachable_states": 64,
  "maximum_history_length": 8,
  "maximum_competence_sequences": 40320,
  "maximum_utility_histories": 65536,
  "maximum_normalized_guards": 512,
  "maximum_training_observations": 256,
  "training_observations_per_curriculum": 16,
  "utility_worlds_per_curriculum": 6,
  "development_seeds": {"start": 851001, "count": 16, "step": 1},
  "validation_seeds": {"start": 852001, "count": 24, "step": 1},
  "locked_curricula": 32,
  "maximum_generator_attempts": 32,
  "generator_work_cap_per_attempt": 1000000,
  "generator_work_cap_per_curriculum": 32000000,
  "development_power_outer_replicates": 2000,
  "development_power_inner_replicates": 2000,
  "bootstrap_replicates": 10000,
  "randomization_replicates": 10000,
  "minimum_locked_work_reduction": 0.15,
  "minimum_locked_saving_coverage": 0.80,
  "minimum_locked_power": 0.80,
  "alpha": 0.05,
  "randomness_version": "sha256-counter-index/v1",
  "maximum_logical_record_bytes": 65536,
  "maximum_pack_bytes": 16777216,
  "maximum_pack_index_rows": 4096,
  "maximum_pack_index_bytes": 1048576,
  "maximum_development_evidence_bytes": 268435456,
  "maximum_validation_evidence_bytes": 402653184,
  "maximum_locked_evidence_bytes": 536870912,
  "maximum_report_bytes": 16777216,
  "tie_policy": "maximum-positive-coverage-then-minimum-literals-retain-all-ties-unanimous-use",
  "mutation_enabled": false
}
```

All ceilings are rejection bounds, not generation targets. Any fixture whose
reachable state space, complete history universe, or serialized evidence
crosses a bound is rejected before policy execution and is not replaced with a
post-hoc easier seed.

## Canonical action system

### State

A state wire is:

```text
["finite-action-state/v1", [[cellName,value]...], [eventSymbol...]]
```

There are one to three cells. Names are lowercase ASCII identifiers of one to
eight bytes, unique and sorted in canonical bytes. Values are integers 0
through 3. The event trace has at most eight lowercase ASCII symbols of one to
eight bytes and preserves order. Event order is therefore observable. Parsing
rejects duplicate cells, unsorted rows, unknown fields, noncanonical JSON,
invalid UTF-8, trailing bytes, and oversized values.

Exact future state is the complete canonical state, not a projection that can
hide applicability or event differences. A terminal behavior additionally
contains the sorted semantic identities of all unconsumed action occurrences:

```text
["action-terminal/v1", state, [remainingOccurrenceDigest...], terminal]
```

`terminal` is `complete` when no occurrence remains and `deadlock` when
occurrences remain but none is applicable. A sleep-set-blocked node is not a
terminal behavior and contributes no behavior row.

Every world supplies a canonical alpha map by sorting its declared cell names
and replacing them with roles `c0`, `c1`, and `c2`. Cell names are presentation
only after parsing. Every referenced cell must occur in the supplied state;
the generator rejects a world that violates this invariant. Applying a valid
action to any separately supplied state that lacks one of its cells returns
`inapplicable`, never a partial update or `invalid-input`.

### Actions

Every action has the fixed wire:

```text
["finite-action/v1", name, kind, x, y, n, symbol]
```

`name` is presentation identity only. Semantic matching never consumes it.
Unused fields are the canonical empty string or zero. The eight kinds are:

| Kind | Legal operands | Applicability and effect | Read/write footprint |
| --- | --- | --- | --- |
| `add` | `x`, `n` in `{-2,-1,1,2}` | applicable iff `0 <= x+n <= 3`; assign `x+n` | read/write `x` |
| `set` | `x`, `n` in `0..3` | always; assign `n` | write `x` |
| `transfer` | distinct `x,y`, `n` in `{1,2}` | applicable iff `x>=n` and `y+n<=3`; subtract/add | read/write `x,y` |
| `swap` | distinct `x,y` | always; exchange values | read/write `x,y` |
| `claim` | `x` | applicable iff `x==0`; assign `1` | read/write `x` |
| `release` | `x` | applicable iff `x==1`; assign `0` | read/write `x` |
| `check` | `x`, `n` in `0..3` | applicable iff `x==n`; no state change | read `x` |
| `emit` | `symbol` | applicable iff trace length is below 8; append symbol | write event trace |

After applying the world's alpha map, the name-free semantic wire is:

```text
["finite-action-semantic/v1",kind,xRole,yRole,n,symbol]
```

An occurrence is
`["action-occurrence/v1",semanticAction,ordinal]`. Within each equal-semantic
group, ordinals are the consecutive integers starting at zero; no two
occurrences may share an ordinal. Groups are sorted by semantic bytes before
ordinals are assigned. Presentation names and declaration order are never
inputs. Consequently cell renaming, action renaming, and declaration
permutation preserve semantic occurrence bytes. Canonical pair order is by
occurrence bytes. Relation artifacts contain neither presentation names nor
occurrence ordinals.

Production semantics may parse/canonicalize one state or action, report one
explicit action's local facts, test one action's applicability, apply one action,
compare two complete states, evaluate one named guard literal, extend one guard
by one literal, or validate one already assembled artifact. It may execute one
explicit supplied history for competence. It may not classify an action pair,
execute both orders behind one word, construct a diamond, enumerate histories,
learn a guard, compute a persistent/sleep set, or perform partial-order
reduction.

## Relation and guard language

### Local commutativity

Actions `a` and `b` commute at state `s` exactly when:

1. both are applicable at `s`;
2. `b` is applicable after applying `a` and `a` is applicable after applying
   `b`;
3. all four applications terminate without error; and
4. `apply(apply(s,a),b)` and `apply(apply(s,b),a)` are byte-identical complete
   canonical states.

Because the event trace is part of state, different event orders cannot be
mistaken for commutativity. Observation classification is total and oriented;
`a` and `b` retain canonical pair order throughout:

| `a(s)` | `b(s)` | `b(a(s))` | `a(b(s))` | equal finals | label |
| --- | --- | --- | --- | --- | --- |
| yes | yes | yes | yes | yes | `commutes` |
| yes | yes | yes | yes | no | `conflicts` |
| yes | yes | no | yes | n/a | `a-disables-b` |
| yes | yes | yes | no | n/a | `b-disables-a` |
| yes | yes | no | no | n/a | `mutual-disables` |
| no | yes | n/a | yes | n/a | `b-enables-a` |
| no | yes | n/a | no | n/a | `inapplicable` |
| yes | no | yes | n/a | n/a | `a-enables-b` |
| yes | no | no | n/a | n/a | `inapplicable` |
| no | no | n/a | n/a | n/a | `inapplicable` |

`n/a` applications are not executed. A malformed action, state, or returned
transition is `invalid`. Every non-`commutes` label is negative evidence; the
directional labels are diagnostics rather than learned capabilities.

The exact observation wire is:

```text
["action-pair-observation/v1",stateDigest,aOccurrenceDigest,bOccurrenceDigest,
 aInitialRowDigest,bInitialRowDigest,bAfterARowDigestOrNull,
 aAfterBRowDigestOrNull,abStateDigestOrNull,baStateDigestOrNull,label]
```

Null positions correspond exactly to `n/a` cells in the table. Assemblers
reject omitted required rows, non-null forbidden rows, a noncanonical pair
orientation, or a label that does not reconstruct.

### Pattern

A relation pattern is the canonical unordered pair of action kinds plus the
alpha-normalized alias topology among `a.x`, `a.y`, `b.x`, and `b.y`. Missing
operands occupy an explicit `none` role. Cell names are replaced by
first-occurrence role integers. Numeric operands, symbols, action names, and
occurrence ordinals are not pattern constants.

### Guard

A guard is a sorted conjunction of zero, one, or two signed literals. For
`add`, `set`, `claim`, `release`, and `check`, primary is `x` and secondary is
absent; for `transfer` and `swap`, primary is `x` and secondary is `y`; for
`emit`, both are absent. An action has an argument exactly for `add`, `set`,
`transfer`, and `check`. The event trace is the distinguished footprint cell
`event`. The 15 atoms, evaluated without executing either action, are:

1. `read-write-disjoint`;
2. `primary-same`;
3. `secondary-same`;
4. `a-primary-b-secondary`;
5. `a-secondary-b-primary`;
6. `argument-equal`;
7. `argument-opposite`;
8. `symbol-equal`;
9. `shared-value-zero`;
10. `shared-value-max`;
11. `a-primary-zero`;
12. `a-primary-max`;
13. `b-primary-zero`;
14. `b-primary-max`; and
15. `combined-adds-in-bounds`.

Their exact Boolean formulae are:

- `read-write-disjoint` iff
  `Wa ∩ (Rb ∪ Wb) = ∅` and `Wb ∩ (Ra ∪ Wa) = ∅`, including
  `event` in an `emit` write footprint;
- the four `*-same` atoms iff both named roles are present and their
  alpha-normalized cell roles are equal;
- `argument-equal` iff both arguments exist and are equal;
- `argument-opposite` iff both actions are `add` and their arguments sum to
  zero;
- `symbol-equal` iff both actions are `emit` and their symbols are equal;
- the two `shared-value-*` atoms inspect the lowest alpha-role referenced by
  both actions and are false when no role is shared;
- each `a-primary-*` or `b-primary-*` atom is true iff that primary role is
  present and its current value is respectively zero or three; and
- `combined-adds-in-bounds` iff both are initially applicable `add` actions on
  the same role and the current value plus both arguments is in `0..3`.

Every condition not made true by those formulae is false, including every
missing-role case; literal polarity is then applied normally. A guard cannot
repeat an atom, contain both polarities of one atom, or include literals
outside ascending atom order. The empty conjunction is true. There are exactly
`1 + 2*15 + 4*C(15,2) = 451` normalized guards, below the frozen ceiling.

The arithmetic atom is a local fact, not a diamond result: it does not execute
either order, inspect event output, or authorize pruning.

The guard wire is:

```text
["action-guard/v1", [[atom,polarity]...]]
```

A guarded relation wire is:

```text
["guarded-action-relation/v1","commutes",pattern,guard,
 positiveObservationDigests,negativeObservationDigests]
```

Observation digests are sorted, unique, training-only evidence. A relation's
positive array contains every matching positive observation and its negative
array contains every evaluated negative observation, whether or not it
matched. Eligibility is reconstructed from the observation records and match
rows; evidence digests themselves never affect matching. A frozen artifact
retains every tied winning relation:

```text
["guarded-action-artifact/v1", [relation...], trainingRoot]
```

At causal-use time the artifact matches a state/pair only when every retained
relation has the pair's pattern and every retained guard evaluates true.
Unanimous use makes tie retention conservative rather than silently choosing a
favorable schema after training.

## Acquisition protocol

Each curriculum provides 16 public training observations in a committed order:
eight locally commuting and eight negative examples for one latent pair motif.
They span at least two independent alpha-renamings. Negatives include
inapplicability, one-way enabling/disabling, final-state conflict, event-order
conflict, and irrelevant surface similarity.

Ordinary CUE heuristics must construct each observation by visibly:

1. checking both initial applications;
2. applying `a`, applying `b`, and retaining both intermediate states;
3. applying the opposite action to each intermediate state when legal;
4. comparing the two complete results; and
5. materializing a canonical observation with its positive or diagnostic
   negative label and the digests of every input and result.

No Go helper returns the relation label. The experiment reducer reconstructs
the label after the heuristic has committed it and never returns the answer to
the VM.

For the observed pair pattern, heuristics allocate the unconditional root and
traverse the complete 451-guard refinement graph through one-literal extension
edges. The root has no parent. Each one-literal guard has the root as parent.
Each two-literal guard has exactly one parent: remove the lexicographically
larger signed literal. Thus there are exactly 450 refinement edges; no
alternative parent edge is materialized. Each candidate is evaluated against
every acquired observation through explicit guard matches. A candidate is
eligible only if it:

- matches at least four positive observations;
- covers positives from at least two alpha-renamings;
- matches no negative observation; and
- has a complete evidence row for all 16 observations.

Selection first maximizes positive coverage, then minimizes literal count, and
retains every exact tie in canonical byte order. A closed evidence barrier must
name all 451 candidates, every refinement parent, all 7,216 guard evaluations,
statuses, coverage counts, and tied survivors. The 7,216 rows are candidate-
observation conjunction results composed from exactly 13,920 signed-literal
rows (30 one-literal and 420 two-literal guards over 16 observations); the root
needs no literal row. Its identity is semantic guard
content, never its evidence-array digest. Freeze then constructs one artifact
from the barrier; no candidate can be evaluated, added, removed, or reordered
afterward.

Acquisition is driven by ordinary CUE tasks, not a Go learner. For each
observation, the driver publishes one explicit application or comparison task;
CUE calls the primitive words, writes one result row, and closes an observation
barrier only after all required rows exist. For every relation pattern CUE then
materializes one row for each of the 451 guards, one row for each of the 7,216
guard/observation results, all 13,920 constituent literal rows, and the unanimous
winner barrier. The driver
may validate exactly one supplied row, expose the next opaque task token, or
reject it; it may not evaluate a whole artifact, choose a candidate, or return
an omitted classification.

Terminals are `completed`, `no-discovery`, and `budget-exhausted`. `completed`
requires a nonempty artifact whose training evidence reconstructs exactly.
`no-discovery` requires that no eligible candidate exists after the complete
barrier. Exhaustion requires the next reserved operation to exceed the frozen
budget and may not be used to assert absence.

## Scoped DSL and Store boundary

`domains/actionrelations` is loaded with `domains/common` only and declares the
scoped extension `actionrelations`. Every word consumes the named input unit,
writes at most the named output unit, and has the following Store effect:

| Word | Input -> output | Store effect |
| --- | --- | --- |
| `ar-state-valid?`, `ar-action-valid?` | one byte object -> one validity row | append row |
| `ar-action-facts` | state/action -> one alpha-normalized fact row | append row |
| `ar-applicable?` | state/action -> one applicability row | append row |
| `ar-apply` | state/action/applicability -> one transition row | append state and row |
| `ar-state-equal?` | two state digests -> one equality row | append row |
| `ar-guard-root`, `ar-guard-extend` | explicit parent/literal -> one guard/edge | append candidate and edge |
| `ar-guard-match` | one signed literal/fact rows -> one Boolean evaluation | append row |
| `ar-observation-assemble` | required application rows/claimed label -> observation | append or reject |
| `ar-candidate-allocate`, `ar-candidate-result` | guard/evaluation rows -> candidate row | append row |
| `ar-close-guard-search` | complete ordered candidate roots -> barrier | append immutable barrier |
| `ar-freeze-relation` | barrier/tied rows -> relation or artifact | append immutable object |
| `ar-pattern-match` | one relation plus state/pair facts -> match row | append row |
| `ar-certificate-assemble` | witness/application rows -> certificate attempt | append row |
| `ar-certificate-attach` | node/edge/certificate/proof source -> propagation row | append row |
| `ar-meter` | no input -> current adapter-owned counter vector | none; read-only |

There is deliberately no whole-artifact Boolean word. During utility, the
driver publishes one search task naming a node, taken occurrence, sleeper
candidate, and policy witness variant. For a learned witness, CUE must write a
pattern row for every retained relation, its individual literal rows, every
relation-match row, a unanimous-use barrier, and only then a certificate
request. Static and dynamic policies write their respective explicit witness
row. Certificate construction likewise uses individual applicability, apply,
and state-equality calls. The Go driver validates one row and advances one DFS
edge; it cannot inspect an artifact and choose a pair on the policy's behalf.

Assembly words accept and validate explicitly supplied components; they do not
derive missing states, labels, guards, winners, certificates, sleep sets, or
search decisions. Every semantic action automatically emits a charged operation
through the adapter; a heuristic cannot increment or suppress a counter.
Scoped words are absent when another domain is loaded. The base VM, engine,
agenda, mutation machinery, common pack, existing domains, and existing
vocabulary semantics remain unchanged.

The Store holds first-class training cases, observations, guard candidates,
refinement edges, counterexamples, the closed barrier, tied relations, and the
frozen artifact. Exact occupied-name collisions are handled with deterministic
content-derived suffixes. Artifact identity is canonical content, never unit
name. Store-byte stability is tested across fresh runs and occupied-name
fixtures.

## Certified sleep-set search

### Local certificate

A certificate is built at one exact search state for two exact remaining
occurrences. The local diamond is authority-bearing; the policy-specific
eligibility witness only explains why the policy attempted it. The witness is
exactly one of:

```text
["learned-witness/v1",unanimousUseBarrierDigest]
["static-witness/v1",exactFootprintRowDigest]
["dynamic-witness/v1","all-pairs",candidateRowDigest]
```

Thus static and dynamic policies never depend on, or pretend to possess, a
learned artifact. A certificate contains:

- state, two action-occurrence, and eligibility-witness digests;
- both initial applicability results;
- both intermediate state digests;
- both crossed applicability results;
- both final state digests;
- the exact-state comparison result;
- the canonical representative occurrence; and
- the charged operation range that produced it.

The wire is:

```text
["local-diamond-certificate/v1",stateDigest,aDigest,bDigest,witness,
 abDigest,baDigest,true,
 representativeDigest,operationRoot]
```

It is valid only if the witness variant reconstructs and the four-transition
local commutativity definition independently reconstructs. A false or corrupted
relation can cause a failed learned attempt but cannot authorize pruning. No
certificate may be reused at a different state, for different occurrences, or
across worlds.

### Search rule

The utility search is deterministic depth-first search over
`(state, sortedRemainingOccurrences, proofMapRoot)`. The proof map maps each
sleeping occurrence to its oriented ownership proof. Enabled occurrences and
loop order use semantic occurrence bytes, not presentation names.

At a node `N`, let `S` be its valid prior sleepers and `E` the completed earlier
sibling occurrences. For each enabled `t` not in `S`, the exact child equation
is:

```text
candidateSleepers = (S union E) intersect childRemainingOccurrences
childSleepers = {u in candidateSleepers |
                 u is enabled at N and freshCertificate(N,t,u) succeeds}
```

Every failed, absent, stale, or ineligible certificate drops `u`; there is no
inherit-by-default rule. Each retained `u` gets this durable oriented wire:

```text
["sleep-propagation/v1",parentNodeDigest,takenOccurrenceDigest,
 sleepingOccurrenceDigest,source,sourceProofOrExploredBranchDigest,
 localCertificateDigest,childNodeDigest]
```

`source` is exactly `earlier-sibling` or `prior-sleep`. The former must name a
completed earlier representative subtree at the same parent. The latter must
name the valid parent proof for that exact sleeper. `childNodeDigest` commits
the successor state, remaining occurrences, and entire child proof-map root.
A sleep-blocked node is legal only when every enabled sleeper has a recursively
valid chain, through fresh local diamonds, to a completed earlier subtree.
The verifier replays that adjacent-swap chain; a certificate about a canonical
minimum pair cannot substitute for the actual oriented sleeper/taken pair.

Cache lifetime is one policy/world. Its key is
`[stateDigest,minOccurrenceDigest,maxOccurrenceDigest]` and its value contains
the complete success or failure result plus ordered proof calls. Successes and
failures are both cached; all certified policies get identical lookup,
construction, and reuse rights. Learned matching is still charged before a
cache lookup. Exact node deduplication is keyed by state, remaining occurrences,
and proof-map root, costs a lookup, and never substitutes for a propagation
proof.

The learned policy may omit certificate attempts when unanimous artifact use
fails; this can lose reduction but cannot lose behavior. It may not prune on a
relation match alone. Utility validation replays certificates and representative
chains only; it never discovers skipped schedules by secretly running complete
exploration after the learned artifact exists.

## Policies and controls

All policies receive identical public states, actions, occurrence sets, action
semantics, task order, budgets, and terminal scorer commitment.

1. `complete`: full deterministic exploration with exact state-node
   deduplication and no sleep set.
2. `lexical-order`: the complete policy with a fixed presentation-independent
   action order; it orders but never prunes.
3. `static-rw-sleep`: certified sleep search whose eligibility rule is the
   conventional static read/write independence test.
4. `dynamic-diamond-sleep`: the primary conventional baseline; it attempts a
   local certificate for every distinct candidate pair without a learned
   filter or semantic prefilter.
5. `nous-guarded-sleep`: acquisition plus the frozen artifact as the
   certificate-attempt filter and causal-use witness.
6. `no-guard-sleep`: the same acquisition path with only the unconditional root
   eligible.
7. `learned-no-use`: performs identical acquisition and artifact persistence,
   then uses complete exploration without the artifact.

The primary endpoint compares `nous-guarded-sleep` with
`dynamic-diamond-sleep`. Static read/write is a secondary conventional
baseline. Complete and lexical policies establish the unreduced behavior set.
No-guard tests conditionality; learned-no-use separates artifact acquisition
from causal search use.

For every node/taken occurrence, all three certified policies enumerate the
same candidates `(prior sleepers union completed earlier siblings)` in
occurrence order. Dynamic emits an `all-pairs` witness for every distinct
co-enabled candidate. Static emits a witness only when the exact footprint
formula is true. Nous emits one only after its per-relation CUE rows close a
true unanimous barrier. Candidate enumeration, child equation, DFS order,
node/cache keys, cache lifetime, success/failure caching, and retry prohibition
are otherwise byte-identical. Static and dynamic acquisition vectors are all
zero. The shared Nous acquisition transcript is stored once but its identical
work total is charged to `nous-guarded-sleep` and `learned-no-use`; no-guard
runs a separate root-only acquisition path.

Wrong-context, state/action alpha-renaming, artifact deletion, a bit-corrupted
guard, a relation with swapped pattern roles, forged certificates, and exact
state/action cache controls are mandatory ordinary tests. Corrupted artifacts
may waste work or cause no reduction but can never yield an accepted skip.

## Fixture families and information rights

The generator draws balanced curricula from eight learnable families:

1. disjoint-cell updates;
2. bounded additions on one shared cell;
3. equal assignments on one shared cell;
4. disjoint transfers;
5. disjoint swaps and transfers;
6. read-only checks with independent updates;
7. identical observable emissions; and
8. mixed longer histories embedding one learned motif among dependencies.

Every family includes negative observations from alias collisions, bound
failures, different assignments, competing transfers, event-symbol order,
one-way enabling, and destructive conflicts. Utility worlds rename every cell
and action, change legal numeric values, reorder declarations, preoccupy Store
names, add irrelevant actions, and place the motif at different history depths.

For each curriculum, acquisition uses the 16 training observations and utility
uses exactly six fresh worlds with six to eight occurrences. They comprise two
worlds in each policy-blind semantic stratum:

- `positive-effect`: at least four reachable, true local commutations have the
  latent training pattern and guard while failing static read/write
  independence;
- `neutral`: no reachable nonterminal state has more than one applicable
  occurrence, and no reachable pair has the latent training pattern; and
- `adverse`: at least four reachable true commutations lie outside the latent
  pattern or guard, while no reachable pair matches it.

Those predicates use only independent action semantics and the fixture's
precommitted latent motif. They cannot execute a policy or inspect a learned
artifact, work, search history, certificate count, reduction, or effect size.
The strata deliberately test helpful transfer, no-op transfer, and filtering
whose opportunity cost favors the all-pairs baseline.

Curriculum construction tries attempts `0..31` in order. Attempt bytes are
`SHA-256(canonical-json([seedAuthority,panel,panelAuthority,
curriculumOrdinal,attempt,purpose]))`; subsequent draws use the frozen counter
mapping below. Each attempt may consume at most 1,000,000 independent-generator
operations, and the curriculum at most 32,000,000. It is accepted only if all
eight training positives/negatives, alpha-renaming, all six fixed strata slots,
identifiability, state/history ceilings, and evidence preflight pass. The
attempt ledger commits every rejection predicate and work total. No replacement
seed is drawn. Exhausting attempt 31, either work cap, or any stratum slot makes
the panel mechanically invalid before a policy runs.

Family counts are exact: two curricula per family in development, three in
validation, and four in locked. A committed policy-blind permutation assigns
family and within-family ordinal before attempt generation.

Before a policy starts it may see canonical training state/action pairs,
utility initial states and action occurrences, the action semantics, budget,
opaque task tokens, and its own committed randomness. It may not see relation
labels, latent family, latent guard, complete terminal sets, generator accepted
attempt, other policies' traces, or scorer bytes. Training labels become
visible only after the policy pays for and commits the explicit four-transition
observation. Utility terminal truth is decoded only after artifact freeze and
policy termination.

During policy-blind generation the independent enumerator commits each world's
complete sorted terminal-behavior set and every reachable state/pair observation
label into a sealed truth object. Its digest is
in the fixture root before acquisition; its bytes are held in a separate scorer
pack that the experiment process cannot map until every policy terminates.
After disclosure, validation compares each explored terminal set to this prior
commitment. It replays only the submitted certificates, proof chains, and
representative subtrees for omissions. Complete exploration is permitted while
building the sealed fixture and in competence tests, never as a post-policy
utility audit or hidden reducer.

Development and validation generators use public frozen seed ranges. Locked
generation uses a receipt-committed random 32-byte root and domain-separated
HMAC seeds, erases the root before any policy runs, and reveals only its
commitment. Rejected attempts remain within one derived curriculum seed and do
not draw replacement seeds.

## Lifecycle-work ledger and budgets

The primary scalar is the sum of this 12-counter vector, with every weight
exactly one:

1. guard/candidate allocation;
2. guard refinement edge;
3. training action transition;
4. training applicability/state comparison;
5. candidate guard evaluation;
6. artifact Store write;
7. artifact Store read or relation lookup;
8. utility search transition;
9. certificate-only transition;
10. certificate predicate or guard check;
11. exact search-node/cache lookup; and
12. terminal construction/classification.

The adapter, not CUE and not `ar-meter`, emits and increments every primitive.
The exhaustive word-to-counter map is:

| Code | Word or driver event | Counter |
| --- | --- | --- |
| 1 | `ar-guard-root` | 1 |
| 2 | `ar-candidate-allocate` | 1 |
| 3 | `ar-guard-extend` | 2 |
| 4 | training `ar-apply` | 3 |
| 5 | training `ar-applicable?` | 4 |
| 6 | training `ar-state-equal?` | 4 |
| 7 | acquisition `ar-guard-match` | 5 |
| 8 | artifact-producing `ar-freeze-relation` | 6 |
| 9 | utility `ar-pattern-match` | 7 |
| 10 | artifact Store load | 7 |
| 11 | utility transition `ar-apply` | 8 |
| 12 | certificate-only `ar-apply` | 9 |
| 13 | certificate `ar-applicable?` | 10 |
| 14 | certificate `ar-state-equal?` | 10 |
| 15 | learned literal check | 10 |
| 16 | search-node lookup | 11 |
| 17 | proof-map lookup | 11 |
| 18 | certificate-cache lookup | 11 |
| 19 | terminal/deadlock construction | 12 |
| 20 | acquisition `ar-candidate-result` | 5 |

Validity parsing, evidence serialization, CUE scheduling, and task-token
movement are uncharged infrastructure and separately counted; they may not
perform any operation named in the table. Each listed invocation emits exactly
one operation and increments exactly one counter before semantic execution.
Failure and rejection cost the same as success. A cache hit costs counter 11
and cannot erase construction cost. A compound task publishes its complete
ordered primitive reservation first; if the reservation would cross a cap,
none executes and the terminal is `budget-exhausted`. The verifier derives each
counter from operation codes, rejects caller-supplied deltas, and proves
`total = sum(vector)`.

Each policy/curriculum has a 2,000,000-unit lifecycle cap, 65,536 complete-
history cap, 20,000 attributed-unit cap, and 10,000 engine-cycle cap. Physical
operation-event caps are separate and fixed per curriculum: shared Nous
acquisition 24,000; no-guard acquisition 128; complete 4,096; lexical 4,096;
static 4,096; dynamic 8,192; Nous 8,192; no-guard 4,096; learned-no-use 4,096.
Crossing any cap yields the honest frozen terminal; it never licenses partial
compound execution.

Total lifecycle work across acquisition plus all six utility worlds determines
the primary comparison. Post-freeze utility work, histories, transitions,
certificate attempts/successes, guard precision/recall, artifact size, and
amortization crossover are diagnostics only.

## Independent oracle and competence

`internal/actionrelationoracle` reimplements parsing, all eight action kinds,
applicability, transitions, guard atoms, local diamonds, proof-map validation,
terminal behavior normalization, and work reconstruction without importing
production semantics, DSL, fixtures, or experiment helpers. Its protected
utility path can replay only submitted search edges, certificates, proof chains,
and disclosed terminal commitments; its complete enumerator is a separately
guarded fixture/competence entry point that cannot accept a policy artifact.
The production package does not import the oracle.

Before protected execution, semantic competence exhaustively covers:

- every valid state for one-, two-, and three-cell tiny algebras whose reachable
  universe is at most 64 states;
- every ordered action-kind pair and legal operand-alias topology represented by
  the fixture families;
- all 451 guards against independently derived atom truth;
- every local-diamond truth table, including inapplicability and event order;
- every complete sequence up to the applicable 40,320 ceiling;
- production/oracle equality for complete and certified sleep terminal sets;
  and
- hand-checked unconditional, conditional, enabling, disabling, conflict,
  event, deadlock, and sleep-propagation witnesses.

Competence work is reported separately and cannot count toward marginal
utility. Passing it establishes bounded semantics, not the empirical claim.

## Packed evidence and repository authority

The transformation lane demonstrated that one Git file per semantic object is
operationally unacceptable. This lane instead commits bounded content-addressed
packs without weakening leaf verification.

There are two pack classes. Object packs start with `AROP1\n`, followed by
`uint32-big-endian length || canonical-record-bytes`; records are unique and
sorted by `SHA-256(recordBytes)`. Operation-journal packs start with `ARJR1\n`
and contain contiguous sequence-order records. Every journal record is exactly
128 bytes and contains, in order, version `uint8`, phase `uint8`, operation
`uint8`, status `uint8`, counter `uint8`, three zero bytes (8 total), sequence
`uint32` (4), run ID (16), previous-record digest (32),
ordered-input-vector root (32), ordered-output-vector root (32), and reserved
zero bytes (4). Its call ID is `SHA-256(record)`. Sequence starts at zero and
`previous` is zero only there; every subsequent record links its predecessor.
Input and output vectors point to individually canonical object records.
Version is 1; phases are acquisition 1 and utility 2; operation codes are the
numeric codes frozen in the ledger table; statuses
are success 1, semantic-failure 2, reservation-rejected 3, and cache-hit 4;
counter is 1 through 12. All integers are unsigned and big-endian where wider
than one byte. Compound reservations are canonical input objects rather than
mutable journal header fields.

Each index row is `[digest,offset,length,kind,sequenceOrNull]`. Object indexes
are digest ordered; journal indexes are sequence ordered. An index has at most
4,096 rows and 1 MiB. A logical object is at most 65,536 bytes. Any pack is at
most 16 MiB; index shards and deterministic split points are named in a root
manifest. The verifier checks headers, lengths, digests, canonical decoders,
complete index coverage, object uniqueness, journal sequence/chain continuity,
absence of trailing bytes, and root closure. Terminal behaviors, search nodes,
search edges, eligibility rows, certificate attempts, propagations, proof-map
entries, and work reservations are distinct logical leaves, never one summary
blob.

The evidence graph lists only regular committed pack and index files plus the
fixture root, execution manifests, review authority, competence root, and
report prerequisites. Logical leaves remain individually addressable and
digest-checked through pack indexes. The graph does not enumerate hundreds of
thousands of filesystem paths. Development, validation, and locked total-byte
caps are enforced before publication.

For each policy/curriculum the evidence includes canonical fixture views,
execution manifest, ordered calls, exact Store boundary where applicable,
frozen artifact, search nodes and edges, certificate records, oriented skip
records, terminal behavior leaves, work vector, and terminal. A certificate
names its contiguous call-ID range. A skip names a completed representative
subtree and every adjacent-swap propagation. The verifier reconstructs DFS
order, cache reuse, every omitted edge, and work from these leaves rather than
trusting policy summaries. Primary and audit executions independently reload
the committed fixture bytes and must produce identical semantic rows,
transcript roots, artifacts, behaviors, and work.

Before any panel attempt, an algebraic preflight reserves the worst-case bytes:
the fixed per-curriculum event caps total 56,992 records, or 7,294,976 journal
bytes; object payload and indexes/roots are each capped at 4,000,000 bytes.
Including up to 1,482,240 bytes of pack headers and root manifests, the
curriculum reservation is 16 MiB. Therefore development
(16), validation (24), and locked (32) fit their exact 256, 384, and 512 MiB
ceilings. This is one quarter of revision 1's locked cap. The preflight runs
before fixture construction or policy work; a
logical record, pack, index shard, curriculum, or panel that cannot fit is
mechanically invalid rather than truncated.

Plan acceptances are committed in `docs/actionrelations-plan-reviews.json` and
implementation acceptances in
`docs/actionrelations-implementation-reviews.json`. Each ordered row is:

```text
[scope,reviewerTask,round,reviewedCommit,archiveSHA256,
 verdictSHA256,"accepted",attestationSHA256]
```

The attestation is SHA-256 of the canonical preceding seven fields. Review
authority comes from the archived independent reviewer result; the hash binds
rather than impersonates reviewer identity. All three scopes must name the
same exact commit and source archive. A rejection invalidates the whole round.

Before development, `docs/actionrelations-build-authority.json` binds the
accepted plan and implementation commits, both review-manifest digests, every
tracked Go/CUE/C/compiler input, module/toolchain file, command, test, plan and
Part 3 umbrella input, the source-tree hash, `go version`, `mise.toml`,
GOOS/GOARCH/CGO, build flags, produced binary SHA-256, and `go version -m`
digest. Only the allowlisted `.nous/bin/actionrelation-nous-v1` may execute a
panel; `go run`, overlays, build tags, `GOFLAGS`, `GOWORK`, and compiler inputs
outside the manifest are rejected. Runtime environment is reduced to an
allowlist recorded in the execution manifest. Git replacement, alternates,
shallow authority, unsafe config, hooks, in-progress operations, symlinks,
dirty protected files, and untracked or ignored compiler inputs are rejected.

Canonical panel paths are:

```text
.nous/actionrelations-v1-development-report.json
.nous/actionrelations-v1-development-evidence/
.nous/actionrelations-v1-validation-receipt.json
.nous/actionrelations-v1-validation-report.json
.nous/actionrelations-v1-validation-evidence/
.nous/actionrelations-v1-locked-receipt.json
.nous/actionrelations-v1-locked-report.json
.nous/actionrelations-v1-locked-evidence/
```

Canonical top-level wires are closed arrays, not extensible objects:

```text
["actionrelation-fixture-root/v1",panel,authority,curriculumRoots,scorerRoot]
["actionrelation-execution/v1",panel,authority,sourceRoot,binaryDigest,
 environmentRows,fixtureRoot,runIDs,evidenceRoot]
["actionrelation-policy-row/v1",panel,curriculum,family,stratum,policy,
 acquisitionTerminal,searchTerminal,artifactDigest,workVector,totalWork,
 matchCounts,certificateCounts,sleepCount,historyCount,terminalSetDigest,
 behaviorEqual,budgetRemaining,operationRoot]
["actionrelation-evidence-graph/v1",fixtureRoot,executionDigest,
 objectPackRoots,journalPackRoots,indexRoots,policyRowRoot,reportDigest]
["actionrelation-receipt/v1",panel,state,claimCommit,sourceRoot,
 fixtureCommitment,attemptRoot,reportDigest,evidenceRoot,reason]
["actionrelation-report/v1",panel,authority,manifestDigest,reviewDigests,
 buildDigest,fixtureRoot,receiptDigest,policyRowsRoot,mechanicalGates,
 primaryRatio,confidenceInterval,randomizationP,savingCoverage,power,
 classification,evidenceRoot]
```

Arrays have exactly the displayed arity and canonical nested row order.
Unknown, missing, duplicated, or reordered fields are invalid. Policy rows are
ordered by curriculum, world stratum/ordinal, then frozen policy order. The
evidence graph closes over every logical object and journal call; no referenced
or unreferenced leaf may be missing or extra.

Validation and locked use a committed two-phase preclaim. `-stage claim`
creates a `claimed` receipt without constructing a seed; it must be committed
and pushed on append-only `main`. `-stage execute` requires clean HEAD equal to
`origin/main`, the exact claimed receipt at HEAD, and no previous attempt root,
then atomically persists `running` before the seed constructor is callable.
Every exit finalizes and requires publication as `published` or `invalid`;
neither state can run again. This is a repository-level one-shot claim under
the stated append-only-history authority, not protection against a malicious
history rewrite. Development is repeatable only in a new absent evidence
namespace and never overwrites an existing report.

## Reports, inference, and progression

The policy-row wire above is authoritative; every aggregate in the report is
recomputed from those leaves. Training precision/recall and per-stratum
matched pairs are contained in `matchCounts`; attempted/successful/cached-
success/cached-failure counts are in `certificateCounts`.

Mechanical validity precedes empirical classification and requires:

- exact manifest, review, fixture, source, pack, and evidence-graph authority;
- primary/audit equality and independent replay;
- production/oracle semantic and guard agreement;
- exact work conservation and budget reservations;
- immutable post-freeze artifacts and delayed scorer access;
- zero false matched commutativity claims for the Nous artifact over every
  reachable utility state/pair in the panel;
- exact terminal-behavior equality for Nous, static, dynamic, complete, and
  learned-no-use policies; and
- every accepted sleep propagation backed by a fresh valid certificate.

No-guard may have false relation matches as an ablation diagnostic, but no false
match may authorize a skip; its behavior set must still remain exact. A failed
certificate is valid measured work. A missing, forged, reused, or false accepted
certificate is mechanical invalidity.

The paired primary ratio is:

```text
sum(total lifecycle work of nous-guarded-sleep)
------------------------------------------------
sum(total lifecycle work of dynamic-diamond-sleep)
```

Only curricula where both primary policies mechanically complete enter the
ratio; any omitted curriculum makes a protected result `valid-null`, never a
smaller denominator. A zero dynamic denominator is mechanical invalidity.
Ratios and threshold comparisons use nonnegative arbitrary-precision integer
cross-products; decimal renderings are diagnostics.

All pseudo-random draws use
`u = bigEndian(first8(SHA-256(canonical-json([seedAuthority,panel,
authority,replicate,purpose,draw]))))`. An index into `n` rows is
`floor(u*n/2^64)` (multiply-high); a randomization sign is `u & 1`. Bootstrap
resamples within each family with the panel's exact count: 2 development, 3
validation, or 4 locked rows per family. Its statistic is the ratio of summed
work. The 10,000 sorted ratios use zero-based indices 249 and 9,749, breaking
equal rational values by replicate index.

The paired randomization statistic is
`abs(sum_i(nousWork_i - dynamicWork_i))`. Each replicate independently swaps
the two work values within every curriculum. A replicate is extreme when its
statistic is greater than or equal to the observed statistic, including ties;
`p = (1 + extremeCount) / 10001`.

Development power draws 2,000 synthetic locked panels, each with four
development curricula sampled with replacement from each of the eight family
strata. Every synthetic panel recomputes all locked predicates using 2,000
inner bootstrap and randomization replicates, whose interval indices are 49
and 1,949 and whose p-value denominator is 2,001. Inherited mechanical
completion, behavior equality, and false-match status are required. Power is
the fraction satisfying every locked positive predicate; progression needs at
least 1,600 successes.

A mechanically valid locked result is `valid-positive` only when:

1. all 32 paired curricula complete with exact terminal behavior;
2. aggregate Nous work is at most 85% of dynamic-diamond work;
3. the bootstrap upper confidence bound for the ratio is below 1;
4. at least 80% of curricula have strictly lower Nous work;
5. the paired-randomization p-value is below 0.05; and
6. the Nous artifact has zero false matched commutativity claims.

Otherwise it is `valid-null`. Development is
`interim-power-authorized` or `interim-power-unauthorized`. Validation is
`interim-valid` only after a committed authorized development bundle and all
mechanical gates. Locked requires committed authorized development and
validation bundles plus an exact `actionrelations/v1:<clean-HEAD>` token.

## Closed terminal taxonomy

| Level | Terminals |
| --- | --- |
| action application | `applied`, `inapplicable`, `invalid-input` |
| observation | `commutes`, `a-enables-b`, `b-enables-a`, `a-disables-b`, `b-disables-a`, `mutual-disables`, `inapplicable`, `conflicts`, `invalid` |
| certificate | `certified`, `not-certified`, `invalid` |
| acquisition | `completed`, `no-discovery`, `budget-exhausted` |
| utility search | `completed`, `budget-exhausted` |
| execution | `mechanically-valid`, `invalid` |
| development | `interim-power-authorized`, `interim-power-unauthorized`, `invalid` |
| validation | `interim-valid`, `invalid` |
| locked | `valid-positive`, `valid-null`, `invalid` |
| receipt | `claimed`, `running`, `published`, `invalid` |

Wrong answers, false no-discovery, false exhaustion, terminal-set mismatch, or
accepted invalid certificates are mechanical invalidity. Honest
`no-discovery`, failed certificate attempts, and budget exhaustion are valid
empirical outcomes when their exact predicates reconstruct.

## Required tests and adversarial audits

Before implementation review:

- canonical parsers reject malformed, noncanonical, duplicate, oversized, and
  trailing-byte inputs;
- every action kind and footprint agrees with the independent oracle;
- missing-cell, semantic-occurrence, declaration permutation, presentation
  rename, and duplicate-ordinal behavior is exact;
- the total oriented observation table and every null/not-null position agree
  with the independent oracle;
- all guard formulae, counts, polarity, normalization, 451 candidates, and 450
  canonical refinement edges are exhaustive and exact;
- observation construction cannot omit, reorder, duplicate, or relabel any of
  the four transition applications;
- relation selection and tied unanimous-use behavior reconstruct from Store;
- alpha-renaming and occupied-name tests preserve semantic artifacts;
- complete and certified sleep search agree on exhaustive tiny universes;
- prior-sleep and earlier-sibling ownership chains, the exact child equation,
  cache success/failure reuse, dropped failed sleepers, and proof-map node
  identities survive adversarial replay;
- local, propagated, stale-state, wrong-occurrence, wrong-artifact, and
  event-hidden certificate forgeries are rejected;
- learned relation deletion removes its causal use; corruption cannot authorize
  a skip; no-guard and learned-no-use controls behave as defined;
- every work counter has success, failure, cap-boundary, and forged-terminal
  tests; caller-supplied increments and partial reservations fail;
- dynamic considers every distinct co-enabled candidate; static and learned
  differ only in their explicit witness rows; cache and acquisition charges
  reconstruct exactly;
- all 32 generator attempts, six stratum slots, policy-blind acceptance rules,
  family counts, exhaustion ledgers, and scorer mapping restrictions are
  mutation-tested;
- exact SHA-256 draws, multiply-high indices, rational bootstrap bounds,
  randomization ties, zero denominators, and nested power are golden-tested;
- pack split, index, digest, truncation, duplicate, reordering, oversized-file,
  journal-chain, sequence, call-range, capacity-preflight, and path traversal
  attacks fail;
- fixture/scorer access traces prove truth is committed before policy and
  unavailable until termination; utility validation cannot invoke the complete
  enumerator;
- plan/implementation review rows, build binary and environment, claimed/running
  receipt transitions, dirty Git, and replayed protected attempts fail closed;
- dependency and source-surface scans enforce every package boundary; and
- protected constructors have one direct guarded caller and no test, reflection,
  linkname, function-variable, or generic-panel backdoor.

Implementation reviewers must attack the source archive rather than the dirty
working tree and must not run development, validation, or locked panels. They
must independently try to forge a relation, evidence barrier, work ledger,
sleep propagation, terminal behavior set, packed leaf, report, receipt, and Git
authority. Any blocker produces a new implementation commit and a complete new
three-scope review round.

## Delivery sequence

1. obtain unqualified architecture, semantics, and experimental-validity
   acceptance of one exact plan commit;
2. commit and push that accepted plan before touching production code;
3. implement pure semantics and the independent oracle first;
4. implement scoped words, CUE acquisition, Store artifacts, and ordinary safe
   competence tests;
5. implement certified sleep policies, baselines, fixtures, ledger, packed
   evidence, reports, and repository guards;
6. run only ordinary and safe development-shaped tests;
7. obtain unanimous adversarial implementation acceptance of one exact commit;
8. commit review authority and run the public development panel once;
9. continue only through frozen progression gates; and
10. record and push the result, limitations, exact verification commands, and
    Part 3 capability-matrix update before beginning Vocabulary 4.

## Research anchors

- Peled's
  [representative-sequence formulation](https://ai.dmi.unibas.ch/research/reading_group/peled-cav1993.pdf)
  motivates retaining one representative per execution-equivalence class.
- Godefroid, Peled, and Staskauskas's
  [state-space reduction study](https://patricegodefroid.github.io/public_psfiles/ieee-tse96.pdf)
  motivates the conventional partial-order baselines and explicit preservation
  check.
- Flanagan and Godefroid's
  [dynamic partial-order reduction](https://doi.org/10.1145/1040305.1040315)
  motivates discovering dependence during concrete executions and is a
  stronger baseline than lexical or static ordering alone.
- Abdulla et al.'s
  [source-set and optimal DPOR account](https://doi.org/10.1145/3073408)
  clarifies that minimal trace coverage is an algorithmic baseline, not proof
  that Nous learned a reusable relation.

These algorithms are research anchors and conventional controls. The Nous
claim requires a first-class guarded relation, retained counterevidence,
alpha-invariant transfer, and independently replayed causal use.
