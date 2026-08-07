# Iterated-game strategies vocabulary plan

## Status

Stabilization draft after the first architecture, game-theory, and experimental
validity reviews. Implementation begins only after all reviewers accept this
contract. The stabilized design is committed before any implementation-backed
trial is run; later fixture changes require a new experiment version and must
preserve the v1 report.

## Research question

Can Nous enumerate and evaluate a complete bounded policy space, retain
inspectable adversarial evidence, and identify nondominated objective
trade-offs that a single tournament score hides?

This is not a "rediscover Tit-for-Tat" demonstration. Named strategies are
opponent fixtures and explanatory labels only. Candidate generation is
exhaustive over semantic policy records. Promotion uses a multi-objective
frontier after every candidate has complete evidence.

The experiment represents a context-dependent, adversarial, vector-valued
quality relation: a policy's metrics depend on a fixed evaluation profile,
self-play, and perturbations. The fixed profile is stationary. Non-stationary
populations and ecological evolution remain later work.

## Scope and non-claims

The first pack models the finitely repeated Prisoner's Dilemma with
deterministic memory-one policies. It excludes probabilistic policies, longer
memory, learning during a match, open-ended mutation, changing populations,
equilibrium computation, and automatic diversity rewards.

Pareto membership means only "not dominated under these four declared
objectives." It is not an independent usefulness criterion. The pack cannot
show that Nous invented game theory, found a universally best policy, learned
diversity preservation, or beat exhaustive search. The complete space has only
32 policies.

Held-out fixture units, labels, evaluation-case units, schedules, and their use
in opponent roles must be absent from the training descriptor and its
attributed artifacts. Semantically identical policies necessarily occur among
the 32 unlabeled candidates; those candidates are not held-out opponent roles.
Candidates may be frozen for reporting only after complete training-state
verification.

## Objects and bounds

### Strategy

A deterministic memory-one policy has five actions:

1. initial action;
2. action after `CC`;
3. action after `CD`;
4. action after `DC`;
5. action after `DD`.

The first letter is the policy's previous realized action and the second is its
opponent's. Each action is `C` or `D`. Canonical records are exactly:

```text
initial:C
after-CC:C
after-CD:D
after-DC:C
after-DD:D
```

The pure layer rejects missing, duplicate, unknown, reordered, noncanonical,
or oversized records. Semantic-code order treats `D` as zero and `C` as one in
the displayed `initial,CC,CD,DC,DD` order. Reports always display the five
actions as well as the integer code. Unit names are allocation identity only.

### Game and simulator

The seed payoff tuple is `(T,R,P,S) = (5,3,1,0)`, constrained by
`T > R > P > S` and `2R > T+S`. From either player's perspective, `DC` receives
temptation, `CC` reward, `DD` punishment, and `CD` sucker payoff.

Round zero uses each role's initial action. For round `r > 0`, if the prior
realized pair from the candidate's perspective is `(a,b)`, the candidate uses
`after-ab` and the opponent uses `after-ba`. Both choose simultaneously.
Scheduled flips are independently applied to the chosen actions, payoffs use
the flipped pair, and neither role observes a flip until that realized pair
becomes the next history.

A match result contains both scores, both cooperation counts, mutual
cooperation count, both realized traces, and round count. Valid simulation is
deterministic and total.

Bounds are fixed: exactly 32 candidates with cap 32; rounds `[1,200]`; payoffs
`[0,100]`; at most 200 unique zero-based flips per role; names at most 256
bytes; and traces at most 200 actions. A role's match score is in `[0,20000]`.
Descriptors contain `[3,16]` cases: `[1,14]` training cases, exactly one self
case, and exactly one perturbation case. Training totals are at most 280,000.
Per experiment there are at most 32 aggregate evidence units, 512 results, 512
observations, one selection-evidence unit, 32 schemas, 32 conjectures, and 1,024
ordered dominance decisions. Alternate descriptors vary training-case count;
the singular self and perturbation aggregation contract remains unchanged.

### Evaluation case

An evaluation case supplies either one fixed opponent policy or `self=true`,
candidate and opponent flip schedules, and exactly one axis:

- `training`: contributes to training total and training worst payoff;
- `self`: supplies self-play candidate payoff; or
- `perturbation`: supplies candidate payoff in one deterministic perturbed-TFT
  match.

V1 contains exactly four training cases, one self case, and one perturbation
case. Every authoritative case has a distinct semantic case digest. Category
injection cannot add cases.

### Experiment descriptor and semantic identity

`IteratedGameExperiment` declares:

- experiment key `game/memory-one-pd/profile-a/v1`;
- comparison method `exhaustive-memory-one-profile/v1`;
- candidate, opponent, case, result, observation, aggregate-evidence,
  selection-evidence, schema, and conjecture categories;
- opponent strategy slot and an explicit ordered `evaluationCases` list;
- payoffs, rounds, exact cardinalities, caps, credit context;
- distinct generation, evaluation, and finalization task slots; and
- strictly descending task priorities.

Every declared category must exist. Category names are pairwise unequal, and no
declared artifact category may transitively be `isA` another declared artifact
category. Membership scans exclude the category unit itself. Listed opponents
and cases are unique exact members of their declared categories and have unique
semantic digests. Changing payoffs, rounds, opponents, flips, training-case
count/order, categories, task slots, or aliases does not require heuristic
edits.

The descriptor's `profileKey` is a SHA-256 digest of versioned canonical JSON
with fixed field names for the experiment key, comparison method, payoff tuple,
rounds, and each ordered case's semantic axis, opponent policy or self marker,
and flip schedules. A case digest uses the same unambiguous canonical JSON
encoding for its semantic fields. Unit names are excluded. Renaming cases and
updating the descriptor preserves the key; semantic changes require a new key
and version.

Every candidate and attributed artifact stores and revalidates the profile key.
Behavior, frontier, and report lists use semantic-code order.

## Preregistered fixtures

The following five-action records are frozen before implementation:

| Fixture | `initial,CC,CD,DC,DD` |
|---|---|
| All-C | `CCCCC` |
| All-D | `DDDDD` |
| Tit-for-Tat | `CCDCD` |
| Alternator | `CDDCC` |
| Pavlov / win-stay-lose-shift | `CCDDC` |
| Repeat-own-action, initially cooperate | `CCCDD` |
| Suspicious Tit-for-Tat | `DCDCD` |

The authoritative training cases, all 60 rounds, are:

| Order | Axis | Opponent | Candidate flips | Opponent flips |
|---:|---|---|---|---|
| 1 | training | All-C | `[]` | `[]` |
| 2 | training | All-D | `[]` | `[]` |
| 3 | training | Tit-for-Tat | `[]` | `[]` |
| 4 | training | Alternator | `[]` | `[]` |
| 5 | self | candidate itself | `[]` | `[]` |
| 6 | perturbation | Tit-for-Tat | `[10]` | `[20]` |

The perturbation axis measures only raw candidate payoff in that declared
match. It is not noise recovery or robustness; defection or exploitation may
produce a high score.

## Candidate generation

`H-EnumerateMemoryOneStrategies` runs once from the descriptor's startup task.
A vocabulary-scoped semantic operation returns all 32 canonical policies in
semantic-code order. The heuristic creates one `GameStrategyCandidate` per
policy with self-contained data, semantic code, decision key, experiment and
profile links, generation provenance, creation worth 500, and one evaluation
task. It records the authoritative candidate list and expected count.

This enumeration is domain semantics, not a generic engine feature. The
independent oracle separately implements bit-table enumeration without calling
production code.

## Evaluation and evidence

`H-EvaluateGameStrategy` runs one candidate against the descriptor's six cases.
It creates exactly one result and observation per pair, records an ordinary
application, and creates exactly one aggregate evidence unit.

Each result stores experiment, profile key, candidate code, semantic case
digest, both scores, cooperation counts, mutual cooperation, traces, and
rounds. Its observation links the candidate, authoritative case, result,
candidate trace, status, and profile. Aggregate evidence stores ordered case,
result, and observation lists; evaluated/invalid counts; training total and
worst score; self score; perturbation score; training cooperation totals;
four-training-axis behavioral signature; and comparison method.

The behavior signature frames each training case's semantic digest, trace
length, and candidate trace in descriptor order. It excludes unit names and
self/perturbation cases.

Valid evaluated candidates remain at worth 500. Raw payoff is never copied into
worth. Invalid or incomplete candidates fall to 300. This prevents scalar
tournament score from becoming the selection rule implicitly.

## Pre-selection integrity barrier

The last completed evaluation schedules one finalization task. A structural
verifier must prove before selection:

- descriptor, ordered corpus, profile key, and category/task boundaries remain
  valid;
- the authoritative case count equals the descriptor-declared count, and every
  descriptor-declared authoritative case has a distinct semantic case digest;
- all 32 policies exist exactly once;
- every candidate has exactly one aggregate evidence unit;
- every candidate/case pair has exactly one result and observation;
- every stored match, metric, trace, signature, semantic key, and link equals
  fresh execution; and
- no extra attributed candidate, evidence, result, or observation exists.

Missing, duplicate, partial, forged, or extra attributed evidence blocks
finalization.

## Selection

Candidate `A` dominates `B` when it is at least as good on all four objectives
and strictly better on at least one:

1. training score total, maximized;
2. training worst score, maximized;
3. self-play score, maximized;
4. perturbed-TFT candidate payoff, maximized.

`H-SelectGameStrategyFrontier` computes all nondominated policies after the
barrier. It separately records scalar leaders with maximum training total. Ties
remain ties; names and agenda order never choose one.

Frontier candidates rise from 500 to 800 and receive one schema and conjecture
linked to their aggregate and global selection evidence. Dominated candidates
remain at 500. No credit is assigned to action bits as learned components.

Global selection evidence stores every candidate, frontier, scalar leaders,
objective names, objective-vector equivalence classes, behavioral classes, and
all ordered pairwise dominance decisions.

## Post-selection integrity

`game-experiment-complete?` recomputes the frontier, scalar leaders, objective
and behavior classes, and dominance matrix from fresh execution. It verifies
the exact selection evidence, schemas, conjectures, worths, links, profile keys,
and absence of extra attributed selection artifacts. CLI trials and acceptance
tests trust this verifier, never a bare `finalizationComplete` flag.

## Allocation and idempotence

Game-specific semantic allocators cover candidates, results, observations,
aggregate evidence, selection evidence, schemas, and conjectures. Every
artifact has `gameExperiment`, `profileKey`, `artifactKind`, and a bounded
semantic key. Match artifacts also identify candidate code and case digest.

An allocator reuses exactly one fully valid artifact with matching attribution
and semantics, suffixes a base occupied by an unrelated unit, and rejects
multiple, partial, or conflicting attributed artifacts. It does not depend on
`create-unit` or `make-protoconjec` reusing occupied names. Repeated valid tasks
leave evidence, worths, and unit counts unchanged.

## Behavioral diagnostics

The experiment records objective-vector classes, four-training-axis behavior
classes, frontier class coverage, and class memberships. It does not reward
novelty or select a representative per class.

Held-out reporting measures how many training classes and candidate pairs split
under held-out traces. A split shows only that the four training opponents hid
a policy branch; it is not proof of diversity preservation.

## Held-out trial and report schema

`game-trials` requires `game-experiment-complete?`, freezes the 32 candidates,
and snapshots the canonical training store. It then evaluates these exact
60-round cases outside the store:

1. Pavlov `CCDDC`, flips `[]/[]`;
2. Repeat-own-action `CCCDD`, flips `[]/[]`;
3. Suspicious Tit-for-Tat `DCDCD`, flips `[]/[]`;
4. Tit-for-Tat `CCDCD`, candidate flips `[1]`, opponent flips `[]`.

The fourth is a held-out evaluation case, not a held-out policy. It is frozen
because it visits branches hidden by the four training opponents.

The JSON report contains version and profile key; completion status; candidate,
case, artifact, frontier, scalar-leader, objective-class, and behavior-class
counts; oracle agreement counts; class-split counts; and a complete 32-policy
table. Each row contains semantic code and actions, all four training
objectives, objective and training behavior classes, scalar/frontier flags, all
four held-out scores, held-out total and worst, and held-out behavior class.

Set intersection and differences are reported for scalar leaders and the
frontier. Unequal sets are not compared only by their best member. There is no
post-hoc transfer threshold. Held-out reporting must leave the canonical
training store byte-identical. Repeated commands must produce byte-identical
JSON.

## Independent oracle

The test/trial oracle may not import or call `internal/vocab/game`, DSL
builtins, or production aggregation/frontier helpers. It independently
implements:

- all 32 bit tables and canonical ordering;
- perspective-correct simultaneous play, flips, and payoff lookup;
- training and held-out aggregation;
- objective vectors and dominance;
- scalar leaders and Pareto frontier; and
- behavior signatures, classes, and splits.

This is an exhaustive correctness oracle, not a competitive baseline or
evidence of search advantage.

## Scoped implementation boundary

`internal/vocab/game` owns bounded parsing, canonicalization, enumeration,
simulation, metric aggregation, signatures, and Pareto comparison without
depending on units, the DSL, or the engine.

The `game` DSL extension provides only adapters required by ordinary CUE
heuristics: strategy enumeration and identity, match execution and accessors,
descriptor/profile validation, dominance, semantic allocation, pre-selection
verification, and post-selection verification. Unselected VMs cannot see these
words; child VMs inherit the selected immutable registry. Malformed inputs
return semantic nil or false and cannot become positive evidence.

The engine, common vocabulary, agenda, mutation machinery, and math pack remain
unchanged.

## Mechanical acceptance

Implementation is accepted only if tests prove:

- all 32 unique canonical policies and semantic codes;
- exact simulator payoffs, perspective, traces, and both-sided flips;
- rejection of malformed policies, payoffs, descriptors, cases, and schedules;
- exact 32-by-6 result and observation materialization with complete links;
- independent oracle agreement for every match, aggregate, objective vector,
  dominance pair, frontier, scalar leader, behavior class, and split;
- opaque opponent/case aliases and a runtime-built alternate descriptor that
  changes payoffs, rounds, case count/order, aliases, and flip schedules without
  heuristic edits;
- case reordering changes the profile key and framed signatures, but leaves
  objective vectors, scalar/frontier membership, and behavioral/objective
  equivalence partitions invariant when compared by semantic code; each
  descriptor order separately produces byte-identical reports;
- objective-axis ablation with exact production/oracle frontier agreement;
- collision safety for every artifact kind, including schemas and conjectures;
- task idempotence;
- pre-selection rejection of missing, duplicate, forged, or extra evidence;
- post-selection rejection of corrupted or extra selection artifacts;
- held-out fixture/case units, labels, schedules, and opponent-role use absent
  from training artifacts;
- a guaranteed synthetic class-split control and the preregistered class split;
- byte-identical training store before and after held-out reporting;
- vocabulary isolation and child-VM inheritance;
- deterministic store snapshots with mutation disabled and enabled;
- `mise exec -- go run ./cmd/nous run -domain games -cycles 500 -no-mutate`
  reaches verified completion with a drained agenda; and
- all existing repository tests and math behavior remain green.

## Preregistered seed observations

These are sensitivity-case regression expectations, not mechanical proof of
usefulness:

- maximum training total `604`;
- scalar leaders `DCCDD`, `DCDDD`, `DDCDD`, and `DDDDD`;
- Pareto frontier size 14: `CCCCC`, `CCCCD`, `CCDCC`, `CCDCD`, `CCDDD`,
  `CDDCD`, `CDDDC`, `DCCCC`, `DCCCD`, `DCCDC`, `DCCDD`, `DCDCC`, `DCDDC`,
  and `DDDCC`;
- only `DCCDD` among scalar leaders is nondominated: the four tie on
  `(training total, training worst, self) = (604,60,60)`, while perturbation
  scores are respectively `158`, `71`, `73`, and `71` in displayed order;
- 26 four-training-axis behavior classes, with two four-member classes:
  `CCCCC/CCCCD/CCCDC/CCCDD` and `DCCDD/DCDDD/DDCDD/DDDDD`;
- 13 frontier training behavior classes; and
- the held-out flip-at-round-1 TFT case splits each four-member class into
  three held-out behaviors.

The report emits these observations even if implementation contradicts them;
contradictions fail regression and cannot be hidden by retuning v1.

## Interpretation

Passing the experiment means Nous hosted a complete adversarial
multi-objective search and represented why several policies were nondominated
under the declared profile. Advantage over exhaustive search, learned diversity
policy, adaptive population search, open-ended invention, and real-world
utility remain unproven.
