# Guarded action-relations vocabulary plan

## Status and authority

Status: provisional Part 3 Vocabulary 3 plan, revision 4.

Revision 1 was committed at
`971aad8b223e98d5e4d56f8e395c8de96543663e` and unanimously rejected by
architecture, action-semantics, and experimental-validity review. Revision 2
closes the reported blockers by freezing oriented sleep-proof chains,
baseline-specific eligibility witnesses, total action/observation/guard
semantics, ordinary-CUE utility tasks, policy-blind fixture attempts and
strata, exact baseline/cache/statistical algorithms, bounded ordered evidence
journals, canonical report wires, and committed review/build/attempt authority.

Revision 2 was committed at
`69d92effdeb9f975e591a1eaa4f084080ea04216` and unanimously rejected. Revision
3 removes two impossible digest cycles, uses a presentation-independent whole-
world canonical labeling, freezes every generator and statistical draw,
separates world from curriculum accounting, derives evidence capacity by leaf
class, and narrows protected-panel authority to precommitted attempt identity
and cooperative replay detection.

Revision 3 was committed at
`3584db169b2b5a7ac62a917da25e9df48fe8eeb1` and unanimously rejected. Revision
4 makes initial applicability part of relation-instance matching, splits
presentation wrappers from semantic support, closes every proof-root wire,
uses finite stratum skeleton catalogs instead of improbable random filtering,
freezes the full generator cost schedule and inner-power ordinal, and adds
capacity-derived acquisition tables plus a realizable call-detail layout.

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
  "maximum_development_evidence_bytes": 654311424,
  "maximum_validation_evidence_bytes": 973078528,
  "maximum_locked_evidence_bytes": 1291845632,
  "maximum_report_bytes": 14680064,
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

Every world obtains its alpha map by enumerating all `n!` bijections from its
one to three declared cells to roles `c0..c(n-1)`. For each bijection it
normalizes the initial state and complete semantic action multiset, sorts equal
actions with multiplicity preserved, and serializes:

```text
["finite-action-world-core/v1",normalizedInitialState,
 sortedSemanticActionMultiset]
```

The lexicographically least canonical bytes win. If symmetric bijections tie,
their normalized world bytes are already identical, so no presentation map is
retained as a tiebreaker. All execution, occurrence assignment, pattern
orientation, DFS ordering, and terminal construction use only this normalized
world. Cell names are presentation only after parsing; arbitrary bijective
renaming therefore cannot change semantic bytes. Every referenced cell must
occur in the supplied state; the generator rejects a world that violates this
invariant. Applying a valid action to any separately supplied state that lacks
one of its cells returns `inapplicable`, never a partial update or
`invalid-input`.

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
occurrences in the same equal-semantic group may share an ordinal. Groups are
sorted by semantic bytes before ordinals are assigned. Presentation names and
declaration order are never inputs. Consequently cell renaming, action renaming, and declaration
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

The semantic observation core is:

```text
["action-pair-observation/v1",stateDigest,aOccurrenceDigest,bOccurrenceDigest,
 aInitialRowDigest,bInitialRowDigest,bAfterARowDigestOrNull,
 aAfterBRowDigestOrNull,abStateDigestOrNull,baStateDigestOrNull,label]
```

Null positions correspond exactly to `n/a` cells in the table. Assemblers
reject omitted required rows, non-null forbidden rows, a noncanonical pair
orientation, or a label that does not reconstruct. Presentation lineage is a
separate canonical pair:

```text
["action-presentation-view/v1",originalState,orderedOriginalActions,
 semanticWorldCoreDigest,[[originalActionName,occurrenceDigest]...]]
["action-view-evidence/v1",semanticObservationDigest,
 presentationViewDigest,normalizationProofDigest]
["action-normalization-proof/v1",presentationViewDigest,
 [[originalCellName,canonicalRole]...],semanticWorldCoreDigest]
```

The mapping is ordered by original declaration position and must be bijective.
The verifier reruns whole-world normalization and requires the view's state and
mapped observations to equal the named semantic world/core. Arbitrary digests,
missing actions, and two wrappers for the same presentation bytes are invalid.
When semantic-world candidates tie, the proof alone uses the lexicographically
least canonical mapping-row array; it never enters semantic identity.

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

Observation digests are sorted, unique, presentation-free training cores. A relation's
positive array contains every matching positive observation and its negative
array contains every evaluated negative observation, whether or not it
matched. Eligibility is reconstructed from the observation records and match
rows; evidence digests themselves never affect matching. A frozen artifact
retains every tied winning relation:

```text
["guarded-action-artifact/v1", [relation...], semanticTrainingRoot]
```

The separate
`["action-training-evidence/v1",semanticTrainingRoot,viewEvidenceRoot]`
commits presentation lineage without changing relation or artifact identity.
Support and precision count unique semantic observation cores exactly once.
Alpha-transfer coverage requires valid wrappers from both frozen name banks but
never adds support.

At causal-use time the artifact matches a state/pair only when every retained
relation has the pair's pattern and every retained guard evaluates true.
Unanimous use makes tie retention conservative rather than silently choosing a
favorable schema after training.

A relation *instance* matches only after two additional nonlearned predicates:
both occurrences are explicitly and successfully checked applicable at the
current state, and the event trace length is `0..6`. Those checks are individual
charged CUE rows and precede pattern/guard rows. Six is the maximum admissible
trace when two occurrences remain in an initially empty world of at most eight
occurrences. Pattern-plus-guard is the reusable artifact; applicability and
context are public instance preconditions. Acquisition candidate truth,
identifiability, utility match counts, and the zero-false-match gate all use the
complete conjunction:

```text
admissible-context and both-initially-applicable and pattern and guard
```

## Acquisition protocol

Each curriculum provides 16 unique semantic training observations in committed
order: eight locally commuting and eight negative examples for one latent pair
motif. Every core has exactly two verified presentation wrappers, one per
frozen name bank. Negatives include
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
- every covered positive core has both verified alpha-renaming wrappers;
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

The remaining set, proof map, and earlier-sibling authority have exact wires:

```text
["remaining-occurrences/v1",[occurrenceDigest...]]
["sleep-proof-map/v1",[[sleeperDigest,propagationCoreDigest]...]]
["completed-subtree/v1",parentNodeDigest,takenOccurrenceDigest,
 subtreeRoot,terminalSetRoot,"completed"]
```

Occurrence and proof-map rows are sorted by occurrence digest and unique. Empty
remaining/proof-map roots are SHA-256 of their respective empty canonical wire.
A completed subtree is constructible only after every reachable submitted edge
under that earlier sibling has terminated or been validly sleep-blocked and its
terminal-set root is closed. `subtreeRoot` is SHA-256 of
`["sleep-subtree-root/v1",rootNodeDigest,[edgeDigest...]]` in recorded DFS
preorder; `terminalSetRoot` is SHA-256 of
`["sleep-terminal-set/v1",[terminalBehaviorDigest...]]` in sorted unique order.

Every failed, absent, stale, or ineligible certificate drops `u`; there is no
inherit-by-default rule. Each retained `u` gets this durable oriented wire:

```text
["sleep-propagation-core/v1",parentNodeDigest,takenOccurrenceDigest,
 sleepingOccurrenceDigest,source,sourceProofOrExploredBranchDigest,
 localCertificateDigest,successorStateDigest,childRemainingDigest]
```

`source` is exactly `earlier-sibling` or `prior-sleep`. The former must name a
completed earlier representative subtree at the same parent. The latter must
name the valid parent proof for that exact sleeper. After all propagation cores
are complete, they are sorted by sleeper occurrence and hashed into the child
proof-map root. Only then is the final child constructed:

```text
["sleep-search-node/v1",successorStateDigest,childRemainingDigest,
 childProofMapRoot]
["sleep-search-edge/v1",parentNodeDigest,takenOccurrenceDigest,
 propagationCoreDigests,childNodeDigest]
```

No propagation core names the final child node, so the evidence is an acyclic
DAG. The edge is the sole object that binds the completed propagation map to
the final child. Its propagation list is ordered by sleeper and must equal the
proof-map entries' propagation digests exactly; empty means the canonical empty
proof map.
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

The generator uses eight learnable families. `ci` denotes canonical role `ci`;
these templates are instantiated before presentation renaming:

| Index/name | Ordered motif pair | Latent guard |
| --- | --- | --- |
| 0 `disjoint-adds` | `add c0 +1`; `add c1 +1` | `read-write-disjoint` |
| 1 `bounded-shared-adds` | `add c0 +1`; `add c0 +1` | `primary-same` and `combined-adds-in-bounds` |
| 2 `equal-sets` | `set c0 1`; `set c0 1` | `primary-same` and `argument-equal` |
| 3 `emit-independent-add` | `emit e1`; `add c0 +1` | `read-write-disjoint` |
| 4 `repeated-swap` | `swap c0 c1`; `swap c0 c1` | root |
| 5 `emit-independent-transfer` | `emit e2`; `transfer c0 c1 1` | `read-write-disjoint` |
| 6 `identical-emits` | `emit e0`; `emit e0` | `symbol-equal` |
| 7 `embedded-bounded-adds` | family 1 motif at non-root history depth | family 1 guard |

Every family's finite training pool includes candidates from alias collisions,
bound failures, different assignments, competing transfers, event-symbol order,
one-way enabling, and destructive conflicts; the exact selected diagnostics are
governed by the closed selection predicates below. Utility worlds rename every cell
and action, vary initial numeric values, reorder declarations, preoccupy Store
names, add irrelevant actions, and place the motif at different history depths.

For each curriculum, acquisition uses the 16 training observations and utility
uses exactly six fresh worlds with six to eight occurrences. They comprise two
worlds in each policy-blind semantic stratum:

- `positive-effect`: at least four reachable, true local commutations have the
  latent training pattern and guard; for overlap families 1, 2, 4, 6, and 7,
  at least four must also fail static read/write independence;
- `neutral`: no reachable nonterminal state has more than one applicable
  occurrence, and no reachable pair has the latent training pattern; and
- `adverse`: at least four reachable true commutations lie outside the latent
  pattern or guard, while no reachable pair matches it.

Those predicates use only independent action semantics and the fixture's
precommitted latent motif. They cannot execute a policy or inspect a learned
artifact, work, search history, certificate count, reduction, or effect size.
The strata deliberately test helpful transfer, no-op transfer, and filtering
whose opportunity cost favors the all-pairs baseline.

Curriculum construction tries attempts `0..31` in order. Each attempt may
consume at most 1,000,000 independent-generator
operations, and the curriculum at most 32,000,000. It is accepted only if all
eight training positives/negatives, alpha-renaming, all six fixed strata slots,
identifiability, state/history ceilings, and evidence preflight pass. The
attempt ledger commits every rejection predicate and work total. No replacement
seed is drawn. Exhausting attempt 31, either work cap, or any stratum slot makes
the panel mechanically invalid before a policy runs.

`Identifiable` has one exact meaning. The family universe contains every
canonical pair with that family's kind/alias pattern and all legal numeric or
symbol arguments, every `0..3` assignment to its roles, and traces of zero
through six `e3` symbols. Truth is actual `commutes` under the complete
admissibility/applicability/pattern/guard predicate. Acquisition has at least
one winner and every tied winner is extensionally equal to that truth table.
For the root family, root is the expected winner.

The complete action alphabet is the canonical-byte-sorted set of every valid
semantic action over `c0,c1,c2`, all arguments in the action table, and emit
symbols `e0,e1,e2,e3`.

The training pool is the family universe plus the lexicographically first
attainable witness for every remaining directional diagnostic from the complete
action alphabet: enumerate state bytes, then canonical pair bytes, both
ascending, and stop a label after its first row. Semantic cores are unique and canonical-byte sorted.
Depth-first selection visits indices in ascending order, explores include
before exclude, and prunes only when a positive/negative count exceeds eight or
remaining rows cannot fill it. The selected result is the first set with eight
positive and eight negative cores, every attainable directional label,
inapplicability, conflict, an event-order conflict, and identifiability. Every
selected core then receives, without affecting support, the two fixed verified
presentation wrappers `[xa,xb,xc]/[aa,ab,ac,ad,ae,af,ag,ah]` and
`[red,green,blue]/[joba,jobb,jobc,jobd,jobe,jobf,jobg,jobh]`.

Family assignment is nonrandom and exact:
`familyIndex = curriculumOrdinal mod 8` and
`withinFamilyOrdinal = floor(curriculumOrdinal/8)`. It yields two curricula per
family in development, three in validation, and four in locked.

### Deterministic draw and decoding schedule

All JSON below is canonical compact UTF-8, all ordinals are zero-based, and all
SHA-256 words are read as unsigned big-endian. Fixture draw `i` is:

```text
F(panel,authority,curriculum,curriculumSeed,attempt,namespace,i) =
 first8(SHA-256(canonical-json(["actionrelation-fixture-draw/v1",
panel,authority,curriculum,curriculumSeed,attempt,namespace,i])))
```

`pick(u,n) = floor(u*n/2^64)`. Utility worlds come from finite skeleton
catalogs, not unconstrained rejection sampling. A positive skeleton for
families 0 through 6 contains two copies of each motif action plus two
permanently inapplicable `check c2 3` occurrences. Family 7 contains
`set c0 0`, four `add c0 +1` occurrences, and one inert check, and requires the
motif absent initially but present after the set. A neutral skeleton is
`claim c0` plus five `check c1 3` occurrences. An adverse skeleton contains
two copies of each alternative pair plus two inert checks; for families 0, 3,
and 5 those checks are replaced by one latent pair required to satisfy raw
pattern/guard but fail initial applicability at every reachable occurrence.
Alternatives by
family index are `repeated-swap`, `equal-sets`, `identical-emits`,
`repeated-swap`, `identical-emits`, `equal-sets`, `equal-sets`, and
`repeated-swap`.

For each skeleton, the catalog enumerates all 64 initial value vectors with an
empty trace, rejects an applicable inert check, canonicalizes and deduplicates
worlds, retains exactly the target semantic stratum, and sorts by world bytes.
An empty catalog invalidates the attempt. Slots 0/1, 2/3, and 4/5 draw without
replacement from the positive, neutral, and adverse catalog respectively:
slot one uses `pick(F(...,"skeleton-variant",slot),n)`; slot two removes the
first selection and uses `pick(...,n-1)`. Thus every world has exactly six
occurrences, and the two members of a stratum are semantically distinct.

The cell-name bank is selected by
`pick(F(...,"cell-name-bank",slot),4)` from
`[[xa,xb,xc],[p,q,r],[red,green,blue],[u,v,w]]`; action banks are
`[aa,ab,ac,ad,ae,af,ag,ah]`,
`[opa,opb,opc,opd,ope,opf,opg,oph]`,
`[stepa,stepb,stepc,stepd,stepe,stepf,stepg,steph]`, and
`[joba,jobb,jobc,jobd,jobe,jobf,jobg,jobh]`, selected by
`pick(F(...,"action-name-bank",slot),4)`. Canonical role `cr` receives cell
name `bank[r]`; canonical occurrences in occurrence-byte order receive action
names in bank order. Cell and action declaration rows are then independently
permuted by descending Fisher-Yates. At step `k=n-1..1`, swap `k` with
`pick(F(...,"cell-permutation" or "action-permutation",8*slot+k),k+1)`.
Store preoccupations are the first
`pick(F(...,"store-preoccupation-count",slot),5)` names from the fixed list
`AR.Candidate`, `AR.Edge`, `AR.Observation`, `AR.Relation`. There are no other
generator draws. The attempt ledger records every `F` preimage and result.

Generator work is a separate unit ledger. Exactly one unit is charged for each
`F` draw, tested world-core bijection, classified state/action pair, evaluated
guard/candidate pair during identifiability, visited training-DFS node, tested
skeleton initial vector, dequeued reachable world state, classified reachable
ordered pair, evaluated stratum predicate, sealed-truth row, and evidence-
preflight component, and named acceptance/bound predicate. Hashing, canonical serialization, sorting, and deduplication
are zero-unit infrastructure and may not perform semantic tests. Attempt phases
are, in order: build/classify family universe; establish identifiability; select
training cores/views; build all three skeleton catalogs; draw two worlds per
catalog and presentation data; enumerate/seal truth in slot order; evidence
preflight. The next unit is reserved before each event. A failed predicate or
reservation ends that attempt immediately; later phases perform no work. These
are the only generator charges and rejection points.

Development authority is literal `development-public-v1` and seed is
`851001+curriculum`; validation uses `validation-public-v1` and
`852001+curriculum`. Locked authority is the lowercase hex attempt-root
commitment, and its curriculum seed is lowercase hex
`HMAC-SHA256(root, canonical-json(["actionrelation-locked-curriculum/v1",
curriculum]))`. These are the only interpretations of `authority` and
`curriculumSeed` in `F`.

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
in the fixture root before acquisition. A generator/supervisor process writes
the scorer pack, closes it, and launches the policy runner with only a read-only
public-view descriptor; the scorer path, descriptor, and capability are absent
from the runner environment and process. After every runner exits, the
supervisor reopens the scorer pack and validation compares each explored terminal set to this prior
commitment. It replays only the submitted certificates, proof chains, and
representative subtrees for omissions. Complete exploration is permitted while
building the sealed fixture and in competence tests, never as a post-policy
utility audit or hidden reducer.

Development and validation generators use public frozen seed ranges. Locked
generation uses a receipt-committed `SHA-256(root)` of a random 32-byte root and
domain-separated HMAC seeds, erases the root before any policy runs, and reveals
only its commitment. Rejected attempts remain within one derived curriculum seed and do
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
An invoked primitive's semantic failure or row rejection costs the same as
success. A cache hit costs counter 11 and cannot erase construction cost. The
last lifecycle unit is permanently reserved for code 19. A compound task
publishes its complete ordered primitive reservation first; if it would use
that terminal unit or cross another cap, no primitive executes, an uncharged
infrastructure reservation-rejection row is written, and code 19 consumes the
reserved unit as `budget-exhausted`. The verifier derives each
counter from operation codes, rejects caller-supplied deltas, and proves
`total = sum(vector)`.

Each policy/curriculum has a 2,000,000-unit lifecycle cap, 65,536 complete-
history cap. Physical
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

There are four pack classes. Object packs start with `AROP1\n`, followed by
`uint32-big-endian length || canonical-record-bytes`; records are unique and
sorted by `SHA-256(recordBytes)`. Operation-journal packs start with `ARJR1\n`
and contain contiguous sequence-order records. Every journal record is exactly
128 bytes and contains, in order, version `uint8`, phase `uint8`, operation
`uint8`, status `uint8`, counter `uint8`, three zero bytes (8 total), sequence
`uint32` (4), run ID (16), previous-record digest (32),
ordered-input-vector root (32), ordered-output-vector root (32), and reserved
zero bytes (4). Its call ID is `SHA-256(record)`. Sequence starts at zero and
`previous` is zero only there; every subsequent record links its predecessor.
Input and output vector roots are recomputed from the ordered digest slices in
the aligned call detail; those digests point to individually canonical evidence
leaves. Vector wrappers are not separately stored objects.
Version is 1; phases are acquisition 1 and utility 2; operation codes are the
numeric codes frozen in the ledger table; statuses
are success 1, semantic-failure-or-row-rejection 2, and cache-hit 3;
counter is 1 through 12. All integers are unsigned and big-endian where wider
than one byte. Compound reservations are canonical input objects rather than
mutable journal header fields.

Call-detail packs start with `ARCD1\n` and have one exactly 320-byte record per
journal sequence. Its frozen offsets are: call ID `[0,32)`, kind `uint16`
`[32,34)`, phase/operation/status/counter bytes `[34,38)`, sequence `uint32`
`[38,42)`, source-task digest `[42,74)`, input count `uint8` at 74, output count
`uint8` at 75, zero bytes `[76,96)`,
six ordered 32-byte evidence digests `[96,288)`, and zero bytes `[288,320)`.
Counts sum to `0..6`; inputs precede outputs and unused digest slots are zero.
The journal roots are SHA-256 of
`["actionrelation-input-vector/v1",[digest...]]` and the corresponding output
wire over these slices.
Search nodes, edges, eligibility rows, certificate attempts, propagation cores,
proof-map entries, terminal behaviors, and reservations use separate kind codes
and therefore remain distinct logical leaves. A shard manifest fixes its
inclusive sequence range. Journal and detail offsets are derived from fixed
record sizes and sequence ranges; neither has a per-record index.

Acquisition-table packs start with `ARTB1\n`. Each manifest fixes one row kind,
record size, count, inclusive ordinal range, and Merkle root over
`SHA-256(["actionrelation-table-leaf/v1",kind,ordinal,recordBytes])` in ordinal
order, duplicating the final node at an odd level. The fixed tables are 13,920
signed-literal rows at 128 bytes, 7,216 guard/observation results at 96 bytes,
451 candidates at 128 bytes, 450 refinement edges at 96 bytes, 16 observation
cores at 512 bytes, 32 view-evidence rows at 512 bytes, and at most 144 training
transition/applicability/comparison rows at 256 bytes. A detail references the
leaf digest. The verifier locates it by kind/ordinal and recomputes the table
root, preserving individual verification. These rows are the exact CUE Store
boundary; they are not duplicated in object packs or object indexes.

An object-index row is exactly 96 bytes: digest (32), offset `uint64` (8),
length `uint32` (4), kind `uint16` (2), pack ordinal `uint16` (2), and 48 zero
bytes. Rows are digest ordered. A shard has at most 4,096 rows and 1 MiB. A
small object is at most 1,024 bytes; a large object is at most 65,536 bytes.
Any pack is at most 16 MiB; deterministic split points are named in a root
manifest. The verifier checks headers, lengths, digests, canonical decoders,
complete index coverage, object uniqueness, journal and detail sequence
alignment, journal hash-chain continuity, zero padding, absence of trailing
bytes, and root closure.

The evidence payload lists only regular committed pack and index files plus the
fixture root, execution manifests, review authority, competence root, and
report prerequisites. Logical leaves remain individually addressable and
digest-checked through pack indexes or fixed-table manifests. The graph does not enumerate hundreds of
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

Before any panel attempt, algebraic preflight uses this exhaustive per-
curriculum capacity table:

| Class | Maximum | Bytes each | Maximum bytes |
| --- | ---: | ---: | ---: |
| operation journal | 60,992 | 128 | 7,806,976 |
| call detail | 60,992 | 320 | 19,517,440 |
| acquisition fixed-row tables | 22,229 | kind-specific | 2,636,864 |
| fixture state/action small object | 512 | 1,024 | 524,288 |
| acquisition small object | 1,024 | 1,024 | 1,048,576 |
| search/cache/proof small object | 2,048 | 1,024 | 2,097,152 |
| terminal/statistics small object | 512 | 1,024 | 524,288 |
| large barrier/artifact object | 32 | 65,536 | 2,097,152 |
| large scorer-truth object | 32 | 65,536 | 2,097,152 |
| object index | 4,160 | 96 | 399,360 |
| all pack headers and manifests | fixed aggregate cap | n/a | 1,048,576 |

The table-row count is
`13,920+7,216+451+450+16+32+144 = 22,229`; its byte total is the sum of
the fixed classes above. The event count is exactly
`24,000 + 128 + 5*4,096 + 2*8,192 = 60,992`. The capacity table totals
39,797,824 bytes, below
the frozen 38 MiB curriculum reservation. The panel additionally reserves 16
MiB: at most 14 MiB for its report and 2 MiB aggregate for receipts and the
publication manifest. Hence the exact
development, validation, and locked caps are respectively 624, 928, and 1,232
MiB as frozen in the manifest. The locked cap remains below
revision 1's bound. Preflight runs before fixture construction or policy work;
a logical record, pack, index shard, curriculum, or panel that cannot fit is
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
panel; `go run`, overlays, build tags, nonempty `GOFLAGS`, any `GOWORK` other
than literal `off`, and compiler inputs outside the manifest are rejected.
Ignored local `go.work` and `go.work.sum` may exist but are recorded as
non-inputs and never read. Runtime environment is reduced to an
allowlist recorded in the execution manifest. Git replacement, alternates,
shallow authority, unsafe config, hooks, in-progress operations, symlinks,
dirty protected files, and untracked or ignored compiler inputs are rejected.

Canonical panel paths are:

```text
.nous/actionrelations-v1-development-report.json
.nous/actionrelations-v1-development-evidence/
.nous/actionrelations-v1-validation-claim.json
.nous/actionrelations-v1-validation-running.json
.nous/actionrelations-v1-validation-terminal-receipt.json
.nous/actionrelations-v1-validation-report.json
.nous/actionrelations-v1-validation-evidence/
.nous/actionrelations-v1-locked-claim.json
.nous/actionrelations-v1-locked-running.json
.nous/actionrelations-v1-locked-terminal-receipt.json
.nous/actionrelations-v1-locked-report.json
.nous/actionrelations-v1-locked-evidence/
```

Canonical top-level wires are closed arrays, not extensible objects:

```text
["actionrelation-fixture-root/v1",panel,authority,curriculumRoots,scorerRoot]
[
 "actionrelation-claim/v1",panel,"claimed",baseCommit,sourceRoot,authority
]
["actionrelation-running/v1",panel,"running",claimReceiptDigest,
 claimCommit,sourceRoot,attemptCommitment,secretLocationDigestOrNull]
["actionrelation-execution-core/v1",panel,authority,sourceRoot,binaryDigest,
 environmentRows,fixtureRoot,runIDs,runningReceiptDigest]
["actionrelation-world-policy-row/v1",panel,curriculum,family,worldOrdinal,
 stratum,worldDigest,policy,searchTerminal,utilityWorkVector,utilityTotal,
 matchCounts,certificateCounts,sleepCount,historyCount,terminalSetDigest,
 behaviorEqual,budgetRemaining,operationRoot]
["actionrelation-curriculum-policy-row/v1",panel,curriculum,family,policy,
 acquisitionTerminal,artifactDigest,acquisitionWorkVector,
 sixOrderedWorldRowDigests,aggregateTerminal,curriculumWorkVector,
 curriculumTotal,behaviorEqual,budgetRemaining,operationRoot]
["actionrelation-evidence-payload/v1",fixtureRoot,executionCoreDigest,
 objectPackRoots,journalPackRoots,detailPackRoots,acquisitionTableRoots,indexRoots,
 worldPolicyRowsRoot,curriculumPolicyRowsRoot]
["actionrelation-report/v1",panel,authority,manifestDigest,reviewDigests,
 buildDigest,fixtureRoot,runningReceiptDigest,curriculumPolicyRowsRoot,
 mechanicalGates,
 primaryRatio,confidenceInterval,randomizationP,savingCoverage,power,
 classification,evidencePayloadDigest]
["actionrelation-terminal-receipt/v1",panel,state,runningReceiptDigest,
 sourceRoot,fixtureRoot,attemptCommitment,reportDigest,evidencePayloadDigest,
 reason]
["actionrelation-publication/v1",planReviewDigest,implementationReviewDigest,
 buildDigest,claimReceiptDigest,runningReceiptDigest,executionCoreDigest,
 evidencePayloadDigest,reportDigest,terminalReceiptDigest]
```

Arrays have exactly the displayed arity and canonical nested row order.
Unknown, missing, duplicated, or reordered fields are invalid. World rows are
ordered by curriculum, world ordinal, then policy; curriculum rows by
curriculum then policy. Authority is the acyclic DAG
`fixture -> execution core -> evidence payload -> report -> terminal receipt ->
publication`; claim/running receipts enter as earlier prerequisites. The payload
contains neither report nor terminal-receipt digests, and the execution core
contains no payload digest. No referenced or unreferenced leaf may be missing
or extra.

For every curriculum-policy pair,
`curriculumWorkVector = acquisitionWorkVector + sum(six utilityWorkVectors)`
componentwise and `curriculumTotal = sum(curriculumWorkVector)`. Acquisition is
charged exactly once. Static/dynamic/complete/lexical acquisition vectors are
zero; Nous and learned-no-use each reference and charge the same shared
acquisition; no-guard references and charges its root-only acquisition. The
aggregate terminal is `completed` when acquisition reaches `not-applicable`,
`completed`, or honest `no-discovery` and all six searches complete; otherwise
it is the first `budget-exhausted` or invalid terminal in acquisition then world
ordinal order. Only curriculum totals enter inference.

Validation and locked use three committed stages. `-stage claim` creates a
`claimed` receipt without seed material and must be committed/pushed on `main`.
`-stage prepare` requires that claim at clean `origin/main`, selects the one
attempt root, writes its immutable commitment in `running`, and must itself be
committed/pushed before any fixture constructor can derive or receive a seed.
For locked, the 32-byte preimage is stored mode `0600` under the resolved Git
common directory and its location string is committed only as a digest.
`-stage execute` requires clean HEAD equal to `origin/main`, the exact running
receipt, a matching preimage where applicable, no terminal receipt, and no
local append-only start marker; it writes that marker before construction.
Every exit produces `published` or `invalid` terminal evidence and never
overwrites it.

This is deliberately a cooperative, root-precommitted, replay-detecting
protocol. It prevents result/seed shopping and ordinary overwrite under
append-only operation, but does not claim repository-enforced one-shot behavior
against deletion of local Git-common state or history rewriting. Development
is repeatable only in a new absent evidence namespace and never overwrites an
existing report.

## Reports, inference, and progression

The world- and curriculum-policy wires above are authoritative; every aggregate
in the report is recomputed from those leaves. Training precision/recall and
per-stratum matched pairs are contained in `matchCounts`; attempted/successful/
cached-success/cached-failure counts are in `certificateCounts`. Bootstrap and
randomization consume exactly the 16, 24, or 32 curriculum-policy totals, never
world rows.

Mechanical validity precedes empirical classification and requires:

- exact manifest, review, fixture, source, pack, acyclic payload, receipt, and
  publication authority;
- primary/audit equality and independent replay;
- production/oracle semantic and guard agreement;
- exact work conservation and budget reservations;
- immutable post-freeze artifacts and delayed scorer access;
- zero false complete relation-instance matches for the Nous artifact over every
  reachable utility state/pair in the panel, including naturally occurring
  inapplicable pattern/guard matches as rejected preconditions;
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

Statistical draw is
`S(namespace,fields...) = bigEndian(first8(SHA-256(canonical-json([
"actionrelation-stat-draw/v1",panel,authority,namespace,[fields...]]))))`.
The only legal namespaces and zero-based field tuples are:

| Namespace | Fields | Range and use |
| --- | --- | --- |
| `bootstrap-family-row` | replicate, family, slot | replicate `0..9999`, family `0..7`, slot `0..m-1`; select one of that family's `m` panel rows |
| `randomization-swap` | replicate, curriculum | replicate `0..9999`, curriculum in report order; low bit chooses swap |
| `power-outer-family-row` | outer, family, slot | outer `0..1999`, family `0..7`, slot `0..3`; select one of two development rows |
| `power-inner-bootstrap-row` | outer, inner, family, slot | inner `0..1999`; select one of four synthetic family rows |
| `power-inner-randomization-swap` | outer, inner, syntheticCurriculum | inner `0..1999`; synthetic ordinal is `8*slot+family` in `0..31`; low bit chooses swap |

Selection uses `pick(S(...),n) = floor(S(...)*n/2^64)`. No stream state,
implicit increment, discarded draw, or other namespace exists. Bootstrap
resamples within each family with the panel's exact `m`: 2 development, 3
validation, or 4 locked curriculum rows. Its statistic is the ratio of summed
curriculum totals. The 10,000 sorted ratios use zero-based indices 249 and
9,749, breaking equal rational values by replicate index.

The paired randomization statistic is
`abs(sum_i(nousWork_i - dynamicWork_i))`. Each replicate independently swaps
the two work values within every curriculum. A replicate is extreme when its
statistic is greater than or equal to the observed statistic, including ties;
`p = (1 + extremeCount) / 10001`.

Development power draws 2,000 synthetic locked panels using exactly the
`power-outer-family-row` table, four development curricula with replacement per
family. Every synthetic panel recomputes all locked predicates using the two
`power-inner-*` namespaces and exactly 2,000 inner bootstrap and randomization
replicates, whose interval indices are 49 and 1,949 and whose p-value
denominator is 2,001. Inherited mechanical
completion, behavior equality, and false-match status are required. Power is
the fraction satisfying every locked positive predicate; progression needs at
least 1,600 successes.
Repeated selections retain distinct `(family,slot)` identities; inner ordering
is slot-major then family, exactly ordinal `8*slot+family`, regardless of equal
underlying curriculum digests.

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
| acquisition | `not-applicable`, `completed`, `no-discovery`, `budget-exhausted` |
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
- presentation views normalize to their semantic core, cannot inflate support,
  and leave relation/artifact bytes invariant; relation-instance matching
  charges and requires both applicability rows and trace context;
- all guard formulae, counts, polarity, normalization, 451 candidates, and 450
  canonical refinement edges are exhaustive and exact;
- observation construction cannot omit, reorder, duplicate, or relabel any of
  the four transition applications;
- relation selection and tied unanimous-use behavior reconstruct from Store;
- alpha-renaming and occupied-name tests preserve semantic artifacts;
- complete and certified sleep search agree on exhaustive tiny universes;
- prior-sleep and earlier-sibling ownership chains, the exact child equation,
  cache success/failure reuse, dropped failed sleepers, and proof-map node
  identities survive adversarial replay; propagation, node, and edge digests
  form the frozen acyclic construction order;
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
  every namespace/index schedule, family/attempt decoding, randomization ties,
  zero denominators, and nested power are golden-tested;
- pack split, index, digest, truncation, duplicate, reordering, oversized-file,
  journal-chain, detail offsets/padding, table ordinal/Merkle proofs, sequence,
  call-range, capacity-preflight, and path traversal attacks fail;
- fixture/scorer access traces prove truth is committed before policy and
  unavailable until termination; utility validation cannot invoke the complete
  enumerator;
- six world rows plus acquisition reconstruct each curriculum row exactly once;
  evidence payload, report, terminal receipt, and publication hashes form an
  acyclic construction order;
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
