# Phase 30: Evidence Data Flow - Research

**Researched:** 2026-02-06
**Domain:** Go pipeline extension -- adding evidence data to scoring types and JSON output
**Confidence:** HIGH

## Summary

This phase extends the ARS scoring pipeline to carry evidence data (top-5 worst offenders per metric) from the extractCx functions through to JSON output. The research focused on understanding the current codebase architecture, identifying exactly where changes are needed, what data is already available in each category's metrics structs, and what the JSON schema change implications are.

The primary challenge is not technical complexity but rather coordination: the change touches 5 files across 3 packages (types, scoring, output), requires modifying all 7 extractCx functions, and must maintain backward compatibility for both terminal output and baseline JSON loading. The existing codebase already stores rich per-file/per-function data in the CxMetrics structs (C1 has `Functions []FunctionMetric`, C3 has `DeadExports []DeadExport`, C5 has `TopHotspots []FileChurn`, C6 has `TestFunctions []TestFunctionMetric`), so evidence extraction is mostly sorting and selecting from data that already exists.

A critical finding is that the success criterion's jq path `.categories[0].sub_scores[0].evidence` implies a JSON schema change: the current JSON uses `"metrics"` as the field name for per-metric data within categories, but the target schema uses `"sub_scores"`. Additionally, the current JSON only includes per-metric data in verbose mode, but the success criterion implies evidence should be present in non-verbose JSON output. This needs careful handling for backward compatibility.

**Primary recommendation:** Add an `Evidence []EvidenceItem` field to `SubScore`, extend `MetricExtractor` to return evidence alongside raw values, and rename the JSON field from `"metrics"` to `"sub_scores"` while always populating sub_scores in JSON output (not gated by verbose flag).

## Standard Stack

This phase uses no new external libraries. All changes are internal to the existing Go codebase.

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `encoding/json` | stdlib | JSON serialization with struct tags | Already used throughout output package |
| `sort` | stdlib | Sorting offenders by severity | Already used in terminal.go for top-5 rendering |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `testing` | stdlib | Unit tests for new evidence extraction | All test files |

### Alternatives Considered
None -- this is pure internal Go refactoring with no external dependencies.

## Architecture Patterns

### Recommended Change Architecture

The changes flow through 3 layers:

```
Layer 1: Types (pkg/types/)
  - Add EvidenceItem struct
  - Add Evidence field to SubScore

Layer 2: Scoring (internal/scoring/)
  - Extend MetricExtractor signature to return evidence
  - Update all 7 extractCx functions
  - Wire evidence through scoreMetrics()

Layer 3: Output (internal/output/)
  - Add Evidence field to JSONMetric
  - Rename JSON field from "metrics" to "sub_scores"
  - Always populate sub_scores in JSON (not verbose-gated)
  - Ensure terminal output is unchanged
```

### Current Data Flow (Before)

```
Analyzer.Analyze() -> AnalysisResult{Metrics: {"c1": *C1Metrics}}
                           |
                           v
extractC1(ar) -> (rawValues map[string]float64, unavailable map[string]bool)
                           |
                           v
scoreMetrics() -> []SubScore{MetricName, RawValue, Score, Weight, Available}
                           |
                           v
BuildJSONReport() -> JSONCategory{Metrics: []JSONMetric} (verbose only)
```

### Target Data Flow (After)

```
Analyzer.Analyze() -> AnalysisResult{Metrics: {"c1": *C1Metrics}}  [UNCHANGED]
                           |
                           v
extractC1(ar) -> (rawValues, unavailable, evidence map[string][]EvidenceItem)
                           |
                           v
scoreMetrics() -> []SubScore{..., Evidence: []EvidenceItem}
                           |
                           v
BuildJSONReport() -> JSONCategory{SubScores: []JSONMetric{..., Evidence}} (always)
```

### Pattern 1: Evidence Extraction per Category

**What:** Each extractCx function examines the existing CxMetrics data structures and selects the top-5 worst offenders for each metric. The data is already there; extraction is sorting + slicing.

**When to use:** Every extractCx function must return evidence.

**Key insight:** C1, C3, C5, C6 already have per-item data (functions, dead exports, hotspots, test functions). C2 and C4 are aggregate-only -- they need a different approach (file-level evidence or empty arrays for purely aggregate metrics).

### Pattern 2: Three-Return MetricExtractor

**What:** Change `MetricExtractor` from returning `(rawValues, unavailable)` to returning `(rawValues, unavailable, evidence)`. The evidence map keys match the metric names (same keys as rawValues).

**Why:** This keeps the extraction co-located with the raw value computation, which is the natural place to know which items are worst offenders.

### Pattern 3: JSON Schema Migration with Backward Compatibility

**What:** The JSON field name changes from `"metrics"` to `"sub_scores"`, and sub_scores are always populated (not gated by verbose flag). Evidence uses empty arrays (never null) per the CONTEXT.md decision. Old baselines that don't have `sub_scores` still load fine because `loadBaseline()` only reads top-level category fields.

**Why:** The success criterion jq path `.categories[0].sub_scores[0].evidence` requires this exact field name. The `loadBaseline()` function in pipeline.go only extracts `Name`, `Score`, `Weight` from categories, so it never reads `sub_scores`/`metrics` -- backward compatibility is maintained.

### Anti-Patterns to Avoid

- **Storing evidence in AnalysisResult:** Evidence belongs in SubScore (the scoring layer), not AnalysisResult (the analysis layer). The analysis data stays unchanged; evidence is a scoring-layer concern that selects from analysis data.
- **Making evidence configurable (top-N):** The CONTEXT.md decision locks this at top-5. Do not add configuration.
- **Gating evidence behind verbose flag:** Evidence must always flow through the pipeline. It may be hidden in terminal output, but JSON always includes it.
- **Using `omitempty` on the Evidence field:** The CONTEXT.md decision says "No offenders: Return empty array []". The field must always be present. Use `json:"evidence"` (no omitempty).

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Sorting offenders | Custom sort per metric | `sort.Slice()` with metric-specific comparator | stdlib sort is battle-tested, 1-2 lines per metric |
| Evidence for aggregate metrics (C2, C4) | Fake per-file breakdowns | Empty `[]EvidenceItem{}` | Some metrics (naming_consistency, type_strictness) are global aggregates with no per-file data; empty array is the honest answer |
| JSON backward compatibility | Version negotiation / migration | `loadBaseline()` already ignores unknown fields | Go's `json.Unmarshal` silently ignores missing fields |

**Key insight:** Most evidence data already exists in CxMetrics structs. The extraction is trivial sorting. Do not re-analyze or re-compute anything.

## Common Pitfalls

### Pitfall 1: Breaking Terminal Output

**What goes wrong:** Adding evidence fields to SubScore causes terminal rendering to change (e.g., extra output lines, different formatting).
**Why it happens:** The `renderSubScores()` function in terminal.go iterates over SubScore fields. If evidence is displayed by default, terminal output changes.
**How to avoid:** Terminal rendering code must NOT display evidence. The success criterion explicitly requires "identical terminal output to v0.0.5". Evidence is invisible unless consumed by JSON or HTML.
**Warning signs:** Any change to terminal.go's renderSubScores or renderCx functions.

### Pitfall 2: JSON Schema Breakage for Baseline Loading

**What goes wrong:** Renaming `"metrics"` to `"sub_scores"` in JSON output means old JSON files have `"metrics"` and new ones have `"sub_scores"`. If baseline loading code expects the new field name, it fails on old baselines.
**Why it happens:** `loadBaseline()` in pipeline.go parses `JSONReport` and only reads category-level `Name`, `Score`, `Weight`. It never reads `Metrics`/`SubScores`. So the rename is safe for baseline comparison.
**How to avoid:** Verify `loadBaseline()` code path -- it does NOT read sub-scores from baseline JSON. Verify with a test: load v0.0.5 JSON, confirm comparison works.
**Warning signs:** `loadBaseline()` code accessing `.Metrics` or `.SubScores` fields from loaded baseline.

### Pitfall 3: Nil Evidence Arrays in JSON

**What goes wrong:** When no offenders exist for a metric, the Evidence field serializes as `null` instead of `[]`.
**Why it happens:** Go slices default to `nil`, and `json.Marshal(nil)` produces `null`. The CONTEXT.md decision requires `[]` (empty array).
**How to avoid:** Initialize Evidence as `[]EvidenceItem{}` (not nil) in all code paths. Alternatively, use a custom JSON marshaler. The simplest approach: in `scoreMetrics()`, always set `Evidence: make([]EvidenceItem, 0)` as the default.
**Warning signs:** `jq '.categories[0].sub_scores[0].evidence'` returning `null` instead of `[]`.

### Pitfall 4: C7 Evidence Confusion

**What goes wrong:** Populating evidence for C7 metrics when the CONTEXT.md says C7 uses C7DebugSample data instead.
**Why it happens:** The natural reflex is "all 7 categories get evidence". But C7's decision is explicit: return empty evidence arrays for structural consistency, use C7DebugSample for trace modals.
**How to avoid:** extractC7 returns evidence with empty arrays for all 5 MECE metrics. Remove overall_score metric entirely.
**Warning signs:** Non-empty evidence arrays for C7 metrics.

### Pitfall 5: Forgetting to Update the "verbose" Gating

**What goes wrong:** Sub-scores still only appear in JSON when verbose=true.
**Why it happens:** Current `BuildJSONReport()` has `if verbose { ... }` around sub-score population.
**How to avoid:** Remove the verbose gate for sub-scores in JSON. Always populate sub_scores. The success criterion's jq command doesn't use `--verbose`.
**Warning signs:** Running `ars scan . --json | jq '.categories[0].sub_scores'` returns `null`.

### Pitfall 6: Removing overall_score Breaks Test Assertions

**What goes wrong:** Existing tests (e.g., `TestExtractC7_ReturnsAllMetrics`) check for 6 metrics including `overall_score`. After removal, these tests fail.
**Why it happens:** The CONTEXT.md decision to remove `overall_score` entirely means tests expecting 6 C7 metrics need updating to 5.
**How to avoid:** Update all tests that reference C7 overall_score: scorer_test.go, config.go, and any test checking C7 metric counts.
**Warning signs:** Test failures mentioning "overall_score" or unexpected metric count.

## Code Examples

### EvidenceItem Type Definition

```go
// Source: New type in pkg/types/scoring.go
// EvidenceItem represents a single worst-offender for a metric.
type EvidenceItem struct {
    FilePath    string  `json:"file_path"`
    Line        int     `json:"line"`
    Value       float64 `json:"value"`
    Description string  `json:"description"`
}
```

### Updated SubScore Type

```go
// Source: Modified type in pkg/types/scoring.go
type SubScore struct {
    MetricName string         `json:"metric_name"`
    RawValue   float64        `json:"raw_value"`
    Score      float64        `json:"score"`
    Weight     float64        `json:"weight"`
    Available  bool           `json:"available"`
    Evidence   []EvidenceItem `json:"evidence"`
}
```

### Updated MetricExtractor Signature

```go
// Source: Modified type in internal/scoring/scorer.go
type MetricExtractor func(ar *types.AnalysisResult) (
    rawValues   map[string]float64,
    unavailable map[string]bool,
    evidence    map[string][]types.EvidenceItem,
)
```

### Evidence Extraction Example (C1 complexity_avg)

```go
// Source: Example for extractC1 in internal/scoring/scorer.go
func extractC1(ar *types.AnalysisResult) (map[string]float64, map[string]bool, map[string][]types.EvidenceItem) {
    raw, ok := ar.Metrics["c1"]
    if !ok {
        return nil, nil, nil
    }
    m, ok := raw.(*types.C1Metrics)
    if !ok {
        return nil, nil, nil
    }

    evidence := make(map[string][]types.EvidenceItem)

    // complexity_avg: top-5 most complex functions
    if len(m.Functions) > 0 {
        sorted := make([]types.FunctionMetric, len(m.Functions))
        copy(sorted, m.Functions)
        sort.Slice(sorted, func(i, j int) bool {
            return sorted[i].Complexity > sorted[j].Complexity
        })
        limit := 5
        if len(sorted) < limit {
            limit = len(sorted)
        }
        var items []types.EvidenceItem
        for _, f := range sorted[:limit] {
            items = append(items, types.EvidenceItem{
                FilePath:    f.File,
                Line:        f.Line,
                Value:       float64(f.Complexity),
                Description: fmt.Sprintf("%s.%s has complexity %d", f.Package, f.Name, f.Complexity),
            })
        }
        evidence["complexity_avg"] = items
    }

    // func_length_avg: top-5 longest functions
    // ... similar pattern ...

    // Ensure all metric keys have at least empty arrays
    for _, key := range []string{"complexity_avg", "func_length_avg", "file_size_avg",
        "afferent_coupling_avg", "efferent_coupling_avg", "duplication_rate"} {
        if evidence[key] == nil {
            evidence[key] = []types.EvidenceItem{}
        }
    }

    return map[string]float64{
        "complexity_avg":        m.CyclomaticComplexity.Avg,
        // ... same as before ...
    }, nil, evidence
}
```

### Updated scoreMetrics Wiring

```go
// Source: Modified function in internal/scoring/scorer.go
func scoreMetrics(catConfig CategoryConfig, rawValues map[string]float64, unavailable map[string]bool, evidence map[string][]types.EvidenceItem) ([]types.SubScore, float64) {
    var subScores []types.SubScore

    for _, mt := range catConfig.Metrics {
        rv := rawValues[mt.Name]
        ev := evidence[mt.Name]
        if ev == nil {
            ev = make([]types.EvidenceItem, 0)
        }
        ss := types.SubScore{
            MetricName: mt.Name,
            RawValue:   rv,
            Weight:     mt.Weight,
            Available:  true,
            Evidence:   ev,
        }
        // ... rest same as before ...
    }
    // ...
}
```

### Updated JSON Output Types

```go
// Source: Modified types in internal/output/json.go
type JSONCategory struct {
    Name      string       `json:"name"`
    Score     float64      `json:"score"`
    Weight    float64      `json:"weight"`
    SubScores []JSONMetric `json:"sub_scores"` // renamed from "metrics", always populated
}

type JSONMetric struct {
    Name      string               `json:"name"`
    RawValue  float64              `json:"raw_value"`
    Score     float64              `json:"score"`
    Weight    float64              `json:"weight"`
    Available bool                 `json:"available"`
    Evidence  []types.EvidenceItem `json:"evidence"` // no omitempty
}
```

## Data Availability per Category

This is critical for the planner -- it shows what per-item data exists for evidence extraction:

### C1: Code Health (6 metrics, rich per-item data)
| Metric | Source Data | Evidence Strategy |
|--------|------------|-------------------|
| `complexity_avg` | `C1Metrics.Functions []FunctionMetric` | Sort by `.Complexity` desc, take top 5 |
| `func_length_avg` | `C1Metrics.Functions []FunctionMetric` | Sort by `.LineCount` desc, take top 5 |
| `file_size_avg` | Only `MetricSummary{Avg, Max, MaxEntity}` | MaxEntity gives 1 offender; need file-level data from analyzer. Use MaxEntity as single evidence item, or extract from Go packages. |
| `afferent_coupling_avg` | `C1Metrics.AfferentCoupling map[string]int` | Sort map by value desc, take top 5 packages |
| `efferent_coupling_avg` | `C1Metrics.EfferentCoupling map[string]int` | Sort map by value desc, take top 5 packages |
| `duplication_rate` | `C1Metrics.DuplicatedBlocks []DuplicateBlock` | Sort by `.LineCount` desc, take top 5 blocks |

### C2: Semantic Explicitness (5 metrics, mostly aggregate)
| Metric | Source Data | Evidence Strategy |
|--------|------------|-------------------|
| `type_annotation_coverage` | Aggregate percentage only | Empty array (no per-file data available) |
| `naming_consistency` | Aggregate percentage only | Empty array (no per-identifier data stored) |
| `magic_number_ratio` | Aggregate count only | Empty array (individual magic numbers not stored) |
| `type_strictness` | Boolean 0/1 | Empty array (global setting, no offenders) |
| `null_safety` | Aggregate percentage only | Empty array (no per-usage data stored) |

### C3: Architecture (5 metrics, good per-item data)
| Metric | Source Data | Evidence Strategy |
|--------|------------|-------------------|
| `max_dir_depth` | Integer only | Empty array (no per-directory data stored). Could use MaxEntity if available. |
| `module_fanout_avg` | `MetricSummary` only | Single MaxEntity item if available. |
| `circular_deps` | `C3Metrics.CircularDeps [][]string` | Top 5 cycles, describe as "A -> B -> C -> A" |
| `import_complexity_avg` | `MetricSummary` only | Single MaxEntity item if available. |
| `dead_exports` | `C3Metrics.DeadExports []DeadExport` | Top 5 dead exports with file, line, kind |

### C4: Documentation Quality (7 metrics, mostly boolean/aggregate)
| Metric | Source Data | Evidence Strategy |
|--------|------------|-------------------|
| `readme_word_count` | Integer count | Empty array (single global metric) |
| `comment_density` | Aggregate percentage | Empty array (no per-file data stored) |
| `api_doc_coverage` | Aggregate percentage | Empty array (no per-function data stored) |
| `changelog_present` | Boolean | Empty array |
| `examples_present` | Boolean | Empty array |
| `contributing_present` | Boolean | Empty array |
| `diagrams_present` | Boolean | Empty array |

### C5: Temporal Dynamics (5 metrics, good per-item data)
| Metric | Source Data | Evidence Strategy |
|--------|------------|-------------------|
| `churn_rate` | Aggregate only | Use TopHotspots (top churning files) |
| `temporal_coupling_pct` | `C5Metrics.CoupledPairs []CoupledPair` | Top 5 coupled pairs |
| `author_fragmentation` | Aggregate only | Use TopHotspots by author count |
| `commit_stability` | Aggregate only | Empty array (no per-file stability stored) |
| `hotspot_concentration` | `C5Metrics.TopHotspots []FileChurn` | Top 5 hotspots (already sorted by churn) |

### C6: Testing (5 metrics, good per-item data)
| Metric | Source Data | Evidence Strategy |
|--------|------------|-------------------|
| `test_to_code_ratio` | Aggregate ratio | Empty array (global metric) |
| `coverage_percent` | Aggregate percentage | Empty array (no per-file coverage in struct) |
| `test_isolation` | Aggregate percentage | Filter TestFunctions by HasExternalDep=true, take top 5 |
| `assertion_density_avg` | `C6Metrics.TestFunctions []TestFunctionMetric` | Sort by `.AssertionCount` asc, take top 5 (lowest assertion density) |
| `test_file_ratio` | Aggregate ratio | Empty array (global metric) |

### C7: Agent Evaluation (5 MECE metrics + removal of overall_score)
| Metric | Evidence Strategy |
|--------|-------------------|
| `task_execution_consistency` | Empty array (per CONTEXT.md decision) |
| `code_behavior_comprehension` | Empty array |
| `cross_file_navigation` | Empty array |
| `identifier_interpretability` | Empty array |
| `documentation_accuracy_detection` | Empty array |
| `overall_score` | REMOVED entirely (per CONTEXT.md decision) |

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| JSON metrics only in verbose mode | Sub_scores always in JSON | Phase 30 | Non-verbose JSON becomes much more useful |
| `"metrics"` JSON field name | `"sub_scores"` JSON field name | Phase 30 | Success criterion requires this exact path |
| `overall_score` in C7 (zero weight) | Removed entirely | Phase 30 | Cleaner C7 with exactly 5 MECE metrics |
| Evidence not tracked | Top-5 evidence per metric | Phase 30 | Foundation for Phase 32 (trace modals) and Phase 33 (improvement prompts) |

**Deprecated/outdated:**
- `overall_score` in C7 config: Removed entirely (not just zero-weight). Config, extractor, and tests all need updating.
- Verbose-gated sub-scores in JSON: Sub-scores always present. The `verbose` parameter to `BuildJSONReport` may be removed or repurposed (could control additional detail level).

## Open Questions

1. **JSON field rename backward compatibility with external consumers**
   - What we know: `loadBaseline()` only reads category-level fields, so ARS-to-ARS comparison is safe. The jq path in the success criterion requires `sub_scores`.
   - What's unclear: Whether any external tools consume the `"metrics"` field name from ARS JSON output.
   - Recommendation: Proceed with rename. The tool is pre-1.0 and the roadmap explicitly calls for this change. Bump JSON version from "1" to "2" as a courtesy signal.

2. **Whether `verbose` flag should still control some JSON detail**
   - What we know: Sub-scores must always be in JSON (success criterion). Evidence must always be present (CONTEXT.md).
   - What's unclear: Whether `verbose` should control anything else (e.g., extended descriptions in evidence items).
   - Recommendation: Remove the verbose gate entirely for sub-scores and evidence in JSON. The `verbose` flag can remain for terminal output detail level.

3. **File-level evidence for MetricSummary-only metrics (file_size_avg, module_fanout_avg)**
   - What we know: These metrics only store Avg/Max/MaxEntity, not per-file breakdowns.
   - What's unclear: Whether we should extract per-file data from the raw analyzer packages or accept single-item evidence.
   - Recommendation: Use MaxEntity as a single evidence item where available. Per-file breakdown would require passing analyzer data into the scoring layer, which crosses the current architecture boundary. A single "worst offender" is better than nothing and avoids architectural violation.

## Specific Changes by File

### `pkg/types/scoring.go`
- Add `EvidenceItem` struct with `FilePath`, `Line`, `Value`, `Description` (all with json tags)
- Add `Evidence []EvidenceItem` field to `SubScore` struct (json tag: `"evidence"`)
- Add json tags to ALL existing SubScore fields (currently has none -- needed for JSON output consistency)

### `internal/scoring/scorer.go`
- Change `MetricExtractor` type to three-return signature
- Update all 7 extractCx functions to return evidence map
- Update `scoreMetrics()` to accept and wire evidence
- Update `Score()` method to pass evidence through
- Remove `overall_score` from extractC7 and its unavailable set
- Update `metricExtractors` map (no change needed -- functions just get new return)

### `internal/scoring/config.go`
- Remove `overall_score` MetricThresholds from C7 category config

### `internal/output/json.go`
- Rename `JSONCategory.Metrics` field to `SubScores` with tag `json:"sub_scores"`
- Add `Evidence []types.EvidenceItem` to `JSONMetric` with tag `json:"evidence"`
- In `BuildJSONReport()`: always populate sub_scores (remove verbose gate)

### `internal/output/terminal.go`
- No changes to rendering (evidence invisible in terminal)
- `renderSubScores()` already skips zero-weight metrics, no change needed

### Test files
- `internal/scoring/scorer_test.go`: Update C7 tests (5 metrics instead of 6)
- `internal/output/json_test.go`: Update for new field names and always-present sub_scores

## Sources

### Primary (HIGH confidence)
- Direct codebase analysis of all files in `internal/scoring/`, `internal/output/`, `pkg/types/`
- Current `scorer.go` lines 12-14: MetricExtractor signature
- Current `scorer.go` lines 176-376: All 7 extractCx functions
- Current `json.go` lines 22-28: JSONCategory and JSONMetric types
- Current `json.go` lines 56-108: BuildJSONReport function
- Current `scoring.go` (pkg/types) lines 1-27: SubScore and CategoryScore types
- Current `types.go` (pkg/types) lines 122-332: All CxMetrics types with per-item data

### Secondary (MEDIUM confidence)
- ROADMAP.md Phase 30 description and success criteria
- CONTEXT.md decisions on evidence selection, C7 handling, and fallback behavior

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - All changes are internal Go code, no external dependencies
- Architecture: HIGH - Direct analysis of current code reveals exact change points
- Pitfalls: HIGH - Based on specific code paths (loadBaseline, renderSubScores, json tags)
- Data availability: HIGH - Verified by reading each CxMetrics struct and analyzer source

**Research date:** 2026-02-06
**Valid until:** 2026-03-06 (stable -- internal codebase, no external dependency drift)
