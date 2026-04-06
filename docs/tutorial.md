# Tutorial: Running the Full Pipeline

This walks through the complete loop: agents observe a codebase, nous reasons over the observations, and discoveries flow back to pudl for querying.

## Prerequisites

```bash
# Build both tools
cd ~/dev/go/pudl && go build -o pudl .
cd ~/dev/go/nous && go build -o nous ./cmd/nous

# Ensure pudl is initialized
pudl init
```

## Step 1: Record Observations

Pick a repo you're working in and record observations about it. These can come from you, from agents during code review, or from CI hooks. The key is to be honest and specific.

```bash
# Obstacles — things blocking progress or causing pain
pudl observe "circular dependency between auth and user packages" \
    --kind obstacle --repo pkg/auth

pudl observe "no integration tests for the API layer" \
    --kind obstacle --repo cmd/api

# Patterns — recurring structures you've noticed
pudl observe "all handlers follow the middleware chain pattern" \
    --kind pattern --repo cmd/api

pudl observe "every service has its own connection pool" \
    --kind pattern --repo internal/db

# Antipatterns — recurring problems
pudl observe "error handling is inconsistent across the API" \
    --kind antipattern --repo cmd/api

# Suggestions — ideas for improvement
pudl observe "Config struct has too many fields, should split" \
    --kind suggestion --repo internal/config

# Facts — verified truths
pudl observe "all 20 packages pass tests" --kind fact --repo myproject

# Bugs
pudl observe "race condition in cache invalidation" \
    --kind bug --repo internal/cache
```

**Tips:**
- Use `--source agent-name` if recording on behalf of an agent (defaults to your OS username)
- One observation per concern. Be specific.
- `--repo` is optional but makes observations joinable by Datalog rules

**Simulate multiple agents** by varying `--source`:

```bash
pudl observe "auth package is overly complex" \
    --kind obstacle --repo pkg/auth --source claude-code

pudl observe "auth package has too many responsibilities" \
    --kind obstacle --repo pkg/auth --source review-agent
```

Corroboration (multiple sources flagging the same area) is signal that nous picks up on.

## Step 2: Check What's There

```bash
# List all observations
pudl facts list --relation observation

# Filter by kind
pudl facts list --relation observation | grep obstacle

# Detailed view of one
pudl facts show <id-prefix>

# Machine-readable
pudl facts list --relation observation --json
```

## Step 3: Run nous

```bash
nous run -domain observations -pudl ~/.pudl -cycles 50 -no-mutate -v 1
```

Watch the output. nous will:
- Report how many observations it loaded
- Fire heuristics against each observation
- Print when it finds hotspots or systemic patterns
- Report what it writes back to pudl on exit

**What to look for:**
- `Repo hotspot: X (N observations)` — repos with concentrated attention
- `Systemic pattern across N repos` — the same kind of issue appearing everywhere
- `X corroborated by N other source(s)` — independent confirmation
- `nous → pudl: ...` — discoveries written back

**Adjust cycles:** If nous runs out of things to do quickly (< 20 cycles), that's fine — it means it reached a fixed point. If it's still finding things at 50 cycles, try more.

## Step 4: See What nous Found

```bash
# Observations written by nous
pudl facts list --relation observation --source nous

# Repo hotspots
pudl facts list --relation repo_hotspot

# Everything, verbose
pudl facts list --relation observation -v
pudl facts list --relation repo_hotspot -v
```

## Step 5: Write Datalog Rules (Optional)

If you see patterns in the observations, codify them as rules:

```bash
cat > /tmp/my-rules.cue << 'EOF'
// Repos with both obstacles and no tests are high risk
highRisk: {
    name: "high_risk_repo"
    head: { rel: "high_risk", args: { repo: "$R" } }
    body: [
        { rel: "observation", args: { kind: "obstacle", repo: "$R" } },
        { rel: "observation", args: { kind: "bug", repo: "$R" } },
    ]
}
EOF

# Validate and install
pudl rule add /tmp/my-rules.cue --global

# Query the derived relation
pudl query high_risk
```

Rules are evaluated on every `pudl query` call — they don't need nous.

## Step 6: Iterate

The pipeline is designed to run repeatedly:

```bash
# More work happens, agents make more observations...
pudl observe "fixed the race condition in cache" --kind fact --repo internal/cache

# Invalidate observations that are no longer true
pudl facts invalidate <id-of-race-condition-bug>

# Run nous again — it sees the updated state
nous run -domain observations -pudl ~/.pudl -cycles 50 -no-mutate
```

Over time:
- New observations accumulate
- nous finds new patterns each run
- Old observations decay in worth (H-PenalizeStaleObservations)
- Corroborated observations rise in worth
- You can retract wrong observations: `pudl facts retract <id>`

## Step 7: Temporal Queries (Post-Mortem)

After a few sessions, you can query historical state:

```bash
# What observations existed last Tuesday?
pudl facts list --relation observation --as-of-tx "2026-04-01T00:00:00Z"

# What would Datalog have derived from last week's data?
pudl query high_risk --as-of-valid "2026-03-31T00:00:00Z"

# What did nous write back?
pudl facts list --relation observation --source nous
```

## Example Session

Here's a concrete example using pudl's own codebase:

```bash
# Record observations about pudl
pudl observe "CatalogDB has 39 methods across 8 files" \
    --kind suggestion --repo internal/database

pudl observe "streaming imports bypass CUE validation (TODO stubs)" \
    --kind obstacle --repo internal/streaming

pudl observe "importer.go is 978 lines handling multiple concerns" \
    --kind suggestion --repo internal/importer

pudl observe "bootstrap schemas require pudl init to propagate changes" \
    --kind pattern --repo internal/importer

pudl observe "schemagen handles all formats inline at 893 lines" \
    --kind suggestion --repo internal/schemagen

# Different source corroborates
pudl observe "importer.go mixes format detection with schema inference" \
    --kind suggestion --repo internal/importer --source code-review-agent

# Run nous
nous run -domain observations -pudl ~/.pudl -cycles 50 -no-mutate -v 1

# Check results
pudl facts list --relation repo_hotspot
pudl facts list --relation observation --source nous

# Write a rule based on what you see
cat > /tmp/refactor-candidates.cue << 'EOF'
refactorCandidate: {
    name: "refactor_candidate"
    head: { rel: "refactor_candidate", args: { repo: "$R" } }
    body: [
        { rel: "repo_hotspot", args: { repo: "$R" } },
        { rel: "observation",  args: { kind: "suggestion", repo: "$R" } },
    ]
}
EOF

pudl rule add /tmp/refactor-candidates.cue --global
pudl query refactor_candidate
```

## What's Not Built Yet

- **`pudl promote`** — converting a validated observation into a permanent Datalog rule. For now, write rules manually with `pudl rule add`.
- **Automatic observation from CI** — hook `pudl observe` into your CI pipeline or Claude Code hooks.
- **Incremental nous runs** — each run loads all observations from scratch. Fine for hundreds, may need optimization for thousands.
- **Mode 2 mutation** — nous's self-modification doesn't work in Mode 2 yet. Use `-no-mutate`.
