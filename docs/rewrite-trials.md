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

## Credit-guided curriculum

Each curriculum has two phases. Phase one runs the actual engine on a generated
problem. Its unique successful composite grows from worth 500 to 800, causing
the existing kernel to reward both component units from 600 to 750.

Phase two presents a different, uniquely identifiable corpus over the same
four primitives. The trial compares four candidate orderings:

- **learned** — descending sum of the real component worths left by phase one;
- **reset** — all component worths reset to 600, with stable identity order;
- **randomized** — a seeded random permutation;
- **exhaustive** — all 12 candidates, used as the solvability oracle.

The first three receive a budget of four candidate evaluations. Curricula are
split evenly among exact reuse of both credited components, reuse of one
component, and an unrelated target sharing neither component.

The phase-two ranking is an experimental policy in this harness. The current
engine does not yet use component worth to globally prioritize a bounded
composition search. The trial therefore measures whether the recorded credit
would be useful to such a policy, without claiming that the current scheduler
already realizes the benefit.

### Observed result: seed 4242

| Cohort | Learned solved | Reset solved | Random solved | Learned mean evaluations |
|---|---:|---:|---:|---:|
| Reuse both | 100/100 | 31/100 | 36/100 | 1.52 |
| Reuse one | 22/100 | 36/100 | 41/100 | 3.90 |
| Unrelated | 0/100 | 33/100 | 38/100 | 4.00 |
| Overall | 122/300 | 100/300 | 115/300 | 3.14 |

Exhaustive search solved all 300 tasks with a mean of 6.58 evaluations.

## Interpretation

The robustness result is cleanly positive. The credit result is conditional:

- credit is extremely useful when the complete successful composition recurs;
- scalar component credit does not reliably identify a new composition that
  shares only one component; and
- stale credit causes severe negative transfer when the next task is
  unrelated.

The observed overall improvement over reset and randomized ordering is driven
entirely by exact-reuse curricula. This is not evidence for a generally useful
search policy. A subsequent design should make credit contextual to a task,
relation, or component role and should reserve exploration capacity so high
historic worth cannot starve unrelated candidates.

