# Domain vocabularies

A domain vocabulary is the experimental boundary of a Nous run. It is a CUE
directory loaded after `domains/common`; no other domain directory is loaded.

## Minimal contract

Each CUE file contributes a top-level `units` list. Every unit needs `name`,
`worth`, and `isA`. Slots are open-ended but should use the canonical slot
units in `domains/common/slots.cue` where one exists.

Operations must be reachable from `Op`, carry `domain` and `range` slots, and
put executable stack code in `defn`. Concrete inputs are units with `data`.
Heuristics must be reachable from `Heuristic` and supply one or more IfParts
and ThenParts programs.

A pack that needs semantic primitives outside the kernel declares one
`Vocabulary` instance with a registered `dslExtension`. Words selected this
way exist only in that run and its child interpreters; an unselected pack
cannot call them. Empty, duplicate, and unknown extension identifiers fail VM
initialization.

Operations receive an `examples` task automatically. A vocabulary can enqueue
other startup work without changing the engine by adding `initialTasks` to a
unit, for example:

```cue
initialTasks: [{priority: 700, slot: "examples", reason: "Seed this search"}]
```

The pack should answer five questions explicitly:

1. What are the objects?
2. Which examples seed exploration?
3. What can be done to them?
4. What observations count as interesting or bad?
5. Which conjectures should the run emit rather than silently accept?

## Reference pack: math

`domains/math` is the EURISKO-parity reference vocabulary. It contains the
mathematical ontology, operations, predicates, examples, and the source-derived
heuristic corpus. Changes to the kernel must keep this pack's integrated
self-modification proof working.

## Experimental pack: buildgraphs

`domains/buildgraphs` represents a graph as a list of canonical edge strings:

```cue
data: ["web>api", "api>core"]
```

It defines union, intersection, subtraction, and equality over graphs plus
heuristics that exercise graph operations and propose extensional-equality
conjectures. It intentionally does not know about repositories, CI providers,
PUDL facts, or Mu DAGs. Those are possible adapters, not ontology.

## Acceptance test for a new pack

A vocabulary is credible when all of these hold:

- it loads with `common` and without `math`;
- its operations execute against concrete examples;
- at least one domain heuristic produces a new unit or conjecture;
- two identical no-mutation runs have identical summaries; and
- discoveries can be inspected without granting authority to change the
  represented world.

## Experimental pack: protocols

`domains/protocols` represents partial deterministic finite automata as
canonical state, event, start, accept, and transition records. A pure Go layer
implements validation, reachability, rejecting-trap analysis, trace acceptance,
and accepted-language equivalence with shortest counterexample traces.

Its control heuristic demonstrates domain integration by reporting rejecting
traps. Its separate discovery heuristic is blinded to operation identities: it
evaluates the Cartesian product of protocol transforms and relations over all
training machines, materializes every observation, adjusts candidate worth
from support and failure, and promotes fully supported schemas. Decoy operations,
opaque-alias tests, an independent exhaustive evaluator, and a held-out corpus
make the result stronger than a scripted named conjecture.

See [the experiment plan](finite-state-protocol-vocabulary-plan.md) and
[the five candidate vocabularies](vocabulary-experiments.md).

## Experimental pack: rewrite

`domains/rewrite` synthesizes executable operations by concatenating the DSL
definitions of every ordered pair of primitive bounded rewrite rules. A second
heuristic evaluates every constructed program against a complete input/output
corpus, retaining linked result, observation, application, and evidence units.

The experiment promotes one exact program, rejects reversed and decoy
compositions, and lets the kernel's ordinary worth-growth mechanism credit its
two component operations. Opaque aliases, an entirely different runtime-built
corpus, collision tests, an independent scanner, primitive deletion, and
held-out execution prevent the result from depending on seed names or a
facade that delegates to the original units.

See [the stabilized rewrite plan](string-rewrite-vocabulary-plan.md).
The [follow-up trial report](rewrite-trials.md) records generated-problem
robustness, the limits of scalar credit, and the improvement from
[contextual exact-decision credit](contextual-credit-mechanics-plan.md) plus a
bounded exploration reserve.

## Experimental pack: configuration repair

`domains/configrepair` represents bounded typed configurations and schemas as
lists of canonical records. Six neutral primitive assignments generate every
valid unordered subset of size one through three. A separate evaluation
heuristic applies all 41 plans to four examples, materializes 164 linked
results and observations, and promotes only the plan that satisfies every
schema, preserves protected fields, and is idempotent.

The seed includes protected-field shortcuts that satisfy all constraints but
destroy operator intent. An independent evaluator, opaque aliases, occupied
name tests, primitive deletion, held-out inputs, malformed and no-solution
controls, and a runtime-built vocabulary with different fields and values test
that the discovery is behavioral. The winning decision assigns ordinary and
contextual credit to the synthesis heuristic and its three repair components.

See [the stabilized configuration-repair plan](configuration-repair-vocabulary-plan.md).

## Experimental pack: tiny stack programs

`domains/tinystack` is the first descriptor-driven bounded program-synthesis
pack. Seven unary instructions generate all 399 ordered programs of length one
through three. Generic CUE heuristics evaluate the complete corpus, retain
underflow and mismatch evidence, select every shortest exact program behind a
structural barrier, and compare all two-instruction programs with primitives
on a bounded partial-function probe set.

An independent test interpreter checks every candidate and simplification
pair. A fully renamed alternate descriptor changes all category, primitive,
example, probe, method, and credit-context identities without changing the
heuristics. Ambiguity, vacuity, malformed-descriptor, no-solution, collision,
held-out, exhaustive, forged-evidence, contextual-credit, and deterministic
store controls bound the result.

See the [cross-experiment synthesis analysis](bounded-program-synthesis-analysis.md)
and [stabilized tiny-stack plan](tiny-stack-vocabulary-plan.md).

The point is not to prove that EURISKO transfers after one small pack. The
point is to make transfer an empirical question with a cheap, repeatable test.
