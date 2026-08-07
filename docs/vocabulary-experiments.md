# Vocabulary experiments beyond mathematics

Nous should be judged by whether the EURISKO mechanism transfers between
different representations and evaluators, not by how many units it can create
inside one mathematical ontology. These five deliberately small vocabularies
exercise different kinds of discovery while keeping the world model cheap to
enumerate and the oracle exact enough to test.

## 1. Finite-state protocols

Represent deterministic protocols as states, events, transitions, a start
state, and accepting states. Operations execute traces, remove unreachable
states, locate dead ends, and compare protocols by accepted behavior.

Goals:

- discover rejecting traps and unreachable structure;
- distinguish syntactic difference from behavioral difference;
- propose equivalence and defect conjectures with evidence;
- test whether composition, restriction, specialization, credit, and
  HindSight remain useful outside the mathematical vocabulary; and
- provide an exact, deterministic first transfer experiment.

## 2. Configuration repair

Represent small configurations with six to ten keys, typed values, and a few
cross-field constraints. Seed both valid and invalid configurations. Operations
set, remove, rename, or substitute fields and apply candidate repairs.

Goals:

- discover minimal repairs for invalid configurations;
- assign failures to the transformations that caused them;
- learn repair heuristics that transfer between related schemas;
- separate causal repairs from correlations in the seed examples; and
- test a small, safe version of the original infrastructure-engineering aim.

## 3. String-rewrite grammars

Represent a small alphabet, example strings, and rewrite productions.
Operations apply, compose, restrict, invert, or splice productions. Evaluation
measures which examples can be generated, whether rewriting terminates, and
whether multiple derivations produce ambiguity.

Goals:

- discover recurring transformations and useful production fragments;
- invent composite rewrite operations;
- identify ambiguous, redundant, and non-terminating rules;
- test structure discovery over ordered symbolic data; and
- test whether learned heuristics transfer between two tiny grammars.

## 4. Tiny stack programs

Represent pure stack programs over a very small instruction set, together with
input/output examples. Operations compose, specialize, inline, delete, swap,
or replace instructions. Programs are evaluated in a bounded sandbox.

Goals:

- discover shorter programs that preserve behavior;
- invent reusable combinators and peephole transformations;
- distinguish general improvements from examples-only overfitting;
- exercise Nous metacircularly on programs resembling its own heuristic DSL;
  and
- test whether provenance and failure-driven avoidance improve synthesis.

## 5. Iterated-game strategies

Represent policies for a tiny repeated game, such as iterated prisoner's
dilemma or repeated rock-paper-scissors. A policy maps bounded recent history
to an action. Operations mutate, combine, restrict, and generalize policies;
tournaments provide the evaluator.

Goals:

- test credit assignment in an adversarial rather than static environment;
- discover opponent-specific and robust strategies;
- expose overfitting to the current population;
- test whether diversity can remain valuable despite lower immediate worth;
  and
- probe the limits of scalar worth before attempting a consequential domain.

## Experimental standard

Every vocabulary must have deterministic seeds, behavior-level unit tests, and
at least one held-out case. Success means rediscovering a held-out invariant,
repair, equivalence, or strategy and recording why it was proposed. Raw unit
count and final worth are diagnostics, not research results.
