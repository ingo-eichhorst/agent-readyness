# Feature Research: Interactive HTML Report Enhancements

**Domain:** Developer tool HTML reports -- call trace displays and AI improvement prompts
**Researched:** 2026-02-06
**Confidence:** HIGH (features scoped by GitHub issues #56 and #57, codebase thoroughly analyzed)

## Feature Landscape

### Table Stakes (Users Expect These)

Features users assume exist once they see "View Trace" or "Improve" buttons. Missing these = feature feels broken.

#### Call Trace Modal (#56)

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| Modal overlay with backdrop dim | Standard UI pattern for overlays; users will be confused without it | LOW | Vanilla JS `<dialog>` element or custom overlay div. Already have click-toggle pattern in report.html for metric expansion |
| Close via X button, Escape key, click-outside | Three closure methods are universal modal conventions. Missing any one feels broken | LOW | Escape via `keydown` listener, click-outside via backdrop click, X via button. ~20 lines JS |
| Scrollable content area | Trace data can be very long (especially C7 prompts/responses); non-scrollable modal is unusable | LOW | `overflow-y: auto; max-height: 80vh` on modal content div |
| Code-formatted output blocks | Command outputs and prompts are code. Displaying as plain text is unreadable | LOW | `<pre><code>` blocks with monospace font. CSS already uses system font stack; add code variant |
| Chronological step display | Users expect trace steps in execution order: "Step 1 -> Step 2 -> Step 3" | LOW | Ordered list or numbered sections. Data already flows through pipeline in order |
| Per-metric trigger button | Each metric row needs its own "View Trace" button; a single global button defeats the purpose | LOW | Add column to existing metric table or icon button in existing row. Template change |
| Collapsible long outputs | Full `go test` output or C7 responses can be hundreds of lines. Showing all at once buries the important parts | MEDIUM | Truncate with "Show full output" toggle. Reuse existing chevron expand pattern from metric rows |

#### Improve Prompt Modal (#57)

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| Modal overlay (same pattern as trace) | Same UI pattern reuse; users expect consistency between the two modals | LOW | Share modal infrastructure with trace modal. One modal component, two content sources |
| Copy to clipboard button | This is the entire point -- prompts exist to be copied. Without copy, users must manually select text | LOW | `navigator.clipboard.writeText()` -- supported in all modern browsers on HTTPS/localhost. Already used in badge section of report.html |
| Pre-formatted prompt text | Prompts must be rendered in a code block so users can verify what they're copying | LOW | `<pre>` block with monospace styling |
| Metric-specific data in prompt | Generic "improve your code" is worthless. Prompt must include actual score, raw value, specific file names | MEDIUM | Data already exists in `HTMLSubScore` struct (RawValue, FormattedValue, Score). Need to wire metric-specific templates to HTML |
| Build/test commands in prompt | Research shows operational context first improves agent performance by 28.64% (Impact of AGENTS.md, 2026) | MEDIUM | Detect from `.arsrc.yml`, `Makefile`, `package.json`, or `go.mod`. Fallback to generic commands per language |
| Verification step in prompt | Research shows agents claim completion prematurely. Verification command closes the loop | LOW | Always include `ars scan . --output-json` as final verification. Template constant |

### Differentiators (Competitive Advantage)

Features that set ARS apart from SonarQube, CodeClimate, and other code analysis tools. Not required, but valuable.

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| Research-backed prompt templates | No other code analysis tool generates AI agent prompts grounded in academic research with citations. SonarQube gives "fix this issue" -- ARS gives structured, research-informed prompts with explicit targets and verification | MEDIUM | Issue #57 provides the template structure and 10 research citations. Templates already partially exist in `descriptions.go` "How to Improve" sections |
| C7 full prompt/response replay | Showing actual Claude CLI prompts and responses is unique to agent evaluation tools. Users see exactly what the agent was asked and how it responded | LOW | C7DebugSample already captures prompt, response, score, score_trace. Data exists; just needs HTML rendering |
| Score trace visualization | Showing the scoring breakdown (base score + indicator adjustments = final) gives full transparency into how scores are derived | MEDIUM | C7ScoreTrace type already captures base_score, indicators[], final_score. Need visual step-by-step rendering |
| Worst-offender file lists in prompts | Including specific file names with worst scores makes prompts immediately actionable. Agent can start working without searching | MEDIUM | C1Metrics.Functions has per-function data, C6Metrics.TestFunctions has per-test data. Need to extract top-N worst offenders per metric and pass to prompt template |
| Prompt adaptation by score tier | Different prompts for score 2 vs score 7. A score-2 metric needs foundational work; score-7 needs polish. One-size-fits-all prompts waste agent tokens | MEDIUM | Threshold-based template selection. Three tiers: critical (<4), needs-work (4-7), fine-tuning (>7) |
| Inline syntax highlighting | Command outputs and code snippets with syntax highlighting (even basic keyword coloring) look more professional and are easier to scan | MEDIUM | Could embed a lightweight highlighter or use CSS-only approach for common keywords. Self-contained HTML constraint limits options |
| Keyboard navigation between modals | Tab/Shift+Tab through metrics, Enter to open, arrow keys to navigate within modal | HIGH | Full keyboard nav requires focus management, ARIA roles, screen reader support. High effort but important for accessibility |

### Anti-Features (Commonly Requested, Often Problematic)

Features that seem good but create problems. Deliberately NOT building these.

| Feature | Why Requested | Why Problematic | Alternative |
|---------|---------------|-----------------|-------------|
| Live "Run Fix" from report | Users want one-click improvement | Self-contained HTML cannot execute commands. Would require a running server, authentication, workspace management. Massive scope creep | Copy-paste prompt to your agent. The report is a document, not an IDE |
| Editable prompts within modal | Users want to customize prompts before copying | Adds text editor complexity, state management, risk of losing edits on modal close. The modal becomes an application | Copy to clipboard, edit in your editor/agent chat. Keep modal read-only |
| Real-time trace streaming | Show analysis progress as it happens | Requires WebSocket or SSE connection, fundamentally incompatible with static HTML report. Would need server architecture | Post-hoc trace display. Generate report after scan completes, include all trace data |
| External dependency for syntax highlighting | Fetch highlight.js or Prism from CDN | Breaks self-contained HTML constraint. Report must work offline, on air-gapped machines, via file:// protocol | CSS-only basic highlighting or embedded micro-library (<3KB inline) |
| Trace data for EVERY intermediate computation | Show raw AST nodes, every file walked, every regex matched | Trace data would be massive (megabytes for large projects). HTML report becomes unwieldy. Most users only care about the scoring-relevant steps | Show: (1) what was measured, (2) what the raw value was, (3) how it was scored. Skip internal implementation details |
| Diff view between baseline and current | Show what changed between two scans in the trace | Requires storing previous trace data, diffing algorithm, complex UI. Trend chart already exists for score comparison | Existing trend chart for score comparison. Users can open two reports side by side for detailed comparison |
| AI-generated fix patches | Go beyond prompts to generate actual code patches | Requires LLM API call at report generation time, adds cost and latency. Quality would be unpredictable. Not what users expect from a static report | Provide structured prompt that user feeds to their preferred agent, which has full project context |
| Per-file drill-down from trace | Click a filename in trace to see the full file content | Would embed entire source files in the HTML report, making it enormous. Privacy/security concerns with shipping source in reports | Show file path and relevant metric (e.g., "complexity: 23"). User can open the file in their editor |

## Feature Dependencies

```
[Shared Modal Infrastructure]
    |
    +--- [Call Trace Modal (#56)]
    |        |
    |        +--- requires ---> [Trace Data Capture in Analyzers]
    |        |                       |
    |        |                       +--- C7 traces already captured (C7DebugSample)
    |        |                       +--- C1-C6 traces need new capture mechanism
    |        |
    |        +--- requires ---> [Trace Data in Pipeline Flow]
    |                               (AnalysisResult -> ScoredResult -> HTMLReportData)
    |
    +--- [Improve Prompt Modal (#57)]
             |
             +--- requires ---> [Prompt Template System]
             |                       |
             |                       +--- Per-metric templates (38 metrics)
             |                       +--- Score-tier adaptation
             |                       +--- Build/test command detection
             |
             +--- requires ---> [Worst-Offender Extraction]
             |                       (pull specific files/functions from metric data)
             |
             +--- enhances ---> [Existing Recommendation System]
                                     (recommend package already has action templates)
```

### Dependency Notes

- **Shared Modal Infrastructure must come first:** Both features (#56 and #57) need the same overlay, close-handler, and scrollable-content pattern. Build once, reuse twice.
- **C7 trace data already exists:** `C7DebugSample` captures prompt, response, score, score_trace. This is the easiest trace to display -- just render what's already captured.
- **C1-C6 trace data does NOT exist in full:** Current analyzers return only final metric values. They do not capture commands run or raw tool output. However, they DO capture structured detail data (FunctionMetric, DuplicateBlock, DeadExport, TestFunctionMetric, FileChurn, CoupledPair) that can serve as trace-like content.
- **Prompt templates reuse existing content:** `descriptions.go` already has "How to Improve" sections per metric. `recommend.go` already has `actionTemplates` and `agentImpact` maps. Prompt generation can compose from these existing sources plus metric-specific data.
- **Build/test command detection enhances both features:** Trace modals can show "here's the command we ran" and improve prompts can include "here's how to verify." Shared infrastructure.

## Data Availability for Traces

This is the critical analysis. What data already exists in the pipeline that can populate trace modals, and what needs to be added?

### Already Captured (ready to render)

| Data | Type | Location | Trace Use |
|------|------|----------|-----------|
| C7 prompts | `string` | `C7DebugSample.Prompt` | Show exact prompt sent to Claude CLI |
| C7 responses | `string` | `C7DebugSample.Response` | Show exact response received |
| C7 score breakdown | `C7ScoreTrace` | `C7DebugSample.ScoreTrace` | Show base_score + indicator adjustments |
| C7 sample file paths | `string` | `C7DebugSample.FilePath` | Show which files were evaluated |
| Per-function complexity | `FunctionMetric` | `C1Metrics.Functions` | Show top-N complex functions with file:line |
| Duplicate code blocks | `DuplicateBlock` | `C1Metrics.DuplicatedBlocks` | Show file pairs with line ranges |
| Coupling per package | `map[string]int` | `C1Metrics.AfferentCoupling/EfferentCoupling` | Show which packages have high coupling |
| Circular dependencies | `[][]string` | `C3Metrics.CircularDeps` | Show dependency cycles as chains |
| Dead exports | `DeadExport` | `C3Metrics.DeadExports` | Show unused symbols with file:line |
| Per-test function data | `TestFunctionMetric` | `C6Metrics.TestFunctions` | Show tests with assertion counts |
| Top hotspot files | `FileChurn` | `C5Metrics.TopHotspots` | Show highest-churn files |
| Temporally coupled pairs | `CoupledPair` | `C5Metrics.CoupledPairs` | Show files that change together |
| Scoring breakpoints | `[]Breakpoint` | `scoring.ScoringConfig` | Show how raw value maps to score |

### NOT Captured (would need pipeline changes)

| Data | What It Would Show | Effort | Recommendation |
|------|--------------------|--------|----------------|
| Shell commands run by analyzers | `go test -cover ./...`, `git log --numstat` | MEDIUM | Defer. C1-C6 analyzers use Go APIs, not shell commands. The "commands" are internal function calls, not user-visible commands |
| Raw tool output (coverage reports, etc.) | Full stdout of `go test -cover` | HIGH | Defer. Outputs are parsed immediately; raw text is discarded. Capturing would require buffering in every analyzer |
| Discovery file list | Which files were included/excluded and why | MEDIUM | Useful but not per-metric. Could add a global "Discovery" trace section |
| Scoring interpolation steps | "Your raw value 12.3 falls between breakpoints (10, score 7) and (15, score 5), interpolated to 5.7" | LOW | High value, low cost. Scoring config is already available; just render the interpolation math |

### Recommendation: "Scoring Explanation" Not "Command Replay"

For C1-C6, the MVP trace should be a **scoring explanation** showing:
1. What the metric measures (brief description -- already in `descriptions.go`)
2. The raw value and how it was calculated (structured data already captured)
3. The scoring breakpoints and where this value falls (scoring config already available)
4. The worst offenders driving the score (per-function/per-file data already captured)

This is distinct from a "command replay" that shows `go test -cover ./... > output`. The scoring explanation is MORE useful to the user (it answers "why this score?") and requires ZERO pipeline changes.

For C7, the trace IS a command replay because C7DebugSample already captures the full prompt/response cycle.

## MVP Definition

### Launch With (v1)

Minimum viable product -- what's needed to validate the concept.

- [ ] **Shared modal component** -- Single reusable overlay with backdrop, X/Escape/click-outside close, scrollable content. Vanilla JS, no dependencies. Use native `<dialog>` element for accessibility.
- [ ] **C7 trace modal** -- Render existing `C7DebugSample` data (prompt, response, score trace) in modal. This is the easiest trace because data already exists.
- [ ] **C1-C6 scoring explanation trace** -- For static analysis, show: metric name, raw value, scoring breakpoints used, where the value falls on the breakpoint curve, and worst-offender examples from existing structured data.
- [ ] **Copy to clipboard for prompts** -- `navigator.clipboard.writeText()` with visual feedback (button text changes to "Copied!" for 2 seconds).
- [ ] **Basic prompt templates for all 7 categories** -- One template per category (not per-metric). Include: context, current score, target, task description, verification command.
- [ ] **Prompt data population** -- Fill templates with actual metric values from `HTMLSubScore` data.
- [ ] **Trace and Improve buttons in metric table** -- Icon buttons in each metric row. Small, unobtrusive, consistent with existing design.

### Add After Validation (v1.x)

Features to add once core is working.

- [ ] **Per-metric prompt templates (38 total)** -- Upgrade from per-category to per-metric prompts with specific actionable instructions. Trigger: users report category-level prompts are too generic
- [ ] **Worst-offender file lists in prompts** -- Extract top-3 worst files per metric from `C1Metrics.Functions`, `C6Metrics.TestFunctions`, etc. Trigger: users want more specific prompts
- [ ] **Score-tier adapted prompts** -- Different prompt text for critical vs needs-work vs fine-tuning. Trigger: users report prompts for high-scoring metrics are too aggressive
- [ ] **Scoring interpolation visualization** -- Show the breakpoint curve with a marker at the current value. Simple SVG inline chart. Trigger: users want visual explanation of scoring
- [ ] **Basic syntax highlighting in trace** -- CSS-only keyword highlighting for common patterns (commands, file paths, numbers). Trigger: readability feedback
- [ ] **Build/test command auto-detection** -- Parse `.arsrc.yml`, `Makefile`, `package.json`, `go.mod` to populate operational context in prompts. Trigger: users manually editing prompts to add build commands

### Future Consideration (v2+)

Features to defer until product-market fit is established.

- [ ] **Full keyboard navigation** -- Complete ARIA roles, focus trapping, screen reader announcements. Trigger: accessibility audit or enterprise adoption
- [ ] **Trace data in JSON output** -- Include trace information in JSON format for programmatic access. Trigger: CI/CD integration requests
- [ ] **Prompt effectiveness tracking** -- Compare scores before/after users apply prompted improvements. Trigger: need to validate prompt quality
- [ ] **Custom prompt template overrides** -- Let users provide their own prompt templates via `.arsrc.yml`. Trigger: enterprise customization requests
- [ ] **Rich C1-C6 command trace capture** -- Full analyzer instrumentation to capture commands/outputs. Trigger: scoring explanation proves insufficient

## Feature Prioritization Matrix

| Feature | User Value | Implementation Cost | Priority |
|---------|------------|---------------------|----------|
| Shared modal component | HIGH | LOW | P1 |
| C7 trace rendering (data exists) | HIGH | LOW | P1 |
| Copy to clipboard | HIGH | LOW | P1 |
| Basic prompt templates (7 categories) | HIGH | MEDIUM | P1 |
| Prompt data population | HIGH | MEDIUM | P1 |
| Trace/Improve buttons in UI | HIGH | LOW | P1 |
| C1-C6 scoring explanation trace | MEDIUM | MEDIUM | P1 |
| Per-metric prompt templates (38) | HIGH | HIGH | P2 |
| Worst-offender file lists | HIGH | MEDIUM | P2 |
| Score-tier adapted prompts | MEDIUM | MEDIUM | P2 |
| Scoring interpolation visualization | MEDIUM | MEDIUM | P2 |
| Basic syntax highlighting | LOW | MEDIUM | P2 |
| Build/test auto-detection | MEDIUM | MEDIUM | P2 |
| Keyboard navigation / a11y | MEDIUM | HIGH | P3 |
| JSON trace output | LOW | MEDIUM | P3 |
| Custom prompt templates | LOW | MEDIUM | P3 |

**Priority key:**
- P1: Must have for launch -- features that make the buttons functional and useful
- P2: Should have -- features that make the experience polished and differentiated
- P3: Nice to have -- features for specific use cases or compliance requirements

## Competitor Feature Analysis

| Feature | SonarQube | CodeClimate | Lighthouse | ARS (Our Approach) |
|---------|-----------|-------------|------------|-------------------|
| Score explanation | Drill-down to rule violations | Letter grade with issue list | Expandable audit details with "Learn more" | Per-metric trace showing how score was derived, including scoring breakpoints and worst offenders |
| Improvement guidance | Rule-specific "How to fix" snippets | Generic suggestions | Best practice links | Research-backed, copy-paste-ready prompts populated with actual project data and verification steps |
| Trace/debug visibility | Issue location (file:line) | File-level issues | Audit detail steps | Full execution trace for C7 (prompt/response), scoring explanation for C1-C6 |
| AI-agent integration | None | None | None (web-focused) | Prompts designed specifically for AI coding agents with research-backed structure |
| Report format | Web dashboard (requires server) | Web dashboard (requires server) | Self-contained HTML | Self-contained HTML (no server needed) |
| Offline capability | No (server required) | No (SaaS) | Yes (HTML file) | Yes (HTML file, all data embedded) |
| Copy-paste workflow | Manual | Manual | Manual | One-click copy of structured prompts |

**Key differentiator:** No existing code analysis tool generates AI-agent-optimized improvement prompts. SonarQube and CodeClimate predate the AI coding agent era. Lighthouse is web-performance-focused. ARS is uniquely positioned to bridge "what's wrong" to "here's exactly what to tell your AI agent to fix it."

## Critical Implementation Constraints

### Self-Contained HTML Requirement

The HTML report must remain a single file with no external dependencies. This means:
- All CSS must be inlined (already done via `embed.FS`)
- All JS must be inlined (already done in `<script>` block)
- No CDN dependencies for syntax highlighting, icons, or fonts
- Modal HTML templates must be embedded in the Go template
- Trace and prompt data must be serialized into the HTML as either `data-*` attributes on elements or `<script type="application/json">` blocks that JS reads at runtime

### Data Serialization Strategy

Two approaches for getting trace/prompt data into the HTML:

**Option A: Pre-rendered HTML (recommended for MVP)**
- Go template renders all modal content as hidden `<div>` elements
- JS just shows/hides modals
- Pro: No JSON parsing in JS, simpler template logic
- Con: HTML file size grows with all modal content pre-rendered

**Option B: JSON data + JS rendering**
- Go template serializes trace/prompt data as `<script type="application/json">` blocks
- JS reads JSON and builds modal content dynamically
- Pro: Smaller initial HTML, lazy rendering
- Con: More complex JS, potential for rendering bugs

**Recommendation:** Option A for MVP. The report is already a substantial HTML file with embedded CSS, SVG charts, and per-metric descriptions. Adding pre-rendered modal content adds maybe 20-50KB for a typical project. This keeps JS minimal and avoids the need for a JS templating system.

### Prompt Template Architecture

Prompts need to be generated at scan time (when all data is available) and embedded in the HTML. The template system should:

1. **Live in Go code** -- prompt templates as Go string constants in a new `internal/output/prompts.go` file
2. **Use existing data** -- compose from `descriptions.go` (metric descriptions), `recommend.go` (action templates), and `scoring.ScoringConfig` (breakpoints)
3. **Follow research structure** -- Context, Build/Test Commands, Task, Current State, Target State, Constraints, Verification (per issue #57)
4. **Produce plain text** -- prompts are Markdown text that users paste into AI agents, not HTML

## User Workflow Analysis

### When would users click "View Trace"?

1. **Unexpected low score** -- "Why did my complexity get a 3.2?" User wants to see which functions dragged the score down and how scoring breakpoints work.
2. **Validating C7 results** -- "Did the agent actually understand my code?" User wants to see the exact prompt and response.
3. **Debugging score changes** -- "Score dropped from 7.1 to 5.3 after my refactor." User wants to see what was measured differently.
4. **Building trust** -- First-time users want to understand the methodology before acting on recommendations.

### When would users click "Improve"?

1. **After seeing a low score** -- "Score is 4.2. How do I fix this?" User wants immediate, actionable guidance.
2. **Delegating to AI agent** -- "I'll have Claude Code handle this." User wants a ready-made prompt to paste.
3. **Prioritizing work** -- User scans all improve prompts to decide which metric to tackle first.
4. **Team onboarding** -- Senior dev generates prompts for junior devs or AI agents to execute.

### Expected interaction flow:

```
1. User opens HTML report
2. Scans radar chart and composite score
3. Finds low-scoring category
4. Expands metric details (existing feature)
5. Clicks "View Trace" to understand WHY the score is low
   --> Modal shows: raw data, scoring breakdown, worst offenders
6. Clicks "Improve" to get actionable fix
   --> Modal shows: structured prompt with project-specific data
7. Clicks "Copy to clipboard"
8. Pastes prompt into AI coding agent (Claude Code, Copilot, Cursor, etc.)
9. Agent executes improvements
10. User re-runs `ars scan` to verify improvement
```

## Prompt Template Structure (from Issue #57 Research)

Based on the research cited in issue #57, each improvement prompt should follow this structure:

```markdown
## Context
- Project: {project_name}
- Language: {detected_language}
- Current {metric_display_name} score: {score}/10 (target: >=8.0)
- Current raw value: {raw_value} {unit}

## Build & Test Commands
{build_command}
{test_command}

## Task
Improve {metric_display_name} by {specific_actionable_instruction}.

### Current State
{metric_brief_description}
{worst_offender_list_if_available}

### Target State
{concrete_measurable_target_from_breakpoints}

### Constraints
- Do not break existing tests
- Maintain backward compatibility
- Keep changes focused on {metric_name} only
- Make incremental changes; do not refactor everything at once

## Verification
After making changes, verify improvement:
{verification_command}

Expected: {metric_name} score should increase from {current_score} toward {target_score}.
```

Research backing for this structure:
- Explicit metric targets improve code quality by up to 35% (Enhancing LLM Code Generation with Complexity Metrics, 2025)
- Including static analysis output gives agents concrete context (Augmenting LLMs with Static Code Analysis, 2025)
- Operational context first reduces agent runtime by 28.64% (Impact of AGENTS.md Files, 2026)
- Mandatory verification steps prevent premature completion claims (Self-Refine, Madaan et al., 2023)
- Specifying refactoring level is necessary for architectural improvements (Agentic Refactoring, 2025)

## Sources

### Codebase Analysis (HIGH confidence)
- `/Users/ingo/agent-readyness/internal/output/html.go` -- Current HTML generation, data structures (HTMLSubScore, HTMLCategory)
- `/Users/ingo/agent-readyness/internal/output/descriptions.go` -- Per-metric descriptions with "How to Improve" sections (38 metrics)
- `/Users/ingo/agent-readyness/internal/output/citations.go` -- Research citations already embedded per category
- `/Users/ingo/agent-readyness/internal/output/templates/report.html` -- Current template with expand/collapse JS and badge copy pattern
- `/Users/ingo/agent-readyness/pkg/types/types.go` -- C7DebugSample, C7ScoreTrace, FunctionMetric, DuplicateBlock, DeadExport, TestFunctionMetric, FileChurn, CoupledPair
- `/Users/ingo/agent-readyness/internal/recommend/recommend.go` -- Existing actionTemplates and agentImpact maps
- `/Users/ingo/agent-readyness/internal/scoring/scorer.go` -- Scoring interpolation logic and config

### GitHub Issues (HIGH confidence)
- [Issue #56](https://github.com/ingo-eichhorst/agent-readyness/issues/56) -- Call trace requirements, UI design, data flow
- [Issue #57](https://github.com/ingo-eichhorst/agent-readyness/issues/57) -- Improvement prompt requirements, research foundation, template structure

### UI Pattern Research (MEDIUM confidence)
- [Playwright Trace Viewer](https://playwright.dev/docs/trace-viewer-intro) -- Step-by-step trace visualization; gold standard for trace UX
- [Lighthouse HTML Report](https://developer.chrome.com/docs/lighthouse/overview/) -- Self-contained HTML with expandable audit details; closest analog to ARS report
- [MDN `<dialog>` element](https://developer.mozilla.org/en-US/docs/Web/HTML/Reference/Elements/dialog) -- Native modal dialog with built-in accessibility (showModal, close, ::backdrop)
- [MDN Clipboard API](https://developer.mozilla.org/en-US/docs/Web/API/Clipboard/writeText) -- `navigator.clipboard.writeText()` supported in all modern browsers; requires HTTPS or localhost
- [Vanilla JS Modal Patterns](https://jasonwatmore.com/post/2023/01/04/vanilla-js-css-modal-popup-dialog-tutorial-with-example) -- Implementation without framework dependencies

### Research Foundation (HIGH confidence -- from Issue #57)
- [Enhancing LLM Code Generation with Complexity Metrics (2025)](https://arxiv.org/html/2505.23953) -- Explicit metric targets improve quality by 35%
- [Augmenting LLMs with Static Code Analysis (2025)](https://arxiv.org/html/2506.10330v1) -- Static analysis output as prompt context
- [Agent READMEs: An Empirical Study (2025)](https://arxiv.org/html/2511.12884v1) -- Structured shallow hierarchy for context
- [Agentic Refactoring: An Empirical Study (2025)](https://arxiv.org/html/2511.04824) -- Must specify refactoring level explicitly
- [Impact of AGENTS.md Files (ICSE JAWs 2026)](https://arxiv.org/abs/2601.20404) -- Context files reduce runtime by 28.64%
- [Self-Refine (Madaan et al., 2023)](https://arxiv.org/abs/2303.17651) -- Verification steps for iterative refinement
- [Decoding Configuration of AI Coding Agents (2025)](https://arxiv.org/html/2511.09268v1) -- Operational context before task description

---
*Feature research for: Interactive HTML Report Enhancements (Call Traces + AI Improvement Prompts)*
*Researched: 2026-02-06*
