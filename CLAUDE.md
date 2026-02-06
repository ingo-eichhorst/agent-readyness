# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Agent Readiness Score (ARS) is a CLI tool that measures how ready a codebase is for AI agents. It analyzes code across 7 categories (C1-C7) and produces a composite score with tier classification (Agent-Ready, Agent-Assisted, Agent-Limited, Agent-Hostile).

**Supported languages:** Go, Python, TypeScript (auto-detected from project files)

## Build & Test Commands

```bash
# Build
go build -o ars .

# Run scan
go run . scan <directory>

# Run all tests
go test ./...

# Run tests with coverage
go test ./... -coverprofile=cover.out

# Run single package tests
go test ./internal/analyzer/...

# Run single test
go test ./internal/analyzer -run TestC1Analyzer

# Debug C7 metrics with verbose output
go run . scan . --debug-c7

# Save C7 responses for replay (avoids Claude CLI on subsequent runs)
go run . scan . --debug-c7 --debug-dir ./debug-out

# Tidy modules
go mod tidy
```

## Architecture

### Pipeline Flow

The scan executes in stages orchestrated by `internal/pipeline/pipeline.go`:

1. **Discovery** (`internal/discovery/`) - Walk filesystem, classify files (source/test/generated/excluded), respect .gitignore
2. **Parse** (`internal/parser/`) - Go uses `go/packages`, Python/TypeScript use Tree-sitter
3. **Analyze** (`internal/analyzer/`) - Run C1-C7 analyzers in parallel on `AnalysisTarget` structs
4. **Score** (`internal/scoring/`) - Convert raw metrics to 1-10 scores via piecewise linear interpolation
5. **Recommend** (`internal/recommend/`) - Generate improvement suggestions based on scores
6. **Output** (`internal/output/`) - Render to terminal, JSON, or HTML

### Category Analyzers

Each category has its own package in `internal/analyzer/`:

| Category | Package | Measures |
|----------|---------|----------|
| C1 | `c1_code_quality/` | Complexity, function length, file size, coupling, duplication |
| C2 | `c2_semantics/` | Type annotations, naming consistency, magic numbers |
| C3 | `c3_architecture/` | Directory depth, module fanout, circular deps, dead exports |
| C4 | `c4_documentation/` | README, comments, API docs, examples (optional LLM analysis) |
| C5 | `c5_temporal/` | Git-based: churn rate, temporal coupling, hotspots |
| C6 | `c6_testing/` | Test ratio, coverage, isolation, assertions |
| C7 | `c7_agent/` | Live agent evaluation via Claude CLI (optional) |

Language-specific implementations follow the pattern `{language}.go` within each package (e.g., `c1_code_quality/python.go`, `c2_semantics/typescript.go`).

### Key Types

- `types.AnalysisTarget` - Language-agnostic analysis unit containing files for one language
- `types.SourceFile` - Single file with path, language, classification, and content
- `types.ScoredResult` - Final output with composite score, tier, and per-category breakdowns
- `scoring.ScoringConfig` - Configurable breakpoints and weights for scoring
- `types.EvidenceItem` - Proof of metric findings (file path, line, value, description)

**MetricExtractor signature:** All extractCx functions return 3 values: `(raw float64, score float64, evidence []EvidenceItem)`

### Scoring System

Scoring uses piecewise linear interpolation defined in `internal/scoring/config.go`:
- Each metric has breakpoints mapping raw values to scores (1-10)
- Categories have configurable weights (sum to 1.0)
- Composite = weighted average of category scores
- Tiers: Agent-Ready (≥8), Agent-Assisted (≥6), Agent-Limited (≥4), Agent-Hostile (<4)

### Multi-Language Support

Go analysis uses `go/packages` for type-checked AST. Python and TypeScript use Tree-sitter (`internal/parser/treesitter.go`). The `GoAwareAnalyzer` interface allows analyzers to receive parsed Go packages separately from the generic `AnalysisTarget` flow.

### Configuration

- Project config: `.arsrc.yml` or `.arsrc.yaml` - override weights, thresholds per-project
- Scoring config: `internal/scoring/config.go` DefaultConfig() - breakpoints for all metrics

## Testing Patterns

Test files are colocated with implementation (`*_test.go`). Use `testdata/` directories for fixture projects:
- `testdata/valid-go-project/` - Standard Go project for discovery tests
- `testdata/complexity/` - High-complexity code for C1 tests
- `testdata/polyglot-project/` - Multi-language project
- `testdata/c7_responses/` - Real Claude responses for C7 metric heuristic tests

When adding test fixtures: validate both structure AND empty array behavior (`[]` vs `null` in JSON assertions)

## HTML Report System

HTML output uses Go embedded templates (`//go:embed`) in `internal/output/templates/`:
- `report.html` - Main template with expand/collapse JavaScript
- `styles.css` - All styling (inlined at render time)

Key helper functions in `internal/output/html.go` that must be updated when adding categories/metrics:
- `categoryImpact()` - Short description per category
- `categoryDisplayName()` - Display name mapping
- `metricDisplayName()` - Per-metric display names

Per-metric content:
- `internal/output/descriptions.go` - Brief/detailed descriptions with `MetricDescription` structs
- `internal/output/citations.go` - Research citations filtered by category

## Gotchas

- Zero-weight metrics in `internal/scoring/config.go` (e.g., C7 `overall_score`) are deprecated but kept for backward compatibility. Output renderers must filter `ss.Weight == 0.0`.
- Full repo scans are slow (>30s). For quick HTML testing: `./ars scan internal/analyzer --output-html /tmp/test.html`
- Evidence arrays must be `[]` not `null` in JSON - convert `nil` to `make([]EvidenceItem, 0)` before returning
- JSON schema version bumps (internal/output/json.go) required for breaking changes to output format

## Optional Features

- C4 LLM analysis - Auto-enabled when Claude CLI is detected; use `--no-llm` to disable
- `--enable-c7` - Live agent evaluation using Claude CLI (requires `claude` installed)
- `--output-html` - Generate self-contained HTML report with charts
- `--baseline` - Compare against previous JSON output for trend analysis
- `--no-llm` - Disable LLM features even when Claude CLI is available

## Development Workflow

Project uses GSD (Get Stuff Done) methodology with phase-based development:
- `.planning/PROJECT.md` - Project overview and validated requirements
- `.planning/phases/*/` - Phase plans, research, and verification docs
- `.planning/ROADMAP.md` - Current milestone with phase breakdown
