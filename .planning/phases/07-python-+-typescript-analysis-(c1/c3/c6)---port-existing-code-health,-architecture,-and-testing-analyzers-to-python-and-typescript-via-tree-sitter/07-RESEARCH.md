# Phase 7: Python + TypeScript Analysis (C1/C3/C6) - Research

**Researched:** 2026-02-01
**Domain:** Tree-sitter AST analysis for C1 (Code Health), C3 (Architecture), C6 (Testing) across Python and TypeScript
**Confidence:** HIGH

## Summary

This phase extends the three core analysis categories (C1, C3, C6) from Go-only to Python and TypeScript. The existing Go analyzers are tightly coupled to `go/ast` and `go/packages` types -- they cannot be directly reused. Instead, new language-specific analyzer implementations must be created that use Tree-sitter ASTs to compute equivalent metrics, then feed into the same `types.C1Metrics`, `types.C3Metrics`, and `types.C6Metrics` structures.

The codebase already has a working Tree-sitter integration from Phase 6: `parser.TreeSitterParser` handles Python/TypeScript/TSX parsing, `parser.ParsedTreeSitterFile` holds parsed trees, and `c2_python.go`/`c2_typescript.go` demonstrate the pattern of walking Tree-sitter nodes with `walkTree()` and `nodeText()` helpers. The C2 dispatcher pattern (`c2_semantics.go`) shows how to route analysis to language-specific implementations while producing unified results. This same pattern should be applied to C1, C3, and C6.

The key architectural decision is how to restructure C1/C3/C6 analyzers. Currently they implement `GoAwareAnalyzer` and only use `SetGoPackages`. They must be refactored to also accept `[]*types.AnalysisTarget` for Python/TypeScript, dispatching to language-specific implementations while preserving Go analysis via `SetGoPackages`. The C2 analyzer's dispatch pattern is the proven template.

**Primary recommendation:** Follow the C2 dispatcher pattern exactly -- create `c1_python.go`, `c1_typescript.go`, `c3_python.go`, `c3_typescript.go`, `c6_python.go`, `c6_typescript.go` files, refactor the main C1/C3/C6 analyzers to dispatch per-language, and aggregate results.

## Standard Stack

The established libraries/tools for this domain:

### Core (Already in go.mod)
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `tree-sitter/go-tree-sitter` | v0.25.0 | Tree-sitter Go bindings | Already integrated. Provides Parser, Tree, Node for AST walking. |
| `tree-sitter/tree-sitter-python` | v0.25.0 | Python grammar | Already integrated. Covers all Python 3.x syntax. |
| `tree-sitter/tree-sitter-typescript` | v0.23.2 | TypeScript + TSX grammar | Already integrated. Two grammars: LanguageTypescript() and LanguageTSX(). |
| `fzipp/gocyclo` | v0.6.0 | Go cyclomatic complexity | Stays for Go only. Python/TS complexity computed via Tree-sitter. |

### Supporting (Already in go.mod)
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `golang.org/x/tools/cover` | v0.41.0 | Go coverage profile parsing | Go coverage only. Python uses coverage.py XML, TS uses LCOV. |
| `hash/fnv` | stdlib | FNV hashing for duplication | Reuse same hash approach for Tree-sitter token-based duplication. |

### No New Dependencies
No new libraries are needed. All functionality can be built using Tree-sitter AST walking (already integrated) and Go stdlib.

## Architecture Patterns

### Recommended File Structure
```
internal/analyzer/
  c1_codehealth.go         # REFACTORED: dispatch to Go vs Python/TS
  c1_python.go             # NEW: Python C1 via Tree-sitter
  c1_python_test.go        # NEW
  c1_typescript.go         # NEW: TypeScript C1 via Tree-sitter
  c1_typescript_test.go    # NEW
  c3_architecture.go       # REFACTORED: dispatch to Go vs Python/TS
  c3_python.go             # NEW: Python C3 via Tree-sitter
  c3_python_test.go        # NEW
  c3_typescript.go         # NEW: TypeScript C3 via Tree-sitter
  c3_typescript_test.go    # NEW
  c6_testing.go            # REFACTORED: dispatch to Go vs Python/TS
  c6_python.go             # NEW: Python C6 (pytest/unittest detection, coverage.py)
  c6_python_test.go        # NEW
  c6_typescript.go         # NEW: TypeScript C6 (Jest/Vitest/Mocha, LCOV)
  c6_typescript_test.go    # NEW
  helpers.go               # EXTENDED: add Tree-sitter import graph builder
testdata/
  valid-python-project/    # EXTENDED: add complex Python for C1/C3/C6 testing
  valid-ts-project/        # EXTENDED: add complex TypeScript for C1/C3/C6 testing
```

### Pattern 1: C2-Style Language Dispatcher (Proven Pattern)
**What:** Each category analyzer dispatches to language-specific implementations, then merges results.
**When to use:** All C1, C3, C6 refactoring.
**Example:**
```go
// Source: existing c2_semantics.go pattern
type C1Analyzer struct {
    pkgs       []*parser.ParsedPackage   // Go packages (from SetGoPackages)
    tsParser   *parser.TreeSitterParser  // Tree-sitter for Python/TS
}

func NewC1Analyzer(tsParser *parser.TreeSitterParser) *C1Analyzer {
    return &C1Analyzer{tsParser: tsParser}
}

func (a *C1Analyzer) SetGoPackages(pkgs []*parser.ParsedPackage) {
    a.pkgs = pkgs
}

func (a *C1Analyzer) Analyze(targets []*types.AnalysisTarget) (*types.AnalysisResult, error) {
    // Go analysis: use a.pkgs (existing logic, unchanged)
    goMetrics := a.analyzeGo()

    // Python/TS analysis: iterate targets, dispatch per language
    for _, target := range targets {
        switch target.Language {
        case types.LangPython:
            pyMetrics := a.analyzePython(target)
            // merge into combined metrics
        case types.LangTypeScript:
            tsMetrics := a.analyzeTypeScript(target)
            // merge into combined metrics
        }
    }

    return combinedResult, nil
}
```

### Pattern 2: Tree-sitter Cyclomatic Complexity via Node Walking
**What:** Count control flow nodes to compute cyclomatic complexity, using the same walkTree/nodeText helpers from C2.
**When to use:** C1 complexity for Python and TypeScript.
**Example:**
```go
// Python control flow nodes that add +1 complexity:
// if_statement, elif_clause, for_statement, while_statement,
// except_clause, case_clause, boolean_operator (and/or)
//
// TypeScript control flow nodes that add +1 complexity:
// if_statement, for_statement, for_in_statement, while_statement,
// do_statement, switch_case, catch_clause, ternary_expression,
// binary_expression (&&, ||), optional_chain (?.)

func pyComplexity(root *tree_sitter.Node, content []byte) int {
    complexity := 1 // base complexity
    walkTree(root, func(node *tree_sitter.Node) {
        switch node.Kind() {
        case "if_statement", "elif_clause", "for_statement",
             "while_statement", "except_clause", "case_clause":
            complexity++
        case "boolean_operator":
            // Python "and" / "or"
            complexity++
        }
    })
    return complexity
}
```

### Pattern 3: Tree-sitter Import Graph for Python/TypeScript
**What:** Build import graphs from Tree-sitter import nodes instead of Go's typed import info.
**When to use:** C3 architecture analysis for Python and TypeScript.
**Example:**
```go
// Python imports: import_statement, import_from_statement
// Extract module paths from import nodes
func pyBuildImportGraph(files []*parser.ParsedTreeSitterFile) *ImportGraph {
    graph := &ImportGraph{
        Forward: make(map[string][]string),
        Reverse: make(map[string][]string),
    }
    for _, f := range files {
        walkTree(f.Tree.RootNode(), func(node *tree_sitter.Node) {
            switch node.Kind() {
            case "import_statement":
                // "import foo.bar" -> module name from child
                name := node.ChildByFieldName("name")
                if name != nil {
                    target := nodeText(name, f.Content)
                    graph.Forward[f.RelPath] = append(graph.Forward[f.RelPath], target)
                    graph.Reverse[target] = append(graph.Reverse[target], f.RelPath)
                }
            case "import_from_statement":
                // "from foo.bar import baz"
                module := node.ChildByFieldName("module_name")
                if module != nil {
                    target := nodeText(module, f.Content)
                    graph.Forward[f.RelPath] = append(graph.Forward[f.RelPath], target)
                    graph.Reverse[target] = append(graph.Reverse[target], f.RelPath)
                }
            }
        })
    }
    return graph
}

// TypeScript imports: import_statement
// "import { X } from './module'" or "const X = require('./module')"
func tsBuildImportGraph(files []*parser.ParsedTreeSitterFile) *ImportGraph {
    graph := &ImportGraph{
        Forward: make(map[string][]string),
        Reverse: make(map[string][]string),
    }
    for _, f := range files {
        walkTree(f.Tree.RootNode(), func(node *tree_sitter.Node) {
            if node.Kind() == "import_statement" {
                source := node.ChildByFieldName("source")
                if source != nil {
                    target := strings.Trim(nodeText(source, f.Content), "\"'")
                    graph.Forward[f.RelPath] = append(graph.Forward[f.RelPath], target)
                    graph.Reverse[target] = append(graph.Reverse[target], f.RelPath)
                }
            }
            // Also handle require() calls for CommonJS
            if node.Kind() == "call_expression" {
                fn := node.ChildByFieldName("function")
                if fn != nil && nodeText(fn, f.Content) == "require" {
                    args := node.ChildByFieldName("arguments")
                    if args != nil && args.ChildCount() > 1 {
                        arg := args.Child(1) // skip "(" which is child 0
                        if arg != nil && arg.Kind() == "string" {
                            target := strings.Trim(nodeText(arg, f.Content), "\"'")
                            graph.Forward[f.RelPath] = append(graph.Forward[f.RelPath], target)
                        }
                    }
                }
            }
        })
    }
    return graph
}
```

### Pattern 4: Token-Based Duplication for Tree-sitter
**What:** Hash sequences of Tree-sitter child node types/structures (analogous to Go's AST statement hashing).
**When to use:** C1 duplication detection for Python and TypeScript.
**Example:**
```go
// Walk compound statement bodies (function_definition.body, if_statement.body, etc.)
// For each block, extract child statement nodes and hash sequences
func tsDuplication(files []*parser.ParsedTreeSitterFile) ([]types.DuplicateBlock, float64) {
    const minStatements = 3
    const minLines = 6

    type stmtSeq struct {
        hash      uint64
        file      string
        startLine int
        endLine   int
    }

    var sequences []stmtSeq
    totalLines := 0

    for _, f := range files {
        totalLines += countLines(f.Content)
        // Find all statement blocks and hash sliding windows
        walkTree(f.Tree.RootNode(), func(node *tree_sitter.Node) {
            if !isBlockNode(node) {
                return
            }
            // Collect statement children
            var stmts []*tree_sitter.Node
            for i := uint(0); i < node.ChildCount(); i++ {
                child := node.Child(i)
                if child != nil && child.IsNamed() {
                    stmts = append(stmts, child)
                }
            }
            // Sliding window hashing (same algorithm as Go's analyzeDuplication)
            for i := 0; i <= len(stmts)-minStatements; i++ {
                for ws := minStatements; ws <= len(stmts)-i; ws++ {
                    window := stmts[i : i+ws]
                    startRow := int(window[0].StartPosition().Row) + 1
                    endRow := int(window[len(window)-1].EndPosition().Row) + 1
                    if endRow-startRow+1 < minLines {
                        continue
                    }
                    h := hashTSStatements(window, f.Content)
                    sequences = append(sequences, stmtSeq{h, f.RelPath, startRow, endRow})
                }
            }
        })
    }
    // Group by hash and find duplicates (same logic as Go)
    // ...
}
```

### Anti-Patterns to Avoid
- **Modifying Go analysis paths:** The existing Go C1/C3/C6 logic using `go/ast` and `go/packages` must remain untouched. New code adds Python/TS paths alongside, never replaces.
- **Sharing analyzer state between languages:** Each language analysis should be independent. Do not try to build a combined import graph across Go + Python + TypeScript.
- **Forgetting Tree.Close():** Every `ParsedTreeSitterFile.Tree` must be closed after use. The existing `parser.CloseAll()` helper handles this.
- **Using Tree-sitter queries when node walking suffices:** The C2 analyzers already use `walkTree()` (not query language). Stay consistent -- queries add complexity for simple pattern matching.

## Don't Hand-Roll

Problems that look simple but have existing solutions:

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Python cyclomatic complexity | Custom recursive AST walker from scratch | `walkTree()` counting control flow nodes | Same walkTree helper from C2 works. Just count different node types. |
| Coverage parsing (Python) | Custom XML parser for coverage.py | Existing `parseCobertura()` in c6_testing.go | coverage.py outputs Cobertura XML format. The parser already exists. |
| Coverage parsing (TypeScript) | Custom LCOV parser | Existing `parseLCOV()` in c6_testing.go | Istanbul/NYC output LCOV. The parser already exists. |
| Tree-sitter file parsing | Per-file parser creation | `parser.TreeSitterParser.ParseTargetFiles()` | Handles .ts vs .tsx distinction, pooled parsers, error handling. |
| File classification (test vs source) | New classification logic | Existing `ClassifyPythonFile()` / `ClassifyTypeScriptFile()` in classifier.go | Already handles test_*.py, *_test.py, *.test.ts, *.spec.ts patterns. |
| Import graph data structure | New graph type | Existing `ImportGraph` struct in helpers.go | Same Forward/Reverse adjacency list, same cycle detection algorithm. |
| Duplication hashing | New hash scheme | FNV-based approach from existing `hashStatements` | Same sliding-window approach, adapted for Tree-sitter nodes. |

**Key insight:** Phase 6 built all the Tree-sitter infrastructure and C6 already has LCOV + Cobertura parsers. Phase 7 is about wiring new Tree-sitter node walkers to existing metric types and scoring, not building new infrastructure.

## Common Pitfalls

### Pitfall 1: Python Import Resolution Ambiguity
**What goes wrong:** Python imports like `from . import foo` or `from ..utils import bar` are relative and context-dependent. Resolving them to actual file paths requires knowing the package structure.
**Why it happens:** Python's import system is more complex than Go's (relative imports, __init__.py packages, namespace packages).
**How to avoid:** For C3 import graph analysis, use a simplified approach: (1) Map file paths to module names based on directory structure, (2) Resolve relative imports using the importing file's position, (3) Only track intra-project imports (ignore stdlib and third-party). Do not attempt to replicate Python's full import machinery.
**Warning signs:** Tests failing on projects with complex package hierarchies or namespace packages.

### Pitfall 2: TypeScript Path Aliases and Module Resolution
**What goes wrong:** TypeScript projects use `tsconfig.json` path aliases (e.g., `@/components/Button`), barrel files (`index.ts`), and various module resolution strategies.
**How to avoid:** Parse `tsconfig.json` for `paths` and `baseUrl` configuration. Map aliases to filesystem paths. For barrel files, count the barrel file itself as a module boundary. Do not attempt to replicate TypeScript's full module resolution algorithm.
**Warning signs:** Import graphs with many unresolved nodes.

### Pitfall 3: Python Decorator Complexity
**What goes wrong:** Per the CONTEXT.md decision, decorators like `@retry`, `@cached` add implicit complexity. But Tree-sitter just shows them as `decorator` nodes wrapping `function_definition`.
**Why it happens:** Decorators fundamentally change function behavior but their effect is not visible in the syntax tree.
**How to avoid:** Define a small set of known complexity-adding decorators (retry, cache, contextmanager, etc.) and add +1 complexity for each. For unknown decorators, add +0. Document this as a heuristic.
**Warning signs:** Functions with many decorators scoring the same as simple functions.

### Pitfall 4: Test Detection Needs Both File-Level and AST-Level Checks
**What goes wrong:** Files classified as `ClassTest` by filename patterns may not contain actual test functions (e.g., test helpers, conftest.py). Conversely, test functions may exist in files not matching test patterns (rare but possible).
**Why it happens:** File-level classification is a heuristic, not definitive.
**How to avoid:** For C6, use hybrid detection as specified in CONTEXT.md: file classification for initial filtering, then AST scanning for test function signatures. For Python: `def test_*` functions and classes inheriting `unittest.TestCase`. For TypeScript: `describe()`, `it()`, `test()` function calls.
**Warning signs:** Test-to-code ratio far from expected values on known projects.

### Pitfall 5: C1 Metrics Must Produce Same types.C1Metrics Structure
**What goes wrong:** Creating separate metric types for Python/TS that don't flow into the existing scoring pipeline.
**Why it happens:** Temptation to define PythonC1Metrics with different fields.
**How to avoid:** All language-specific C1 analysis MUST produce `types.C1Metrics` (same struct). The scoring extractor `extractC1()` expects exactly these fields. For fields that don't apply (e.g., AfferentCoupling/EfferentCoupling are Go-package-centric), either compute file-level equivalents or leave as zero/empty.
**Warning signs:** Scoring producing zero or panic for Python/TypeScript projects.

### Pitfall 6: Merging Multi-Language Metrics
**What goes wrong:** A polyglot project (Go + Python + TypeScript) should produce one C1 score, not three separate ones.
**Why it happens:** Each language produces its own metrics, but scoring expects one AnalysisResult per category.
**How to avoid:** Follow the C2 pattern: aggregate per-language metrics into a single result, weighted by LOC. The C1/C3/C6 analyzers should merge their language-specific results before returning.
**Warning signs:** Polyglot projects showing only one language's metrics.

## Code Examples

### Python Function Extraction for C1
```go
// Source: Pattern derived from c2_python.go walkTree usage
// Finds all function/method definitions and computes complexity + line count
func pyAnalyzeFunctions(files []*parser.ParsedTreeSitterFile) []types.FunctionMetric {
    var functions []types.FunctionMetric

    for _, f := range files {
        if f.Language != types.LangPython {
            continue
        }
        walkTree(f.Tree.RootNode(), func(node *tree_sitter.Node) {
            if node.Kind() != "function_definition" {
                return
            }

            nameNode := node.ChildByFieldName("name")
            if nameNode == nil {
                return
            }
            name := nodeText(nameNode, f.Content)

            // Check if method (parent is class body)
            parent := node.Parent()
            if parent != nil && parent.Parent() != nil {
                if parent.Parent().Kind() == "class_definition" {
                    classNameNode := parent.Parent().ChildByFieldName("name")
                    if classNameNode != nil {
                        name = nodeText(classNameNode, f.Content) + "." + name
                    }
                }
            }

            startRow := int(node.StartPosition().Row) + 1
            endRow := int(node.EndPosition().Row) + 1
            lineCount := endRow - startRow + 1

            // Compute complexity by counting control flow within this function's body
            body := node.ChildByFieldName("body")
            complexity := 1
            if body != nil {
                walkTree(body, func(n *tree_sitter.Node) {
                    switch n.Kind() {
                    case "if_statement", "elif_clause", "for_statement",
                         "while_statement", "except_clause", "case_clause":
                        complexity++
                    case "boolean_operator":
                        complexity++
                    case "conditional_expression": // ternary
                        complexity++
                    }
                })
            }

            functions = append(functions, types.FunctionMetric{
                Package:    filepath.Dir(f.RelPath),
                Name:       name,
                File:       f.Path,
                Line:       startRow,
                Complexity: complexity,
                LineCount:  lineCount,
            })
        })
    }
    return functions
}
```

### TypeScript Function Extraction for C1
```go
// TypeScript has multiple function forms
func tsAnalyzeFunctions(files []*parser.ParsedTreeSitterFile) []types.FunctionMetric {
    var functions []types.FunctionMetric

    for _, f := range files {
        if f.Language != types.LangTypeScript {
            continue
        }
        walkTree(f.Tree.RootNode(), func(node *tree_sitter.Node) {
            var name string
            var body *tree_sitter.Node

            switch node.Kind() {
            case "function_declaration":
                nameNode := node.ChildByFieldName("name")
                if nameNode != nil {
                    name = nodeText(nameNode, f.Content)
                }
                body = node.ChildByFieldName("body")

            case "method_definition":
                nameNode := node.ChildByFieldName("name")
                if nameNode != nil {
                    name = nodeText(nameNode, f.Content)
                }
                // Prefix with class name if available
                parent := node.Parent()
                if parent != nil && parent.Parent() != nil {
                    if parent.Parent().Kind() == "class_declaration" {
                        classNameNode := parent.Parent().ChildByFieldName("name")
                        if classNameNode != nil {
                            name = nodeText(classNameNode, f.Content) + "." + name
                        }
                    }
                }
                body = node.ChildByFieldName("body")

            case "arrow_function":
                // Named arrow functions: const foo = () => {}
                parent := node.Parent()
                if parent != nil && parent.Kind() == "variable_declarator" {
                    nameNode := parent.ChildByFieldName("name")
                    if nameNode != nil {
                        name = nodeText(nameNode, f.Content)
                    }
                }
                if name == "" {
                    name = "<anonymous>"
                }
                body = node.ChildByFieldName("body")

            default:
                return
            }

            if name == "" {
                return
            }

            startRow := int(node.StartPosition().Row) + 1
            endRow := int(node.EndPosition().Row) + 1
            lineCount := endRow - startRow + 1

            complexity := 1
            if body != nil {
                walkTree(body, func(n *tree_sitter.Node) {
                    switch n.Kind() {
                    case "if_statement", "for_statement", "for_in_statement",
                         "while_statement", "do_statement", "switch_case",
                         "catch_clause":
                        complexity++
                    case "ternary_expression":
                        complexity++
                    case "binary_expression":
                        op := n.ChildByFieldName("operator")
                        if op != nil {
                            opText := nodeText(op, f.Content)
                            if opText == "&&" || opText == "||" || opText == "??" {
                                complexity++
                            }
                        }
                    }
                    // Skip nested functions -- they have their own complexity
                    if n.Kind() == "function_declaration" || n.Kind() == "arrow_function" {
                        // Don't walk into nested functions
                        // Note: walkTree visits all children, so we need
                        // a more selective approach or subtract nested counts
                    }
                })
            }

            functions = append(functions, types.FunctionMetric{
                Package:    filepath.Dir(f.RelPath),
                Name:       name,
                File:       f.Path,
                Line:       startRow,
                Complexity: complexity,
                LineCount:  lineCount,
            })
        })
    }
    return functions
}
```

### Python Dead Code Detection (C3)
```go
// Simplified dead code: find all top-level definitions (functions, classes),
// then check which are never referenced in other files via import
func pyDetectDeadCode(files []*parser.ParsedTreeSitterFile) []types.DeadExport {
    type definition struct {
        name    string
        file    string
        line    int
        kind    string
        module  string // module path derived from file path
    }

    var defs []definition
    importedNames := make(map[string]map[string]bool) // module -> set of imported names

    for _, f := range files {
        modulePath := pyModulePath(f.RelPath)
        root := f.Tree.RootNode()

        // Collect top-level definitions
        for i := uint(0); i < root.ChildCount(); i++ {
            child := root.Child(i)
            if child == nil {
                continue
            }
            switch child.Kind() {
            case "function_definition":
                nameNode := child.ChildByFieldName("name")
                if nameNode != nil {
                    name := nodeText(nameNode, f.Content)
                    if !strings.HasPrefix(name, "_") { // Skip private
                        defs = append(defs, definition{
                            name: name, file: f.RelPath,
                            line: int(child.StartPosition().Row) + 1,
                            kind: "func", module: modulePath,
                        })
                    }
                }
            case "class_definition":
                nameNode := child.ChildByFieldName("name")
                if nameNode != nil {
                    name := nodeText(nameNode, f.Content)
                    if !strings.HasPrefix(name, "_") {
                        defs = append(defs, definition{
                            name: name, file: f.RelPath,
                            line: int(child.StartPosition().Row) + 1,
                            kind: "type", module: modulePath,
                        })
                    }
                }
            case "decorated_definition":
                // Look inside for function_definition or class_definition
                // ...
            }
        }

        // Collect imports (what each file uses from other modules)
        walkTree(root, func(node *tree_sitter.Node) {
            if node.Kind() == "import_from_statement" {
                moduleNode := node.ChildByFieldName("module_name")
                if moduleNode == nil {
                    return
                }
                mod := nodeText(moduleNode, f.Content)
                // Collect imported names
                for i := uint(0); i < node.ChildCount(); i++ {
                    child := node.Child(i)
                    if child != nil && child.Kind() == "dotted_name" {
                        if importedNames[mod] == nil {
                            importedNames[mod] = make(map[string]bool)
                        }
                        importedNames[mod][nodeText(child, f.Content)] = true
                    }
                }
            }
        })
    }

    // Flag definitions not imported by any other file
    var dead []types.DeadExport
    for _, d := range defs {
        referenced := false
        for _, names := range importedNames {
            if names[d.name] {
                referenced = true
                break
            }
        }
        if !referenced && len(files) > 1 {
            dead = append(dead, types.DeadExport{
                Package: d.module,
                Name:    d.name,
                File:    filepath.Base(d.file),
                Line:    d.line,
                Kind:    d.kind,
            })
        }
    }
    return dead
}
```

### Python Test Detection (C6)
```go
// Detect pytest-style and unittest-style test functions
func pyDetectTests(files []*parser.ParsedTreeSitterFile) (testFuncs []types.TestFunctionMetric, testFileCount, srcFileCount int) {
    for _, f := range files {
        isTestFile := f.Language == types.LangPython && isTestFileByPath(f.RelPath)

        if isTestFile {
            testFileCount++
        } else {
            srcFileCount++
        }

        if !isTestFile {
            continue
        }

        root := f.Tree.RootNode()
        walkTree(root, func(node *tree_sitter.Node) {
            if node.Kind() != "function_definition" {
                return
            }
            nameNode := node.ChildByFieldName("name")
            if nameNode == nil {
                return
            }
            name := nodeText(nameNode, f.Content)

            // pytest: test_ prefix
            if strings.HasPrefix(name, "test_") {
                assertCount := pyCountAssertions(node, f.Content)
                testFuncs = append(testFuncs, types.TestFunctionMetric{
                    Package:        filepath.Dir(f.RelPath),
                    Name:           name,
                    File:           f.Path,
                    Line:           int(node.StartPosition().Row) + 1,
                    AssertionCount: assertCount,
                })
            }
        })
    }
    return
}

// Count Python assertion patterns: assert statements, self.assert*, pytest.raises
func pyCountAssertions(funcNode *tree_sitter.Node, content []byte) int {
    count := 0
    walkTree(funcNode, func(node *tree_sitter.Node) {
        switch node.Kind() {
        case "assert_statement":
            count++
        case "call":
            // Check for self.assert* or self.fail
            fn := node.ChildByFieldName("function")
            if fn != nil {
                text := nodeText(fn, content)
                if strings.HasPrefix(text, "self.assert") || text == "self.fail" {
                    count++
                }
            }
        }
    })
    return count
}
```

## Tree-sitter Node Type Reference

Critical node types used across all analyzers:

### Python Node Types
| Category | Node Types |
|----------|-----------|
| Functions | `function_definition`, `decorated_definition` |
| Classes | `class_definition` |
| Control flow (+1 complexity) | `if_statement`, `elif_clause`, `for_statement`, `while_statement`, `except_clause`, `case_clause` |
| Boolean operators (+1) | `boolean_operator` (and/or) |
| Ternary (+1) | `conditional_expression` |
| Imports | `import_statement`, `import_from_statement` |
| Test assertions | `assert_statement` |
| Block bodies | `block` (child of if_statement, for_statement, function_definition, etc.) |
| Statements for duplication | All named children of `block` nodes |

### TypeScript Node Types
| Category | Node Types |
|----------|-----------|
| Functions | `function_declaration`, `method_definition`, `arrow_function`, `function_expression` |
| Classes | `class_declaration`, `abstract_class_declaration` |
| Control flow (+1) | `if_statement`, `for_statement`, `for_in_statement`, `while_statement`, `do_statement`, `switch_case`, `catch_clause` |
| Boolean operators (+1) | `binary_expression` with `&&`, `\|\|`, `??` operators |
| Ternary (+1) | `ternary_expression` |
| Imports (ESM) | `import_statement` with `source` field |
| Imports (CJS) | `call_expression` where function is `require` |
| Exports | `export_statement` |
| Test calls | `call_expression` where function is `describe`, `it`, `test` |
| Block bodies | `statement_block` |
| Statements for duplication | All named children of `statement_block` nodes |

## State of the Art

| Old Approach (Go-only C1/C3/C6) | Current Approach (Phase 7) | Impact |
|----------------------------------|---------------------------|--------|
| Go analyzers use `go/ast` + `go/packages` directly | Go analyzers still use go/ast; Python/TS use Tree-sitter via same interfaces | Backward compatible |
| `C1Analyzer` only implements `GoAwareAnalyzer` | `C1Analyzer` implements both `GoAwareAnalyzer` and `Analyzer` with target dispatch | Handles polyglot |
| Coverage only from Go `cover.out` or LCOV/Cobertura on disk | ARS runs tests to generate fresh coverage (per CONTEXT.md decision) | Active test running |
| No Python/TS test detection | AST-based test function detection + framework config | Accurate C6 metrics |

## Coverage Generation Strategy

Per CONTEXT.md decisions, ARS should actively run tests to generate fresh coverage data:

### Python Coverage Generation
```bash
# Try pytest with coverage first
python -m pytest --cov=. --cov-report=xml:coverage.xml --timeout=120
# Fall back to unittest
python -m coverage run -m unittest discover && python -m coverage xml -o coverage.xml
```
- Output: `coverage.xml` (Cobertura format) -- already parsed by `parseCobertura()`
- Timeout: Hard limit (e.g., 120s) to prevent hanging

### TypeScript Coverage Generation
```bash
# Try common test runners in order
npx vitest run --coverage --reporter=lcov 2>/dev/null ||
npx jest --coverage --coverageReporters=lcov 2>/dev/null ||
npx nyc mocha 2>/dev/null
```
- Output: `lcov.info` or `coverage/lcov.info` -- already parsed by `parseLCOV()`
- Timeout: Hard limit

### Go Coverage Generation (existing, for completeness)
```bash
go test -coverprofile=cover.out ./...
```

### Failure Handling
Per CONTEXT.md: if tests fail or timeout, skip C6 analysis and continue with C1/C3. Return `CoveragePercent: -1` and `CoverageSource: "none"`.

## Open Questions

1. **Python module boundary definition**
   - What we know: Python modules can be defined by `__init__.py` files (traditional packages) or any directory with `.py` files (implicit namespace packages, PEP 420).
   - What's unclear: Should we require `__init__.py` for module boundaries, or treat any directory with `.py` files as a module?
   - Recommendation: Start with directory-based (any dir with `.py` files = module). This is simpler and handles modern Python. Add `__init__.py` detection as a refinement signal.

2. **Nested function complexity counting**
   - What we know: Tree-sitter `walkTree()` visits ALL descendants, including nested function definitions. A function containing a nested function would double-count the inner function's complexity.
   - What's unclear: The exact approach to handle this cleanly with the recursive `walkTree` helper.
   - Recommendation: When computing complexity for a function, skip `function_definition` / `arrow_function` children (do not recurse into them). This requires either a modified walkTree or post-hoc subtraction.

3. **TypeScript path alias resolution**
   - What we know: Many TS projects use `@/` aliases mapped in `tsconfig.json`.
   - What's unclear: How many real projects use aliases vs relative imports, and whether incomplete alias resolution causes more harm than good.
   - Recommendation: Parse `tsconfig.json` `paths` if present. If alias resolution fails, include the import as-is (don't drop it). Log a warning.

4. **Active test execution safety**
   - What we know: CONTEXT.md says ARS should run tests to generate coverage.
   - What's unclear: Running arbitrary project tests has security and reliability implications (network access, database writes, system modifications).
   - Recommendation: Run tests in a subprocess with timeout, limited environment (no network unless --allow-network), and capture stderr for diagnostics. Default to opt-in (`--run-tests`) rather than automatic.

## Sources

### Primary (HIGH confidence)
- Existing codebase: `internal/analyzer/c1_codehealth.go`, `c3_architecture.go`, `c6_testing.go` -- exact Go analysis patterns to replicate
- Existing codebase: `internal/analyzer/c2_python.go`, `c2_typescript.go` -- proven Tree-sitter node walking patterns
- Existing codebase: `internal/analyzer/c2_semantics.go` -- dispatcher pattern for multi-language
- Existing codebase: `internal/parser/treesitter.go` -- TreeSitterParser API and ParseTargetFiles()
- Existing codebase: `pkg/types/types.go` -- C1Metrics, C3Metrics, C6Metrics structures
- Existing codebase: `internal/scoring/scorer.go` -- extractC1, extractC3, extractC6 functions

### Secondary (MEDIUM confidence)
- [tree-sitter-python node-types.json](https://github.com/tree-sitter/tree-sitter-python/blob/master/src/node-types.json) -- Python AST node type names
- [tree-sitter-typescript node-types.json](https://github.com/tree-sitter/tree-sitter-typescript/blob/master/typescript/src/node-types.json) -- TypeScript AST node type names
- [tree-sitter static node types docs](https://tree-sitter.github.io/tree-sitter/using-parsers/6-static-node-types.html) -- node type system
- Phase 7 CONTEXT.md -- user decisions on test frameworks, complexity mapping, decorator handling

### Tertiary (LOW confidence)
- TypeScript `require()` call expression node structure -- needs validation with actual Tree-sitter parse output
- Python decorator complexity heuristics -- no standard source, recommendation is project-specific

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- all libraries already integrated and proven in Phase 6
- Architecture: HIGH -- C2 dispatcher pattern exists and works, direct template for C1/C3/C6
- Tree-sitter node types: MEDIUM -- node-types.json from official repos, but field names need validation during implementation
- Coverage generation: MEDIUM -- command patterns are standard but actual execution behavior varies by project
- Pitfalls: HIGH -- derived from actual codebase analysis and common static analysis challenges

**Research date:** 2026-02-01
**Valid until:** 2026-03-01 (stable domain, no fast-moving dependencies)
