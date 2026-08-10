# Guarded action-relations vocabulary plan

## Status and authority

Status: provisional Part 3 Vocabulary 3 plan, revision 1.

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
  "utility_worlds_per_curriculum": 4,
  "development_seeds": {"start": 851001, "count": 32, "step": 1},
  "validation_seeds": {"start": 852001, "count": 64, "step": 1},
  "locked_curricula": 128,
  "development_power_outer_replicates": 2000,
  "development_power_inner_replicates": 2000,
  "bootstrap_replicates": 10000,
  "randomization_replicates": 10000,
  "minimum_locked_work_reduction": 0.15,
  "minimum_locked_saving_coverage": 0.80,
  "minimum_locked_power": 0.80,
  "alpha": 0.05,
  "maximum_pack_bytes": 16777216,
  "maximum_development_evidence_bytes": 536870912,
  "maximum_validation_evidence_bytes": 1073741824,
  "maximum_locked_evidence_bytes": 2147483648,
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

An action occurrence is the action's semantic bytes plus an occurrence ordinal.
Worlds may contain repeated semantic actions, but occurrence digests are unique.
Canonical action-pair ordering is by semantic action bytes and then occurrence
ordinal; relation artifacts contain neither presentation names nor occurrence
ordinals.

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
mistaken for commutativity. Inapplicability is never positive commutativity.
Asymmetric applicability is labelled `enables` or `disables`; divergent final
states or event traces are labelled `conflicts`. Those labels are diagnostic
negative evidence only.

### Pattern

A relation pattern is the canonical unordered pair of action kinds plus the
alpha-normalized alias topology among `a.x`, `a.y`, `b.x`, and `b.y`. Missing
operands occupy an explicit `none` role. Cell names are replaced by
first-occurrence role integers. Numeric operands, symbols, action names, and
occurrence ordinals are not pattern constants.

### Guard

A guard is a sorted conjunction of zero, one, or two signed literals. It uses
exactly these 15 local atoms:

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

A literal is `[atom,polarity]`. Absent operands make atoms 2 through 15 false;
polarity is then applied normally. A guard cannot repeat an atom, contain both
polarities of one atom, or include literals outside ascending atom order. The
empty conjunction is true. There are exactly
`1 + 2*15 + 4*C(15,2) = 451` normalized guards, below the frozen ceiling.

`combined-adds-in-bounds` is true only for two `add` actions on the same cell
when applying their summed deltas to the current value remains in `0..3` and
each action is initially applicable. It is a local arithmetic fact, not a
diamond result: it does not execute either order, inspect event output, or
authorize pruning.

The guard wire is:

```text
["action-guard/v1", [[atom,polarity]...]]
```

A guarded relation wire is:

```text
["guarded-action-relation/v1","commutes",pattern,guard,
 positiveObservationDigests,negativeObservationDigests]
```

Observation digests are sorted, unique, training-only evidence. A frozen
artifact retains every tied winning relation:

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
edges. Each candidate is evaluated against every acquired observation through
explicit guard matches. A candidate is eligible only if it:

- matches at least four positive observations;
- covers positives from at least two alpha-renamings;
- matches no negative observation; and
- has a complete evidence row for all 16 observations.

Selection first maximizes positive coverage, then minimizes literal count, and
retains every exact tie in canonical byte order. A closed evidence barrier must
name all 451 candidates, every refinement parent, all 7,216 guard evaluations,
statuses, coverage counts, and tied survivors. Freeze then constructs one
artifact from the barrier; no candidate can be evaluated, added, removed, or
reordered afterward.

Terminals are `completed`, `no-discovery`, and `budget-exhausted`. `completed`
requires a nonempty artifact whose training evidence reconstructs exactly.
`no-discovery` requires that no eligible candidate exists after the complete
barrier. Exhaustion requires the next reserved operation to exceed the frozen
budget and may not be used to assert absence.

## Scoped DSL and Store boundary

`domains/actionrelations` is loaded with `domains/common` only and declares the
scoped extension `actionrelations`. The initial word surface is:

- `ar-state-valid?`, `ar-action-valid?`, `ar-action-facts`;
- `ar-applicable?`, `ar-apply`, `ar-state-equal?`;
- `ar-guard-root`, `ar-guard-extend`, `ar-guard-match`;
- `ar-observation-assemble`, `ar-candidate-allocate`;
- `ar-candidate-result`, `ar-close-guard-search`;
- `ar-freeze-relation`, `ar-artifact-match`;
- `ar-certificate-assemble`, `ar-certificate-attach`; and
- `ar-meter`.

Assembly words accept and validate explicitly supplied components; they do not
derive missing states, labels, guards, winners, certificates, sleep sets, or
search decisions. Every semantic action emits a charged operation containing
canonical input/output digests. Scoped words are absent when another domain is
loaded. The base VM, engine, agenda, mutation machinery, common pack, existing
domains, and existing vocabulary semantics remain unchanged.

The Store holds first-class training cases, observations, guard candidates,
refinement edges, counterexamples, the closed barrier, tied relations, and the
frozen artifact. Exact occupied-name collisions are handled with deterministic
content-derived suffixes. Artifact identity is canonical content, never unit
name. Store-byte stability is tested across fresh runs and occupied-name
fixtures.

## Certified sleep-set search

### Local certificate

A certificate is built at one exact search state for two exact remaining
occurrences. It contains:

- state, action-occurrence, relation-artifact, and guard-match digests;
- both initial applicability results;
- both intermediate state digests;
- both crossed applicability results;
- both final state digests;
- the exact-state comparison result;
- the canonical representative occurrence; and
- the charged operation range that produced it.

The wire is:

```text
["local-diamond-certificate/v1",stateDigest,aDigest,bDigest,
 artifactDigest,guardEvidenceDigest,abDigest,baDigest,true,
 representativeDigest,operationRoot]
```

It is valid only if artifact matching succeeded and the four-transition local
commutativity definition independently reconstructs. A false or corrupted
relation can cause a failed certificate attempt but cannot authorize pruning.
No certificate may be reused at a different state, for different occurrences,
or across worlds.

### Search rule

The utility search is a deterministic depth-first sleep-set search over
`(state, sortedRemainingOccurrences, sleepSet)`. Enabled occurrences and loop
order use semantic occurrence digest, not presentation name.

For each enabled occurrence `t` not in the current sleep set, the policy:

1. explores `t` once;
2. forms the child's sleep set from prior sleeping occurrences and earlier
   sibling occurrences only when the policy's eligibility rule proposes the
   pair and a fresh local certificate succeeds at the current state;
3. drops a sleeping occurrence when its fresh propagation certificate fails;
4. recurses on the exact successor with `t` removed; and
5. records but does not score a node at which all enabled actions sleep, because
   its behaviors are represented by an earlier trace-equivalent branch.

Every propagation therefore corresponds to one adjacent swap with an explicit
local diamond. The independent oracle replays the certificate chain that moves
the slept occurrence left to its already explored representative. It also runs
complete exploration and requires exact equality of sorted terminal-behavior
sets. Deduplication by exact `(state,remaining,sleepSet)` is permitted only
after a charged lookup and never substitutes for a certificate.

The learned policy is allowed to omit certificate attempts when its artifact
does not match; this can lose reduction but cannot lose behavior. It may not
prune on relation match alone.

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
   local certificate for every eligible sleep propagation without a learned
   filter.
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
uses four fresh worlds with six to eight occurrences each. The generator
independently enumerates reachable states and complete histories, rejecting any
world beyond the frozen ceilings. Family assignment is balanced by a committed
permutation independent of policy randomness.

Before a policy starts it may see canonical training state/action pairs,
utility initial states and action occurrences, the action semantics, budget,
opaque task tokens, and its own committed randomness. It may not see relation
labels, latent family, latent guard, complete terminal sets, generator accepted
attempt, other policies' traces, or scorer bytes. Training labels become
visible only after the policy pays for and commits the explicit four-transition
observation. Utility terminal truth is decoded only after artifact freeze and
policy termination.

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

One primitive operation increments exactly one counter before execution. Failed
and rejected operations cost the same as successful ones. A cache hit costs a
lookup and cannot erase the original construction, transition, or certificate
cost. Transcript compression, independent post-termination oracle work, and
filesystem hashing are reported separately and cannot create a search
advantage.

Each policy/curriculum has a 2,000,000-unit lifecycle cap, 200,000 transcript
event cap, 65,536 complete-history cap, 20,000 attributed-unit cap, and 10,000
engine-cycle cap. A reservation is made before a compound heuristic action;
partial execution cannot exceed the cap or be reclassified as a cheaper
terminal. Reported totals must reconstruct from event categories, operation
objects, and the terminal row.

Total lifecycle work across acquisition plus all four utility worlds determines
the primary comparison. Post-freeze utility work, histories, transitions,
certificate attempts/successes, guard precision/recall, artifact size, and
amortization crossover are diagnostics only.

## Independent oracle and competence

`internal/actionrelationoracle` reimplements parsing, all eight action kinds,
applicability, transitions, guard atoms, local diamonds, complete exploration,
certified sleep search, terminal behavior normalization, and work reconstruction
without importing production semantics, DSL, fixtures, or experiment helpers.
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

Each pack is a binary sequence with header `ARPK1\n` followed by records
`uint32-big-endian length || canonical-record-bytes`. Records are sorted by
`SHA-256(recordBytes)` and unique. A canonical JSON index contains
`[digest,offset,length,kind]` rows in digest order. The verifier checks the pack
header, every bound, every record digest and canonical decoder, index coverage,
sorting, uniqueness, and absence of trailing bytes. Packs split before 16 MiB;
pack numbering and split points are deterministic. No Git blob may exceed the
pack bound.

The evidence graph lists only regular committed pack and index files plus the
fixture root, execution manifests, review authority, competence root, and
report prerequisites. Logical leaves remain individually addressable and
digest-checked through pack indexes. The graph does not enumerate hundreds of
thousands of filesystem paths. Development, validation, and locked total-byte
caps are enforced before publication.

For each policy/curriculum the evidence includes canonical fixture views,
premanifest, raw event stream, object records, exact Store boundary where
applicable, frozen artifact, certificate records, terminal behavior set, work
vector, and terminal. Primary and audit executions independently reload the
committed fixture bytes and must produce identical semantic rows, transcript
roots, artifacts, behaviors, and work.

Before development, a canonical implementation-review manifest binds the
accepted plan commit, exact implementation commit, ordered review scopes
`architecture`, `semantics`, and `experiment`, and SHA-256 for every tracked Go,
CUE, assembly/C compiler input, module/toolchain file, command, test, plan, and
Part 3 umbrella input. The guard rejects dirty repositories, `go.work`, ignored
or untracked compiler inputs, symlinks, Git replacement/shallow authority,
unsafe local Git configuration, in-progress operations, and changed protected
files.

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

Validation and locked receipts are atomically claimed before their constructors
receive a seed. Any failure finalizes that receipt `invalid`; a published or
invalid receipt cannot transition or retry. Development is repeatable only in a
new absent evidence namespace and never overwrites an existing report.

## Reports, inference, and progression

Each policy row contains panel, curriculum ordinal, family, policy, acquisition
terminal, search terminal, artifact digest, work vector, total work, training
precision/recall, utility matched pairs, certificate attempts/successes,
sleep propagations, explored histories, terminal-set digest, behavior equality,
and budget remainder. Report order is curriculum then frozen policy order.

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
ratio; any omitted locked curriculum makes the locked result `valid-null`, not
a smaller denominator. Statistics use family-stratified paired bootstrap ratios
and a two-sided paired randomization test that swaps the two work totals within
curriculum. Exact PCG streams are SHA-256 domain-separated by panel, authority,
replicate, and purpose. The 95% interval uses sorted zero-based indices 249 and
9749 of 10,000 replicates, with original replicate index breaking exact ties.

Development estimates locked-panel power by drawing 2,000 balanced 128-row
synthetic panels from its family strata and running 2,000 inner replicates.
Progression is authorized only when at least 1,600 outer panels satisfy every
locked positive predicate.

A mechanically valid locked result is `valid-positive` only when:

1. all 128 paired curricula complete with exact terminal behavior;
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
| observation | `commutes`, `enables`, `disables`, `conflicts`, `invalid` |
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
- all guard counts, atoms, polarity, normalization, and refinement edges are
  exhaustive and exact;
- observation construction cannot omit, reorder, duplicate, or relabel any of
  the four transition applications;
- relation selection and tied unanimous-use behavior reconstruct from Store;
- alpha-renaming and occupied-name tests preserve semantic artifacts;
- complete and certified sleep search agree on exhaustive tiny universes;
- local, propagated, stale-state, wrong-occurrence, wrong-artifact, and
  event-hidden certificate forgeries are rejected;
- learned relation deletion removes its causal use; corruption cannot authorize
  a skip; no-guard and learned-no-use controls behave as defined;
- every work counter has success, failure, cap-boundary, and forged-terminal
  tests;
- pack split, index, digest, truncation, duplicate, reordering, oversized-file,
  and path traversal attacks fail;
- fixture/scorer access traces prove truth is unavailable before termination;
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
