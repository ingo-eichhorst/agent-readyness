# Phase 34: Testing & Quality - Research

**Researched:** 2026-02-07
**Domain:** Go testing patterns for evidence extraction, HTML output validation, JSON backward compatibility, prompt template coverage, accessibility
**Confidence:** HIGH

## Summary

This phase adds automated tests that validate features built in phases 30-33 across six requirements (TEST-01 through TEST-06). The codebase already has extensive test patterns to follow -- colocated `*_test.go` files, table-driven tests, `bytes.Buffer` for capturing output, and helper functions like `buildAllCategoriesScoredResult()`.

The research focused on understanding the exact structures, functions, and patterns needed for each test requirement. All test data can be constructed from existing types (`types.ScoredResult`, `types.CategoryScore`, `types.SubScore`, `types.EvidenceItem`) without needing external fixtures. The HTML template already has some ARIA attributes (one `aria-label="Close"` on the modal close button) and responsive CSS (`@media (max-width: 640px)`), so accessibility tests validate existing markup patterns.

**Primary recommendation:** Follow existing test patterns exactly. Use `buildAllCategoriesScoredResult()` as a base for evidence and prompt tests. Use `bytes.Buffer` + `gen.GenerateReport()` for HTML validation. Use `json.Unmarshal` round-trips for JSON compatibility.

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `testing` | stdlib | Go test framework | Only option for Go tests |
| `bytes` | stdlib | Buffer for capturing output | Used throughout existing tests |
| `strings` | stdlib | HTML content assertions | Used throughout existing tests |
| `encoding/json` | stdlib | JSON round-trip validation | Already used in json_test.go |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `regexp` | stdlib | Pattern matching in HTML | For ARIA attribute validation |
| `math` | stdlib | Float comparison | For score threshold assertions |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| strings.Contains | goquery/colly | HTML parsing is more robust but adds dependency; strings.Contains matches existing pattern |
| testify | stdlib testing | Would be more expressive but project uses stdlib throughout |

**Installation:** No additional dependencies needed. All tests use stdlib only.

## Architecture Patterns

### Test File Placement
Tests go in the same package as the code they test (white-box testing pattern used throughout codebase):

```
internal/
  output/
    html_test.go       # existing - ADD evidence/size/accessibility tests here
    json_test.go       # existing - ADD backward compatibility test here
    prompt_test.go     # existing - ADD prompt coverage test here
  scoring/
    scorer_test.go     # existing - ADD evidence extraction tests here
```

### Pattern 1: Synthetic ScoredResult Construction
**What:** Build complete `types.ScoredResult` with all 7 categories and all 38 metrics for test assertions.
**When to use:** Evidence extraction, prompt coverage, HTML generation tests.
**Example:**
```go
// Source: existing buildAllCategoriesScoredResult() in html_test.go
// Extended to include Evidence on every SubScore
func buildFullScoredResult() *types.ScoredResult {
    evidence := []types.EvidenceItem{
        {FilePath: "internal/foo.go", Line: 42, Value: 15.0, Description: "test evidence"},
    }
    // ... all 7 categories with all metrics, each with evidence
}
```

### Pattern 2: HTML Buffer Capture + strings.Contains
**What:** Generate HTML report into `bytes.Buffer`, then assert substrings.
**When to use:** All HTML validation tests (file size, ARIA, responsive, prompts).
**Example:**
```go
// Source: existing TestHTMLGenerator_GenerateReport in html_test.go
gen, _ := NewHTMLGenerator()
var buf bytes.Buffer
gen.GenerateReport(&buf, scored, nil, nil, trace)
html := buf.String()
if !strings.Contains(html, `aria-label="Close"`) { t.Error("missing ARIA") }
```

### Pattern 3: JSON Round-Trip with Old Schema
**What:** Unmarshal v1 JSON (with `"metrics"` field name) into current `JSONReport` struct.
**When to use:** Backward compatibility test.
**Example:**
```go
// Source: existing TestJSONBaselineBackwardCompatibility in json_test.go
oldJSON := `{"version": "1", "composite_score": 7.5, ...}`
var report JSONReport
json.Unmarshal([]byte(oldJSON), &report)
// Assert category-level fields load correctly
```

### Pattern 4: Metric Count Validation via ScoringConfig
**What:** Iterate `scoring.DefaultConfig().Categories` to get authoritative metric list.
**When to use:** Prompt template coverage test (TEST-04).
**Example:**
```go
cfg := scoring.DefaultConfig()
var allMetrics []string
for _, cat := range cfg.Categories {
    for _, m := range cat.Metrics {
        allMetrics = append(allMetrics, m.Name)
    }
}
// Assert len(allMetrics) == 38
// Assert each metric has a prompt template in generated HTML
```

### Anti-Patterns to Avoid
- **External file I/O in tests:** Do not write HTML to temp files and read back. Use `bytes.Buffer` in-memory.
- **Hardcoded metric counts:** Use `scoring.DefaultConfig()` as the source of truth for the 38-metric count, not a hardcoded number.
- **Substring matching for structured data:** For JSON tests, unmarshal and check fields rather than string matching.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| HTML parsing | Custom parser | `strings.Contains` / `strings.Count` | Matches existing test patterns; full HTML parsing unnecessary |
| Metric enumeration | Hardcoded list of 38 names | `scoring.DefaultConfig()` iteration | Single source of truth; auto-updates if metrics change |
| Test data construction | External JSON fixtures | In-code `types.ScoredResult` construction | Existing pattern (`buildAllCategoriesScoredResult`); type-safe |

## Common Pitfalls

### Pitfall 1: Evidence May Be Empty for "Clean" Categories
**What goes wrong:** Expecting non-empty evidence for all 7 categories when scanning a healthy codebase.
**Why it happens:** Some metrics produce evidence only when violations exist (e.g., `circular_deps` evidence is empty when no cycles found).
**How to avoid:** For TEST-01, construct synthetic `AnalysisResult` with known violations, not rely on scanning real code. Alternatively, test that evidence is `[]` (not `nil`) and non-empty only for categories where synthetic data includes offenders.
**Warning signs:** Tests that pass locally but fail on clean testdata.

### Pitfall 2: HTML Template Escaping in String Assertions
**What goes wrong:** Searching for `"Build & Test"` in HTML when template outputs `"Build &amp; Test"`.
**Why it happens:** Go's `html/template` escapes `&` to `&amp;` in rendered output.
**How to avoid:** Use escaped versions in assertions: `"Build &amp; Test"`. The existing prompt_test.go already does this correctly.
**Warning signs:** Assertions failing with "missing substring" for strings containing `&`, `<`, `>`.

### Pitfall 3: Prompt Templates Only Generated for Scores < 9.0
**What goes wrong:** Asserting 38 prompt templates exist when some metrics score >= 9.0.
**Why it happens:** The `buildHTMLSubScores` function only generates `PromptHTML` when `ss.Score < 9.0`.
**How to avoid:** For TEST-04, construct a `ScoredResult` where ALL metrics have scores below 9.0 (e.g., use score 5.0 as in existing `buildAllCategoriesScoredResult(5.0)`). Then count `<template id="prompt-` occurrences.
**Warning signs:** Template count less than expected.

### Pitfall 4: File Size Budget Depends on All Features Being Enabled
**What goes wrong:** HTML file size under 500KB in test but over 500KB in production.
**Why it happens:** Test may omit trace data, prompts, baseline comparison, or recommendations.
**How to avoid:** For TEST-02, generate a "full" HTML report with TraceData, all 7 categories with C7 debug samples, baseline comparison, and recommendations. This represents the maximum realistic size.
**Warning signs:** Test passes but production reports fail the budget.

### Pitfall 5: Zero-Weight Metrics Filtered in HTML But Not in Config
**What goes wrong:** Expecting metrics like C7's deprecated `overall_score` in HTML output.
**Why it happens:** `buildHTMLSubScores` skips metrics with `ss.Weight == 0.0`.
**How to avoid:** Only count metrics that have `Weight > 0` in the scoring config when validating prompt template coverage.
**Warning signs:** Metric count mismatch between config (which may include zero-weight) and HTML output.

### Pitfall 6: Responsive Layout Test Cannot Run a Real Browser
**What goes wrong:** Attempting to test responsive layout with viewport simulation in Go tests.
**Why it happens:** Go tests cannot execute CSS media queries.
**How to avoid:** For TEST-06, validate the presence of responsive CSS patterns (`@media (max-width: 640px)`) and key mobile-friendly CSS properties in the generated HTML. Check that modal has appropriate max-width/height CSS. Do not attempt visual testing.
**Warning signs:** Over-engineering a browser-based test harness for a CLI tool.

## Code Examples

### Evidence Extraction Test Pattern
```go
// Verify extractC1 returns evidence for all metrics
func TestExtractC1_EvidenceNonEmpty(t *testing.T) {
    ar := &types.AnalysisResult{
        Category: "C1",
        Metrics: map[string]interface{}{
            "c1": &types.C1Metrics{
                CyclomaticComplexity: types.MetricSummary{Avg: 15.0, Max: 30},
                Functions: []types.FunctionMetric{
                    {Name: "foo", Complexity: 30, File: "a.go", Line: 10, LineCount: 80},
                    {Name: "bar", Complexity: 25, File: "b.go", Line: 20, LineCount: 60},
                },
                // ... other metrics with data
            },
        },
    }
    _, _, evidence := extractC1(ar)
    for metric, items := range evidence {
        if len(items) == 0 {
            t.Errorf("evidence for %s should be non-empty", metric)
        }
    }
}
```

### HTML File Size Budget Test Pattern
```go
func TestHTMLFileSizeBudget(t *testing.T) {
    gen, _ := NewHTMLGenerator()
    scored := buildFullScoredResultWithC7() // All 7 categories, evidence, trace
    trace := &TraceData{ScoringConfig: scoring.DefaultConfig(), Languages: []string{"go"}}
    var buf bytes.Buffer
    gen.GenerateReport(&buf, scored, recs, baseline, trace)

    const maxBytes = 500 * 1024 // 500KB
    if buf.Len() > maxBytes {
        t.Errorf("HTML report size %d bytes exceeds budget %d bytes", buf.Len(), maxBytes)
    }
}
```

### Prompt Template Coverage Test Pattern
```go
func TestPromptTemplateCoverage_All38Metrics(t *testing.T) {
    gen, _ := NewHTMLGenerator()
    scored := buildAllCategoriesScoredResult(5.0) // All metrics score < 9.0
    // Ensure ALL 38 metrics are present across 7 categories
    trace := &TraceData{ScoringConfig: scoring.DefaultConfig(), Languages: []string{"go"}}
    var buf bytes.Buffer
    gen.GenerateReport(&buf, scored, nil, nil, trace)
    html := buf.String()

    cfg := scoring.DefaultConfig()
    for catName, cat := range cfg.Categories {
        for _, m := range cat.Metrics {
            if m.Weight == 0 { continue } // skip deprecated
            templateID := fmt.Sprintf(`<template id="prompt-%s">`, m.Name)
            if !strings.Contains(html, templateID) {
                t.Errorf("missing prompt template for %s/%s", catName, m.Name)
            }
        }
    }
}
```

### Accessibility Validation Test Pattern
```go
func TestHTMLAccessibility(t *testing.T) {
    gen, _ := NewHTMLGenerator()
    scored := buildAllCategoriesScoredResult(5.0)
    trace := &TraceData{ScoringConfig: scoring.DefaultConfig(), Languages: []string{"go"}}
    var buf bytes.Buffer
    gen.GenerateReport(&buf, scored, nil, nil, trace)
    html := buf.String()

    checks := []struct {
        substring string
        desc      string
    }{
        {`aria-label="Close"`, "modal close button should have aria-label"},
        {`<dialog id="ars-modal"`, "should use native dialog element"},
        {`showModal()`, "should use showModal for focus trapping"},
        {`lang="en"`, "html should have lang attribute"},
        {`<noscript>`, "progressive enhancement for no-JS"},
        {`autofocus`, "close button should have autofocus for keyboard nav"},
    }
    for _, c := range checks {
        if !strings.Contains(html, c.substring) {
            t.Errorf("%s (missing %q)", c.desc, c.substring)
        }
    }
}
```

### JSON Backward Compatibility Test Pattern
```go
func TestJSONBaselineV1Compatibility(t *testing.T) {
    // v0.0.5 JSON used "metrics" instead of "sub_scores" and version "1"
    v1JSON := `{
        "version": "1",
        "composite_score": 6.5,
        "tier": "Agent-Assisted",
        "categories": [
            {"name": "C1", "score": 7.0, "weight": 0.25, "metrics": [
                {"name": "complexity_avg", "raw_value": 8.0, "score": 7.5, "weight": 0.25, "available": true}
            ]},
            {"name": "C3", "score": 6.0, "weight": 0.20}
        ]
    }`
    var report JSONReport
    if err := json.Unmarshal([]byte(v1JSON), &report); err != nil {
        t.Fatalf("v1 JSON should unmarshal without error: %v", err)
    }
    if report.CompositeScore != 6.5 { t.Error("composite should load") }
    if report.Tier != "Agent-Assisted" { t.Error("tier should load") }
    if len(report.Categories) != 2 { t.Error("categories should load") }
    if report.Categories[0].Score != 7.0 { t.Error("category score should load") }
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| JSON "metrics" field | JSON "sub_scores" field | v0.0.6 (version "2") | Backward compat test must handle both |
| C7 overall_score included | C7 overall_score removed (zero-weight) | v0.0.6 | Skip zero-weight metrics in prompt coverage count |
| No evidence in SubScore | Evidence always `[]` not null | v0.0.6 | Evidence extraction tests validate non-nil arrays |
| No prompt templates | Prompt templates for metrics < 9.0 | v0.0.6 | New test: prompt coverage for all 38 metrics |

## Key Data Points

### Metric Count by Category
| Category | Metric Count | Notes |
|----------|-------------|-------|
| C1 | 6 | complexity_avg, func_length_avg, file_size_avg, afferent_coupling_avg, efferent_coupling_avg, duplication_rate |
| C2 | 5 | type_annotation_coverage, naming_consistency, magic_number_ratio, type_strictness, null_safety |
| C3 | 5 | max_dir_depth, module_fanout_avg, circular_deps, import_complexity_avg, dead_exports |
| C4 | 7 | readme_word_count, comment_density, api_doc_coverage, changelog_present, examples_present, contributing_present, diagrams_present |
| C5 | 5 | churn_rate, temporal_coupling_pct, author_fragmentation, commit_stability, hotspot_concentration |
| C6 | 5 | test_to_code_ratio, coverage_percent, test_isolation, assertion_density_avg, test_file_ratio |
| C7 | 5 | task_execution_consistency, code_behavior_comprehension, cross_file_navigation, identifier_interpretability, documentation_accuracy_detection |
| **Total** | **38** | All have entries in `metricDescriptions` map and `metricDisplayName` map |

### Existing Test Helpers to Reuse
| Helper | Location | Purpose |
|--------|----------|---------|
| `buildAllCategoriesScoredResult(score)` | `internal/output/html_test.go` | Creates ScoredResult with all 7 categories, configurable score |
| `buildPromptTestEvidence()` | `internal/output/html_test.go` | Returns sample evidence items |
| `newTestScoredResult()` | `internal/output/json_test.go` | Creates ScoredResult for JSON tests |
| `makeHealthyC1()` | `internal/scoring/scorer_test.go` | Creates C1 AnalysisResult with healthy metrics |
| `scoreCategory(s, ar)` | `internal/scoring/scorer_test.go` | Scores a single AnalysisResult |

### HTML Accessibility Attributes Already Present
| Element | Attribute | Location |
|---------|-----------|----------|
| `<html>` | `lang="en"` | report.html line 2 |
| Modal close button | `aria-label="Close"` | report.html line 202 |
| Modal close button | `autofocus` | report.html line 202 |
| `<noscript>` | Hides `.ars-modal-trigger` | report.html line 9 |
| `<dialog>` | Native focus trapping via `showModal()` | report.html line 198 |

### Responsive CSS Already Present
| Feature | Implementation |
|---------|---------------|
| Mobile breakpoint | `@media (max-width: 640px)` with font/padding adjustments |
| Print styles | `@media print` with page break control |
| Viewport meta | `<meta name="viewport" content="width=device-width, initial-scale=1.0">` |

## Open Questions

1. **Evidence extraction for C4/C5 metrics**
   - What we know: extractC4 and extractC5 exist in scorer.go and return evidence maps
   - What's unclear: Whether all C4/C5 metrics produce meaningful evidence (binary metrics like `changelog_present` may have empty evidence)
   - Recommendation: Test that evidence map is returned (not nil), allow empty evidence for binary/boolean metrics

2. **File size budget realism**
   - What we know: 500KB limit specified in requirements
   - What's unclear: Whether a report with all 7 categories, full trace data, 38 prompt templates, baseline comparison, and recommendations stays under 500KB
   - Recommendation: Generate a maximally-loaded report in the test and measure; if it exceeds 500KB, flag for discussion rather than failing silently

## Sources

### Primary (HIGH confidence)
- Codebase analysis: `internal/output/html.go`, `html_test.go`, `json.go`, `json_test.go`, `prompt.go`, `prompt_test.go`
- Codebase analysis: `internal/scoring/config.go` (38 metrics), `scorer.go` (extractors), `scorer_test.go`
- Codebase analysis: `internal/output/templates/report.html` (ARIA attributes, responsive CSS)
- Codebase analysis: `pkg/types/types.go`, `pkg/types/scoring.go` (data structures)

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - all stdlib Go, no new dependencies
- Architecture: HIGH - follows established patterns visible in existing tests
- Pitfalls: HIGH - derived from reading actual code and understanding template escaping, score thresholds, zero-weight metrics
- Test data strategy: HIGH - reuses existing helpers and types

**Research date:** 2026-02-07
**Valid until:** 2026-03-07 (stable; these are tests for existing code)
