---
phase: 09-c4-documentation-quality-html-reports
plan: 03
subsystem: output
tags: [html-reports, radar-charts, go-charts, citations, visualization]

# Dependency graph
requires:
  - phase: 09-01
    provides: C4Analyzer with static documentation metrics
  - phase: 09-02
    provides: LLM client for optional content evaluation
provides:
  - HTML report generator with radar charts and metric breakdowns
  - Research citations for all metric categories
  - --output-html CLI flag for report generation
  - --baseline flag for trend comparison
affects: [future-ci-integration, future-dashboards]

# Tech tracking
tech-stack:
  added: [go-charts-v2]
  patterns: [embed-fs-templates, html-template-escaping, svg-chart-generation]

key-files:
  created:
    - internal/output/charts.go
    - internal/output/charts_test.go
    - internal/output/html.go
    - internal/output/html_test.go
    - internal/output/citations.go
    - internal/output/templates/report.html
    - internal/output/templates/styles.css
  modified:
    - cmd/scan.go
    - internal/pipeline/pipeline.go
    - pkg/types/scoring.go
    - go.mod
    - go.sum

key-decisions:
  - "go-charts/v2 for radar chart SVG generation (no external dependencies)"
  - "Embedded templates via embed.FS for self-contained binary"
  - "html/template for XSS-safe rendering of user content"
  - "Radar chart requires minimum 3 categories (go-charts library constraint)"
  - "Professional CSS design with score-based color coding"
  - "Research citations hardcoded (not fetched dynamically)"

patterns-established:
  - "HTMLGenerator with embedded templates and template.HTML for trusted SVG"
  - "Baseline loading from previous JSON output for trend comparison"
  - "Score-to-class mapping for CSS styling (ready/assisted/limited)"

# Metrics
duration: 8min
completed: 2026-02-03
---

# Phase 09 Plan 03: HTML Report Generation Summary

**Self-contained HTML report generator with radar charts, metric breakdowns, research citations, and baseline trend comparison using go-charts/v2 for SVG generation**

## Performance

- **Duration:** 8 min
- **Started:** 2026-02-03T13:57:30Z
- **Completed:** 2026-02-03T14:05:34Z
- **Tasks:** 3
- **Files created:** 7
- **Files modified:** 5

## Accomplishments

- Added go-charts/v2 dependency for radar and line chart SVG generation
- Created embedded HTML templates with professional CSS styling
- Implemented XSS-safe HTML generator using html/template
- Added research citations for all six metric categories (C1-C6)
- Wired --output-html and --baseline CLI flags
- HTML reports render correctly offline with no external dependencies

## Task Commits

Each task was committed atomically:

1. **Task 1: Add go-charts dependency and create chart generation** - `d0acfe3` (feat)
   - go.mod/go.sum with go-charts/v2 dependency
   - internal/output/charts.go with generateRadarChart and generateTrendChart
   - internal/output/charts_test.go with comprehensive tests

2. **Task 2: Create HTML templates and generator** - `f8f9398` (feat)
   - internal/output/templates/report.html - main report template
   - internal/output/templates/styles.css - professional CSS design
   - internal/output/citations.go - research citations for C1-C6
   - internal/output/html.go - HTMLGenerator with embedded templates
   - internal/output/html_test.go - XSS prevention and structure tests
   - pkg/types/scoring.go - added ProjectName field to ScoredResult
   - internal/pipeline/pipeline.go - set ProjectName from directory basename

3. **Task 3: Wire CLI flags and integrate with pipeline** - `62abd52` (feat)
   - cmd/scan.go - added --output-html and --baseline flags
   - internal/pipeline/pipeline.go - SetHTMLOutput, generateHTMLReport, loadBaseline

## Files Created/Modified

### Created
- `internal/output/charts.go` - Radar and trend chart SVG generation using go-charts
- `internal/output/charts_test.go` - Chart generation tests including edge cases
- `internal/output/html.go` - HTMLGenerator with template.HTML for safe SVG embedding
- `internal/output/html_test.go` - 15 tests including XSS prevention verification
- `internal/output/citations.go` - 12 research citations covering all categories
- `internal/output/templates/report.html` - Semantic HTML template with radar chart placeholder
- `internal/output/templates/styles.css` - CSS custom properties, responsive design, print styles

### Modified
- `cmd/scan.go` - Added outputHTML and baselinePath flags, configured pipeline
- `internal/pipeline/pipeline.go` - HTML generation support with baseline loading
- `pkg/types/scoring.go` - Added ProjectName field for report title
- `go.mod` / `go.sum` - Added github.com/vicanso/go-charts/v2

## Decisions Made

1. **go-charts/v2 for visualization** - Pure Go library that generates SVG without external dependencies. Radar charts require minimum 3 indicators (categories), which we handle gracefully by returning empty SVG for fewer.

2. **Embedded templates via embed.FS** - Templates are compiled into the binary, ensuring reports work without additional files. This follows Go best practices for asset embedding.

3. **html/template for XSS protection** - All user content (project names, metric values) goes through template escaping. Only trusted SVG (generated by us) uses template.HTML bypass.

4. **Professional CSS design** - Score-based color coding (green >= 8, yellow >= 6, red < 6), system fonts, responsive layout, print-friendly styles. Avoids external stylesheets or JavaScript.

5. **Research citations hardcoded** - Citations are compiled into the binary rather than fetched dynamically. This ensures offline operation and stable references.

6. **Baseline from JSON output** - The --baseline flag accepts a previous JSON report from `--json` output. We parse the JSON and extract category scores for trend comparison.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Radar chart minimum indicators requirement**
- **Found during:** Task 2 testing
- **Issue:** go-charts RadarRender requires at least 3 indicators, test data only had 2 categories
- **Fix:** Updated generateRadarChart to return empty SVG for <3 categories; updated tests to use 3+ categories
- **Files modified:** internal/output/charts.go, internal/output/charts_test.go, internal/output/html_test.go
- **Committed in:** f8f9398

---

**Total deviations:** 1 auto-fixed (blocking issue with library constraint)
**Impact on plan:** Minor test fixture adjustment, no scope change.

## Verification Results

All verification criteria passed:

1. `go build ./...` - PASS
2. `go test ./...` - PASS (all 12 packages)
3. `go run . scan . --output-html report.html` - PASS (produces valid HTML)
4. Radar chart SVG in HTML - PASS (verified `<svg` tag presence)
5. HTML is self-contained - PASS (inline CSS, no external scripts)
6. With --baseline: trend chart appears - PASS (Score Comparison section)
7. XSS test passes - PASS (script tags escaped to `&lt;script&gt;`)

## Usage Examples

```bash
# Generate HTML report
ars scan ./my-project --output-html report.html

# Generate HTML with trend comparison
ars scan ./my-project --json > baseline.json
# ... make changes ...
ars scan ./my-project --output-html report.html --baseline baseline.json

# Generate both JSON and HTML
ars scan ./my-project --json > results.json && \
ars scan ./my-project --output-html report.html
```

## Next Phase Readiness

**Blockers:** None

**Phase 09 Complete:**
- 09-01: C4 static documentation metrics
- 09-02: LLM client and content evaluation
- 09-03: HTML report generation

**Ready for Phase 10:**
- All C1-C6 analyzers operational
- All output formats available (terminal, JSON, HTML)
- Scoring system complete with all categories
- Ready for CI/CD integration features

---
*Phase: 09-c4-documentation-quality-html-reports*
*Plan: 03*
*Completed: 2026-02-03*
