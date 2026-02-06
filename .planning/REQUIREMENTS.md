# Requirements: Agent Readiness Score (ARS)

**Defined:** 2026-02-06
**Core Value:** Accurate, evidence-based scoring that predicts agent success and identifies specific improvements teams should make before adopting AI coding agents.

## v0.0.6 Requirements (Interactive HTML Report)

Requirements for adding call trace and improvement prompt modals to HTML reports.

### Evidence Data Flow

- [ ] **EV-01**: SubScore type includes Evidence field with top-5 worst offenders per metric
- [ ] **EV-02**: MetricExtractor signature returns evidence alongside score and raw value
- [ ] **EV-03**: All 7 extractCx functions populate evidence for their metrics
- [ ] **EV-04**: Evidence includes file path, line number, value, and description fields
- [ ] **EV-05**: JSON output includes evidence data with omitempty for backward compatibility

### Modal UI Infrastructure

- [ ] **UI-01**: Native HTML `<dialog>` element for all modals
- [ ] **UI-02**: Modal opens via JavaScript showModal() method
- [ ] **UI-03**: Modal closes via Escape key, X button, or backdrop click
- [ ] **UI-04**: Modal has keyboard focus trapping and ARIA attributes
- [ ] **UI-05**: Modal width is responsive (min(90vw, 700px) for mobile support)
- [ ] **UI-06**: Modal content is scrollable with max-height constraint
- [ ] **UI-07**: iOS Safari scroll lock using position:fixed workaround

### Call Trace Modal (Issue #56)

- [ ] **TR-01**: Per-metric "View Trace" button in HTML report metric rows
- [ ] **TR-02**: C7 trace modal shows full prompts and responses for all samples
- [ ] **TR-03**: C7 trace shows score breakdown with matched indicators
- [ ] **TR-04**: C1-C6 trace modal shows scoring explanation (current value, breakpoints, target)
- [ ] **TR-05**: C1-C6 trace shows top-5 worst offenders (files/functions with highest values)
- [ ] **TR-06**: Syntax highlighting for JSON and shell command content
- [ ] **TR-07**: Trace data respects 500KB total file size budget
- [ ] **TR-08**: Progressive enhancement: content accessible in <details> fallback without JS

### Improvement Prompt Modal (Issue #57)

- [ ] **PR-01**: Per-metric "Improve" button in HTML report metric rows
- [ ] **PR-02**: Modal shows research-backed prompt template for the category
- [ ] **PR-03**: Prompt includes current score, target score, and metric-specific guidance
- [ ] **PR-04**: Prompt interpolates project-specific data (file names, metric values, thresholds)
- [ ] **PR-05**: Copy-to-clipboard button with "Copied!" visual feedback
- [ ] **PR-06**: Clipboard fallback chain: Clipboard API → execCommand → visible <pre> block
- [ ] **PR-07**: 7 category-level prompt templates (C1-C7) based on research structure
- [ ] **PR-08**: Prompt structure: Context → Build/Test → Task → Verification
- [ ] **PR-09**: Progressive enhancement: prompt visible in <details> fallback without JS

### Testing & Quality

- [ ] **TEST-01**: Unit tests verify evidence extraction for all metrics
- [ ] **TEST-02**: File size budget test fails if HTML report exceeds 500KB
- [ ] **TEST-03**: JSON schema compatibility test validates backward compatibility
- [ ] **TEST-04**: Prompt template coverage test ensures all 38 metrics mapped to category templates
- [ ] **TEST-05**: Accessibility test validates keyboard navigation and ARIA attributes
- [ ] **TEST-06**: Mobile responsive test validates modal layout on small screens

## Out of Scope

| Feature | Reason |
|---------|--------|
| Per-metric prompt templates (38 total) | Too many to maintain; category-level (7 templates) sufficient |
| LLM-generated dynamic prompts | Adds latency and cost; static templates with interpolation sufficient |
| Interactive prompt editing in modal | Scope creep; copy-paste workflow is simpler |
| Real-time trace streaming | Batch analysis only; no streaming infrastructure |
| Trace filtering/search UI | Show everything; users can browser search if needed |
| C2/C4 per-file evidence | Aggregate metrics only; synthesizing fake evidence adds complexity |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| EV-01 | Phase 30 | Pending |
| EV-02 | Phase 30 | Pending |
| EV-03 | Phase 30 | Pending |
| EV-04 | Phase 30 | Pending |
| EV-05 | Phase 30 | Pending |
| UI-01 | Phase 31 | Pending |
| UI-02 | Phase 31 | Pending |
| UI-03 | Phase 31 | Pending |
| UI-04 | Phase 31 | Pending |
| UI-05 | Phase 31 | Pending |
| UI-06 | Phase 31 | Pending |
| UI-07 | Phase 31 | Pending |
| TR-01 | Phase 32 | Pending |
| TR-02 | Phase 32 | Pending |
| TR-03 | Phase 32 | Pending |
| TR-04 | Phase 32 | Pending |
| TR-05 | Phase 32 | Pending |
| TR-06 | Phase 32 | Pending |
| TR-07 | Phase 32 | Pending |
| TR-08 | Phase 32 | Pending |
| PR-01 | Phase 33 | Pending |
| PR-02 | Phase 33 | Pending |
| PR-03 | Phase 33 | Pending |
| PR-04 | Phase 33 | Pending |
| PR-05 | Phase 33 | Pending |
| PR-06 | Phase 33 | Pending |
| PR-07 | Phase 33 | Pending |
| PR-08 | Phase 33 | Pending |
| PR-09 | Phase 33 | Pending |
| TEST-01 | Phase 34 | Pending |
| TEST-02 | Phase 34 | Pending |
| TEST-03 | Phase 34 | Pending |
| TEST-04 | Phase 34 | Pending |
| TEST-05 | Phase 34 | Pending |
| TEST-06 | Phase 34 | Pending |

**Coverage:**
- v0.0.6 requirements: 35 total
- Mapped to phases: 35/35
- Unmapped: 0

---
*Requirements defined: 2026-02-06*
*Last updated: 2026-02-06 after roadmap creation*
