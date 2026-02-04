# Requirements: ARS v0.0.3

**Defined:** 2026-02-03
**Core Value:** Accurate, evidence-based scoring that predicts agent success and identifies specific improvements teams should make before adopting AI coding agents.

## v0.0.3 Requirements

Requirements for v0.0.3 milestone. Each maps to roadmap phases.

### LLM Integration (Issue #6)

- [x] **LLM-01**: Remove `--enable-c4-llm` flag — LLM analysis always active when Claude CLI available
- [x] **LLM-02**: C4 documentation quality uses Claude Code CLI (`claude -p`) instead of Anthropic SDK
- [x] **LLM-03**: C7 agent evaluation continues using Claude Code CLI (already implemented)
- [x] **LLM-04**: Remove Anthropic SDK dependency from go.mod
- [x] **LLM-05**: Remove `ANTHROPIC_API_KEY` requirement — Claude CLI handles auth

### Badge Generation (Issue #5)

- [x] **BADGE-01**: `--badge` flag generates shields.io markdown URL to stdout
- [x] **BADGE-02**: Badge color reflects score (red <4, orange 4-6, yellow 6-8, green 8+)
- [x] **BADGE-03**: Badge shows tier name and score (e.g., "Agent-Ready 8.2/10")

### HTML Report (Issue #7)

- [x] **HTML-01**: Each metric has brief description (1-2 sentences) always visible
- [x] **HTML-02**: Each metric has expandable detailed description with research citations
- [x] **HTML-03**: Expandable sections use CSS-only `<details>/<summary>` (no JavaScript)
- [x] **HTML-04**: Categories scoring <6.0 start expanded by default

### README (Issue #4)

- [x] **README-01**: Add Go Reference badge
- [x] **README-02**: Add Go Report Card badge
- [x] **README-03**: Add License badge
- [x] **README-04**: Add Release badge

### Codebase Organization (Issue #3)

- [x] **REORG-01**: Create `internal/analyzer/c1/`, `c2/`, ... `c7/` subdirectories
- [x] **REORG-02**: Move category-specific files into respective subdirectories
- [x] **REORG-03**: Fix all import paths affected by restructure
- [x] **REORG-04**: Re-exports in `internal/analyzer/analyzer.go` for backward compatibility

### Testing (Issue #2)

- [x] **TEST-01**: Test commands always include `-coverprofile` flag
- [x] **TEST-02**: Coverage data available for C6 analysis

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
| LLM-01 | Phase 15 | Complete |
| LLM-02 | Phase 15 | Complete |
| LLM-03 | Phase 15 | Complete |
| LLM-04 | Phase 15 | Complete |
| LLM-05 | Phase 15 | Complete |
| BADGE-01 | Phase 13 | Complete |
| BADGE-02 | Phase 13 | Complete |
| BADGE-03 | Phase 13 | Complete |
| HTML-01 | Phase 14 | Complete |
| HTML-02 | Phase 14 | Complete |
| HTML-03 | Phase 14 | Complete |
| HTML-04 | Phase 14 | Complete |
| README-01 | Phase 17 | Complete |
| README-02 | Phase 17 | Complete |
| README-03 | Phase 17 | Complete |
| README-04 | Phase 17 | Complete |
| REORG-01 | Phase 16 | Complete |
| REORG-02 | Phase 16 | Complete |
| REORG-03 | Phase 16 | Complete |
| REORG-04 | Phase 16 | Complete |
| TEST-01 | Phase 17 | Complete |
| TEST-02 | Phase 17 | Complete |

**Coverage:**
- v0.0.3 requirements: 22 total
- Mapped to phases: 22
- Unmapped: 0

---
*Requirements defined: 2026-02-03*
*Last updated: 2026-02-03 after roadmap creation*
