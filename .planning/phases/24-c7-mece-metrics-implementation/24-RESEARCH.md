# Phase 24: C7 MECE Metrics Implementation - Research

**Researched:** 2026-02-05
**Domain:** AI Agent Evaluation Metrics, Claude CLI Integration, Parallel Execution
**Confidence:** MEDIUM

## Summary

This research investigates how to replace C7's single `overall_score` metric with 5 MECE (Mutually Exclusive, Collectively Exhaustive) agent-assessable metrics. The current C7 implementation uses a 4-task sequential evaluation system; this phase extends it to 5 MECE metrics with parallel execution capability, enhanced progress display, and research-based scoring thresholds.

Key findings:
- **Existing infrastructure is solid**: The current `internal/agent/` package provides executor, scorer, evaluator, and workspace management that can be extended rather than replaced
- **Parallel execution via errgroup**: Go's `golang.org/x/sync/errgroup` (already used in pipeline.go) is the standard for concurrent task execution with error handling
- **Claude Code subagents**: Claude Code supports spawning subagents for parallel evaluation, with configurable tools, timeouts, and permission modes
- **MECE metric definitions**: Research from SWE-bench, RepoGraph, and code comprehension benchmarks provides grounding for 5 isolated agent capabilities

**Primary recommendation:** Extend the existing C7 analyzer with 5 new metric implementations, use errgroup for parallel execution, and add a real-time progress display component that tracks tokens and costs.

## Standard Stack

The established libraries/tools for this domain:

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `golang.org/x/sync/errgroup` | latest | Parallel goroutine management | Already used in pipeline.go, handles errors and context cancellation |
| `github.com/mattn/go-isatty` | v0.0.20 | TTY detection for progress display | Already used in pipeline/progress.go |
| `os/exec` | stdlib | Claude CLI subprocess execution | Already used in internal/agent/executor.go |
| `encoding/json` | stdlib | CLI JSON output parsing | Already used for structured output |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `sync` | stdlib | Mutex for concurrent metric updates | Progress display state management |
| `context` | stdlib | Timeout and cancellation | Per-metric timeout control |
| `time` | stdlib | Duration tracking, cost estimation | Token counting, progress display |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| errgroup | sync.WaitGroup | errgroup handles errors and SetLimit; WaitGroup is simpler but no error propagation |
| os/exec | ssh subprocesses | os/exec is simpler, SSH would add complexity for no benefit |
| Custom progress | github.com/schollz/progressbar | External dep adds complexity; custom matches existing Spinner pattern |

**Installation:**
```bash
# No new dependencies required - all already in go.mod
go mod tidy
```

## Architecture Patterns

### Recommended Project Structure
```
internal/
├── agent/
│   ├── metrics/           # NEW: 5 MECE metric implementations
│   │   ├── m1_consistency.go
│   │   ├── m2_comprehension.go
│   │   ├── m3_navigation.go
│   │   ├── m4_identifiers.go
│   │   ├── m5_documentation.go
│   │   └── registry.go    # Metric interface and registration
│   ├── progress.go        # NEW: Real-time progress display
│   ├── executor.go        # EXTEND: Add parallel execution
│   ├── scorer.go          # EXTEND: Per-metric rubrics
│   ├── evaluator.go       # EXISTING: Reuse for LLM evaluation
│   ├── tasks.go           # EXISTING: Reference for task patterns
│   └── types.go           # EXTEND: New metric result types
├── analyzer/c7_agent/
│   └── agent.go           # EXTEND: Orchestrate 5 metrics
└── scoring/
    └── config.go          # EXTEND: 5 new metric breakpoints
```

### Pattern 1: Metric Interface
**What:** Each metric implements a common interface for uniform execution
**When to use:** All 5 MECE metrics follow this pattern
**Example:**
```go
// Source: Pattern derived from existing analyzer interfaces
type Metric interface {
    ID() string                                    // e.g., "task_execution_consistency"
    Name() string                                  // e.g., "Task Execution Consistency"
    Description() string                           // What this metric measures
    Timeout() time.Duration                        // Per-metric timeout
    SampleCount() int                              // Number of samples (1-5)
    SelectSamples(targets []*types.AnalysisTarget) []Sample  // Heuristic selection
    Execute(ctx context.Context, workspace string, samples []Sample) MetricResult
}

type MetricResult struct {
    MetricID    string
    Score       int     // 1-10 scale
    Samples     []SampleResult
    TokensUsed  int
    Duration    time.Duration
    Error       string  // Empty if successful
}
```

### Pattern 2: Parallel Metric Execution with errgroup
**What:** Execute all 5 metrics concurrently with error aggregation
**When to use:** Main C7 evaluation entry point
**Example:**
```go
// Source: Follows pipeline.go parallel analyzer pattern
func (a *C7Analyzer) executeMetrics(ctx context.Context, workspace string, targets []*types.AnalysisTarget) []MetricResult {
    g, ctx := errgroup.WithContext(ctx)
    results := make([]MetricResult, len(metrics))
    var mu sync.Mutex

    for i, metric := range allMetrics {
        i, metric := i, metric // capture loop vars
        g.Go(func() error {
            samples := metric.SelectSamples(targets)
            result := metric.Execute(ctx, workspace, samples)

            mu.Lock()
            results[i] = result
            a.progress.UpdateMetric(metric.ID(), result)
            mu.Unlock()

            return nil // Don't abort other metrics on failure
        })
    }

    _ = g.Wait()
    return results
}
```

### Pattern 3: Real-Time Progress Display
**What:** Thread-safe progress display showing metric status, tokens, and cost
**When to use:** During C7 evaluation when terminal is TTY
**Example:**
```go
// Source: Extends existing pipeline/progress.go Spinner pattern
type C7Progress struct {
    mu          sync.Mutex
    isTTY       bool
    writer      *os.File
    metrics     map[string]*MetricProgress
    totalTokens int
    startTime   time.Time
}

type MetricProgress struct {
    Name       string
    Status     string  // "pending", "running", "complete", "error"
    Sample     int     // Current sample (e.g., 2/5)
    TotalSamples int
    Score      int     // Final score when complete
}

func (p *C7Progress) Render() {
    // Clear and redraw progress display
    // Format: "M1: Running (2/5) | M2: Complete (8/10) | M3: Pending | Tokens: 12,345 | Est. $0.15"
}
```

### Pattern 4: Heuristic-Based Sample Selection
**What:** Deterministic selection based on code characteristics, not random
**When to use:** Each metric's SelectSamples implementation
**Example:**
```go
// Source: Derived from SWE-bench reproducibility requirements
func (m *M2Comprehension) SelectSamples(targets []*types.AnalysisTarget) []Sample {
    // Select functions by complexity tiers for reproducibility
    var candidates []Sample
    for _, target := range targets {
        for _, file := range target.Files {
            if file.Class != types.ClassSource {
                continue
            }
            // Use deterministic scoring: complexity * (1 / sqrt(LOC))
            // Higher complexity, moderate size = better comprehension test
        }
    }

    // Sort by score, take top N
    sort.Slice(candidates, func(i, j int) bool {
        return candidates[i].SelectionScore > candidates[j].SelectionScore
    })

    if len(candidates) > m.sampleCount {
        candidates = candidates[:m.sampleCount]
    }
    return candidates
}
```

### Anti-Patterns to Avoid
- **Random sampling**: Use deterministic heuristics for reproducibility across runs
- **Sequential execution without progress**: Always show progress for long-running operations
- **Blocking on single metric failure**: Let other metrics complete; aggregate errors
- **Hardcoded sample counts**: Make sample count configurable per metric
- **Ignoring context cancellation**: Always check ctx.Done() in loops

## Don't Hand-Roll

Problems that look simple but have existing solutions:

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Token counting | Custom tokenizer | Claude CLI --output-format json | CLI returns token usage in response |
| Cost estimation | Custom pricing tables | Anthropic pricing constants | Pricing changes; use documented rates |
| JSON schema validation | Manual parsing | Claude CLI --json-schema | CLI handles structured output |
| TTY detection | Manual isatty checks | github.com/mattn/go-isatty | Already used, handles edge cases |
| Goroutine error handling | Custom channels | errgroup | Standard pattern, context-aware |
| Git worktree creation | Custom git commands | agent.CreateWorkspace() | Already implemented, handles fallbacks |

**Key insight:** The existing `internal/agent/` package already handles the hard problems (CLI invocation, workspace isolation, JSON parsing, timeout handling). Extend it rather than reimplementing.

## Common Pitfalls

### Pitfall 1: Non-Reproducible Results
**What goes wrong:** Different runs produce different scores due to random sampling or non-deterministic agent responses
**Why it happens:** Agent output varies by temperature; random file selection adds variance
**How to avoid:**
- Use deterministic heuristics for sample selection (complexity-based, not random)
- Run each metric task 3 times and take median/mode for M1 (Task Execution Consistency)
- Document expected variance bounds (research suggests 13% variance is common)
**Warning signs:** Same codebase scores vary by >15% between runs

### Pitfall 2: Blocking Progress Display
**What goes wrong:** Progress updates block metric execution or cause race conditions
**Why it happens:** Shared state between goroutines without proper synchronization
**How to avoid:**
- Use mutex for all progress state updates
- Non-blocking channel for progress events if needed
- Separate render goroutine with ticker (like existing Spinner)
**Warning signs:** Deadlocks during parallel execution, garbled terminal output

### Pitfall 3: Context Cancellation Ignored
**What goes wrong:** Metric execution continues after timeout, wasting tokens/cost
**Why it happens:** Long-running operations don't check context
**How to avoid:**
- Pass context to all metric.Execute() calls
- Check ctx.Err() in loops and after subprocess calls
- Use context.WithTimeout per metric, not global
**Warning signs:** Metrics complete after parent context cancelled

### Pitfall 4: Token Count Estimation Errors
**What goes wrong:** Cost estimates wildly inaccurate
**Why it happens:** Relying on character/4 approximation instead of actual CLI output
**How to avoid:**
- Parse actual token counts from Claude CLI JSON response
- Track input and output tokens separately (different pricing)
- Use official pricing: Sonnet 4.5 at $3/$15 per MTok
**Warning signs:** Estimated cost differs from actual by >20%

### Pitfall 5: Metric Interdependence
**What goes wrong:** Metrics test overlapping capabilities, violating MECE principle
**Why it happens:** Not clearly defining metric boundaries
**How to avoid:**
- M1 (Consistency): Tests variance across runs, not quality
- M2 (Comprehension): Tests understanding of code behavior
- M3 (Navigation): Tests cross-file dependency tracing
- M4 (Identifiers): Tests naming interpretation in isolation
- M5 (Documentation): Tests comment/code alignment detection
**Warning signs:** High correlation (>0.8) between metric scores

## Code Examples

Verified patterns from official sources:

### Claude CLI Structured Output
```go
// Source: Existing internal/agent/evaluator.go pattern
func executeWithSchema(ctx context.Context, workDir, prompt, schema string) (map[string]interface{}, error) {
    args := []string{
        "-p", prompt,
        "--output-format", "json",
        "--json-schema", schema,
        "--allowedTools", "Read,Glob,Grep",
    }

    ctx, cancel := context.WithTimeout(ctx, timeout)
    defer cancel()

    cmd := exec.CommandContext(ctx, "claude", args...)
    cmd.Dir = workDir
    cmd.Cancel = func() error {
        return cmd.Process.Signal(os.Interrupt)
    }
    cmd.WaitDelay = 10 * time.Second

    output, err := cmd.CombinedOutput()
    // Parse JSON response...
}
```

### Parallel Execution with Progress
```go
// Source: Combines pipeline.go errgroup pattern with progress updates
func (a *C7Analyzer) runParallelMetrics(ctx context.Context, workspace string, targets []*types.AnalysisTarget, progress *C7Progress) ([]MetricResult, error) {
    g, ctx := errgroup.WithContext(ctx)
    results := make([]MetricResult, len(allMetrics))

    for i, m := range allMetrics {
        i, m := i, m
        g.Go(func() error {
            progress.SetStatus(m.ID(), "running")

            samples := m.SelectSamples(targets)
            for j, sample := range samples {
                progress.SetSample(m.ID(), j+1, len(samples))
                // Execute sample...
            }

            result := m.Execute(ctx, workspace, samples)
            results[i] = result
            progress.SetComplete(m.ID(), result.Score)
            progress.AddTokens(result.TokensUsed)

            return nil
        })
    }

    if err := g.Wait(); err != nil {
        return nil, err
    }
    return results, nil
}
```

### Weighted Score Aggregation
```go
// Source: Follows existing scoring/scorer.go CategoryScore pattern
func aggregateC7Score(results []MetricResult, weights map[string]float64) float64 {
    totalWeight := 0.0
    weightedSum := 0.0

    for _, r := range results {
        if r.Error != "" {
            continue // Skip failed metrics
        }
        weight := weights[r.MetricID]
        weightedSum += float64(r.Score) * weight
        totalWeight += weight
    }

    if totalWeight == 0 {
        return 0.0
    }
    return weightedSum / totalWeight
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Single overall_score | 5 MECE metrics | This phase | Granular understanding of agent capabilities |
| Sequential task execution | Parallel execution | This phase | ~5x faster evaluation (5 metrics in parallel) |
| Simple spinner | Multi-metric progress | This phase | Better UX during long evaluations |
| 4 compound tasks | 5 isolated capabilities | This phase | Clearer metric interpretation |

**Deprecated/outdated:**
- `IntentClarityTask`, `ModificationConfidenceTask`, `CrossFileCoherenceTask`, `SemanticCompletenessTask`: These 4 compound tasks are superseded by 5 MECE metrics but the task execution infrastructure should be preserved

## MECE Metric Definitions

Based on research from SWE-bench, RepoGraph, and code comprehension benchmarks:

### M1: Task Execution Consistency
**Measures:** Reproducibility of agent task completion across multiple runs
**Method:** Execute same task 3 times, measure variance in completion and output
**Research basis:** Agent benchmarks show 13% variance in results; consistency is critical for reliability
**Scoring:**
- 10: <5% variance across runs
- 7: <15% variance
- 4: <30% variance
- 1: >30% variance or frequent failures

### M2: Code Behavior Comprehension
**Measures:** Agent's ability to understand what code does (not syntax, but semantics)
**Method:** Select complex functions, ask agent to explain behavior, score accuracy
**Research basis:** Code comprehension benchmarks show LLMs struggle with semantic understanding vs syntactic correctness
**Scoring:**
- 10: Accurately explains all edge cases and error handling
- 7: Correct main path, misses some edge cases
- 4: Partially correct, significant gaps
- 1: Fundamentally misunderstands behavior

### M3: Cross-File Navigation
**Measures:** Agent's ability to trace dependencies across files
**Method:** Select import chains, ask agent to trace data flow
**Research basis:** RepoGraph shows 32.8% improvement when agents have repository-level understanding
**Scoring:**
- 10: Traces complete dependency chain accurately
- 7: Traces most of chain, minor gaps
- 4: Traces direct dependencies only
- 1: Cannot navigate beyond single file

### M4: Identifier Interpretability
**Measures:** Agent's ability to infer meaning from identifier names
**Method:** Present identifiers in isolation, score semantic interpretation
**Research basis:** Descriptive compound identifiers improve comprehension; tests agent's ability to leverage naming
**Scoring:**
- 10: Correctly interprets purpose from names alone
- 7: Mostly correct, some ambiguity
- 4: Requires context to interpret
- 1: Misinterprets identifier meanings

### M5: Documentation Accuracy Detection
**Measures:** Agent's ability to detect comment/code mismatches
**Method:** Present code with potentially outdated comments, ask agent to identify discrepancies
**Research basis:** Code comment inconsistency detection research shows this is a distinct, measurable capability
**Scoring:**
- 10: Identifies all mismatches with correct explanations
- 7: Identifies most mismatches
- 4: Identifies obvious mismatches only
- 1: Cannot reliably detect mismatches

## Open Questions

Things that couldn't be fully resolved:

1. **Optimal sample count per metric**
   - What we know: More samples = better coverage but higher cost
   - What's unclear: Exact diminishing returns threshold
   - Recommendation: Start with 3 samples per metric, allow configuration

2. **Metric weight derivation**
   - What we know: Weights should reflect research-derived impact on agent success
   - What's unclear: Exact weight values without empirical calibration
   - Recommendation: Start with equal weights (0.2 each), calibrate in Phase 25 with citations

3. **Timeout values per metric**
   - What we know: Complex metrics need more time; simple metrics less
   - What's unclear: Optimal timeout values without empirical data
   - Recommendation: M1=180s (3 runs), M2=120s, M3=120s, M4=60s, M5=60s

4. **Failure scoring policy**
   - What we know: Timeouts/errors need to produce informative scores
   - What's unclear: Should timeout = 0, = 1, or = partial credit?
   - Recommendation: Timeout = 0 (counts as unavailable), Error = 0 with explanation logged

## Sources

### Primary (HIGH confidence)
- Existing codebase: `internal/agent/`, `internal/scoring/`, `internal/pipeline/` - Verified patterns for executor, scorer, progress, parallel execution
- [errgroup documentation](https://pkg.go.dev/golang.org/x/sync/errgroup) - Goroutine synchronization patterns
- [Claude Code subagents](https://code.claude.com/docs/en/sub-agents) - Agent spawning, tools, permissions

### Secondary (MEDIUM confidence)
- [SWE-bench evaluation methodology](https://www.swebench.com/SWE-bench/) - Resolve rate metrics, variance handling
- [RepoGraph ICLR 2025](https://openreview.net/forum?id=dw9VUsSHGB) - Repository-level code understanding
- [Agent CI consistency evaluations](https://agent-ci.com/docs/evaluations/consistency/) - Multi-run variance measurement
- [Anthropic pricing](https://platform.claude.com/docs/en/about-claude/pricing) - Sonnet 4.5 at $3/$15 per MTok

### Tertiary (LOW confidence)
- [Code comprehension benchmark](https://arxiv.org/abs/2507.10641) - Semantic vs syntactic understanding
- [Code comment inconsistency research](https://dl.acm.org/doi/10.1109/TSE.2024.3358489) - CCI detection methodology
- [Go progress bar libraries](https://github.com/schollz/progressbar) - Alternative to custom progress

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - Uses existing codebase patterns and stdlib
- Architecture: MEDIUM - New metric interface pattern not yet validated
- MECE definitions: MEDIUM - Research-grounded but specific scoring thresholds are estimates
- Parallel execution: HIGH - errgroup pattern well-established in codebase
- Progress display: MEDIUM - Extends existing Spinner but new multi-metric display

**Research date:** 2026-02-05
**Valid until:** 30 days (patterns stable, but verify Claude CLI features)
