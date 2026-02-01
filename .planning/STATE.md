# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-02-01)

**Core value:** Accurate, evidence-based scoring that predicts agent success and identifies specific improvements teams should make before adopting AI coding agents.
**Current focus:** Defining requirements for v2 milestone

## Current Position

Phase: Not started (defining requirements)
Plan: —
Status: Defining requirements for v2 milestone
Last activity: 2026-02-01 -- Milestone v2 started

Progress: [                              ] 0%

## Performance Metrics

**Velocity:**
- Total plans completed: 16
- Average duration: 5 min
- Total execution time: 82 min

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| 01-foundation | 3 | 9 min | 3 min |
| 02-core-analysis | 5 | 44 min | 9 min |
| 03-scoring-model | 3 | 10 min | 3 min |
| 04-recommendations-and-output | 3 | 14 min | 5 min |
| 05-hardening | 2 | 5 min | 3 min |

**Recent Trend:**
- Last 5 plans: 04-01 (4 min), 04-02 (2 min), 04-03 (8 min), 05-01 (2 min), 05-02 (3 min)
- Trend: Consistent fast execution through hardening phase

*Updated after each plan completion*

## Accumulated Context

### Decisions

Decisions are logged in PROJECT.md Key Decisions table.
Recent decisions affecting current work:

- [Roadmap]: 5-phase structure following Foundation -> Analysis -> Scoring -> Output -> Hardening dependency chain
- [Roadmap]: Use go/packages from day one for type-aware parsing (research pitfall #1)
- [Roadmap]: Edge cases and performance optimization deferred to Phase 5 (requires full tool first)
- [01-01]: Cobra CLI with root + scan subcommand pattern
- [01-01]: Shared types in pkg/types for cross-package use
- [01-01]: Version set via ldflags (default 'dev')
- [01-01]: Go project detection: go.mod first, fallback to .go file scan
- [01-02]: Vendor dirs walked (not SkipDir) so files recorded as ClassExcluded with reason
- [01-02]: Generated file detection stops at package declaration
- [01-02]: Root-level .gitignore only in Phase 1
- [01-02]: Package-level compiled regex for generated file pattern
- [01-03]: Pipeline uses interface-based stages (Parser, Analyzer) for Phase 2 plug-in
- [01-03]: fatih/color auto-disables ANSI when not a TTY
- [01-03]: Output rendering separated from pipeline logic in internal/output
- [02-01]: NeedForTest flag required for go/packages test package identification
- [02-01]: ParsedPackage as new type in internal/parser (not evolution of ParsedFile)
- [02-01]: Parser.Parse takes rootDir string, not []DiscoveredFile
- [02-02]: gocyclo complexity matched via fset position key to merge with function length data
- [02-02]: AST statement-sequence FNV hashing for duplication detection
- [02-02]: Stub C3/C6 analyzer types added to unblock pre-existing test files
- [02-03]: Dead code detection uses go/types scope + cross-package Uses map
- [02-03]: Single-package modules skip dead code detection (avoids false positives)
- [02-03]: filterSourcePackages utility filters test packages for all C3 metrics
- [02-04]: Coverage search order: cover.out -> lcov.info/coverage.lcov -> cobertura.xml/coverage.xml
- [02-04]: Test isolation uses file-level imports not function-level
- [02-04]: Assertion density counts both std testing and testify selector expressions
- [02-05]: Analyzer errors logged as warnings, do not abort pipeline
- [02-05]: Color thresholds: complexity avg >10 yellow, >20 red; similar bands for other metrics
- [02-05]: Verbose mode shows top-5 lists for complexity and function length
- [03-01]: Breakpoints sorted by Value ascending; Score direction encodes lower/higher-is-better
- [03-01]: Composite normalizes by sum of active weights (0.60), not 1.0
- [03-01]: Tier boundaries use >= semantics (8.0 is Agent-Ready)
- [03-01]: categoryScore returns 5.0 (neutral) when no sub-scores available
- [03-02]: scoreMetrics generic helper avoids code duplication across scoreC1/C3/C6
- [03-02]: Unavailable metrics passed as map[string]bool to scoreMetrics rather than sentinel values
- [03-02]: Config metric names used as raw value map keys (complexity_avg not cyclomatic_complexity_avg)
- [03-03]: LoadConfig unmarshals YAML into DefaultConfig copy so missing fields keep defaults
- [03-03]: Scoring errors produce warnings, do not crash pipeline
- [03-03]: RenderScores is separate function from RenderSummary
- [03-03]: Score color thresholds: green >= 8.0, yellow >= 6.0, red < 6.0
- [04-01]: findTargetBreakpoint selects minimal next-better breakpoint (smallest improvement step)
- [04-01]: Hard metrics (complexity_avg, duplication_rate) get +1 effort level bump
- [04-01]: Effort thresholds: gap < 1.0 = Low, < 2.5 = Medium, >= 2.5 = High
- [04-01]: simulateComposite deep-copies categories to avoid mutation of input ScoredResult
- [04-02]: Impact color thresholds: green >= 0.5, yellow >= 0.2, red < 0.2 composite points
- [04-02]: JSON version field "1" for future schema evolution
- [04-02]: JSONMetric omitempty controls verbose metric inclusion in JSON
- [04-02]: RenderJSON uses json.NewEncoder for streaming to io.Writer
- [04-03]: ExitError in pkg/types to avoid cmd<->pipeline import cycle
- [04-03]: Threshold check AFTER rendering so output always displayed before exit
- [04-03]: SilenceUsage on scan command prevents usage dump on ExitError
- [04-03]: SilenceErrors on root command prevents Cobra double-printing
- [05-01]: Walker warnings go to os.Stderr to preserve JSON stdout output
- [05-01]: Symlink detection uses d.Type()&fs.ModeSymlink before IsDir check
- [05-01]: Error recovery returns fs.SkipDir for directory errors, nil for file errors
- [05-02]: Spinner writes to os.Stderr only, preventing --json stdout corruption
- [05-02]: TTY detection via go-isatty gates all spinner output (suppressed in CI/pipes)
- [05-02]: Parallel analyzer errors return nil to avoid aborting sibling analyzers
- [05-02]: Results sorted by Category string for deterministic C1/C3/C6 ordering

### Pending Todos

None.

### Blockers/Concerns

None.

## Session Continuity

Last session: 2026-02-01T10:49:31Z
Stopped at: Completed 05-02-PLAN.md (Phase 5 complete - all phases done)
Resume file: None
