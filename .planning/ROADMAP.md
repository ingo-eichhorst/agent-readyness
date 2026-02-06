# Roadmap: ARS v0.0.6 - Interactive HTML Report Enhancements

## Overview

This milestone makes HTML reports transparent and actionable. Evidence data flows through the scoring pipeline to power two new interactive features: call trace modals (showing how each score was derived) and improvement prompt modals (providing copy-paste prompts to fix low-scoring areas). Five phases deliver the complete feature set, starting with the data foundation and shared UI, then building each modal type, and finishing with cross-cutting quality validation.

## Milestones

- [Archived] **v1 MVP** - Phases 1-5 (shipped 2026-02-01)
- [Archived] **v0.0.2 Complete Analysis Framework** - Phases 6-12 (shipped 2026-02-03)
- [Archived] **v0.0.3 Simplification & Polish** - Phases 13-17 (shipped 2026-02-04)
- [Archived] **v0.0.4 Metric Research & C7 Implementation** - Phases 18-25 (shipped 2026-02-05)
- [Archived] **v0.0.5 C7 Debug Infrastructure** - Phases 26-29 (shipped 2026-02-06)
- Active **v0.0.6 Interactive HTML Report Enhancements** - Phases 30-34 (in progress)

## Phases

- [x] **Phase 30: Evidence Data Flow** - Evidence types and extraction for all 7 categories
- [ ] **Phase 31: Modal UI Infrastructure** - Native dialog component with accessibility and clipboard support
- [ ] **Phase 32: Call Trace Modals** - Per-metric trace modals for C7 and C1-C6 scoring transparency
- [ ] **Phase 33: Improvement Prompt Modals** - Per-metric prompt modals with copy-to-clipboard
- [ ] **Phase 34: Testing & Quality** - Cross-cutting tests for evidence, size budget, schema, prompts, and accessibility

## Phase Details

### Phase 30: Evidence Data Flow
**Goal**: Every scored metric carries its top-5 worst offenders through the pipeline, visible in JSON output
**Depends on**: Nothing (first phase of milestone)
**Requirements**: EV-01, EV-02, EV-03, EV-04, EV-05
**Success Criteria** (what must be TRUE):
  1. Running `ars scan . --json | jq '.categories[0].sub_scores[0].evidence'` returns an array of evidence items with file_path, line, value, and description fields
  2. All 7 extractCx functions return evidence data (not empty arrays) for metrics that have offenders
  3. Running `ars scan . --json` with a baseline file from v0.0.5 still produces a valid comparison (no schema breakage)
  4. Running `ars scan .` without --json produces identical terminal output to v0.0.5 (evidence is invisible unless consumed)
**Plans**: 3 plans

Plans:
- [x] 30-01-PLAN.md -- Define EvidenceItem type, update SubScore and MetricExtractor, remove C7 overall_score
- [x] 30-02-PLAN.md -- Populate evidence in all 7 extractCx functions from existing CxMetrics data
- [x] 30-03-PLAN.md -- Wire evidence into JSON output with sub_scores field, validate backward compatibility

### Phase 31: Modal UI Infrastructure
**Goal**: HTML reports contain a reusable modal component that opens, scrolls, and closes correctly on desktop and mobile
**Depends on**: Nothing (can run in parallel with Phase 30)
**Requirements**: UI-01, UI-02, UI-03, UI-04, UI-05, UI-06, UI-07
**Success Criteria** (what must be TRUE):
  1. Opening a modal via a button in the HTML report displays a centered dialog with backdrop overlay
  2. Modal closes via Escape key, X button, or clicking the backdrop (all three methods work)
  3. Tab key cycles focus within the modal (does not escape to page behind)
  4. Modal content scrolls independently when content exceeds viewport height
  5. On mobile viewports (375px wide), the modal fills available width without horizontal overflow
**Plans**: 2 plans

Plans:
- [ ] 31-01-PLAN.md -- Native dialog element with showModal/close JS, responsive CSS, and iOS scroll lock
- [ ] 31-02-PLAN.md -- Progressive enhancement fallback, trigger button styling, and modal presence test

### Phase 32: Call Trace Modals
**Goal**: Users can click "View Trace" on any metric to see exactly how the score was derived
**Depends on**: Phase 30 (evidence data), Phase 31 (modal component)
**Requirements**: TR-01, TR-02, TR-03, TR-04, TR-05, TR-06, TR-07, TR-08
**Success Criteria** (what must be TRUE):
  1. Every metric row in the HTML report has a "View Trace" button that opens a modal
  2. C7 trace modal displays the full prompt sent to Claude, the full response received, and the score breakdown with matched indicators
  3. C1-C6 trace modal displays the current raw value, scoring breakpoints with where the value falls, and the top-5 worst offending files/functions
  4. JSON and shell command content in trace modals has syntax highlighting (distinct colors for keys, values, strings)
  5. Generated HTML report with C7 trace data embedded stays under 500KB total file size
**Plans**: TBD

Plans:
- [ ] 32-01: Add "View Trace" buttons and C7 trace modal rendering (prompts, responses, score breakdown)
- [ ] 32-02: Add C1-C6 scoring explanation trace (breakpoints, worst offenders) with syntax highlighting
- [ ] 32-03: Implement progressive enhancement fallback and enforce 500KB file size budget with truncation

### Phase 33: Improvement Prompt Modals
**Goal**: Users can click "Improve" on any metric to get a research-backed, project-specific prompt they can paste into an AI agent
**Depends on**: Phase 30 (evidence data for interpolation), Phase 31 (modal component)
**Requirements**: PR-01, PR-02, PR-03, PR-04, PR-05, PR-06, PR-07, PR-08, PR-09
**Success Criteria** (what must be TRUE):
  1. Every metric row in the HTML report has an "Improve" button that opens a modal with a prompt
  2. The prompt contains the metric's current score, a target score, and specific file/function names from the evidence data
  3. Clicking "Copy" places the full prompt text on the clipboard, and a "Copied!" confirmation appears
  4. On file:// protocol (local HTML files), copy still works via the execCommand fallback or shows a selectable pre block
  5. All 7 categories (C1-C7) have prompt templates with the structure: Context, Build/Test Commands, Task, Verification
**Plans**: TBD

Plans:
- [ ] 33-01: Create 7 category-level prompt templates with Context/Build/Task/Verification structure
- [ ] 33-02: Implement prompt interpolation with evidence data and GeneratePrompts() in recommend package
- [ ] 33-03: Render "Improve" buttons, prompt modals, copy-to-clipboard with fallback chain, and progressive enhancement

### Phase 34: Testing & Quality
**Goal**: Automated tests validate evidence extraction, file size budget, JSON compatibility, prompt coverage, accessibility, and responsive layout
**Depends on**: Phase 30, Phase 31, Phase 32, Phase 33 (tests validate all prior work)
**Requirements**: TEST-01, TEST-02, TEST-03, TEST-04, TEST-05, TEST-06
**Success Criteria** (what must be TRUE):
  1. `go test ./...` includes tests that verify evidence extraction produces non-empty results for all 7 categories
  2. A test generates an HTML report with C7 data and asserts file size is under 500KB
  3. A test loads a v0.0.5-era JSON baseline and verifies backward-compatible comparison still works
  4. A test verifies all 38 metrics (across 7 categories) map to a category-level prompt template
  5. A test validates ARIA attributes and keyboard navigation patterns in generated HTML
**Plans**: TBD

Plans:
- [ ] 34-01: Evidence extraction tests for all categories and JSON schema backward compatibility test
- [ ] 34-02: HTML file size budget test, prompt template coverage test, and accessibility validation test

## Progress

**Execution Order:** 30 and 31 can run in parallel, then 32 and 33 (both depend on 30+31), then 34

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 30. Evidence Data Flow | 3/3 | ✓ Complete | 2026-02-06 |
| 31. Modal UI Infrastructure | 0/2 | Not started | - |
| 32. Call Trace Modals | 0/3 | Not started | - |
| 33. Improvement Prompt Modals | 0/3 | Not started | - |
| 34. Testing & Quality | 0/2 | Not started | - |
