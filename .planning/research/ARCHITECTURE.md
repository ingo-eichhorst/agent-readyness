# Architecture Patterns: Interactive HTML Modal Enhancements

**Domain:** Interactive modal features for ARS HTML report generator
**Researched:** 2026-02-06
**Confidence:** HIGH (based on direct codebase analysis, not external sources)

---

## Executive Summary

The ARS codebase already contains nearly all the data needed for interactive modals -- it just does not propagate it to the HTML renderer. Each CxMetrics struct stores detailed evidence (functions, file pairs, dead exports, hotspots) that the scoring layer discards when extracting numeric values. The integration strategy is: (1) extend the scoring pipeline to carry evidence alongside scores, (2) generate copy-paste improvement prompts after scoring, (3) render both in modal overlays in the HTML template.

The architecture change is additive. No analyzer modifications are needed. The `SubScore` type gains an `Evidence` field. The `MetricExtractor` signature gains a third return value. The HTML template gains modal markup and ~30 lines of JavaScript. Total scope: 10 modified files, 2 new files.

---

## Current Architecture Summary

### Pipeline Flow

```
Discovery -> Parse -> Analyze (parallel, C1-C7)
    -> Score (piecewise interpolation)
    -> Recommend (top-5 improvement actions)
    -> Output (terminal | JSON | HTML)
```

### Key Data Flow

```
types.AnalysisTarget[]
    -> Analyzer.Analyze() -> types.AnalysisResult { Metrics: map[string]interface{} }
    -> Scorer.Score()      -> types.ScoredResult  { Categories: []CategoryScore }
    -> HTMLGenerator        -> HTMLReportData      -> template -> HTML file
```

### Current Type Chain (files with line references)

| Type | File | Line | Role |
|------|------|------|------|
| `AnalysisResult` | `pkg/types/types.go` | 88 | Analyzer output, carries `Metrics: map[string]interface{}` |
| `C1Metrics` | `pkg/types/types.go` | 123 | Has `Functions`, `DuplicatedBlocks` (evidence data) |
| `C3Metrics` | `pkg/types/types.go` | 135 | Has `CircularDeps`, `DeadExports` (evidence data) |
| `C5Metrics` | `pkg/types/types.go` | 196 | Has `TopHotspots`, `CoupledPairs` (evidence data) |
| `C6Metrics` | `pkg/types/types.go` | 173 | Has `TestFunctions` (evidence data) |
| `C7Metrics` | `pkg/types/types.go` | 253 | Has `MetricResults` with `DebugSamples` (full trace data) |
| `CategoryScore` | `pkg/types/scoring.go` | 12 | Scored output with `SubScores []SubScore` |
| `SubScore` | `pkg/types/scoring.go` | 20 | **Only carries: MetricName, RawValue, Score, Weight, Available** |
| `HTMLReportData` | `internal/output/html.go` | 25 | Template data root |
| `HTMLCategory` | `internal/output/html.go` | 44 | Per-category template data |
| `HTMLSubScore` | `internal/output/html.go` | 55 | Per-metric template data with descriptions |

### The Gap

**SubScore** (the bridge between scoring and rendering) carries only numeric values. All evidence data in CxMetrics is lost during scoring. This is the single integration point that must change.

---

## Recommended Architecture

### Design Principle: Extend the Existing Data Path

Do NOT create a parallel data flow. Embed evidence in `SubScore` so it flows through the existing pipeline to all consumers (HTML, JSON, terminal) without new plumbing.

### Component Boundaries

```
+------------------------------------------------------------------+
|                        EXISTING PIPELINE                          |
|                                                                   |
|  Analyzers (C1-C7)  -- NO CHANGES                               |
|      |                                                            |
|      v                                                            |
|  AnalysisResult.Metrics (CxMetrics with evidence)  -- NO CHANGES |
|      |                                                            |
|      v                                                            |
|  Scorer -> ScoredResult -> SubScore  -- EXTEND with Evidence     |
|      |                                                            |
|      v                                                            |
|  HTMLGenerator  -- EXTEND to render Evidence + Prompts           |
+------------------------------------------------------------------+

+------------------------------------------------------------------+
|                      NEW / MODIFIED COMPONENTS                    |
|                                                                   |
|  [A] EvidenceItem type (pkg/types/scoring.go)                    |
|      - Generic evidence struct: File, Line, Entity, Value, Detail|
|      - Added as []EvidenceItem field on SubScore                 |
|                                                                   |
|  [B] MetricExtractor extension (internal/scoring/scorer.go)      |
|      - Third return value: map[string][]EvidenceItem             |
|      - extractC1..extractC7 populate evidence from CxMetrics     |
|                                                                   |
|  [C] Improvement Prompt Generator (internal/recommend/prompts.go)|
|      - Takes ScoredResult + ScoringConfig -> []ImprovementPrompt |
|      - Uses Evidence data to make prompts specific               |
|      - Runs AFTER scoring, BEFORE HTML rendering                 |
|                                                                   |
|  [D] HTML Modal UI (internal/output/)                            |
|      - HTMLSubScore extensions: Evidence, ImprovementPrompt      |
|      - Modal overlay in report.html                              |
|      - CSS + JS for modal open/close                             |
|                                                                   |
|  [E] JSON Output Extension (internal/output/json.go)            |
|      - JSONMetric gains Evidence + Prompt fields (omitempty)     |
+------------------------------------------------------------------+
```

---

## Integration Points (with file/line references)

### Integration Point 1: Evidence Already Exists in CxMetrics

**Key insight: No new data capture is needed.** The analyzers already store the evidence that the modals need to display. It is simply not propagated.

| Category | Evidence Field | Type | File:Line |
|----------|---------------|------|-----------|
| C1 | `Functions` | `[]FunctionMetric` | `pkg/types/types.go:131` |
| C1 | `DuplicatedBlocks` | `[]DuplicateBlock` | `pkg/types/types.go:130` |
| C1 | `AfferentCoupling` | `map[string]int` | `pkg/types/types.go:127` |
| C1 | `EfferentCoupling` | `map[string]int` | `pkg/types/types.go:128` |
| C3 | `CircularDeps` | `[][]string` | `pkg/types/types.go:139` |
| C3 | `DeadExports` | `[]DeadExport` | `pkg/types/types.go:141` |
| C5 | `TopHotspots` | `[]FileChurn` | `pkg/types/types.go:202` |
| C5 | `CoupledPairs` | `[]CoupledPair` | `pkg/types/types.go:203` |
| C6 | `TestFunctions` | `[]TestFunctionMetric` | `pkg/types/types.go:181` |
| C7 | `MetricResults[].DebugSamples` | `[]C7DebugSample` | `pkg/types/types.go:303` |

C2 and C4 have aggregate metrics but no per-file evidence lists. Their modals will show the aggregate values with improvement prompts but no file-level evidence table. This is acceptable -- not every metric has file-level detail.

### Integration Point 2: Type Extensions for Evidence Flow

**Where:** `pkg/types/scoring.go`

**Current SubScore** (line 20):
```go
type SubScore struct {
    MetricName string
    RawValue   float64
    Score      float64
    Weight     float64
    Available  bool
}
```

**Extended SubScore:**
```go
type SubScore struct {
    MetricName string         // existing
    RawValue   float64        // existing
    Score      float64        // existing
    Weight     float64        // existing
    Available  bool           // existing
    Evidence   []EvidenceItem `json:"evidence,omitempty"` // NEW
}

// EvidenceItem represents one piece of evidence for a metric score.
type EvidenceItem struct {
    File   string  `json:"file"`
    Line   int     `json:"line,omitempty"`
    Entity string  `json:"entity,omitempty"`   // function name, package, etc.
    Value  float64 `json:"value,omitempty"`     // the measured value for this item
    Detail string  `json:"detail,omitempty"`    // human-readable note
}
```

**Why this approach:**
- Evidence is conceptually part of a SubScore (it explains the score)
- `json:"omitempty"` preserves backward compatibility for JSON output
- No new plumbing -- evidence flows with SubScore through existing pipeline
- EvidenceItem is generic enough for all categories

**Alternatives considered and rejected:**
- **Parallel AnalysisResult passthrough:** Would break `GenerateReport` signature and require type assertions in templates
- **Evidence on CategoryScore:** Too coarse -- evidence belongs to individual metrics, not categories
- **CxMetrics passthrough:** Would require `interface{}` in SubScore (untyped, messy)

### Integration Point 3: Scorer Evidence Population

**Where:** `internal/scoring/scorer.go`

**Current MetricExtractor** (line 14):
```go
type MetricExtractor func(ar *types.AnalysisResult) (
    rawValues   map[string]float64,
    unavailable map[string]bool,
)
```

**Extended MetricExtractor:**
```go
type MetricExtractor func(ar *types.AnalysisResult) (
    rawValues   map[string]float64,
    unavailable map[string]bool,
    evidence    map[string][]types.EvidenceItem,  // NEW: keyed by metric name
)
```

**Modified scoreMetrics** (line 382):
```go
func scoreMetrics(catConfig CategoryConfig, rawValues map[string]float64,
    unavailable map[string]bool, evidence map[string][]types.EvidenceItem) ([]types.SubScore, float64) {

    // ... existing logic ...

    for _, mt := range catConfig.Metrics {
        ss := types.SubScore{
            MetricName: mt.Name,
            RawValue:   rawValues[mt.Name],
            Weight:     mt.Weight,
            Available:  true,
            Evidence:   evidence[mt.Name],  // NEW: attach evidence
        }
        // ... rest unchanged ...
    }
}
```

**Evidence extraction examples:**

For `extractC1` (line 176), add evidence for `complexity_avg`:
```go
func extractC1(ar *types.AnalysisResult) (map[string]float64, map[string]bool, map[string][]types.EvidenceItem) {
    // ... existing raw value extraction ...

    evidence := make(map[string][]types.EvidenceItem)

    // Top 5 most complex functions as evidence for complexity_avg
    // (m.Functions already sorted or sortable)
    for i, fn := range topN(m.Functions, 5, byComplexity) {
        evidence["complexity_avg"] = append(evidence["complexity_avg"], types.EvidenceItem{
            File:   fn.File,
            Line:   fn.Line,
            Entity: fn.Name,
            Value:  float64(fn.Complexity),
            Detail: fmt.Sprintf("complexity %d", fn.Complexity),
        })
    }

    // ... similar for func_length_avg, duplication_rate, coupling ...

    return rawValues, nil, evidence
}
```

**Impact analysis:** All 7 `extractCx` functions need a third return value. Functions that have no evidence return `nil`. This is mechanical -- the compiler enforces it.

### Integration Point 4: Improvement Prompt Generation

**Where:** New file `internal/recommend/prompts.go`

**Design:**
```go
package recommend

type ImprovementPrompt struct {
    MetricName string // e.g., "complexity_avg"
    Category   string // e.g., "C1"
    Prompt     string // Copy-paste ready prompt for Claude/GPT
    Context    string // Brief explanation of what the prompt does
}

// GeneratePrompts creates improvement prompts for metrics scoring below threshold.
// Uses evidence data to make prompts specific to the actual codebase.
func GeneratePrompts(scored *types.ScoredResult, cfg *scoring.ScoringConfig) []ImprovementPrompt {
    var prompts []ImprovementPrompt

    for _, cat := range scored.Categories {
        for _, ss := range cat.SubScores {
            if !ss.Available || ss.Score >= 8.0 {
                continue  // Only generate prompts for metrics needing improvement
            }

            prompt := buildPromptForMetric(cat.Name, ss)
            if prompt != nil {
                prompts = append(prompts, *prompt)
            }
        }
    }

    return prompts
}
```

**Prompt content strategy:** Each prompt is a self-contained instruction that references specific files/functions from the Evidence data. Example for `complexity_avg`:

```
Review the following Go functions that have high cyclomatic complexity and refactor each
to reduce complexity while preserving behavior:

- internal/pipeline/pipeline.go:164 Run() (complexity: 22)
- internal/scoring/scorer.go:130 Score() (complexity: 15)
- internal/output/html.go:94 GenerateReport() (complexity: 13)

For each function:
1. Identify the sources of complexity (nested conditionals, switch statements)
2. Apply extract-method refactoring to isolate logical sections
3. Replace nested conditionals with guard clauses where appropriate
4. Ensure all existing tests continue to pass after refactoring
```

**Integration with pipeline** (`internal/pipeline/pipeline.go`, after line 269):
```go
// Stage 3.7: Generate improvement prompts (after recommendations, before output)
var prompts []recommend.ImprovementPrompt
if p.scored != nil {
    prompts = recommend.GeneratePrompts(p.scored, p.scorer.Config)
}
```

Then pass `prompts` to the HTML generator.

### Integration Point 5: HTML Template Modal Structure

**Where:** `internal/output/templates/report.html` and `internal/output/templates/styles.css`

**Current state:** The report already has JavaScript for chevron-click expandable rows (lines 126-158) and clipboard copy (line 34). Adding modal open/close follows the same pattern.

**Modal placement in template:**

The modal trigger button goes inside the existing `metric-details-row` (between `BriefDescription` and `DetailedDescription` content):

```html
<tr class="metric-details-row" data-for="{{.Key}}" ...>
    <td></td>
    <td colspan="4" class="metric-details-content">
        {{if .BriefDescription}}
        <div class="metric-brief">{{.BriefDescription}}</div>
        {{end}}
        {{if .DetailedDescription}}
        <div class="metric-detailed">{{.DetailedDescription}}</div>
        {{end}}
        <!-- NEW: Modal trigger -->
        {{if .HasModal}}
        <button class="modal-trigger" data-modal="modal-{{.Key}}">
            View Evidence & Improvement Prompt
        </button>
        {{end}}
    </td>
</tr>
```

The modal overlays go at the end of `<body>`, before `<footer>`:

```html
<!-- Modal overlays (one per metric with evidence/prompt) -->
{{range .Categories}}
{{range .SubScores}}
{{if .HasModal}}
<div class="modal-overlay" id="modal-{{.Key}}">
    <div class="modal-content">
        <button class="modal-close">&times;</button>
        <h3>{{.DisplayName}}</h3>

        {{if .HasEvidence}}
        <div class="modal-section">
            <h4>Evidence (top contributors)</h4>
            <table class="evidence-table">
                <thead><tr><th>File</th><th>Entity</th><th>Value</th></tr></thead>
                <tbody>
                {{range .Evidence}}
                <tr>
                    <td>{{.File}}{{if .Line}}:{{.Line}}{{end}}</td>
                    <td>{{.Entity}}</td>
                    <td>{{.FormattedValue}}</td>
                </tr>
                {{end}}
                </tbody>
            </table>
        </div>
        {{end}}

        {{if .ImprovementPrompt}}
        <div class="modal-section">
            <h4>Improvement Prompt</h4>
            <p class="prompt-context">{{.PromptContext}}</p>
            <pre class="prompt-content"><code>{{.ImprovementPrompt}}</code></pre>
            <button class="copy-btn"
                onclick="navigator.clipboard.writeText(
                    this.previousElementSibling.querySelector('code').textContent
                )">Copy Prompt</button>
        </div>
        {{end}}
    </div>
</div>
{{end}}
{{end}}
{{end}}
```

**JavaScript for modals (~30 lines):**

```javascript
// Modal open
document.querySelectorAll('.modal-trigger').forEach(btn => {
    btn.addEventListener('click', () => {
        document.getElementById(btn.dataset.modal).classList.add('active');
    });
});
// Modal close (X button)
document.querySelectorAll('.modal-close').forEach(btn => {
    btn.addEventListener('click', () => {
        btn.closest('.modal-overlay').classList.remove('active');
    });
});
// Modal close (backdrop click)
document.querySelectorAll('.modal-overlay').forEach(overlay => {
    overlay.addEventListener('click', (e) => {
        if (e.target === overlay) overlay.classList.remove('active');
    });
});
// Modal close (Escape key)
document.addEventListener('keydown', (e) => {
    if (e.key === 'Escape') {
        document.querySelectorAll('.modal-overlay.active').forEach(m => {
            m.classList.remove('active');
        });
    }
});
```

**Self-contained constraint:** Follows the existing pattern exactly. CSS inlined via `{{.InlineCSS}}`, JS inline in `<script>` tags. Zero external dependencies.

### Integration Point 6: HTMLSubScore Extensions

**Where:** `internal/output/html.go`, `HTMLSubScore` struct (line 55)

**Current HTMLSubScore:**
```go
type HTMLSubScore struct {
    Key                 string
    MetricName          string
    DisplayName         string
    RawValue            float64
    FormattedValue      string
    Score               float64
    ScoreClass          string
    WeightPct           float64
    Available           bool
    BriefDescription    string
    DetailedDescription template.HTML
    ShouldExpand        bool
}
```

**Extended HTMLSubScore:**
```go
type HTMLSubScore struct {
    // ... all existing fields unchanged ...

    // NEW: Evidence and prompt data for modal
    Evidence          []HTMLEvidence // Populated from SubScore.Evidence
    HasEvidence       bool           // Template convenience
    ImprovementPrompt string         // Copy-paste prompt text
    PromptContext     string         // Brief explanation
    HasModal          bool           // True if evidence or prompt exists
}

// HTMLEvidence represents one evidence item for template rendering.
type HTMLEvidence struct {
    File           string
    Line           int
    Entity         string
    FormattedValue string
    Detail         string
}
```

**Builder function modification** (`buildHTMLSubScores`, line 197):

The function currently takes `[]types.SubScore` and returns `[]HTMLSubScore`. It needs to also receive prompts:

```go
func buildHTMLSubScores(subScores []types.SubScore,
    prompts map[string]recommend.ImprovementPrompt) []HTMLSubScore {

    result := make([]HTMLSubScore, 0, len(subScores))

    for _, ss := range subScores {
        // ... existing logic for hss construction ...

        // NEW: Populate evidence from SubScore
        for _, ev := range ss.Evidence {
            hss.Evidence = append(hss.Evidence, HTMLEvidence{
                File:           ev.File,
                Line:           ev.Line,
                Entity:         ev.Entity,
                FormattedValue: formatEvidenceValue(ss.MetricName, ev.Value),
                Detail:         ev.Detail,
            })
        }
        hss.HasEvidence = len(hss.Evidence) > 0

        // NEW: Populate prompt
        if prompt, ok := prompts[ss.MetricName]; ok {
            hss.ImprovementPrompt = prompt.Prompt
            hss.PromptContext = prompt.Context
        }
        hss.HasModal = hss.HasEvidence || hss.ImprovementPrompt != ""

        result = append(result, hss)
    }

    return result
}
```

### Integration Point 7: JSON Output Extension

**Where:** `internal/output/json.go`, `JSONMetric` struct (line 30)

**Extended JSONMetric:**
```go
type JSONMetric struct {
    Name      string         `json:"name"`
    RawValue  float64        `json:"raw_value"`
    Score     float64        `json:"score"`
    Weight    float64        `json:"weight"`
    Available bool           `json:"available"`
    Evidence  []JSONEvidence `json:"evidence,omitempty"`          // NEW
    Prompt    string         `json:"improvement_prompt,omitempty"` // NEW
}

type JSONEvidence struct {
    File   string  `json:"file"`
    Line   int     `json:"line,omitempty"`
    Entity string  `json:"entity,omitempty"`
    Value  float64 `json:"value,omitempty"`
    Detail string  `json:"detail,omitempty"`
}
```

**Backward compatibility:** The `omitempty` tags ensure existing JSON consumers see no change when evidence is absent. This matches the C7 `DebugSamples` pattern (`json:"debug_samples,omitempty"` at line 303 of types.go).

### Integration Point 8: GenerateReport Signature

**Where:** `internal/output/html.go`, `GenerateReport` method (line 94)

**Current:**
```go
func (g *HTMLGenerator) GenerateReport(
    w io.Writer,
    scored *types.ScoredResult,
    recs []recommend.Recommendation,
    baseline *types.ScoredResult,
) error
```

**Extended (add prompts parameter):**
```go
func (g *HTMLGenerator) GenerateReport(
    w io.Writer,
    scored *types.ScoredResult,
    recs []recommend.Recommendation,
    baseline *types.ScoredResult,
    prompts []recommend.ImprovementPrompt,
) error
```

**Callers to update:**
- `internal/pipeline/pipeline.go` line 346: `gen.GenerateReport(f, p.scored, recs, baseline)` -> add `prompts`
- `internal/output/html_test.go` lines 75, 125, 172, 298: all test calls need `prompts` parameter (can pass `nil`)

**Alternative considered:** Options struct to avoid parameter growth. Rejected as premature -- 5 parameters is manageable. Refactor to options struct if a 6th parameter arrives.

---

## Anti-Patterns to Avoid

### Anti-Pattern 1: Modifying the Analyzer Interface

**What:** Adding evidence capture to `pipeline.Analyzer.Analyze()`.
**Why bad:** All 7 analyzers need updating. Analyzers already capture evidence in CxMetrics.
**Instead:** Extract evidence during scoring from existing CxMetrics data.

### Anti-Pattern 2: Separate Evidence Pipeline

**What:** Passing `[]*types.AnalysisResult` alongside `*types.ScoredResult` to HTML generator.
**Why bad:** Two data paths that must stay synchronized. Template needs type assertions on `interface{}` metrics.
**Instead:** Embed evidence in SubScore. Single data path, one source of truth.

### Anti-Pattern 3: External JavaScript Libraries

**What:** Pulling in a modal library (Bootstrap, dialog-polyfill).
**Why bad:** Breaks self-contained HTML report design. Report is a single file with inlined CSS/JS.
**Instead:** ~30 lines of vanilla JavaScript. The existing codebase already uses inline JS.

### Anti-Pattern 4: Generating Prompts in Analyzers

**What:** Having analyzers generate improvement prompts during analysis.
**Why bad:** Analyzers know code structure but not scoring thresholds or priorities. Prompts need score context.
**Instead:** Generate prompts after scoring, in the recommend package.

### Anti-Pattern 5: Modal for Every Metric

**What:** Generating modal content for all 26+ metrics regardless of score.
**Why bad:** Bloats HTML file size with unused content.
**Instead:** Only generate modals for metrics scoring below threshold (e.g., < 8.0) or where evidence exists. Follow the existing `ShouldExpand` pattern (line 67, `ss.Score < desc.Threshold`).

---

## Suggested Build Order

### Phase 1: Evidence Types and Scorer Wiring

**Goal:** Get evidence data flowing through the scoring pipeline.

**Changes:**
1. Add `EvidenceItem` type to `pkg/types/scoring.go`
2. Add `Evidence []EvidenceItem` field to `SubScore`
3. Extend `MetricExtractor` signature in `internal/scoring/scorer.go` (third return)
4. Update `extractC1` through `extractC7` to return evidence
5. Update `scoreMetrics` to attach evidence to SubScore

**Modified files:**
- `pkg/types/scoring.go`
- `internal/scoring/scorer.go`
- `internal/scoring/scorer_test.go`

**Tests:** Verify SubScore.Evidence populated for each category. All existing scoring tests pass unchanged.

**Why first:** Everything downstream depends on evidence in SubScore.

### Phase 2: JSON Output Integration

**Goal:** Evidence visible in JSON output (simplest consumer, validates data flow).

**Changes:**
1. Add `JSONEvidence` type and `Evidence` field to `JSONMetric` in `internal/output/json.go`
2. Populate evidence in `BuildJSONReport`

**Modified files:**
- `internal/output/json.go`
- `internal/output/json_test.go`

**Why second:** Simplest output consumer. Validates evidence flows correctly before HTML complexity.

### Phase 3: Improvement Prompt Generation

**Goal:** Generate copy-paste prompts for underperforming metrics.

**Changes:**
1. Create `internal/recommend/prompts.go` with `ImprovementPrompt` type and `GeneratePrompts`
2. Create `internal/recommend/prompts_test.go`
3. Add `GeneratePrompts` call in `internal/pipeline/pipeline.go`

**New files:**
- `internal/recommend/prompts.go`
- `internal/recommend/prompts_test.go`

**Modified files:**
- `internal/pipeline/pipeline.go`

**Why third:** Prompts use evidence data (Phase 1). Must be stable before HTML rendering.

### Phase 4: HTML Modal Rendering

**Goal:** Interactive modals showing evidence and improvement prompts.

**Changes:**
1. Extend `HTMLSubScore` in `internal/output/html.go`
2. Add `HTMLEvidence` type
3. Modify `buildHTMLSubScores` to populate evidence + prompts
4. Modify `buildHTMLCategories` to pass prompts through
5. Extend `GenerateReport` signature (add prompts parameter)
6. Add modal overlay template section in `internal/output/templates/report.html`
7. Add modal trigger button in metric-details-row
8. Add modal JavaScript (~30 lines)
9. Add modal CSS to `internal/output/templates/styles.css`
10. Update tests in `internal/output/html_test.go`

**Modified files:**
- `internal/output/html.go`
- `internal/output/templates/report.html`
- `internal/output/templates/styles.css`
- `internal/output/html_test.go`

**Why last:** Most complex consumer, depends on all upstream data.

### Dependency Graph

```
Phase 1: Evidence Types + Scorer
    |
    +---> Phase 2: JSON Output (can parallelize with Phase 3)
    |
    +---> Phase 3: Prompts (depends on Evidence)
              |
              v
         Phase 4: HTML Modals (depends on Evidence + Prompts)
```

---

## File Change Matrix

| File | Change | Scope | Phase |
|------|--------|-------|-------|
| `pkg/types/scoring.go` | MODIFY | Add EvidenceItem type, Evidence field on SubScore | 1 |
| `internal/scoring/scorer.go` | MODIFY | MetricExtractor signature, extractCx, scoreMetrics | 1 |
| `internal/scoring/scorer_test.go` | MODIFY | Test evidence population | 1 |
| `internal/output/json.go` | MODIFY | JSONMetric extensions, JSONEvidence type | 2 |
| `internal/output/json_test.go` | MODIFY | Test evidence in JSON output | 2 |
| `internal/recommend/prompts.go` | **NEW** | ImprovementPrompt type, GeneratePrompts | 3 |
| `internal/recommend/prompts_test.go` | **NEW** | Prompt generation tests | 3 |
| `internal/pipeline/pipeline.go` | MODIFY | Call GeneratePrompts, pass prompts to HTML | 3-4 |
| `internal/output/html.go` | MODIFY | HTMLSubScore extensions, builder changes, GenerateReport signature | 4 |
| `internal/output/templates/report.html` | MODIFY | Modal template, trigger buttons, JS | 4 |
| `internal/output/templates/styles.css` | MODIFY | Modal styles | 4 |
| `internal/output/html_test.go` | MODIFY | Test modal rendering, update GenerateReport calls | 4 |

**Total: 10 modified files, 2 new files**

---

## Risk Assessment

### Low Risk
- **EvidenceItem type:** Additive field with `omitempty`, zero impact on existing behavior
- **JSON output extension:** Same additive pattern, backward compatible
- **Modal CSS/JS:** Self-contained, follows existing patterns, no external deps
- **HTML template changes:** Template-only, easy to preview and iterate

### Medium Risk
- **MetricExtractor signature change:** All 7 extractCx functions must update. Mechanical but wide-reaching. The compiler catches missed updates (missing return value). Mitigated by updating one at a time and running tests.
- **Prompt quality:** Generated prompts must be genuinely useful. Risk of generic prompts. Mitigated by using evidence data (actual file names, function names, line numbers) to make prompts specific.

### Low-Medium Risk
- **HTML file size:** Adding modals increases file size. Current report ~50-100KB. Adding 10-15 modals may add 20-30KB. Still reasonable for self-contained report. Mitigated by only generating modals for metrics below threshold.

---

## Scalability Considerations

| Concern | 5 metrics below threshold | 15 metrics below threshold | All 26 metrics |
|---------|--------------------------|----------------------------|----------------|
| HTML size | +10KB (minimal) | +30KB (acceptable) | +50KB (avoid: use threshold) |
| Evidence queries | Negligible (data already in CxMetrics) | Same | Same |
| Prompt generation | <1ms | <5ms | <10ms |
| Template rendering | Negligible | +50ms | +100ms |

---

## Sources

All findings derived from direct codebase analysis:

| File | Purpose | Key Lines |
|------|---------|-----------|
| `pkg/types/types.go` | Core types, CxMetrics evidence fields | 88, 123-332 |
| `pkg/types/scoring.go` | SubScore type (extension target) | 20-26 |
| `internal/scoring/scorer.go` | MetricExtractor, extractCx, scoreMetrics | 14, 176-406 |
| `internal/pipeline/pipeline.go` | Pipeline orchestration, stage ordering | 164-351 |
| `internal/output/html.go` | HTMLGenerator, HTMLSubScore, template data | 25-68, 94-138, 197-225 |
| `internal/output/templates/report.html` | HTML template structure, JS patterns | 39-98, 125-159 |
| `internal/output/templates/styles.css` | CSS architecture, existing patterns | 1-567 |
| `internal/output/json.go` | JSON output format | 12-115 |
| `internal/output/descriptions.go` | Metric description system | 1-1207 |
| `internal/recommend/recommend.go` | Recommendation generation pattern | 1-371 |
| `internal/analyzer/c7_agent/agent.go` | C7 trace data pattern (precedent) | 56-206 |
| `internal/pipeline/interfaces.go` | Analyzer interface (NOT to modify) | 17-19 |

**Confidence: HIGH.** All integration points verified against actual source code with line-number references. This is an internal architecture question answered entirely by reading the codebase.
