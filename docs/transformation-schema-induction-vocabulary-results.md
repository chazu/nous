# Transformation-schema induction vocabulary: implementation and trial record

Status: **v2 development classified `interim-power-authorized`; implementation
revision 5 was unanimously accepted, v1 was closed unexecuted, and validation
and locked panels have not run**

This document records Vocabulary 2 of the
[Part 3 vocabulary research program](vocabulary-research-program-v3.md) as it
is implemented, adversarially accepted, and evaluated on its public development
panel. It is an interim empirical result, not a final locked-panel claim.
Validation and locked execution remain forbidden by their later progression
and evidence-commitment gates.

## Frozen identity and goal

- accepted plan: `268b0065f80e86a617fa956e90f228da2d64cfa0` (revision 11);
- vocabulary: `domains/transformschema` and
  `internal/vocab/transformschema`;
- seed authority: `part3/transform-schema/v2`;
- experiment manifest: `transform-schema/v2`;
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
artifact freezes. Prepared fixture reload preserves scorer bytes as an opaque
envelope: it does not decode latent schema, expected outputs, accepted attempt,
seed commitment, or acceptance ledger. Scorer truth is decoded only after every
policy terminal and its eight held-out results are immutable. Construction
acceptance is then recomputed by fixture-generator code and separately audited
by the independent oracle; neither ledger is derived from the other.

## Evidence and independent reconstruction

Every semantic action emits a charged, hash-chained event naming canonical
input, output, and operation objects. The trace now authenticates:

- concrete edit discovery and application;
- complete ascending per-node output comparison for each acquired program,
  followed by orchestration-owned acquire-time serialization and verification
  of the exact four-row batch after the closed acquisition barrier;
- all 13 partial-candidate allocations and 12 refinement-parent edges;
- semantic facts and predicates used to evaluate each factor;
- exactly five canonical closure objects, including every alternative,
  committed factor result, status, parent, and sole survivor;
- one verified frozen schema artifact;
- positive-training applications through immediate evidence attachment and one
  exact comparison per output node, with the final node as the reservation
  endpoint;
- abstaining training applications and all truth-independent held-out
  applications through their immediate evidence endpoint;
- replay applications with the same evidence boundary; and
- a terminal object whose work, application, and sequence totals reconstruct.

The independent reducer validates operation semantics, lifecycle order,
closure graph, artifact immutability, application credits, work conservation,
object completeness, gzip framing, and terminal totals. A separate oracle that
does not import the production vocabulary reconstructs the learned programs,
every frozen-schema or replay application, and the score. Committed replay runs
that oracle again from Git-bound fixture, artifact, program-batch, transcript,
and scorer leaves; fixture acceptance is not a substitute for policy parity.

Factor truth remains heuristic-owned: each CUE factor action emits its own
boolean claim. The reducer reconstructs that claim afterward from the four
acquired programs and explicit local observations. It does not return an
answer to the VM or alter candidate selection. Definition-only scope and guard
normalization uses one enum comparison, explicit `redundant-noncanonical`
closure status, and no example scan. The reducer retains the frozen atom
grammar and binds ordinary ID and ID-set comparisons to exact ordered source
blocks: canonical forest, edited-node observations, prescribed node/parent/
target calls, and final per-row comparison operands. Target observations and
comparisons must interleave per acquired program, so repeated enum atoms from
one case cannot stand in for another. Missing, extra, cross-row, or reordered
factor evidence is rejected.

Committed protected evidence is checked more strongly than the in-memory safe
trial. The verifier rebuilds the exact evidence graph and competence root from
Git blobs, then independently derives each execution row's frozen artifact,
eight held-out outcomes, correctness bit vector, false-application count,
nonmatching work, family assignment, and premanifest fixture bindings. Each
held-out application input digest must match the corresponding committed case,
not merely its position in an eight-row result stream. Report and
execution-manifest agreement alone is not accepted as evidence.

Generator and oracle acceptance diagnostics are independently recomputed from
the already authorized committed fixture leaves and feed the frozen
oracle-parity gate; they add neither a new evidence-graph leaf nor a protected
payload field. Committed verification checks exact application/work counts and
the independently derived matrix roots. Persisted development, validation, and
locked curricula are scrubbed of seed commitment, accepted attempt, latent
schema, expected outputs, generation ledger, and scorer bytes before policy
execution begins.

Store-backed policies persist their actual canonical CUE Store, not a summary.
Independent verification reruns the exact acquisition configuration and
requires byte-identical Store and promoted-program evidence. Bounded and random
PBE remain stateless and are forbidden from emitting Store leaves. Locked
statistics contain no private root or expanded random-pair leaf: the fixture
root instead commits one `transform-statistics-authority/v2` leaf, and replay
derives all 20,000 PCG seed pairs from the receipt's public root commitment.

## Pre-review trials

On 2026-08-08 the safe six-policy trial and all scoped packages passed:

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
receipt and repository attacks, actual Store replay, public locked-statistics
reconstruction, and the source-authority call graph including test files.

The competence suite also passed its frozen exhaustive bounds. A second
implementation in `internal/transformoracle` independently enumerates and
executes the same finite forest, program, and schema universes, and the
committed verifier reruns the full production and oracle paths rather than
trusting the recorded cardinalities:

| Check | Count |
| --- | ---: |
| Canonical forests | 351 |
| Schema applications | 25,272 |
| Concrete-program applications | 7,020 |
| Committed executable microcases | 14 |

The 14 cases now include an actual zero-request forest, execution of ordinary
acquisition with all four destination program-unit names preoccupied, and the
actual bounded-PBE enumeration/retention path cross-checked against an
independent oracle ranking of its minimum-description tier.

## Adversarial implementation review history

Architecture, semantics, and experimental-validity reviewers unanimously
rejected implementation revision 1 at
`77acc437511cc88b291deb3c674d42ccc12d791e`. Their blocking findings were:

- stale v1 profile and held-out-result identities;
- replay without an immediate evidence-link endpoint and a contradictory
  concrete-replay artifact gate;
- scorer and oracle truth decoded before policy execution;
- generator acceptance reconstructed from the oracle instead of independently;
- committed `oracleParity` accepted without rerunning policy and score audits;
- incomplete acquisition, factor-comparison, batch-verification, and baseline
  freeze evidence; and
- competence labels broader than their executable checks.

Architecture, semantics, and experimental-validity reviewers then unanimously
rejected revision 2 at
`55925784755bb186f5b98857e1fabfe52ef87e20`. Their remaining blockers were:

- persisted protected curricula retained truth-bearing generator, latent,
  expected-output, and acceptance fields;
- generator/oracle acceptance diagnostics were not committed evidence, while
  noncompleted Store-backed rows did not receive an independent program audit;
- the competence and minimum-tier checks reused production enumeration or a
  helper rather than the actual bounded-PBE path plus an independent oracle;
- factor evidence counted atom kinds but did not authenticate exact operands,
  order, or case identity, and invalid-operation/verify closure remained loose;
- batch verification was exposed as a whole-experiment CUE capability and was
  not terminal-required; and
- held-out reconstruction trusted positional result assignment without
  authenticating each application input.

Revision 3 at `9d48ec16863266375c0b337b85ba1bde484c2fc4`
fixed the experimental-validity blockers, and architecture accepted it. The
experimental reviewer withdrew two initially reported findings after tracing
the committed full-competence and actual bounded-PBE enumeration calls, but
correctly rejected the revision because it expanded the frozen protected
payload from 14 to 16 fields. Semantics also rejected it because two new scoped
atom kinds were outside the frozen grammar, comparisons were not yet bound to
their source observation blocks, and closure verification was not freeze-phase
bound.

Revision 4 at `47171ba786582addb93d75938b08e497990b7a17`
restored the exact protected and atom wires, bound factor proofs to row
observations, and made closure verification freeze-only. Architecture accepted
revision 3 but had not yet returned on revision 4. Experimental validity found
one remaining exact-plan violation: the acceptance diagnostics had moved from
the payload into two unauthorized evidence-graph leaves. Semantics found that
non-target comparisons could still precede their observations, structural
prefix evidence escaped block checks, normalization could inspect parents or
targets, and factor operations could be relabeled across legal phases.

Revision 5 at `bd374884662db7c0db562b0428ca032e79eded8b` removes the unauthorized leaves while recomputing the same
diagnostics directly from authorized fixtures for the oracle-parity gate. It
also phase-binds every proof operation, rejects all structural normalization
observations and pre-block prefixes, requires each row's typed comparisons to
follow its final prescribed observation, and permits no other factor comparison
besides the one final aggregate claim. Existing regression tests cover
batch-proof omission, unsupported verify wires, malformed-operation semantics,
held-out input rebinding, out-of-grammar atom kinds, detached target
comparisons, and acquire-phase closure forgery. Architecture, semantics, and
experimental-validity reviewers each accepted that exact commit without
qualification. Their canonical authority is recorded in
`docs/transformation-schema-implementation-reviews.json`. No protected command
was run while repairing or reviewing any revision.

The repository-wide suite was also run on this implementation-review
candidate. It completed every other listed package, including
`internal/transformexp`, but remained red because the pre-existing
`internal/causalexpv2.TestDependencyProofPreflightCoversCurrentTrackedTree`
rejects the current causal dependency proof. That failure is outside this
vocabulary and was not changed or waived.

## Protected development result

After all three reviewers accepted exact implementation commit
`bd374884662db7c0db562b0428ca032e79eded8b`, their authority and all 334
protected source hashes were committed at
`3fbbeb6dae1554cc89f28202872ed5611415c30e`. The public development panel then
ran once. Its canonical report has payload digest
`8a70df6e4c528fa50d5e146bdb59e876d5453079aef740461a4044ebc038ab1d`
and classification `interim-power-authorized`.

All 12 mechanical gates passed, the limitations array was empty, primary and
audit executions agreed, and the competence result remained exactly 351
forests, 25,272 schema applications, 7,020 concrete-program applications, and
14 executable microcases.

The preregistered primary comparison was decisive on the 48 development
curricula:

| Measure | Development result |
| --- | ---: |
| Nous perfect held-out curricula | 48 / 48 |
| bounded-PBE perfect held-out curricula | 14 / 48 |
| paired success advantage | 34 / 48 (70.8 percentage points) |
| stratified bootstrap interval | 30 / 48 to 37 / 48 |
| paired-randomization p-value | 1 / 10,001 |
| Nous false applications | 0 |
| Nous nonmatching work | 3,651 |
| bounded-PBE nonmatching work | 409,051 |
| synthetic locked panels passing power gate | 2,000 / 2,000 |

The controls behaved in the expected directions. Positive LGG completed every
curriculum but made one false application in every curriculum and was never
perfect. Concrete replay completed every curriculum but was never perfect on
held-out cases. Removing the equality guard retained a perfect result in 27
curricula and produced `no-discovery` in 21. Random PBE was perfect in 16 and
budget-exhausted in 32; bounded PBE was perfect in 14 and budget-exhausted in
34.

This is good evidence for the intended mechanism on the frozen synthetic
distribution: factorized refinement discovered an executable schema reliably,
generalized beyond memorized examples, abstained without false application,
and used much less nonmatching search work than bounded PBE. It still does not
establish a locked-panel effect or utility on naturally occurring software
transformations.

### Evidence-delivery constraint

The run restored the checkout's ignored `go.work` and `go.work.sum` files
byte-for-byte and did not claim validation or locked receipts. It persisted the
canonical development report and evidence locally under `.nous/`, comprising
638,361 files and approximately 2.7 GB. The single canonical
`evidence-graph.json` is approximately 113 MB.

The v2 guard requires every development evidence leaf and graph to be an
ordinary committed Git blob before validation. Committing and pushing this
bundle is therefore not a routine documentation commit and has not been done
implicitly. Progression now requires an explicit evidence-storage decision:
accept the repository cost and remote constraints for v2, or amend the next
experiment identity to use a scalable content-addressed evidence transport
without weakening committed reconstruction.

## What these trials do and do not show

The protected development result shows that the implemented vocabulary can
recover and execute the expected schema families across its frozen public
distribution and that its report can be reconstructed from semantic evidence
rather than trusted counters. Ablation and control behavior supports—but does
not by itself prove—the causal value of the factorized Store artifacts and
evidence barriers.
They also show expected qualitative separation: concrete replay memorizes
positive cases but cannot generalize, LGG over-applies, and removing the
equality guard prevents safe completion where that guard is required.

They estimate a development effect and authorize progression by the frozen
power rule. They do not establish the final preregistered claim: validation and
locked panels remain unrun, and their evidence prerequisites have not yet been
committed.

## Next protocol step

Choose how to preserve the 2.7 GB development evidence under the guard's
committed-blob contract. Do not run validation until that evidence is durably
committed, the repository is clean, and `verifyCommittedPanel` can reconstruct
the development report from `HEAD`. Do not run locked until a later committed
validation report independently passes its gate.
