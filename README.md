# nous

Nous is a direct, executable port of the core ideas in Doug Lenat's EURISKO:
units and slots, an agenda, source-derived heuristics, credit assignment,
HindSight rules, and heuristic mutation.

This branch deliberately treats that mechanism as a discovery kernel rather
than as a DevOps product. Domains are vocabulary packs. A pack supplies its
own concepts, examples, executable operations and predicates, heuristics, and
interestingness criteria. The engine supplies only representation, scheduling,
interpretation, provenance, credit, and mutation.

The reference `math` pack preserves the EURISKO-parity corpus. The
`buildgraphs` pack is a small independent experiment: build graphs are sets of
`consumer>dependency` edges, with graph operations and graph-specific
heuristics. The `protocols` pack is the first blinded transfer experiment: it
tests every finite-state protocol transform against every candidate relation,
retaining evidence for the schemas it promotes. The `rewrite` pack advances
from selection to construction by synthesizing executable two-step string
transformations, recording contextual credit for successful decisions and
components, and testing that credit under a hard exploration budget. The
`configrepair` pack constructs bounded unordered sets of typed configuration
assignments and selects the unique plan that satisfies two related schemas
without changing protected operator intent. The `games` pack exhaustively
evaluates all deterministic memory-one Prisoner's Dilemma policies, retains
complete match evidence, and promotes a Pareto frontier instead of collapsing
adversarial trade-offs into one worth score.

## Run it

This repository uses the toolchain pinned by `mise`:

```sh
mise exec -- go run ./cmd/nous run -domain math -cycles 300
mise exec -- go run ./cmd/nous run -domain buildgraphs -cycles 100 -no-mutate
mise exec -- go run ./cmd/nous run -domain protocols -cycles 120 -no-mutate
mise exec -- go run ./cmd/nous run -domain rewrite -cycles 220 -no-mutate
mise exec -- go run ./cmd/nous run -domain configrepair -cycles 700 -no-mutate
mise exec -- go run ./cmd/nous run -domain games -cycles 500 -no-mutate
mise exec -- go run ./cmd/nous rewrite-trials -problems 100 -curricula 300 -budget 4
mise exec -- go run ./cmd/nous game-trials
mise exec -- go test -race ./...
```

Run commands from the repository root, or pass `-domains-dir` explicitly.

## Domain packs

Every run loads `domains/common` followed by exactly one named directory under
`domains/`. `common` contains the universal unit/slot vocabulary. Domain packs
do not see one another.

A useful pack normally contains:

- a domain ontology rooted in `Anything`;
- concrete data-bearing example units;
- operations classified under `Op`, usually `UnaryOp` or `BinaryOp`;
- predicates classified under `Pred`;
- stack-language definitions in `defn` slots; and
- heuristics classified under `Heuristic` that decide what is worth trying.

See [Domain vocabularies](docs/domain-vocabularies.md) for the contract and the
current boundary around external systems such as PUDL and Mu.

## Historical material

The parity ledger and source-gap analysis remain in `docs/`. The superseded
PUDL/Mu product vision is retained in `docs/archive/` as history, not as the
active architecture.
