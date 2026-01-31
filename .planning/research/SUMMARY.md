# Project Research Summary

**Project:** ARS (Agent Readiness Score)
**Domain:** Go CLI static analysis tool for AI agent codebase assessment
**Researched:** 2026-01-31
**Confidence:** HIGH

## Executive Summary

ARS is a Go CLI static analysis tool that scores codebases on their readiness for AI agent workflows. The research reveals this is a well-established technical domain (static analysis) applied to a novel problem space (agent readiness). The recommended approach leverages Go's standard `go/ast`, `go/parser`, and `golang.org/x/tools/go/packages` stack for AST-based analysis, following proven patterns from golangci-lint and staticcheck. Build a pipeline architecture with clear separation between parsing, analysis, scoring, and output layers.

The critical insight is that ARS does not compete with existing linters -- it answers a different question ("How ready is this codebase for AI agents?") using research-backed scoring rather than arbitrary thresholds. The main technical risks are (1) attempting AST-only analysis without type information, which will require expensive refactoring later, and (2) creating a gameable scoring model that users optimize for rather than genuine quality improvement. Both are addressable through architectural decisions in the foundation phase.

The research validates that all required metrics (cyclomatic complexity, test coverage, import graph analysis, circular dependencies) are achievable with established Go tooling. No novel algorithms or experimental libraries are needed. The competitive differentiation lies in the scoring model and agent-readiness framing, not the underlying analysis engine.

## Key Findings

### Recommended Stack

Go's standard library and official extended tools provide everything needed for ARS. The core is `go/ast` + `go/parser` + `go/token` for AST parsing, `go/types` for semantic analysis, and `golang.org/x/tools/go/packages` for package loading with full dependency resolution. This is the same foundation used by golangci-lint, staticcheck, and every production Go analysis tool.

**Core technologies:**
- **Go 1.24+**: Target for modern features (tool directives in go.mod) while maintaining broad compatibility
- **`golang.org/x/tools/go/packages`**: Official package loader that handles modules, build tags, and type information correctly -- replaces the old `go/build` API
- **`spf13/cobra` v1.10.2**: De facto standard for Go CLIs, used by kubectl, hugo, gh -- provides subcommands, flags, help generation, shell completion
- **`fzipp/gocyclo`**: Library for cyclomatic complexity calculation, avoiding re-implementation of well-tested algorithms
- **`golang.org/x/tools/cover`**: Official coverage profile parsing for test coverage metrics
- **`fatih/color` v1.18.0**: Simple ANSI color output for human-friendly terminal rendering

**What to build vs import:**
- Cyclomatic complexity: Use `gocyclo` as a library (don't re-implement)
- Coupling metrics: Build on top of `go/packages` import graphs (straightforward ~50 lines)
- Test detection: Build yourself (trivial: scan for `*_test.go` files)
- Coverage parsing: Use `golang.org/x/tools/cover` (official tooling)
- Dead code detection: Build simple heuristic for MVP (unreferenced exported functions), consider full SSA-based RTA analysis in v2

**Critical architectural decision:** Build on Go's standard analysis primitives (`go/ast` + `go/types` + `go/packages`) rather than importing heavy frameworks. Most metrics are straightforward calculations on top of these primitives.

### Expected Features

Research reveals a clear split between table stakes features (required for basic usability), competitive differentiators (unique value of ARS), and anti-features (deliberately excluded to avoid complexity traps).

**Must have (table stakes):**
- Directory path as input with auto-detection of Go projects
- Non-zero exit codes for CI integration (exit 2 when below threshold)
- Per-category score breakdown (C1, C3, C6 individual scores)
- Composite score with transparent methodology showing weights
- Human-readable terminal output with color
- Actionable improvement recommendations ranked by impact
- Reasonable performance on real codebases (<30s for 50k LOC)
- Standard `--help` and `--version` flags
- Clear error messages pointing to root cause

**Should have (competitive advantage):**
- **Agent-readiness tier rating**: Agent-Ready / Agent-Assisted / Agent-Limited / Agent-Hostile labels -- this is the headline differentiator
- **Research-backed scoring model**: Weights grounded in published research (Borg et al., SWE-bench, RepoGraph) creates credibility
- **Improvement recommendations ranked by agent impact**: Frame advice in terms of agent workflows, not abstract quality metrics
- **Circular dependency detection**: Architectural issue specifically flagged in RepoGraph research as hindering agent navigation
- **Dead code detection**: Unused functions increase cognitive load for agents
- **Threshold flag for CI gating**: `--threshold 7` exits non-zero if score < 7, enabling quality gates

**Defer (v2+):**
- HTML report generation (presentation layer, defer until analysis is solid)
- Multi-language support (requires new parsers per language, validate Go-only model first)
- Score trend tracking (requires historical persistence, design for this but implement later)
- LLM-based evaluation (C7 category -- adds cost, latency, non-determinism)
- Plugin/extension system (architectural commitment, avoid public API in v1)

**Anti-features (deliberately excluded):**
- Auto-fix / code mutation: dangerous without human review, violates diagnostic-tool principle
- Granular per-file scoring: noisy and misleading, focus on project/package level
- Real-time watch mode: static analysis is not sub-second, creates false expectations
- Comparing against external benchmarks: requires maintaining database, comparisons mislead without context

### Architecture Approach

The standard architecture for Go static analysis tools is a pipeline with clear stage boundaries: Discover -> Parse -> Analyze -> Score -> Recommend -> Render. Each stage has explicit input/output contracts, enabling independent testing and natural parallelism opportunities. This pattern is proven by golangci-lint's architecture and should be followed exactly.

**Major components:**
1. **CLI Layer (cobra)**: Parse flags, validate input, wire dependencies -- no business logic
2. **Scanner**: Discover Go files, coordinate analysis pipeline -- owns file classification (`_test.go`, generated code, build-tagged files)
3. **Parser**: Thin wrapper around `go/parser.ParseFile` with shared `token.FileSet` for position tracking
4. **Analyzer Interface**: Common interface implemented by C1 (Code Health), C3 (Architecture), C6 (Testing) analyzers -- each receives parsed ASTs, returns structured metrics
5. **Scorer**: Two-phase design -- collect raw metrics first, apply configurable thresholds to compute 1-10 scores second (separation enables threshold tuning)
6. **Recommender**: Identifies top 5 improvements by distance from ideal, generates actionable text
7. **Terminal Renderer**: Formats and prints results to stdout, isolated so adding JSON/HTML later is additive

**Project structure:**
```
ars/
├── cmd/              # Cobra commands (root.go, scan.go)
├── internal/
│   ├── scanner/      # File discovery, pipeline orchestration
│   ├── parser/       # go/parser wrapper
│   ├── analyzer/     # C1, C3, C6 implementations
│   ├── scorer/       # Category + composite scoring
│   ├── recommend/    # Top-5 improvement generator
│   └── output/       # Terminal renderer
├── pkg/types/        # Shared structs (FileMetrics, Score, etc.)
└── testdata/         # Sample Go code for testing analyzers
```

**Key patterns:**
- **Pipeline architecture**: Data flows through stages, each independently testable
- **Analyzer interface**: Common contract for all categories, enables concurrent execution
- **Metric collection then scoring**: Separate measurement from judgment, makes threshold tuning independent of analysis code

**Scaling considerations:**
- Small repos (<100 files): Sequential processing, <1s total time
- Medium repos (100-1000 files): Concurrent parsing with worker pool (N=GOMAXPROCS), ~5-15s
- Large repos (1000-10000+ files): Batched processing to avoid holding all ASTs in memory, ~30s-2min
- First bottleneck is parsing (parallelize), second is memory (process in batches, release ASTs)

### Critical Pitfalls

Research identified five critical pitfalls that will break the project if not addressed early. Each has clear prevention strategies and must be mapped to specific roadmap phases.

1. **Using go/ast alone without go/types**: AST-only analysis breaks on real code with imports, type aliases, and interface satisfaction. Prevention: Use `golang.org/x/tools/go/packages` from day one to get type information alongside AST. Recovery cost is HIGH (refactor loading layer, update all analyzers, rebuild tests). **Address in Phase 1.**

2. **Loading entire repository into memory at once**: Parsing 10k+ files into ASTs simultaneously causes multi-gigabyte memory usage and OOM kills. Prevention: Process packages incrementally, release ASTs after analysis, use bounded worker pool for concurrency. golangci-lint reduced memory 1.5x by removing AST cache. Recovery cost is MEDIUM (add streaming/incremental processing). **Address in Phase 1 architecture + Phase 2 validation.**

3. **Scoring model that is gameable (Goodhart's Law)**: When scoring becomes a target, people optimize for the metric rather than quality (e.g., trivial tests to inflate coverage). Prevention: Multi-dimensional scoring with no single dominant metric, transparent methodology, validation that scores correlate with actual agent performance. Recovery cost is MEDIUM (add metrics, recalibrate weights, add gaming detection). **Address in Phase 3 scoring design.**

4. **False positives destroying user trust**: Incorrect warnings kill adoption -- one bad false positive on a well-known codebase dismisses all results. Prevention: Start with high precision/low recall, exclude generated code and vendor directories, handle Go build tags correctly, provide escape hatches (`// ars:ignore`), test against diverse real repos. Recovery cost is LOW-MEDIUM (incremental fixes). **Address in every phase, especially Phase 2 implementation.**

5. **Ignoring Go-specific file organization patterns**: Treating all `.go` files uniformly leads to incorrect analysis when `_test.go` files, platform-specific files (`_<os>.go`), build-constrained files, and generated code all have different semantics. Prevention: Classify files during traversal, use `go/packages` to understand build configurations, separate metrics for test vs production code. Recovery cost is MEDIUM (retrofit classifier). **Address in Phase 1 foundation.**

## Implications for Roadmap

Based on research, suggested phase structure follows the natural dependency order: foundation (parsing + file classification) -> core analysis (individual metrics) -> scoring model (convert metrics to scores) -> polish (recommendations + output). This order minimizes rework and enables early validation.

### Phase 1: Foundation (Parsing + File Discovery)
**Rationale:** All analysis depends on correctly loading and classifying Go files. Must establish type-aware parsing from day one (Pitfall #1) and proper file classification (Pitfall #5). This phase proves the pipeline architecture works end-to-end with one simple analyzer.

**Delivers:**
- File discovery with proper classification (`_test.go`, generated code, vendor exclusion)
- AST + type information loading via `go/packages`
- Pipeline orchestration (Scanner -> Parser -> Analyzer -> Output)
- One working analyzer (C1: cyclomatic complexity) to validate architecture

**Addresses:**
- Table stakes: Directory path input, Go project auto-detection, error messages
- Pitfall #1: type information from start
- Pitfall #5: file classification patterns

**Needs research:** No -- well-documented patterns in go/packages and golangci-lint architecture

### Phase 2: Core Analysis (C1, C3, C6 Metrics)
**Rationale:** With foundation in place, analyzers plug into the established interface. C1, C3, C6 can be developed in parallel since they're independent. This phase validates performance at scale (Pitfall #2) and minimizes false positives (Pitfall #4).

**Delivers:**
- C1 (Code Health): cyclomatic complexity, function length, file size
- C3 (Architecture): directory depth, import graph, circular dependency detection
- C6 (Testing): test file detection, test-to-code ratio, coverage via `go test -coverprofile`
- Performance validation on large repos (10k+ files)
- False positive testing on diverse open source repos

**Uses:**
- `fzipp/gocyclo` for complexity calculation
- `golang.org/x/tools/cover` for coverage parsing
- Import graph analysis via `go/packages`

**Implements:**
- Analyzer interface for each category
- Concurrent file processing with bounded worker pool

**Avoids:**
- Pitfall #2: Memory blowup (incremental processing, profiling)
- Pitfall #4: False positives (test on real repos, exclude generated code)

**Needs research:** Possibly for circular dependency detection algorithm (graph cycle detection) if team is unfamiliar -- otherwise standard patterns

### Phase 3: Scoring Model
**Rationale:** Scoring is the last layer -- requires all metrics to be collected first. This phase implements the weighted composite score (C1: 25%, C3: 20%, C6: 15%) and tier rating system. Critical to address gameability (Pitfall #3) through transparent methodology and multi-dimensional weighting.

**Delivers:**
- Per-category scoring with configurable thresholds
- Composite score calculation with documented weights
- Tier rating (Agent-Ready / Assisted / Limited / Hostile)
- Score validation on 5+ diverse repos (does higher score correlate with better agent readiness?)
- Verbose mode showing per-metric breakdown

**Addresses:**
- Differentiator: Research-backed scoring model
- Differentiator: Agent-readiness tier rating
- Table stakes: Composite score with clear methodology
- Pitfall #3: Gameability through multi-dimensional metrics and transparency

**Needs research:** No -- scoring thresholds and weights are defined in PROJECT.md, implementation is straightforward

### Phase 4: Recommendations + Output
**Rationale:** With scoring complete, recommendations identify top improvements by score impact. Terminal output makes results actionable. This phase delivers the complete user experience.

**Delivers:**
- Top 5 improvement recommendations ranked by impact
- Terminal output with color (fatih/color)
- Exit codes (0/1/2) with `--threshold` flag for CI gating
- `--help` and `--version` flags
- Clear error messages with actionable guidance

**Addresses:**
- Differentiator: Recommendations ranked by agent impact
- Table stakes: Human-readable output, CI integration, help text
- Differentiator: Threshold flag for quality gates

**Needs research:** No -- terminal rendering and recommendation ranking are application logic

### Phase 5: Real-world Validation
**Rationale:** Test against 10+ open source Go repos of varying sizes and styles. Validate that scores are meaningful, reproducible, and correlate with actual agent readiness. Fix any false positives, performance issues, or UX problems discovered.

**Delivers:**
- Validation on diverse repos (CLI tools, web servers, libraries)
- Performance benchmarks (time and memory on repos of different scales)
- False positive rate analysis
- Edge case handling (symlinks, syntax errors, Unicode paths)
- Reproducibility verification (same repo -> same score)

**Addresses:**
- Pitfall #4: False positives on real code
- Table stakes: Reasonable performance on real codebases
- All UX pitfalls from research

**Needs research:** No -- this is testing and validation

### Phase Ordering Rationale

- **Foundation first** because parsing + type information is a prerequisite for all analysis
- **Core analysis second** because metrics must exist before scoring them
- **Scoring third** because it's a layer on top of collected metrics
- **Recommendations + output fourth** because they consume scored results
- **Real-world validation last** but with early smoke testing throughout

This order minimizes rework: establishing type-aware parsing in Phase 1 avoids expensive refactoring later (Pitfall #1). Building scoring as the last layer (after understanding what we can reliably measure) avoids premature threshold decisions. Each phase delivers working software that can be tested.

**Dependency flow:**
```
Phase 1 (Foundation)
    |
    v
Phase 2 (Core Analysis) -- can work on C1, C3, C6 in parallel
    |
    v
Phase 3 (Scoring Model) -- requires all metrics collected
    |
    v
Phase 4 (Recommendations + Output) -- requires scores computed
    |
    v
Phase 5 (Real-world Validation) -- validates everything
```

### Research Flags

Phases with standard patterns (skip research-phase):
- **Phase 1**: Well-documented go/packages patterns, golangci-lint architecture reference
- **Phase 3**: Scoring implementation is straightforward application logic
- **Phase 4**: Terminal output and CLI patterns are standard
- **Phase 5**: Testing and validation, no research needed

Phases possibly needing deeper research during planning:
- **Phase 2**: Only if team lacks graph algorithm experience for circular dependency detection (DFS cycle detection). Otherwise skip -- import graph construction is well-documented in go/packages examples.

**Overall assessment:** This project needs minimal additional research. The domain (static analysis) is well-established with extensive documentation. All required libraries are mature and official Go tooling. Research-phase can be skipped for all phases unless specific implementation questions arise during planning.

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | All recommendations use official Go tooling (go/ast, go/packages, x/tools) or de facto standards (cobra, gocyclo). Versions verified as current. Zero experimental dependencies. |
| Features | HIGH | Table stakes validated against golangci-lint, goreportcard, SonarQube patterns. Differentiators are research-backed (agent readiness framing grounded in SWE-bench findings). Anti-features identified from failed metric systems (code coverage gaming, etc.). |
| Architecture | HIGH | Pipeline pattern proven by golangci-lint architecture. Component boundaries match standard Go project layouts. Scaling considerations based on profiled performance data. All patterns have reference implementations. |
| Pitfalls | HIGH | All five critical pitfalls sourced from production Go analysis tool experiences (golangci-lint issues, staticcheck philosophy, go-tools commit history). Prevention strategies tested in real projects. |

**Overall confidence:** HIGH

### Gaps to Address

No significant gaps identified. All required information for roadmap creation is available with high confidence. Minor points to validate during implementation:

- **Test coverage assertion density calculation**: Research mentions this as a C6 metric but doesn't detail the algorithm. Defer to v1.x if implementation is unclear -- simple assertion counting via AST traversal is sufficient for MVP.
- **Exact threshold values for scoring**: PROJECT.md defines weights (C1: 25%, C3: 20%, C6: 15%) but individual metric thresholds (e.g., "complexity > 15 = score 2/10") need calibration. Plan to make configurable from start and tune during validation phase.
- **Cognitive complexity vs cyclomatic complexity**: Research mentions gocognit as potentially better than gocyclo for agent readiness. Evaluate during Phase 2 implementation -- start with cyclomatic (simpler, established), consider cognitive as enhancement.

None of these gaps block roadmap creation. All are implementation details that can be resolved during the corresponding phase.

## Sources

### Primary (HIGH confidence)
- [golang.org/x/tools/go/packages](https://pkg.go.dev/golang.org/x/tools/go/packages) -- Official package loading API
- [golang.org/x/tools/go/analysis](https://pkg.go.dev/golang.org/x/tools/go/analysis) -- Official analysis framework
- [golangci-lint architecture docs](https://golangci-lint.run/docs/contributing/architecture/) -- Production pipeline pattern
- [spf13/cobra releases](https://github.com/spf13/cobra/releases) -- v1.10.2 released Dec 2024
- [fzipp/gocyclo](https://github.com/fzipp/gocyclo) -- Cyclomatic complexity library
- [go/ast package docs](https://pkg.go.dev/go/ast) -- AST structure reference
- [staticcheck](https://staticcheck.dev/) -- Precision-over-recall philosophy

### Secondary (MEDIUM confidence)
- [Anthropic Agentic Coding Trends 2026](https://resources.anthropic.com/hubfs/2026%20Agentic%20Coding%20Trends%20Report.pdf) -- Agent readiness context
- [Stack Overflow: Code coverage worse with better code](https://stackoverflow.blog/2025/12/22/making-your-code-base-better-will-make-your-code-coverage-worse/) -- Metric gaming examples
- [Cloudflare: Building Go static analysis tool](https://blog.cloudflare.com/building-the-simplest-go-static-analysis-tool/) -- AST traversal patterns
- [golangci-lint memory optimization commit](https://github.com/golangci/golangci-lint/commit/df4f6766baff8f2ce10ae7a6a4d81fe37b729989) -- AST cache removal performance data

### Tertiary (LOW confidence)
- None -- all research findings are backed by primary or secondary sources

---
*Research completed: 2026-01-31*
*Ready for roadmap: yes*
