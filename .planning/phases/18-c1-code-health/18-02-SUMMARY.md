---
phase: 18-c1-code-health
plan: 02
subsystem: documentation
tags: [citations, research-evidence, c1-code-health, McCabe, Fowler, Borg, Parnas, Martin]

# Dependency graph
requires:
  - phase: 18-01
    provides: Citation quality protocols (style guide, URL verification, source quality)
provides:
  - All 6 C1 metrics with inline (Author, Year) citations
  - 7 C1 reference entries in citations.go
  - Each metric has foundational + AI-era citations
affects: [19-c2-semantics, 20-c3-architecture, 21-c4-documentation, 22-c5-temporal, 23-c6-testing, 24-c7-agent]

# Tech tracking
tech-stack:
  added: []
  patterns: ["foundational + AI-era citation balance per metric"]

key-files:
  created: []
  modified: [internal/output/descriptions.go, internal/output/citations.go]

key-decisions:
  - "Duplicate Parnas/Gamma entries for C1 (self-contained category references)"
  - "Chowdhury (2022) used for func_length despite Java-specific study"
  - "ACM URLs kept as DOI-format despite 403 bot protection (DOIs are permanent)"

patterns-established:
  - "2-3 citations per metric with Research Evidence section primary location"
  - "Brief descriptions include key citation when making quantified claim"
  - "All quantified claims (36-44%, under 25 lines) have explicit attribution"

# Metrics
duration: 5min
completed: 2026-02-04
---

# Phase 18 Plan 02: C1 Code Health Citations Summary

**Research-backed inline citations added to all 6 C1 metrics (complexity, function length, file size, coupling, duplication) with 7 reference entries covering McCabe (1976), Fowler (1999), Parnas (1972), Martin (2003), Gamma (1994), Chowdhury (2022), and Borg (2026).**

## Performance

- **Duration:** 5 min
- **Started:** 2026-02-04T20:05:00Z
- **Completed:** 2026-02-04T20:10:00Z
- **Tasks:** 3
- **Files modified:** 2

## Accomplishments

- Added inline citations to all 6 C1 metric descriptions with consistent span.citation format
- Expanded C1 References from 3 to 7 entries covering foundational and AI-era sources
- Verified all 7 C1 citation URLs accessible (IEEE, ACM with DOI, ArXiv, Pearson, Wikipedia)
- Each metric now has at least 1 foundational (pre-2021) and 1 AI-era (2021+) citation

## Task Commits

Each task was committed atomically:

1. **Task 1: Update C1 metric descriptions with citations** - `e322ca4` (feat)
2. **Task 2: Add C1 reference entries to citations.go** - `a3222df` (feat)
3. **Task 3: Verify all C1 citation URLs are accessible** - No commit (verification only)

## Files Created/Modified

- `internal/output/descriptions.go` - C1 metrics now have inline citations with Research Evidence sections
- `internal/output/citations.go` - Added Parnas, Martin, Gamma, Chowdhury entries for C1 category

## Citation Coverage Summary

| Metric | Foundational | AI-Era |
|--------|--------------|--------|
| complexity_avg | McCabe (1976), Fowler (1999) | Borg (2026) |
| func_length_avg | Fowler (1999), Chowdhury (2022) | Borg (2026) |
| file_size_avg | Parnas (1972), Gamma (1994) | Borg (2026) |
| afferent_coupling_avg | Parnas (1972), Martin (2003) | Borg (2026) |
| efferent_coupling_avg | Martin (2003), Parnas (1972) | Borg (2026) |
| duplication_rate | Fowler (1999) | Borg (2026) |

## Decisions Made

- **Duplicate entries for C1:** Added Parnas (1972) and Gamma (1994) as C1 entries even though they exist in C3. Each category should be self-contained in References section.
- **Chowdhury (2022) inclusion:** Used despite being Java-specific because it provides the only empirical threshold (24 SLOC) for function length. Documented as supporting evidence.
- **ACM URL format:** Kept `dl.acm.org/doi/` format despite HTTP 403 bot protection response. DOI links are permanent; 403 is only for automated requests, works in browsers.

## Deviations from Plan

None - plan executed exactly as written.

## URL Verification Results

| Source | URL | Status |
|--------|-----|--------|
| McCabe (1976) | ieeexplore.ieee.org | HTTP 200 (with UA) |
| Fowler (1999) | martinfowler.com | HTTP 200 |
| Borg (2026) | arxiv.org | HTTP 200 |
| Parnas (1972) | dl.acm.org (DOI) | DOI resolves correctly |
| Martin (2003) | pearson.com | HTTP 200 |
| Gamma (1994) | wikipedia.org | HTTP 200 |
| Chowdhury (2022) | arxiv.org | HTTP 200 |

## Issues Encountered

- **IEEE/ACM bot protection:** Both ieeexplore.ieee.org and dl.acm.org return 418/403 for automated requests. Resolved by testing with browser User-Agent and confirming DOI resolution works. URLs are valid and work in browsers.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- C1 citations complete, pattern established for C2-C7 phases
- Citation style guide (18-01) and C1 implementation (18-02) serve as reference
- Phase 19 (C2 Semantics) can follow same pattern

---
*Phase: 18-c1-code-health*
*Completed: 2026-02-04*
