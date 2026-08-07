# DevOps configuration-repair reality gate

## Goal

This trial asks whether the configuration-repair vocabulary can select safe,
minimal repairs for recognizable Kubernetes and Terraform failures. It is a
reality gate for the vocabulary, not a claim that Nous currently reads or
repairs Kubernetes YAML or Terraform HCL.

The 20 incidents were proposed independently, then translated into the current
bounded representation. Each incident runs in a fresh Nous store with:

- four training fact configurations;
- two held-out fact configurations;
- one to three required assignment primitives;
- two unsafe primitives that evade a conditional policy by changing protected
  environment or ownership intent;
- an exhaustive direct-search baseline over the identical primitive catalog.

The actual Nous synthesis and evaluation heuristics enumerate assignment
subsets of size one through three, apply them, check schema constraints,
preserve protected scalar values, check idempotence, and promote qualifying
plans. The baseline applies the same pure configuration semantics without the
Nous engine.

Run it with:

```sh
mise exec -- go run ./cmd/nous configrepair-trials
```

## Scenario catalog

| ID | Technology | Incident and goal | Expected translated repair | Translation fidelity |
|---|---|---|---|---|
| K01 | Kubernetes | Restore Service endpoints without changing Service or immutable Deployment selectors. | `pod_template_app=api` | Selector relations become scalar facts. |
| K02 | Kubernetes | Keep one ready replica throughout a single-replica rolling update. | `max_unavailable=0` | Scalar proxy for rollout simulation. |
| K03 | Kubernetes | Make a required readiness probe resolve to the declared application port. | `readiness_port_ref=name_http` | Reference resolution becomes an enum. |
| K04 | Kubernetes | Preserve a 1 GiB memory request while making request less than or equal to limit. | `memory_limit_mib=1024` | Kubernetes quantities become integer MiB. |
| K05 | Kubernetes | Satisfy the restricted Pod security profile without weakening namespace enforcement. | `pod_security_profile=restricted` | Four concrete security edits become one profile enum. |
| K06 | Kubernetes | Restore Service connectivity without changing public or container ports. | `service_target_port=name_http` | Reference resolution becomes an enum. |
| K07 | Kubernetes | Admit only payments API pods to PostgreSQL while preserving that legitimate path. | `ingress_peer_scope=payments_api_pods` | Selectors and reachability become one enum. |
| K08 | Kubernetes | Permit one voluntary eviction while retaining two of three replicas. | `pdb_min_available=2` | Scalar proxy for disruption semantics. |
| K09 | Kubernetes | Keep HPA scaling above a three-replica availability floor. | `hpa_min_replicas=3` | Scalar proxy for HPA interval semantics. |
| K10 | Kubernetes | Replace wildcard controller RBAC while retaining reconciliation capabilities. | `rbac_profile=deployment_controller_minimal` | Permission tuple sets become a profile enum. |
| T11 | Terraform | Put a CloudFront certificate in `us-east-1` without changing the default provider. | `certificate_provider=aws_us_east_1` | Provider reference becomes a string. |
| T12 | Terraform | Separate production state from staging while preserving the backend. | `backend_key=prod_platform.tfstate` | Corpus uniqueness becomes a prescribed key. |
| T13 | Terraform | Prevent a production database destroy from a synthetic replacement plan. | `prevent_destroy=true` | Plan and lifecycle semantics become a boolean obligation. |
| T14 | Terraform | Preserve subnet identity when availability zones reorder. | `iteration_mode=for_each_set`, `availability_zone_ref=each_key` | Address evaluation becomes two enums. |
| T15 | Terraform | Guarantee migration runs after database creation without sleeps or global serialization. | `database_dependency_path=true` | A dependency graph path becomes a boolean. |
| T16 | Terraform | Remove public SSH while retaining corporate administrator access. | `ssh_source=admin_cidrs` | CIDR containment and reachability become an enum. |
| T17 | Terraform | Keep production at a two-instance floor with autoscaling and multi-AZ placement. | `asg_min_size=2`, `asg_desired_capacity=2` | Placement semantics become integer constraints. |
| T18 | Terraform | Narrow wildcard IAM while retaining reports read/write. | `iam_policy_profile=reports_read_write` | Permission tuple sets become a profile enum. |
| T19 | Terraform | Make create-before-destroy compatible with remote name uniqueness. | `fixed_name_present=false`, `name_prefix=api_prod` | Unset and insert operations become assignments. |
| T20 | Terraform | Redact a generated password from ordinary output without breaking consumers. | `output_sensitive=true` | Expression taint becomes a boolean obligation. |

The original semantic goals remain useful future validator targets: selector and
reference resolution, rollout and disruption simulation, quantity and CIDR
parsing, reachability, permission tuples, provider resolution, state identity,
Terraform dependency and address graphs, synthetic plans, remote-name
uniqueness, and sensitive-expression taint.

## Observed result

The deterministic run produced:

| Measurement | Result |
|---|---:|
| Scenarios | 20: 10 Kubernetes, 10 Terraform |
| Intended plans recovered by Nous | 20/20 |
| Unique Nous promotions | 20/20 |
| Held-out fact variants | 40 |
| Held-out failures | 0 |
| Candidates containing unsafe shortcuts | 135 |
| Unsafe candidates rejected | 135/135 |
| Intended plans recovered by exhaustive baseline | 20/20 |
| Exact Nous/baseline solution-set agreements | 20/20 |

This demonstrates that the current vocabulary can correctly compose up to
three supplied scalar assignments, evaluate them across several examples,
preserve explicitly protected scalar intent, reject shortcut candidates, and
retain the plan on similarly shaped held-out facts.

It does **not** demonstrate useful real-life Kubernetes or Terraform repair.
The translation supplies both the policy target and the exact repair values.
Most technology semantics are collapsed into facts whose desired value is
already stated by an equality or minimum constraint. The baseline finds exactly
the same answers with the same bounded enumeration. Nous has therefore shown a
sound orchestration and evidence path, but no search advantage, repair-value
invention, or semantic understanding in this trial.

## Representation work required for a genuine trial

A higher-fidelity vocabulary needs:

- hierarchical paths and typed maps, lists, sets, quantities, CIDRs,
  percentages, references, expressions, absent values, nulls, and unknowns;
- multiple named artifacts and relations such as selects, grants, depends-on,
  and shares-state-with;
- patch operations including set, unset, insert, remove, replace-set, re-key,
  create, and delete;
- ordered, possibly dependent plans rather than only unordered distinct-key
  assignments;
- protected semantic predicates for reachability, availability, privilege,
  stable addresses, output contracts, destruction, and downtime;
- technology-specific deterministic validators and normalized evidence;
- costs for privilege expansion, destructive actions, downtime, state churn,
  and dependency scope—not changed-key count alone.

The next useful gate should choose a narrow vertical slice rather than build all
of this at once. Selector/reference repair for Kubernetes is a good candidate:
it needs multiple artifacts, paths, and relations, but can still be validated
locally and deterministically without a cluster.
