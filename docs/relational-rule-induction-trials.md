# Relational rule-induction trial

## Outcome

The locked v1 trial is mechanically valid but empirically `valid-null`. Nous
learned a bounded two-clause Horn definition, froze it as a first-class shared
library relation, reused or rejected it in a second task, and made perfect
held-out predictions. Sharing substantially reduced total charged work and
beneficial-case search relative to direct, task-local, and recomputed controls.
It did not reach the preregistered minimum isolated execution benefit relative
to an equal-accuracy, equal-candidate-schedule inlined control.

This is a successful bounded vocabulary implementation and a valid negative
result for the stronger combined capability claim. It is not `valid-positive`.

## Frozen provenance

- accepted plan: `77b67226fabb61c9295e4cb80c879815e46567ce`;
- implementation candidate: `12806a268b3bdb81544b99f59cfbcc528c8575c2`;
- locked panel: 64 seeds, `41001..41064`, assigned to 40 beneficial, 12
  neutral, and 12 harmful fixtures;
- report version: `rule-induction-report/v1`; and
- observed report SHA-256:
  `22007cda174019cf3d49226d8f41874fddfc95d1c5d6d02650f7d61f750bfb6f`.

The locked runner resolved the candidate commit, required it to equal `HEAD`,
and required a completely clean checkout before exposing any locked seed.

## Mechanical result

- all 448 policy-fixture runs identified a theory and drained their agendas;
- every policy scored 8,192/8,192 held-out predictions;
- the independent oracle recorded 30,669 agreements and zero disagreements;
- stage boundaries and held-out stores remained byte-stable;
- all 15 adversarial controls passed;
- the report contained no null fields and was 644,537 bytes, below 8 MiB; and
- observed single-fixture maxima were 69,196 semantic work, 29 evaluations,
  17,449 fixed-point steps, 1,000 engine cycles, and 1,258 attributed units,
  all within their frozen ceilings.

## Preregistered contrasts

| Control | Shared-library reduction | 95% paired bootstrap CI | Paired randomization p | Gate |
| --- | ---: | ---: | ---: | --- |
| `lff-direct` | 38.332% | 31.972%–44.285% | 0.00009999 | pass, minimum 15% |
| `lff-task-local-invention` | 38.822% | 32.609%–44.767% | 0.00009999 | pass, minimum 15% |
| `shared-recomputed` | 38.524% | 32.181%–44.441% | 0.00009999 | pass, minimum 15% |
| `shared-inlined` | 3.284% | 2.811%–3.866% | 0.00009999 | fail, minimum 5% |

The shared-library and shared-inlined runs had identical accuracy and candidate
schedules, so the last comparison isolates materialized-relation execution
under the v1 cost model. The positive effect is statistically clear but smaller
than the frozen minimum effect; changing the threshold after seeing it would
invalidate the experiment.

On beneficial fixtures, shared-library reduced mean stage-2 candidate
dispositions by 87.302% relative to `lff-direct` (95% CI 84.674%–89.160%,
`p=0.00009999`), passing the 25% search gate. On harmful fixtures its work ratio
to direct was 1.0158, passing the maximum 2.0 overhead gate.

## Interpretation

The result supports these bounded claims:

- Nous can represent, search, evaluate, and independently verify the frozen
  finite Horn hypothesis space;
- it can materialize an invented relation with provenance and reuse it across a
  temporal task boundary;
- the shared representation can sharply reduce downstream search while
  preserving predictions; and
- the descriptor, transcript, oracle, corruption-control, and locked-run
  patterns survive transfer to relational induction.

The result does not establish unrestricted inductive logic programming,
open-ended predicate invention, general EURISKO behavior, real-world utility,
or the preregistered minimum materialized-execution effect. Most of the measured
advantage over conventional learners comes from carrying the discovered
definition and avoiding repeated search; materialized fixed-point reuse adds a
smaller 3.284% benefit in this bounded workload.

## Independent review

After the locked run, Chandrasekhar, Lovelace, and Harvey independently returned
`ACCEPT` for provenance, mechanics, status assignment, and the interpretation
above. All agreed that `valid-null` is required because the inlined-effect gate
alone failed.

## Reproduction

From the exact clean implementation candidate:

```sh
mise exec -- go run ./cmd/nous ruleinduction-trials \
  -panel locked \
  -implementation-commit 12806a268b3bdb81544b99f59cfbcc528c8575c2
```

Candidate verification completed with:

```sh
mise exec -- go test ./...
mise exec -- go vet ./...
mise exec -- go test -race ./internal/vocab/ruleinduction \
  ./internal/ruleinductionoracle ./internal/dsl
git diff --check
```
