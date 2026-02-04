---
phase: 18-c1-code-health
plan: 01
subsystem: documentation
tags: [citations, quality-protocols, style-guide, url-verification]

# Dependency graph
requires:
  - phase: none
    provides: First phase of v0.0.4 milestone
provides:
  - Citation style guide with (Author, Year) format
  - URL verification protocol with curl -I checks
  - Source quality checklist for foundational vs AI-era sources
  - Retraction Watch check process
affects: [19-c2-semantics, 20-c3-architecture, 21-c4-documentation, 22-c5-temporal, 23-c6-testing, 24-c7-agent]

# Tech tracking
tech-stack:
  added: []
  patterns: ["inline citation with span.citation", "foundational + AI-era citation mix"]

key-files:
  created: [docs/CITATION-GUIDE.md]
  modified: []

key-decisions:
  - "2-3 citations per metric target (not academic over-citation)"
  - "DOI preferred but ArXiv acceptable for AI-era research"
  - "Retraction Watch only for suspicious sources (default trust reputable venues)"

patterns-established:
  - "Citation format: <span class=\"citation\">(Author, Year)</span>"
  - "Research Evidence section as primary citation location"
  - "Foundational (pre-2021) + AI-era (2021+) balance per metric"

# Metrics
duration: 3min
completed: 2026-02-04
---

# Phase 18 Plan 01: Citation Quality Guide Summary

**Citation quality protocols establishing (Author, Year) inline format, URL verification with curl -I, source quality checklist distinguishing foundational from AI-era sources, and Retraction Watch check process for all C1-C7 metric descriptions.**

## Performance

- **Duration:** 3 min
- **Started:** 2026-02-04T20:02:58Z
- **Completed:** 2026-02-04T20:05:20Z
- **Tasks:** 1
- **Files created:** 1

## Accomplishments

- Created comprehensive citation style guide (441 lines) with inline format rules
- Documented URL verification protocol including curl -I, DOI resolution, and browser checks
- Established source quality criteria separating foundational (pre-2021) from AI-era (2021+)
- Defined Retraction Watch check process with trust levels by source type
- Included examples from actual ARS citations (McCabe, Fowler, Borg)

## Task Commits

Each task was committed atomically:

1. **Task 1: Create citation style guide document** - `cd38d5e` (docs)

## Files Created/Modified

- `docs/CITATION-GUIDE.md` - Citation quality protocols for all C1-C7 phases (441 lines)

## Decisions Made

- **Citation density:** 2-3 per metric (1 foundational + 1-2 AI-era) - avoids academic over-citation while maintaining evidence-based documentation
- **DOI preference:** DOIs preferred for link stability but ArXiv URLs acceptable for AI-era research (standard in the field)
- **Retraction checking:** Default trust for reputable venues (IEEE, ACM, ArXiv), only check Retraction Watch if source seems suspicious
- **"How to Improve" sections:** No citations - actionable guidance doesn't need academic backing

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Citation quality protocols ready for use in Plan 02 (C1 metric citations)
- All subsequent phases (C2-C7) can reference this guide
- docs/ directory created and ready for future documentation

---
*Phase: 18-c1-code-health*
*Completed: 2026-02-04*
