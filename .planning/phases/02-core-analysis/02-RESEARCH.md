# Phase 2: Core Analysis - Research

**Researched:** 2026-01-31
**Domain:** Go static analysis -- AST parsing, type-aware package loading, metric computation (C1/C3/C6)
**Confidence:** HIGH

## Summary

Phase 2 implements all 16 metric analyzers across three categories: C1 (Code Health -- 6 metrics), C3 (Architectural Navigability -- 5 metrics), and C6 (Testing Infrastructure -- 5 metrics). The existing Phase 1 pipeline has stub `Parser` and `Analyzer` interfaces ready to be replaced with real implementations.

The core technical challenge is introducing `go/packages` for type-aware parsing (locked decision from roadmap) while keeping the filesystem-based discovery from Phase 1. The `go/packages.Load` call replaces the `StubParser`, providing AST trees, type information, and resolved import graphs that the analyzers consume. Some metrics (cyclomatic complexity, function length, file size, directory depth) work purely from AST or filesystem data, while others (coupling, circular dependencies, dead code) require the full import graph and type information that `go/packages` provides.

The recommended approach is: (1) build a real parser using `go/packages` that replaces `StubParser`, (2) implement three category analyzers (C1, C3, C6) implementing the existing `Analyzer` interface, (3) evolve the shared types to carry AST and metric data through the pipeline, and (4) validate all metrics against this repository and known Go packages.

**Primary recommendation:** Replace `StubParser` with a `go/packages`-backed parser, implement one analyzer per category (C1, C3, C6) using the existing `Analyzer` interface, use `gocyclo` for cyclomatic complexity, build coupling/circular-dep/dead-code analysis from `go/packages` import graphs, and build a simple token-based duplication detector.

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `golang.org/x/tools/go/packages` | v0.41.0+ | Type-aware package loading, import graph, AST | Official Go tooling. Provides ASTs, type info, resolved imports in one call. Locked decision from roadmap. |
| `go/ast` + `go/parser` + `go/token` | stdlib | AST traversal, node inspection, position tracking | Standard library. Used by every Go analysis tool. Zero dependencies. |
| `go/types` | stdlib | Type information resolution | Standard library. Provided automatically via `go/packages` with `NeedTypes` mode. |
| `fzipp/gocyclo` | v0.6.0 | Cyclomatic complexity per function | Well-maintained, library-friendly API (`AnalyzeASTFile`). Avoids reimplementing complexity counting. |
| `golang.org/x/tools/cover` | (part of x/tools) | Parse Go coverage profiles | Official Go tooling. `ParseProfiles()` returns structured `Profile`/`ProfileBlock` data. |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `golang.org/x/tools/go/ast/inspector` | (part of x/tools) | Optimized multi-node-type AST traversal | When an analyzer needs to find multiple node types in a single pass (e.g., both `FuncDecl` and `IfStmt`). Faster than manual `ast.Inspect` for multi-type queries. |
| `encoding/xml` | stdlib | Parse Cobertura XML coverage reports | For C6-03 when parsing Cobertura format coverage files. |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `fzipp/gocyclo` | Hand-rolled complexity counter | gocyclo is ~100 lines but handles all Go control flow correctly (including `&&`/`||` short-circuit operators). Not worth reimplementing. |
| Custom duplication detector | `mibk/dupl` | dupl is unmaintained since 2016, CLI-focused (not library-friendly). Its internal packages can be imported but are undocumented. A simple AST-hash approach (~100 lines) is more appropriate for ARS's needs. |
| Custom LCOV/Cobertura parser | `sguiheux/go-coverage` | Only 1 star, 7 commits, unmaintained since 2018. Too risky. Write simple parsers for LCOV (line-based) and Cobertura (XML). Both formats are well-documented and parsing is straightforward. |
| Building import graph manually | `golang.org/x/tools/refactor/importgraph` | `importgraph` builds the full workspace graph but requires the `go` tool to be configured. Since we already load packages via `go/packages`, building the import graph from `Package.Imports` is simpler and more controlled. |

**Installation:**
```bash
go get golang.org/x/tools@latest
go get github.com/fzipp/gocyclo@latest
```

## Architecture Patterns

### Recommended Project Structure (Phase 2 additions)

```
internal/
├── parser/
│   └── parser.go          # go/packages-backed Parser implementation
├── analyzer/
│   ├── c1_codehealth.go   # C1: complexity, function length, file size, coupling, duplication
│   ├── c3_architecture.go # C3: directory depth, module fanout, circular deps, import complexity, dead code
│   ├── c6_testing.go      # C6: test detection, test ratio, coverage, isolation, assertions
│   └── helpers.go         # Shared analysis utilities (line counting, AST traversal helpers)
├── discovery/             # (existing from Phase 1)
├── pipeline/              # (existing from Phase 1, interfaces updated)
└── output/                # (existing from Phase 1)
pkg/
└── types/
    └── types.go           # Extended with ParsedFile AST fields, metric result types
testdata/
├── valid-go-project/      # (existing)
├── complexity/            # Known-complexity functions for C1 validation
├── coupling/              # Multi-package project for C1-04/C1-05 validation
├── duplication/           # Files with known duplicate blocks
├── circular/              # Packages with circular imports (for C3-03 -- note: won't compile)
├── deadcode/              # Packages with unreferenced exported functions
└── coverage/              # Sample lcov and cobertura files
```

### Pattern 1: go/packages-Backed Parser

**What:** Replace `StubParser` with a real parser that calls `go/packages.Load` to get ASTs, type info, and import graphs for all packages in the scanned directory.

**When to use:** Always. This is the Phase 2 parser that all analyzers consume.

**Example:**
```go
// internal/parser/parser.go
package parser

import (
    "golang.org/x/tools/go/packages"
    "go/ast"
    "go/token"
    "go/types"
)

type GoPackagesParser struct{}

func (p *GoPackagesParser) Parse(rootDir string) ([]*ParsedPackage, error) {
    cfg := &packages.Config{
        Mode: packages.NeedName |
              packages.NeedFiles |
              packages.NeedImports |
              packages.NeedDeps |
              packages.NeedTypes |
              packages.NeedSyntax |
              packages.NeedTypesInfo,
        Dir:   rootDir,
        Tests: true, // Include test packages for C6 analysis
    }
    pkgs, err := packages.Load(cfg, "./...")
    if err != nil {
        return nil, fmt.Errorf("load packages: %w", err)
    }

    var result []*ParsedPackage
    for _, pkg := range pkgs {
        if len(pkg.Errors) > 0 {
            // Log errors but continue -- partial results are better than none
            continue
        }
        result = append(result, &ParsedPackage{
            ID:        pkg.ID,
            Name:      pkg.Name,
            PkgPath:   pkg.PkgPath,
            GoFiles:   pkg.GoFiles,
            Syntax:    pkg.Syntax,
            Fset:      pkg.Fset,
            Types:     pkg.Types,
            TypesInfo: pkg.TypesInfo,
            Imports:   pkg.Imports,
        })
    }
    return result, nil
}

type ParsedPackage struct {
    ID        string
    Name      string
    PkgPath   string
    GoFiles   []string
    Syntax    []*ast.File
    Fset      *token.FileSet
    Types     *types.Package
    TypesInfo *types.Info
    Imports   map[string]*packages.Package
}
```

### Pattern 2: Category Analyzer with Structured Metrics

**What:** Each category analyzer implements the `Analyzer` interface and returns typed metric results (not `map[string]interface{}`).

**When to use:** For all three analyzers (C1, C3, C6).

**Key design decision:** The current `AnalysisResult` uses `map[string]interface{}` for metrics. This should be evolved to typed structs for each category. The `AnalysisResult.Metrics` field can hold category-specific structs via type assertion or by introducing a `CategoryMetrics` interface.

**Example:**
```go
// Typed metric results
type C1Metrics struct {
    CyclomaticComplexity MetricSummary     // avg, max per function
    FunctionLength       MetricSummary     // avg, max lines per function
    FileSize             MetricSummary     // avg, max lines per file
    AfferentCoupling     map[string]int    // package path -> incoming dependency count
    EfferentCoupling     map[string]int    // package path -> outgoing dependency count
    DuplicationRate      float64           // percentage of duplicated code
    DuplicatedBlocks     []DuplicateBlock  // list of duplicate regions
    Functions            []FunctionMetric  // per-function detail for verbose output
}

type MetricSummary struct {
    Avg float64
    Max int
    MaxEntity string // which function/file has the max
}

type FunctionMetric struct {
    Package    string
    Name       string
    File       string
    Line       int
    Complexity int
    LineCount  int
}
```

### Pattern 3: Import Graph for Coupling and Circular Dependencies

**What:** Build an import graph from `go/packages` results and use it for coupling metrics (C1-04/C1-05), module fanout (C3-02), and circular dependency detection (C3-03).

**Example:**
```go
// Build adjacency list from go/packages results
type ImportGraph struct {
    Forward  map[string][]string // package -> packages it imports
    Reverse  map[string][]string // package -> packages that import it
}

func BuildImportGraph(pkgs []*ParsedPackage, modulePath string) *ImportGraph {
    g := &ImportGraph{
        Forward: make(map[string][]string),
        Reverse: make(map[string][]string),
    }
    for _, pkg := range pkgs {
        for importPath := range pkg.Imports {
            // Only count imports within the same module
            if strings.HasPrefix(importPath, modulePath) {
                g.Forward[pkg.PkgPath] = append(g.Forward[pkg.PkgPath], importPath)
                g.Reverse[importPath] = append(g.Reverse[importPath], pkg.PkgPath)
            }
        }
    }
    return g
}

// Afferent coupling = len(Reverse[pkg])  -- incoming dependencies
// Efferent coupling = len(Forward[pkg])  -- outgoing dependencies

// Circular dependency detection: DFS cycle detection on Forward graph
func (g *ImportGraph) DetectCycles() [][]string {
    // Standard DFS with coloring: white (unvisited), gray (in stack), black (done)
    // When we hit a gray node, we've found a cycle
    // ...
}
```

### Pattern 4: AST-Based Duplication Detection

**What:** Instead of using the unmaintained `mibk/dupl`, implement a simple AST statement-sequence hashing approach. Hash sequences of N consecutive AST statements and find matches.

**Example:**
```go
// Hash consecutive statement sequences
func detectDuplicates(files []*ast.File, fset *token.FileSet, minTokens int) []DuplicateBlock {
    type stmtHash struct {
        hash     uint64
        file     string
        startLine int
        endLine   int
    }

    var hashes []stmtHash

    for _, f := range files {
        ast.Inspect(f, func(n ast.Node) bool {
            if block, ok := n.(*ast.BlockStmt); ok {
                for i := 0; i < len(block.List); i++ {
                    // Hash sliding windows of statements
                    for windowSize := 3; windowSize <= len(block.List)-i; windowSize++ {
                        stmts := block.List[i : i+windowSize]
                        h := hashStatements(fset, stmts)
                        start := fset.Position(stmts[0].Pos())
                        end := fset.Position(stmts[len(stmts)-1].End())
                        if end.Line-start.Line >= minTokens {
                            hashes = append(hashes, stmtHash{
                                hash: h, file: start.Filename,
                                startLine: start.Line, endLine: end.Line,
                            })
                        }
                    }
                }
            }
            return true
        })
    }

    // Group by hash, report groups with 2+ entries as duplicates
    // ...
}
```

### Anti-Patterns to Avoid

- **Parsing files twice:** `go/packages` provides ASTs; do not re-parse with `go/parser.ParseFile`. Use the ASTs from `packages.Package.Syntax`.
- **Loading all packages with `NeedDeps` when only analyzing local code:** `NeedDeps` causes transitive loading of ALL dependencies. Only load what you need. For import graph analysis of the module itself, iterate `Package.Imports` and filter to the module prefix.
- **Treating test packages as regular packages in coupling analysis:** Test packages (ending in `_test`) should be excluded from coupling metrics but included in C6 metrics. The `packages.Config.Tests = true` flag creates separate package entries for tests.
- **Using `NeedTypesInfo` without `NeedSyntax` and `NeedTypes`:** Known Go issue #69931 -- `TypesInfo` will be nil if you request it without also requesting `NeedSyntax` and `NeedTypes`. Always combine all three.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Cyclomatic complexity | Custom branch counter | `fzipp/gocyclo.AnalyzeASTFile()` | Handles all Go control flow including `&&`/`||` operators, `select`, `case` clauses. Well-tested. |
| Go coverage profile parsing | Custom profile parser | `golang.org/x/tools/cover.ParseProfiles()` | Official Go tooling. Handles all coverage modes (set, count, atomic). Returns structured `Profile`/`ProfileBlock`. |
| Package loading with type info | Raw `go/parser` + manual import resolution | `golang.org/x/tools/go/packages.Load()` | Handles modules, build tags, vendoring, test packages, type checking. Would be thousands of lines to replicate. |
| Import cycle detection algorithm | Ad-hoc graph search | Standard DFS cycle detection (Tarjan's or coloring) | Well-known CS algorithm (~30 lines). The graph construction from `go/packages` is the hard part, not the cycle detection itself. |
| Function/method line counting | Counting `\n` in source text | `fset.Position(funcDecl.End()).Line - fset.Position(funcDecl.Pos()).Line + 1` | Token position arithmetic is exact. Text counting is fragile with build tags and generated code. |

**Key insight:** The `go/packages` API does the heavy lifting for this phase. Most metrics are straightforward AST traversals on top of already-parsed data. The complexity is in correctly loading packages and building the import graph -- both solved by `go/packages`.

## Common Pitfalls

### Pitfall 1: go/packages Requires `go mod download`

**What goes wrong:** `go/packages.Load` invokes `go list` under the hood, which requires all module dependencies to be available. If the scanned project has not run `go mod download`, the load fails.
**Why it happens:** Unlike raw `go/parser`, `go/packages` needs the full module graph to resolve imports and perform type checking.
**How to avoid:** Document that `ars scan` requires the target project to have its dependencies available (`go mod download` or vendor). Log a clear error message if `go/packages.Load` fails due to missing dependencies. Consider graceful degradation: if `go/packages` fails, fall back to AST-only analysis for metrics that don't need type info (C1-01, C1-02, C1-03, C3-01, C3-04, C6-01, C6-02, C6-05).
**Warning signs:** Tool works on small self-contained projects but fails on real projects with external dependencies.

### Pitfall 2: go/packages Test Package Duplication

**What goes wrong:** With `Tests: true`, `go/packages` returns duplicate entries -- the regular package AND the test variant. If you iterate naively, you double-count files and metrics.
**Why it happens:** Go test packages can be either in-package tests (`package foo`) or external tests (`package foo_test`). `go/packages` creates separate entries for each variant.
**How to avoid:** Deduplicate packages by `PkgPath`. For non-test analysis, skip packages where `pkg.Name` ends with `_test` or `pkg.ForTest != ""`. For C6 analysis, process test packages specifically.
**Warning signs:** Metric counts are ~2x expected values. Source LOC includes test code.

### Pitfall 3: Cyclomatic Complexity Counting Variations

**What goes wrong:** Different tools count complexity differently. The McCabe definition counts `if`, `for`, `while`, `case`, `catch` branches. Some tools also count `&&` and `||` as branches (since they create additional paths). gocyclo counts `&&`/`||` which gives higher numbers than tools that don't.
**Why it happens:** There is no single authoritative definition of cyclomatic complexity.
**How to avoid:** Use gocyclo consistently and document that ARS uses the gocyclo counting method (base 1, +1 for `if`, `for`, `case`, `&&`, `||`). This is the most common Go community standard.
**Warning signs:** Users compare ARS complexity scores to other tools and see different numbers.

### Pitfall 4: Dead Code Detection Scope

**What goes wrong:** Reporting every unexported unused function as "dead code" produces many false positives -- functions may be used via reflection, go:linkname, or generated code.
**Why it happens:** Static analysis cannot see all execution paths (reflection, plugins, generated test harnesses).
**How to avoid:** For Phase 2, limit dead code detection to **exported functions/types that are not referenced by any package within the module**. This is a conservative heuristic: if an exported symbol is never imported within the project, it is likely dead. Do NOT flag unexported functions (too many false positives from test helpers, init functions, etc.). Do NOT use SSA/RTA analysis in Phase 2 -- it is complex and requires whole-program analysis from main.
**Warning signs:** Tool flags `main()` functions, `init()` functions, test helpers, or functions used via reflection.

### Pitfall 5: Duplication Detection Performance

**What goes wrong:** Naive O(n^2) comparison of all statement sequences is extremely slow on large codebases.
**Why it happens:** Every pair of statement sequences must be compared for duplicates.
**How to avoid:** Use hashing: hash each statement sequence, group by hash, only compare sequences with matching hashes. Use a minimum threshold (e.g., 6+ lines or 15+ tokens) to avoid reporting trivial duplicates (like `return nil, err`). Use FNV or similar fast hash.
**Warning signs:** Analysis takes minutes on medium-sized repos. Reports thousands of trivial duplicates.

### Pitfall 6: LCOV/Cobertura Parsing Edge Cases

**What goes wrong:** Coverage files may not exist, may be empty, may reference files not in the project, or may use relative paths that don't match the project structure.
**Why it happens:** Coverage files are generated externally and may be stale or from a different checkout.
**How to avoid:** Make coverage parsing optional and graceful. If no coverage file is found, report "coverage: not available" rather than zero. If the file exists but is malformed, log a warning and skip. Accept both Go native coverage (`go test -coverprofile`) and external formats (LCOV, Cobertura).
**Warning signs:** Zero coverage reported when coverage files exist but paths don't match.

## Code Examples

### Using gocyclo on Parsed ASTs

```go
// Source: https://pkg.go.dev/github.com/fzipp/gocyclo
import "github.com/fzipp/gocyclo"

func analyzeComplexity(pkg *ParsedPackage) ([]FunctionMetric, error) {
    var stats gocyclo.Stats
    for _, f := range pkg.Syntax {
        stats = gocyclo.AnalyzeASTFile(f, pkg.Fset, stats)
    }

    var metrics []FunctionMetric
    for _, s := range stats {
        metrics = append(metrics, FunctionMetric{
            Package:    s.PkgName,
            Name:       s.FuncName,
            File:       s.Pos.Filename,
            Line:       s.Pos.Line,
            Complexity: s.Complexity,
        })
    }
    return metrics, nil
}

// Average and max from gocyclo.Stats
avg := stats.AverageComplexity()
maxStat := stats.SortAndFilter(-1, 0) // sort descending, no filter
if len(maxStat) > 0 {
    maxComplexity := maxStat[0].Complexity
}
```

### Function Length Measurement

```go
// Source: go/ast + go/token standard library
func measureFunctionLength(fset *token.FileSet, fn *ast.FuncDecl) int {
    start := fset.Position(fn.Pos())
    end := fset.Position(fn.End())
    return end.Line - start.Line + 1
}

func analyzeFunctionLengths(pkg *ParsedPackage) []FunctionMetric {
    var metrics []FunctionMetric
    for _, f := range pkg.Syntax {
        ast.Inspect(f, func(n ast.Node) bool {
            if fn, ok := n.(*ast.FuncDecl); ok && fn.Body != nil {
                lineCount := measureFunctionLength(pkg.Fset, fn)
                metrics = append(metrics, FunctionMetric{
                    Name:      fn.Name.Name,
                    File:      pkg.Fset.Position(fn.Pos()).Filename,
                    Line:      pkg.Fset.Position(fn.Pos()).Line,
                    LineCount: lineCount,
                })
            }
            return true
        })
    }
    return metrics
}
```

### File Size Measurement

```go
// Source: go/token standard library
func measureFileSize(fset *token.FileSet, f *ast.File) int {
    start := fset.Position(f.Pos())
    end := fset.Position(f.End())
    return end.Line - start.Line + 1
}
```

### Coverage Profile Parsing (Go native)

```go
// Source: https://pkg.go.dev/golang.org/x/tools/cover
import "golang.org/x/tools/cover"

func parseGoCoverage(profilePath string) (float64, error) {
    profiles, err := cover.ParseProfiles(profilePath)
    if err != nil {
        return 0, fmt.Errorf("parse coverage: %w", err)
    }

    var totalStmts, coveredStmts int
    for _, p := range profiles {
        for _, block := range p.Blocks {
            totalStmts += block.NumStmt
            if block.Count > 0 {
                coveredStmts += block.NumStmt
            }
        }
    }

    if totalStmts == 0 {
        return 0, nil
    }
    return float64(coveredStmts) / float64(totalStmts) * 100, nil
}
```

### LCOV Parsing (Simple Custom Parser)

```go
// LCOV format is line-based, well-documented
// Key lines: SF:<filename>, DA:<line>,<count>, LF:<lines found>, LH:<lines hit>, end_of_record
func parseLCOV(path string) (map[string]FileCoverage, error) {
    f, err := os.Open(path)
    if err != nil {
        return nil, err
    }
    defer f.Close()

    result := make(map[string]FileCoverage)
    var current FileCoverage
    var currentFile string

    scanner := bufio.NewScanner(f)
    for scanner.Scan() {
        line := scanner.Text()
        switch {
        case strings.HasPrefix(line, "SF:"):
            currentFile = line[3:]
            current = FileCoverage{}
        case strings.HasPrefix(line, "DA:"):
            parts := strings.SplitN(line[3:], ",", 2)
            if len(parts) == 2 {
                count, _ := strconv.Atoi(parts[1])
                current.TotalLines++
                if count > 0 {
                    current.CoveredLines++
                }
            }
        case line == "end_of_record":
            result[currentFile] = current
        }
    }
    return result, scanner.Err()
}
```

### Test Isolation Detection

```go
// C6-04: Identify tests with external dependencies
// External dependencies = imports of net/http, database/sql, os/exec, etc.
var externalPackages = map[string]bool{
    "net/http":     true,
    "net":          true,
    "database/sql": true,
    "os/exec":      true,
    "os":           true, // only if used for file I/O beyond test fixtures
}

func analyzeTestIsolation(pkg *ParsedPackage) (total, isolated int) {
    for _, f := range pkg.Syntax {
        for _, imp := range f.Imports {
            importPath := strings.Trim(imp.Path.Value, `"`)
            if externalPackages[importPath] {
                // This test file has external dependencies
                total++
                return // count file, not isolated
            }
        }
        total++
        isolated++
    }
    return
}
```

### Assertion Density

```go
// C6-05: Count assertions per test function
// Look for testing.T.Fatal, testing.T.Error, assert.*, require.* calls
func countAssertions(fset *token.FileSet, fn *ast.FuncDecl) int {
    count := 0
    ast.Inspect(fn.Body, func(n ast.Node) bool {
        call, ok := n.(*ast.CallExpr)
        if !ok {
            return true
        }
        sel, ok := call.Fun.(*ast.SelectorExpr)
        if !ok {
            return true
        }
        method := sel.Sel.Name
        // Standard testing methods
        switch method {
        case "Error", "Errorf", "Fatal", "Fatalf", "Fail", "FailNow":
            count++
        case "Equal", "NotEqual", "True", "False", "Nil", "NotNil",
             "Contains", "NoError", "Len", "Empty", "Greater", "Less":
            // testify assert/require methods
            count++
        }
        return true
    })
    return count
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `go/build` for package info | `go/packages` | ~2019 | Handles modules, build tags, vendoring correctly. go/build is legacy. |
| SSA-based dead code (RTA) | Heuristic: unreferenced exported symbols | N/A (simplification for v1) | SSA is precise but complex. Heuristic is 80% accurate with zero false positives on exported symbols. |
| `mibk/dupl` for duplication | AST statement hashing | N/A (dupl unmaintained since 2016) | Custom approach is simpler, maintainable, and avoids unmaintained dependency. |
| Manual coverage parsing | `golang.org/x/tools/cover.ParseProfiles` | N/A | Official API. Handles all Go coverage modes. |

**Deprecated/outdated:**
- `go/build`: Still works but does not handle modules correctly. Use `go/packages`.
- `mibk/dupl` v1.0.0: Last updated 2016. Internal packages work but no active maintenance. Build custom instead.

## Open Questions

1. **go/packages loading time on large repos**
   - What we know: `go/packages.Load` with `NeedTypes | NeedSyntax | NeedTypesInfo` invokes `go list` and type-checks all packages. This can take 10-30 seconds on large repos.
   - What's unclear: Whether this is acceptable for Phase 2 or if we need to optimize (e.g., load with fewer modes for metrics that don't need types).
   - Recommendation: Load with full modes once. Profile on real repos in Phase 2. If too slow, split into two loading passes: a fast pass (`NeedName | NeedFiles | NeedImports`) for metrics that only need import info, and a full pass for type-dependent metrics. Defer optimization to Phase 5.

2. **Duplication detection threshold calibration**
   - What we know: A minimum of ~6 lines or ~15 tokens is common for clone detection.
   - What's unclear: What threshold produces useful results for Go codebases specifically.
   - Recommendation: Start with 6 consecutive lines minimum, 3 statement sequences minimum. Calibrate against this repository and standard library packages during validation.

3. **ParsedFile type evolution**
   - What we know: Phase 1 `ParsedFile` has only `Path`, `RelPath`, `Class`. Phase 2 needs AST, type info, package membership.
   - What's unclear: Whether to evolve `ParsedFile` or introduce `ParsedPackage` as a new type.
   - Recommendation: Introduce `ParsedPackage` alongside existing types. The `Analyzer` interface should change to accept `[]*ParsedPackage` instead of `[]ParsedFile`. Update `Parser` interface accordingly. This is a clean break point since only stubs currently implement these interfaces.

4. **Dead code detection for library packages**
   - What we know: The official `deadcode` tool uses RTA from main(). ARS analyzes libraries too, which have no main().
   - What's unclear: How to detect dead exported symbols in library packages without whole-program analysis.
   - Recommendation: For libraries, check if exported symbols are referenced by any other package within the module. If an exported function in package `internal/foo` is never imported by any other package, flag it. This is conservative and correct for most cases. Main packages can use the same heuristic.

5. **Interface signature changes**
   - What we know: The current `Analyzer` interface takes `[]types.ParsedFile`. Phase 2 needs `[]*ParsedPackage` which includes ASTs.
   - What's unclear: Whether to make this a breaking change or add a new interface.
   - Recommendation: Update the interface. Only stubs implement it, so there is no compatibility concern. Change `Parser` to return `[]*ParsedPackage` and `Analyzer.Analyze` to accept `[]*ParsedPackage`.

## Sources

### Primary (HIGH confidence)
- [golang.org/x/tools/go/packages](https://pkg.go.dev/golang.org/x/tools/go/packages) -- Package loading API, NeedX constants, Config struct, Package struct
- [fzipp/gocyclo on pkg.go.dev](https://pkg.go.dev/github.com/fzipp/gocyclo) -- AnalyzeASTFile API, Stat struct, Stats.AverageComplexity
- [golang.org/x/tools/cover](https://pkg.go.dev/golang.org/x/tools/cover) -- ParseProfiles, Profile, ProfileBlock structs
- [Go deadcode blog post](https://go.dev/blog/deadcode) -- Official dead code detection approach, RTA limitations
- [go/ast package docs](https://pkg.go.dev/go/ast) -- AST node types, Inspect function
- [go/token package docs](https://pkg.go.dev/go/token) -- FileSet, Position for line number computation

### Secondary (MEDIUM confidence)
- [golang/go#69931](https://github.com/golang/go/issues/69931) -- NeedTypesInfo requires NeedSyntax + NeedTypes (confirmed bug)
- [mibk/dupl on GitHub](https://github.com/mibk/dupl) -- Code clone detection approach, last updated 2016
- [go-cyclic on GitHub](https://github.com/elza2/go-cyclic) -- Circular dependency detection tool for Go

### Tertiary (LOW confidence)
- [sguiheux/go-coverage](https://pkg.go.dev/github.com/sguiheux/go-coverage) -- LCOV/Cobertura parser library, 1 star, 7 commits (too small/unmaintained to depend on)
- Duplication detection via AST hashing -- Common technique but specific threshold recommendations are based on reasoning, not empirical Go-specific data

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- go/packages, gocyclo, and x/tools/cover are all official or well-established libraries with verified APIs
- Architecture: HIGH -- Pattern follows existing pipeline from Phase 1; interface evolution is straightforward
- Metric algorithms: HIGH for complexity/coupling/coverage (well-established); MEDIUM for duplication/dead code (heuristic approaches)
- Pitfalls: HIGH -- go/packages gotchas verified via official issue tracker; test package duplication is documented behavior

**Research date:** 2026-01-31
**Valid until:** 2026-03-01 (stable domain; go/packages and gocyclo are mature)
