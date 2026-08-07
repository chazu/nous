# Reusable bounded-program synthesis

## Why extract this now

The rewrite and configuration-repair vocabularies both succeeded, but their
heuristics are separate CUE programs. A third hand-written enumerator would
show that Nous can host another experiment; it would not show that a useful
discovery method transfers between vocabularies.

The next experiment therefore extracts the smallest common protocol that is
actually supported by those two runs. It deliberately does not move the
protocol into `domains/common`, change the engine, or claim that Nous learned
the abstraction. The tiny-stack pack will be its first declaratively
parameterized implementation. A later pack must reuse the same contract before
we can claim cross-vocabulary heuristic transfer.

## What the previous experiments share

Despite different representations, rewrite and configuration repair follow
the same evidence-producing sequence:

1. A vocabulary declares executable primitive units with structured semantic
   identity and self-contained `defn` bodies.
2. A synthesis heuristic enumerates a bounded, explicit family of component
   combinations.
3. Every combination becomes an ordinary executable operation with components,
   provenance, a semantic decision key, and one evaluation task.
4. A separate heuristic applies every candidate to every declared example.
5. Results, direct applications, per-example observations, and aggregate
   evidence remain inspectable even for rejected candidates.
6. Promotion depends only on complete behavioral evidence, never candidate
   names or enumeration position.
7. Worth growth on the promoted candidate assigns ordinary and contextual
   credit to the synthesis heuristic and its components.
8. Independent oracles, aliases, occupied names, alternate corpora, held-out
   cases, and primitive deletion distinguish an executable discovery from a
   scripted answer.

This is a reusable experimental protocol. The ordering algebra is not:
rewrite uses ordered distinct pairs, while configuration repair uses unordered
distinct-key subsets. Constraint satisfaction and protected intent are also
configuration-specific. Those parts must remain vocabulary semantics.

## Proposed abstraction boundary

A `BoundedProgramSynthesisExperiment` descriptor supplies the generic protocol
with names of categories and slots rather than domain logic:

- a stable experiment key and version;
- primitive, candidate, example, value, result, observation, evidence, and
  promoted-schema categories;
- input and expected-output slot names;
- the result-value slot name;
- input and expected-output validator operation names;
- a binary comparator operation name;
- the descriptor-selected primitive semantic-key slot;
- maximum sequence length and minimum corpus size;
- primitive, example, probe, candidate, and simplification-comparison caps;
- synthesis method and contextual-credit namespace; and
- probe, simplification execution/comparison/evidence/schema categories and
  their bounded comparison cap.

Primitive units supply:

- a bounded semantic identity in the descriptor-selected slot;
- a self-contained unary `defn`; and
- ordinary operation metadata.

The generic mechanism may concatenate definitions, execute candidates with
`apply-op`, compare returned values through the descriptor-selected comparator,
retain evidence, choose the shortest completely supported sequence, and assign
position-sensitive credit.
It must not parse integer stacks, know instruction effects, contain expected
outputs, or name the target opcodes.

The stack vocabulary remains responsible for validating stacks and executing
one opcode. This keeps the reusable layer representation-agnostic while
allowing semantic nil to represent bounded execution failure.

## Three-stage control protocol

The generic flow has an explicit barrier:

```text
synthesize (priority 800)
    -> one evaluation task per candidate (priority 700)
    -> select all co-minimal shortest exact candidates (priority 600)
    -> compare short candidates with primitives (priority 550)
```

Synthesis records the exact candidate count on the descriptor. Selection is
ineligible until every candidate belonging to that descriptor has complete
evidence. This prevents agenda order from making the first exact program win.
All exact programs remain visible; every co-minimal shortest program is
promoted. A tie is a real experimental result and must prevent a unique-winner
claim rather than be broken by name or creation order.

## Identity and credit

Allocated unit names encode the experiment and ordered component identities,
with deterministic collision suffixes. They are provenance, not semantics.
The contextual decision key hashes the synthesis-method version and ordered
descriptor-selected semantic-key sequence. Alias identities therefore produce the same
decision, while reordered instructions do not.

Creditors are the synthesis heuristic followed by component occurrences.
Roles are `synthesis`, `step-1`, `step-2`, and `step-3`. Repeated instructions
are allowed; distinct positional roles keep their contextual attribution
separate. Ordinary scalar credit is also occurrence-weighted, so a primitive
used twice receives two shares. The first stack target uses distinct
instructions; repeated-component behavior is a control, not a claimed
benefit.

Evaluation must not raise any candidate above its creation worth. The final
selection step alone changes the unique target from 500 to 800. Promoted
schemas and simplification schemas set creation and last-rewarded worth to
their final worth, preventing secondary rewards.

## What the stack experiment established

The implemented seed and fully renamed alternate trials establish that this
parameterized protocol can:

- build and execute ordered programs with repeated components;
- retain underflow and mismatch evidence rather than discarding failures;
- delay selection until a complete search matrix exists;
- prefer a shorter exact program over a longer behaviorally exact program;
- discover bounded behavioral simplifications independently of the training
  target; and
- operate under alternate categories, opcodes, examples, and aliases without
  editing its heuristics.

It does not yet establish learned heuristic transfer. Humans extracted the
protocol from prior experiments, and only one new vocabulary will use it.
That stronger claim requires an additional vocabulary to consume the same
descriptor and unmodified heuristic definitions.

## Rejected abstractions

- Moving the heuristics into `domains/common` would make every vocabulary pay
  for an unproven experiment mechanism and blur the restored EURISKO kernel.
- Adding a new engine-level planner would turn a vocabulary experiment into a
  kernel feature and hide the metacircular heuristic behavior we want to
  observe.
- A stack-specific builtin that searches all programs would make the answer a
  Go oracle rather than a Nous discovery.
- Selecting the first exact candidate would make agenda and lexical order part
  of the semantics.
- Generalizing rewrite and configuration repair immediately would disturb two
  accepted baselines before the abstraction has a second consumer.

The bounded descriptor and ordinary CUE heuristics are therefore the smallest
honest next step.
