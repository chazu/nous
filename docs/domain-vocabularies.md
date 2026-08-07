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

The point is not to prove that EURISKO transfers after one small pack. The
point is to make transfer an empirical question with a cheap, repeatable test.
