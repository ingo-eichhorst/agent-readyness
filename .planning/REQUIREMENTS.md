# Requirements: ARS v0.0.3

**Defined:** 2026-02-03
**Core Value:** Accurate, evidence-based scoring that predicts agent success and identifies specific improvements teams should make before adopting AI coding agents.

## v0.0.3 Requirements

Requirements for v0.0.3 milestone. Each maps to roadmap phases.

### LLM Integration (Issue #6)

- [ ] **LLM-01**: Remove `--enable-c4-llm` flag — LLM analysis always active when Claude CLI available
- [ ] **LLM-02**: C4 documentation quality uses Claude Code CLI (`claude -p`) instead of Anthropic SDK
- [ ] **LLM-03**: C7 agent evaluation continues using Claude Code CLI (already implemented)
- [ ] **LLM-04**: Remove Anthropic SDK dependency from go.mod
- [ ] **LLM-05**: Remove `ANTHROPIC_API_KEY` requirement — Claude CLI handles auth

### Badge Generation (Issue #5)

- [ ] **BADGE-01**: `--badge` flag generates shields.io markdown URL to stdout
- [ ] **BADGE-02**: Badge color reflects score (red <4, orange 4-6, yellow 6-8, green 8+)
- [ ] **BADGE-03**: Badge shows tier name and score (e.g., "Agent-Ready 8.2/10")

### HTML Report (Issue #7)

- [ ] **HTML-01**: Each metric has brief description (1-2 sentences) always visible
- [ ] **HTML-02**: Each metric has expandable detailed description with research citations
- [ ] **HTML-03**: Expandable sections use CSS-only `<details>/<summary>` (no JavaScript)
- [ ] **HTML-04**: Categories scoring <6.0 start expanded by default

### README (Issue #4)

- [ ] **README-01**: Add Go Reference badge
- [ ] **README-02**: Add Go Report Card badge
- [ ] **README-03**: Add License badge
- [ ] **README-04**: Add Release badge

### Codebase Organization (Issue #3)

- [ ] **REORG-01**: Create `internal/analyzer/c1/`, `c2/`, ... `c7/` subdirectories
- [ ] **REORG-02**: Move category-specific files into respective subdirectories
- [ ] **REORG-03**: Fix all import paths affected by restructure
- [ ] **REORG-04**: Re-exports in `internal/analyzer/analyzer.go` for backward compatibility

### Testing (Issue #2)

- [ ] **TEST-01**: Test commands always include `-coverprofile` flag
- [ ] **TEST-02**: Coverage data available for C6 analysis

## Future Requirements

Deferred to later milestones.

### Badge Generation (v0.0.4+)

- **BADGE-04**: Local SVG badge generation (offline-first)
- **BADGE-05**: Badge customization options (label text, style)

### HTML Report (v0.0.4+)

- **HTML-05**: Expand all/collapse all JavaScript controls
- **HTML-06**: Dark mode support

## Out of Scope

Explicitly excluded. Documented to prevent scope creep.

| Feature | Reason |
|---------|--------|
| Multiple LLM providers | Single provider (Claude CLI) simplifies auth and maintenance |
| Badge hosting service | shields.io URLs are sufficient; no need for custom badge server |
| Interactive HTML charts | Current go-charts radar/bar charts are sufficient |
| Real-time badge updates | Static badge generation is appropriate for CLI tool |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| LLM-01 | Phase 15 | Pending |
| LLM-02 | Phase 15 | Pending |
| LLM-03 | Phase 15 | Pending |
| LLM-04 | Phase 15 | Pending |
| LLM-05 | Phase 15 | Pending |
| BADGE-01 | Phase 13 | Pending |
| BADGE-02 | Phase 13 | Pending |
| BADGE-03 | Phase 13 | Pending |
| HTML-01 | Phase 14 | Pending |
| HTML-02 | Phase 14 | Pending |
| HTML-03 | Phase 14 | Pending |
| HTML-04 | Phase 14 | Pending |
| README-01 | Phase 17 | Pending |
| README-02 | Phase 17 | Pending |
| README-03 | Phase 17 | Pending |
| README-04 | Phase 17 | Pending |
| REORG-01 | Phase 16 | Pending |
| REORG-02 | Phase 16 | Pending |
| REORG-03 | Phase 16 | Pending |
| REORG-04 | Phase 16 | Pending |
| TEST-01 | Phase 17 | Pending |
| TEST-02 | Phase 17 | Pending |

**Coverage:**
- v0.0.3 requirements: 22 total
- Mapped to phases: 22
- Unmapped: 0

---
*Requirements defined: 2026-02-03*
*Last updated: 2026-02-03 after roadmap creation*
