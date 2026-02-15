# AGENTS.md

**AI Coding Agent Instructions for Agent Readiness Score (ARS)**

This file provides precise, executable instructions for AI coding agents working on this repository. For human contributors, see [CONTRIBUTING.md](CONTRIBUTING.md).

---

## Project Identity

**Name:** Agent Readiness Score (ARS)
**Type:** CLI tool for measuring codebase AI-readiness
**Language:** Go 1.21+
**Stack:** Go stdlib, Tree-sitter (Python/TypeScript parsing), go-charts (HTML reports)

---

## Quick Reference Commands

### Build & Test
```bash
# Build binary
go build -o ars ./cmd/ars

# Run all tests
go test ./... -v

# Run tests with coverage
go test ./... -coverprofile=cover.out

# Run specific package tests
go test ./internal/analyzer/c1_code_quality/... -v

# Run specific test
go test ./internal/analyzer/c1_code_quality -run TestComplexity -v

# Build all packages (check compilation)
go build ./...

# Tidy dependencies
go mod tidy

# Format code
gofmt -w .

# Run linter (if installed)
golangci-lint run ./...
```

### Running the Tool
```bash
# Scan current directory
go run . scan .

# Scan with JSON output (pipe normal output)
go run . scan . --json 2>/dev/null

# Generate HTML report
go run . scan . --output-html /tmp/test-report.html

# Enable C7 debug mode
go run . scan . --debug-c7

# Save C7 responses for replay
go run . scan . --debug-c7 --debug-dir ./debug-out

# Disable LLM features (auto-enabled when Claude CLI is detected)
go run . --no-llm

# Enable C7 agent evaluation (requires claude CLI)
go run . scan . --enable-c7
```

### Git Workflow
```bash
# Create feature branch
git checkout -b feat/your-feature-name

# Stage changes
git add <files>

# Commit with conventional format
git commit -m "feat(c1): add duplication detection"

# Push branch
git push origin feat/your-feature-name
```

---

## Code Style: The ARS Way

### Commit Message Format
**Always use Conventional Commits:**

```
<type>(<scope>): <subject>

Types: feat, fix, docs, test, refactor, perf, chore
Scopes: c1-c7 (categories), or phase numbers (26-01, 27-02)
```

**Examples:**
```
feat(c1): add cyclomatic complexity for Python
fix(scoring): correct piecewise interpolation edge case
docs(readme): add installation instructions
test(c6): add coverage metric fixtures
refactor(c3): extract common AST utilities
```

### Go Code Patterns

#### Pattern: Category Analyzer Structure
```go
// Each category has analyzer files: {language}.go
// Example: internal/analyzer/c1_code_quality/python.go

package c1_code_quality

import (
    "github.com/ingo-eichhorst/agent-readyness/pkg/types"
)

// analyzeComplexityPython computes cyclomatic complexity for Python
func analyzeComplexityPython(target *types.AnalysisTarget) (float64, []types.EvidenceItem) {
    total := 0.0
    evidence := make([]types.EvidenceItem, 0)

    for _, file := range target.SourceFiles {
        complexity := extractComplexityFromPython(file)
        total += complexity

        evidence = append(evidence, types.EvidenceItem{
            File:        file.Path,
            Line:        0,
            Value:       complexity,
            Description: fmt.Sprintf("Complexity: %.1f", complexity),
        })
    }

    avg := total / float64(len(target.SourceFiles))
    return avg, evidence
}
```

#### Pattern: Metric Extractor Signature (3-return)
```go
// ALL extractCx functions return (raw, score, evidence)
// Example from internal/scoring/scorer.go

func extractC1(results *types.AnalyzedResult) (float64, float64, []types.EvidenceItem) {
    // 1. Extract raw metric value
    raw := results.C1.Complexity

    // 2. Compute score using config breakpoints
    score := cfg.ScoreMetric("c1", "complexity", raw)

    // 3. Return evidence from analyzer
    evidence := results.C1.ComplexityEvidence

    return raw, score, evidence
}
```

#### Pattern: Table-Driven Tests
```go
func TestC1_Complexity(t *testing.T) {
    tests := []struct {
        name     string
        code     string
        expected float64
    }{
        {
            name:     "simple function",
            code:     "func Add(a, b int) int { return a + b }",
            expected: 1.0,
        },
        {
            name:     "conditional branch",
            code:     "func Max(a, b int) int { if a > b { return a } return b }",
            expected: 2.0,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := analyzeComplexity(tt.code)
            if result != tt.expected {
                t.Errorf("got %v, want %v", result, tt.expected)
            }
        })
    }
}
```

---

## Architecture Map

### Pipeline Flow (internal/pipeline/pipeline.go)
```
1. Discovery  → internal/discovery/        (walk files, classify, respect .gitignore)
2. Parse      → internal/parser/           (go/packages or Tree-sitter)
3. Analyze    → internal/analyzer/         (parallel C1-C7 execution)
4. Score      → internal/scoring/          (piecewise linear interpolation)
5. Recommend  → internal/recommend/        (generate improvement suggestions)
6. Output     → internal/output/           (terminal/JSON/HTML rendering)
```

### Category Structure
```
internal/analyzer/
├── c1_code_quality/          # Code health (complexity, length, coupling)
│   ├── go.go
│   ├── python.go
│   └── typescript.go
├── c2_semantics/             # Type annotations, naming
├── c3_architecture/          # Module structure, dependencies
├── c4_documentation/         # READMEs, comments, API docs
├── c5_temporal/              # Git history (churn, coupling)
├── c6_testing/               # Test ratio, coverage, isolation
└── c7_agent/                 # Live agent evaluation (Claude CLI)
```

### Key Types (pkg/types/types.go)
```go
type AnalysisTarget struct {
    Language    string
    SourceFiles []SourceFile
    TestFiles   []SourceFile
}

type SourceFile struct {
    Path     string
    Language string
    Content  string
}

type AnalyzedResult struct {
    C1 C1Metrics
    C2 C2Metrics
    // ... C3-C7
}

type EvidenceItem struct {
    File        string
    Line        int
    Value       float64
    Description string
}
```

---

## Always Do

### ✅ Before Committing
- Run `go test ./...` (all tests must pass)
- Run `go build ./...` (compilation must succeed)
- Run `gofmt -w .` (format all Go files)
- Verify commit message follows `type(scope): subject` format
- Check that evidence arrays are `[]` not `nil` in JSON output

### ✅ When Adding Metrics
- Update `internal/scoring/config.go` with breakpoints
- Add metric to `extractCx()` function in `internal/scoring/scorer.go`
- Return 3 values: `(raw float64, score float64, evidence []EvidenceItem)`
- Add evidence collection in analyzer
- Update `internal/output/descriptions.go` with metric description
- Add research citations to `internal/output/citations.go` if relevant

### ✅ When Adding Language Support
- Create `{language}.go` in each `internal/analyzer/c*/` package
- Implement Tree-sitter parsing in `internal/parser/treesitter.go`
- Add language detection in `internal/discovery/discovery.go`
- Update `types.Language` enum in `pkg/types/types.go`
- Add test fixtures in `testdata/` directories

### ✅ For Test Coverage
- Colocate tests with implementation (`*_test.go`)
- Use `testdata/` for fixtures
- Name test files clearly: `TestCategoryAnalyzer_MetricName`
- Document expected values in test comments
- Validate both structure AND array behavior (`[]` vs `null`)

---

## Ask First

### ⚠️ Before Making These Changes
- Changing scoring algorithm weights or thresholds
- Adding new required CLI flags or changing defaults
- Modifying JSON output schema (requires version bump)
- Adding external dependencies (discuss tradeoffs)
- Changing HTML report template structure significantly
- Removing or renaming public API functions
- Adding new LLM-based features (cost implications)

### ⚠️ Architectural Decisions
- New category beyond C1-C7
- Alternative scoring models
- Different output formats
- Integration with external tools
- Changes to Git-based analysis approach (C5)

---

## Never Touch

### 🚫 Forbidden Changes
- **`.planning/` directory** — Project planning artifacts, managed by GSD workflow
- **`.git/` directory** — Git internals
- **`vendor/` directory** — Vendored dependencies (if present)
- **Production credentials** — Never commit API keys, tokens, or secrets
- **Scoring config weights** without discussion — Changes affect all users
- **Citation URLs** without verification — All citations must be stable (DOI/ArXiv/publisher)
- **Test fixtures in testdata/c7_responses/** — Real Claude responses, do not modify

### 🚫 Anti-Patterns to Avoid
- Single-letter variable names (except loop counters `i`, `j`)
- Magic numbers without constants or comments
- Functions longer than 100 lines
- Nested conditionals deeper than 3 levels
- Global mutable state
- Panics in library code (return errors)
- `fmt.Print` statements (use `debugWriter` or proper logging)
- Modifying `go.mod` without `go mod tidy`

---

## Common Tasks: Step-by-Step

### Add a New Metric to C1 (Code Quality)

1. **Add scoring config** (`internal/scoring/config.go`):
   ```go
   "duplication_ratio": {
       {Raw: 0.0, Score: 10.0},
       {Raw: 0.05, Score: 8.0},
       {Raw: 0.15, Score: 5.0},
       {Raw: 0.30, Score: 1.0},
   },
   ```

2. **Update C1Metrics struct** (`pkg/types/types.go`):
   ```go
   type C1Metrics struct {
       // ... existing fields
       DuplicationRatio    float64          `json:"duplication_ratio"`
       DuplicationEvidence []EvidenceItem   `json:"duplication_evidence,omitempty"`
   }
   ```

3. **Implement analyzer** (`internal/analyzer/c1_code_quality/go.go`):
   ```go
   func (a *Analyzer) analyzeDuplication(target *types.AnalysisTarget) (float64, []types.EvidenceItem) {
       // Implementation here
       ratio := computeDuplicationRatio(target.SourceFiles)
       evidence := collectDuplicationEvidence(target.SourceFiles)
       return ratio, evidence
   }
   ```

4. **Wire into Analyze()** (`internal/analyzer/c1_code_quality/go.go`):
   ```go
   func (a *Analyzer) Analyze(target *types.AnalysisTarget) types.C1Metrics {
       metrics := types.C1Metrics{}
       // ... existing metrics
       metrics.DuplicationRatio, metrics.DuplicationEvidence = a.analyzeDuplication(target)
       return metrics
   }
   ```

5. **Update extractC1** (`internal/scoring/scorer.go`):
   ```go
   func extractC1(results *types.AnalyzedResult) (float64, float64, []types.EvidenceItem) {
       // Add new metric extraction
       dupRaw := results.C1.DuplicationRatio
       dupScore := cfg.ScoreMetric("c1", "duplication_ratio", dupRaw)
       evidence = append(evidence, results.C1.DuplicationEvidence...)
       // Update composite calculation
   }
   ```

6. **Add description** (`internal/output/descriptions.go`):
   ```go
   "duplication_ratio": {
       Brief: "Code duplication percentage",
       Detailed: "Measures token-level duplication...",
   },
   ```

7. **Add tests** (`internal/analyzer/c1_code_quality/go_test.go`):
   ```go
   func TestAnalyzeDuplication(t *testing.T) {
       // Test with fixtures
   }
   ```

### Fix a Bug in HTML Rendering

1. **Reproduce** — Create minimal test case:
   ```bash
   go run . scan . --output-html /tmp/bug-test.html
   # Open in browser, identify issue
   ```

2. **Locate** — HTML rendering in `internal/output/html.go`

3. **Fix** — Modify template or helper functions

4. **Test** — Add test in `internal/output/html_test.go`:
   ```go
   func TestRenderHTML_YourBugFix(t *testing.T) {
       // Test case
   }
   ```

5. **Verify** — Run tests and regenerate HTML:
   ```bash
   go test ./internal/output/... -v
   go run . scan . --output-html /tmp/fixed.html
   ```

---

## File Paths Quick Reference

```
Key files to know:
  cmd/scan.go                           — CLI entry point
  internal/pipeline/pipeline.go         — Orchestration
  internal/analyzer/{c1-c7}/            — Category analyzers
  internal/scoring/scorer.go            — Metric extraction
  internal/scoring/config.go            — Scoring thresholds
  internal/output/terminal.go           — Terminal rendering
  internal/output/html.go               — HTML generation
  internal/output/json.go               — JSON output
  internal/output/descriptions.go       — Metric descriptions
  internal/output/citations.go          — Research citations
  pkg/types/types.go                    — Core data structures
  testdata/                             — Test fixtures
```

---

## Output Behavior

**Terminal (default):**
- Colored output with ANSI codes
- Category scores with sub-metrics
- Recommendations ranked by impact

**JSON (`--json`):**
- Machine-readable structured output
- Schema version field for compatibility
- Zero-weight metrics filtered out (e.g., deprecated `overall_score`)

**HTML (`--output-html`):**
- Self-contained single file
- Embedded CSS and JavaScript
- Radar chart visualization
- Expandable metric descriptions with research citations

---

## Special Notes

### Evidence Arrays
- **Must be `[]` not `nil`** for JSON compatibility
- Convert: `evidence := make([]types.EvidenceItem, 0)` before returning
- Empty arrays serialize as `[]`, nil serializes as `null`

### Zero-Weight Metrics
- Some metrics have `Weight: 0.0` in scoring config (backward compatibility)
- Renderers must filter: `if ss.Weight == 0.0 { continue }`
- Example: C7's `overall_score` deprecated in favor of M1-M5

### HTML Report Testing
- Full repo scans are slow (>30s)
- For quick HTML testing: `go run . scan internal/analyzer --output-html /tmp/test.html`
- Test with small, focused directory

### C7 Debug Mode
- `--debug-c7` writes to stderr (stdout unchanged)
- `--debug-dir ./debug-out` saves responses to JSON
- Replay avoids Claude CLI on subsequent runs (fast iteration)
- Response fixtures in `testdata/c7_responses/` are READ-ONLY

---

## Where to Get Help

1. **Architecture details:** Read `CLAUDE.md`
2. **Project roadmap:** See `.planning/PROJECT.md` and `.planning/ROADMAP.md`
3. **Phase plans:** Check `.planning/phases/{number}/` for context
4. **Research citations:** See `docs/CITATION-GUIDE.md`
5. **Human contributors:** Ask in GitHub Issues or reference `CONTRIBUTING.md`

---

**Last Updated:** 2026-02-06
**For:** AI Coding Agents (Claude, Copilot, Cursor, Windsurf, etc.)
**Companion File:** [CONTRIBUTING.md](CONTRIBUTING.md) (for humans)
