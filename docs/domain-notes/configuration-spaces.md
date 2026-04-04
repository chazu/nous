# Configuration Space Exploration Domain

## Motivation

Real systems have configuration knobs — timeouts, retry counts, batch sizes, pool sizes, cache TTLs, buffer lengths, concurrency limits. These interact in non-obvious ways. Given even a simple model of how a system behaves as a function of its configuration, an EURISKO-style system could explore the parameter space, discover dominated configurations, find Pareto frontiers, and learn which knobs actually matter.

The output is directly actionable: "for this workload profile, these 3 configurations cover the useful space" or "batch size and pool size are redundant — increasing either has the same effect on throughput."

## Why this fits Mode 1

- Configuration is a vector of numeric values — fully computable
- A system model maps configuration → performance metrics (even a simple model works)
- Quality is multi-objective: throughput, latency, resource usage, cost
- The space is continuous but can be discretized and explored systematically
- Pareto dominance is a computable evaluation function

## Concept space

### Core types

- **Config** — a named configuration vector. Data slot is a map of knob names to values.
- **SystemModel** — a function from Config → Metrics. Implemented as a DSL program.
- **Metric** — a named performance measurement (throughput, p99Latency, memoryUsage, cost).
- **ConfigOp** — a transform on configurations (scale a knob, combine two configs, interpolate).
- **WorkloadProfile** — parameters describing the workload (request rate, payload size, read/write ratio).
- **ParetoFrontier** — a set of non-dominated configurations.

### Data representation

A Config as a unit:
```
name: "Config-Baseline"
data: {
    batchSize: 100,
    poolSize: 10,
    timeoutMs: 5000,
    retryCount: 3,
    cacheTTL: 300,
    bufferSize: 1024,
    concurrencyLimit: 50,
}
metrics: {
    throughput: 1200,
    p99Latency: 450,
    memoryMB: 512,
    costPerHour: 2.50,
}
```

A SystemModel as a unit:
```
name: "SimpleQueueModel"
defn: `
    # Throughput: limited by min(poolSize * processingRate, concurrencyLimit)
    "cfg" @ "poolSize" get-field 100 *
    "cfg" @ "concurrencyLimit" get-field
    min
    "throughput" !

    # Latency: base + (batchSize / throughput) * 1000 + retry overhead
    10 "cfg" @ "batchSize" get-field "throughput" @ / 1000 * +
    "cfg" @ "retryCount" get-field 5 * +
    "latency" !

    # Memory: proportional to poolSize * bufferSize
    "cfg" @ "poolSize" get-field "cfg" @ "bufferSize" get-field * 1024 /
    "memory" !
`
```

### Operations

**On configurations:**
- **ScaleKnob** — multiply a single knob by a factor (e.g., double the pool size)
- **PerturbKnob** — add/subtract a small amount from a knob
- **Interpolate** — blend two configurations: config_new = α * config_A + (1-α) * config_B
- **Extremize** — set a knob to its minimum or maximum
- **ZeroOut** — set a knob to its minimum (test if it matters)
- **Combine** — take knob values from two configs (A's batch size with B's pool size)

**On configuration spaces:**
- **GridSearch** — evaluate all combinations at discrete points
- **SensitivityAnalysis** — vary one knob while holding others constant, measure effect
- **ParetoFilter** — given a set of evaluated configs, remove dominated ones
- **FindKnee** — find the configuration where improving one metric starts hurting another rapidly

**On models:**
- **ComposeModels** — chain two system models (output of one feeds input of next)
- **RestrictModel** — fix some knobs, reducing the search space

### Quality attributes

- `throughput` — higher is better
- `p99Latency` — lower is better
- `memoryUsage` — lower is better
- `cost` — lower is better
- `paretoDominates` — does this config dominate another? (better on all metrics)
- `paretoRank` — how many configs dominate this one?
- `sensitivity` — how much does each metric change when this knob changes?
- `knobCorrelation` — do two knobs have the same effect? (redundancy detection)

### What the system could discover

**Dominated configurations:**
- Config A with batchSize=100, poolSize=20 is strictly worse than Config B with batchSize=200, poolSize=10 on all metrics. A is dominated and can be discarded.

**Knob redundancy:**
- "Increasing batchSize by 2x has the same effect on throughput as increasing poolSize by 2x. These knobs are interchangeable for throughput — use whichever is cheaper on memory."

**Pareto frontiers:**
- "There are exactly 4 non-dominated configurations in this space. Everything else is strictly worse on at least one metric." The system discovers the efficient frontier.

**Phase transitions:**
- "Below poolSize=5, latency is flat. Above poolSize=50, memory grows linearly but throughput stops improving. The useful range is 5-50." The system discovers the knee points.

**Interaction effects:**
- "Increasing retryCount helps throughput when timeoutMs is low (retries recover from timeouts) but hurts throughput when timeoutMs is high (retries just waste resources on slow requests)." A non-obvious interaction.

**Optimal defaults:**
- "For the common workload profile, this single configuration is within 10% of optimal on all metrics." The system finds robust defaults.

### DSL builtins needed

- `config-new` — create from knob-value pairs
- `config-get-knob`, `config-set-knob`
- `config-scale-knob`, `config-perturb-knob`
- `config-interpolate` — blend two configs
- `config-evaluate` — run a system model on a config, produce metrics
- `config-dominates?` — does config A dominate config B on all metrics?
- `config-pareto-filter` — given a list of configs, return non-dominated ones
- `config-sensitivity` — vary one knob, measure metric changes
- `metrics-compare` — compare two metric vectors

### System models

The domain needs system models — functions from configuration to metrics. These can range from trivial to realistic:

**Trivial:** Linear models. `throughput = poolSize * 100`. Good enough to test the exploration machinery.

**Simple queuing:** M/M/c queue model. `latency = f(arrivalRate, serviceRate, poolSize)`. Captures saturation and queuing delay.

**Empirical:** A lookup table from real measurements. pudl could ingest actual benchmark results and nous could explore interpolations.

The key insight: the system model doesn't need to be accurate. It needs to be consistent enough that the exploration is meaningful. An inaccurate model still produces valid Pareto frontiers *within the model* — and the discovered relationships (knob redundancy, phase transitions) often hold in reality even when the absolute numbers don't.

### Implementation complexity

**Low.** Configurations are maps of strings to numbers. System models are DSL programs that do arithmetic. Pareto dominance is pairwise comparison. Sensitivity analysis is evaluating the model with perturbed inputs. No graph algorithms, no NP-hard problems.

This is probably the simplest domain to implement after math.

### Seed content

- A simple system model (queue-based: throughput, latency, memory as functions of 5 knobs)
- 3-5 baseline configurations spanning the space
- Basic operations: ScaleKnob, PerturbKnob, Interpolate
- Evaluation: ParetoFilter, Sensitivity
- A workload profile to evaluate against
