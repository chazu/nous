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
heuristics.

## Run it

This repository uses the toolchain pinned by `mise`:

```sh
mise exec -- go run ./cmd/nous run -domain math -cycles 300
mise exec -- go run ./cmd/nous run -domain buildgraphs -cycles 100 -no-mutate
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
