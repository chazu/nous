# Constraint and nogood learning vocabulary: results

Status: **valid-null; stopped before validation by the frozen power gate**

This document records the first empirical result for Vocabulary 1 of the
[Part 3 vocabulary research program](vocabulary-research-program-v3.md). It
interprets the canonical machine-readable evidence without changing the
preregistered endpoint, thresholds, task distribution, cost model, or power
rule.

## Frozen experiment identity

- accepted plan: `23ba4ff097c1c6ded9f488eabf4d96d4eccfbea3`;
- reviewed implementation: `53ffb261073acb72cfb464806ef42a5246f3dfea`;
- experiment: `nogoods/v2`;
- seed authority: `part3/nogoods/v1`;
- cost model: `nogood-lifecycle-events/v2`;
- report schema: `nogood-trials/v2`;
- panel: the public 96-task development panel; and
- canonical report:
  `.nous/nogoods-v2-development-report.json`.

The implementation review authority is recorded in
`docs/nogood-v2-implementation-reviews.json`. The full evidence bundle retains
the generated fixtures, execution manifests, and all 13 policy transcripts for
both the primary and independently decoded audit executions under
`.nous/nogoods-v2-development-transcripts/`.

## Result

The development report is mechanically valid and classified `valid-null`.
All integrity gates passed:

- semantic competence;
- byte-equal primary and audit semantic payloads;
- positional transcript-hash equality;
- transcript conservation;
- independent-oracle parity; and
- soundness of every proposed prune.

The learned generalized nogood was useful on its intended reusable cohort.
`nous-generalized` used 7,000 work units there, compared with 10,290 for the
frozen MAC-CBJ baseline: a saving of 3,290 work units, or about 31.97%.

That local gain did not become a lifecycle gain:

| Quantity | Frozen result |
| --- | ---: |
| Nous search work, all 96 tasks | 27,689 |
| MAC-CBJ work, all 96 tasks | 28,443 |
| Raw search saving before acquisition | 754 |
| Nogood acquisition work | 1,554 |
| Lifecycle excess work, Nous minus MAC-CBJ | 800 |
| Primary effect, excess/baseline work | `800 / 28,443` (about 2.81% worse) |
| 95% stratified bootstrap interval | `792 / 28,745` to `808 / 28,149` |
| Paired randomization p-value | `1,980 / 10,001` (about 0.198) |
| Frozen near-miss plus irrelevant harm | `2,240 / 17,754` (about 12.62%) |
| Synthetic locked panels passing all gates | `0 / 2,000` |

The frozen positive criterion required a negative primary numerator, a wholly
negative interval, p less than 0.05, and no more than 10% harm on the protected
nonreusable cohorts. This result satisfies none of those empirical conditions.
The development-power estimate is therefore unauthorized.

## Protocol consequence

Validation and locked execution were not run. This is mandatory rather than a
discretionary stopping choice: `0 / 2,000` passing power replicates is below the
preregistered 0.80 gate, and the protected entry points reject progression.
Changing the acquisition price, horizon, cohort mix, or harm threshold after
seeing this result would create a different experiment and cannot upgrade this
one.

The retained classification is `valid-null`, not `invalid`. The vocabulary
successfully materialized and safely reused an inspectable learned nogood; the
independent oracle confirms that its prunes preserve the exact solution set.
What failed was the stronger Part 3 claim that the acquired artifact repays its
full cost against a strong solver on this frozen workload.

## What was learned

This trial separates semantic capability from economic utility:

1. Nous can acquire a generalized negative constraint from bounded conflict
   evidence and causally prune alpha-renamed later problems.
2. The promoted artifact is real, replayable, scoped, and safe under the tested
   grammar; it is not merely a copied source constraint or report-only label.
3. Reuse is materially beneficial when the learned shape recurs.
4. The current bridge is too expensive on nonmatching cases, and four training
   examples cost too much for the preregistered 96-task lifecycle.
5. A future experiment may study cheaper indexing, a longer independently
   justified reuse horizon, or a task stream with naturally measured recurrence,
   but it must receive a new identity and be planned without reclassifying this
   result.

For the Part 3 capability matrix, **learn from failure** is semantically
demonstrated while the vocabulary's preregistered lifecycle-utility claim is
`valid-null`.

## Reproduction boundary

The development command was:

```sh
GOWORK=off mise exec -- go run ./cmd/nous nogood-trials \
  -repo-root . -domains-dir domains -panel development
```

Repository-local `go.work` and `go.work.sum` were absent for the execution and
restored afterward. Reproduction must start from the exact reviewed inputs and
must not overwrite the canonical evidence paths. The evidence guard deliberately
requires an exact clean repository, immutable reviewed source inputs, and no Go
workspace override.
