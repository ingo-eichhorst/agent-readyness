# Features Research: ARS v0.0.3

**Domain:** CLI tool enhancement (badge generation, HTML reports, agent migration)
**Researched:** 2026-02-03
**Overall Confidence:** MEDIUM

## Executive Summary

This research covers four feature areas for ARS v0.0.3: badge generation (Issue #5), HTML report enhancements with expandable scientific descriptions (Issue #7), Claude Code migration removing direct API usage (Issue #6), and codebase reorganization (Issues #3, #2, #4). The findings are grounded in established patterns from shields.io badge conventions, WCAG accessibility guidelines for expandable content, and official Claude Code headless mode documentation.

Key findings:
- Badge generation should follow shields.io format conventions with local SVG generation (no external dependencies)
- Expandable HTML sections must use `<details>`/`<summary>` elements or ARIA-compliant buttons for accessibility
- Claude Code migration is straightforward: replace direct Anthropic API calls with `claude -p` subprocess invocation
- Scientific citations should use expandable progressive disclosure to avoid overwhelming users

---

## Badge Generation (Issue #5)

### Table Stakes

| Feature | Why Expected | Complexity | Dependencies |
|---------|--------------|------------|--------------|
| SVG output format | Standard badge format, scales infinitely, works in all contexts | Low | None |
| Score display (X.X/10) | Users expect to see their score on the badge | Low | Scoring module |
| Tier display (Agent-Ready, etc.) | Visual classification matches report | Low | Scoring module |
| Color coding by tier | Green/yellow/orange/red pattern is universal | Low | None |
| Local generation (no network) | CI environments may lack internet; deterministic output | Medium | SVG template |
| `--badge` or `--output-badge` flag | Standard CLI pattern for optional outputs | Low | CLI module |
| Output to file path | `--badge badge.svg` or `--badge ./badges/ars.svg` | Low | CLI module |
| Stdout option | `--badge -` outputs to stdout for piping | Low | CLI module |

**Format expectations (shields.io convention):**
- Left side: label (e.g., "ARS" or "Agent Readiness")
- Right side: value (e.g., "7.2" or "Agent-Ready")
- Color: corresponds to tier classification
- Dimensions: ~90-120px width, 20px height (standard badge size)

**Color mapping (shields.io standard colors):**
| Tier | Color Name | Hex |
|------|------------|-----|
| Agent-Ready (8+) | Bright Green | #97ca00 |
| Agent-Assisted (6-8) | Yellow-Green | #a4a61d |
| Agent-Limited (4-6) | Orange | #fe7d37 |
| Agent-Hostile (<4) | Red | #e05d44 |

**SVG template structure (shields.io flat style):**
```svg
<svg xmlns="http://www.w3.org/2000/svg" width="90" height="20">
  <linearGradient id="b" x2="0" y2="100%">
    <stop offset="0" stop-color="#bbb" stop-opacity=".1"/>
    <stop offset="1" stop-opacity=".1"/>
  </linearGradient>
  <clipPath id="a">
    <rect width="90" height="20" rx="3" fill="#fff"/>
  </clipPath>
  <g clip-path="url(#a)">
    <rect width="31" height="20" fill="#555"/>
    <rect x="31" width="59" height="20" fill="#97ca00"/>
    <rect width="90" height="20" fill="url(#b)"/>
  </g>
  <g fill="#fff" text-anchor="middle" font-family="DejaVu Sans,Verdana,Geneva,sans-serif" font-size="11">
    <text x="16" y="15" fill="#010101" fill-opacity=".3">ARS</text>
    <text x="16" y="14">ARS</text>
    <text x="60" y="15" fill="#010101" fill-opacity=".3">7.2</text>
    <text x="60" y="14">7.2</text>
  </g>
</svg>
```

### Differentiators

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| Markdown snippet output | `--badge-markdown` outputs `![ARS](badge.svg)` ready to paste | Low | Convenience |
| Category mini-badges | Generate individual C1-C7 badges for detailed README | Medium | 7 SVGs instead of 1 |
| Trend indicator | Arrow up/down if baseline provided | Medium | Requires baseline comparison |
| Custom label text | `--badge-label "Code Quality"` | Low | User preference |
| JSON endpoint format | Output shields.io JSON endpoint format for dynamic badges | Low | Alternative to static SVG |

**Shields.io dynamic badge JSON format:**
```json
{
  "schemaVersion": 1,
  "label": "ARS",
  "message": "7.2",
  "color": "green"
}
```
This allows hosting a JSON file that shields.io can read dynamically.

### Anti-features

| Anti-Feature | Why Avoid | What to Do Instead |
|--------------|-----------|-------------------|
| Remote shields.io dependency | Adds network dependency, slower, fails offline | Local SVG generation |
| PNG/raster output | SVG is universally preferred, scales better | SVG only |
| Animated badges | Distracting, accessibility issues | Static badges only |
| Multiple badge styles | Complexity without value; flat style is standard | Single flat style |
| Badge in HTML report | Report already shows score prominently | Badge is for external embedding (README) |

---

## HTML Report Enhancements (Issue #7)

### Table Stakes

| Feature | Why Expected | Complexity | Dependencies |
|---------|--------------|------------|--------------|
| Expandable metric descriptions | Users need to understand what metrics mean | Medium | Template changes |
| `<details>`/`<summary>` elements | Native HTML5, accessible, no JS required | Low | None |
| ARIA attributes (`aria-expanded`) | Screen reader accessibility | Low | Template |
| Keyboard navigation (Enter/Space) | WCAG 2.1 requirement | Low | Native with `<details>` |
| Visual expand/collapse indicators | Clear affordance (chevron, +/-) | Low | CSS |
| Research citations per category | Already exists in citations.go; enhance with expandability | Medium | Existing citations.go |

**Accessibility requirements (WCAG 2.1 compliant):**
- `aria-expanded="true/false"` on toggle buttons (automatic with `<details>`)
- `aria-controls` linking button to content
- Color contrast 4.5:1 for text, 3:1 for icons
- Visible focus indicators
- Content hidden with `hidden` attribute or `display: none` (not just visually)

**Native HTML pattern (recommended - no JavaScript required):**
```html
<details class="metric-detail">
  <summary>
    <span class="metric-name">C1: Code Health</span>
    <span class="expand-icon">+</span>
  </summary>
  <div class="citation-content">
    <h4>What it measures</h4>
    <p>Cyclomatic complexity, function length, file size...</p>

    <h4>Why it matters for agents</h4>
    <p>Lower complexity and smaller functions help agents reason about
       and modify code safely.</p>

    <h4>Research backing</h4>
    <ul>
      <li>McCabe, T.J. (1976). "A Complexity Measure" - Original cyclomatic
          complexity definition. <a href="https://ieeexplore.ieee.org/document/1702388">IEEE</a></li>
      <li>Fowler et al. (1999). "Refactoring" - Complexity as maintainability
          indicator.</li>
    </ul>
  </div>
</details>
```

**CSS for expand/collapse indicator:**
```css
details summary .expand-icon::before { content: "+"; }
details[open] summary .expand-icon::before { content: "-"; }

details summary::-webkit-details-marker { display: none; }
details summary { list-style: none; cursor: pointer; }

@media print {
  details { display: block !important; }
  details > * { display: block !important; }
}
```

### Differentiators

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| "Why this matters for agents" section | Connects metrics to agent-readiness impact | Medium | Content creation |
| Per-metric expandable detail | Click on "Complexity avg" to see calculation method | High | Many expansion points |
| Citation tooltips | Hover to see abbreviated citation | Medium | CSS/JS |
| Print-friendly expansion | All sections expanded in print stylesheet | Low | CSS `@media print` |
| Benchmark context | "Your score vs. SWE-bench projects average" comparison | High | Requires benchmark data |

**Scientific description structure per category:**
```
[Category Name] Score: X.X/10

[Collapsed by default - click to expand]
----------------------------------------
What it measures:
  Brief explanation of the metrics in this category.

Why it matters for agents:
  How this affects LLM code understanding/modification.
  Reference to relevant research (e.g., "23% higher accuracy
  on typed code per CrossCodeEval 2023").

Research backing:
  - McCabe (1976): Original cyclomatic complexity definition
  - NIST235: Recommended complexity threshold of 10
  - SWE-bench: Correlation between code quality and agent success
----------------------------------------
```

### Anti-features

| Anti-Feature | Why Avoid | What to Do Instead |
|--------------|-----------|-------------------|
| JavaScript-only expandables | Fails without JS, accessibility issues | Use native `<details>` or progressive enhancement |
| Auto-expand all on load | Defeats purpose of progressive disclosure | Collapsed by default |
| Deeply nested expansions | Confusing UX, cognitive overload | Max 2 levels of nesting |
| External CSS/JS dependencies | Report should be self-contained | Inline all resources (already the pattern) |
| Accordion (one-at-a-time) | Frustrating when comparing sections | Allow multiple open |

---

## Claude Code Migration (Issue #6)

### Table Stakes

| Feature | Why Expected | Complexity | Dependencies |
|---------|--------------|------------|--------------|
| Remove direct Anthropic API usage | Issue #6 requirement; simplifies auth | Medium | Refactor internal/llm/client.go |
| Use `claude -p` for all LLM calls | Headless mode is official pattern | Medium | Subprocess execution |
| JSON output parsing | `--output-format json` returns structured data | Low | Already in internal/agent/executor.go |
| Graceful degradation when CLI missing | Skip LLM features, don't crash | Low | Already in CheckClaudeCLI() |
| Remove ANTHROPIC_API_KEY requirement | Claude CLI handles auth via `claude login` | Low | Remove from docs |

**Claude Code headless invocation pattern (from official docs):**
```go
cmd := exec.CommandContext(ctx, "claude",
    "-p", prompt,
    "--output-format", "json",
)
output, err := cmd.CombinedOutput()
```

**Response structure:**
```json
{
  "type": "result",
  "session_id": "abc123",
  "result": "The agent's text response"
}
```

**Current usage to migrate:**

| Current Location | Current Method | Migration Target |
|------------------|----------------|------------------|
| `internal/llm/client.go` | Anthropic SDK direct API (`anthropic.NewClient`) | `claude -p` subprocess |
| `internal/analyzer/c7_agent.go` | Uses llm.Client for scoring | Use `claude -p` for scoring |
| C4 LLM analysis (opt-in) | Anthropic SDK | `claude -p` subprocess |

**Unified executor pattern (already exists in internal/agent/executor.go):**
The existing `executor.go` already implements Claude CLI subprocess invocation correctly:
- JSON output parsing
- Timeout handling with graceful SIGINT
- Error handling for missing CLI
- Working directory isolation

Migration approach: Extend/reuse this pattern for C4 LLM evaluation, replacing the direct API client in `internal/llm/client.go`.

### Differentiators

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| Structured output with JSON schema | Use `--json-schema` for typed responses | Medium | Cleaner parsing |
| Custom system prompts | `--append-system-prompt` for evaluation rubrics | Low | Already structured |
| Tool restrictions | `--allowedTools "Read"` for read-only evaluation | Low | Prevents unintended modifications |

**Structured output example (for C4 doc quality scoring):**
```bash
claude -p "Rate this documentation quality 1-10. Return JSON with score and reason." \
  --output-format json \
  --json-schema '{"type":"object","properties":{"score":{"type":"integer","minimum":1,"maximum":10},"reason":{"type":"string"}},"required":["score","reason"]}'
```

### Anti-features

| Anti-Feature | Why Avoid | What to Do Instead |
|--------------|-----------|-------------------|
| Keep Anthropic SDK as fallback | Dual code paths, maintenance burden | Remove entirely |
| Use Python/TypeScript Agent SDK | Go project, subprocess is cleaner | CLI subprocess |
| Interactive mode features | `/commit`, etc. not available in `-p` mode | Describe tasks directly |
| Persist sessions | Evaluation tasks are independent | Fresh session per task |

---

## Codebase Organization (Issue #3)

### Table Stakes

| Feature | Why Expected | Complexity | Dependencies |
|---------|--------------|------------|--------------|
| Group analyzers by category | C1/, C2/, etc. subdirectories | Low | Move files |
| Preserve package structure | `analyzer/c1/` imports as `c1` package | Low | Go packages |
| Update imports | All internal references | Low | Find/replace |
| Consistent naming | `c1/analyzer.go` not `c1/c1_codehealth.go` | Low | Rename files |

**Current structure:**
```
internal/analyzer/
  c1_codehealth.go
  c1_go.go
  c1_python.go
  c1_typescript.go
  c2_semantics.go
  c2_go.go
  ...
  c7_agent.go
```

**Proposed structure (Issue #3 requirement):**
```
internal/analyzer/
  c1/                    # Code Health
    analyzer.go          # Main C1 analyzer
    go.go               # Go-specific implementation
    python.go           # Python-specific
    typescript.go       # TypeScript-specific
  c2/                    # Semantic Explicitness
    analyzer.go
    go.go
    python.go
    typescript.go
  c3/                    # Architecture
    analyzer.go
    ...
  c4/                    # Documentation Quality
  c5/                    # Temporal Dynamics
  c6/                    # Testing
  c7/                    # Agent Evaluation
  registry.go           # Analyzer registration (stays at root)
```

**Import changes:**
```go
// Before
import "github.com/ingo/agent-readyness/internal/analyzer"
analyzer.NewC1Analyzer()

// After
import "github.com/ingo/agent-readyness/internal/analyzer/c1"
c1.NewAnalyzer()
```

---

## Test Coverage Flag (Issue #2)

### Table Stakes

| Feature | Why Expected | Complexity | Dependencies |
|---------|--------------|------------|--------------|
| Always run `go test -coverprofile` | Ensure coverage data available | Low | Makefile/scripts |
| Document in CLAUDE.md | Developer guidance | Low | Documentation |
| CI configuration | GitHub Actions runs with coverage | Low | .github/workflows |

**Implementation:** This is a process/configuration change, not a feature. Update:
1. `CLAUDE.md` commands section (already includes coverage command)
2. `Makefile` (if exists) or document standard command
3. `.github/workflows/*.yml` if CI exists

---

## README Badges (Issue #4)

### Table Stakes

| Feature | Why Expected | Complexity | Dependencies |
|---------|--------------|------------|--------------|
| Add badges to ARS README | Dog-fooding the badge feature | Low | Badge generation working |
| Standard badge placement | Top of README, after title | Low | README.md edit |

**Standard README badge section:**
```markdown
# Agent Readiness Score (ARS)

![ARS](badge.svg) ![Go Version](https://img.shields.io/badge/go-1.21+-blue) ![License](https://img.shields.io/badge/license-MIT-green)

Agent Readiness Score is a CLI tool...
```

---

## Feature Dependencies

```
Badge Generation (Issue #5)
  - Requires: Scoring module (composite score, tier) -- already exists
  - Independent of: HTML report, Claude Code migration
  - Produces: SVG file, optional markdown snippet

HTML Enhancements (Issue #7)
  - Requires: Existing citations.go -- already exists
  - Requires: Template modifications (internal/output/templates/report.html)
  - Independent of: Badge, Claude Code migration

Claude Code Migration (Issue #6)
  - Requires: Claude CLI installed (user environment)
  - Affects: C4 LLM analysis, C7 agent evaluation
  - Removes: internal/llm/client.go dependency on Anthropic SDK
  - Reuses: internal/agent/executor.go patterns

Codebase Reorganization (Issue #3)
  - Independent of: All features
  - Should be done first or last: Either clean base for other changes,
    or clean up after functional changes

README Badges (Issue #4)
  - Requires: Badge generation working (Issue #5)
  - Simple: Just update README.md

Test Coverage (Issue #2)
  - Independent of: All features
  - Documentation/process change only
```

**Recommended implementation order:**
1. **Issue #3** (Codebase reorg) - Clean foundation
2. **Issue #6** (Claude Code migration) - Removes API dependency
3. **Issue #5** (Badge generation) - New feature, self-contained
4. **Issue #7** (HTML enhancements) - Enhances existing report
5. **Issue #4** (README badges) - Depends on #5
6. **Issue #2** (Test coverage) - Documentation update

---

## MVP Recommendation

**For v0.0.3, prioritize:**

1. **Claude Code Migration** (Issue #6)
   - Removes external API dependency (Anthropic SDK)
   - Simplifies authentication (no API key management, uses `claude login`)
   - Aligns with official Anthropic tooling direction
   - Existing executor.go provides the pattern

2. **Badge Generation** (Issue #5)
   - High visibility feature for README embedding
   - Low complexity (SVG template + color mapping)
   - Immediately useful for users

3. **HTML Expandable Sections** (Issue #7)
   - Enhances existing report with scientific backing
   - Accessibility improvements (native HTML5)
   - Educational value explaining why metrics matter

**Lower priority for v0.0.3:**
- Issue #3 (Codebase reorg) - Nice to have, not user-facing
- Issue #4 (README badges) - Quick win after #5
- Issue #2 (Test coverage) - Documentation only

---

## Sources

### Badge Generation
- [Shields.io - Official badge service](https://shields.io/)
- [badges/shields GitHub - Reference implementation](https://github.com/badges/shields)
- [python-genbadge - CLI badge generation patterns](https://smarie.github.io/python-genbadge/)
- [SpaceBadgers - Alternative SVG badge approach](https://github.com/SplittyDev/spacebadgers)
- [Codecov Status Badges](https://docs.codecov.com/docs/status-badges)

### HTML Expandable Sections (Accessibility)
- [Harvard Digital Accessibility - Expandable sections technique](https://accessibility.huit.harvard.edu/technique-expandable-sections)
- [Inclusive Components - Collapsible sections](https://inclusive-components.design/collapsible-sections/)
- [W3C WAI-ARIA - aria-expanded state](https://www.w3.org/WAI/GL/wiki/Using_the_WAI-ARIA_aria-expanded_state_to_mark_expandable_and_collapsible_regions)
- [PatternFly - Expandable section accessibility](https://www.patternfly.org/components/expandable-section/accessibility/)
- [Aditus - Accessible accordion patterns](https://www.aditus.io/patterns/accordion/)
- [A11Y Collective - Accessible accordion components](https://www.a11y-collective.com/blog/accessible-accordion/)

### Claude Code Headless Mode
- [Claude Code Docs - Run programmatically](https://code.claude.com/docs/en/headless)
- [Anthropic - Claude Code best practices](https://www.anthropic.com/engineering/claude-code-best-practices)
- [Claude Code 101 - Headless mode tutorial](https://www.claudecode101.com/en/tutorial/advanced/headless-mode)

### Scientific Metrics Research
- [McCabe (1976) - Cyclomatic complexity](https://ieeexplore.ieee.org/document/1702388)
- [Wikipedia - Cyclomatic complexity with research summary](https://en.wikipedia.org/wiki/Cyclomatic_complexity)
- [NIST235 - Structured testing methodology](https://www.nist.gov/publications/structured-testing-methodology-testing-methodology)
- [SWE-bench - Agent evaluation benchmark](https://www.swebench.com/)
- [SWE-bench GitHub](https://github.com/SWE-bench/SWE-bench)
- [SWE-Bench Pro leaderboard (Jan 2026)](https://scale.com/leaderboard/swe_bench_pro_public)
- [Hatton - Cyclomatic complexity critique](https://www.cs.du.edu/~snarayan/sada/teaching/COMP3705/lecture/p1/cycl-1.pdf)

### Existing Citations in Codebase
The existing `internal/output/citations.go` already contains well-structured research references:
- McCabe (1976) - Cyclomatic complexity
- Fowler (1999) - Refactoring
- Gao et al. (2017) - Type annotations
- Parnas (1972) - Module decomposition
- Tornhill (2015) - Temporal coupling
- Beck (2002) - TDD
- Kim et al. (2007) - Change history defect prediction
