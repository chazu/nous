# Kubernetes Selector and Reference Repair Results

## V1: preserved invalid run

V1 is closed. A standing adversarial reviewer invoked the unguarded locked
command while reviewing the uncommitted implementation. The command was:

```sh
mise exec -- go run ./cmd/nous kuberepair-trials -panel locked > /tmp/kuberepair-locked.json
```

The captured JSON was 96,821 bytes with SHA-256
`8a70e230376e104bde09b4655f3552496b25117018dd3ce16d13520384a65ed8`.
It reported `valid-null` and `integrity_valid: true`; those labels are not
accepted as valid evidence because review found that the implementation did not
enforce the preregistered integrity contract. In particular, it used estimated
rather than observed work, consulted oracle minima in the stop rule, did not
independently verify the edit universe, left the unsafe-acceptance flag
unimplemented, omitted required Phase-A panels and component-distance gates,
and exposed the locked panel without a one-shot guard.

The locked v1 summary was:

| Measure | Contextual | Constraint | No credit |
| --- | ---: | ---: | ---: |
| Component tasks solved | 32/32 | 32/32 | 32/32 |
| Mean charged loss | 9,832.25 | 14,503.00 | 5,857.75 |
| Contextual relative effect | - | -32.21% | +67.85% |
| 95% bootstrap interval | - | [-37.89%, -25.08%] | [+34.95%, +114.33%] |

Even under the rejected accounting, contextual credit failed the required
comparison with blind ordering. No v1 population claim is made, and v1 will not
be rerun. Corrections proceed under a separately reviewed v2 contract and new
seeds.

## V2: integrity-valid development stop

V2 replaced the rejected machinery with concrete resource-bound paths, an
independent standard-library-only edit enumerator and oracle, length-stratified
oracle-free policy stopping, evaluator/oracle agreement checks on every tested
state, exact terminal-call accounting, and explicit component strata. The
architecture, semantic, and experiment contracts were accepted independently
before implementation.

The fixed development command was:

```sh
mise exec -- go run ./cmd/nous kuberepair-trials \
  -panel development -domains-dir domains
```

The canonical output captured at `/tmp/kuberepair-v2-development.json` was
99,808 bytes with SHA-256
`57650faac8d5fbe22a018ff0b438d70feec3cba54428603ee603bb17805d5853`.
It reported `valid-null` with `integrity_valid: true`.

Phase A was positive on all six development tasks. Production and oracle edit
universes, terminal classes, minimum lengths, complete syntactic minimum sets,
and deduplicated semantic result sets agreed. The panel included unique,
co-minimal two- and three-edit, already-correct, no-solution, and reference
repair cases at the literal preregistered seeds `761001..761006`. The three
unique one-edit training discoveries created 568 nonempty candidates and made
375 observed terminal calls including their empty-plan checks. The call log's
SHA-256 is
`2571d8b818b56b5008c6c41b6a899646c209380d6992dd25b55e48830d5f7b18`.

The primary preregistered result includes `375 / 32 = 11.71875` acquisition
calls in every contextual component loss:

| Cohort | Contextual | Constraint | No credit | Contextual vs constraint | Contextual vs no credit |
| --- | ---: | ---: | ---: | ---: | ---: |
| All component, 12 | 36.80208 | 25.91667 | 34.91667 | +42.00% | +5.40% |
| Two-feature, 6 | 20.88542 | 9.33333 | 22.00000 | +123.77% | -5.07% |
| Three-feature, 6 | 52.71875 | 42.50000 | 47.83333 | +24.04% | +10.21% |

The development power simulation produced 0 passing panels out of 2,000, for
power `0.0`. The preregistered minimum was `0.80`; validation and locked v2
runs are therefore forbidden and were not opened.

Three independent final adversarial reviews covered architecture and leakage,
semantic and causal validity, and experimental accounting. All accepted the
corrected implementation, canonical artifact, and feasibility stop. The review
round found and closed two material pre-acceptance defects: Phase A originally
derived its seeds instead of using `761001..761006`, and the first credit
validator trusted declared concrete provenance instead of independently
reconstructing it. The final tests pin both corrections.

There is nevertheless a real secondary signal in the frozen learned profile.
If acquisition is excluded and only post-training inference calls are compared,
contextual ordering used fewer calls than blind ordering in both strata:

| Cohort | Contextual inference | Constraint | No credit | Contextual vs constraint | Contextual vs no credit |
| --- | ---: | ---: | ---: | ---: | ---: |
| All component | 25.08333 | 25.91667 | 34.91667 | -3.22% | -28.16% |
| Two-feature | 9.16667 | 9.33333 | 22.00000 | -1.79% | -58.33% |
| Three-feature | 41.00000 | 42.50000 | 47.83333 | -3.53% | -14.29% |

This secondary observation is not a positive v2 result: it excludes the cost of
learning, was not the primary endpoint, and the advantage over the conventional
constraint baseline is too small for the 15% gate even with free training. It
does show that alias-independent structural credit transferred across renamed,
recombined tasks and improved within-length ordering. The failed overall result
is caused by two visible factors rather than evaluator leakage: exhaustive
one-edit acquisition is expensive, and minimum-proof search must evaluate all
shorter plans before learned ordering can help inside the winning length.
