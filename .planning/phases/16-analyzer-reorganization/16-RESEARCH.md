# Phase 16: Analyzer Reorganization - Research

**Researched:** 2026-02-04
**Domain:** Go package reorganization, code migration patterns
**Confidence:** HIGH

## Summary

This phase involves reorganizing the analyzer package from a flat structure (31 files, 10,462 LOC) into category-based subdirectories. The research focused on Go package migration patterns, type alias re-exports for backward compatibility, and identifying internal dependencies that must be handled during the reorganization.

The codebase currently has one external consumer of the analyzer package (`internal/pipeline/pipeline.go`), making this a controlled refactoring. Go's type alias feature (`type T = pkg.T`) is the standard mechanism for maintaining backward compatibility when moving types between packages. Internal functions shared across category files (like `nodeText`, `walkTree`, `countLines`) must be carefully placed in a shared location.

**Primary recommendation:** Use type aliases in the root `analyzer.go` file to re-export all public types and constructors, ensuring the pipeline package continues working without modification.

## Standard Stack

This is a refactoring task using standard Go tooling.

### Core
| Tool | Version | Purpose | Why Standard |
|------|---------|---------|--------------|
| Go | 1.25.1 | Project's Go version | Already in use |
| go build | N/A | Verify compilation | Standard Go tool |
| go test | N/A | Verify tests pass | Standard Go tool |

### Supporting
| Tool | Purpose | When to Use |
|------|---------|-------------|
| goimports | Fix imports after moves | On each moved file |
| go vet | Static analysis | After reorganization |

**No new dependencies needed.** This is purely a structural refactoring.

## Architecture Patterns

### Target Directory Structure (Per CONTEXT.md Decisions)
```
internal/analyzer/
├── analyzer.go              # Re-exports for backward compatibility
├── shared.go                # Shared utilities (ImportGraph, nodeText, etc.)
├── c1_code_quality/
│   ├── analyzer.go          # Main C1Analyzer, NewC1Analyzer
│   ├── codehealth.go        # Go-specific analysis
│   ├── python.go            # Python analysis via Tree-sitter
│   ├── typescript.go        # TypeScript analysis via Tree-sitter
│   ├── codehealth_test.go
│   ├── python_test.go
│   └── typescript_test.go
├── c2_semantics/
│   ├── analyzer.go          # Main C2Analyzer, NewC2Analyzer
│   ├── go.go                # Go semantic analysis
│   ├── python.go            # Python semantic analysis
│   ├── typescript.go        # TypeScript semantic analysis
│   ├── go_test.go
│   ├── python_test.go
│   └── typescript_test.go
├── c3_architecture/
│   ├── analyzer.go          # Main C3Analyzer, NewC3Analyzer
│   ├── architecture.go      # Go-specific architecture analysis
│   ├── python.go            # Python architecture analysis
│   ├── typescript.go        # TypeScript architecture analysis
│   ├── architecture_test.go
│   ├── python_test.go
│   └── typescript_test.go
├── c4_documentation/
│   ├── analyzer.go          # Main C4Analyzer, NewC4Analyzer
│   ├── documentation.go     # Documentation analysis logic
│   └── documentation_test.go
├── c5_temporal/
│   ├── analyzer.go          # Main C5Analyzer, NewC5Analyzer
│   ├── temporal.go          # Git-based temporal analysis
│   └── temporal_test.go
├── c6_testing/
│   ├── analyzer.go          # Main C6Analyzer, NewC6Analyzer
│   ├── testing.go           # Go-specific test analysis
│   ├── python.go            # Python test analysis
│   ├── typescript.go        # TypeScript test analysis
│   ├── testing_test.go
│   ├── python_test.go
│   └── typescript_test.go
└── c7_agent/
    ├── analyzer.go          # Main C7Analyzer, NewC7Analyzer
    ├── agent.go             # Agent evaluation logic
    └── agent_test.go
```

### Pattern 1: Type Alias Re-exports for Backward Compatibility
**What:** Use Go type aliases to re-export types from new package locations
**When to use:** When moving public types to subdirectories while maintaining existing import paths
**Example:**
```go
// Source: https://go.dev/blog/alias-names
// internal/analyzer/analyzer.go (root level re-exports)
package analyzer

import (
    "github.com/ingo/agent-readyness/internal/analyzer/c1_code_quality"
    "github.com/ingo/agent-readyness/internal/analyzer/c2_semantics"
    // ... etc
)

// Type aliases - exact same type identity
type C1Analyzer = c1.C1Analyzer
type C2Analyzer = c2.C2Analyzer
type C3Analyzer = c3.C3Analyzer
type C4Analyzer = c4.C4Analyzer
type C5Analyzer = c5.C5Analyzer
type C6Analyzer = c6.C6Analyzer
type C7Analyzer = c7.C7Analyzer

// Function wrappers for constructors
func NewC1Analyzer(tsParser *parser.TreeSitterParser) *C1Analyzer {
    return c1.NewC1Analyzer(tsParser)
}

func NewC2Analyzer(tsParser *parser.TreeSitterParser) *C2Analyzer {
    return c2.NewC2Analyzer(tsParser)
}
// ... etc
```

### Pattern 2: Shared Utilities at Root Level
**What:** Keep commonly-used internal functions in a shared location at root
**When to use:** When multiple subdirectories need the same helper functions
**Example:**
```go
// internal/analyzer/shared.go
package analyzer

import (
    tree_sitter "github.com/tree-sitter/go-tree-sitter"
    "github.com/ingo/agent-readyness/internal/parser"
)

// ImportGraph holds forward and reverse adjacency lists for intra-module imports.
type ImportGraph struct {
    Forward map[string][]string
    Reverse map[string][]string
}

// BuildImportGraph constructs an import graph from parsed packages.
func BuildImportGraph(pkgs []*parser.ParsedPackage, modulePath string) *ImportGraph {
    // ... existing implementation
}

// WalkTree walks a Tree-sitter tree depth-first, calling fn for each node.
func WalkTree(node *tree_sitter.Node, fn func(*tree_sitter.Node)) {
    // ... existing implementation
}

// NodeText extracts the text content of a Tree-sitter node.
func NodeText(node *tree_sitter.Node, content []byte) string {
    // ... existing implementation
}

// CountLines counts lines in source content.
func CountLines(content []byte) int {
    // ... existing implementation
}
```

### Anti-Patterns to Avoid
- **Circular imports:** Category packages should import from shared, not from each other
- **Breaking type identity:** Use `type T = pkg.T` (alias), not `type T pkg.T` (new type)
- **Exposing internal functions:** Keep helper functions unexported if only used within category

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Re-export types | Manual wrapper structs | Go type aliases (`type T = pkg.T`) | Maintains type identity, no runtime cost |
| Fix imports | Manual search/replace | `goimports` after each file move | Automatically fixes imports |
| Verify refactoring | Manual testing | `go build ./...` and `go test ./...` | Catches all import/type errors |

**Key insight:** Go's type alias feature was specifically designed for this use case (moving types between packages during refactoring).

## Common Pitfalls

### Pitfall 1: Breaking Type Identity
**What goes wrong:** Using `type T struct { pkg.T }` or `type T pkg.T` instead of `type T = pkg.T`
**Why it happens:** Confusion between type definitions and type aliases
**How to avoid:** Always use `=` for backward-compatible re-exports
**Warning signs:** Compilation errors about incompatible types when passing values

### Pitfall 2: Circular Import Dependencies
**What goes wrong:** Category package c1 imports from c2, and c2 imports from c1
**Why it happens:** Not identifying shared dependencies before moving files
**How to avoid:** Extract shared utilities to root-level shared.go FIRST
**Warning signs:** Import cycle errors from Go compiler

### Pitfall 3: Unexported Functions Becoming Inaccessible
**What goes wrong:** Helper functions like `nodeText` (lowercase) can't be called from other packages
**Why it happens:** Go visibility rules - lowercase functions are package-private
**How to avoid:** Either export shared functions (NodeText) or duplicate in each package
**Warning signs:** "undefined: nodeText" errors after moving files

### Pitfall 4: Forgetting Test Files
**What goes wrong:** Test files reference functions that moved, breaking tests
**Why it happens:** Tests often have their own helper functions that reference implementation
**How to avoid:** Move test files with their implementation files
**Warning signs:** Test compilation failures

### Pitfall 5: Package Name Conflicts
**What goes wrong:** Using `package c1_code_quality` creates awkward import syntax
**Why it happens:** Go package names should be simple identifiers
**How to avoid:** Use `package c1` (short form as decided in CONTEXT.md)
**Warning signs:** Import statements looking like `c1_code_quality.NewC1Analyzer()`

## Code Examples

### Current File to New Location Mapping

| Current File | New Location | Package |
|--------------|--------------|---------|
| `c1_codehealth.go` | `c1_code_quality/codehealth.go` | `c1` |
| `c1_python.go` | `c1_code_quality/python.go` | `c1` |
| `c1_typescript.go` | `c1_code_quality/typescript.go` | `c1` |
| `c2_semantics.go` | `c2_semantics/semantics.go` | `c2` |
| `c2_go.go` | `c2_semantics/go.go` | `c2` |
| `c2_python.go` | `c2_semantics/python.go` | `c2` |
| `c2_typescript.go` | `c2_semantics/typescript.go` | `c2` |
| `c3_architecture.go` | `c3_architecture/architecture.go` | `c3` |
| `c3_python.go` | `c3_architecture/python.go` | `c3` |
| `c3_typescript.go` | `c3_architecture/typescript.go` | `c3` |
| `c4_documentation.go` | `c4_documentation/documentation.go` | `c4` |
| `c5_temporal.go` | `c5_temporal/temporal.go` | `c5` |
| `c6_testing.go` | `c6_testing/testing.go` | `c6` |
| `c6_python.go` | `c6_testing/python.go` | `c6` |
| `c6_typescript.go` | `c6_testing/typescript.go` | `c6` |
| `c7_agent.go` | `c7_agent/agent.go` | `c7` |
| `helpers.go` | `shared.go` (root) | `analyzer` |

### Shared Functions Analysis

Functions currently shared across multiple files (must stay at root or be exported):

| Function | Defined In | Used By | Action |
|----------|------------|---------|--------|
| `BuildImportGraph` | `helpers.go` | c1, c3 | Keep in `shared.go` |
| `ImportGraph` (type) | `helpers.go` | c1, c3 | Keep in `shared.go` |
| `nodeText` | `c2_python.go` | c1, c2, c3, c6 (TS/Py) | Export as `NodeText` in `shared.go` |
| `walkTree` | `c2_python.go` | c2, c3, c6 (TS/Py) | Export as `WalkTree` in `shared.go` |
| `countLines` | `c2_python.go` | c2 (TS/Py) | Export as `CountLines` in `shared.go` |
| `computeComplexitySummary` | `c1_codehealth.go` | Only c1 | Move to c1 package |
| `computeFunctionLengthSummary` | `c1_codehealth.go` | Only c1 | Move to c1 package |
| `pyFilterSourceFiles` | `c1_python.go` | c1, c3 | Export in c1, import in c3 OR duplicate |
| `tsFilterSourceFiles` | `c1_typescript.go` | c1, c3 | Export in c1, import in c3 OR duplicate |

### Analyzer Entry Point Pattern (Per Category)

Each category subdirectory should have an `analyzer.go` that serves as the entry point:

```go
// internal/analyzer/c1_code_quality/analyzer.go
package c1

import (
    "github.com/ingo/agent-readyness/internal/analyzer"  // for shared types
    "github.com/ingo/agent-readyness/internal/parser"
    "github.com/ingo/agent-readyness/pkg/types"
)

// C1Analyzer implements the pipeline.Analyzer interface for C1: Code Health.
type C1Analyzer struct {
    pkgs     []*parser.ParsedPackage
    tsParser *parser.TreeSitterParser
}

// NewC1Analyzer creates a C1Analyzer with Tree-sitter parser.
func NewC1Analyzer(tsParser *parser.TreeSitterParser) *C1Analyzer {
    return &C1Analyzer{tsParser: tsParser}
}

// Name returns the analyzer display name.
func (a *C1Analyzer) Name() string {
    return "C1: Code Health"
}

// SetGoPackages stores Go-specific parsed packages.
func (a *C1Analyzer) SetGoPackages(pkgs []*parser.ParsedPackage) {
    a.pkgs = pkgs
}

// Analyze runs C1 analysis on targets.
func (a *C1Analyzer) Analyze(targets []*types.AnalysisTarget) (*types.AnalysisResult, error) {
    // ... implementation calls into codehealth.go, python.go, typescript.go
}
```

### Root-Level Re-export File

```go
// internal/analyzer/analyzer.go
package analyzer

import (
    "github.com/ingo/agent-readyness/internal/analyzer/c1_code_quality"
    "github.com/ingo/agent-readyness/internal/analyzer/c2_semantics"
    "github.com/ingo/agent-readyness/internal/analyzer/c3_architecture"
    "github.com/ingo/agent-readyness/internal/analyzer/c4_documentation"
    "github.com/ingo/agent-readyness/internal/analyzer/c5_temporal"
    "github.com/ingo/agent-readyness/internal/analyzer/c6_testing"
    "github.com/ingo/agent-readyness/internal/analyzer/c7_agent"
    "github.com/ingo/agent-readyness/internal/parser"
)

// Type aliases for backward compatibility
type C1Analyzer = c1.C1Analyzer
type C2Analyzer = c2.C2Analyzer
type C3Analyzer = c3.C3Analyzer
type C4Analyzer = c4.C4Analyzer
type C5Analyzer = c5.C5Analyzer
type C6Analyzer = c6.C6Analyzer
type C7Analyzer = c7.C7Analyzer

// Constructor wrappers
func NewC1Analyzer(tsParser *parser.TreeSitterParser) *C1Analyzer {
    return c1.NewC1Analyzer(tsParser)
}

func NewC2Analyzer(tsParser *parser.TreeSitterParser) *C2Analyzer {
    return c2.NewC2Analyzer(tsParser)
}

func NewC3Analyzer(tsParser *parser.TreeSitterParser) *C3Analyzer {
    return c3.NewC3Analyzer(tsParser)
}

func NewC4Analyzer(tsParser *parser.TreeSitterParser) *C4Analyzer {
    return c4.NewC4Analyzer(tsParser)
}

func NewC5Analyzer() *C5Analyzer {
    return c5.NewC5Analyzer()
}

func NewC6Analyzer(tsParser *parser.TreeSitterParser) *C6Analyzer {
    return c6.NewC6Analyzer(tsParser)
}

func NewC7Analyzer() *C7Analyzer {
    return c7.NewC7Analyzer()
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Embedding for re-export | Type aliases | Go 1.9 (2017) | Type identity preserved |
| Manual import fixing | goimports | Mature tooling | Automatic import management |
| Atomic refactors | Gradual migration | Always preferred | Lower risk, incremental |

**Note:** Go 1.24 added generic type alias support, but not needed here (no generic types in analyzer).

## Open Questions

1. **Filter function duplication vs import**
   - What we know: `pyFilterSourceFiles` and `tsFilterSourceFiles` are used by both c1 and c3
   - What's unclear: Is it better to duplicate these ~20-line functions or create c1->c3 dependency?
   - Recommendation: Duplicate them (they're small, avoids cross-category dependency)

2. **Test helper function location**
   - What we know: Test files have helpers like `testdataDir()`, `loadTestPackages()`
   - What's unclear: Should these move with tests or be in a shared test helper?
   - Recommendation: Keep with tests (they're test-specific, not shared)

## Sources

### Primary (HIGH confidence)
- [Go Modules Layout](https://go.dev/doc/modules/layout) - Official Go team guidance on package organization
- [Type Aliases in Go](https://go.dev/blog/alias-names) - Official blog post on type alias feature
- [Codebase Refactoring](https://go.dev/talks/2016/refactor.article) - Russ Cox's talk on gradual code repair

### Secondary (MEDIUM confidence)
- [Codilime Go Refactoring](https://codilime.com/blog/golang-code-refactoring-use-case/) - Practical refactoring patterns
- [Dave Cheney Practical Go](https://dave.cheney.net/practical-go/presentations/qcon-china.html) - Package design advice
- [Internal Packages for Monorepos](https://konradreiche.com/blog/use-internal-packages-for-monorepos/) - Internal package patterns

### Tertiary (LOW confidence)
- WebSearch results on Go monorepo patterns (used for general ecosystem understanding)

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - Standard Go tooling, no new dependencies
- Architecture: HIGH - Clear CONTEXT.md decisions + official Go guidance
- Pitfalls: HIGH - Well-documented Go refactoring patterns
- Shared function analysis: HIGH - Grep analysis of actual codebase

**Research date:** 2026-02-04
**Valid until:** 60 days (stable Go patterns, no external dependencies changing)

---

## Appendix: Full File Inventory

Current analyzer files (31 files, 10,462 LOC):

| File | Lines | Category | Has Tests |
|------|-------|----------|-----------|
| c1_codehealth.go | 592 | C1 | Yes |
| c1_python.go | 425 | C1 | Yes |
| c1_typescript.go | 464 | C1 | Yes |
| c2_semantics.go | 131 | C2 | No (dispatcher) |
| c2_go.go | 379 | C2 | Yes |
| c2_python.go | 402 | C2 | Yes |
| c2_typescript.go | 332 | C2 | Yes |
| c3_architecture.go | 457 | C3 | Yes |
| c3_python.go | 307 | C3 | Yes |
| c3_typescript.go | 339 | C3 | Yes |
| c4_documentation.go | 872 | C4 | Yes |
| c5_temporal.go | 484 | C5 | Yes |
| c6_testing.go | 607 | C6 | Yes |
| c6_python.go | 239 | C6 | Yes |
| c6_typescript.go | 305 | C6 | Yes |
| c7_agent.go | 142 | C7 | Yes |
| helpers.go | 37 | Shared | No |

External consumers:
- `internal/pipeline/pipeline.go` - imports `analyzer` package, uses all `NewCxAnalyzer` functions
