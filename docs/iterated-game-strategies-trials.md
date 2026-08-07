# Iterated-game strategies trial

## Outcome

The v1 trial passed its preregistered mechanical contract. Nous enumerated all
32 deterministic memory-one policies, evaluated the exact 32-by-6 matrix,
retained complete evidence, and selected a 14-policy Pareto frontier. The
independent oracle agreed on all 32 enumerations, 192 matches, 32 objective
vectors, 1,024 ordered dominance decisions, the frontier, the scalar leaders,
and all 1,024 pairwise behavioral-equivalence decisions.

Run the reproducible report from the repository root:

```sh
mise exec -- go run ./cmd/nous game-trials
```

The ordinary engine path also reaches verified finalization and drains the
agenda after the finalization task:

```sh
mise exec -- go run ./cmd/nous run -domain games -cycles 500 -no-mutate
```

## Observations

- The run created 192 result units, 192 observation units, 32 aggregate
  evidence units, one selection unit, and 14 evidence-linked schema/conjecture
  pairs.
- Four policies tied for maximum training total: `DDDDD`, `DDCDD`, `DCDDD`,
  and `DCCDD`. Only `DCCDD` survived the four-axis Pareto comparison because
  its perturbation score was higher without sacrificing the other tied axes.
- The Pareto frontier retained 13 additional policies that a scalar
  training-total selection would discard. They expose different trade-offs in
  worst-case, self-play, and perturbation behavior.
- The best held-out total was `DCCDC` with 834, while `DCCDD`, the sole policy
  shared by scalar and Pareto selection, scored 719. This is descriptive only;
  held-out scores were not used to choose policies and define no post-hoc
  success threshold.
- Training produced 26 behavioral classes. The held-out suite produced 28 and
  split two training-equivalent classes across ten candidate pairs, showing
  that the training profile hid real policy distinctions.
- Held-out reporting left the canonical training store byte-identical, and
  repeated reports were byte-identical.

## What this establishes

The vocabulary successfully exercises a form of evaluation absent from the
earlier static packs: a candidate's outcome depends on another policy, quality
is vector-valued, and a single scalar leader set loses relevant alternatives.
The experiment also demonstrates that the descriptor, evidence barriers,
semantic allocation, and independent-oracle pattern transfer to this domain.

It does not establish that Nous searches better than enumeration, invents new
game-theoretic concepts, adapts an ecological population, or finds policies
with real-world utility. All 32 candidates are enumerated, the opponent profile
is stationary, and there is no learning, stochastic policy, longer memory,
diversity reward, or component-credit transfer.

## Controls

Tests cover malformed strategies and descriptors, perspective and realized
flips, opaque fixture aliases, case reordering, an alternate runtime-built
profile, semantic-name collisions, task replay, pre- and post-selection
corruption, objective ablation, a synthetic class-split witness, vocabulary
isolation, child-VM inheritance, and deterministic mutation-on/off stores.
