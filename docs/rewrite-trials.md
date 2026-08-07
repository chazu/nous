# Rewrite vocabulary follow-up trials

## Purpose

The seed rewrite experiment proved that Nous can construct an executable
two-operation program and propagate worth-growth credit to its components.
These follow-up trials ask two different questions:

1. Does the same mechanism recover hidden programs across generated problems?
2. Does previously earned component credit make a later bounded search more
   effective?

Run both experiments from the repository root:

```sh
mise exec -- go run ./cmd/nous rewrite-trials \
  -problems 100 -curricula 300 -budget 4 -seed 4242
```

The command emits a deterministic JSON report.

## Generated-problem robustness

Each problem contains four generated primitive rewrite rules over lowercase
ASCII and a randomly selected hidden ordered pair. Rules include shrinking,
deleting, preserving, and expanding replacements. The generator searches a
bounded input space for a corpus that behaviorally distinguishes the hidden
pair from all 11 alternatives, then reserves 32 additional inputs for held-out
execution.

For every problem the trial replaces all seed rules and examples, runs the
actual CUE rewrite heuristics and DSL adapter, and inspects the promoted schema.
Expected values come from the independent scanner in the experiment package,
not from the production rewrite implementation.

Additional regression cases cover an unsatisfiable corpus, an ambiguous corpus
that must retain multiple hypotheses, and unsupported repeated-rule and
three-step targets that must not produce false promotions.

### Observed result: seed 4242

```text
generated problems:       100
hidden pairs recovered:   100
unique promotions:        100
false promotions:           0
held-out executions:     3200
held-out failures:           0
```

Within the declared two-distinct-rule hypothesis class, the vocabulary is not
overfit to the original four rules or examples. This is a robustness result,
not evidence that the current search can express repeated or three-step
programs; the explicit unsupported-target tests correctly return no schema.

## Contextual-credit curriculum

Each curriculum has two phases. Phase one runs the actual engine on a generated
problem. Its unique successful composite grows from worth 500 to 800, causing
the existing kernel to reward both component units from 600 to 750. The
revised kernel also aggregates inspectable `ContextualCredit` units for the
exact semantic decision and each creditor's role. Allocated composite names
do not enter the decision key.

Phase two presents a different, uniquely identifiable corpus over the same
four primitives. The trial compares six candidate orderings:

- **contextual** — the credited exact decision once, followed by seeded
  exploration;
- **scalar** — descending sum of the real component worths left by phase one;
- **scalar reserved** — the greatest scalar-score candidate once, followed by
  the same exploration order;
- **reset** — all component worths reset to 600, with stable identity order;
- **randomized** — a seeded random permutation;
- **exhaustive** — all 12 candidates, used as the solvability oracle.

Every non-exhaustive strategy receives a hard budget of four candidate
evaluations. Evaluation stops on a corpus-satisfying candidate or at the
budget; reports include actual totals and maxima. Curricula are split evenly
among exact reuse of both credited components, reuse of one component, and an
unrelated target sharing neither component. Generation and per-curriculum
policy randomness use independent streams. Contextual, scalar-reserved, and
randomized strategies share the same permutation, making their outcomes
paired comparisons.

The phase-two ranking remains an experimental policy in this harness. The
engine now represents and exposes contextual credit, but its general agenda
does not yet perform this global bounded planning. Role credit is recorded but
not used after only one successful composition.

### Observed result: seed 4242

| Cohort | Contextual | Scalar | Scalar reserved | Random | Contextual mean evaluations |
|---|---:|---:|---:|---:|---:|
| Reuse both | 100/100 | 100/100 | 53/100 | 27/100 | 1.00 |
| Reuse one | 28/100 | 20/100 | 29/100 | 35/100 | 3.74 |
| Unrelated | 32/100 | 0/100 | 32/100 | 39/100 | 3.56 |
| Overall | 160/300 | 120/300 | 114/300 | 101/300 | 2.77 |

Exhaustive search solved all 300 tasks. Contextual search never exceeded four
evaluations. Against scalar it had 55 paired wins, 15 losses, and 230 ties;
against scalar-reserved it had 47 wins, one loss, and 252 ties. Wrong-context
and absent-context controls matched the shared random order in all 300 tasks.

The preregistered five-seed panel produced:

| Metric | Contextual | Scalar | Scalar reserved | Random |
|---|---:|---:|---:|---:|
| Overall | 782/1500 | 609/1500 | 569/1500 | 527/1500 |
| Reuse both | 500/500 | — | — | — |
| Reuse one | 136/500 | — | — | 171/500 |
| Unrelated | 146/500 | — | — | 189/500 |

Contextual therefore improved overall success by 11.5 percentage points over
scalar and 14.2 points over scalar-reserved. Its unrelated rate was 29.2%, 8.6
points below pure random, within the expected cost of reserving one of four
slots for exploitation.

## Interpretation

The robustness result remains cleanly positive. The revised credit result is
also useful, but deliberately narrow:

- exact episodic credit finds a recurring complete composition in one
  evaluation;
- reserving the remaining budget for exploration removes scalar credit's
  complete starvation of unrelated candidates;
- contextual search improves overall success over both legacy scalar ranking
  and scalar ranking given the same exploration reserve; and
- one-component and unrelated cohorts remain worse than pure random search,
  as expected when one slot is spent exploiting history.

This validates compact exact-decision credit plus bounded exploration in the
controlled rewrite harness. It does not yet validate role-credit transfer or
show that the engine's agenda scheduler has improved. Those require multiple
independent histories and a metacircular bounded-search heuristic.

See [the stabilized mechanics plan](contextual-credit-mechanics-plan.md) for
the representation contract, ablations, and acceptance gates.
