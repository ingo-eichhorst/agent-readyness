# Phase 3: Scoring Model - Research

**Researched:** 2026-01-31
**Domain:** Numeric scoring -- piecewise linear interpolation, weighted composites, tier classification, configurable thresholds
**Confidence:** HIGH

## Summary

Phase 3 transforms the raw metric data produced by the C1/C3/C6 analyzers (Phase 2) into meaningful 1-10 category scores and a composite score with tier ratings. The existing codebase provides fully typed metric structs (`C1Metrics`, `C3Metrics`, `C6Metrics`) stored in `AnalysisResult.Metrics` as `map[string]interface{}` values. The scoring layer sits between the analyzer output and the terminal renderer, consuming `[]*types.AnalysisResult` and producing scored results.

The core technical challenge is defining the piecewise linear interpolation that maps each raw metric (e.g., avg cyclomatic complexity of 8.5) to a sub-score (e.g., 7.2 out of 10), then combining sub-scores into per-category scores (weighted average of sub-scores within a category), and finally combining category scores into a composite score using the specified weights (C1: 25%, C3: 20%, C6: 15%). The tier rating is a simple threshold lookup on the composite score. All threshold values must be configurable via a Go struct with sensible defaults, with optional YAML file override for tuning.

The mathematical operations involved are trivially implementable in pure Go (no external libraries needed). Piecewise linear interpolation is ~20 lines, weighted average is ~5 lines, and tier classification is a simple if-else chain. The research focus is on the right architecture patterns, threshold calibration strategy, and integration with the existing pipeline.

**Primary recommendation:** Create an `internal/scoring` package with a `Scorer` type that takes `[]*types.AnalysisResult` and a `Config` struct of threshold breakpoints, and returns `ScoredResult` containing per-category scores, sub-scores, composite score, and tier rating. Use Go struct literals for default thresholds. Add optional `--config` flag for YAML override.

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go stdlib (`math`, `sort`) | stdlib | Piecewise linear interpolation, sorting breakpoints | No external dependencies needed. The interpolation is < 30 lines of code. |
| `gopkg.in/yaml.v3` | v3.0.1 | Parse optional YAML config file for threshold overrides | Go community standard for YAML. Lightweight, well-maintained. Only needed if config file support is added. |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `encoding/json` | stdlib | Alternative config format (JSON) | If YAML dependency is considered too heavy for just config. JSON is stdlib. |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Hand-rolled interpolation | `gonum.org/v1/gonum/interp` or `sgreben/piecewiselinear` | Gonum adds ~15MB to binary for a 20-line function. `sgreben/piecewiselinear` is tiny but still unnecessary -- the interpolation here is simple monotonic decreasing/increasing with known breakpoints. |
| YAML config | Viper | Viper adds significant dependency tree (consul, etcd, etc.) for features we do not need. A single YAML file parsed with `yaml.v3` is sufficient. |
| YAML config | JSON config (stdlib only) | JSON is less human-friendly for editing thresholds. YAML is the standard for tool configuration files in the Go ecosystem (golangci-lint, goreleaser, etc.). |
| YAML config | Go struct defaults only | Meets "configurable" requirement minimally (change defaults = recompile). YAML adds runtime configurability without code changes, which the requirement explicitly calls for. |

**Installation:**
```bash
go get gopkg.in/yaml.v3@v3.0.1
```

## Architecture Patterns

### Recommended Project Structure (Phase 3 additions)

```
internal/
├── scoring/
│   ├── scorer.go          # Scorer type, piecewise linear interpolation, composite calculation
│   ├── scorer_test.go     # Unit tests with known inputs -> expected scores
│   ├── config.go          # ScoringConfig struct, default thresholds, YAML loading
│   └── config_test.go     # Config loading and validation tests
├── analyzer/              # (existing, unchanged)
├── pipeline/              # (existing, updated to call Scorer)
│   └── pipeline.go        # Add scoring stage between analyze and output
├── output/                # (existing, updated to render scores)
│   └── terminal.go        # Add score rendering, tier badge, verbose sub-score breakdown
pkg/
└── types/
    └── types.go           # Add ScoredResult, CategoryScore, SubScore, TierRating types
```

### Pattern 1: Piecewise Linear Interpolation via Breakpoints

**What:** Each metric-to-score mapping is defined as a sorted list of (raw_value, score) breakpoint pairs. For any raw value between two breakpoints, the score is linearly interpolated. Values below the first breakpoint clamp to its score; values above the last breakpoint clamp to its score.

**When to use:** For all 16 metric-to-sub-score conversions (6 C1 metrics, 5 C3 metrics, 5 C6 metrics).

**Key design insight:** Some metrics are "lower is better" (complexity, duplication) and some are "higher is better" (coverage, test ratio). The breakpoint table handles both directions naturally -- just order the breakpoints so that the score decreases for worse values.

**Example:**
```go
// internal/scoring/scorer.go

// Breakpoint defines a mapping from a raw metric value to a score.
type Breakpoint struct {
    Value float64 // raw metric value
    Score float64 // corresponding score (1-10)
}

// Interpolate computes the score for a given raw value using piecewise linear interpolation.
// Breakpoints must be sorted by Value in ascending order.
func Interpolate(breakpoints []Breakpoint, rawValue float64) float64 {
    if len(breakpoints) == 0 {
        return 5.0 // neutral default
    }

    // Clamp below first breakpoint
    if rawValue <= breakpoints[0].Value {
        return breakpoints[0].Score
    }

    // Clamp above last breakpoint
    last := breakpoints[len(breakpoints)-1]
    if rawValue >= last.Value {
        return last.Score
    }

    // Find enclosing segment and interpolate
    for i := 1; i < len(breakpoints); i++ {
        if rawValue <= breakpoints[i].Value {
            lo := breakpoints[i-1]
            hi := breakpoints[i]
            t := (rawValue - lo.Value) / (hi.Value - lo.Value)
            return lo.Score + t*(hi.Score-lo.Score)
        }
    }

    return last.Score
}
```

### Pattern 2: Scoring Config as Nested Struct

**What:** All threshold breakpoints live in a single `ScoringConfig` struct, with per-category, per-metric breakpoint arrays. The struct has a `Default()` constructor and can be loaded from YAML.

**When to use:** Always. This is the central configuration for the scoring model.

**Example:**
```go
// internal/scoring/config.go

// MetricThresholds defines the breakpoints for scoring a single metric.
type MetricThresholds struct {
    Name        string       `yaml:"name"`
    Weight      float64      `yaml:"weight"`      // relative weight within category
    Breakpoints []Breakpoint `yaml:"breakpoints"`
}

// CategoryConfig defines the scoring configuration for one category.
type CategoryConfig struct {
    Name    string             `yaml:"name"`
    Weight  float64            `yaml:"weight"`  // weight in composite score
    Metrics []MetricThresholds `yaml:"metrics"`
}

// ScoringConfig holds all scoring thresholds and weights.
type ScoringConfig struct {
    C1 CategoryConfig `yaml:"c1"`
    C3 CategoryConfig `yaml:"c3"`
    C6 CategoryConfig `yaml:"c6"`

    Tiers []TierConfig `yaml:"tiers"`
}

// TierConfig defines a tier rating boundary.
type TierConfig struct {
    Name     string  `yaml:"name"`
    MinScore float64 `yaml:"min_score"`
}

// DefaultConfig returns the default scoring configuration.
func DefaultConfig() *ScoringConfig {
    return &ScoringConfig{
        C1: CategoryConfig{
            Name:   "Code Health",
            Weight: 0.25,
            Metrics: []MetricThresholds{
                {
                    Name:   "cyclomatic_complexity_avg",
                    Weight: 0.25,
                    Breakpoints: []Breakpoint{
                        {Value: 1, Score: 10},   // perfect: avg complexity 1
                        {Value: 5, Score: 8},    // good
                        {Value: 10, Score: 6},   // moderate
                        {Value: 20, Score: 3},   // poor
                        {Value: 40, Score: 1},   // terrible
                    },
                },
                // ... other C1 metrics
            },
        },
        // ... C3, C6
        Tiers: []TierConfig{
            {Name: "Agent-Ready", MinScore: 8.0},
            {Name: "Agent-Assisted", MinScore: 6.0},
            {Name: "Agent-Limited", MinScore: 4.0},
            {Name: "Agent-Hostile", MinScore: 1.0},
        },
    }
}
```

### Pattern 3: Scorer as Pipeline Stage

**What:** The `Scorer` takes analysis results and config, produces `ScoredResult`. It becomes a new stage in the pipeline between analyze and output.

**Example:**
```go
// internal/scoring/scorer.go

type Scorer struct {
    Config *ScoringConfig
}

type ScoredResult struct {
    Categories []CategoryScore
    Composite  float64
    Tier       string
}

type CategoryScore struct {
    Name      string
    Score     float64
    SubScores []SubScore
}

type SubScore struct {
    MetricName string
    RawValue   float64
    Score      float64
    Weight     float64
}

func (s *Scorer) Score(results []*types.AnalysisResult) (*ScoredResult, error) {
    scored := &ScoredResult{}

    for _, ar := range results {
        switch ar.Category {
        case "C1":
            cs := s.scoreC1(ar)
            scored.Categories = append(scored.Categories, cs)
        case "C3":
            cs := s.scoreC3(ar)
            scored.Categories = append(scored.Categories, cs)
        case "C6":
            cs := s.scoreC6(ar)
            scored.Categories = append(scored.Categories, cs)
        }
    }

    // Composite = weighted average of category scores
    scored.Composite = s.computeComposite(scored.Categories)
    scored.Tier = s.classifyTier(scored.Composite)

    return scored, nil
}
```

### Pattern 4: Verbose Sub-Score Breakdown

**What:** In verbose mode, each sub-score is displayed showing the raw metric value, the interpolated score, and its weight contribution to the category score.

**Example output:**
```
C1: Code Health                    7.2 / 10
  Complexity avg:      8.5  -> 7.0  (weight: 25%)
  Function length avg: 22.3 -> 8.1  (weight: 20%)
  File size avg:       180  -> 8.5  (weight: 15%)
  Coupling (afferent): 3.2  -> 7.5  (weight: 15%)
  Coupling (efferent): 2.1  -> 8.0  (weight: 10%)
  Duplication rate:    4.2% -> 6.5  (weight: 15%)
```

### Anti-Patterns to Avoid

- **Hardcoded threshold values scattered across code:** All thresholds must live in the `ScoringConfig` struct. No magic numbers in scoring functions. Every comparison uses config values.
- **Scoring logic in the output renderer:** The `terminal.go` file should only render pre-computed scores. All scoring math happens in `internal/scoring`.
- **Non-deterministic scoring:** The same raw metrics must always produce the same scores. Do not use randomness, time-based factors, or machine-specific values.
- **Tight coupling between scorer and analyzer types:** The Scorer reads from `AnalysisResult.Metrics` via type assertion but should not import internal analyzer types. Use `pkg/types` types (C1Metrics, C3Metrics, C6Metrics) which are already defined there.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| YAML config parsing | Custom config parser | `gopkg.in/yaml.v3` | Well-tested, handles edge cases (anchors, multiline strings, comments). Struct tags match Go patterns. |
| Piecewise linear interpolation | External math library | ~20 lines of Go code in `scoring/scorer.go` | Too simple to warrant a dependency. gonum would add 15MB+ to binary. |

**Key insight:** The scoring model is pure arithmetic -- no I/O, no concurrency, no external calls. Every function is a pure function mapping inputs to outputs. This makes it trivially testable with table-driven tests.

## Common Pitfalls

### Pitfall 1: Breakpoint Ordering Confusion

**What goes wrong:** For "lower is better" metrics (complexity, duplication), the breakpoints have decreasing scores as values increase. For "higher is better" metrics (coverage, test ratio), the breakpoints have increasing scores as values increase. Mixing up the direction produces inverted scores.
**Why it happens:** Natural assumption that breakpoints always go low-value-to-high-value with low-score-to-high-score.
**How to avoid:** Always sort breakpoints by `Value` ascending. The `Score` direction (ascending or descending) encodes whether the metric is "higher is better" or "lower is better." Add a test for each metric that verifies a known-good value scores higher than a known-bad value.
**Warning signs:** Coverage of 80% scores lower than coverage of 20%.

### Pitfall 2: Composite Weight Sum != 1.0

**What goes wrong:** The requirements specify C1: 25%, C3: 20%, C6: 15%. That sums to 60%, not 100%. The remaining 40% is for future categories (C2, C4, C5, C7, etc.) not yet implemented. If you divide by 1.0, scores will be artificially low.
**Why it happens:** The weights represent the final-state allocation. With only 3 of ~7 categories implemented, the raw weighted sum will be at most 60% of the maximum.
**How to avoid:** Normalize the composite score by dividing by the sum of active category weights (0.60), not by 1.0. This way, a project scoring 10/10 on all three active categories gets a composite of 10, not 6.
**Warning signs:** Perfect codebases scoring 6.0 instead of 10.0.

### Pitfall 3: Missing or Zero Metrics

**What goes wrong:** Some metrics may not be available (e.g., coverage = -1 when no coverage file exists, single-package modules have no coupling data). If the scorer treats missing data as 0, it penalizes projects unfairly.
**Why it happens:** The C6Metrics type uses -1 for "not available" coverage, and single-package modules skip dead code detection.
**How to avoid:** Define a "not available" sentinel value (e.g., `CoveragePercent == -1`). When a metric is not available, exclude it from the category score and redistribute its weight among the remaining metrics. Document which metrics can be unavailable and what happens when they are.
**Warning signs:** Projects without coverage data score significantly lower than identical projects with a `cover.out` file showing 0% coverage.

### Pitfall 4: Non-Linear Metric Ranges

**What goes wrong:** Some metrics have very different natural ranges. Cyclomatic complexity avg might range from 1-50, while duplication rate ranges from 0-100%. If breakpoints are not carefully calibrated, one metric can dominate the category score.
**Why it happens:** Breakpoints are defined independently per metric, but their weights interact.
**How to avoid:** Each metric's breakpoints should be calibrated so that "typical Go project" values map to scores around 5-7, "excellent" maps to 8-10, and "poor" maps to 1-4. Test against known open-source Go projects to validate that scores feel reasonable.
**Warning signs:** Category scores dominated by a single metric regardless of the others.

### Pitfall 5: Tier Boundary Edge Cases

**What goes wrong:** A composite score of exactly 8.0 -- is that "Agent-Ready" (min 8.0) or "Agent-Assisted" (max 8.0)? Without clear boundary semantics, tests are flaky.
**Why it happens:** Floating-point equality and inclusive vs exclusive boundaries.
**How to avoid:** Define tier boundaries as inclusive on the lower bound: score >= 8.0 is Agent-Ready, >= 6.0 is Agent-Assisted, >= 4.0 is Agent-Limited, < 4.0 is Agent-Hostile. Document this convention and test edge cases explicitly.
**Warning signs:** Tests pass intermittently due to floating-point rounding.

## Code Examples

### Extracting Raw Metrics from AnalysisResult

```go
// Source: Derived from existing codebase types
func extractC1Metrics(ar *types.AnalysisResult) (*types.C1Metrics, bool) {
    raw, ok := ar.Metrics["c1"]
    if !ok {
        return nil, false
    }
    m, ok := raw.(*types.C1Metrics)
    return m, ok
}
```

### Computing Category Score from Sub-Scores

```go
// Source: Standard weighted average pattern
func categoryScore(subScores []SubScore) float64 {
    totalWeight := 0.0
    weightedSum := 0.0

    for _, ss := range subScores {
        if ss.Score < 0 {
            continue // skip unavailable metrics
        }
        weightedSum += ss.Score * ss.Weight
        totalWeight += ss.Weight
    }

    if totalWeight == 0 {
        return 5.0 // neutral default
    }
    return weightedSum / totalWeight
}
```

### Computing Composite Score with Weight Normalization

```go
// Source: Requirement SCORE-02 specifies C1: 25%, C3: 20%, C6: 15%
func computeComposite(categories []CategoryScore, config *ScoringConfig) float64 {
    totalWeight := 0.0
    weightedSum := 0.0

    weightMap := map[string]float64{
        "C1": config.C1.Weight,
        "C3": config.C3.Weight,
        "C6": config.C6.Weight,
    }

    for _, cat := range categories {
        w, ok := weightMap[cat.Name]
        if !ok {
            continue
        }
        weightedSum += cat.Score * w
        totalWeight += w
    }

    if totalWeight == 0 {
        return 0
    }
    return weightedSum / totalWeight
}
```

### Tier Classification

```go
// Source: Requirement SCORE-03
func classifyTier(score float64, tiers []TierConfig) string {
    // Tiers must be sorted by MinScore descending
    for _, tier := range tiers {
        if score >= tier.MinScore {
            return tier.Name
        }
    }
    return "Agent-Hostile" // fallback
}
```

### Loading YAML Config with Defaults Fallback

```go
// Source: Standard Go pattern for optional config files
func LoadConfig(path string) (*ScoringConfig, error) {
    cfg := DefaultConfig()

    if path == "" {
        return cfg, nil
    }

    data, err := os.ReadFile(path)
    if err != nil {
        return nil, fmt.Errorf("read scoring config: %w", err)
    }

    if err := yaml.Unmarshal(data, cfg); err != nil {
        return nil, fmt.Errorf("parse scoring config: %w", err)
    }

    return cfg, nil
}
```

### Table-Driven Test for Interpolation

```go
func TestInterpolate(t *testing.T) {
    breakpoints := []Breakpoint{
        {Value: 1, Score: 10},
        {Value: 5, Score: 8},
        {Value: 10, Score: 6},
        {Value: 20, Score: 3},
        {Value: 40, Score: 1},
    }

    tests := []struct {
        name  string
        value float64
        want  float64
    }{
        {"below first", 0, 10.0},
        {"exact first", 1, 10.0},
        {"midpoint 1-5", 3, 9.0},
        {"exact second", 5, 8.0},
        {"midpoint 5-10", 7.5, 7.0},
        {"exact third", 10, 6.0},
        {"above last", 50, 1.0},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := Interpolate(breakpoints, tt.value)
            if math.Abs(got-tt.want) > 0.01 {
                t.Errorf("Interpolate(%v) = %v, want %v", tt.value, got, tt.want)
            }
        })
    }
}
```

## Metric-to-Score Mapping: Default Breakpoints

These are the recommended default breakpoints for each metric. They should be calibrated against real Go projects but serve as a starting point.

### C1: Code Health (Weight: 25%)

| Metric | Raw Values | Score Range | Direction | Weight |
|--------|-----------|-------------|-----------|--------|
| Complexity avg | 1-40+ | 10-1 | Lower is better | 25% |
| Function length avg | 5-100+ lines | 10-1 | Lower is better | 20% |
| File size avg | 50-1000+ lines | 10-1 | Lower is better | 15% |
| Afferent coupling avg | 0-20+ | 10-1 | Lower is better | 15% |
| Efferent coupling avg | 0-20+ | 10-1 | Lower is better | 10% |
| Duplication rate | 0-50%+ | 10-1 | Lower is better | 15% |

### C3: Architecture (Weight: 20%)

| Metric | Raw Values | Score Range | Direction | Weight |
|--------|-----------|-------------|-----------|--------|
| Max directory depth | 1-10+ | 10-1 | Lower is better | 20% |
| Module fanout avg | 0-15+ | 10-1 | Lower is better | 20% |
| Circular deps count | 0-10+ | 10-1 | Lower is better | 25% |
| Import complexity avg | 1-8+ | 10-1 | Lower is better | 15% |
| Dead exports count | 0-50+ | 10-1 | Lower is better | 20% |

### C6: Testing (Weight: 15%)

| Metric | Raw Values | Score Range | Direction | Weight |
|--------|-----------|-------------|-----------|--------|
| Test-to-code ratio | 0-2.0+ | 1-10 | Higher is better | 25% |
| Coverage percent | 0-100% | 1-10 | Higher is better | 30% |
| Test isolation | 0-100% | 1-10 | Higher is better | 15% |
| Assertion density avg | 0-10+ | 1-10 | Higher is better (up to a point) | 15% |
| Test file ratio | 0-1.0+ | 1-10 | Higher is better | 15% |

Note: Coverage may be unavailable (-1), in which case its weight redistributes to other C6 metrics.

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Hardcoded thresholds in code | Config file with struct defaults | Industry standard | Enables tuning without recompilation |
| Single overall score | Per-category + composite | Standard in code quality tools | Better actionability -- users see which area needs work |
| Boolean pass/fail | 1-10 continuous score with tiers | Standard in maturity models | Captures gradations, not just binary |

**Deprecated/outdated:**
- N/A -- scoring models are a mature domain with no recent paradigm shifts.

## Open Questions

1. **Exact breakpoint calibration values**
   - What we know: The breakpoint values in the default config need calibrating against real Go projects.
   - What's unclear: What "typical" values look like across a range of Go project sizes and quality levels.
   - Recommendation: Start with reasonable estimates based on common Go community standards (e.g., complexity avg of 5 is good, 10 is moderate, 20 is poor). Run Phase 3 output against this repository and 2-3 known open-source Go projects. Adjust in Phase 5 hardening if needed. The YAML config makes this easy to tune.

2. **Coupling metric aggregation**
   - What we know: C1Metrics stores coupling as `map[string]int` (per-package). The scorer needs a single number.
   - What's unclear: Whether to use average, max, or some other aggregation across packages.
   - Recommendation: Use average afferent and average efferent coupling as the raw values. Max coupling could be a verbose detail. This matches how MetricSummary works for other metrics.

3. **Future category weight allocation**
   - What we know: C1(25%) + C3(20%) + C6(15%) = 60% of the final model. Categories C2, C4, C5, C7 are not yet implemented.
   - What's unclear: Whether future categories will change existing weights.
   - Recommendation: Normalize to 100% across active categories for now (divide by sum of active weights). The YAML config makes it trivial to adjust weights when new categories are added.

## Sources

### Primary (HIGH confidence)
- Existing codebase: `pkg/types/types.go` -- C1Metrics, C3Metrics, C6Metrics struct definitions
- Existing codebase: `internal/analyzer/*.go` -- how metrics are computed and stored
- Existing codebase: `internal/output/terminal.go` -- current rendering pattern
- Existing codebase: `internal/pipeline/pipeline.go` -- pipeline stage pattern
- Requirements: SCORE-01 through SCORE-06 in ROADMAP.md

### Secondary (MEDIUM confidence)
- [sgreben/piecewiselinear](https://github.com/sgreben/piecewiselinear) -- confirmed that piecewise linear interpolation is simple enough to implement inline
- [gonum interp](https://pkg.go.dev/gonum.org/v1/gonum/interp) -- confirmed gonum is overkill for this use case
- [gopkg.in/yaml.v3](https://pkg.go.dev/gopkg.in/yaml.v3) -- standard Go YAML library

### Tertiary (LOW confidence)
- Default breakpoint values -- based on general Go community standards and reasoning about typical project metrics, not empirical calibration data

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- pure Go stdlib + optional yaml.v3; no exotic dependencies
- Architecture: HIGH -- follows existing pipeline pattern with a clean new package; types already defined
- Scoring math: HIGH -- piecewise linear interpolation is textbook; weighted average is trivial
- Default breakpoints: LOW -- values need empirical calibration; starting estimates are reasonable but unvalidated
- Pitfalls: HIGH -- weight normalization and missing-metric handling are real issues with clear solutions

**Research date:** 2026-01-31
**Valid until:** 2026-06-01 (stable domain; scoring models do not change frequently)
