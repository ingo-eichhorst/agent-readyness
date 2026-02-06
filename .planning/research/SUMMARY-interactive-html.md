# Project Research Summary: Interactive HTML Report Enhancements

**Project:** ARS v0.0.5 - Interactive HTML Report Enhancements (Modals, Traces, AI Prompts)
**Domain:** Developer tool static analysis reports with AI agent integration
**Researched:** 2026-02-06
**Confidence:** HIGH

## Executive Summary

The ARS v0.0.5 milestone adds interactive modals to the HTML report for displaying call traces (especially C7 agent evaluation details) and AI improvement prompts. The research reveals this is a **zero new dependencies** implementation that leverages existing Go stdlib patterns and modern web platform features. All required data already exists in the codebase -- it just needs to flow through the scoring pipeline to the HTML renderer.

The recommended approach is additive and low-risk: extend `SubScore` with an `Evidence` field to carry trace data through the existing pipeline, generate improvement prompts after scoring, and render both in native HTML `<dialog>` modals with vanilla JavaScript. The critical architectural insight is that analyzers already capture all necessary evidence (top offending functions, duplicate blocks, dead exports, etc.) in structured CxMetrics types -- this data is simply discarded during scoring today.

Key risks are HTML file size explosion (mitigated by truncation strategy and size budget), modal accessibility (mitigated by native `<dialog>` element), and generic non-actionable prompts (mitigated by evidence interpolation). The file size constraint is critical: reports must remain self-contained and under 500KB even with full C7 trace data embedded.

## Key Findings

### Recommended Stack

**Stack research reveals zero new Go dependencies are needed.** All capabilities map to existing tools already in the project or Go stdlib:

**Core technologies:**
- **Native HTML `<dialog>` element** (95.81% browser support): Modal overlays with built-in focus trapping, Escape key handling, and `::backdrop` styling. Replaces need for modal libraries.
- **Vanilla JavaScript** (~70 lines total): Modal open/close, copy-to-clipboard, basic JSON/command syntax highlighting. Already following existing inline JS pattern.
- **Go stdlib types extension** (SubScore + Evidence): Evidence data flows through existing scoring pipeline with one new field. No new types needed, just field additions.
- **shields.io badge URLs** (pure string formatting): No SVG generation libraries needed, just URL formatting with `net/url.PathEscape`.

**What NOT to add (explicitly rejected):**
- Any JavaScript libraries (highlight.js, Prism, Alpine.js, jQuery): All capabilities achievable with ~70 lines of vanilla JS.
- Anthropic SDK: Remove it entirely (v0.0.3). Claude Code headless replaces direct API for both C4 and C7.
- BibTeX/CSL tooling: Manual citation format with ~100 citations is faster than tooling overhead.
- SVG generation libraries: shields.io URLs are simpler and industry-standard.

**Version requirements:**
- Go 1.21+ (already required): `html/template`, `embed`, `net/url` stdlib
- Browser support: 95.81% for `<dialog>`, 95.68% for `navigator.clipboard.writeText()`

### Expected Features

**Must have (table stakes):**
- Modal overlay with backdrop, close via X/Escape/click-outside: Universal UI pattern, users expect all three dismissal methods
- Scrollable modal content: Trace data can be long (C7 prompts/responses), non-scrollable modals are unusable
- Copy-to-clipboard for prompts: The entire point of improvement prompts is to paste into AI agents
- Metric-specific data in prompts: Generic "improve your code" is worthless; prompts must reference actual scores, files, functions
- C7 full prompt/response display: Show exactly what the agent was asked and how it responded

**Should have (competitive differentiators):**
- Research-backed prompt templates: No other code analysis tool generates AI-optimized prompts with academic citations
- Score trace visualization: Show heuristic breakdown (base score + indicators = final) for full transparency
- Worst-offender file lists in prompts: Including specific file names makes prompts immediately actionable
- Smart defaults for modal state: Low-scoring categories start expanded, high-scoring collapsed

**Defer (v2+):**
- Live "Run Fix" from report: Requires server, authentication, workspace management. Reports are documents, not IDEs.
- Editable prompts within modal: Adds text editor complexity. Users should copy, then edit in their agent.
- Real-time trace streaming: Fundamentally incompatible with static HTML report architecture.
- Per-file drill-down from trace: Would embed entire source in HTML, privacy/security concerns.

### Architecture Approach

**The architecture is additive, not invasive.** Evidence already exists in CxMetrics structs (FunctionMetric, DuplicateBlock, DeadExport, FileChurn, etc.) but is discarded when extractors convert to raw float values. The integration strategy extends the scoring pipeline to carry evidence alongside scores, then generates prompts after scoring using that evidence data.

**Major components:**

1. **Evidence Flow** (pkg/types/scoring.go + internal/scoring/scorer.go)
   - Add `EvidenceItem` type and `Evidence []EvidenceItem` field to `SubScore`
   - Extend `MetricExtractor` signature with third return value: `map[string][]EvidenceItem`
   - extractC1-C7 functions populate evidence from existing CxMetrics data
   - Evidence flows to HTML renderer without new pipeline plumbing

2. **Prompt Generation** (internal/recommend/prompts.go - NEW file)
   - `GeneratePrompts(scored, config) -> []ImprovementPrompt`
   - Runs after scoring, before HTML rendering
   - Uses evidence data to populate metric-specific prompts with actual file names and values
   - Structured format: Context, Build/Test Commands, Task, Current/Target State, Verification

3. **Modal UI** (internal/output/templates/report.html + styles.css)
   - Native `<dialog>` element with `showModal()` / `close()` JavaScript
   - Single reusable dialog populated on-demand from JSON data island
   - ~70 lines total JavaScript: modal control, clipboard, syntax highlighting
   - Progressive enhancement: fallback to `<details>/<summary>` without JS

**Data flow:**
```
AnalysisResult (CxMetrics with evidence)
  -> Scorer extractCx (now returns evidence)
  -> SubScore (with Evidence field)
  -> PromptGenerator (uses evidence)
  -> HTMLGenerator (renders modals)
  -> Self-contained HTML file
```

### Critical Pitfalls

1. **HTML file size explosion from embedded trace data**
   - C7 trace data (prompts, responses, score breakdowns) can add 50-150KB per evaluation
   - Avoid: Set 500KB hard budget, truncate at embedding layer (not capture layer), use single JSON data island instead of duplicated hidden divs
   - Test: Assert HTML file size under budget in CI

2. **Modal accessibility failures without keyboard navigation**
   - Native `<dialog>` element provides focus trapping, Escape key, and backdrop for free
   - Avoid: Use `<dialog>` with `showModal()`, not custom div modals. Add `aria-labelledby` and focus management.
   - Test: Tab/Shift-Tab cycles within modal, Escape closes, focus returns to trigger

3. **JSON schema breaking changes for existing consumers**
   - --baseline comparison and CI parsers depend on stable schema
   - Avoid: All new fields use `json:"omitempty"`, add new fields alongside existing (never rename/restructure), version checking for baseline files
   - Test: Load v1 baseline file, verify comparison still works

4. **Generic non-actionable improvement prompts**
   - Templates without interpolation produce "reduce complexity" advice that users already know
   - Avoid: Every prompt must interpolate actual metric value, target, and specific file/function names from evidence data
   - Test: Validate every template contains at least one interpolated value

5. **Mobile/responsive modal breakage**
   - iOS Safari scroll lock failures, viewport overflow, touch target sizing
   - Avoid: Modal width `min(90vw, 700px)`, max-height `80vh`, position:fixed scroll lock for iOS, 44px minimum tap targets
   - Test: Actual iOS Safari testing, not just Chrome DevTools

## Implications for Roadmap

Based on research, the milestone naturally decomposes into five sequential phases with clear dependencies:

### Phase 1: Evidence Types and Data Flow
**Rationale:** All downstream phases depend on evidence flowing through the scoring pipeline. This is the foundation.

**Delivers:**
- `EvidenceItem` type in pkg/types/scoring.go
- `Evidence` field on `SubScore` type
- Extended `MetricExtractor` signature (three return values)
- All extractC1-C7 functions updated to return evidence
- Evidence visible in JSON output (simplest consumer, validates data flow)

**Addresses:** Data availability requirements from FEATURES.md

**Avoids:** Integration pitfall from ARCHITECTURE.md (modifying analyzer interface -- we extract from existing CxMetrics instead)

**Research needed:** None (standard Go type extension)

### Phase 2: Modal UI Infrastructure
**Rationale:** Prompts (Phase 3) need somewhere to display. Modal infrastructure must be stable before prompt integration.

**Delivers:**
- Native `<dialog>` element implementation with vanilla JS
- Modal open/close with X/Escape/click-outside
- Copy-to-clipboard with fallback for `file://` protocol
- Mobile-responsive CSS with iOS scroll lock
- Progressive enhancement fallback (`<details>/<summary>`)

**Uses:**
- HTML `<dialog>` element (STACK.md: 95.81% browser support)
- `navigator.clipboard.writeText()` (STACK.md: 95.68% support)
- Vanilla JavaScript (~70 lines, STACK.md: custom inline approach)

**Implements:** Modal component from ARCHITECTURE.md shared infrastructure

**Avoids:**
- Pitfall #2: Modal accessibility failures (use native `<dialog>`)
- Pitfall #5: Mobile breakage (responsive CSS from day 1)
- Pitfall #6: Progressive enhancement failure (`<details>` fallback)
- Pitfall #7: Copy-to-clipboard failures (fallback chain)

**Research needed:** None (web platform standards, verified browser support)

### Phase 3: Improvement Prompt Generation
**Rationale:** Requires evidence data (Phase 1) and modal UI (Phase 2) to be functional. Prompt quality iteration depends on seeing rendered output.

**Delivers:**
- internal/recommend/prompts.go with `GeneratePrompts()` function
- 7 category-level prompt templates (MVP: one per category, not per-metric)
- Prompt structure: Context, Build/Test Commands, Task, Current/Target State, Verification
- Prompt data population from `SubScore.Evidence` (actual file names, metric values)
- Integration in pipeline: call GeneratePrompts after scoring, before HTML rendering

**Addresses:**
- Table stakes from FEATURES.md: metric-specific data in prompts
- Differentiator from FEATURES.md: research-backed prompt templates

**Avoids:** Pitfall #4 (generic prompts) by interpolating evidence data

**Research needed:** Per-metric template refinement (defer to Phase 3 execution based on evidence data availability)

### Phase 4: C7 Trace Modal Rendering
**Rationale:** C7 trace data already exists (`C7DebugSample`), making this the easiest trace to render. Proves the modal + evidence pattern before tackling C1-C6.

**Delivers:**
- C7 trace modal showing prompt, response, score breakdown
- Syntax highlighting for JSON responses and shell commands (~30 lines custom JS)
- Score trace visualization (base score + indicators = final)
- Per-sample display (all 3 samples per metric)

**Uses:**
- C7DebugSample data (already captured, FEATURES.md: "data exists")
- Custom JSON/command highlighter (STACK.md: ~30 lines, no libraries)

**Implements:** C7 trace display from FEATURES.md table stakes

**Avoids:** Pitfall #1 (file size explosion) by truncating responses at embedding layer, setting 500KB budget

**Research needed:** None (data already exists, rendering is template work)

### Phase 5: C1-C6 Scoring Explanation Traces
**Rationale:** More complex than C7 (no command replay), requires "scoring explanation" approach showing breakpoints and worst offenders.

**Delivers:**
- Scoring explanation modals for C1-C6 metrics
- Display: metric description, raw value, scoring breakpoints, where value falls, worst offenders
- Reuses evidence data from Phase 1 (FunctionMetric, DuplicateBlock, etc.)

**Addresses:** FEATURES.md: C1-C6 scoring explanation trace (distinct from command replay)

**Implements:** ARCHITECTURE.md scoring explanation pattern (not command replay)

**Avoids:** ARCHITECTURE.md anti-pattern (not modifying analyzers)

**Research needed:** None (evidence extraction pattern established in Phase 1)

### Phase Ordering Rationale

- **Sequential dependency chain:** Phase 1 (evidence types) -> Phase 2 (modal UI) -> Phase 3 (prompts) -> Phase 4 (C7 traces) -> Phase 5 (C1-C6 traces)
- **Risk management:** Phase 4 uses existing C7DebugSample data (low risk) before Phase 5 tackles evidence extraction (medium risk)
- **Incremental value:** Phase 2 enables manual content viewing, Phase 3 adds copyable prompts, Phase 4 adds C7 transparency, Phase 5 completes the feature
- **Testing isolation:** Each phase has clear deliverables and can be tested independently

### Research Flags

**No deeper research needed for any phase:**
- Phase 1: Standard Go type extension, compiler-enforced pattern
- Phase 2: Web platform standards (`<dialog>`, Clipboard API), browser support verified
- Phase 3: Template authoring, evidence interpolation established
- Phase 4: Data already exists, rendering is straightforward
- Phase 5: Extends Phase 1 evidence extraction pattern

**All phases use well-documented patterns:**
- Evidence flow: Extend existing SubScore type (Go struct field addition)
- Modal UI: Native HTML `<dialog>` (MDN docs, 95.81% browser support)
- Prompts: Template string composition (Go string formatting)
- Trace display: HTML template rendering (existing pattern in report.html)

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | All technologies verified against official sources (MDN, Can I Use, Go stdlib docs); zero new dependencies validated |
| Features | HIGH | Feature requirements scoped by GitHub issues #56 and #57; data availability verified by codebase analysis with line references |
| Architecture | HIGH | All integration points identified with file:line references; additive changes only, no invasive modifications |
| Pitfalls | HIGH | Grounded in codebase analysis (current HTML size, JSON schema, existing JS patterns); precedent from Cucumber HTML formatter file size issue |

**Overall confidence:** HIGH

### Gaps to Address

**File size budget enforcement:**
- Current report: ~165KB. Budget: 500KB. Gap: Need test that asserts output size.
- Handle during Phase 1: Add test in internal/output/html_test.go that generates report with C7 data, asserts size < 500KB

**Prompt template quality validation:**
- Gap: No way to validate 7 category-level templates produce actionable prompts without user feedback
- Handle during Phase 3 execution: Manual review of generated prompts, iterate based on interpolated values, consider A/B testing if templates prove generic

**iOS Safari scroll lock verification:**
- Gap: Cannot verify iOS-specific behavior without real device
- Handle during Phase 2: Use BrowserStack or local iOS device for testing, document the position:fixed workaround

**Baseline file version migration:**
- Gap: Version "1" files exist in the wild, no migration logic yet
- Handle during Phase 1: Implement version checking in baseline loader, handle missing fields gracefully

## Sources

### Primary (HIGH confidence)

**Codebase Analysis:**
- /Users/ingo/agent-readyness/pkg/types/types.go (lines 88, 123-332) - CxMetrics evidence fields, C7DebugSample structure
- /Users/ingo/agent-readyness/pkg/types/scoring.go (lines 20-26) - SubScore type (extension target)
- /Users/ingo/agent-readyness/internal/scoring/scorer.go (lines 14, 176-406) - MetricExtractor pattern, extractCx functions
- /Users/ingo/agent-readyness/internal/output/html.go (lines 25-68, 94-138) - HTMLGenerator, template data structures
- /Users/ingo/agent-readyness/internal/output/templates/report.html (lines 39-98, 125-159) - Existing JS patterns, expand/collapse
- /Users/ingo/agent-readyness/internal/output/descriptions.go (lines 1-1207) - Metric descriptions with "How to Improve" sections
- /Users/ingo/agent-readyness/internal/recommend/recommend.go (lines 12-74) - Recommendation system, action templates

**Web Platform Standards:**
- [MDN: dialog element](https://developer.mozilla.org/en-US/docs/Web/HTML/Reference/Elements/dialog) - Native dialog capabilities, showModal() API
- [Can I Use: dialog](https://caniuse.com/dialog) - 95.81% global browser support, Baseline Widely Available since March 2022
- [MDN: Clipboard.writeText()](https://developer.mozilla.org/en-US/docs/Web/API/Clipboard/writeText) - Clipboard API reference, browser support
- [Can I Use: Clipboard writeText](https://caniuse.com/mdn-api_clipboard_writetext) - 95.68% global browser support

**Stack Research:**
- [Claude Code Headless Documentation](https://code.claude.com/docs/en/headless) - Official headless mode docs
- [Shields.io Static Badge](https://shields.io/badges) - Badge URL format and encoding rules
- [Go stdlib documentation](https://pkg.go.dev/std) - html/template, encoding/json, net/url, io.Writer patterns

### Secondary (MEDIUM confidence)

**UI Patterns:**
- [W3C WAI ARIA APG: Dialog (Modal) Pattern](https://www.w3.org/WAI/ARIA/apg/patterns/dialog-modal/) - Modal accessibility requirements
- [A11Y Collective: Mastering Accessible Modals](https://www.a11y-collective.com/blog/modal-accessibility/) - Focus trapping patterns
- [Jay Freestone: Locking Body Scroll for Modals on iOS](https://www.jayfreestone.com/writing/locking-body-scroll-ios/) - iOS scroll lock implementation

**Syntax Highlighting:**
- [JSON Syntax Highlighting Gist](https://gist.github.com/faffyman/6183311) - Regex-based JSON highlighting pattern (~15 lines)

**Prompt Engineering Research (from Issue #57):**
- [Enhancing LLM Code Generation with Complexity Metrics (2025)](https://arxiv.org/html/2505.23953) - Explicit metric targets improve quality by 35%
- [Augmenting LLMs with Static Code Analysis (2025)](https://arxiv.org/html/2506.10330v1) - Static analysis output as prompt context
- [Impact of AGENTS.md Files (ICSE JAWs 2026)](https://arxiv.org/abs/2601.20404) - Context files reduce runtime by 28.64%

### Tertiary (Context - file size precedent)

- [Cucumber HTML Formatter Issue #62](https://github.com/cucumber/html-formatter/issues/62) - 314MB report from embedded trace data (direct precedent for file size explosion pitfall)

---
*Research completed: 2026-02-06*
*Ready for roadmap: yes*
