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
