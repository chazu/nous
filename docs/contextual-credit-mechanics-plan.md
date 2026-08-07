# Contextual credit mechanics plan

## Status

Implemented and verified after three architecture-review rounds and two
experiment-review rounds. The fixed-seed and five-seed results are recorded in
[the rewrite trial report](rewrite-trials.md).

## Problem

The rewrite curriculum showed that Nous's scalar worth credit has a real but
narrow use. A previously successful ordered pair was found immediately, but
the same component worths suppressed every unrelated pair and did not improve
one-component transfer. With a four-candidate budget, the scalar policy solved
100/100 exact-reuse tasks, 22/100 one-component tasks, and 0/100 unrelated
tasks.

Scalar worth answers the global question "has this unit ever been useful?" It
cannot also answer "was this unit useful in this role, in this kind of
decision?" Nor should a search policy spend its entire budget repeatedly
acting on one old success.

## Goal

Retain the exact-reuse benefit while preventing historic credit from starving
new candidates. Credit must remain represented as units that heuristics can
inspect, remain deterministic, and coexist with EURISKO's global worth rather
than silently changing its meaning.

The revised 300-curriculum trial should:

- preserve 100% exact-pair recovery within a four-candidate budget;
- solve at least 20% of both non-exact cohorts, versus the existing scalar
  policy's 0% on unrelated tasks;
- trail pure random search by no more than 15 percentage points on the
  unrelated cohort, since one of four slots is intentionally spent exploiting
  historic evidence;
- improve overall solved count over both the existing scalar policy and a
  scalar policy given the same exploration reserve by at least 20 tasks on the
  fixed 300-curriculum run;
- never exceed the declared candidate-evaluation budget; and
- keep exhaustive solvability and generated-problem robustness unchanged.

The expected non-exact rate is approximately `3/11 = 27.3%`, while pure random
search receives `4/12 = 33.3%`. One-component transfer is measured, but a
single prior success is not enough to require dominance over random search.
The experiment must report the result without treating component reuse as
known before evaluation.

## Credit representation

Global `worth` and the existing worth-growth reward remain unchanged for
backward compatibility. A useful derived unit may opt into contextual
attribution with three slots:

- `creditContext`: a versioned vocabulary-defined context string;
- `creditDecision`: the stable identity of the complete decision that produced
  the useful unit; and
- `creditRoles`: a list aligned with `creditors`, naming each creditor's role.

When such a unit grows enough that the existing reward is positive
(`delta/2 > 0`), the ordinary reward still changes each creditor's scalar
worth. In the same single-threaded idempotent processing step, the kernel
upserts compact `ContextualCredit` units for:

- `(context, decision, "decision")`, credited by the full worth delta; and
- `(context, creditor, role)`, credited by the creditor's actual scalar-worth
  increase after clamping.

`ContextualCredit` is a common vocabulary category with fixed worth 0 and
`isA: ["Anything"]`. Each credit-record instance has fixed worth 0,
`isA: ["ContextualCredit", "Anything"]`, and string slots
`creditContext`, `creditSubject`, and `creditRole`; integer slots
`rewardTotal`, `evidenceCount`, and `lastRewardTaskNum`; and string slot
`lastSourceUnit`. It has no creditors, creation baseline, or rewarded baseline,
so it cannot recursively generate credit. Heuristics can inspect the records
through `"ContextualCredit" examples` and ordinary slot access; a general
`context-credit` DSL word also returns the exact tuple's `rewardTotal`.

Record identity is SHA-256 over the canonical JSON encoding of the three-string
tuple, using the full lowercase hexadecimal digest after the
`ContextualCredit-` prefix. Lookup scans `ContextualCredit` instances and
verifies the tuple, so it does not depend on suffix continuity. On creation,
if the digest name is occupied by a non-credit or mismatched unit, the writer
first scans and reuses any existing matching tuple; only when none exists does
it use the digest name or first free deterministic `-collision-N` suffix.
Tests force base and intermediate suffix occupants, repeat the upsert, and
delete a suffix to prove aggregation and lookup remain tuple-based.

Aggregation avoids a per-reward event log. It bounds storage by the number of
distinct accepted tuples, not by a global constant: vocabulary-provided tuples
may continue to introduce records. A declaration is accepted only when there
are at most 32 creditors, context and every role are non-empty and 256 bytes or
less, the decision and every creditor identity are non-empty and 512 bytes or
less, and roles align exactly with creditors. Each reward pass therefore upserts at most
`1 + len(creditors)` records. Records retain worth 0 and remain ordinary
inspectable units; they may eventually consume a unit-focus cycle. The rewrite
regression must therefore prove that the same 220-cycle cap still completes
robustness and exhaustive discovery.

If `creditContext` is absent, behavior is exactly the legacy behavior. If a
context is present but the decision is empty or `creditRoles` does not align
with `creditors`, scalar reward still occurs but contextual attribution fails
closed: no partial contextual records are written.

The reward pass is single-threaded, not transactional. It retains the legacy
positive-reward threshold: a one-point rise leaves `lastRewardedWorth`
untouched, so later growth can accumulate into a reward. Within the existing
`delta/2 > 0` branch it validates the whole contextual declaration first,
applies legacy scalar rewards unchanged while recording each existing
creditor's actual post-clamp worth increase, upserts the decision record by the
full accumulated source worth delta and creditor records by those actual
increases, then advances `lastRewardedWorth` once. Upsert is a total in-memory
operation with collision fallback. Missing creditors get no contextual
record; capped creditors record only their actual increase. A second complete
reward pass must leave scalar and contextual state unchanged, and a
500-to-501-to-502 regression must retain the legacy eventual +1 reward.
`lastRewardTaskNum` intentionally means the engine task counter at reward
processing time; it does not claim to identify the earlier task that changed
the source's worth.

The kernel exposes exact lookup by `(context, subject, role)`. The lookup reads
`rewardTotal`; it does not overload the record's scalar worth.

## Rewrite integration

Every synthesized ordered pair declares:

- context `rewrite/ordered-distinct-pairs/v1`;
- a canonical decision identity encoded from the versioned synthesis method
  and the two ordered component identities, independent of its allocated unit
  name; and
- roles `synthesis`, `first`, and `second`, aligned with the synthesis
  heuristic and two primitive creditors.

The scoped rewrite adapter provides the canonical decision-key constructor,
and the experiment derives keys through the same pure rewrite semantic helper.
Collision suffixes and repeated synthesis must leave the key unchanged.

The existing successful-program worth growth therefore leaves both the old
scalar component worths and inspectable contextual records. Seed tests prove
the exact decision record, all three role records, their amounts, tuple
isolation, and one-shot behavior.

## Bounded search policy

The follow-up curriculum adds a `contextual` strategy while retaining the old
scalar strategy as an ablation.

For each phase-two task, one seeded permutation of all 12 candidates is made
and shared by the contextual and randomized strategies. The contextual policy:

1. finds the candidate with the greatest positive exact-decision credit in the
   current context, breaking ties by the shared permutation;
2. evaluates that one candidate first; and
3. if it fails, continues in the shared permutation, excluding candidates
   already evaluated, until the budget is exhausted.

Only one slot is reserved for exploitation. Role credit is recorded and
inspectable but is not used to order the remaining candidates in this trial:
the previous experiment showed that one observation of a component is too
weak to justify starving alternatives. Later trials may add confidence bounds
once multiple independent successes exist.

This is an explicit bounded-search policy in the experiment package, not a
claim that the general engine agenda already performs global candidate
planning. The vocabulary produces the credit; the harness tests the policy
that consumes it. Moving that policy into a metacircular heuristic is a
separate step after its utility is demonstrated.

## Fair comparisons

Every curriculum is generated once and evaluated by all strategies. Reports
include:

- `contextual`: exact contextual exploit plus shared exploration order;
- `scalar`: the existing descending sum of component worths;
- `scalarReserved`: the greatest scalar-score candidate once, with ties broken
  by the shared permutation, followed by that same exploration order;
- `reset`: stable ordering with every component at seed worth;
- `randomized`: the same seeded permutation used for contextual exploration;
- `exhaustive`: all candidates as a solvability oracle.

Candidate evaluations, not primitive calls, remain the cost unit, and every
budgeted strategy receives four. The evaluator stops when the target is found
or the budget is exhausted; it must not scan the remainder and cap the number
afterward. Context lookup and ordering do not execute a candidate. Reports
include actual total and maximum evaluations per strategy, and tests require
the maximum never to exceed the budget. Mean evaluations charges the actual
evaluations, which equals the full budget for an unsolved task.

Benchmark generation and policy ordering use separate RNG streams. Each
curriculum's policy seed is derived solely from the top-level seed and
curriculum index, so changing a strategy cannot perturb later generated
problems. Contextual, scalar-reserved, and randomized strategies share the
same per-task permutation. Reports include paired contextual wins, losses, and
ties against scalar, scalar-reserved, and randomized policies.

The report also includes isolation controls: wrong-context and absent-context
lookup must produce exactly the shared randomized order.

## Tests

Kernel tests cover legacy compatibility, contextual aggregation, idempotence,
role alignment failure, context isolation, deterministic record identity, and
the absence of recursive reward.

Rewrite seed tests cover the emitted declarations and exact accumulated
records, actual-versus-nominal capped reward, missing creditors, semantic keys
under name collisions, and fixed-cycle completion. Experiment tests require
deterministic reports, independent RNG streams, shared permutations, strict
budget accounting, exact-reuse preservation, the numeric gates above, paired
comparisons, isolation controls, and unchanged exhaustive solvability.

The fixed seed 4242 report is the reproducible primary comparison. A declared
five-seed panel (`4242`, `1701`, `8675309`, `271828`, `314159`) checks that the
stochastic non-exact result is not a lucky permutation. Across the 1,500 panel
curricula, contextual must retain 100% exact reuse, solve at least 20% of each
non-exact cohort, remain within 15 percentage points of randomized on
unrelated tasks, and beat scalar and scalar-reserved overall by at least eight
percentage points.

Verification is:

```sh
mise exec -- go test ./...
mise exec -- go test -race ./...
mise exec -- go vet ./...
git diff --check
mise exec -- go run ./cmd/nous rewrite-trials \
  -problems 100 -curricula 300 -budget 4 -seed 4242
# Repeat the curriculum comparison for seeds 1701, 8675309, 271828, and 314159.
```

## Non-goals and limits

- Do not remove or reinterpret scalar worth.
- Do not infer that a failed phase-two candidate invalidates its earlier
  success; failures are task-local evidence.
- Do not smuggle target identity or cohort labels into the context.
- Do not use primitive pre-screening outside the candidate budget.
- Do not claim general component transfer from one rewarded composition.
- Do not move the planner into the agenda until the controlled policy proves
  useful.
- Cross-domain transfer, credit decay, negative contextual credit, and
  confidence-aware role exploitation remain later experiments.

The accepted conclusion, if the gates pass, is deliberately narrow: compact
episodic exact-decision credit plus a bounded exploration reserve improves this
harness. The trial does not establish contextual component-role transfer or an
improvement to the engine's general agenda scheduler.
