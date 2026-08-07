# Architecture

Nous has one kernel and many vocabularies.

```text
                 domain vocabulary
       concepts + examples + ops + heuristics
                         |
                         v
  CUE loader -> unit store <-> stack interpreter
                    |                 |
                    v                 v
                  agenda       discoveries/applics
                    ^                 |
                    |                 v
              credit, HindSight, mutation
```

## Kernel responsibilities

The kernel owns the mechanisms that make a EURISKO run possible:

- uniformly represented named units with open-ended slots;
- transitive `isA` and inverse-slot maintenance;
- a stable priority agenda with duplicate merging;
- vocabulary-defined startup tasks;
- the two-level task-focus/unit-focus control loop;
- a small stack language and primitive dispatch;
- application records, creditors, worth changes, and graveyard provenance;
- HindSight avoidance heuristics; and
- mutation of heuristic programs.

The kernel does not own a build model, an observation schema, infrastructure
state, or a delivery workflow.

## Vocabulary responsibilities

A vocabulary owns the meanings on which discovery depends:

- ontology and canonical terms;
- examples and counterexamples;
- executable operations and predicates;
- generators or transforms;
- domain heuristics;
- interestingness signals and conjecture kinds; and
- any adapters that translate external state into units or proposals out of
  units.

The math pack is special only because it is the source-parity reference. Its
source-derived H1-H29 family lives in `domains/math`, not in the universal
vocabulary.

## External systems

PUDL and Mu are outside the kernel. A future adapter may snapshot approved
PUDL facts into a domain vocabulary or render Nous conjectures as proposals.
It must not make observation counts into truth, let Nous mutate a Mu execution
plan, or bypass either project's approval and verification boundaries.

This makes the seam intentionally one-way on each side:

```text
world snapshot -> adapter -> vocabulary units -> Nous run -> proposals
```

Applying a proposal remains the responsibility of the owning system or a
human operator.

## Reproducibility

The engine uses a fixed random seed. Unit enumeration, example enumeration,
slot enumeration, and equal-priority agenda order are stable, so a vocabulary
can be regression-tested by comparing complete run summaries. Randomness is a
search choice, not permission for Go map iteration to choose the experiment.
