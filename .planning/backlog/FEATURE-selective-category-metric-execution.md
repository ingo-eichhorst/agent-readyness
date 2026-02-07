# Feature Request: Selective Category/Metric Execution

**Type**: Enhancement
**Priority**: Medium
**Complexity**: Medium
**Status**: Backlog
**Created**: 2026-02-07

## Summary

Add CLI flags to run analysis for specific categories or metrics only, rather than always running all 7 categories. This enables faster iteration, targeted debugging, cost control, and flexible CI/CD pipelines.

## Motivation

Currently, `ars scan` always runs all 7 categories (C1-C7) and computes scores for all metrics within each category. This has several limitations:

1. **Slow iteration cycles** - When improving a specific category (e.g., C1 code quality), users must wait for all categories to complete
2. **Wasted resources** - CI/CD pipelines that only care about specific metrics (e.g., test coverage) pay for full scans
3. **Cost concerns** - C7 agent evaluation uses LLM API calls; users may want to run it separately from static analysis
4. **Debugging friction** - When troubleshooting a specific metric, full scans generate noise and take longer

## User Stories

### Story 1: Developer Iterating on Code Quality
**As a** developer refactoring complex functions
**I want to** run only C1 (Code Quality) analysis
**So that** I can get instant feedback without waiting for C2-C7

```bash
# Current: Must run all categories (~30s)
ars scan . --json | jq '.categories[] | select(.name == "C1")'

# Desired: Run only C1 (~2s)
ars scan . --categories C1
```

### Story 2: CI/CD Gating on Test Coverage
**As a** DevOps engineer
**I want to** fail CI if test coverage drops below threshold
**So that** pull requests maintain quality without paying for full scans

```bash
# Current: Must run all categories, extract one metric
ars scan . --json --threshold 8.0  # Fails on composite, not coverage

# Desired: Run only C6, gate on specific metric
ars scan . --categories C6 --threshold-metric coverage_percent:80
```

### Story 3: Cost-Conscious LLM Usage
**As a** project maintainer
**I want to** run C7 separately from C1-C6
**So that** I can control when LLM API costs are incurred

```bash
# Morning: Quick static analysis (free)
ars scan . --exclude-categories C7 --output-html daily-report.html

# Weekly: Full analysis with live agent eval (costs ~$0.50)
ars scan . --categories C7 --baseline daily-report.json
```

### Story 4: Debugging a Specific Metric
**As a** developer investigating a low score
**I want to** run only the metrics I'm debugging
**So that** I can iterate quickly with verbose output

```bash
# Debug why complexity is high
ars scan . --metrics complexity_avg --verbose --debug

# Compare before/after refactoring a single metric
ars scan . --metrics func_length_avg --baseline before.json
```

## Proposed CLI Flags

### Option 1: Category Selection

```bash
# Run specific categories (comma-separated)
ars scan . --categories C1,C3,C6

# Exclude specific categories
ars scan . --exclude-categories C7

# Run all static analysis (C1-C6), skip LLM
ars scan . --exclude-categories C4,C7
# Equivalent to --no-llm but more explicit
```

### Option 2: Metric Selection

```bash
# Run specific metrics (comma-separated, any category)
ars scan . --metrics complexity_avg,test_to_code_ratio

# List available metrics
ars scan . --list-metrics
```

### Option 3: Category + Metric Filtering (Advanced)

```bash
# Run all C1 metrics except duplication
ars scan . --categories C1 --exclude-metrics duplication_rate

# Run only coverage and test metrics from C6
ars scan . --categories C6 --metrics coverage_percent,test_to_code_ratio
```

## Technical Design

### 1. Flag Parsing

Add new fields to `cmd/scan.go`:

```go
var (
    // ... existing flags
    categories        []string  // --categories flag
    excludeCategories []string  // --exclude-categories flag
    metrics           []string  // --metrics flag
    excludeMetrics    []string  // --exclude-metrics flag
    listMetrics       bool      // --list-metrics flag
)
```

### 2. Pipeline Filtering

Extend `internal/pipeline/pipeline.go`:

```go
// FilterConfig specifies which analyzers and metrics to run
type FilterConfig struct {
    Categories        []string // If empty, run all
    ExcludeCategories []string
    Metrics           []string // If empty, run all metrics in selected categories
    ExcludeMetrics    []string
}

// New creates a Pipeline with optional filtering
func New(w io.Writer, verbose bool, cfg *scoring.ScoringConfig,
         threshold float64, jsonOutput bool, onProgress ProgressFunc,
         filter *FilterConfig) *Pipeline
```

### 3. Analyzer Filtering

Modify analyzer execution in `Pipeline.Run()`:

```go
// Stage 3: Analyze packages (with filtering)
for _, a := range p.analyzers {
    // Skip if category excluded
    if shouldSkipCategory(a.Name(), filter) {
        continue
    }

    // Run analyzer
    result, err := a.Analyze(targets)

    // Filter metrics if specified
    if filter != nil && len(filter.Metrics) > 0 {
        result = filterMetrics(result, filter)
    }

    // ... rest of analysis
}
```

### 4. Score Recalculation

Update `internal/scoring/scorer.go` to handle partial results:

```go
// Score computes composite score from available categories
// Weight redistribution: If C7 is missing, its 10% weight is redistributed
// proportionally across remaining categories
func (s *Scorer) Score(results []*types.AnalysisResult) (*types.ScoredResult, error) {
    availableCategories := getAvailableCategories(results)
    adjustedWeights := redistributeWeights(s.Config, availableCategories)

    // Compute weighted average using only available categories
    // ... scoring logic
}
```

### 5. Output Handling

Modify output renderers to handle partial results:

```go
// Terminal output shows "Skipped" for excluded categories
C1: Code Quality          8.5 / 10
C2: Semantics            7.2 / 10
C3: Architecture         6.8 / 10
C4: Documentation        Skipped (--exclude-categories)
C5: Temporal Dynamics    Skipped (--exclude-categories)
C6: Testing              9.1 / 10
C7: Agent Evaluation     Skipped (--exclude-categories)
────────────────────────────────────
Composite Score:         7.9 / 10 (partial: C1,C2,C3,C6)
```

## Implementation Phases

### Phase 1: Category Filtering (MVP)
**Effort**: 1-2 days
**Deliverables**:
- `--categories` flag (whitelist)
- `--exclude-categories` flag (blacklist)
- Weight redistribution in scoring
- Terminal/JSON output shows partial results
- Tests for filtering logic

**Exit Criteria**:
- `ars scan . --categories C1,C6` runs only C1 and C6
- Composite score excludes missing categories
- `--exclude-categories C7` equivalent to `--no-llm` for C7

### Phase 2: Metric Filtering
**Effort**: 2-3 days
**Deliverables**:
- `--metrics` flag
- `--exclude-metrics` flag
- Per-category metric filtering
- `--list-metrics` command

**Exit Criteria**:
- `ars scan . --metrics complexity_avg,coverage_percent` runs only those metrics
- Score calculation handles missing metrics within categories
- `ars scan . --list-metrics` shows all available metrics with descriptions

### Phase 3: Advanced Filtering
**Effort**: 1-2 days
**Deliverables**:
- Combined category + metric filtering
- Validation and error messages
- Documentation and examples

**Exit Criteria**:
- `ars scan . --categories C1 --exclude-metrics duplication_rate` works correctly
- Clear error messages for invalid category/metric names
- README examples for common use cases

## Edge Cases & Considerations

### 1. Empty Results
**Problem**: What if filtering excludes all categories?
**Solution**: Return error: `"No categories selected. Use --categories or remove --exclude-categories"`

### 2. Composite Score with Partial Data
**Problem**: How to calculate composite when C7 (10% weight) is missing?
**Solution**: Redistribute C7's weight proportionally: C1-C6 each gain +1.67% weight

### 3. Baseline Comparison
**Problem**: Comparing partial scan to full baseline
**Solution**: Compare only categories present in both. Show warning if categories differ.

### 4. HTML Report with Partial Data
**Problem**: Missing categories break chart rendering
**Solution**: Render only available categories, add note about excluded categories

### 5. --threshold with Partial Scans
**Problem**: Composite threshold doesn't make sense for partial scans
**Solution**: Add `--threshold-metric` flag for gating on specific metrics

## Dependencies

- None (standalone feature)

## Breaking Changes

- None (all flags are opt-in)

## Documentation Updates

1. **README.md** - Add examples for selective execution
2. **CLI Help** - Document new flags
3. **RESEARCH.md** - Note that selective execution may affect score interpretation

## Success Metrics

1. **Performance**: `ars scan . --categories C1` completes in <10% time of full scan
2. **Adoption**: >20% of scans use selective execution within 1 month of release
3. **CI/CD Integration**: Users report using `--categories` in build pipelines
4. **Cost Savings**: Users avoid C7 LLM costs on routine scans

## Alternatives Considered

### Alternative 1: Post-Scan Filtering
**Approach**: Always run all categories, filter output
**Rejected**: Doesn't solve performance or cost issues

### Alternative 2: Separate CLI Commands
**Approach**: `ars scan-c1`, `ars scan-c7`, etc.
**Rejected**: Creates maintenance burden, inconsistent UX

### Alternative 3: Config File Filtering
**Approach**: `.arsrc.yml` specifies which categories to run
**Rejected**: Less flexible than CLI flags for one-off scans

## Future Enhancements

1. **Category Groups**: `--categories static` (C1-C3,C5-C6) or `--categories llm` (C4,C7)
2. **Metric Regex**: `--metrics "coverage_.*"` to match multiple metrics
3. **Watch Mode**: `ars scan . --categories C1 --watch` for live feedback during refactoring
4. **Parallel Execution**: Run different category subsets in parallel for faster CI

## References

- Similar features in other tools:
  - ESLint: `--rule` flag to run specific rules
  - pytest: `-k` flag for test selection
  - golangci-lint: `--disable` flag for linters

## Tasks

- [ ] Design flag parsing and validation logic
- [ ] Implement FilterConfig and pipeline filtering
- [ ] Update weight redistribution in Scorer
- [ ] Handle partial results in terminal output
- [ ] Handle partial results in JSON output
- [ ] Handle partial results in HTML output
- [ ] Add `--list-metrics` command
- [ ] Write tests for category filtering
- [ ] Write tests for metric filtering
- [ ] Write tests for weight redistribution
- [ ] Update README with examples
- [ ] Update CLI help text
- [ ] Add integration tests for CI/CD use cases
