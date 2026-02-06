# Pitfalls Research: Interactive HTML Enhancements (Modal UI, AI Prompts, Trace Data)

**Domain:** Adding modal dialogs, call trace data, and AI improvement prompts to self-contained HTML reports
**Researched:** 2026-02-06
**Confidence:** HIGH (grounded in codebase analysis of current HTML output, CSS, JSON schema, and recommendations system)

This document catalogs pitfalls specific to adding interactive modal UI, C7 call trace visibility, and 38 AI improvement prompt templates to the existing ARS HTML report. The HTML report is currently ~165KB self-contained with no external dependencies. All CSS is inlined and all JS is inline at the bottom of the document.

> **Complements:** Previous PITFALLS.md (debug infrastructure). That document covered C7 debug output routing, CLI flag design, and concurrent metric state. This document covers the HTML report enhancement layer that consumes that debug data.

---

## Critical Pitfalls

These mistakes cause broken reports, accessibility failures, or require significant rework.

### 1. HTML File Size Explosion from Embedded Trace Data

**What goes wrong:**
C7 evaluation generates substantial trace data per metric: full prompts (500-2000 chars each), full Claude responses (200-2000 chars each), and scoring breakdowns. With 5 metrics x 3 samples each = 15 prompt/response pairs. Each `C7DebugSample` struct (defined in `pkg/types/types.go:322-331`) includes:
- `Prompt` (string, typically 500-2000 chars)
- `Response` (string, typically 200-2000 chars -- some can be much longer)
- `ScoreTrace` with `Indicators` array (10-20 entries per sample)
- `FilePath`, `Description`, metadata

Naive embedding of all trace data as inline JSON in the HTML template would add 50-150KB per C7 evaluation. Combined with the existing ~87KB `descriptions.go` content already inlined as HTML fragments, and the ~10KB CSS, the report could balloon from 165KB to 300-500KB. For projects scanned repeatedly with `--baseline`, the trend comparison embeds two SVG charts already.

Worse, the Cucumber HTML formatter project hit 314MB reports (from 7MB) when embedding redundant per-scenario data -- the exact same class of bug: including full execution trace data that grows linearly with test/metric count. Their root cause was duplicated `stepDefinition` messages embedded per scenario rather than once.

**Why it happens:**
The natural implementation is to add C7 trace data to `HTMLReportData` and render it in hidden `<div>` elements that modals reveal. Since the HTML must be self-contained (no external dependencies, no fetching), ALL data must be present in the initial HTML. Go's `html/template` makes it easy to embed data inline -- too easy. There is no size gate or warning when the template output exceeds a threshold.

**How to avoid:**
- Set a hard file size budget: the HTML report must not exceed 500KB total (current baseline: 165KB). Add a test that generates a report with C7 data and asserts the output size.
- Truncate trace data at the embedding layer, not at the capture layer. The full data stays in JSON output (`--json --verbose`). The HTML report shows truncated versions:
  - Prompts: first 500 chars + "... (see full prompt in JSON output)"
  - Responses: first 1000 chars + "... (see full response in JSON output)"
  - Score traces: full (these are small -- ~20 indicator entries per sample)
- Use a single JSON blob in a `<script type="application/json" id="trace-data">` tag rather than duplicating data across multiple hidden `<div>` elements. JS reads from this blob to populate modals on demand. This avoids HTML entity encoding overhead (which adds ~20% to text size due to `&quot;`, `&lt;`, etc.).
- Deduplicate shared content: if 3 samples for one metric use the same prompt template, store the template once and reference it by ID.

**Warning signs:**
- HTML report file size exceeds 300KB when C7 is enabled
- Report takes more than 2 seconds to open in a browser (measured on a mid-range laptop)
- `go test` output shows the HTML template rendering allocating more than 1MB for a single report
- Diff between C7-enabled and C7-disabled reports exceeds 200KB

**Phase to address:** Phase 1 (data model) -- establish the truncation strategy and size budget before building modal UI. Test the size budget in CI.

---

### 2. Modal Implementation Without Keyboard Navigation and Focus Trapping

**What goes wrong:**
The modal opens visually but is not accessible: keyboard users cannot reach it, cannot navigate within it, cannot close it, and focus does not return to the trigger element. Specifically:

1. **No focus trap:** Users press Tab and focus escapes the modal to background page elements. Screen reader users perceive content outside the modal that they cannot interact with.
2. **No Escape key handler:** The modal can only be closed by clicking the X button. Keyboard-only users are trapped (WCAG 2.1.2 No Keyboard Trap violation, Level A).
3. **No focus restoration:** After closing the modal, focus jumps to the top of the page instead of returning to the button that opened the modal. Users lose their place in a long report (7 categories x 5+ metrics each = 35+ expandable rows already).
4. **Missing ARIA attributes:** `role="dialog"`, `aria-modal="true"`, and `aria-labelledby` are omitted, so assistive technology does not announce the modal as a dialog.

The current HTML report already has interactive elements (chevron-based expand/collapse for metrics and references, sections at lines 41-42 of `report.html`) but these are NOT keyboard-accessible today -- click handlers are on `<td>` and `<h4>` elements that are not focusable. Adding modals that follow the same pattern would compound the accessibility debt.

**Why it happens:**
The report currently uses minimal inline JS (~35 lines in `report.html:125-159`) with direct DOM manipulation: `addEventListener('click', ...)` on chevron cells and reference headers. This pattern does not include keyboard handling. Developers adding modals follow the existing pattern (click-only), not realizing the existing code already has accessibility gaps.

Additionally, self-contained HTML constraints mean no ARIA library or modal helper can be imported. All focus management must be hand-written in inline JS.

**How to avoid:**
- Use the native `<dialog>` element with `showModal()` method. This provides focus trapping, Escape key handling, and backdrop automatically -- all without external dependencies. Browser support is excellent (97%+ as of 2025). The `<dialog>` element invoked via `showModal()` implicitly sets `aria-modal="true"`.
- Add `aria-labelledby` pointing to a visible heading inside the dialog.
- Explicitly manage focus: on open, focus the first interactive element or a static heading with `tabindex="-1"`; on close, return focus to the trigger element using a stored reference.
- Fix the existing expand/collapse pattern while adding modals: make chevron cells keyboard-accessible with `tabindex="0"` and `keydown` handlers for Enter/Space. Do not add a new accessibility pattern that is inconsistent with existing controls.
- Add a keyboard interaction test (or at minimum a manual test checklist): Tab cycles within modal, Shift+Tab cycles backward, Escape closes, focus returns to trigger.

**Warning signs:**
- Modal opens but Tab moves focus to elements behind it
- Pressing Escape does nothing
- Screen reader announces "dialog" but then reads background page content
- After closing a modal, the user's scroll position in the category section is lost
- No `<dialog>` element in the HTML output (custom `<div>` modal instead)

**Phase to address:** Phase 2 (modal UI) -- modal implementation must include keyboard handling from the start, not as a follow-up. The `<dialog>` element decision should be made in Phase 1 (data model) to inform the HTML template structure.

---

### 3. JSON Schema Breaking Changes for Existing Consumers

**What goes wrong:**
The JSON output schema (`JSONReport` in `json.go:12-20`) is consumed by downstream tools: CI pipelines that parse scores, `--baseline` comparison mode that loads previous JSON files, and potentially external integrations. Adding new fields for trace data, improvement prompts, or modal-related metadata in ways that break existing consumers:

1. **Renaming fields:** Changing `"recommendations"` to `"improvement_prompts"` breaks all existing `jq` commands and baseline files.
2. **Restructuring nested objects:** Moving metric data from `categories[].metrics[]` to a new `categories[].metrics[].trace` nesting level breaks parsers that expect flat metric objects.
3. **Changing field types:** If `"action"` (currently a string in `JSONRecommendation`) becomes an object `{"prompt": "...", "context": "..."}`, every consumer breaks.
4. **Required new fields without defaults:** Adding a required field to the schema that is absent in older baseline files causes `--baseline` comparison to fail.

The current schema has `"version": "1"` (set in `BuildJSONReport` at `json.go:58`) but there is no version-checking logic in the baseline loading code. Bumping the version to "2" without migration logic means old baseline files silently produce incorrect comparisons.

**Why it happens:**
When adding improvement prompts, the natural approach is to extend `JSONRecommendation` with prompt-specific fields. But the existing struct is tightly mapped to the current recommendation model (`recommend.Recommendation` in `recommend.go:12-24`). Adding new fields to represent AI prompts (multi-paragraph, with context and model-specific formatting) requires either expanding the flat struct or nesting -- both are breaking changes if done carelessly.

**How to avoid:**
- All new fields must use `omitempty` JSON tags. Existing consumers that do not request new data should see identical output. The current code already uses this pattern for `BadgeURL` and `BadgeMarkdown` (`json.go:18-19`).
- Add new data in NEW fields alongside existing ones, never rename or restructure existing fields. For example, add `"improvement_prompt": "..."` alongside the existing `"action": "..."` in `JSONRecommendation`.
- Add a new top-level section for trace data: `"c7_traces": [...]` rather than nesting it inside `"categories"`. This is purely additive and `omitempty` makes it invisible when C7 is not enabled.
- Implement baseline version checking: when loading a baseline file, check its `"version"` field and handle version "1" files gracefully (skip fields that do not exist, use defaults).
- Write a schema compatibility test: serialize a report with the new code, deserialize it with the old struct definitions, and verify no data loss or panic.

**Warning signs:**
- Existing `--baseline` comparison produces errors or incorrect deltas after the change
- `jq '.recommendations[0].action'` returns null or wrong type on new output
- CI pipelines that parse JSON output start failing
- The `JSONReport` struct changes any existing field name or type

**Phase to address:** Phase 1 (data model) -- define the JSON schema extensions before building any features. Write compatibility tests that load a v1 baseline file and verify it still works.

---

### 4. Improvement Prompt Templates That Are Generic and Non-Actionable

**What goes wrong:**
The 38 improvement prompt templates (one per metric, likely derived from the existing `actionTemplates` in `recommend.go:52-74` and `metricDescriptions` in `descriptions.go`) produce generic advice that users cannot act on:

- "Reduce complexity in your codebase" -- users already know this, the score told them
- "Add more tests" -- no specificity about which code to test or how
- "Improve documentation" -- does not reference actual undocumented APIs

The existing `actionTemplates` already have this partial problem. For example, `"complexity_avg"` produces `"Refactor functions with cyclomatic complexity > %.0f into smaller units"` -- this tells the user WHAT to do but not HOW to instruct an AI agent to do it. An "improvement prompt" should be a ready-to-paste prompt for Claude/ChatGPT that references the specific project context.

**Why it happens:**
At template authoring time, the template author does not have access to project-specific data. Templates are static strings in Go code. The temptation is to write a good-sounding generic prompt like "Analyze this codebase and reduce complexity" rather than a context-aware prompt that interpolates the actual metric values, file names, and specific thresholds.

Additionally, the 38 templates must be maintained as ARS evolves. When scoring breakpoints change (in `scoring/config.go`), when new metrics are added, or when best practices evolve, the templates become stale. But unlike code, stale templates do not produce compile errors -- they silently give outdated advice.

**How to avoid:**
- Every prompt template MUST interpolate at least: (1) the actual metric value, (2) the target value, (3) specific file names or modules that are worst offenders. The existing `buildAction` function in `recommend.go:336-370` already interpolates values -- extend this pattern, do not replace it.
- Structure prompts in three parts: CONTEXT (what the tool measured), GOAL (what score improvement is possible), INSTRUCTION (specific refactoring request with file references). Example: "The function `parseInput` in `parser.go` has cyclomatic complexity 23 (target: <10). Refactor it using guard clauses and extract helper functions. Preserve all existing test assertions in `parser_test.go`."
- For each template, include a "copy to clipboard" button in the modal AND a "view in JSON" reference so users can access the full prompt programmatically.
- Add a version/date marker to each template so users know when it was last updated. Store the template version in the JSON output so stale templates are detectable.
- Create a test that validates every metric name in `scoring.DefaultConfig()` has a corresponding prompt template. This prevents silent gaps when new metrics are added.

**Warning signs:**
- Prompt templates do not contain `%s` or `%f` format verbs (no interpolation = generic advice)
- Templates reference concepts not measured by ARS (e.g., "use dependency injection" for a metric that measures file size)
- A metric is added to `config.go` but no corresponding prompt template exists
- User feedback indicates prompts are too generic to use (this is a post-launch signal)

**Phase to address:** Phase 3 (prompt templates) -- but the template structure and interpolation API should be defined in Phase 1 (data model) to ensure the data needed for interpolation is available.

---

### 5. Mobile/Responsive Modal Breakage

**What goes wrong:**
The current HTML report has a single responsive breakpoint (`@media (max-width: 640px)` in `styles.css:549-566`). Modals added without mobile-first design cause:

1. **Modal exceeds viewport:** Full-width modals with fixed dimensions overflow on phones. Users cannot see the close button or scroll within the modal content.
2. **iOS Safari scroll lock failure:** Setting `overflow: hidden` on `<body>` does not reliably prevent background scrolling on iOS Safari. Users scroll the background page while trying to scroll the modal content, especially with the rubber-band overscroll effect.
3. **Keyboard overlap:** On mobile, tapping a text input (if the modal has a search/filter for prompts) opens the virtual keyboard, which pushes the modal content up and may hide the close button.
4. **Touch target size:** Close buttons and copy-to-clipboard buttons smaller than 44x44px are difficult to tap on mobile devices.

The current report works passably on mobile because it is a simple vertical scroll with no overlays. Adding modals introduces a second scrolling context (modal content within viewport) that is notoriously difficult on mobile browsers.

**Why it happens:**
Desktop-first development: the modal looks great on a 1440px wide screen with a mouse. Mobile testing is deferred or done only in Chrome DevTools responsive mode (which does not replicate iOS Safari scroll behavior). The `<dialog>` element helps with some of these issues (it centers itself, handles backdrop) but does not solve iOS scroll lock or touch target sizing.

**How to avoid:**
- Use the native `<dialog>` element, which handles centering and backdrop rendering across devices. Chrome 144+ (December 2025) added `overscroll-behavior: contain` support on dialog and backdrop elements, though Safari may not have caught up yet.
- Set modal width to `min(90vw, 700px)` and max-height to `80vh` with `overflow-y: auto` on the content area. This prevents viewport overflow on any screen size.
- For iOS Safari scroll lock: apply `position: fixed; width: 100%` to the `<body>` when the modal opens, saving and restoring the scroll position on close. This is the most reliable cross-browser approach as of 2026.
- Ensure all interactive elements in the modal are at least 44x44px tap targets.
- Test on actual iOS Safari (or BrowserStack equivalent), not just Chrome DevTools responsive mode. The scroll lock behavior differs significantly.
- Keep modal content minimal: prompts are text, not interactive forms. Avoid putting complex interactive widgets inside modals.

**Warning signs:**
- Modal close button is not visible without scrolling on a 375px-wide viewport (iPhone SE)
- Background page scrolls while modal is open on iOS Safari
- Modal content is cut off at the bottom with no scroll indicator
- Touch targets in the modal fail the 44px minimum check

**Phase to address:** Phase 2 (modal UI) -- responsive behavior must be tested from the start. Use `min()` and `vh` units in the modal CSS rather than fixed pixel dimensions.

---

### 6. Progressive Enhancement Failure: Report Breaks Without JavaScript

**What goes wrong:**
The project explicitly requires progressive enhancement ("work without JS" per the constraints). The current report achieves this partially -- the expand/collapse for metric details uses JS (`report.html:126-158`), and metrics with `ShouldExpand: true` render pre-expanded via inline `style="display: none;"` on the details row. Without JS, some metrics are expanded and some are hidden, but the core data (scores, categories, values) is always visible in the main table.

Adding modals breaks this model: if the modal trigger button is visible but JS is disabled, clicking the button does nothing. The user sees a button that appears interactive but is dead. Worse, the improvement prompt content is ONLY accessible via the modal -- without JS, the content is completely invisible.

**Why it happens:**
Modals are inherently JS-dependent UI patterns. The `<dialog>` element's `showModal()` requires JS. There is no pure CSS/HTML way to open a modal dialog. Developers assume "everyone has JS" and do not provide a fallback.

**How to avoid:**
- Use the `<details>/<summary>` HTML pattern as the no-JS fallback. The improvement prompt content should be visible in a `<details>` element that collapses by default. With JS enabled, the `<details>` is hidden and replaced with a modal trigger button. This is true progressive enhancement.
- Alternatively, use `<noscript>` blocks to show a "copy from JSON output" message when JS is unavailable.
- Test the report with JS disabled in the browser: all scores, metric values, and improvement content must be readable. Interactive features (modals, copy-to-clipboard) can degrade gracefully.
- The "Expand All / Collapse All" buttons (`report.html:41-42`) already have this problem -- they do nothing without JS. Do not compound this debt. Consider converting the existing expand/collapse to `<details>/<summary>` as part of this milestone.

**Warning signs:**
- Improvement prompt content is only accessible via `showModal()` with no HTML fallback
- Buttons exist in the rendered HTML that do nothing without JS
- The `<noscript>` tag is not used anywhere in the template
- Searching the HTML source for prompt text returns no results (it is only in a JS variable)

**Phase to address:** Phase 2 (modal UI) -- the HTML template must render prompt content in `<details>` elements first, then enhance with modal triggers via JS.

---

### 7. Copy-to-Clipboard Failing Silently Across Browsers

**What goes wrong:**
The "copy prompt to clipboard" feature uses `navigator.clipboard.writeText()` (already used for badge copying at `report.html:34`). This API has restrictions:

1. **HTTPS required in most browsers:** `navigator.clipboard.writeText()` is only available in secure contexts. HTML files opened via `file://` protocol (the primary use case for ARS reports) have inconsistent secure context treatment across browsers.
2. **User gesture required:** The API requires a recent user interaction (click). Opening the modal and then clicking "copy" works, but programmatically copying on modal open does not.
3. **Permissions may be denied:** Some browsers prompt the user, others silently fail. The current badge copy button (`report.html:34`) has no error handling -- it calls `navigator.clipboard.writeText(...)` inline with no fallback.
4. **Firefox `file://` restriction:** Firefox blocks clipboard access on `file://` URLs entirely.

For ARS reports, the primary use case is opening the HTML file directly from disk (`file:///path/to/report.html`). This is exactly the context where clipboard APIs are most restricted.

**Why it happens:**
The `navigator.clipboard` API works perfectly in development (localhost or served via HTTP). The `file://` restrictions only surface when users open the report as intended. The existing badge copy button has this same silent failure, but it is less critical (users can manually select and copy markdown text). For improvement prompts that may be multi-paragraph, manual selection is much harder.

**How to avoid:**
- Implement a fallback: if `navigator.clipboard.writeText()` fails (wrap in try/catch), fall back to creating a temporary `<textarea>`, selecting its content, and using `document.execCommand('copy')`. While `execCommand` is deprecated, it works on `file://` URLs in most browsers.
- Show visual feedback for both success and failure. On success: change button text to "Copied!" for 2 seconds. On failure: change button text to "Select text below to copy manually" and select the prompt text in the visible element.
- Include the prompt text as a visible, selectable `<pre>` or `<code>` block in the modal (not hidden). This ensures users can always manually select and copy even if all programmatic clipboard approaches fail.
- Test clipboard functionality by opening the report via `file://` protocol, not via a local dev server.

**Warning signs:**
- Copy button uses `navigator.clipboard.writeText()` without try/catch
- No visible prompt text in the modal (only a "Copy" button that programmatically accesses hidden data)
- Copy button works on `http://localhost` but fails when opening the `.html` file directly
- No visual feedback on copy success or failure

**Phase to address:** Phase 2 (modal UI) -- clipboard handling must be implemented with fallback from the start. Do not defer error handling.

---

## Technical Debt Patterns

Shortcuts that seem reasonable but create long-term problems.

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|----------------|-----------------|
| Embedding full prompt/response text as HTML attributes (`data-prompt="..."`) | Simple to implement, no JS data management | HTML entity encoding bloats size 20%+; special characters in prompts break attribute parsing; XSS risk from user-controlled paths in prompt text | Never -- use `<script type="application/json">` data island instead |
| Duplicating recommendation data in both the visible table and hidden modal content | No JS needed to populate modals | Data appears twice in the HTML, doubling size for that section; inconsistency if one is updated but not the other | Only in MVP if modals are deferred to a later phase |
| Hardcoding 38 prompt templates as string literals in Go code | Fast to implement, easy to read | Cannot be updated without recompiling; no A/B testing; no version tracking; new metrics silently lack templates | Acceptable for MVP; migrate to config file or embedded YAML by v1.0 |
| Using `<div>` with custom JS for modals instead of `<dialog>` | Works identically in older browsers (though `<dialog>` is 97%+ supported) | Must reimplement focus trapping, Escape handling, backdrop, aria-modal -- hundreds of lines of JS that `<dialog>` provides free | Never -- `<dialog>` support is universal enough for a developer tool |
| Inlining all modal CSS into the existing stylesheet without scoping | No new CSS architecture needed | Modal styles conflict with existing styles; specificity wars; hard to maintain | Acceptable if CSS is well-organized with clear section comments; use a `/* === Modal Styles === */` separator block |
| Skipping truncation of trace data and relying on "it is usually small enough" | Faster implementation, no truncation logic | One large response (e.g., verbose Claude output) blows up the HTML file; no guarantee on response size | Never -- always truncate at the embedding layer with a configurable max |

## Integration Gotchas

Common mistakes when connecting modal UI to existing ARS subsystems.

| Integration | Common Mistake | Correct Approach |
|-------------|----------------|------------------|
| HTML template (`report.html`) | Adding modal HTML inline for each metric row, creating 35+ hidden `<div>` elements | Create ONE modal `<dialog>` element at the end of `<body>`; populate its content dynamically from a JSON data island when the trigger is clicked |
| CSS (`styles.css`) | Modal styles conflicting with existing `.metric-details-row` expand/collapse styling | Scope modal styles under `.modal-*` prefix; use `::backdrop` pseudo-element for dialog backdrop instead of custom overlay `<div>` |
| Go template data (`HTMLReportData`) | Adding 38 prompt template strings to every `HTMLSubScore` struct, inflating template data | Add prompts to a separate `PromptTemplates map[string]string` field on `HTMLReportData`; render as JSON data island; look up by metric key in JS |
| JSON output (`JSONReport`) | Adding trace data inside `categories[].metrics[]` changing existing struct shape | Add `"c7_traces"` as a new top-level field with `omitempty`; keep existing `metrics[]` structure untouched |
| Recommendation system (`recommend.go`) | Replacing existing `Action` field content with prompt-formatted content | Keep `Action` as-is (it serves terminal and existing HTML output); add new `ImprovementPrompt` field alongside it |
| Baseline comparison (`--baseline`) | New JSON fields cause comparison logic to report false deltas | Comparison logic should only compare fields present in BOTH baseline and current; new fields should be ignored in delta calculation |
| Existing expand/collapse JS | Modal open/close JS conflicting with metric row expand/collapse event handlers | Use event delegation on a parent container rather than individual `addEventListener` calls; prevent event bubbling from modal triggers to metric row click handlers |

## Performance Traps

Patterns that work at small scale but fail as usage grows.

| Trap | Symptoms | Prevention | When It Breaks |
|------|----------|------------|----------------|
| Rendering all modal content eagerly in the initial HTML | Page load time increases proportionally with number of metrics; 35+ hidden DOM elements add parsing overhead | Use a single reusable `<dialog>` populated on-demand from JSON data | When C7 is enabled (adds 5 more metrics with trace data) or when users scan multi-language projects (more categories) |
| Inline `<style>` growing beyond 15KB | First Contentful Paint delay; browser parses all CSS before rendering any content | Keep CSS under 15KB; use CSS custom properties for modal theme consistency; avoid duplicating existing styles in modal section | Current CSS is 9.7KB; adding modal styles should not double it |
| Attaching click handlers to every metric row individually | O(n) event listeners where n = number of metric rows; memory overhead on pages with 50+ metrics | Use event delegation: one `click` listener on `.categories` container that checks `event.target` | At 35+ metric rows (7 categories x 5 metrics) -- already the current scale |
| Storing prompt templates as escaped HTML strings in `data-*` attributes | DOM parsing overhead for large data attributes; HTML entity encoding overhead (~20% size increase) | Use a `<script type="application/json">` data island; parse once on first modal open; cache the parsed object | When prompt templates exceed 200 chars each (likely, given they include instructions and context) |

## Security Considerations

Domain-specific security issues for self-contained HTML reports with embedded data.

| Concern | Risk | Prevention |
|---------|------|------------|
| XSS from project paths in prompt templates | File paths from the scanned project are interpolated into prompts; a path like `"><script>alert(1)</script>` could execute | All interpolated values must be HTML-escaped when rendered in the DOM. Go's `html/template` auto-escapes in HTML context, but JS-side DOM manipulation (`innerHTML`) does not. Use `textContent` for all dynamic text insertion in JS. |
| Sensitive data in trace responses | C7 responses may contain code snippets from the scanned project; the HTML report is shareable | Truncate code snippets in trace data; add a visible warning "This report may contain code from your project" in the C7 trace modal header |
| Clipboard exfiltration concern | Users may paste improvement prompts into external LLM services, inadvertently sharing proprietary code context | Prompts should reference metric values and file paths, NOT embed source code. Add a note: "This prompt references your code structure but does not contain source code." |

## UX Pitfalls

Common user experience mistakes when adding interactive features to static reports.

| Pitfall | User Impact | Better Approach |
|---------|-------------|-----------------|
| Modal for every metric regardless of score | Users with good scores (8+) are presented with "improvement" prompts for metrics that do not need improvement; noise overwhelms signal | Only show modal triggers for metrics scoring below the `ShouldExpand` threshold (currently 6.0 in `descriptions.go:9`). Green-scored metrics show a brief "Good" note, not a modal. |
| Modal content identical to existing detailed description | Users open the modal and see the same "How to Improve" bullets already visible in the expanded metric details row | Modal content must be DISTINCT from the existing detailed description: detailed description explains the metric; modal provides a COPYABLE, STRUCTURED PROMPT for AI tools. Different content, different purpose. |
| No indication of which modals have been viewed | In a long report (7+ categories), users lose track of which improvement prompts they have already reviewed | Change the trigger button appearance after the modal has been opened (e.g., outline vs filled button). Use `sessionStorage` (not `localStorage`, since this is a transient file) to persist state within the browser session. |
| Copy button requires clicking inside the modal | Extra click to copy after opening the modal; mobile users must navigate a modal and then find the copy button | Consider a "copy prompt" button directly on the metric row (outside the modal) for the most common action. Modal provides additional context and full prompt preview. |
| Prompt text not selectable | Users cannot manually select prompt text as a fallback when clipboard API fails | Render prompt text in a `<pre>` element with `user-select: all` CSS property. Clicking anywhere in the prompt block selects all text for easy copy. |

## "Looks Done But Isn't" Checklist

Things that appear complete but are missing critical pieces.

- [ ] **Modal UI added:** Often missing focus restoration on close -- verify that pressing Escape returns focus to the trigger button, not the top of the page
- [ ] **Keyboard navigation works:** Often missing Shift+Tab backward cycle -- verify Tab cycles forward through modal elements and wraps; Shift+Tab cycles backward and wraps
- [ ] **Copy to clipboard works:** Often missing `file://` protocol test -- verify copy works when opening the HTML file directly from disk, not via localhost
- [ ] **Prompt templates complete:** Often missing coverage check -- verify every metric in `scoring.DefaultConfig()` has a corresponding prompt template; run `go test` to validate
- [ ] **JSON schema backward compatible:** Often missing baseline test -- load a pre-change JSON baseline file and verify `--baseline` comparison still works with no errors
- [ ] **Mobile modal works:** Often missing iOS Safari test -- verify modal scroll lock works on actual iOS Safari, not just Chrome DevTools responsive mode
- [ ] **Progressive enhancement:** Often missing no-JS test -- disable JavaScript in browser and verify all scores and prompt content are still readable (in `<details>` elements or visible HTML)
- [ ] **File size budget met:** Often missing after feature completion -- generate a report with C7 enabled and verify total HTML size is under 500KB
- [ ] **Trace data truncated:** Often missing edge case -- generate a report where one C7 response is 10KB+ and verify it is truncated to the budget limit
- [ ] **Print styles work:** Often missing -- verify `@media print` still produces clean output with modals closed and prompt content hidden (current print styles at `styles.css:529-546` force references open)
- [ ] **Existing expand/collapse not broken:** Often missing regression test -- verify the existing chevron expand/collapse on metric rows still works after modal JS is added

## Recovery Strategies

When pitfalls occur despite prevention, how to recover.

| Pitfall | Recovery Cost | Recovery Steps |
|---------|---------------|----------------|
| HTML file size explosion | LOW | Add truncation to the Go template data layer; set max lengths on `Prompt` and `Response` fields in the HTML-specific structs; add size assertion test |
| Modal not accessible | MEDIUM | Refactor custom `<div>` modal to `<dialog>` element; add focus trap and Escape handler; add `aria-labelledby`; test with screen reader |
| JSON schema broken | HIGH | This is hard to un-break if consumers have adapted to the broken schema. Must maintain backward compatibility with both old and new schemas. Add version negotiation. |
| Generic prompt templates | LOW | Add interpolation parameters to templates; update `buildAction` to pass file names and specific values; iterate based on user feedback |
| Mobile modal breakage | LOW-MEDIUM | Add `min(90vw, 700px)` width and `80vh` max-height; add iOS scroll lock workaround; test on real device |
| Progressive enhancement failure | MEDIUM | Add `<details>/<summary>` fallback elements; wrap modal triggers in JS-only enhancement; add no-JS test |
| Clipboard copy failure | LOW | Add try/catch with `execCommand('copy')` fallback; add visible `<pre>` block with `user-select: all`; add visual feedback |

## Pitfall-to-Phase Mapping

How roadmap phases should address these pitfalls.

| Pitfall | Prevention Phase | Verification |
|---------|------------------|--------------|
| File size explosion | Phase 1: Data model with truncation strategy | Test: HTML output with C7 data is under 500KB |
| Modal accessibility | Phase 2: Modal UI with `<dialog>` element | Test: Tab/Shift-Tab/Escape all work; screen reader announces dialog |
| JSON breaking changes | Phase 1: Schema extension with compatibility tests | Test: v1 baseline file loads and compares correctly |
| Generic prompts | Phase 3: Template authoring with interpolation | Test: every template contains at least one interpolated value |
| Mobile breakage | Phase 2: Modal UI with responsive CSS | Test: modal opens and closes correctly on 375px viewport |
| Progressive enhancement | Phase 2: `<details>` fallback before modal enhancement | Test: all content readable with JS disabled |
| Clipboard failure | Phase 2: Copy functionality with fallback chain | Test: copy works on `file://` URL; visible `<pre>` exists as fallback |
| Stale prompts | Phase 3: Template versioning and coverage tests | Test: every metric in config has a template; templates have version dates |
| Existing feature regression | All phases: Regression test suite | Test: expand/collapse, print styles, badge copy all still work after changes |

## Sources

- ARS codebase analysis: `internal/output/html.go` (HTML generation), `internal/output/templates/report.html` (template structure), `internal/output/templates/styles.css` (styling), `internal/output/json.go` (JSON schema), `internal/output/descriptions.go` (metric descriptions), `internal/recommend/recommend.go` (recommendation/action templates), `pkg/types/types.go:320-331` (C7DebugSample struct)
- [Cucumber HTML formatter issue #62: 314MB report from embedded trace data](https://github.com/cucumber/html-formatter/issues/62) -- direct precedent for file size explosion
- [W3C WAI ARIA APG: Dialog (Modal) Pattern](https://www.w3.org/WAI/ARIA/apg/patterns/dialog-modal/) -- authoritative modal accessibility requirements
- [MDN: HTML dialog element](https://developer.mozilla.org/en-US/docs/Web/HTML/Reference/Elements/dialog) -- native dialog capabilities and browser support
- [A11Y Collective: Mastering Accessible Modals](https://www.a11y-collective.com/blog/modal-accessibility/) -- focus trapping and keyboard navigation patterns
- [TestParty: WCAG 2.1.2 No Keyboard Trap Guide](https://testparty.ai/blog/wcag-2-1-2-no-keyboard-trap-2025-guide) -- Level A compliance requirements
- [CSS-Tricks: Prevent Page Scrolling When Modal is Open](https://css-tricks.com/prevent-page-scrolling-when-a-modal-is-open/) -- iOS Safari scroll lock techniques
- [Jay Freestone: Locking Body Scroll for Modals on iOS](https://www.jayfreestone.com/writing/locking-body-scroll-ios/) -- iOS-specific scroll lock implementation
- [WHATWG HTML Issue #7732: Prevent page scroll when dialog is visible](https://github.com/whatwg/html/issues/7732) -- Chrome 144 `overscroll-behavior` improvement (December 2025)
- [DebugBear: Avoid Large Base64 Data URLs in HTML and CSS](https://www.debugbear.com/blog/base64-data-urls-html-css) -- inline data performance impact
- [Confluent: Schema Evolution Best Practices](https://docs.confluent.io/platform/current/schema-registry/fundamentals/schema-evolution.html) -- backward compatibility patterns for JSON schemas
- [Jared Cunha: HTML Dialog Getting Accessibility and UX Right](https://jaredcunha.com/blog/html-dialog-getting-accessibility-and-ux-right) -- practical dialog implementation guide
- [Braintrust: Best Prompt Versioning Tools 2025](https://www.braintrust.dev/articles/best-prompt-versioning-tools-2025) -- prompt template lifecycle management

---
*Pitfalls research for: ARS Interactive HTML Enhancements (Modal UI, AI Prompts, Trace Data)*
*Researched: 2026-02-06*
