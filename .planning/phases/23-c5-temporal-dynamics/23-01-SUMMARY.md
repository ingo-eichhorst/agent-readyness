---
phase: 23
plan: 01
subsystem: output
tags: [citations, C5, temporal-dynamics, research]
dependency-graph:
  requires: [18-01, 18-02]
  provides: [C5-citations, temporal-dynamics-evidence]
  affects: [24-01]
tech-stack:
  added: []
  patterns: [inline-citations, research-evidence-sections]
key-files:
  created: []
  modified: [internal/output/descriptions.go, internal/output/citations.go]
decisions:
  - id: "23-01-01"
    description: "Tornhill labeled as practitioner literature throughout"
  - id: "23-01-02"
    description: "Borg et al. noted as indirect support for temporal metrics"
  - id: "23-01-03"
    description: "Commit stability research gap acknowledged with practitioner consensus thresholds"
metrics:
  duration: "~15 minutes"
  completed: 2026-02-05
---

# Phase 23 Plan 01: C5 Temporal Dynamics Citations Summary

**One-liner:** Added 10 C5 citations covering foundational change history research (Graves, Nagappan, Kim, Gall, D'Ambros, Bird, Eick, Hassan) and practitioner synthesis (Tornhill) to all 5 temporal dynamics metrics.

## What Was Built

### C5 Metric Citations (descriptions.go)

All 5 C5 Temporal Dynamics metrics now have:
- **Inline citations** in Brief fields with `<span class="citation">` markup
- **Research Evidence sections** in Detailed fields with 2-3 paragraphs of citations

| Metric | Primary Citation | Additional Citations |
|--------|-----------------|---------------------|
| churn_rate | Kim et al. (2007) | Graves (2000), Nagappan (2005), Tornhill (practitioner), Borg (indirect) |
| temporal_coupling_pct | Gall et al. (1998) | D'Ambros (2009), Tornhill (practitioner), Borg (indirect) |
| author_fragmentation | Bird et al. (2011) | Kim (2007), Tornhill (practitioner), Borg (indirect) |
| commit_stability | Eick et al. (2001) | Graves (2000), Tornhill (practitioner), research gap noted |
| hotspot_concentration | Nagappan & Ball (2005) | Hassan (2009), Tornhill (practitioner), Borg (indirect) |

### C5 Reference Entries (citations.go)

Expanded from 2 to 10 entries:

1. **Tornhill (2015)** - Updated: labeled as "Practitioner synthesis" with ISBN
2. **Kim et al. (2007)** - Updated: DOI format URL
3. **Graves et al. (2000)** - NEW: Process measures outperform product metrics
4. **Nagappan & Ball (2005)** - NEW: 89% accuracy on Windows Server 2003
5. **Gall et al. (1998)** - NEW: Pioneered logical coupling detection
6. **D'Ambros et al. (2009)** - NEW: Change coupling correlates with defects
7. **Bird et al. (2011)** - NEW: Ownership measures relate to faults
8. **Eick et al. (2001)** - NEW: Defines code decay
9. **Hassan (2009)** - NEW: Change complexity predicts faults
10. **Borg et al. (2026)** - NEW: AI agent reliability (indirect support)

## How It Works

The citations follow the established pattern from Phases 18-22:

1. **Brief field**: One inline citation to primary source
2. **Research Evidence section**: 2-3 paragraphs with foundational research first, then practitioner synthesis, finally AI-era context with caveats
3. **Practitioner labeling**: Tornhill explicitly labeled as "influential practitioner literature"
4. **Indirect support**: Borg et al. explicitly noted as supporting code health broadly, not temporal metrics specifically

## Decisions Made

| Decision | Rationale |
|----------|-----------|
| Tornhill labeled as practitioner | Not peer-reviewed research; influential synthesis of academic work |
| Borg et al. as indirect support | Validates code health, not temporal metrics specifically |
| Commit stability research gap | Limited dedicated research; thresholds are practitioner consensus |
| DOI format for all academic papers | Standard for permanent academic links |

## Deviations from Plan

None - plan executed exactly as written.

## Testing Done

- `go build ./...` - passes
- `go test ./internal/output/...` - passes
- All 10 C5 citation URLs verified accessible (DOI redirects, book/ArXiv 200 OK)

## Commits

1. `9fd1faa` - feat(23-01): add inline citations and Research Evidence to C5 metrics
2. `3da2c47` - feat(23-01): add C5 reference entries to citations.go

## Next Phase Readiness

Phase 24 (C7 Agent Metrics) is ready to proceed:
- C5 citations complete with 10 entries
- All temporal metrics have inline citations and Research Evidence sections
- Established pattern for C7: cite adjacent research (LLM code generation, SWE-bench) and acknowledge nascent field gaps
