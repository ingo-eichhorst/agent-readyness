# Phase 6: Multi-Language Foundation + C2 Semantic Explicitness - Research

**Researched:** 2026-02-01
**Domain:** Multi-language static analysis (Go/Python/TypeScript), Tree-sitter parsing, YAML config, C2 semantic explicitness metrics
**Confidence:** HIGH

## Summary

This phase transforms ARS from a Go-only analyzer to a multi-language tool supporting Go, Python, and TypeScript. The core challenges are: (1) generalizing the pipeline interfaces from Go-specific `ParsedPackage` to a language-agnostic `AnalysisTarget`, (2) integrating Tree-sitter for Python/TypeScript parsing while preserving `go/packages` for Go, (3) implementing C2 semantic explicitness metrics across three languages with fundamentally different type systems, and (4) building a `.arsrc.yml` configuration system for customizable scoring.

The existing v1 architecture is well-structured for expansion. The pipeline pattern (Discover -> Parse -> Analyze -> Score -> Recommend -> Render) remains correct. The primary refactoring is the `Parser` and `Analyzer` interfaces, which currently accept `[]*parser.ParsedPackage` (Go-specific). These must accept a language-agnostic type while preserving Go-specific data access for existing C1/C3/C6 analyzers.

Tree-sitter is the right tool for Python/TypeScript parsing -- it provides fast, error-tolerant syntax trees sufficient for all C2 metrics. The official Go bindings (`tree-sitter/go-tree-sitter`) require explicit `Close()` calls on all allocated objects. The C2 metrics themselves are language-specific: Go uses `go/ast` for interface{}/any detection and naming analysis, while Python/TypeScript use Tree-sitter queries for type annotation coverage, naming patterns, and magic numbers.

**Primary recommendation:** Start with the AnalysisTarget abstraction and interface refactoring. Every other piece (Tree-sitter, C2, config, scoring) depends on this foundation being right.

## Standard Stack

The established libraries/tools for this domain:

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `tree-sitter/go-tree-sitter` | v0.25.0+ | Core Tree-sitter Go bindings | Official bindings from tree-sitter org. Clean API (Parser, Tree, Node, Query, QueryCursor). Grammars imported separately. Requires explicit Close() calls. |
| `tree-sitter/tree-sitter-python/bindings/go` | latest | Python grammar for Tree-sitter | Official Python grammar with Go bindings. Supports all Python 3.x syntax including type annotations. |
| `tree-sitter/tree-sitter-typescript/bindings/go` | latest | TypeScript + TSX grammar | Official grammar. Exports both TypeScript and TSX language functions (two separate grammars). |
| `gopkg.in/yaml.v3` | v3.0.1 | YAML config parsing | Already indirect dep in go.mod. De facto Go YAML library. Supports struct tags, KnownFields validation. |

### Already In Use (v1, unchanged)
| Library | Version | Purpose |
|---------|---------|---------|
| `go/ast` + `go/types` + `go/packages` | stdlib + v0.41.0 | Go parsing (preserved for Go analysis) |
| `fzipp/gocyclo` | v0.6.0 | Go cyclomatic complexity |
| `spf13/cobra` | v1.10.2 | CLI framework |
| `fatih/color` | v1.18.0 | Terminal colors |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `tree-sitter/go-tree-sitter` (official) | `smacker/go-tree-sitter` (community) | smacker bundles all grammars (binary bloat), uses buggy SetFinalizer for memory mgmt. Official bindings are newer but require explicit Close(). Official is the correct choice. |
| `gopkg.in/yaml.v3` | `spf13/viper` | Viper is config framework overkill (env vars, remote config, watching). We parse one YAML file. |
| `gopkg.in/yaml.v3` | `goccy/go-yaml` | Better YAML spec compliance but yaml.v3 is already in our dep tree and handles all real configs. |

**Installation:**
```bash
go get github.com/tree-sitter/go-tree-sitter@latest
go get github.com/tree-sitter/tree-sitter-python/bindings/go@latest
go get github.com/tree-sitter/tree-sitter-typescript/bindings/go@latest
# yaml.v3 already indirect -- promote to direct:
go get gopkg.in/yaml.v3@v3.0.1
```

**CGO Requirement:** Tree-sitter requires `CGO_ENABLED=1`. This is the default on macOS/Linux but must be explicit for cross-compilation. The v1 binary was pure Go; v2 introduces a CGO dependency.

## Architecture Patterns

### Recommended Project Structure Changes
```
internal/
  analyzer/
    c1_codehealth.go       # MODIFIED: accept []*AnalysisTarget
    c2_semantics.go        # NEW: C2 analyzer (dispatches per language)
    c2_go.go               # NEW: Go-specific C2 metrics
    c2_python.go           # NEW: Python-specific C2 metrics
    c2_typescript.go       # NEW: TypeScript-specific C2 metrics
    c3_architecture.go     # MODIFIED: accept []*AnalysisTarget
    c6_testing.go          # MODIFIED: accept []*AnalysisTarget
    helpers.go             # MODIFIED: add AnalysisTarget helpers
  config/
    config.go              # NEW: .arsrc.yml loader + validation
    config_test.go         # NEW
  discovery/
    walker.go              # MODIFIED: discover .py, .ts, .tsx files
    classifier.go          # MODIFIED: classify by language
  parser/
    parser.go              # MODIFIED: wrap output as []*AnalysisTarget
    treesitter.go          # NEW: Tree-sitter parser for Python/TS
    treesitter_test.go     # NEW
    queries/               # NEW: per-language Tree-sitter query files
      python.go            # Python S-expression queries as constants
      typescript.go        # TypeScript S-expression queries as constants
  pipeline/
    interfaces.go          # MODIFIED: AnalysisTarget-based interfaces
    pipeline.go            # MODIFIED: multi-parser orchestration
  scoring/
    config.go              # MODIFIED: add C2 category, expand to 7 cats
    scorer.go              # MODIFIED: add scoreC2, registry pattern
  output/
    terminal.go            # MODIFIED: render C2, per-language details
    json.go                # MODIFIED: include C2
pkg/types/
  types.go                 # MODIFIED: add AnalysisTarget, Language, C2Metrics
  scoring.go               # unchanged
testdata/
  valid-python-project/    # NEW: Python test fixtures
  valid-ts-project/        # NEW: TypeScript test fixtures
  polyglot-project/        # NEW: mixed-language test fixtures
```

### Pattern 1: Language-Agnostic AnalysisTarget with Language-Specific Extensions
**What:** Define a unified `AnalysisTarget` struct that all parsers produce and all analyzers consume, with optional language-specific fields for deeper analysis.
**When to use:** Always -- this is the core abstraction for multi-language support.
**Example:**
```go
// pkg/types/types.go

type Language string

const (
    LangGo         Language = "go"
    LangPython     Language = "python"
    LangTypeScript Language = "typescript"
)

// AnalysisTarget is the language-agnostic unit of analysis.
type AnalysisTarget struct {
    Language  Language
    RootDir   string       // Project root
    Files     []SourceFile // Source files for this language
}

type SourceFile struct {
    Path     string
    RelPath  string
    Language Language
    Lines    int
    Content  []byte       // Raw source (needed for Tree-sitter)
    Class    FileClass    // source, test, generated, excluded
}
```

**Key design decision:** Do NOT put Go-specific `*parser.ParsedPackage` on AnalysisTarget. Instead, Go analyzers receive AnalysisTargets and ALSO get access to ParsedPackages via a separate channel. This keeps AnalysisTarget clean. The pipeline passes both data structures to analyzers that need them.

### Pattern 2: Analyzer Interface with AnalysisTarget
**What:** Update the Analyzer interface to accept `[]*AnalysisTarget` instead of `[]*parser.ParsedPackage`.
**Example:**
```go
// pipeline/interfaces.go

type Analyzer interface {
    Name() string
    Analyze(targets []*types.AnalysisTarget) (*types.AnalysisResult, error)
}

// For Go analyzers that need ParsedPackage data:
type GoAwareAnalyzer interface {
    Analyzer
    SetGoPackages(pkgs []*parser.ParsedPackage)
}
```

Existing C1/C3/C6 analyzers implement `GoAwareAnalyzer`. The pipeline calls `SetGoPackages()` after Go parsing, then calls `Analyze()` with all targets. New analyzers (C2 for Python/TS) only implement `Analyzer`.

### Pattern 3: Per-Language C2 Dispatch
**What:** C2 analyzer dispatches to language-specific implementations.
**Example:**
```go
// analyzer/c2_semantics.go

type C2Analyzer struct {
    goAnalyzer *C2GoAnalyzer
    pyAnalyzer *C2PythonAnalyzer
    tsAnalyzer *C2TypeScriptAnalyzer
}

func (a *C2Analyzer) Analyze(targets []*types.AnalysisTarget) (*types.AnalysisResult, error) {
    metrics := &types.C2Metrics{
        PerLanguage: make(map[types.Language]*types.C2LanguageMetrics),
    }

    for _, target := range targets {
        switch target.Language {
        case types.LangGo:
            lm, err := a.goAnalyzer.Analyze(target)
            // ...
        case types.LangPython:
            lm, err := a.pyAnalyzer.Analyze(target)
            // ...
        case types.LangTypeScript:
            lm, err := a.tsAnalyzer.Analyze(target)
            // ...
        }
        metrics.PerLanguage[target.Language] = lm
    }

    // Aggregate across languages (weighted by LOC proportion)
    metrics.Aggregate = aggregateC2(metrics.PerLanguage)
    return &types.AnalysisResult{
        Name:     "C2: Semantic Explicitness",
        Category: "C2",
        Metrics:  map[string]interface{}{"c2": metrics},
    }, nil
}
```

### Pattern 4: Tree-sitter Parser Pooling
**What:** Create one Tree-sitter Parser per language, reuse across files.
**Example:**
```go
// parser/treesitter.go

type TreeSitterParser struct {
    pythonParser *tree_sitter.Parser
    tsParser     *tree_sitter.Parser
}

func NewTreeSitterParser() (*TreeSitterParser, error) {
    pyParser := tree_sitter.NewParser()
    if err := pyParser.SetLanguage(tree_sitter.NewLanguage(
        tree_sitter_python.Language(),
    )); err != nil {
        pyParser.Close()
        return nil, fmt.Errorf("set python language: %w", err)
    }

    tsParser := tree_sitter.NewParser()
    if err := tsParser.SetLanguage(tree_sitter.NewLanguage(
        tree_sitter_typescript.LanguageTypescript(),
    )); err != nil {
        pyParser.Close()
        tsParser.Close()
        return nil, fmt.Errorf("set typescript language: %w", err)
    }

    return &TreeSitterParser{
        pythonParser: pyParser,
        tsParser:     tsParser,
    }, nil
}

func (p *TreeSitterParser) Close() {
    p.pythonParser.Close()
    p.tsParser.Close()
}

func (p *TreeSitterParser) ParseFile(lang types.Language, content []byte) (*tree_sitter.Tree, error) {
    var parser *tree_sitter.Parser
    switch lang {
    case types.LangPython:
        parser = p.pythonParser
    case types.LangTypeScript:
        parser = p.tsParser
    }
    tree := parser.Parse(content, nil)
    return tree, nil // caller must defer tree.Close()
}
```

### Pattern 5: Config Loading with Validation
**What:** Load `.arsrc.yml` early in CLI, validate schema, merge with defaults.
**Example:**
```go
// config/config.go

type Config struct {
    Version    int                       `yaml:"version"`
    Scoring    ScoringOverrides          `yaml:"scoring"`
    Languages  []string                  `yaml:"languages"`
    Thresholds map[string]MetricOverride `yaml:"thresholds"`
}

type ScoringOverrides struct {
    Weights    map[string]float64 `yaml:"weights"`    // category -> weight
    Threshold  float64            `yaml:"threshold"`
}

type MetricOverride struct {
    Enabled    *bool              `yaml:"enabled"`
    Thresholds map[string]float64 `yaml:"thresholds"` // metric -> value
}

func Load(dir string) (*Config, error) {
    path := filepath.Join(dir, ".arsrc.yml")
    data, err := os.ReadFile(path)
    if os.IsNotExist(err) {
        return DefaultConfig(), nil
    }
    if err != nil {
        return nil, fmt.Errorf("read config: %w", err)
    }

    // Use strict decoding to catch typos
    cfg := DefaultConfig()
    dec := yaml.NewDecoder(bytes.NewReader(data))
    dec.KnownFields(true) // reject unknown keys
    if err := dec.Decode(cfg); err != nil {
        return nil, fmt.Errorf("invalid .arsrc.yml: %w", err)
    }

    if err := cfg.Validate(); err != nil {
        return nil, fmt.Errorf("config validation: %w", err)
    }

    return cfg, nil
}
```

### Anti-Patterns to Avoid
- **Lowest common denominator representation:** Do NOT remove `ParsedPackage` access from Go analyzers. Go's type info (go/types) provides capabilities Tree-sitter cannot. Keep language-specific extensions available.
- **Replacing go/packages with Tree-sitter for Go:** Tree-sitter gives syntax-only AST for Go. No type info, no import resolution, no cross-package analysis. Would regress C1/C3/C6 for Go.
- **One parser per file:** Creating a new Tree-sitter Parser for each file adds ~10ms CGO overhead per file. Pool one parser per language.
- **Language-specific pipelines:** Do NOT create separate pipelines for Go, Python, TypeScript. One pipeline, multiple parsers, merged results at the AnalysisTarget level.
- **Shoehorning Tree-sitter nodes into ParsedPackage:** Tree-sitter CST nodes are fundamentally different from Go AST. Do not add `TreeSitterNode` fields to `ParsedPackage`. Keep them separate.

## Don't Hand-Roll

Problems that look simple but have existing solutions:

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Python parsing | Regex-based Python parser | Tree-sitter with tree-sitter-python grammar | Python syntax is complex (decorators, async, type unions). Regex will miss edge cases. Tree-sitter handles all Python 3.x syntax correctly. |
| TypeScript parsing | Regex or babel-based TS parser | Tree-sitter with tree-sitter-typescript grammar | TS has complex syntax (generics, decorators, enums). Tree-sitter provides complete grammar. Exports both TS and TSX parsers. |
| YAML parsing | Custom YAML parser or string splitting | `gopkg.in/yaml.v3` with `KnownFields(true)` | YAML has edge cases (anchors, multiline strings, type coercion). yaml.v3 is battle-tested and already in deps. |
| YAML schema validation | Manual field-by-field checking | `yaml.Decoder.KnownFields(true)` + custom `Validate()` method | KnownFields catches typos (unknown keys). Custom Validate() checks semantic constraints (weights sum, ranges). |
| Naming convention detection | Custom regex per convention | Unicode-aware categorization via `unicode.IsUpper/IsLower` | CamelCase/snake_case detection needs Unicode awareness. Go's stdlib handles this correctly. |
| Magic number detection in Python/TS | Custom numeric literal parser | Tree-sitter `(number)` and `(integer)` node captures | Tree-sitter already identifies numeric literal nodes in the CST. Query for them, check parent context. |

**Key insight:** Tree-sitter queries replace 80% of the custom parsing work. Write S-expression queries instead of walking the AST manually.

## Common Pitfalls

### Pitfall 1: Tree-sitter Memory Leaks via Missing Close() Calls
**What goes wrong:** Every `Parser`, `Tree`, `TreeCursor`, `Query`, and `QueryCursor` object allocates C memory. Forgetting `Close()` causes memory growth over time. On a 10k-file scan, leaked Trees alone can consume 100MB+.
**Why it happens:** The official go-tree-sitter bindings deliberately do NOT use `runtime.SetFinalizer` due to CGO bugs. Memory management is entirely manual.
**How to avoid:** Use `defer obj.Close()` immediately after creation. Never pass Tree objects to goroutines without clear ownership. The `TreeSitterParser.ParseFile()` method returns a `*Tree` -- caller MUST defer `tree.Close()`.
**Warning signs:** RSS memory grows linearly with file count during scans. `go tool pprof` shows C memory not tracked by Go GC.

### Pitfall 2: Tree-sitter Node Types Differ Per Language
**What goes wrong:** A "function definition" is `function_definition` in Python, `function_declaration` in TypeScript, `function_declaration` in Go's Tree-sitter grammar. Writing generic queries that assume the same node type names fails silently (zero matches).
**Why it happens:** Each Tree-sitter grammar defines its own node types based on the language specification.
**How to avoid:** Maintain per-language query constants. Do NOT try to write a single "universal function finder" query. The queries are short (3-5 lines each) -- duplication is acceptable and clearer than abstraction.
**Warning signs:** C2 metrics return 0 for a language. Query captures return empty results.

### Pitfall 3: Tree-sitter Cannot Detect Implicit Types
**What goes wrong:** For TypeScript, Tree-sitter can detect explicit `any` annotations (`let x: any`) but cannot detect implicit `any` (missing type annotations that TypeScript infers as `any` under `noImplicitAny`). This means C2 type coverage for TS measures explicit annotation presence, not full type safety.
**Why it happens:** Tree-sitter is a syntax-only parser. It builds a CST from source text but performs no type inference or type checking.
**How to avoid:** Document clearly that C2 for Python/TS measures "type annotation coverage" (syntactic), not "type correctness" (semantic). This is the appropriate scope -- we are measuring whether developers bothered to add types, not whether the types are correct. For TS, additionally check tsconfig.json `strict` flags as a proxy for type safety intent.
**Warning signs:** Users ask "why does my fully typed TS project get low C2 scores?" -- answer: check if types are explicit or inferred.

### Pitfall 4: ScoringConfig Hardcoded to 3 Categories
**What goes wrong:** The current `ScoringConfig` struct has hardcoded fields `C1`, `C3`, `C6 CategoryConfig`. Adding C2 requires either adding `C2 CategoryConfig` (and later C4, C5, C7), or refactoring to a map/slice.
**Why it happens:** v1 only needed 3 categories, so hardcoding was simpler.
**How to avoid:** Refactor `ScoringConfig` to use a map or slice of `CategoryConfig`: `Categories map[string]CategoryConfig`. This makes adding new categories trivial and supports user-configurable categories in `.arsrc.yml`. The scorer dispatches by category name from the map.
**Warning signs:** Adding C2 requires changing ScoringConfig struct, DefaultConfig(), LoadConfig(), Score(), scoreC1/C3/C6 methods, and all tests.

### Pitfall 5: Discovery Walker Only Finds .go Files
**What goes wrong:** The current `walker.go` has `if !strings.HasSuffix(name, ".go") { return nil }` which silently skips all non-Go files. Multi-language discovery requires processing .py, .ts, .tsx files.
**Why it happens:** v1 was Go-only.
**How to avoid:** Extend the walker to accept a set of target extensions. Add a `Language` field to `DiscoveredFile`. Update `ScanResult` to include per-language counts. The classifier needs language-specific logic (e.g., Python test files are `test_*.py` or `*_test.py`, not `*_test.go`).
**Warning signs:** `ars scan` on a Python project reports 0 files found.

### Pitfall 6: Config Typos Silently Ignored
**What goes wrong:** User writes `wieght: 0.3` (typo) in `.arsrc.yml`. Standard yaml.Unmarshal silently ignores unknown fields, so the typo has no effect and the user thinks they configured something.
**Why it happens:** `yaml.Unmarshal` default behavior is lenient.
**How to avoid:** Use `yaml.NewDecoder()` with `dec.KnownFields(true)` which returns an error for unrecognized keys. Also validate semantic constraints: weights must be >= 0, weights should normalize to 1.0, thresholds must be in valid ranges.
**Warning signs:** User reports that config changes have no effect.

### Pitfall 7: Weight Rebalancing When Categories Unavailable
**What goes wrong:** PRD weights sum to 1.0 across all 7 categories. In Phase 6, only C1/C2/C3/C6 are available. If we use PRD weights (C1=0.25, C2=0.10, C3=0.20, C6=0.15), they sum to 0.70, not 1.0. The composite score calculation must normalize by active weight sum, or scores will be deflated.
**Why it happens:** The existing `computeComposite()` already normalizes by active weight sum. But config overrides could break this if users set weights that assume all 7 categories are present.
**How to avoid:** The existing scorer already handles this correctly (divides by totalWeight of active categories). Document this behavior clearly. Warn users if they set custom weights but not all categories are enabled.

## Code Examples

Verified patterns from official sources:

### Tree-sitter Python: Detect Functions With/Without Type Annotations
```go
// Source: tree-sitter query syntax docs + tree-sitter-python grammar
// Finds functions with return type annotation
const pyAnnotatedFuncQuery = `
(function_definition
  name: (identifier) @func.name
  return_type: (type) @return.type) @annotated
`

// Finds functions without return type annotation
const pyUnannotatedFuncQuery = `
(function_definition
  name: (identifier) @func.name
  !return_type) @unannotated
`

// Finds typed parameters
const pyTypedParamQuery = `
(function_definition
  parameters: (parameters
    (typed_parameter
      (identifier) @param.name
      type: (type) @param.type)))
`

// Finds untyped parameters (plain identifier, not self/cls)
const pyUntypedParamQuery = `
(function_definition
  parameters: (parameters
    (identifier) @param.untyped))
`
```

### Tree-sitter TypeScript: Detect `any` Usage and Missing Types
```go
// Source: tree-sitter-typescript grammar
// Finds explicit 'any' type annotations
const tsAnyTypeQuery = `
(type_identifier) @type
(#eq? @type "any")
`

// Finds function declarations without return type
const tsNoReturnTypeQuery = `
(function_declaration
  name: (identifier) @func.name
  !return_type) @no_return
`

// Finds arrow functions without return type
const tsArrowNoReturnQuery = `
(arrow_function
  !return_type) @arrow_no_return
`
```

### Tree-sitter: Magic Number Detection (Python)
```go
// Source: tree-sitter-python grammar
// Finds numeric literals not inside assignments to UPPER_CASE names
const pyMagicNumberQuery = `
(expression_statement
  (assignment
    left: (identifier) @name
    right: (integer) @value))

(integer) @magic_candidate
`
// Post-processing: filter out numbers in const-like contexts
// (UPPER_CASE variable assignments, enum members, common exceptions 0/1/-1)
```

### Using Tree-sitter Queries in Go
```go
// Source: tree-sitter/go-tree-sitter README
func countAnnotatedFunctions(parser *tree_sitter.Parser, content []byte) (int, int, error) {
    tree := parser.Parse(content, nil)
    defer tree.Close()

    lang := tree.Language()

    annotatedQuery, err := tree_sitter.NewQuery(lang, pyAnnotatedFuncQuery)
    if err != nil {
        return 0, 0, fmt.Errorf("compile query: %w", err)
    }
    defer annotatedQuery.Close()

    cursor := tree_sitter.NewQueryCursor()
    defer cursor.Close()

    cursor.Exec(annotatedQuery, tree.RootNode())
    annotated := 0
    for {
        match, ok := cursor.NextMatch()
        if !ok {
            break
        }
        _ = match
        annotated++
    }

    // Repeat for unannotated query...
    return annotated, unannotated, nil
}
```

### .arsrc.yml Config Structure
```yaml
# .arsrc.yml -- ARS configuration
version: 1

scoring:
  weights:
    c1: 0.25
    c2: 0.10
    c3: 0.20
    c4: 0.15
    c5: 0.10
    c6: 0.15
    c7: 0.05
  threshold: 7.0

thresholds:
  c2:
    type_annotation_coverage:
      python: 0.80  # 80% of functions must have type annotations
      typescript: 0.90
    naming_consistency: 0.95
    magic_number_ratio: 0.05

metrics:
  c2:
    enabled: true
    magic_numbers:
      ignore_values: [0, 1, -1, 2, 100]  # common acceptable literals
```

### Go C2: Interface{}/Any Usage Detection
```go
// Uses existing go/ast infrastructure (no Tree-sitter needed for Go)
func countAnyUsage(pkgs []*parser.ParsedPackage) (anyCount, totalTypes int) {
    for _, pkg := range pkgs {
        for _, file := range pkg.Syntax {
            ast.Inspect(file, func(n ast.Node) bool {
                switch x := n.(type) {
                case *ast.InterfaceType:
                    if len(x.Methods.List) == 0 {
                        anyCount++ // empty interface = interface{} = any
                    }
                    totalTypes++
                case *ast.Ident:
                    if x.Name == "any" {
                        anyCount++
                        totalTypes++
                    }
                }
                return true
            })
        }
    }
    return
}
```

### tsconfig.json Strict Mode Detection
```go
// Read and parse tsconfig.json for strict mode flags
func detectTSStrictMode(rootDir string) (bool, map[string]bool) {
    data, err := os.ReadFile(filepath.Join(rootDir, "tsconfig.json"))
    if err != nil {
        return false, nil
    }

    var tsconfig struct {
        CompilerOptions struct {
            Strict           *bool `json:"strict"`
            StrictNullChecks *bool `json:"strictNullChecks"`
            NoImplicitAny    *bool `json:"noImplicitAny"`
        } `json:"compilerOptions"`
    }

    if err := json.Unmarshal(data, &tsconfig); err != nil {
        return false, nil
    }

    flags := map[string]bool{}
    if tsconfig.CompilerOptions.Strict != nil {
        flags["strict"] = *tsconfig.CompilerOptions.Strict
    }
    if tsconfig.CompilerOptions.StrictNullChecks != nil {
        flags["strictNullChecks"] = *tsconfig.CompilerOptions.StrictNullChecks
    }
    if tsconfig.CompilerOptions.NoImplicitAny != nil {
        flags["noImplicitAny"] = *tsconfig.CompilerOptions.NoImplicitAny
    }

    isStrict := flags["strict"]
    return isStrict, flags
}
```

## C2 Metric Definitions Per Language

Critical for planning: what each C2 metric actually measures per language.

### Type Annotation Coverage (C2-PY-01, C2-TS-01)
| Language | What is Measured | How | Score 10 | Score 1 |
|----------|-----------------|-----|----------|---------|
| Go | N/A (Go is statically typed -- always 100%) | Skip this metric | Always 10 | Always 10 |
| Python | % of function params + return types with type annotations | Tree-sitter: count `typed_parameter` vs `identifier` params; count functions with vs without `return_type` | 100% annotated | <30% annotated |
| TypeScript | % of functions with explicit return types + % of params with types (excluding `any`) | Tree-sitter: count functions with/without `return_type`; count explicit `any` type identifiers | >95% with types, 0% `any` | <50% with types or >20% `any` |

### Naming Consistency (C2-GO-02, C2-PY-02, C2-TS-02)
| Language | Convention | How Measured |
|----------|-----------|-------------|
| Go | Exported: CamelCase starting with uppercase. Unexported: camelCase starting with lowercase | `go/ast`: check `ast.Ident.IsExported()` vs naming pattern |
| Python | snake_case for functions/variables, CamelCase for classes (PEP 8) | Tree-sitter: capture `identifier` nodes, check context (function_definition name vs class_definition name) |
| TypeScript | camelCase for functions/variables, PascalCase for classes/interfaces | Tree-sitter: capture identifier nodes, check parent node type |

### Magic Numbers (C2-GO-03, C2-PY-03, C2-TS-03)
| Language | Detection Method | Exceptions |
|----------|-----------------|------------|
| Go | `go/ast.BasicLit` with `Kind == token.INT/FLOAT`, check parent is not `const` | 0, 1, -1; array sizes; time durations |
| Python | Tree-sitter `(integer)` and `(float)` nodes not in `UPPER_CASE =` assignments | 0, 1, -1; common test values |
| TypeScript | Tree-sitter `(number)` nodes not in `const` declarations or enum members | 0, 1, -1; array indices |

### Language-Specific Metrics
| Metric | Language | Detection |
|--------|----------|-----------|
| interface{}/any usage (C2-GO-01) | Go | `go/ast`: empty interface types + `any` identifiers |
| nil safety (C2-GO-04) | Go | `go/ast`: pointer dereferences without preceding nil check in same scope |
| mypy/pyright config (C2-PY-04) | Python | File existence: `mypy.ini`, `setup.cfg [mypy]`, `pyproject.toml [tool.mypy]`, `pyrightconfig.json` |
| tsconfig strict (C2-TS-02) | TypeScript | Parse `tsconfig.json`, check `strict`, `strictNullChecks`, `noImplicitAny` |
| null safety (C2-TS-04) | TypeScript | Tree-sitter: count optional chaining `?.` usage; check `strictNullChecks` in tsconfig |

## Scoring Expansion

### PRD Category Weights (all 7 categories)
```
C1: 0.25 (Code Health)
C2: 0.10 (Semantic Explicitness)  <-- NEW in this phase
C3: 0.20 (Architecture)
C4: 0.15 (Documentation)          <-- Future phase
C5: 0.10 (Temporal Dynamics)       <-- Future phase
C6: 0.15 (Testing)
C7: 0.05 (Agent Evaluation)       <-- Future phase
```

### Phase 6 Active Categories
After Phase 6, categories C1, C2, C3, C6 are active (weights: 0.25 + 0.10 + 0.20 + 0.15 = 0.70). The existing `computeComposite()` normalizes by active weight sum, so composite scores remain on a 1-10 scale.

### ScoringConfig Refactoring
The current `ScoringConfig` has hardcoded `C1`, `C3`, `C6` fields. Refactor to:
```go
type ScoringConfig struct {
    Categories map[string]CategoryConfig `yaml:"categories"`
    Tiers      []TierConfig              `yaml:"tiers"`
}
```

This supports any number of categories without struct changes.

### C2 Default Scoring Breakpoints
```go
CategoryConfig{
    Name:   "Semantic Explicitness",
    Weight: 0.10,
    Metrics: []MetricThresholds{
        {Name: "type_annotation_coverage", Weight: 0.30, Breakpoints: []Breakpoint{
            {Value: 100, Score: 10}, {Value: 80, Score: 8},
            {Value: 50, Score: 6}, {Value: 30, Score: 3}, {Value: 0, Score: 1},
        }},
        {Name: "naming_consistency", Weight: 0.25, Breakpoints: []Breakpoint{
            {Value: 100, Score: 10}, {Value: 95, Score: 8},
            {Value: 85, Score: 6}, {Value: 70, Score: 3}, {Value: 0, Score: 1},
        }},
        {Name: "magic_number_ratio", Weight: 0.20, Breakpoints: []Breakpoint{
            {Value: 0, Score: 10}, {Value: 5, Score: 8},
            {Value: 15, Score: 6}, {Value: 30, Score: 3}, {Value: 50, Score: 1},
        }},
        {Name: "type_strictness", Weight: 0.15, Breakpoints: []Breakpoint{
            // Binary: strict mode on = 10, off = 3
            {Value: 0, Score: 3}, {Value: 1, Score: 10},
        }},
        {Name: "null_safety", Weight: 0.10, Breakpoints: []Breakpoint{
            {Value: 100, Score: 10}, {Value: 80, Score: 8},
            {Value: 50, Score: 6}, {Value: 30, Score: 3}, {Value: 0, Score: 1},
        }},
    },
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `smacker/go-tree-sitter` (community) | `tree-sitter/go-tree-sitter` (official) | Feb 2025 | Official bindings are now the standard. Separate grammar packages = smaller binaries. |
| `runtime.SetFinalizer` for CGO cleanup | Explicit `Close()` calls required | Feb 2025 | Prevents CGO finalizer bugs. More verbose but safer. |
| go-git for all git ops | Native git CLI for performance-critical ops | 2024-2025 | go-git blame/log is 10-35x slower than native git. Native git recommended for C5. |

**Deprecated/outdated:**
- `smacker/go-tree-sitter`: Superseded by official `tree-sitter/go-tree-sitter`. Do not use for new projects.

## Open Questions

Things that couldn't be fully resolved:

1. **Tree-sitter TSX vs TypeScript grammar selection**
   - What we know: tree-sitter-typescript exports two separate language functions: `LanguageTypescript()` and `LanguageTsx()`. TSX files (.tsx) need the TSX grammar.
   - What's unclear: Do we need two Tree-sitter parsers (one for TS, one for TSX), or can the TSX grammar handle plain .ts files too?
   - Recommendation: Use file extension to select grammar. `.ts` files get TypeScript grammar, `.tsx` files get TSX grammar. Create two parsers if needed (minimal overhead since parsers are pooled).

2. **C2 Go nil safety analysis depth**
   - What we know: Requirement C2-GO-04 asks for "nil safety patterns (nil checks before dereference)". Full nil-safety analysis requires control flow analysis.
   - What's unclear: How deep should the analysis go? Simple pattern matching (nil check in same block) vs. full data flow analysis.
   - Recommendation: Start with simple heuristic: count pointer dereferences and count nil checks. Report ratio. Do NOT build a full data flow analyzer -- that is out of scope for a 30-second tool.

3. **Per-language C2 score aggregation**
   - What we know: A polyglot repo with Go, Python, and TypeScript produces three language-specific C2 scores.
   - What's unclear: How to aggregate into a single C2 category score. Options: weighted by LOC proportion, or take the minimum (weakest link).
   - Recommendation: Weighted average by LOC proportion. This matches intuition: a repo that is 90% well-typed Go and 10% untyped Python should score high, not low.

## Sources

### Primary (HIGH confidence)
- [tree-sitter/go-tree-sitter GitHub](https://github.com/tree-sitter/go-tree-sitter) -- API, memory management requirements, Close() mandate
- [tree-sitter/go-tree-sitter pkg.go.dev](https://pkg.go.dev/github.com/tree-sitter/go-tree-sitter) -- v0.25.0, official Go bindings
- [tree-sitter-typescript Go bindings](https://pkg.go.dev/github.com/tree-sitter/tree-sitter-typescript/bindings/go) -- TSX and TypeScript dual grammar
- [tree-sitter-python GitHub](https://github.com/tree-sitter/tree-sitter-python) -- node-types.json for function_definition, typed_parameter
- [tree-sitter query syntax](https://tree-sitter.github.io/tree-sitter/using-parsers/queries/1-syntax.html) -- S-expression patterns, predicates
- [gopkg.in/yaml.v3 pkg.go.dev](https://pkg.go.dev/gopkg.in/yaml.v3) -- v3.0.1, KnownFields, Decoder API
- Existing codebase analysis (pipeline, interfaces, scoring, analyzers)

### Secondary (MEDIUM confidence)
- [TypeScript tsconfig strict reference](https://www.typescriptlang.org/tsconfig/) -- strict mode sub-flags verified
- v2 architecture research (`.planning/research/ARCHITECTURE.md`) -- dual-parser strategy, AnalysisTarget pattern
- v2 stack research (`.planning/research/STACK.md`) -- library selections verified
- v2 pitfalls research (`.planning/research/PITFALLS.md`) -- CGO memory, Tree-sitter limitations

### Tertiary (LOW confidence)
- Tree-sitter TSX grammar handling of plain .ts files -- needs validation during implementation

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - official tree-sitter bindings verified via GitHub + pkg.go.dev; yaml.v3 already in deps
- Architecture: HIGH - based on direct analysis of v1 codebase + v2 architecture research
- C2 metrics: HIGH for Go (uses existing go/ast); MEDIUM for Python/TS (Tree-sitter queries need implementation validation)
- Config system: HIGH - standard yaml.v3 pattern with KnownFields validation
- Scoring expansion: HIGH - existing scorer pattern extends naturally to map-based categories
- Pitfalls: HIGH - documented from prior research + verified against official tree-sitter docs

**Research date:** 2026-02-01
**Valid until:** 2026-03-01 (stable domain; tree-sitter bindings are mature)
