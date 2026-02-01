# Architecture Research: v2 Multi-Language Expansion

**Domain:** Static analysis CLI tool -- expanding from Go-only to Go/Python/TypeScript
**Researched:** 2026-02-01
**Confidence:** HIGH (based on direct codebase analysis + verified library research)

## Executive Summary

The v1 architecture is well-structured for expansion. The pipeline pattern (Discover -> Parse -> Analyze -> Score -> Recommend -> Render) remains correct. The primary architectural challenge is that the `Parser` interface and `Analyzer` interface are tightly coupled to `*parser.ParsedPackage` (a Go-specific type wrapping `go/packages` output). Multi-language support requires introducing a **language-agnostic intermediate representation** that all analyzers consume, while preserving Go's rich type information for Go-specific analysis.

The recommendation is a **dual-parser strategy**: keep `go/packages` for Go (it provides type info that Tree-sitter cannot), add Tree-sitter for Python/TypeScript, and introduce a unified `AnalysisTarget` type that both parsers produce. New analyzers (C2, C4, C5, C7) plug into the existing `Analyzer` interface with minimal changes. LLM-dependent analyzers (C4, C7) require a new async execution model with cost guards.

## Current Architecture (v1 Baseline)

### Pipeline Flow

```
CLI (cmd/scan.go)
  |
  v
Pipeline.Run(dir)
  |
  +-- Stage 1: discovery.Walker.Discover(dir)  --> *types.ScanResult
  |     (walks filesystem, classifies .go files)
  |
  +-- Stage 2: parser.GoPackagesParser.Parse(dir)  --> []*parser.ParsedPackage
  |     (go/packages.Load with full type info)
  |
  +-- Stage 3: analyzers[].Analyze(pkgs)  --> []*types.AnalysisResult  (parallel via errgroup)
  |     C1Analyzer, C3Analyzer, C6Analyzer
  |
  +-- Stage 3.5: scorer.Score(results)  --> *types.ScoredResult
  |
  +-- Stage 3.6: recommend.Generate(scored, config)  --> []Recommendation
  |
  +-- Stage 4: output.RenderSummary / RenderJSON  --> io.Writer
  |
  +-- Stage 5: threshold check  --> ExitError if below
```

### Current Interfaces (from pipeline/interfaces.go)

```go
type Parser interface {
    Parse(rootDir string) ([]*parser.ParsedPackage, error)
}

type Analyzer interface {
    Name() string
    Analyze(pkgs []*parser.ParsedPackage) (*types.AnalysisResult, error)
}
```

### Key Coupling Points

| Component | Coupled To | Impact on v2 |
|-----------|-----------|--------------|
| `Parser` interface | Returns `[]*parser.ParsedPackage` (Go-specific) | Must generalize for multi-language |
| `Analyzer` interface | Accepts `[]*parser.ParsedPackage` (Go-specific) | Must accept language-agnostic input |
| `ParsedPackage` struct | Contains `*ast.File`, `*types.Package`, `*types.Info` | Go-only; Python/TS need different representation |
| `discovery.Walker` | Only discovers `.go` files | Must discover `.py`, `.ts`, `.tsx`, `.js` |
| `scoring.ScoringConfig` | Hard-codes C1, C3, C6 fields | Must add C2, C4, C5, C7 fields |
| `output.RenderSummary` | Switch on "C1", "C3", "C6" categories | Must handle 7 categories |
| `scoring.Scorer.Score` | Switch on "C1", "C3", "C6" categories | Must handle 7 categories |

## Recommended Architecture (v2)

### Strategy: Unified Analysis Target with Language-Specific Parsers

```
CLI (cmd/scan.go)
  |
  +-- Config: LoadConfig(.arsrc.yml)  --> *Config  [NEW]
  |
  v
Pipeline.Run(dir, config)
  |
  +-- Stage 1: discovery.Walker.Discover(dir)  --> *types.ScanResult  [MODIFIED]
  |     (discovers .go, .py, .ts, .tsx, .js files, classifies by language)
  |
  +-- Stage 2a: parser.GoParser.Parse(dir)          --> []*AnalysisTarget  [KEEP go/packages]
  +-- Stage 2b: parser.TreeSitterParser.Parse(dir)   --> []*AnalysisTarget  [NEW]
  |     (parallel per-language parsing)
  |
  +-- Stage 2c: git.LogParser.Parse(dir)  --> *GitHistory  [NEW, for C5]
  |
  +-- Stage 3: analyzers[].Analyze(targets, gitHistory, config)  --> []*AnalysisResult  [parallel]
  |     C1, C2, C3, C4, C5, C6, C7 analyzers
  |     (C4, C7 are opt-in with LLM cost warnings)
  |
  +-- Stage 3.5: scorer.Score(results)  --> *ScoredResult
  +-- Stage 3.6: recommend.Generate(scored, config)  --> []Recommendation
  |
  +-- Stage 4: output.Render(format, scored, recs)  --> io.Writer
  |     Terminal | JSON | HTML  [HTML is NEW]
  |
  +-- Stage 5: threshold check
```

### Question 1: Multi-Language Strategy

**Recommendation: Unified parser interface, per-language implementations, shared intermediate representation.**

Do NOT create per-language pipelines. The pipeline is the same for all languages -- only the parser stage differs. All analyzers should receive the same `AnalysisTarget` type regardless of source language.

**Why not per-language pipelines:**
- Duplicates scoring, recommendation, and output logic
- Makes cross-language analysis (e.g., "does this polyglot repo have consistent naming?") impossible
- Violates KISS -- one pipeline, multiple parsers is simpler

**Proposed intermediate representation:**

```go
// AnalysisTarget is the language-agnostic unit of analysis.
// For Go: one per package. For Python/TS: one per file or module.
type AnalysisTarget struct {
    Language    Language           // Go, Python, TypeScript
    Path        string             // File or package path
    Files       []SourceFile       // Source files in this target
    Functions   []FunctionInfo     // Extracted function signatures
    Imports     []ImportInfo       // Import/require statements
    Classes     []ClassInfo        // Classes/structs (Python/TS)
    Exports     []ExportInfo       // Exported symbols
    TestFiles   []SourceFile       // Associated test files

    // Language-specific extensions (type-assert when needed)
    GoPackage   *parser.ParsedPackage  // Non-nil only for Go targets
    TreeSitterTree *sitter.Tree        // Non-nil for TS/Python targets
}

type SourceFile struct {
    Path       string
    RelPath    string
    Language   Language
    Lines      int
    Class      types.FileClass  // source, test, generated, excluded
    Content    []byte           // Raw source (needed for Tree-sitter queries)
}

type FunctionInfo struct {
    Name       string
    File       string
    Line       int
    EndLine    int
    Parameters int
    IsExported bool
    IsTest     bool
    Complexity int   // Computed during parsing for Go (gocyclo), estimated for others
}
```

**Key design decision:** Keep `GoPackage *parser.ParsedPackage` as an optional field on `AnalysisTarget`. This lets Go-specific analyzers (C1 uses `go/ast` for duplication detection, C3 uses `go/types` for dead export detection) continue accessing rich Go type information without forcing a lowest-common-denominator representation. Python/TS analyzers use `TreeSitterTree` instead.

**Confidence:** HIGH -- This "adapter" pattern (shared interface + language-specific extensions) is standard in multi-language analysis tools like SonarQube and Semgrep.

### Question 2: Tree-sitter Integration -- Add Alongside, Do NOT Replace

**Recommendation: Keep `go/packages` for Go analysis. Add `smacker/go-tree-sitter` for Python and TypeScript only.**

**Why NOT replace go/packages with Tree-sitter for Go:**

| Capability | go/packages | Tree-sitter |
|-----------|-------------|-------------|
| Type information | Full `go/types.Info` (uses, defs, type assertions) | None |
| Cross-package resolution | Yes (imports resolved, dependency graph) | No |
| Dead export detection (C3) | Yes (via `types.Object` cross-reference) | Not possible |
| Test package separation | Yes (`ForTest` field) | Manual (filename heuristic only) |
| Cyclomatic complexity | Via gocyclo on `*ast.File` | Would need custom Tree-sitter queries |
| AST fidelity for Go | Perfect (official parser) | Grammar may lag official Go spec |

The v1 C3 analyzer's `detectDeadCode()` function uses `pkg.TypesInfo.Uses` to find cross-package references -- this requires full type checking that Tree-sitter cannot provide. The C1 analyzer uses `gocyclo.AnalyzeASTFile()` which operates on `*ast.File`. Replacing go/packages would regress Go analysis quality.

**Tree-sitter for Python/TypeScript provides:**
- Function/class extraction via S-expression queries
- Import statement parsing
- File structure analysis (nesting depth, complexity estimation)
- Fast parsing (~36x faster than traditional parsers per Symflower benchmarks)
- Error-tolerant parsing (partial files still produce useful ASTs)

**Which Go binding to use:**

| Binding | Pros | Cons | Recommendation |
|---------|------|------|----------------|
| `smacker/go-tree-sitter` | Bundled grammars, GC-managed, 398+ importers | Less modular, slightly larger binary | **Use this one** |
| `tree-sitter/go-tree-sitter` | Official, modular grammar loading | Must call Close() manually (CGO finalizer bugs), newer/less proven | Avoid for now |

**Use `smacker/go-tree-sitter`** because:
1. Python and TypeScript grammars are bundled -- no separate grammar management
2. GC-managed memory via `runtime.SetFinalizer` -- no manual Close() calls
3. More mature with wider adoption (398+ importers vs newer official binding)
4. Sufficient for structural analysis (we do not need incremental re-parsing)

**Confidence:** HIGH -- verified via [smacker/go-tree-sitter](https://github.com/smacker/go-tree-sitter) documentation and [tree-sitter/go-tree-sitter](https://github.com/tree-sitter/go-tree-sitter) README.

### Question 3: LLM-Dependent Analysis (C4, C7) Architecture

**C4 (Documentation Quality)** needs LLM for content quality assessment (not just presence checks).
**C7 (Agent Evaluation)** needs headless Claude Code spawning for real agent-in-the-loop tests.

**Recommendation: Separate execution tiers with explicit opt-in and cost warnings.**

```
Tier 1 (Default): C1, C2, C3, C5, C6 -- Fast, local, deterministic
Tier 2 (--enable-llm): C4 -- LLM API call, moderate latency (~5-15s)
Tier 3 (--enable-c7): C7 -- Headless agent spawn, high latency (~60-300s), high cost
```

**Architecture for LLM calls:**

```go
// LLMClient abstracts LLM provider interactions.
type LLMClient interface {
    Evaluate(ctx context.Context, prompt string) (string, error)
    EstimateCost(prompt string) float64
}

// C4Analyzer uses LLM for documentation quality scoring.
type C4Analyzer struct {
    LLM        LLMClient   // nil = skip LLM-based sub-metrics
    Timeout    time.Duration
    MaxRetries int
}

func (a *C4Analyzer) Analyze(targets []*AnalysisTarget) (*types.AnalysisResult, error) {
    // Phase 1: Structural analysis (always runs, fast)
    //   - README presence, API doc coverage, comment density
    metrics := a.analyzeStructure(targets)

    // Phase 2: Content quality (only if LLM client provided)
    if a.LLM != nil {
        ctx, cancel := context.WithTimeout(context.Background(), a.Timeout)
        defer cancel()
        quality, err := a.evaluateContentQuality(ctx, targets)
        if err != nil {
            // Degrade gracefully: log warning, use structural-only score
            log.Printf("C4 LLM evaluation failed: %v (using structural metrics only)", err)
        } else {
            metrics.merge(quality)
        }
    }

    return metrics, nil
}
```

**C7 headless Claude Code integration:**

```go
// C7Analyzer spawns headless Claude Code for agent evaluation.
type C7Analyzer struct {
    ClaudeCodePath string         // Path to claude binary
    Timeout        time.Duration  // Per-task timeout (default 120s)
    Tasks          []AgentTask    // Evaluation tasks to run
}

func (a *C7Analyzer) Analyze(targets []*AnalysisTarget) (*types.AnalysisResult, error) {
    // Pre-flight: estimate cost, warn user
    cost := a.estimateCost(targets)
    fmt.Fprintf(os.Stderr, "C7 estimated cost: $%.2f (API calls: %d)\n", cost.USD, cost.Calls)

    results := make([]TaskResult, len(a.Tasks))
    for i, task := range a.Tasks {
        ctx, cancel := context.WithTimeout(context.Background(), a.Timeout)
        result, err := a.runTask(ctx, task, targets)
        cancel()
        if err != nil {
            results[i] = TaskResult{Status: "error", Error: err.Error()}
            continue
        }
        results[i] = result
    }

    return a.scoreResults(results), nil
}

func (a *C7Analyzer) runTask(ctx context.Context, task AgentTask, targets []*AnalysisTarget) (TaskResult, error) {
    cmd := exec.CommandContext(ctx, a.ClaudeCodePath,
        "-p", task.Prompt,
        "--output-format", "json",
        "--dangerously-skip-permissions",
    )
    cmd.Dir = targets[0].Path // run in project directory
    output, err := cmd.Output()
    // Parse JSON output, evaluate success criteria
    ...
}
```

**Key design decisions:**
1. **Graceful degradation:** If LLM fails, C4 falls back to structural-only metrics (still useful). C7 failure means that category scores "n/a".
2. **Explicit opt-in:** C4 LLM and C7 are behind flags, not default. Users must acknowledge cost.
3. **Timeout per call:** Use `context.WithTimeout` -- 30s for C4 LLM calls, 120s per C7 task.
4. **No retry for C7:** Agent tasks are expensive; retrying doubles cost. Log error and move on.
5. **Sequential C7 tasks:** Do not parallelize agent spawns -- they are resource-intensive and may conflict.

**Confidence:** HIGH for C4 architecture, MEDIUM for C7 (headless Claude Code interface details may change; the `claude -p` CLI is documented at [code.claude.com/docs/en/headless](https://code.claude.com/docs/en/headless) but specifics of output parsing need validation).

### Question 4: Git Analysis (C5) -- Separate Pre-Stage

**Recommendation: Git analysis runs as a separate pre-stage before analyzers, not integrated into the scanner.**

**Rationale:** Git history is a different data source than source files. The scanner discovers files on disk; git analysis reads `.git/objects` and commit logs. They have different failure modes (no `.git` directory = C5 unavailable, not a fatal error), different performance characteristics (git log on a large repo can take seconds), and different data shapes.

```go
// git/analyzer.go
type GitAnalyzer struct{}

type GitHistory struct {
    Commits     []Commit
    FileChanges map[string]FileChangeStats  // path -> change frequency, authors
    HotSpots    []HotSpot                   // files with highest churn
    AuthorStats map[string]AuthorStats      // author -> commit count, files touched
    Age         time.Duration               // time since first commit
}

func (g *GitAnalyzer) Parse(dir string) (*GitHistory, error) {
    // Use go-git (pure Go, no git binary dependency)
    repo, err := git.PlainOpen(dir)
    if err != nil {
        return nil, fmt.Errorf("open git repo: %w (C5 requires .git directory)", err)
    }

    // Iterate commit log
    iter, err := repo.Log(&git.LogOptions{All: true})
    ...
}
```

**Pipeline integration:**

```go
// In pipeline.go, add git parsing as Stage 2c (parallel with source parsing)
g := new(errgroup.Group)

var targets []*AnalysisTarget
var gitHistory *GitHistory

g.Go(func() error {
    var err error
    targets, err = p.parseAll(dir)  // Go + Tree-sitter parsing
    return err
})

g.Go(func() error {
    var err error
    gitHistory, err = p.gitAnalyzer.Parse(dir)
    if err != nil {
        // C5 unavailable -- not fatal
        log.Printf("Git analysis unavailable: %v", err)
    }
    return nil  // never fail the pipeline for git issues
})

g.Wait()
```

**Library choice: `go-git/go-git` v5**
- Pure Go implementation, no `git` binary dependency
- Reads `.git` directory directly
- Supports log iteration, diff stat, blame
- Used by Gitea, Pulumi, and other production tools

**Alternative considered: shelling out to `git log`**
- Faster for simple operations (git's C implementation is optimized)
- Requires git binary installed
- Output parsing is fragile
- **Rejected:** go-git is sufficient and avoids external dependency

**Confidence:** HIGH -- [go-git/go-git](https://github.com/go-git/go-git) is well-established and provides all needed git operations.

### Question 5: Config Loading -- Early, at CLI Init

**Recommendation: Load `.arsrc.yml` early in the CLI layer, pass config through pipeline.**

**Rationale:** Config affects which analyzers run (e.g., `enable_c7: true`), scoring weights, and output format. Loading late (per-analyzer) means each analyzer needs to find and parse the config file independently, violating DRY and making it impossible to validate config before starting analysis.

```go
// config/config.go
type Config struct {
    // Scoring weights (overrides defaults)
    Scoring  *scoring.ScoringConfig `yaml:"scoring"`

    // Analysis toggles
    Analysis AnalysisConfig `yaml:"analysis"`

    // Output preferences
    Output   OutputConfig   `yaml:"output"`
}

type AnalysisConfig struct {
    EnableLLM  bool     `yaml:"enable_llm"`   // Enable C4 LLM evaluation
    EnableC7   bool     `yaml:"enable_c7"`    // Enable headless agent eval
    Languages  []string `yaml:"languages"`    // ["go", "python", "typescript"]
    ExcludeDirs []string `yaml:"exclude_dirs"` // Additional dirs to skip
}

type OutputConfig struct {
    Format     string `yaml:"format"`      // "terminal", "json", "html"
    HTMLOutput string `yaml:"html_output"` // Path for HTML report
    Verbose    bool   `yaml:"verbose"`
}
```

**Config resolution order (highest priority first):**
1. CLI flags (`--json`, `--threshold`, `--enable-c7`)
2. `.arsrc.yml` in project root
3. Built-in defaults

**Loading sequence:**

```go
// In cmd/scan.go RunE:
// 1. Load config file (if exists)
cfg, err := config.Load(dir)  // looks for .arsrc.yml in dir

// 2. Override with CLI flags
if jsonOutput {
    cfg.Output.Format = "json"
}
if threshold > 0 {
    cfg.Scoring.Threshold = threshold
}

// 3. Pass to pipeline
p := pipeline.New(cfg, os.Stdout)
```

**Why `.arsrc.yml` not `.arsrc.yaml`:**
- Shorter, more common in CLI tools (`.eslintrc.yml`, `.golangci.yml`)
- YAML spec allows both extensions

**Integration with existing `scoring.LoadConfig`:**
The v1 `scoring.LoadConfig(path)` loads scoring YAML from a `--config` flag path. In v2, this merges into the broader `.arsrc.yml` config. The `scoring` field in `.arsrc.yml` uses the same structure as the existing `ScoringConfig`, so existing scoring config files remain compatible.

**Confidence:** HIGH -- This follows standard Go CLI patterns (Cobra + YAML config file).

### Question 6: HTML Generation -- html/template + Embedded Charts

**Recommendation: Use Go's `html/template` with embedded CSS/JS for self-contained single-file HTML reports. Use inline SVG for charts (no external dependencies).**

**Why html/template (standard library):**
- No external dependency
- Type-safe template execution
- Auto-escaping prevents XSS in metric values
- Familiar to Go developers
- Supports template composition (`{{template "header" .}}`)

**Why NOT go-echarts:**
- Requires external JS CDN (echarts.min.js is 1MB+) or bundled assets
- Adds CGO complexity if using offline mode
- Over-engineered for radar charts and bar charts
- go-echarts produces standalone HTML files; embedding snippets requires [custom renderers](https://blog.cubieserver.de/2020/how-to-render-standalone-html-snippets-with-go-echarts/)

**Why inline SVG charts:**
- Zero external dependencies (no JS, no CDN)
- Self-contained single HTML file
- Simple to generate programmatically (radar chart = polygon coordinates)
- Sufficient for 7-category radar chart + per-metric bar charts
- Can be styled with embedded CSS

**Implementation approach:**

```go
// output/html.go
type HTMLRenderer struct {
    tmpl *template.Template
}

//go:embed templates/*.html
var templateFS embed.FS

func NewHTMLRenderer() *HTMLRenderer {
    tmpl := template.Must(template.New("report").
        Funcs(template.FuncMap{
            "radarChart": generateRadarSVG,
            "barChart":   generateBarSVG,
            "scoreColor": scoreColorCSS,
        }).
        ParseFS(templateFS, "templates/*.html"))
    return &HTMLRenderer{tmpl: tmpl}
}

func (r *HTMLRenderer) Render(w io.Writer, scored *types.ScoredResult, recs []recommend.Recommendation) error {
    data := buildReportData(scored, recs)
    return r.tmpl.ExecuteTemplate(w, "report.html", data)
}
```

**Template structure:**

```
internal/output/templates/
  report.html      # Main report layout
  header.html      # Score summary with radar chart
  category.html    # Per-category detail section
  recommendations.html  # Improvement recommendations
  research.html    # Research citations and methodology
  styles.css       # Embedded CSS (injected into <style>)
```

**Chart generation (pure Go -> SVG):**

```go
// Generate radar chart as inline SVG string
func generateRadarSVG(categories []CategoryScore) template.HTML {
    // Calculate polygon points for 7 categories on unit circle
    // Each axis: 0 (center) to 10 (edge)
    var points []string
    n := len(categories)
    for i, cat := range categories {
        angle := float64(i) * 2 * math.Pi / float64(n) - math.Pi/2
        r := cat.Score / 10.0 * radius
        x := cx + r*math.Cos(angle)
        y := cy + r*math.Sin(angle)
        points = append(points, fmt.Sprintf("%.1f,%.1f", x, y))
    }
    // Return SVG polygon + axis lines + labels
    ...
}
```

**Confidence:** HIGH -- html/template is battle-tested; SVG generation is straightforward math. GoReporter uses a similar html/template approach for its [HTML reports](https://github.com/360EntSecGroup-Skylar/goreporter).

### Question 7: Performance with 3 Languages -- Parallel Per-Language

**Recommendation: Parse languages in parallel. Go parsing (go/packages) and Tree-sitter parsing (Python + TypeScript) run concurrently.**

```go
func (p *Pipeline) parseAll(dir string) ([]*AnalysisTarget, error) {
    g := new(errgroup.Group)
    var (
        goTargets []*AnalysisTarget
        pyTargets []*AnalysisTarget
        tsTargets []*AnalysisTarget
    )

    // Parse Go packages (uses go/packages, ~2-10s for large repos)
    g.Go(func() error {
        var err error
        goTargets, err = p.goParser.Parse(dir)
        return err
    })

    // Parse Python files (Tree-sitter, very fast ~100ms for 50k LOC)
    g.Go(func() error {
        var err error
        pyTargets, err = p.tsParser.ParsePython(dir, p.scanResult.PythonFiles)
        return err
    })

    // Parse TypeScript files (Tree-sitter, very fast ~100ms for 50k LOC)
    g.Go(func() error {
        var err error
        tsTargets, err = p.tsParser.ParseTypeScript(dir, p.scanResult.TypeScriptFiles)
        return err
    })

    if err := g.Wait(); err != nil {
        return nil, err
    }

    return append(append(goTargets, pyTargets...), tsTargets...), nil
}
```

**Performance budget (50k LOC target):**

| Stage | Expected Time | Bottleneck |
|-------|---------------|------------|
| Discovery (filesystem walk) | ~100ms | I/O |
| Go parsing (go/packages) | ~2-8s | Type checking, dependency resolution |
| Python parsing (Tree-sitter) | ~50-200ms | Very fast, syntax only |
| TypeScript parsing (Tree-sitter) | ~50-200ms | Very fast, syntax only |
| Git history (go-git) | ~1-5s | Depends on repo history depth |
| Analyzers C1-C3, C5-C6 | ~1-3s | CPU-bound AST traversal |
| C4 with LLM (opt-in) | ~5-15s | Network latency |
| C7 with agent (opt-in) | ~60-300s | Agent execution |
| Scoring + recommendations | ~10ms | Trivial |
| Output rendering | ~50ms | Template execution (HTML) |
| **Total (without C4/C7)** | **~5-15s** | **Well within 30s budget** |

**Key insight:** `go/packages` is the bottleneck, not Tree-sitter. Tree-sitter parsing is ~36x faster than traditional parsers. The performance budget is dominated by Go's type checker, which we cannot avoid since C1/C3 analyzers depend on type information.

**Confidence:** HIGH -- Tree-sitter performance claims verified via [Symflower benchmarks](https://symflower.com/en/company/blog/2023/parsing-code-with-tree-sitter/); go/packages latency confirmed from v1 experience.

## Refactoring Plan: v1 -> v2

### Phase 1: Interface Generalization (Minimal Refactoring)

**Goal:** Make the pipeline accept multi-language targets without breaking existing Go analyzers.

**Changes:**

1. **New `AnalysisTarget` type** in `pkg/types/`:
   - Language-agnostic fields (Language, Path, Files, Functions, Imports)
   - `GoPackage *parser.ParsedPackage` for Go-specific data

2. **New `Parser` interface** in `pipeline/interfaces.go`:
   ```go
   type Parser interface {
       Parse(dir string, files []types.DiscoveredFile) ([]*types.AnalysisTarget, error)
   }
   ```

3. **Adapt existing `GoPackagesParser`** to return `[]*types.AnalysisTarget` (wrapper that populates both generic fields and `GoPackage`).

4. **Update `Analyzer` interface** signature:
   ```go
   type Analyzer interface {
       Name() string
       Analyze(targets []*types.AnalysisTarget) (*types.AnalysisResult, error)
   }
   ```

5. **Update existing C1, C3, C6 analyzers** to extract `GoPackage` from targets:
   ```go
   func (a *C1Analyzer) Analyze(targets []*types.AnalysisTarget) (*types.AnalysisResult, error) {
       pkgs := extractGoPackages(targets)  // helper filters Language==Go, returns GoPackage
       // ... rest unchanged
   }
   ```

**This is the critical refactoring.** Once the interface accepts `[]*AnalysisTarget`, new parsers and analyzers plug in without further interface changes.

### Phase 2: Discovery Expansion

**Modify `discovery.Walker` to:**
- Accept configured languages: `walker.Discover(dir, []Language{Go, Python, TypeScript})`
- Classify files by extension: `.go`, `.py`, `.ts`, `.tsx`, `.js`
- Return per-language file lists in `ScanResult`
- Add `Language` field to `DiscoveredFile`

### Phase 3: Tree-sitter Parser

**Add `parser/treesitter.go`:**
- Implement `Parser` interface for Python and TypeScript
- Extract function/class/import information via Tree-sitter queries
- Populate `AnalysisTarget` with structural metrics

### Phase 4: New Analyzers

Each new analyzer implements the `Analyzer` interface:

| Analyzer | Data Source | Special Requirements |
|----------|-----------|---------------------|
| C2 (Semantic Explicitness) | `AnalysisTarget.Functions`, `.Imports` | Naming analysis, type annotation checks |
| C4 (Documentation) | `AnalysisTarget.Files` (source content) | LLM client for content quality (opt-in) |
| C5 (Temporal Dynamics) | `GitHistory` (from git pre-stage) | Separate data source; needs interface extension |
| C7 (Agent Evaluation) | Project directory | Claude Code binary; exec.Command |

**C5 special case:** C5 needs `GitHistory`, not `[]*AnalysisTarget`. Options:

Option A: Extend Analyzer interface to accept both:
```go
type Analyzer interface {
    Name() string
    Analyze(targets []*types.AnalysisTarget, opts ...AnalyzeOption) (*types.AnalysisResult, error)
}
type AnalyzeOption func(*AnalyzeContext)
func WithGitHistory(h *GitHistory) AnalyzeOption { ... }
```

Option B: Give C5Analyzer a `GitHistory` field set during construction:
```go
type C5Analyzer struct {
    History *GitHistory  // Set by pipeline before Analyze() is called
}
func (a *C5Analyzer) Analyze(targets []*types.AnalysisTarget) (*types.AnalysisResult, error) {
    if a.History == nil { return unavailableResult(), nil }
    // use a.History
}
```

**Recommendation: Option B.** Simpler, no interface change needed. The pipeline sets `c5.History` after git parsing completes. If git parsing failed, `History` is nil and C5 returns "unavailable" gracefully.

### Phase 5: Scoring Expansion

**Modify `scoring/config.go`:**
- Add C2, C4, C5, C7 to `ScoringConfig`
- Each new category gets `CategoryConfig` with metrics and breakpoints
- Update `DefaultConfig()` with research-backed defaults
- Update `Scorer.Score()` to handle all 7 categories (replace switch with registry)

```go
// Replace hard-coded switch with scorer registry
type CategoryScorer func(ar *types.AnalysisResult, cfg CategoryConfig) types.CategoryScore

var scorerRegistry = map[string]CategoryScorer{
    "C1": scoreC1,
    "C2": scoreC2,
    "C3": scoreC3,
    "C4": scoreC4,
    "C5": scoreC5,
    "C6": scoreC6,
    "C7": scoreC7,
}

func (s *Scorer) Score(results []*types.AnalysisResult) (*types.ScoredResult, error) {
    var categories []types.CategoryScore
    for _, ar := range results {
        if scorer, ok := scorerRegistry[ar.Category]; ok {
            cfg := s.Config.CategoryByName(ar.Category)
            categories = append(categories, scorer(ar, cfg))
        }
    }
    // ... compute composite, classify tier
}
```

### Phase 6: Output Expansion

**Add `output/html.go`:**
- html/template-based renderer
- Embedded templates via `embed.FS`
- Inline SVG charts (radar + bar)
- Research citations section

**Modify `output/terminal.go`:**
- Add render functions for C2, C4, C5, C7
- Replace category switch with registry pattern (same as scorer)

## Component Dependency Graph

```
                    .arsrc.yml
                        |
                    [Config Loader]
                        |
                    [Pipeline]
                   /    |    \
              [Discovery]  [Git Parser]
                   |           |
            [Go Parser]  [TS Parser]  --> []*AnalysisTarget + *GitHistory
                   \      |      /
                [Analyzer Registry]
               /  |  |  |  |  |  \
             C1  C2  C3  C4  C5  C6  C7
              \   |   |   |   |   |  /
               [Scorer Registry]
                      |
               [Recommender]
                      |
              [Output Registry]
             /     |       \
         Terminal  JSON    HTML
```

## Build Order for v2 Phases

| Order | Work Item | Depends On | Effort |
|-------|-----------|-----------|--------|
| 1 | Generalize `AnalysisTarget` type + update interfaces | Nothing (additive) | Medium |
| 2 | Adapt existing C1/C3/C6 to new interface | #1 | Medium |
| 3 | Expand discovery for multi-language | #1 | Low |
| 4 | Config system (`.arsrc.yml`) | Nothing (parallel) | Medium |
| 5 | Tree-sitter parser (Python + TS) | #1, #3 | High |
| 6 | C2 analyzer (Semantic Explicitness) | #1, #2 | Medium |
| 7 | C5 analyzer (git history) + go-git | #1 | Medium |
| 8 | Scoring expansion (7 categories) | #1, #6, #7 | Medium |
| 9 | C4 analyzer (structural + LLM) | #1, #8 | High |
| 10 | C7 analyzer (headless agent) | #1, #8 | High |
| 11 | HTML output | #8 | Medium |
| 12 | Integration testing on real repos | All above | High |

**Critical path:** #1 -> #2 -> #5 -> #8 -> #11
**Parallel tracks after #1:** Config (#4), Git (#7), C2 (#6) can all proceed independently.

## Anti-Patterns to Avoid

### Anti-Pattern: Lowest Common Denominator Representation

**Trap:** Defining `AnalysisTarget` with only fields available in all languages (functions, imports). Losing Go's type information.

**Why it's wrong:** Go's `go/packages` provides type resolution, cross-package references, and dead export detection that Tree-sitter cannot. Forcing everything through a minimal interface would regress Go analysis quality.

**Prevention:** Keep language-specific extensions (GoPackage, TreeSitterTree) on AnalysisTarget. Analyzers that need language-specific data type-assert into it.

### Anti-Pattern: Re-implementing go/packages with Tree-sitter

**Trap:** "Tree-sitter can parse Go too, so let's use one parser for everything."

**Why it's wrong:** Tree-sitter for Go provides syntax-only AST. No type info, no import resolution, no cross-package analysis. Would lose C3 dead export detection, coupling analysis accuracy, and gocyclo integration.

**Prevention:** Use the right tool for each language. go/packages for Go, Tree-sitter for Python/TS.

### Anti-Pattern: Synchronous LLM Calls Blocking Pipeline

**Trap:** Making C4/C7 part of the default analyzer set, causing every scan to wait 60+ seconds.

**Why it's wrong:** Users expect `ars scan` to complete in <30s. LLM latency is 5-300 seconds. If on by default, every scan feels broken.

**Prevention:** Tier 1/2/3 execution model. C4 LLM and C7 are opt-in flags. Default scan runs only Tier 1 (fast, local).

### Anti-Pattern: Shelling Out to git Binary

**Trap:** Using `exec.Command("git", "log", ...)` for C5 git analysis.

**Why it's wrong:** Requires git binary installed, output parsing is fragile, platform-specific line endings, no structured data. go-git provides typed, structured commit data.

**Prevention:** Use `go-git/go-git` v5 -- pure Go, no external dependency, structured API.

## New Package Structure

```
ars/
├── main.go
├── cmd/
│   ├── root.go
│   └── scan.go                    [MODIFIED: config loading, new flags]
├── internal/
│   ├── config/
│   │   └── config.go              [NEW: .arsrc.yml loader]
│   ├── pipeline/
│   │   ├── interfaces.go          [MODIFIED: AnalysisTarget-based interfaces]
│   │   ├── pipeline.go            [MODIFIED: multi-parser, git stage, tier execution]
│   │   └── progress.go
│   ├── discovery/
│   │   ├── walker.go              [MODIFIED: multi-language file discovery]
│   │   └── classifier.go          [MODIFIED: classify .py, .ts, .tsx, .js]
│   ├── parser/
│   │   ├── parser.go              [MODIFIED: wraps go/packages into AnalysisTarget]
│   │   └── treesitter.go          [NEW: Tree-sitter parser for Python/TS]
│   ├── git/
│   │   └── analyzer.go            [NEW: go-git history parsing]
│   ├── analyzer/
│   │   ├── helpers.go
│   │   ├── c1_codehealth.go       [MODIFIED: accepts AnalysisTarget]
│   │   ├── c2_semantics.go        [NEW]
│   │   ├── c3_architecture.go     [MODIFIED: accepts AnalysisTarget]
│   │   ├── c4_documentation.go    [NEW: structural + LLM]
│   │   ├── c5_temporal.go         [NEW: git-based]
│   │   ├── c6_testing.go          [MODIFIED: accepts AnalysisTarget]
│   │   └── c7_agent.go            [NEW: headless Claude Code]
│   ├── scoring/
│   │   ├── config.go              [MODIFIED: 7 categories]
│   │   └── scorer.go              [MODIFIED: registry pattern]
│   ├── recommend/
│   │   └── recommend.go           [MODIFIED: handle 7 categories]
│   └── output/
│       ├── terminal.go            [MODIFIED: 7 categories]
│       ├── json.go                [MODIFIED: 7 categories]
│       ├── html.go                [NEW: template-based HTML]
│       └── templates/             [NEW: embedded HTML templates]
│           ├── report.html
│           ├── header.html
│           ├── category.html
│           └── styles.css
├── pkg/types/
│   ├── types.go                   [MODIFIED: AnalysisTarget, Language enum, new metric types]
│   └── scoring.go
└── testdata/
    ├── valid-go-project/
    ├── valid-python-project/      [NEW]
    ├── valid-ts-project/          [NEW]
    └── polyglot-project/          [NEW]
```

## Sources

- [smacker/go-tree-sitter](https://github.com/smacker/go-tree-sitter) -- Go bindings with bundled grammars for Python, TypeScript (HIGH confidence)
- [tree-sitter/go-tree-sitter](https://github.com/tree-sitter/go-tree-sitter) -- Official modular Go bindings (HIGH confidence)
- [go-git/go-git](https://github.com/go-git/go-git) -- Pure Go git implementation for commit history analysis (HIGH confidence)
- [Claude Code headless mode docs](https://code.claude.com/docs/en/headless) -- CLI programmatic usage with `-p` flag (HIGH confidence)
- [Symflower Tree-sitter benchmarks](https://symflower.com/en/company/blog/2023/parsing-code-with-tree-sitter/) -- 36x parsing speedup (MEDIUM confidence, 2023 data)
- [GoReporter HTML reports](https://github.com/360EntSecGroup-Skylar/goreporter) -- Reference for html/template-based static analysis reports (MEDIUM confidence)
- [go-echarts standalone snippets](https://blog.cubieserver.de/2020/how-to-render-standalone-html-snippets-with-go-echarts/) -- Custom renderer for embedding charts (MEDIUM confidence)
- [html/template package](https://pkg.go.dev/html/template) -- Standard library template engine (HIGH confidence)
- [go-echarts](https://github.com/go-echarts/go-echarts) -- Evaluated but NOT recommended (see rationale above)

---
*Architecture research for: ARS v2 multi-language expansion*
*Researched: 2026-02-01*
