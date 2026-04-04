# Graph Structures Domain

## Why this fits Mode 1

Graph operations are computable. Graph invariants are measurable. The space of small graphs is finite but combinatorially rich. There are deep known theorems relating invariants to each other and to operations, plus many open conjectures. The system could rediscover classical results and potentially find new relationships on small graphs that generalize.

## Concept space

### Core types

- **Graph** — an undirected graph. Data slot is an adjacency representation.
- **DiGraph** — a directed graph.
- **GraphFamily** — a parameterized class (Complete(n), Cycle(n), Path(n), Petersen, etc.)
- **GraphOp** — an operation that takes one or two graphs and produces a graph.
- **GraphInvariant** — a computable property of a graph (returns a number or boolean).
- **GraphPredicate** — a boolean property (planar?, bipartite?, connected?, etc.)

### Data representation

A graph as a unit:
```
name: "K4"
data: {
    vertices: [0, 1, 2, 3],
    edges: [[0,1], [0,2], [0,3], [1,2], [1,3], [2,3]],
}
isA: ["Graph", "CompleteGraph"]
invariants: {
    vertices: 4,
    edges: 6,
    chromatic: 4,
    clique: 4,
    diameter: 1,
    girth: 3,
}
```

For the DSL, the `data` slot holds the adjacency structure and `invariants` holds computed properties. Operations produce new graphs; invariant-computing functions fill in the invariants slot.

### Operations (on graphs)

**Unary:**
- **Complement** — edges where there weren't, vice versa
- **LineGraph** — vertices are original edges, adjacent if they share an endpoint
- **Subdivision** — insert a vertex in the middle of each edge
- **Square** — connect all vertices at distance ≤ 2
- **Transpose** (digraphs) — reverse all edges

**Binary:**
- **CartesianProduct** (G □ H) — vertex set is V(G)×V(H), edges from both factors
- **TensorProduct** (G × H) — edges only when both factors have edges
- **StrongProduct** (G ⊠ H) — union of cartesian and tensor
- **Join** (G + H) — disjoint union plus all edges between G and H
- **DisjointUnion** (G ⊔ H) — just place side by side

**Constructive:**
- **Complete(n)** — the complete graph on n vertices
- **Cycle(n)** — the cycle on n vertices
- **Path(n)** — the path on n vertices
- **Bipartite(m, n)** — the complete bipartite graph
- **Petersen** — the Petersen graph (a famous counterexample factory)

### Invariants (computable properties)

**Numeric:**
- `vertexCount` — |V|
- `edgeCount` — |E|
- `minDegree`, `maxDegree`, `avgDegree`
- `diameter` — longest shortest path
- `girth` — shortest cycle
- `chromaticNumber` — minimum colors for proper coloring (exact for small graphs, expensive)
- `cliqueNumber` — size of largest clique
- `independenceNumber` — size of largest independent set
- `connectivity` — vertex connectivity
- `edgeConnectivity`

**Boolean (predicates):**
- `connected?`
- `bipartite?`
- `planar?` (computable for small graphs)
- `regular?`
- `eulerian?` (all degrees even + connected)
- `hamiltonian?` (NP-complete but tractable for small graphs)
- `tree?` (connected + |E| = |V| - 1)
- `selfComplementary?` (isomorphic to its complement)

### What the system could discover

**Classical theorems to rediscover:**
- χ(G) ≥ ω(G) — chromatic number is at least clique number
- A graph is bipartite iff χ(G) = 2 iff it has no odd cycles
- Every planar graph has χ ≤ 4 (can verify on small examples)
- K₅ and K₃,₃ are not planar (and every non-planar graph contains one as a minor — Kuratowski)
- The Petersen graph is not Hamiltonian, is 3-regular, has girth 5
- complement(Cycle(5)) = Cycle(5) — C₅ is self-complementary
- LineGraph(Complete(n)) is strongly regular
- For trees: |E| = |V| - 1, diameter ≤ |V| - 1, χ = 2 (if |V| > 1)
- Handshaking lemma: sum of degrees = 2|E|

**Novel discoveries (on small graphs):**
- Relationships between invariants that hold for all graphs up to size N
- Which operations preserve which predicates (e.g., CartesianProduct preserves bipartiteness)
- Extremal properties: "among connected graphs on 6 vertices, which maximizes independence number?"
- Operation equivalences: is there a simpler way to express Subdivision(Complement(G))?

### DSL builtins needed

- `graph-new` — create from vertex/edge lists
- `graph-complement`, `graph-line`, `graph-subdivision`, `graph-square`
- `graph-cartesian-product`, `graph-tensor-product`, `graph-join`, `graph-disjoint-union`
- `graph-complete`, `graph-cycle`, `graph-path`, `graph-bipartite`
- `graph-vertex-count`, `graph-edge-count`, `graph-degree`, `graph-diameter`
- `graph-chromatic` — exact for small graphs (backtracking)
- `graph-clique` — exact for small graphs
- `graph-connected?`, `graph-bipartite?`, `graph-planar?`, `graph-regular?`
- `graph-isomorphic?` — for small graphs (brute force or nauty-style)
- `graph-subgraph?` — does G contain H as a subgraph

### Implementation complexity

**Moderate to high.** Most operations are straightforward on small graphs. The expensive ones are chromatic number (backtracking), Hamiltonian path (NP-complete), and isomorphism testing. Keeping graphs small (≤ 10-12 vertices) makes everything tractable.

**Recommended approach:** Start with graphs on ≤ 8 vertices. Use adjacency matrix representation for easy computation. Implement the core invariants (degree sequence, diameter, chromatic number, clique number) and the core operations (complement, line graph, cartesian product). Seed with the named graphs: K₃, K₄, K₅, C₃, C₄, C₅, C₆, P₃, P₄, K₂₃, K₃₃, Petersen.

### Richness of the space

The number of non-isomorphic graphs grows rapidly:
- 3 vertices: 4 graphs
- 4 vertices: 11 graphs
- 5 vertices: 34 graphs
- 6 vertices: 156 graphs
- 7 vertices: 1,044 graphs

Even restricting to ≤ 7 vertices gives over 1,000 objects to explore, each with a dozen computable invariants, and dozens of operations to compose. This is a very rich exploration space.

### Comparison with the math domain

| Aspect | Math (sets) | Graphs |
|--------|------------|--------|
| Objects | Sets of integers | Adjacency structures |
| Operations | Union, intersection, difference | Complement, product, line graph |
| Evaluation | Equality, subset, size | Invariant comparison, predicate checking |
| Composition depth | Moderate (sets of sets are unusual) | High (line graph of complement of product...) |
| Known theorems | Many, elementary | Many, deep and surprising |
| Surprise potential | Moderate | High — graph theory is full of counterexamples |

Graphs are arguably a better EURISKO domain than sets because the space is richer, the theorems are deeper, and there are more operations that compose in non-obvious ways.
