# Architecture Research

**Domain:** Go CLI static analysis tool
**Researched:** 2026-01-31
**Confidence:** HIGH

## Standard Architecture

### System Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                          CLI Layer                               │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐                       │
│  │  cobra    │  │  config  │  │  flags   │                       │
│  │  commands │  │  loader  │  │  parser  │                       │
│  └─────┬────┘  └─────┬────┘  └─────┬────┘                       │
│        └──────────────┼─────────────┘                            │
├───────────────────────┼─────────────────────────────────────────┤
│                   Orchestrator                                   │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │              Scanner / Pipeline Runner                     │   │
│  │   (discovers files, dispatches to analyzers, collects)    │   │
│  └─────────────────────────┬────────────────────────────────┘   │
├────────────────────────────┼────────────────────────────────────┤
│                      Analysis Layer                              │
│  ┌─────────────┐  ┌───────────────┐  ┌──────────────────┐      │
│  │ C1: Code    │  │ C3: Arch      │  │ C6: Testing      │      │
│  │ Health      │  │ Navigability  │  │ Infrastructure   │      │
│  │ Analyzer    │  │ Analyzer      │  │ Analyzer         │      │
│  └──────┬──────┘  └───────┬───────┘  └────────┬─────────┘      │
│         └─────────────────┼────────────────────┘                │
├────────────────────────────┼────────────────────────────────────┤
│                      Parsing Layer                               │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │   go/parser + go/ast + go/token                           │   │
│  │   (parse files into ASTs, provide token positions)        │   │
│  └──────────────────────────────────────────────────────────┘   │
├─────────────────────────────────────────────────────────────────┤
│                      Scoring Layer                               │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────┐      │
│  │ Per-Category  │  │ Composite    │  │ Recommendations  │      │
│  │ Scorer       │  │ Scorer       │  │ Generator        │      │
│  └──────────────┘  └──────────────┘  └──────────────────┘      │
├─────────────────────────────────────────────────────────────────┤
│                      Output Layer                                │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │   Terminal Renderer (text output, tier badge, colors)     │   │
│  └──────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

### Component Responsibilities

| Component | Responsibility | Typical Implementation |
|-----------|----------------|------------------------|
| CLI Layer | Parse flags, validate input, wire dependencies | Cobra commands in `cmd/` |
| Config Loader | Load config from flags (and optionally file) | Simple struct, no Viper needed for v1 |
| Scanner | Discover Go files, coordinate analysis pipeline | Walk directory, filter `.go` files, dispatch |
| Parsing Layer | Parse Go source into ASTs | `go/parser.ParseFile` with `token.FileSet` |
| C1 Analyzer | Cyclomatic complexity, function length, file size, coupling | AST traversal via `ast.Inspect` |
| C3 Analyzer | Directory depth, module fanout, circular deps, import graph | File path analysis + import extraction from AST |
| C6 Analyzer | Test coverage, test-to-code ratio, test isolation, assertions | `_test.go` detection, AST inspection for test funcs |
| Per-Category Scorer | Convert raw metrics into 1-10 score per category | Threshold tables, linear interpolation |
| Composite Scorer | Weighted average of category scores | C1: 25%, C3: 20%, C6: 15% (from PROJECT.md) |
| Recommendations Generator | Identify top 5 improvements by impact | Sort metrics by distance-from-ideal, format suggestions |
| Terminal Renderer | Format and print results to stdout | `fmt` or `text/tabwriter`, ANSI colors |

## Recommended Project Structure

```
ars/
├── main.go                    # Entry point, calls cmd.Execute()
├── cmd/
│   ├── root.go                # Root cobra command, global flags
│   └── scan.go                # `ars scan <dir>` command
├── internal/
│   ├── scanner/
│   │   └── scanner.go         # File discovery, pipeline orchestration
│   ├── parser/
│   │   └── parser.go          # Thin wrapper around go/parser
│   ├── analyzer/
│   │   ├── analyzer.go        # Analyzer interface definition
│   │   ├── codehealth.go      # C1: complexity, function length, etc.
│   │   ├── architecture.go    # C3: directory depth, imports, etc.
│   │   └── testing.go         # C6: test coverage, ratios, etc.
│   ├── scorer/
│   │   ├── category.go        # Per-category scoring logic
│   │   ├── composite.go       # Weighted composite score
│   │   └── thresholds.go      # Score threshold definitions
│   ├── recommend/
│   │   └── recommend.go       # Top-5 improvement generator
│   └── output/
│       └── terminal.go        # Terminal text renderer
├── pkg/
│   └── types/
│       └── types.go           # Shared types: FileMetrics, Score, etc.
└── testdata/                  # Sample Go files for testing analyzers
    ├── simple/
    ├── complex/
    └── large/
```

### Structure Rationale

- **`cmd/`:** Thin command wrappers. Only CLI concerns (flags, arg validation, wiring). No business logic.
- **`internal/`:** All business logic. Not importable by external packages, enforcing encapsulation.
- **`internal/scanner/`:** Owns file discovery and orchestration. This is the pipeline driver.
- **`internal/analyzer/`:** Each analyzer is a separate file implementing a shared interface. Easy to add new categories later (C2, C4, C5).
- **`internal/scorer/`:** Separate from analysis. Analyzers produce raw metrics; scorers convert metrics to scores. This separation makes threshold tuning independent of metric collection.
- **`internal/recommend/`:** Separate from scoring. Takes scored results, identifies highest-impact improvements.
- **`internal/output/`:** Renderer is isolated so adding JSON/HTML output later is a new file, not a rewrite.
- **`pkg/types/`:** Shared structs used across packages. Kept minimal to avoid circular imports.
- **`testdata/`:** Real Go source files for testing. Analyzers are tested against known code with expected metrics.

## Architectural Patterns

### Pattern 1: Pipeline Architecture

**What:** Data flows through a series of stages: Discover -> Parse -> Analyze -> Score -> Recommend -> Render. Each stage has a clear input/output contract.

**When to use:** Always. This is the core pattern for static analysis tools. Golangci-lint uses exactly this pattern (Init -> Load Packages -> Run Linters -> Postprocess Issues -> Print Issues).

**Trade-offs:**
- Pro: Each stage is independently testable
- Pro: Easy to add new stages or swap implementations
- Pro: Natural parallelism boundaries (parse files concurrently, analyze concurrently)
- Con: Slightly more boilerplate than a monolithic approach

**Example:**
```go
// Pipeline stages with clear boundaries
type Pipeline struct {
    scanner  *scanner.Scanner
    parser   *parser.Parser
    analyzers []analyzer.Analyzer
    scorer   *scorer.Composite
    recommender *recommend.Recommender
    renderer *output.Terminal
}

func (p *Pipeline) Run(dir string) (*types.Report, error) {
    // Stage 1: Discover files
    files, err := p.scanner.Discover(dir)
    if err != nil {
        return nil, fmt.Errorf("scan: %w", err)
    }

    // Stage 2: Parse files into ASTs
    parsed, err := p.parser.ParseAll(files)
    if err != nil {
        return nil, fmt.Errorf("parse: %w", err)
    }

    // Stage 3: Run analyzers
    metrics := make(map[string]*types.CategoryMetrics)
    for _, a := range p.analyzers {
        m, err := a.Analyze(parsed)
        if err != nil {
            return nil, fmt.Errorf("analyze %s: %w", a.Name(), err)
        }
        metrics[a.Name()] = m
    }

    // Stage 4: Score
    scores := p.scorer.Score(metrics)

    // Stage 5: Recommend improvements
    recs := p.recommender.Top(scores, 5)

    return &types.Report{Scores: scores, Recommendations: recs}, nil
}
```

**Confidence:** HIGH -- This is the universal pattern for static analysis tools, confirmed by golangci-lint's architecture and every Go analysis tool surveyed.

### Pattern 2: Analyzer Interface

**What:** A common interface that all analyzers implement. Each analyzer receives parsed files and returns structured metrics.

**When to use:** When you have multiple analysis categories (C1, C3, C6) that need to run on the same parsed data.

**Trade-offs:**
- Pro: Adding a new category means adding one file implementing the interface
- Pro: Each analyzer is independently testable
- Pro: Analyzers can run concurrently since they only read parsed data
- Con: Interface must be general enough to accommodate different analysis types

**Example:**
```go
// analyzer/analyzer.go
type Analyzer interface {
    Name() string
    Analyze(files []*ParsedFile) (*types.CategoryMetrics, error)
}

// analyzer/codehealth.go
type CodeHealth struct{}

func (c *CodeHealth) Name() string { return "code_health" }

func (c *CodeHealth) Analyze(files []*ParsedFile) (*types.CategoryMetrics, error) {
    metrics := &types.CategoryMetrics{}
    for _, f := range files {
        ast.Inspect(f.AST, func(n ast.Node) bool {
            switch node := n.(type) {
            case *ast.FuncDecl:
                metrics.AddFunction(functionMetrics(f.Fset, node))
            }
            return true
        })
    }
    return metrics, nil
}
```

**Confidence:** HIGH -- This is the standard Go interface pattern, directly modeled on how `golang.org/x/tools/go/analysis` structures its `Analyzer` type.

### Pattern 3: Metric Collection then Scoring (Two-Phase)

**What:** Separate raw metric collection from score computation. Analyzers produce numbers (cyclomatic complexity = 12, function length = 45 lines). Scorers convert those numbers into 1-10 scores using configurable thresholds.

**When to use:** Always. Mixing metric collection with scoring makes threshold tuning painful and testing harder.

**Trade-offs:**
- Pro: Can tune scoring thresholds without changing analysis code
- Pro: Can validate metric collection independently of scoring logic
- Pro: Thresholds become configuration, not code
- Con: Extra data structure for intermediate metrics

**Example:**
```go
// Raw metrics from analyzer (no opinion about good/bad)
type FunctionMetrics struct {
    Name            string
    CyclomaticComplexity int
    LineCount       int
    ParameterCount  int
}

// Scorer converts to 1-10 using thresholds
func ScoreCyclomaticComplexity(avg float64) float64 {
    // Thresholds based on Go community norms:
    // <= 5: perfect (10), 5-10: good (7-9), 10-15: moderate (4-6), >15: poor (1-3)
    switch {
    case avg <= 5:
        return 10.0
    case avg <= 10:
        return 7.0 + (10.0-avg)/5.0*3.0
    case avg <= 15:
        return 4.0 + (15.0-avg)/5.0*3.0
    default:
        return max(1.0, 4.0-(avg-15.0)/10.0*3.0)
    }
}
```

**Confidence:** HIGH -- Separation of measurement from judgment is a fundamental static analysis design principle.

## Data Flow

### Primary Analysis Flow

```
[User runs: ars scan ./myproject]
    |
    v
[CLI Layer] -- validates args, creates pipeline
    |
    v
[Scanner] -- walks directory tree
    |          filters .go files (skip vendor/, .git/, testdata/)
    |          returns []FilePath
    v
[Parser] -- calls go/parser.ParseFile for each file
    |         uses shared token.FileSet for position tracking
    |         returns []*ParsedFile{Path, AST, Fset}
    v
[Analyzers] -- each analyzer traverses ASTs independently
    |           C1: walks functions, counts branches, measures length
    |           C3: analyzes import graph, directory structure
    |           C6: identifies test files, counts test funcs, checks patterns
    |           returns map[category]*CategoryMetrics
    v
[Scorer] -- converts raw metrics to 1-10 scores per category
    |         applies weights (C1:25%, C3:20%, C6:15%)
    |         computes composite score
    |         assigns tier (Agent-Ready/Assisted/Limited/Hostile)
    v
[Recommender] -- ranks metrics by distance from ideal
    |             selects top 5 highest-impact improvements
    |             generates actionable text
    v
[Renderer] -- formats report for terminal
              prints category scores, composite score, tier, recommendations
              exits with appropriate code (0/1/2)
```

### Key Data Types Flowing Through Pipeline

```
FilePath (string)
    -> ParsedFile {Path, AST *ast.File, Fset *token.FileSet}
        -> CategoryMetrics {Name, FileMetrics[], FunctionMetrics[], PackageMetrics[]}
            -> CategoryScore {Name, Score float64, Details map[string]float64}
                -> Report {CategoryScores[], CompositeScore, Tier, Recommendations[]}
                    -> terminal output (string)
```

### Key Data Flows

1. **File Discovery Flow:** CLI provides directory path -> Scanner recursively walks it -> filters to `.go` files only -> skips `vendor/`, `.git/`, `testdata/`, `_test.go` (for non-C6 analyzers).

2. **AST Sharing Flow:** Parser creates ASTs once. All analyzers receive the same parsed ASTs as read-only input. No re-parsing. This is the main performance optimization at small scale.

3. **Metrics Aggregation Flow:** Analyzers produce per-file and per-function metrics. Scorer aggregates file-level metrics into package-level, then project-level averages/distributions. Scoring happens at the project level.

## Scaling Considerations

| Scale | Architecture Adjustments |
|-------|--------------------------|
| Small repo (<100 files) | Sequential parse and analyze. No concurrency needed. Total time <1s. |
| Medium repo (100-1000 files) | Parse files concurrently (worker pool, N=GOMAXPROCS). Analyze sequentially on pre-parsed ASTs. Total time ~5-15s. |
| Large repo (1000-10000+ files) | Concurrent parsing with bounded workers. Consider streaming: parse-then-analyze per batch rather than loading all ASTs into memory at once. Total time ~30s-2min. |

### Scaling Priorities

1. **First bottleneck: Parsing.** `go/parser.ParseFile` is the most expensive per-file operation. Parallelize with a worker pool of `runtime.NumCPU()` goroutines. Use `parser.ParseComments` only if needed (C6 analyzer might need comments for `// nolint` detection, but probably not for v1).

2. **Second bottleneck: Memory.** For 10k+ files, holding all ASTs in memory simultaneously could use significant RAM. Mitigation: process in batches -- parse N files, analyze them, discard ASTs, repeat. Per-file metrics are small structs; only the ASTs are large.

3. **Third bottleneck (unlikely for v1): Import graph analysis.** C3's circular dependency detection requires building a full import graph. For very large repos this graph could be large, but Go import graphs are typically manageable. Use adjacency list representation, not matrix.

### Concurrency Recommendation for ARS

Keep it simple for v1:

```go
// Simple bounded concurrency for parsing
func ParseConcurrently(paths []string, workers int) ([]*ParsedFile, error) {
    results := make([]*ParsedFile, len(paths))
    errs := make([]error, len(paths))
    sem := make(chan struct{}, workers)
    var wg sync.WaitGroup

    for i, path := range paths {
        wg.Add(1)
        sem <- struct{}{}
        go func(i int, path string) {
            defer wg.Done()
            defer func() { <-sem }()
            results[i], errs[i] = parseFile(path)
        }(i, path)
    }
    wg.Wait()
    // collect errors...
    return results, firstError(errs)
}
```

No external libraries needed. Go's built-in goroutines + channels + sync.WaitGroup are sufficient. Do NOT use `pond` or other worker pool libraries -- that violates KISS for this use case.

**Confidence:** HIGH -- This is idiomatic Go concurrency. The semaphore pattern with goroutines is standard.

## Anti-Patterns

### Anti-Pattern 1: Using `go/packages` When You Only Need AST

**What people do:** Import `golang.org/x/tools/go/packages` to load full type-checked package information.

**Why it's wrong:** `go/packages` invokes `go list` under the hood, which is slow (often 5-10x slower than direct `go/parser.ParseFile`). It loads type information, dependency graphs, and resolved imports -- most of which ARS does not need for v1. It also requires the target repo to have `go.mod` properly configured and all dependencies downloaded.

**Do this instead:** Use `go/parser.ParseFile` directly. Walk the directory, find `.go` files, parse each one. Extract imports from `ast.File.Imports` directly. You get everything ARS needs (AST, function declarations, import paths, file positions) without the overhead of full type checking.

**When to reconsider:** If a future analyzer needs type-resolved information (e.g., "does this function return an error type?"), then `go/packages` or `go/types` becomes necessary. But for C1/C3/C6 in v1, AST-level analysis is sufficient.

**Confidence:** HIGH -- Verified by surveying gocyclo, staticcheck approach. Cyclomatic complexity, function length, import analysis, and test detection all work at AST level without type info.

### Anti-Pattern 2: Over-Abstracted Plugin System

**What people do:** Build a plugin architecture with dynamic loading, configuration-driven analyzer registration, and hot-swapping.

**Why it's wrong:** ARS has three analyzers. A plugin system adds complexity (interface design, registration, configuration, error handling) that will never pay for itself. Golangci-lint needs plugins because it manages 100+ third-party linters. ARS does not.

**Do this instead:** Use a simple Go interface with concrete implementations compiled in. Adding a new analyzer means adding a file in `internal/analyzer/` and adding one line in the pipeline setup. That is the appropriate level of extensibility.

### Anti-Pattern 3: Storing All Results in a Database

**What people do:** Use SQLite or similar to store intermediate analysis results for querying.

**Why it's wrong:** ARS is a single-run CLI tool. It scans, scores, and exits. There is no query interface, no historical comparison (in v1), no need for persistent storage. An in-memory struct flowing through the pipeline is the right data structure.

**Do this instead:** Use plain Go structs passed between pipeline stages. If historical comparison is needed later, serialize the final `Report` struct to JSON -- do not introduce a database.

### Anti-Pattern 4: Premature Streaming Architecture

**What people do:** Build a channel-based streaming pipeline where every stage communicates via channels.

**Why it's wrong:** For ARS, the pipeline is simple and sequential at the stage level. Each stage needs ALL results from the previous stage (e.g., scoring needs all metrics from all files). Channels add complexity (error propagation, cancellation, backpressure) without enabling any meaningful streaming.

**Do this instead:** Use simple function calls between stages. Concurrency belongs WITHIN a stage (parallel file parsing) not BETWEEN stages. The pipeline function in Pattern 1 above is the right approach.

## Integration Points

### External Services

None for v1. ARS is a standalone CLI tool with no network dependencies.

### Internal Boundaries

| Boundary | Communication | Notes |
|----------|---------------|-------|
| CLI -> Scanner | Function call with directory path | Scanner returns `[]string` of file paths |
| Scanner -> Parser | Function call with file paths | Parser returns `[]*ParsedFile` with ASTs |
| Parser -> Analyzers | Function call with parsed files | Each analyzer gets same `[]*ParsedFile`, returns `*CategoryMetrics` |
| Analyzers -> Scorer | Function call with metrics map | Scorer returns `*ScoredReport` |
| Scorer -> Recommender | Function call with scored report | Recommender returns `[]Recommendation` |
| Recommender -> Renderer | Function call with full report | Renderer writes to `io.Writer` (defaults to `os.Stdout`) |

All boundaries are simple function calls in v1. No RPC, no events, no message passing. The `io.Writer` parameter for the renderer enables testing (write to `bytes.Buffer` instead of stdout).

## Build Order Implications

Based on data flow dependencies, the recommended implementation order is:

| Order | Component | Depends On | Rationale |
|-------|-----------|------------|-----------|
| 1 | Shared types (`pkg/types/`) | Nothing | All components reference these structs |
| 2 | CLI skeleton (`cmd/`) | Nothing | Validates project setup, gives a runnable binary immediately |
| 3 | Scanner (`internal/scanner/`) | Types | File discovery is the pipeline entry point |
| 4 | Parser (`internal/parser/`) | Types, Scanner | Must parse before analyzing |
| 5 | One analyzer (start with C1) | Types, Parser | Proves the full pipeline works end-to-end |
| 6 | Scorer + Thresholds | Types, Analyzer output | Converts metrics to scores |
| 7 | Terminal Renderer | Types, Scorer output | Produces visible output |
| 8 | Wire full pipeline | All above | End-to-end: directory in, report out |
| 9 | Remaining analyzers (C3, C6) | Types, Parser | Add independently once pipeline works |
| 10 | Recommender | Scorer output | Enhancement on top of scoring |

**Key insight:** Get one analyzer (C1: Code Health) working end-to-end through the full pipeline first. This validates the architecture. Then add C3 and C6 as parallel work streams -- they plug into the same interface.

## Sources

- [golangci-lint Architecture](https://golangci-lint.run/docs/contributing/architecture/) -- Official architecture docs showing Init -> Load -> Run -> Postprocess -> Print pipeline (HIGH confidence)
- [golang.org/x/tools/go/analysis package](https://pkg.go.dev/golang.org/x/tools/go/analysis) -- Official Go analysis framework defining Analyzer type pattern (HIGH confidence)
- [gocyclo](https://github.com/fzipp/gocyclo) -- Reference implementation for cyclomatic complexity via AST walking (HIGH confidence)
- [go-complexity-analysis](https://github.com/shoooooman/go-complexity-analysis) -- Halstead + cyclomatic + maintainability index calculation (HIGH confidence)
- [Cloudflare: Building the simplest Go static analysis tool](https://blog.cloudflare.com/building-the-simplest-go-static-analysis-tool/) -- Tutorial on go/parser + ast.Inspect pattern (MEDIUM confidence)
- [PVS-Studio: How to create your own Go static analyzer](https://pvs-studio.com/en/blog/posts/go/1329/) -- Dec 2025, covers go/ast, go/analysis, SSA patterns (MEDIUM confidence)
- [ByteSizeGo: Structuring Go CLI Applications](https://www.bytesizego.com/blog/structure-go-cli-app) -- Go project layout patterns (MEDIUM confidence)
- [Go Concurrency Patterns: Worker Pool](https://gobyexample.com/worker-pools) -- Standard bounded concurrency (HIGH confidence)

---
*Architecture research for: Go CLI static analysis tool (ARS)*
*Researched: 2026-01-31*
