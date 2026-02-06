---
phase: 30-evidence-data-flow
verified: 2026-02-06T21:30:00Z
status: passed
score: 21/21 must-haves verified
re_verification: false
---

# Phase 30: Evidence Data Flow Verification Report

**Phase Goal:** Every scored metric carries its top-5 worst offenders through the pipeline, visible in JSON output
**Verified:** 2026-02-06T21:30:00Z
**Status:** passed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | EvidenceItem type exists with file_path, line, value, description fields and JSON tags | ✓ VERIFIED | Found in pkg/types/scoring.go lines 19-25 with all 4 fields and correct json tags |
| 2 | SubScore type has Evidence []EvidenceItem field with json:evidence tag (no omitempty) | ✓ VERIFIED | Found in pkg/types/scoring.go line 34 with correct json tag, no omitempty |
| 3 | MetricExtractor returns three values: rawValues, unavailable, evidence | ✓ VERIFIED | Signature in scorer.go lines 18-22 matches specification exactly |
| 4 | C7 overall_score metric is removed from config (not just zero-weight) | ✓ VERIFIED | grep finds 0 occurrences of "overall_score" in config.go |
| 5 | All existing tests still pass after signature changes | ✓ VERIFIED | go test ./internal/scoring/... and ./internal/output/... all pass |
| 6 | extractC1 returns top-5 worst offenders for complexity, func_length, duplication, coupling | ✓ VERIFIED | Lines 196-340 show sorting and top-5 extraction with EvidenceItem creation |
| 7 | extractC3 returns evidence for circular_deps and dead_exports metrics | ✓ VERIFIED | Lines 407-450 populate evidence for these metrics |
| 8 | extractC5 returns evidence from TopHotspots and CoupledPairs | ✓ VERIFIED | Lines 643-700 extract evidence from git-based data |
| 9 | extractC6 returns evidence for test_isolation and assertion_density_avg | ✓ VERIFIED | Lines 558-605 extract evidence from TestFunctions |
| 10 | C2, C4, C7 extractors return empty evidence arrays (aggregate-only metrics) | ✓ VERIFIED | extractC2 (lines 365-371), extractC7 (lines 765-770) return empty arrays |
| 11 | All evidence arrays are non-nil (empty []EvidenceItem{} not nil) | ✓ VERIFIED | scoreMetrics (lines 792-795) ensures nil→empty conversion |
| 12 | Running ars scan . --json shows sub_scores field in output | ✓ VERIFIED | Actual scan output shows "sub_scores" array in JSON categories |
| 13 | JSON output uses sub_scores field name (not metrics) | ✓ VERIFIED | JSONCategory struct line 27 uses json:"sub_scores" tag |
| 14 | sub_scores are always present in JSON (not gated by verbose flag) | ✓ VERIFIED | BuildJSONReport lines 72-85 populate unconditionally, comment at line 55 confirms verbose deprecated |
| 15 | Evidence arrays are [] not null in JSON output | ✓ VERIFIED | JSONMetric.Evidence has no omitempty tag (line 37), nil→empty conversion at lines 73-76 |
| 16 | Terminal output is unchanged (no evidence visible in non-JSON mode) | ✓ VERIFIED | Terminal output logic unchanged, evidence only in JSON path |
| 17 | JSON version is "2" (schema change signal) | ✓ VERIFIED | BuildJSONReport line 59 sets Version: "2" |
| 18 | Evidence items have correct field structure in JSON | ✓ VERIFIED | Live scan shows file_path, line, value, description in output |
| 19 | C1 metrics return populated evidence (not empty) when data exists | ✓ VERIFIED | Live scan shows 5 complexity evidence items with actual function data |
| 20 | Evidence descriptions are human-readable | ✓ VERIFIED | Scan output shows "pyDetectDeadCode has complexity 30", "is 131 lines" etc |
| 21 | Loading v0.0.5 baseline JSON still works for comparison | ✓ VERIFIED | TestJSONBaselineBackwardCompatibility passes, loadBaseline only reads category-level fields |

**Score:** 21/21 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| pkg/types/scoring.go | EvidenceItem type and updated SubScore | ✓ VERIFIED | Lines 19-25: EvidenceItem with 4 fields + json tags. Line 34: SubScore.Evidence field |
| pkg/types/scoring.go | All SubScore fields have json tags | ✓ VERIFIED | Lines 29-34: all fields have json tags (metric_name, raw_value, score, weight, available, evidence) |
| internal/scoring/config.go | C7 config with 5 MECE metrics only | ✓ VERIFIED | overall_score completely removed (0 grep matches) |
| internal/scoring/scorer.go | Updated MetricExtractor signature | ✓ VERIFIED | Lines 18-22: returns (rawValues, unavailable, evidence) |
| internal/scoring/scorer.go | All 7 extractCx return 3 values | ✓ VERIFIED | extractC1-C7 all have matching signatures with evidence map return |
| internal/scoring/scorer.go | scoreMetrics wires evidence into SubScore | ✓ VERIFIED | Lines 787-816: accepts evidence param, wires into SubScore.Evidence with nil guard |
| internal/scoring/scorer.go | Populated evidence in C1/C3/C5/C6 | ✓ VERIFIED | Top-5 extraction pattern with sort-copy-limit visible in all 4 extractors |
| internal/output/json.go | Updated JSON types with sub_scores | ✓ VERIFIED | Lines 23-28: JSONCategory.SubScores (no omitempty), line 37: JSONMetric.Evidence |
| internal/output/json.go | BuildJSONReport without verbose gate | ✓ VERIFIED | Lines 72-85: always populate sub_scores, comment at line 55 notes verbose deprecated |
| internal/output/json_test.go | Tests for evidence and backward compatibility | ✓ VERIFIED | TestJSONEvidenceNotNull, TestJSONEvidenceWithData, TestJSONBaselineBackwardCompatibility all present and passing |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| internal/scoring/scorer.go | pkg/types/scoring.go | MetricExtractor returns types.EvidenceItem | ✓ WIRED | Line 21: evidence map[string][]types.EvidenceItem in return signature |
| internal/scoring/scorer.go | internal/scoring/config.go | C7 config no longer has overall_score | ✓ WIRED | extractC7 lines 751-757 list exactly 5 metrics, no overall_score |
| internal/scoring/scorer.go | extractCx functions | Evidence map population | ✓ WIRED | All extractors create evidence map and populate with EvidenceItems |
| internal/scoring/scorer.go | scoreMetrics | Evidence passed through | ✓ WIRED | Line 154: extractor call captures evidence, line 163: passed to scoreMetrics |
| internal/output/json.go | pkg/types/scoring.go | JSONMetric.Evidence uses types.EvidenceItem | ✓ WIRED | Line 37: Evidence []types.EvidenceItem with correct type import |
| internal/output/json.go | BuildJSONReport | Evidence copied to JSONMetric | ✓ WIRED | Lines 73-76: ev := ss.Evidence with nil→empty conversion, line 83: Evidence: ev |
| extractC1 | C1Metrics.Functions | Top-5 by complexity/length | ✓ WIRED | Lines 197-240: copy m.Functions, sort by metric, take top 5, build EvidenceItems |
| extractC3 | C3Metrics.CircularDeps | Top-5 cycles | ✓ WIRED | Lines 407-427: iterate CircularDeps, build evidence with cycle description |
| extractC5 | C5Metrics.TopHotspots | Git churn evidence | ✓ WIRED | Lines 643-700: use TopHotspots and CoupledPairs for evidence |
| extractC6 | C6Metrics.TestFunctions | Test isolation evidence | ✓ WIRED | Lines 558-605: filter TestFunctions by HasExternalDep, build evidence |

### Requirements Coverage

| Requirement | Status | Blocking Issue |
|-------------|--------|----------------|
| EV-01: SubScore includes Evidence field with top-5 worst offenders | ✓ SATISFIED | SubScore.Evidence field exists, extractors populate top-5 |
| EV-02: MetricExtractor returns evidence alongside score and raw value | ✓ SATISFIED | 3-return signature implemented across all extractors |
| EV-03: All 7 extractCx functions populate evidence | ✓ SATISFIED | C1/C3/C5/C6 populate real data, C2/C4/C7 return empty arrays as designed |
| EV-04: Evidence includes file path, line, value, description | ✓ SATISFIED | EvidenceItem has all 4 fields with json tags |
| EV-05: JSON output includes evidence with backward compatibility | ✓ SATISFIED | JSON version 2, evidence always present, old baseline loading tested |

### Anti-Patterns Found

None found. All code follows established patterns:
- No TODO/FIXME/HACK comments in modified files
- No placeholder implementations
- No stub returns (evidence is either populated or explicitly empty array)
- No console.log-only implementations
- Proper error handling with nil checks

### Human Verification Required

None. All success criteria can be verified programmatically and have been verified.

### Summary

Phase 30 goal **ACHIEVED**. Evidence data flows through the complete pipeline:

1. **Type foundation** (Plan 01): EvidenceItem type defined, SubScore.Evidence field added, MetricExtractor signature extended to 3-return, C7 overall_score removed
2. **Evidence extraction** (Plan 02): All 7 extractCx functions populate evidence maps with top-5 worst offenders for applicable metrics
3. **JSON output** (Plan 03): Evidence visible in JSON output via sub_scores field, version bumped to 2, backward compatible baseline loading

**Live verification:** Running `ars scan internal/analyzer --json` produces JSON output with:
- `"version": "2"`
- `"sub_scores"` arrays in each category
- `"evidence"` arrays with file_path, line, value, description fields
- Real worst-offender data (e.g., "pyDetectDeadCode has complexity 30")

All tests pass. No gaps found.

---

*Verified: 2026-02-06T21:30:00Z*
*Verifier: Claude (gsd-verifier)*
