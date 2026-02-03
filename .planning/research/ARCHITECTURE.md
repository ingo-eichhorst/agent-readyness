# Architecture Research: v0.0.3 Feature Integration

**Project:** Agent Readiness Score CLI
**Researched:** 2026-02-03
**Confidence:** HIGH (based on direct codebase analysis)

---

## Executive Summary

The ARS codebase has a clean, well-defined pipeline architecture that makes the v0.0.3 changes straightforward to integrate. The key insight is that the changes involve four distinct concerns:

1. **Claude Code Integration** - Replacing Anthropic SDK with CLI invocation (internal/agent/ already exists and does this for C7)
2. **Badge Generation** - New output format alongside JSON/HTML/Terminal (fits output/ package pattern)
3. **HTML Enhancements** - Template modifications with JavaScript for interactivity
4. **Analyzer Reorganization** - Directory restructure requiring import path updates

The existing architecture supports all these changes with minimal disruption. The most impactful change is analyzer reorganization, which touches import paths across the codebase.

---

## Current Architecture (Post-v0.0.2)

```
cmd/scan.go                    # CLI entry point, flag handling
    |
    v
internal/pipeline/pipeline.go  # Orchestrator: discover -> parse -> analyze -> score -> recommend -> output
    |
    +-- internal/discovery/    # Filesystem walk, file classification
    +-- internal/parser/       # Go: go/packages, Python/TS: Tree-sitter
    +-- internal/analyzer/     # C1-C7 analyzers (31 files, flat structure)
    +-- internal/scoring/      # Config + weighted score calculation
    +-- internal/recommend/    # Generate improvement suggestions
    +-- internal/output/       # Terminal, JSON, HTML renderers
    +-- internal/llm/          # Anthropic SDK client (C4 LLM, C7 scoring)
    +-- internal/agent/        # Claude CLI executor for C7
```

### Key LLM Usage Points

| Component | LLM Usage | Current Implementation |
|-----------|-----------|----------------------|
| C4 Analyzer | Documentation quality evaluation | `internal/llm.Client` (Anthropic SDK) |
| C7 Analyzer | Task execution | `internal/agent.Executor` (Claude CLI) |
| C7 Analyzer | Response scoring | `internal/llm.Client` (Anthropic SDK) |

---

## Question 1: Claude Code Integration

### Current State Analysis

**C4 Analyzer** (`internal/analyzer/c4_documentation.go`, lines 121-193):
- Uses `llm.Client.EvaluateContent()` for documentation quality
- Four evaluation calls: README clarity, example quality, completeness, cross-reference coherence
- Requires `ANTHROPIC_API_KEY` environment variable

**C7 Analyzer** (`internal/analyzer/c7_agent.go`, lines 36-139):
- Uses `agent.Executor.ExecuteTask()` for Claude CLI invocation (already CLI-based)
- Uses `agent.Scorer.Score()` with `llm.Client` for response scoring (SDK-based)

**internal/agent/executor.go** (lines 42-118):
- Already implements Claude CLI invocation pattern
- Uses `claude -p "prompt" --output-format json`
- Handles timeout, graceful cancellation, JSON parsing

### Recommended Integration: Extend agent.Executor

**The internal/agent package already has the Claude CLI invocation pattern. Extend it for evaluation tasks.**

```go
// internal/agent/executor.go - Add new method

// EvaluateContent uses Claude CLI to evaluate content quality.
// This replaces the Anthropic SDK for C4 documentation evaluation.
func (e *Executor) EvaluateContent(ctx context.Context, systemPrompt, content string) (EvaluationResult, error) {
    // Build prompt that includes system context and content
    fullPrompt := fmt.Sprintf("%s\n\nContent to evaluate:\n%s", systemPrompt, content)

    cmd := exec.CommandContext(ctx, "claude",
        "-p", fullPrompt,
        "--output-format", "json",
    )

    output, err := cmd.CombinedOutput()
    if err != nil {
        return EvaluationResult{}, fmt.Errorf("claude eval failed: %w", err)
    }

    // Parse JSON response - expect {"score": N, "reason": "..."}
    return parseEvaluationResponse(output)
}

type EvaluationResult struct {
    Score     int
    Reasoning string
}
```

### Data Flow Change

```
BEFORE (C4 with SDK):
C4Analyzer.runLLMAnalysis()
  -> llm.Client.EvaluateContent()
  -> anthropic-sdk-go
  -> HTTPS -> Anthropic API

AFTER (C4 with CLI):
C4Analyzer.runLLMAnalysis()
  -> agent.Executor.EvaluateContent()
  -> exec.Command("claude", "-p", ...)
  -> Claude CLI handles auth
```

### Impact Analysis

| Component | Change Required | Effort |
|-----------|----------------|--------|
| `internal/agent/executor.go` | Add `EvaluateContent()` method | Medium |
| `internal/analyzer/c4_documentation.go` | Replace `llm.Client` calls with `agent.Executor` | Medium |
| `internal/analyzer/c7_agent.go` | Replace scorer's `llm.Client` with `agent.Executor` | Medium |
| `internal/llm/client.go` | Mark deprecated or remove | Low |
| `internal/llm/cost.go` | Keep for estimation (still useful) | None |
| `internal/llm/prompts.go` | Keep (reuse system prompts) | None |
| `cmd/scan.go` | Update flag handling (no ANTHROPIC_API_KEY for C4) | Low |
| `internal/pipeline/pipeline.go` | Remove `llm.Client` injection, use `agent.Executor` | Low |

### API Key Impact

**Before:** Both C4 LLM and C7 require `ANTHROPIC_API_KEY`

**After:** Neither requires explicit API key - Claude CLI handles authentication

**Benefits:**
- Simpler user experience (no API key management)
- CLI handles auth, rate limiting, retries
- Consistent interface for all LLM operations

### Interface Compatibility

The existing `llm.Client.EvaluateContent()` signature:
```go
func (c *Client) EvaluateContent(ctx context.Context, systemPrompt, content string) (Evaluation, error)
```

The proposed `agent.Executor.EvaluateContent()` signature:
```go
func (e *Executor) EvaluateContent(ctx context.Context, systemPrompt, content string) (EvaluationResult, error)
```

**Near-identical signatures make migration straightforward.** Only type name changes (`Evaluation` -> `EvaluationResult`).

---

## Question 2: Badge Generation

### Current Output Architecture

```
internal/output/
├── terminal.go      # RenderSummary(), RenderScores(), RenderRecommendations()
├── json.go          # BuildJSONReport(), RenderJSON()
├── html.go          # HTMLGenerator.GenerateReport()
├── charts.go        # generateRadarChart(), generateTrendChart()
└── templates/
    ├── report.html
    └── styles.css
```

**Pipeline output stage** (pipeline.go lines 221-247):
```go
// Stage 4: Render output
if p.jsonOutput {
    output.RenderJSON(p.writer, report)
} else {
    output.RenderSummary(p.writer, ...)
    output.RenderScores(p.writer, ...)
    output.RenderRecommendations(p.writer, recs)
}
if p.htmlOutput != "" && p.scored != nil {
    p.generateHTMLReport(recs)
}
```

### Recommended Integration: New Output Flag

**Badge generation is a separate output format, following the existing pattern.**

```go
// cmd/scan.go - Add flag
var badgeOutput string // Path to SVG badge file

// init()
scanCmd.Flags().StringVar(&badgeOutput, "output-badge", "", "generate SVG badge at specified path")
```

```go
// internal/output/badge.go - New file
package output

import (
    "io"
    "github.com/ingo/agent-readyness/pkg/types"
)

// BadgeGenerator creates SVG badges from scored results.
type BadgeGenerator struct{}

func NewBadgeGenerator() *BadgeGenerator {
    return &BadgeGenerator{}
}

// GenerateBadge writes an SVG badge to w.
// Format: [ARS | 7.2 | Agent-Assisted] with tier-appropriate colors
func (g *BadgeGenerator) GenerateBadge(w io.Writer, scored *types.ScoredResult) error {
    // SVG generation - shields.io-style badge
    // Green for Agent-Ready, Yellow for Agent-Assisted, Red for others
}
```

```go
// internal/pipeline/pipeline.go - Add to Stage 4
if p.badgeOutput != "" && p.scored != nil {
    if err := p.generateBadge(); err != nil {
        return fmt.Errorf("generate badge: %w", err)
    }
}

func (p *Pipeline) generateBadge() error {
    gen := output.NewBadgeGenerator()
    f, err := os.Create(p.badgeOutput)
    if err != nil {
        return err
    }
    defer f.Close()
    return gen.GenerateBadge(f, p.scored)
}
```

### Badge Format: Self-Contained SVG

| Format | Pros | Cons |
|--------|------|------|
| **SVG (Recommended)** | Scalable, small file, GitHub-friendly | Slightly more complex generation |
| PNG | Universal | Requires image library (bloat) |
| Shields.io URL | No generation needed | External dependency, privacy concerns |

**Recommendation: Self-contained SVG.**

SVG badge example structure:
```svg
<svg xmlns="http://www.w3.org/2000/svg" width="140" height="20">
  <linearGradient id="b" x2="0" y2="100%">
    <stop offset="0" stop-color="#bbb" stop-opacity=".1"/>
    <stop offset="1" stop-opacity=".1"/>
  </linearGradient>
  <rect rx="3" width="140" height="20" fill="#555"/>
  <rect rx="3" x="35" width="105" height="20" fill="#4c1"/>
  <rect rx="3" width="140" height="20" fill="url(#b)"/>
  <g fill="#fff" text-anchor="middle" font-family="DejaVu Sans,sans-serif" font-size="11">
    <text x="18" y="15">ARS</text>
    <text x="87" y="15">8.2 Agent-Ready</text>
  </g>
</svg>
```

### Pipeline Integration Point

```
Stage 4 (render output)
  |
  +-- Terminal: stdout (existing)
  +-- JSON: file (existing)
  +-- HTML: file (existing)
  +-- Badge (NEW): file (SVG)
```

---

## Question 3: HTML Enhancements

### Current HTML Structure

```go
// internal/output/html.go
//go:embed templates/report.html templates/styles.css
var templateFS embed.FS

type HTMLGenerator struct {
    tmpl *template.Template
}
```

**Current template** (templates/report.html, lines 25-47):
```html
<section class="categories">
    {{range .Categories}}
    <div class="category">
        <h2>{{.DisplayName}} <span class="cat-score">{{.Score}}/10</span></h2>
        <table class="metric-table">
            <!-- metrics rows -->
        </table>
    </div>
    {{end}}
</section>
```

### Recommended Enhancement: HTML5 details/summary

**Use native HTML5 expandable elements - no JavaScript required for basic functionality.**

```html
<section class="categories">
    {{range .Categories}}
    <div class="category">
        <details {{if .DefaultOpen}}open{{end}}>
            <summary>
                <h2>{{.DisplayName}} <span class="cat-score score-{{.ScoreClass}}">{{printf "%.1f" .Score}}/10</span></h2>
            </summary>
            <table class="metric-table">
                <!-- metrics rows -->
            </table>
            {{if .ImpactDescription}}<p class="impact">{{.ImpactDescription}}</p>{{end}}
        </details>
    </div>
    {{end}}
</section>
```

**Add expand/collapse all (optional JavaScript enhancement):**
```html
<div class="controls">
    <button onclick="toggleAll(true)">Expand All</button>
    <button onclick="toggleAll(false)">Collapse All</button>
</div>

<script>
function toggleAll(expand) {
    document.querySelectorAll('details').forEach(d => d.open = expand);
}
</script>
```

### CSS Updates Required

```css
/* templates/styles.css - Add to existing */

details summary {
    cursor: pointer;
    list-style: none; /* Remove default arrow */
}

details summary::-webkit-details-marker {
    display: none;
}

details summary::before {
    content: '+ ';
    font-weight: bold;
}

details[open] summary::before {
    content: '- ';
}

details[open] {
    margin-bottom: 1.5rem;
}

.controls {
    margin: 1rem 0;
    display: flex;
    gap: 0.5rem;
}

.controls button {
    padding: 0.5rem 1rem;
    border: 1px solid var(--color-border);
    border-radius: 0.25rem;
    background: var(--color-surface);
    cursor: pointer;
}
```

### Impact Analysis

| File | Change | Effort |
|------|--------|--------|
| `templates/report.html` | Wrap categories in details/summary | Medium |
| `templates/styles.css` | Add details/summary styles | Low |
| `internal/output/html.go` | Optional: Add `DefaultOpen` field to `HTMLCategory` | Low |

### Browser Support

`<details>/<summary>` has 96%+ browser support. No polyfill needed.

---

## Question 4: Analyzer Reorganization

### Current Structure (31 files, flat)

```
internal/analyzer/
├── helpers.go              # Shared utilities
├── c1_codehealth.go        # Main C1 coordinator
├── c1_codehealth_test.go
├── c1_python.go            # C1 Python-specific
├── c1_python_test.go
├── c1_typescript.go        # C1 TypeScript-specific
├── c1_typescript_test.go
├── c2_semantics.go         # Main C2 coordinator
├── c2_go.go
├── c2_go_test.go
├── c2_python.go
├── c2_python_test.go
├── c2_typescript.go
├── c2_typescript_test.go
├── c3_architecture.go
├── c3_architecture_test.go
├── c3_python.go
├── c3_python_test.go
├── c3_typescript.go
├── c3_typescript_test.go
├── c4_documentation.go
├── c4_documentation_test.go
├── c5_temporal.go
├── c5_temporal_test.go
├── c6_testing.go
├── c6_testing_test.go
├── c6_python.go
├── c6_python_test.go
├── c6_typescript.go
├── c6_typescript_test.go
├── c7_agent.go
└── c7_agent_test.go
```

### Proposed Structure (by category)

```
internal/analyzer/
├── analyzer.go              # Interfaces, common types, re-exports
├── helpers.go               # Shared utilities (keep at root)
├── c1/
│   ├── analyzer.go          # C1Analyzer, NewC1Analyzer(), Analyze()
│   ├── analyzer_test.go
│   ├── go.go                # Go-specific analysis (from c1_codehealth.go)
│   ├── python.go            # From c1_python.go
│   ├── python_test.go
│   ├── typescript.go        # From c1_typescript.go
│   └── typescript_test.go
├── c2/
│   ├── analyzer.go
│   ├── analyzer_test.go
│   ├── go.go
│   ├── go_test.go
│   ├── python.go
│   ├── python_test.go
│   ├── typescript.go
│   └── typescript_test.go
├── c3/
│   └── ... (same pattern)
├── c4/
│   ├── analyzer.go          # No language variants
│   └── analyzer_test.go
├── c5/
│   ├── analyzer.go          # No language variants (git-based)
│   └── analyzer_test.go
├── c6/
│   └── ... (has language variants)
└── c7/
    ├── analyzer.go          # No language variants (agent-based)
    └── analyzer_test.go
```

### Import Path Strategy

**Option A: Category sub-packages with root re-exports (RECOMMENDED)**

```go
// internal/analyzer/c1/analyzer.go
package c1

type Analyzer struct { ... }
func NewAnalyzer(tsParser *parser.TreeSitterParser) *Analyzer { ... }
```

```go
// internal/analyzer/analyzer.go - Re-exports for backward compatibility
package analyzer

import (
    "github.com/ingo/agent-readyness/internal/analyzer/c1"
    "github.com/ingo/agent-readyness/internal/analyzer/c2"
    // ...
)

// Re-export constructors for backward compatibility
func NewC1Analyzer(tsParser *parser.TreeSitterParser) *c1.Analyzer {
    return c1.NewAnalyzer(tsParser)
}

func NewC2Analyzer(tsParser *parser.TreeSitterParser) *c2.Analyzer {
    return c2.NewAnalyzer(tsParser)
}
// ... etc
```

**Usage in pipeline.go remains unchanged:**
```go
import "github.com/ingo/agent-readyness/internal/analyzer"

analyzers: []Analyzer{
    analyzer.NewC1Analyzer(tsParser),  // Works via re-export
    analyzer.NewC2Analyzer(tsParser),
    // ...
}
```

### Migration Steps

1. **Create subdirectory structure**
   ```bash
   mkdir -p internal/analyzer/{c1,c2,c3,c4,c5,c6,c7}
   ```

2. **Move files with rename** (one category at a time)
   ```bash
   # C5 first (simplest - no language variants)
   mv c5_temporal.go c5/analyzer.go
   mv c5_temporal_test.go c5/analyzer_test.go
   # Update package declaration: package c5
   ```

3. **Create re-exports** in `internal/analyzer/analyzer.go`

4. **Update internal imports** within analyzer package
   - Category files may reference helpers.go
   - Keep helpers.go at root, import as `"github.com/ingo/agent-readyness/internal/analyzer"`

5. **Run tests after each category**
   ```bash
   go test ./internal/analyzer/...
   ```

6. **Update pipeline imports** (may not be needed if re-exports work)

### Category Migration Order

| Order | Category | Complexity | Notes |
|-------|----------|------------|-------|
| 1 | C5 | Low | Single file, no language variants, git-based |
| 2 | C7 | Low | Single file, no language variants, agent-based |
| 3 | C4 | Low | Single file, no language variants |
| 4 | C1 | Medium | Has Go/Python/TypeScript variants |
| 5 | C2 | Medium | Has Go/Python/TypeScript variants |
| 6 | C3 | Medium | Has Go/Python/TypeScript variants |
| 7 | C6 | Medium | Has Go/Python/TypeScript variants |

### Risk Assessment

| Risk | Severity | Mitigation |
|------|----------|------------|
| Broken imports | High | Compile-time detection, re-exports preserve compatibility |
| Test failures | Medium | Run `go test ./...` after each category move |
| Circular imports | Low | Categories are independent, no cross-category dependencies |

---

## Component Diagram

```
                                 cmd/scan.go
                                      |
                                      v
                        +------------------------+
                        |   internal/pipeline/   |
                        |      pipeline.go       |
                        +------------------------+
                                      |
        +------------+----------------+----------------+------------+
        |            |                |                |            |
        v            v                v                v            v
  discovery/    parser/         analyzer/         scoring/     output/
                                    |                           |
                    +---------------+---------------+           |
                    |       |       |       |       |           +---> terminal.go
                   c1/     c2/    c3/     c4/     c5/           |
                    |       |       |       |       |           +---> json.go
                    v       v       v       v       v           |
                   c6/     c7/                                  +---> html.go
                           |                                    |
                           v                                    +---> badge.go (NEW)
                        agent/                                  |
                           |                                    +---> templates/
                           v                                           |
                 (Claude CLI exec)                                     +---> report.html (MODIFIED)
                           ^                                           +---> styles.css (MODIFIED)
                           |
                   +-------+-------+
                   |               |
              C7 tasks      C4 eval (NEW)


Legend:
  [box]     = package/directory
  -->       = imports/calls
  (NEW)     = v0.0.3 addition
  (MODIFIED)= v0.0.3 change
```

---

## Suggested Build Order

Based on dependencies and risk analysis:

### Phase 1: Badge Generation (Low Risk, Fully Additive)

**Why first:**
- No changes to existing code paths
- New file in output package
- New flag in cmd/scan.go
- Easy to test in isolation

**Work items:**
1. Create `internal/output/badge.go`
2. Add `--output-badge` flag to `cmd/scan.go`
3. Add badge generation call in pipeline Stage 4
4. Write tests for badge generator

### Phase 2: HTML Enhancements (Low Risk, Template Only)

**Why second:**
- Template-only changes
- No Go code changes (unless adding DefaultOpen field)
- Can iterate on design
- Easy to preview

**Work items:**
1. Modify `templates/report.html` with details/summary
2. Update `templates/styles.css`
3. Optional: Add expand/collapse all buttons with JavaScript
4. Test in browser

### Phase 3: Analyzer Reorganization (Medium Risk, Structural)

**Why third:**
- More invasive but purely structural
- Compile-time verification
- Do after feature work is stable
- Re-exports minimize downstream impact

**Work items:**
1. Create directory structure
2. Move C5, C7, C4 (simplest, no language variants)
3. Create re-exports in analyzer/analyzer.go
4. Move C1, C2, C3, C6 (with language variants)
5. Update any broken imports
6. Full test suite pass

### Phase 4: Claude Code Integration (Medium Risk, Behavior Change)

**Why last:**
- Most complex change
- Requires careful testing
- Other features can ship independently
- May want to verify CLI stability first

**Work items:**
1. Add `EvaluateContent()` to `internal/agent/executor.go`
2. Update C4 analyzer to use agent.Executor instead of llm.Client
3. Update C7 scorer to use agent.Executor instead of llm.Client
4. Update CLI flag handling (remove ANTHROPIC_API_KEY requirement)
5. Update help text and documentation
6. Integration testing with Claude CLI

---

## Confidence Assessment

| Area | Level | Reason |
|------|-------|--------|
| Badge Generation | HIGH | Standard output pattern, no dependencies, additive |
| HTML Enhancements | HIGH | Template-only changes, well-understood HTML5 features |
| Analyzer Reorganization | HIGH | Clear file structure, compile-time import checks |
| Claude Code Integration | MEDIUM | Need to verify CLI JSON output format stability, prompt handling |

---

## Open Questions

1. **C7 Scoring:** Should C7's LLM-as-judge also migrate to Claude CLI? This would eliminate all Anthropic SDK usage.

2. **Badge Customization:** Should badge support customization (colors, format, size)? Or start simple with defaults?

3. **HTML Features:** Add more interactive features (filtering, sorting metrics)? Or keep minimal for v0.0.3?

4. **Analyzer Tests:** During reorganization, consolidate test utilities into a shared testutil package?

5. **LLM Deprecation:** Remove `internal/llm/` package entirely, or keep `cost.go` and `prompts.go`?

---

## Sources

All findings based on direct codebase analysis:

- `/Users/ingo/agent-readyness/internal/pipeline/pipeline.go` - Pipeline orchestration (425 lines)
- `/Users/ingo/agent-readyness/internal/analyzer/c7_agent.go` - C7 analyzer, CLI pattern (144 lines)
- `/Users/ingo/agent-readyness/internal/agent/executor.go` - Claude CLI invocation (138 lines)
- `/Users/ingo/agent-readyness/internal/agent/scorer.go` - LLM scoring (121 lines)
- `/Users/ingo/agent-readyness/internal/llm/client.go` - Anthropic SDK client (163 lines)
- `/Users/ingo/agent-readyness/internal/analyzer/c4_documentation.go` - C4 analyzer with LLM (873 lines)
- `/Users/ingo/agent-readyness/internal/output/html.go` - HTML generation (313 lines)
- `/Users/ingo/agent-readyness/internal/output/templates/report.html` - HTML template (82 lines)
- `/Users/ingo/agent-readyness/internal/output/templates/styles.css` - CSS styles (304 lines)
- `/Users/ingo/agent-readyness/internal/output/json.go` - JSON output (106 lines)
- `/Users/ingo/agent-readyness/cmd/scan.go` - CLI entry point (243 lines)
