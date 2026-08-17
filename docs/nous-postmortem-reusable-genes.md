# Nous post-mortem: reusable genes

Status: **project-conclusion note; patterns identified for selective reuse**

Date: 2026-08-12

## Verdict

Nous did not establish a practical destination for a general-purpose
EURISKO-style discovery engine. It accumulated substantial machinery for
representing heuristics, running synthetic reasoning curricula, and auditing
their outputs, but no result shows that the whole system is useful on a natural
workload or worth its implementation and verification cost.

The project should therefore be treated as concluded research, not as a product
awaiting one more vocabulary. Its useful residue is a set of smaller engineering
patterns. Most of them concern how to make a learned artifact inspectable and
how to evaluate it without letting the implementation grade itself. Those
patterns are more portable than the Nous kernel that motivated them.

In this document, a **gene** is the smallest mechanism whose value survives
removal from the Nous ontology, heuristic engine, and experiment identities.
Genes should be copied into a concrete consumer and simplified there. They
should not be extracted into a new general framework in anticipation of a
consumer.

## What the project actually established

The three Part 3 lanes provide the clearest evidence boundary:

| Lane | Result | Durable lesson |
| --- | --- | --- |
| [Constraint and nogood learning](constraint-nogood-learning-vocabulary-results.md) | `valid-null` | A real, replayable learned constraint safely improved its reusable cohort, but acquisition and nonmatching costs made the full lifecycle about 2.81% worse than the baseline. Representational competence is not economic utility. |
| [Transformation-schema induction](transformation-schema-induction-vocabulary-results.md) | development `interim-power-authorized` | Factorized refinement strongly beat bounded PBE on a frozen synthetic development distribution, but validation and locked panels never ran. The evidence system produced about 638,000 files and 2.7 GB, making the audit mechanism itself a progression blocker. |
| [Guarded action relations](guarded-action-relations-vocabulary-results.md) | `invalid` | The semantic competence suite passed 55,805 cases, but the one development attempt terminalized before policy work because the sandbox denied an absolute-path ancestor metadata lookup. Competence did not create an empirical utility result. |

These results do not support an aggregate claim that “Nous works.” They do
support several narrower claims about experimental design, evidence-bearing
artifacts, and the cost of general machinery.

## Gene ranking

| Rank | Gene | Reuse confidence | Best fit |
| ---: | --- | --- | --- |
| 1 | Artifact-first learning contract | High | compilers, synthesis, repair, optimization, agent tools |
| 2 | Independent semantic replay | High | systems whose output must be checked rather than trusted |
| 3 | Lifecycle work accounting | High | bounded search, evaluation harnesses, optimizers |
| 4 | Information-rights separation | High | benchmarks, competitions, sealed or delayed evaluation |
| 5 | Explicit terminal taxonomy and append-only attempts | High, when attempts are consequential | experiments and one-shot evaluations |
| 6 | Deterministic agenda plus inspectable state | Medium | exploratory symbolic systems and workflow schedulers |
| 7 | Capability matrices instead of aggregate scores | Medium-high | broad AI, agent, and research programs |
| 8 | Observation/proposal/execution separation | Medium-high | automation acting on real systems |
| 9 | Credit, HindSight, and heuristic mutation | Low | research prototypes only |

## 1. Artifact-first learning contract

The strongest Nous idea is that “learning” must end in a first-class artifact,
not a report label or a hidden change in control flow. The Part 3 contract
required every artifact to be:

- canonical and inspectable;
- frozen before held-out use;
- executable or otherwise behaviorally meaningful;
- replayable from retained evidence;
- removable or corruptible in controls; and
- causally responsible for a later change in work or behavior.

This makes several common evaluation errors difficult: renaming a cached
answer as a learned rule, recomputing the answer while pretending to load an
artifact, or reporting a candidate that never affects execution.

The useful transplant is a small promotion state machine:

```text
candidate -> supporting evidence -> validation barrier -> frozen artifact
                                                        |
                                                        v
                                             later task consumes artifact
```

The consumer should retain the candidate's provenance and record the exact
later decision that used the promoted artifact. Add deletion and corruption
controls only where they test a plausible shortcut. Do not copy the complete
Nous object vocabulary or invent a universal learned-artifact schema.

Relevant implementations include the schema promotion path in
[`internal/vocab/transformschema`](../internal/vocab/transformschema) and the
guarded-relation acquisition path in
[`internal/actionrelationacquire`](../internal/actionrelationacquire).

## 2. Independent semantic replay

Nous eventually treated a policy transcript as an untrusted claim. A separate
reducer reopened typed evidence, reconstructed operations in order, checked
object and budget conservation, and compared the result with an independently
implemented oracle. This is substantially stronger than checking that two runs
emit the same summary: two copies of the same bug can agree perfectly.

The portable pattern has four parts:

1. the producer emits canonical inputs, operations, outputs, and terminal state;
2. a reducer reconstructs the result without trusting producer counters;
3. an oracle implements the bounded semantics independently; and
4. the scorer consumes the reconstruction, not a worker-authored success bit.

Use this for migration tools, planners, compilers, financial batch jobs, safety
checks, or autonomous agents where a compact proof of what happened is worth
the cost. For ordinary application logic, a typed event log plus a focused
checker is usually enough. The action-relations implementation demonstrates
both the power and the danger of taking closure too far: replay spread across
[`internal/actionrelationexp`](../internal/actionrelationexp), while the
independent bounded semantics lived in
[`internal/actionrelationoracle`](../internal/actionrelationoracle).

## 3. Lifecycle work accounting

The nogood result is probably the project's most useful negative finding. A
learned artifact saved roughly 32% on the cohort where it recurred, yet lost on
the full lifecycle after paying for acquisition and nonmatching lookups.

Any system claiming that learning, caching, indexing, or synthesis makes later
work cheaper should separately account for:

- acquisition or training;
- validation and promotion;
- successful reuse;
- unsuccessful lookup on nonmatching cases;
- storage and loading; and
- terminal or cleanup work.

Report both post-freeze speedup and total lifecycle cost. State the reuse
horizon required to repay acquisition. Never allow post-freeze savings to imply
amortization automatically.

Nous also developed atomic vector reservations: a compound semantic action had
to reserve its entire operation block before beginning, and budget exhaustion
could not leave half an action charged. The compact version is useful for
bounded search and quota systems. It is represented most directly by
[`internal/actionrelationledger`](../internal/actionrelationledger). Most
consumers should use a short enum and a few integer counters, not a hash-linked
wire object per reservation.

## 4. Information-rights separation

The later experiment lanes explicitly separated what fixture generation knew,
what a policy could read, what the scorer knew, and when delayed inputs became
available. This is a useful security model for evaluation:

```text
fixture constructor ----> public policy input ----> frozen artifact
        |                                             |
        +---- sealed truth ----------------------------+----> scorer
```

The policy must not be able to infer private truth through filenames, paths,
sizes, metadata, environment variables, process capabilities, or an imported
helper that already knows the answer. The scorer must reconstruct policy
behavior from retained evidence rather than copying fields from a worker
summary.

This gene transfers to model evaluation, coding-agent benchmarks, hiring
exercises, fuzzing campaigns, and competitions. A practical implementation
should begin with a tiny threat model and an end-to-end access test. The
action-relations attempt showed why: the sandbox design was extensively
reviewed, but its actual absolute-path canonicalization failed on `/Users`
after the irreversible start transition. Exercise every literal path and
metadata operation in the final process topology before arming a one-shot run.

The relevant separation is visible across
[`internal/actionrelationscore`](../internal/actionrelationscore),
[`internal/actionrelationfixture`](../internal/actionrelationfixture), and
[`internal/actionrelationrun`](../internal/actionrelationrun).

## 5. Explicit terminal taxonomy and append-only attempts

Nous distinguished:

- `valid-positive`: mechanically valid evidence passes the frozen claim;
- `valid-null`: mechanically valid evidence does not pass it; and
- `invalid`: the evidence cannot answer the claim because semantics, leakage,
  accounting, provenance, or execution failed.

That distinction prevented two bad conversions: turning an infrastructure
failure into a null scientific result, and turning a disappointing but valid
result into a retryable implementation bug. Persistent start markers and
terminal receipts also made an attempted one-shot evaluation append-only.

This is a good pattern when data collection is costly, panels are protected, or
post-result tuning is a serious threat. Use an attempt identity, a pre-start
state, one durable start transition, and exactly one terminal record. Make
recovery idempotent.

It is excessive for routine CI and local benchmarks. There, preserve the
taxonomy but allow ordinary retries. One-shot machinery increases operational
risk and should be justified by the value of the claim, not by a desire for
ceremony.

## 6. Deterministic agenda plus inspectable state

The original kernel has a coherent small architecture:

```text
cmd/nous run
    |
    v
seed.LoadDomain ----> CUE domain pack ----> unit.Store
                                              ^
                                              |
engine.Engine ----> agenda.Agenda ----> dsl.VM
       |                                      |
       +---------- applics, credit, mutation -+
```

[`cmd/nous`](../cmd/nous) constructs the Store and Agenda, loads
`domains/common` plus one selected vocabulary through
[`internal/seed`](../internal/seed), seeds one task per operation, and calls
[`engine.Run`](../internal/engine/engine.go). The engine first pops the
highest-priority task; with an empty agenda it focuses the highest-worth
unfocused Unit. [`internal/agenda`](../internal/agenda) provides stable
equal-priority order and merges duplicate tasks while preserving reasons.
[`internal/unit`](../internal/unit) holds named, open-slot Units with transitive
`isA` and inverse relationships. [`internal/dsl`](../internal/dsl) executes the
heuristic programs supplied by the selected CUE pack under
[`domains`](../domains).

The reusable gene is not the EURISKO ontology. It is the combination of:

- a deterministic priority worklist;
- explicit reasons attached to proposed work;
- inspectable state rather than closures hidden in a scheduler; and
- a clean boundary between scheduling mechanism and domain rules.

This can be useful in static analysis, migration planning, incident
remediation, and other workflows where many rules propose overlapping work.
Copy the agenda and provenance concepts if a real workload needs them. Do not
start by copying Units, worth, creditors, meta-heuristics, and mutation as a
package deal.

## 7. Capability matrices instead of aggregate scores

Part 3 eventually framed the research program as independent capabilities:
learn from failure, generalize structure, compress order, revise belief, grow a
library, explain, and choose by context. Each row could be demonstrated,
valid-null, invalid, or unattempted; success in one row could not upgrade
another.

That is a better reporting model for broad systems than a single “intelligence,”
“agent quality,” or benchmark aggregate. It forces every capability to name:

- a learned artifact;
- a causal held-out use;
- an independent correctness check;
- a cost model; and
- a status that preserves invalid and unattempted outcomes.

This gene is documentation rather than code. It is especially useful for
roadmaps that otherwise accumulate unrelated demos under one product claim.

## 8. Observation, proposal, and execution are different authorities

The active [architecture](architecture.md) limits Nous to vocabulary Units and
proposals; it does not allow a vocabulary to mutate an external system. The
older [PUDL/Mu design](archive/pudl-mu-vision.md) expressed the broader pattern
as three roles:

```text
observed facts -> exploratory reasoner -> reviewable proposal -> executor
```

This separation remains useful even though the full integration was historical
and is not an active, proven Nous subsystem. Cataloging and observation should
not silently become truth. A speculative reasoner should not receive execution
authority merely because it can propose an action. The executor should accept
a validated plan, not the reasoner's mutable internal state.

This transfers well to infrastructure automation and coding agents. In a new
project, implement the boundary with the owning system's native schemas and
approval model; do not resurrect PUDL, Nous, and Mu solely to obtain the shape.

## 9. Credit, HindSight, and heuristic mutation

The kernel records application histories, adjusts creator worth, retains a
graveyard, synthesizes avoidance rules from failures, and can mutate tokenized
heuristic programs. These mechanisms are intellectually interesting and are
represented in [`internal/engine`](../internal/engine) and
[`internal/mutate`](../internal/mutate).

They are the least portable genes because the project did not show that their
feedback reliably improves an external objective. Credit assignment can reward
proxy behavior; avoidance rules can fossilize accidents; mutation can expand a
search space faster than evaluation can constrain it. Treat them as hypotheses
for a bounded research prototype, not production components. A transplant
needs an independent objective, a mature baseline, counterfactual controls, and
a cheap rollback path.

## The experiment-level caller map

Although each lane acquired its own packages, the recurring execution shape
was:

```text
CLI trial command
    |
    v
run/experiment driver ----> fixture constructor ----> public curriculum
          |                                           |
          |                                           v
          |                              CUE vocabulary + policy runner
          |                                           |
          |                                  frozen learned artifact
          |                                           |
          +----> retained typed evidence <-------------+
          |                    |
          |                    v
          |          independent reducer/oracle
          |                    |
          +----> sealed truth -+----> scorer ----> terminal report
```

For the final action-relations lane, the main callers were:

- `cmd/nous actionrelation-trials` ->
  [`internal/actionrelationrun`](../internal/actionrelationrun) for preparation,
  competence, claim, isolated execution, and attempt lifecycle;
- `actionrelationrun` ->
  [`internal/actionrelationscore`](../internal/actionrelationscore) for the
  public panel, delayed scorer, mechanical gates, and report;
- `actionrelationscore` ->
  [`internal/actionrelationacquire`](../internal/actionrelationacquire) and
  [`internal/actionrelationutility`](../internal/actionrelationutility) for
  acquisition and later search policies;
- the policy path ->
  [`internal/actionrelationexp`](../internal/actionrelationexp) for retained
  objects, transcript replay, source authority, and evidence closure; and
- independent checks ->
  [`internal/actionrelationoracle`](../internal/actionrelationoracle) for a
  separate bounded action and search implementation.

This map is useful mainly as a warning about scale. The boundaries are sound in
principle, but a future project should combine roles until a concrete threat or
failure mode justifies splitting them.

## What should remain buried with Nous

The following should not be carried into another project without a specific,
measured need:

1. **A framework-first general discovery engine.** No external workload pulled
   the full Unit/Agenda/DSL/credit/mutation stack into existence.
2. **A universal open-slot ontology.** It made self-description easy but moved
   many invariants from types into convention and verification.
3. **Synthetic vocabulary proliferation.** A sequence of bounded demos can map
   capabilities, but it does not discover a practical customer or workload.
4. **Evidence completeness at any cost.** Transformation-schema evidence grew
   to 2.7 GB and blocked the next protocol step. Auditability has a budget too.
5. **Bespoke wire kinds for every semantic fact.** Canonical logs and targeted
   proofs usually provide a better cost/assurance trade-off.
6. **One-shot execution before operational rehearsal.** Irreversibility made a
   mundane sandbox path defect scientifically terminal.
7. **Competence as a proxy for usefulness.** Exhaustive tiny-space agreement is
   valuable, but the action-relations lane shows that it is not an empirical
   utility result, and the nogood lane shows that correct learning can still
   lose economically.
8. **Adversarial review without a proportionality gate.** Review found real
   defects, but the assurance burden eventually became larger than the value of
   the synthetic claims it protected.

## Sensible extraction targets

If another project develops an immediate need, the following are small enough
to reimplement locally:

| Candidate | Minimal shape | Extract only when |
| --- | --- | --- |
| Canonical artifact helper | typed decode, canonical encode, domain-separated digest | artifacts cross a trust or persistence boundary |
| Replay log | ordered typed events plus a deterministic reducer | producer summaries are not sufficient evidence |
| Vector budget | reserve-all-or-reject for a compound operation | partial charging would make comparisons unfair |
| Sealed-evaluation harness | public input, delayed truth, frozen output, independent score | policies must not observe answers during execution |
| Attempt receipt | immutable identity, start marker, terminal classification | retrying could bias or overwrite a protected result |

Prefer copying 100-300 lines into the consumer over publishing a Nous-derived
library. Shared infrastructure should follow the second real consumer, not
precede the first.

## Gate for any future revival

Do not restart Nous because another reasoning operation looks interesting.
Restart only if a concrete external problem supplies all of the following:

- a real user or system owner;
- naturally occurring inputs rather than a generator designed for Nous;
- a mature conventional baseline;
- a decision whose value can be measured end to end;
- a deliberately small artifact and evidence budget;
- a full operational smoke test before any irreversible transition; and
- a kill rule that stops the work if the learned artifact cannot plausibly
  repay acquisition and verification.

Under that gate, the right move is probably to transplant one gene into the
external project. Reopening the general Nous roadmap should be the last option,
not the default.
