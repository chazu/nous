# Transformation-schema induction vocabulary: implementation and trial record

Status: **implementation-review candidate; no protected panel has run**

This document records Vocabulary 2 of the
[Part 3 vocabulary research program](vocabulary-research-program-v3.md) as it
is implemented and tested before adversarial implementation acceptance. It is
not an empirical result. Development, validation, and locked execution remain
forbidden until architecture, semantics, and experimental-validity reviewers
accept the same implementation commit and that authority is committed in the
canonical review manifest.

## Frozen identity and goal

- accepted plan: `baff1990798846b9314c9b42745198098c8087f1`;
- vocabulary: `domains/transformschema` and
  `internal/vocab/transformschema`;
- seed authority: `part3/transform-schema/v1`;
- experiment manifest: `transform-schema/v1`;
- learned artifact: an executable five-factor transformation schema;
- primary comparison: ordinary Nous refinement against bounded PBE; and
- protected claim: improved exact held-out transformation success under the
  frozen lifecycle budget, without false application.

The complete preregistration, term language, fixture distribution, work
ledger, evidence protocol, inference rules, and protected-panel authority are
in the [accepted plan](transformation-schema-induction-vocabulary-plan.md).

## Implemented reasoning path

The production vocabulary learns from four positive and four abstention
examples. Ordinary Store/Agenda heuristics acquire four concrete programs,
allocate a root partial, traverse 12 explicit refinement edges across target,
anchor, reference scope, old-value guard, and locality, retain factor evidence,
and close five evidence barriers. Promotion occurs only when each barrier has
one survivor and all eight training cases validate.

The promoted artifact is canonical `transform-schema/v1` JSON and remains
unchanged for all eight delayed held-out applications. The no-equality ablation
uses the same path but makes the equality-dependent alternative ineligible.
The five comparison policies use independent implementations: positive LGG,
bounded PBE, randomized PBE, concrete replay, and the equality ablation.

The policy-visible curriculum contains training bytes, a commitment to delayed
held-out inputs, opaque task tokens, and committed policy randomness. It does
not contain the family, latent schema, accepted generation attempt, held-out
inputs, or scorer truth. Held-out inputs are decoded only after the policy
artifact freezes; scorer bytes are decoded only after held-out execution.

## Evidence and independent reconstruction

Every semantic action emits a charged, hash-chained event naming canonical
input, output, and operation objects. The trace now authenticates:

- concrete edit discovery and application;
- all 13 partial-candidate allocations and 12 refinement-parent edges;
- semantic facts and predicates used to evaluate each factor;
- exactly five canonical closure objects, including every alternative,
  committed factor result, status, parent, and sole survivor;
- one verified frozen schema artifact;
- training and held-out applications with immediate evidence attachment;
- replay applications with the same evidence boundary; and
- a terminal object whose work, application, and sequence totals reconstruct.

The independent reducer validates operation semantics, lifecycle order,
closure graph, artifact immutability, application credits, work conservation,
object completeness, gzip framing, and terminal totals. A separate oracle that
does not import the production vocabulary reconstructs the learned programs,
applications, and score.

Committed protected evidence is checked more strongly than the in-memory safe
trial. The verifier rebuilds the exact evidence graph and competence root from
Git blobs, then independently derives each execution row's frozen artifact,
eight held-out outcomes, correctness bit vector, false-application count,
nonmatching work, family assignment, and premanifest fixture bindings. Report
and execution-manifest agreement alone is not accepted as evidence.

## Pre-review trials

On 2026-08-09 the safe six-policy trial and all scoped packages passed:

```text
GOWORK=off mise exec -- go test \
  ./internal/dsl \
  ./internal/transformbaseline \
  ./internal/transformfixturecore \
  ./internal/transformoracle \
  ./internal/vocab/transformschema \
  ./internal/transformexp -count=1
```

The tests cover all nine generated semantic families, policy/scorer isolation,
purpose-separated fixture streams and golden vectors, deterministic primary
and audit trials, the six policy controls, exact object reduction, forged
closure rejection, committed score/artifact reconstruction for every policy,
receipt and repository attacks, and the source-authority call graph.

The competence suite also passed its frozen exhaustive bounds:

| Check | Count |
| --- | ---: |
| Canonical forests | 351 |
| Schema applications | 25,272 |
| Concrete-program applications | 7,020 |
| Committed executable microcases | 14 |

The repository-wide `GOWORK=off mise exec -- go test ./... -count=1` run
completed every other listed package, including `internal/transformexp`, but
the overall command remains red because the pre-existing
`internal/causalexpv2.TestDependencyProofPreflightCoversCurrentTrackedTree`
rejects the current causal dependency proof. That failure is outside this
vocabulary and was not changed or waived.

## What these trials do and do not show

They show that the implemented vocabulary can recover and execute the expected
schema families on safe generated curricula, that its distinguishing Store
artifacts and evidence barriers are causally necessary, and that its reports
can be reconstructed from semantic evidence rather than trusted counters.
They also show expected qualitative separation: concrete replay memorizes
positive cases but cannot generalize, LGG over-applies, and removing the
equality guard prevents safe completion where that guard is required.

They do not establish the preregistered empirical claim, estimate an effect, or
authorize protected execution. Those conclusions require accepted reviews of
one exact commit followed by the one-shot development panel. If development
power is unauthorized, this lane stops with that result; validation and locked
panels remain unrun.

## Next protocol step

Commit and push this exact candidate, obtain unqualified architecture,
semantics, and experimental-validity acceptance for that commit, and record the
three decisions in `docs/transformation-schema-implementation-reviews.json`.
Only then may the development entry point run once under a clean, reviewed
repository authority.
