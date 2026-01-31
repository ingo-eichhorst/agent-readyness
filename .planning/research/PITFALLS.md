# Pitfalls Research

**Domain:** Static analysis CLI tool (Go, AST-based code quality scoring)
**Researched:** 2026-01-31
**Confidence:** HIGH (well-documented domain with many prior art examples)

## Critical Pitfalls

### Pitfall 1: Using go/ast Alone Without go/types

**What goes wrong:**
The `go/ast` package only captures syntactic structure -- it does not know what identifiers refer to, what types expressions have, or how packages relate. Teams start building analyzers on pure AST and hit a wall when they need to distinguish between a local function call and an imported one, or determine whether a variable is an `error` type. This leads to either false positives (flagging things incorrectly) or massive rewrites to add type information later.

**Why it happens:**
`go/ast` is the first thing you encounter in tutorials. It feels sufficient for simple checks. The need for type information only becomes apparent when you try to build anything beyond trivial pattern matching -- for example, "does this function return an error?" requires type checking, not just AST inspection.

**How to avoid:**
- Use `golang.org/x/tools/go/packages` to load packages from the start, not raw `parser.ParseFile`. This gives you access to `types.Info` alongside the AST.
- For any check that involves "what kind of thing is this identifier?", plan for `go/types` from day one.
- If building reusable analyzers, use `golang.org/x/tools/go/analysis` framework which provides pre-built type information via `analysis.Pass`.
- For ARS specifically: you likely need type info for dependency graph analysis and function complexity. Wire it in from the beginning.

**Warning signs:**
- Analyzer works on simple test files but breaks on real code with imports
- You find yourself string-matching package names instead of resolving them
- Edge cases multiply as you encounter type aliases, embedded structs, interface satisfaction

**Phase to address:**
Phase 1 (Foundation). The parsing/loading architecture must include type information from the start. Retrofitting is expensive.

**Real-world example:**
golangci-lint's architecture separates linters by "load mode" -- AST-only linters need minimal data, but linters requiring type info force a heavier load for ALL packages. This architectural decision from early on shapes everything. [Source: golangci-lint architecture docs](https://golangci-lint.run/docs/contributing/architecture/)

---

### Pitfall 2: Loading Entire Repository Into Memory At Once

**What goes wrong:**
Parsing every file in a 10k+ file repository into ASTs simultaneously consumes enormous memory. Go ASTs are verbose data structures -- a single large file can produce an AST consuming 10-50x the source file size in memory. For a large repo, this means multi-gigabyte memory usage, OOM kills in CI, and unusable performance on developer laptops.

**Why it happens:**
The naive approach is: walk all files, parse all files, analyze all files, report. This works for small repos and test suites. It only breaks when hitting real-world scale.

**How to avoid:**
- Process packages or directories incrementally, not the whole repo at once.
- Release ASTs after analysis of each package completes (do not cache them). golangci-lint found that [getting rid of the AST cache reduced memory by ~1.5x](https://github.com/golangci/golangci-lint/commit/df4f6766baff8f2ce10ae7a6a4d81fe37b729989).
- Use `runtime.GC()` hints between package processing if memory is tight.
- Set `GOGC` tuning for your memory/speed tradeoff (lower GOGC = more frequent GC = less memory, more CPU).
- Profile memory early with `pprof` on a real large repo, not just test fixtures.

**Warning signs:**
- Tool works fine on your test repo (100 files) but OOMs on real repos
- Memory usage scales linearly (or worse) with repo size
- CI runners with 4GB RAM start failing

**Phase to address:**
Phase 1 (Foundation) for architecture. Phase 2 (Core Analysis) for validation with real repos.

**Real-world example:**
golangci-lint explicitly optimized to load ASTs on-demand and release them after use, rather than caching. GoMetaLinter's approach of spawning separate subprocesses per linter was abandoned precisely because it was resource-inefficient at scale. [Source: golangci-lint FAQ](https://golangci-lint.run/docs/welcome/faq/)

---

### Pitfall 3: Scoring Model That Is Gameable (Goodhart's Law)

**What goes wrong:**
When a scoring metric becomes a target, people optimize for the metric rather than the underlying quality. If ARS scores "test coverage percentage" heavily, teams will write trivial tests to inflate coverage. If it scores "function length," teams will split functions artificially. The score becomes meaningless -- high scores do not correlate with actual agent readiness.

**Why it happens:**
Every proxy metric diverges from the real goal when optimized directly. Code coverage is a proxy for "thoroughly tested" but diverges when low-value tests inflate numbers. Function count is a proxy for "modular code" but diverges when functions are split unnecessarily. This is well-documented across the software metrics literature.

**How to avoid:**
- Use multi-dimensional scoring: no single metric should dominate. Combine structural metrics (complexity, coupling) with behavioral metrics (test coverage, error handling patterns) with documentation metrics.
- Weight metrics to reflect actual agent-readiness, not generic "code quality."
- Make the scoring model transparent: show users WHAT is being measured and WHY, so they improve the right things.
- Include "smell detection" that flags gaming patterns (e.g., test files with no assertions, functions that are just wrappers).
- Design scores as diagnostic, not as pass/fail gates. "Here is what to improve" is more valuable than "your score is 72."
- Regularly validate: does a higher ARS score actually correlate with better agent performance on the repo? If not, recalibrate.

**Warning signs:**
- Users report "I improved my score but my code is not actually better"
- Score changes dramatically from trivial changes (e.g., adding empty test files)
- Teams game the system rather than improving genuine readiness
- Score does not correlate with real-world agent success on the codebase

**Phase to address:**
Phase 3 (Scoring Model). But the metric design should be informed by this pitfall from Phase 1. Build scoring as the LAST layer, after you understand what you can reliably measure.

**Real-world example:**
SonarQube's code coverage gates are widely discussed as gameable. Stack Overflow published an article arguing that [making code DRYer actually makes coverage metrics worse](https://stackoverflow.blog/2025/12/22/making-your-code-base-better-will-make-your-code-coverage-worse/), illustrating how metrics can be anti-correlated with actual quality.

---

### Pitfall 4: False Positives Destroying User Trust

**What goes wrong:**
The tool flags issues that are not actually problems. Users see incorrect warnings, lose trust in the tool, and stop using it entirely. One bad false positive on a user's well-known codebase is enough to dismiss all results. For a scoring tool, false positives mean the score is wrong, which means the tool is useless.

**Why it happens:**
Static analysis is inherently imprecise -- you are reasoning about code without running it. Common causes: not handling language edge cases (build tags, generated code, test files vs production code), overly aggressive pattern matching, not understanding context (e.g., flagging unused error returns in test helpers where it is intentional).

**How to avoid:**
- Start with HIGH precision, LOW recall. It is better to miss some issues than to report false ones. You can always add more checks later; you cannot recover lost trust.
- Exclude generated code (files with `// Code generated` headers).
- Exclude vendor directories, test fixtures, and build artifacts.
- Handle Go build tags correctly -- files with `//go:build ignore` should not be analyzed.
- Provide escape hatches: `// ars:ignore` comments or `.arsignore` files.
- Test every rule against diverse real-world repos, not just synthetic test cases.
- Track false positive reports and fix them aggressively.

**Warning signs:**
- Users report "this is wrong" on GitHub issues
- Tool flags generated code, vendored dependencies, or test fixtures
- Different Go versions or build configurations produce different results
- Score fluctuates without code changes (non-deterministic analysis)

**Phase to address:**
Every phase. But especially Phase 2 (Core Analysis) where individual checks are implemented, and Phase 4 (Real-world Testing) where you validate against open source repos.

**Real-world example:**
Staticcheck and golangci-lint both invest heavily in minimizing false positives. Staticcheck explicitly avoids checks that would have high false positive rates, even at the cost of missing real issues. [Source: Staticcheck docs](https://staticcheck.dev/)

---

### Pitfall 5: Ignoring Go-Specific File Organization Patterns

**What goes wrong:**
The analyzer treats all `.go` files uniformly, but Go has significant conventions that change how files should be analyzed: `_test.go` files are test code with different rules, `_<os>.go` and `_<arch>.go` files are platform-specific, files with build constraints may not compile on the current platform, `internal/` packages have visibility restrictions, and `cmd/` directories contain entry points. Ignoring these distinctions leads to incorrect analysis.

**Why it happens:**
When building a general "walk all Go files" analyzer, it is easy to forget that Go's file naming conventions carry semantic meaning. The parser will happily parse a `_test.go` file, but its functions are in a different package (`package foo_test`) and should not count toward production code metrics.

**How to avoid:**
- Classify files during traversal: production code, test code, generated code, build-constrained code.
- Use `go/build` or `go/packages` to understand which files belong to which build configuration, rather than walking the filesystem manually.
- Separate metrics for test code vs production code (test coverage of tests is meaningless).
- Handle the `_test` package suffix correctly (external test packages).
- Skip `vendor/`, `.git/`, `node_modules/`, and other non-source directories.

**Warning signs:**
- Test file complexity inflates the "production code complexity" score
- Platform-specific files are analyzed even when they would not compile on the analysis platform
- Generated protobuf files dominate metrics

**Phase to address:**
Phase 1 (Foundation) for file classification. Phase 2 (Core Analysis) for per-category metric separation.

---

## Technical Debt Patterns

Shortcuts that seem reasonable but create long-term problems.

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|----------------|-----------------|
| Using `parser.ParseFile` instead of `go/packages` | Simpler code, no dependency resolution | Cannot resolve types, imports, or cross-package references; must rewrite for any non-trivial analysis | Never for this project -- ARS needs cross-package understanding |
| Hardcoding metric thresholds | Quick to implement scoring | Every codebase is different; thresholds that work for web apps fail for CLI tools or libraries | MVP only, with plan to make configurable in next phase |
| String matching instead of AST matching | Faster to write checks | Breaks on formatting changes, comments, multi-line expressions; high false positive rate | Never -- the whole point of AST analysis is to avoid this |
| Skipping concurrency in file processing | Simpler code, fewer race conditions | Unacceptable performance on large repos (10k+ files); users abandon tool | MVP only if profiling shows it is fast enough; plan concurrent processing early |
| Rolling your own test framework instead of using `analysistest` | Feels more flexible | Misses edge cases in diagnostic testing; duplicates work the Go team already solved | Never -- `analysistest` is purpose-built for this |

## Integration Gotchas

Common mistakes when connecting to external data sources or formats.

| Integration | Common Mistake | Correct Approach |
|-------------|----------------|------------------|
| Test coverage parsing (Go) | Assuming `go test -coverprofile` always produces the same format | Parse with `golang.org/x/tools/cover` package; handle missing coverage gracefully (zero coverage is valid, not an error) |
| Test coverage parsing (multi-format) | Building custom parsers for each format (lcov, cobertura, etc.) | Use existing parsing libraries; standardize on an intermediate representation; accept that some formats lose information |
| Git integration (for file history/churn) | Shelling out to `git log` and parsing text output | Use `go-git` library for programmatic access; handle repos without git (tarballs, downloads); handle shallow clones in CI |
| Go module resolution | Assuming `go.mod` is always in the root directory | Walk up directory tree to find `go.mod`; handle multi-module repos (workspaces); handle repos without `go.mod` (pre-modules code) |
| `.gitignore` / file exclusion | Re-implementing gitignore pattern matching | Use a tested library (e.g., `go-gitignore`); gitignore semantics are surprisingly complex (negation, directory-only patterns, nested `.gitignore` files) |

## Performance Traps

Patterns that work at small scale but fail as repo size grows.

| Trap | Symptoms | Prevention | When It Breaks |
|------|----------|------------|----------------|
| `filepath.Walk` instead of `filepath.WalkDir` | Unnecessary `os.Stat` calls on every file; ~1.5x slower on local FS, much worse on network FS | Use `filepath.WalkDir` (Go 1.16+); for extreme perf, use `charlievieth/fastwalk` for parallel traversal | 1k+ files; especially noticeable on NFS/networked filesystems |
| Parsing all files before analyzing any | Memory spike; long time-to-first-result | Stream processing: parse file, analyze, report, release AST, move to next | 5k+ files or files with large ASTs |
| Not skipping irrelevant directories | Traversing `vendor/`, `node_modules/`, `.git/`, `testdata/` with binary fixtures | Skip known non-source directories early in walk function via `filepath.SkipDir` | Any repo with vendored dependencies |
| Single-threaded AST parsing | CPU-bound on single core while other cores idle | Use worker pool with `GOMAXPROCS` goroutines; each goroutine parses one package | 3k+ files; ~4x speedup on modern hardware |
| Repeated package loading | Loading the same package multiple times across different analyzers | Load once, share results across all checks; this is exactly what `go/analysis` framework does | When running multiple checks (which ARS will) |
| Unbounded goroutine spawning | One goroutine per file = thousands of goroutines = scheduler overhead, memory bloat | Use bounded worker pool (e.g., `semaphore.Weighted` or buffered channel) | 10k+ files |

## UX Pitfalls

Common user experience mistakes for CLI analysis tools.

| Pitfall | User Impact | Better Approach |
|---------|-------------|-----------------|
| No progress indication on large repos | User thinks tool is frozen; kills it after 30 seconds | Show progress: "Analyzing package 42/156..." or a spinner with current package name |
| Dumping all results to stdout with no structure | Wall of text; user cannot find what matters | Use structured output (JSON for machines, colored/grouped terminal output for humans); sort by severity |
| No way to suppress known issues | User cannot adopt tool incrementally; must fix everything or see same warnings forever | Support baseline files, `// ars:ignore` comments, and `.arsignore` glob patterns |
| Unclear scoring explanation | "Your score is 62" means nothing without context | Show breakdown: "Score 62/100: Complexity 8/20, TestCoverage 15/25, ..." with explanations of what each sub-score means |
| Exit code does not reflect severity | CI pipeline cannot distinguish "warnings only" from "critical issues" | Use different exit codes: 0 = clean, 1 = warnings, 2 = errors, non-zero for tool failures |
| No machine-readable output | Cannot integrate into CI pipelines, dashboards, or other tools | Support `--format json` and `--format text` from day one; JSON is the integration format |

## "Looks Done But Isn't" Checklist

Things that appear complete but are missing critical pieces.

- [ ] **File walker:** Often missing symlink handling -- verify behavior when repo contains symlinks (follow? skip? error?)
- [ ] **AST parser:** Often missing error recovery -- verify behavior on files with syntax errors (skip file? partial parse? crash?)
- [ ] **Metric calculation:** Often missing normalization -- verify that a 10-line file and a 10,000-line file produce comparable scores (not raw counts)
- [ ] **Test coverage parsing:** Often missing branch coverage -- verify you handle line coverage vs branch coverage vs statement coverage distinctions
- [ ] **Dependency analysis:** Often missing indirect dependencies -- verify you handle transitive dependencies, not just direct imports
- [ ] **Score output:** Often missing reproducibility -- verify the same repo at the same commit always produces the exact same score (no timestamps, random ordering, or environment-dependent results)
- [ ] **CLI flags:** Often missing `--help` documentation quality -- verify every flag has a description, every command has examples
- [ ] **Error messages:** Often missing actionable guidance -- verify errors say what to do, not just what went wrong ("file not found: go.mod -- run from a Go module root or specify --dir")
- [ ] **Large file handling:** Often missing timeout/limit -- verify behavior on pathologically large files (100k+ lines of generated code)
- [ ] **Unicode handling:** Often missing -- verify file paths and source code with non-ASCII characters work correctly

## Recovery Strategies

When pitfalls occur despite prevention, how to recover.

| Pitfall | Recovery Cost | Recovery Steps |
|---------|---------------|----------------|
| AST-only without types | HIGH | Refactor loading layer to use `go/packages`; update all analyzers to accept `types.Info`; rebuild tests with type-aware fixtures |
| Memory blowup on large repos | MEDIUM | Add streaming/incremental processing; profile with pprof to find retention; may require architecture change to processing pipeline |
| Gameable scoring model | MEDIUM | Add multi-dimensional metrics; make weights configurable; add gaming detection heuristics; communicate score meaning better |
| False positive epidemic | LOW-MEDIUM | Add suppression mechanism; tighten pattern matching; add real-world test corpus; each fix is incremental |
| Missing file classification | MEDIUM | Retrofit file classifier into walk phase; update all metrics to filter by classification; re-validate all scores |
| Poor CLI UX | LOW | Incremental improvements; add progress bars, structured output, better error messages one at a time |

## Pitfall-to-Phase Mapping

How roadmap phases should address these pitfalls.

| Pitfall | Prevention Phase | Verification |
|---------|------------------|--------------|
| AST-only without types | Phase 1: Foundation | Verify `types.Info` is available in analysis pass; write one check that requires type info |
| Memory blowup | Phase 1: Foundation + Phase 2: Validation | Profile with pprof on a 10k+ file repo; memory stays under 1GB |
| Gameable scoring | Phase 3: Scoring Model | Validate score correlates with actual agent readiness on 5+ diverse repos |
| False positives | Phase 2: Core Analysis | Run on 10+ open source repos; zero false positives on common patterns |
| File classification | Phase 1: Foundation | Test file walker correctly categorizes `_test.go`, generated code, vendor, build-tagged files |
| Performance at scale | Phase 2: Core Analysis | Benchmark on large repo; complete analysis in under 30 seconds for 10k files |
| Poor UX | Phase 3: CLI Polish | User test with 3+ developers; they can understand output without documentation |
| Scoring transparency | Phase 3: Scoring Model | Output includes per-metric breakdown; user can identify what to improve |
| Non-deterministic results | Phase 2: Core Analysis | Run tool twice on same repo; diff output is empty |
| Missing escape hatches | Phase 2: Core Analysis | Support `// ars:ignore` and `.arsignore` before public release |

## Sources

- [golangci-lint architecture](https://golangci-lint.run/docs/contributing/architecture/) -- load modes, work sharing, AST caching decisions
- [golangci-lint FAQ](https://golangci-lint.run/docs/welcome/faq/) -- performance tuning, GOGC, caching
- [golangci-lint memory optimization commit](https://github.com/golangci/golangci-lint/commit/df4f6766baff8f2ce10ae7a6a4d81fe37b729989) -- AST cache removal
- [golangci-lint performance regression issue](https://github.com/golangci/golangci-lint/issues/5546) -- version upgrade pitfalls
- [golang.org/x/tools/go/analysis](https://pkg.go.dev/golang.org/x/tools/go/analysis) -- official analysis framework
- [golang.org/x/tools/go/analysis/analysistest](https://pkg.go.dev/golang.org/x/tools/go/analysis/analysistest) -- testing analyzers
- [Staticcheck](https://staticcheck.dev/) -- precision-over-recall philosophy
- [PVS-Studio: How to create your own Go static analyzer](https://pvs-studio.com/en/blog/posts/go/1329/) -- go/ast vs go/types distinction
- [Rauljordan: Custom Static Analysis in Go](https://rauljordan.com/custom-static-analysis-in-go-part-1/) -- AST traversal patterns
- [filepath.WalkDir proposal in Staticcheck](https://github.com/dominikh/go-tools/issues/1014) -- Walk vs WalkDir performance
- [charlievieth/fastwalk](https://github.com/charlievieth/fastwalk) -- parallel file traversal benchmarks
- [Stack Overflow: Making your code base better will make your code coverage worse](https://stackoverflow.blog/2025/12/22/making-your-code-base-better-will-make-your-code-coverage-worse/) -- metrics gaming
- [Goodhart's Law in Software Engineering (Jellyfish)](https://jellyfish.co/blog/goodharts-law-in-software-engineering-and-how-to-avoid-gaming-your-metrics/) -- scoring model pitfalls
- [Goodhart's Law: The Hidden Risk in Software Engineering Metrics (Axify)](https://axify.io/blog/goodhart-law) -- metric manipulation patterns

---
*Pitfalls research for: ARS (Agent Readiness Score) -- Go static analysis CLI tool*
*Researched: 2026-01-31*
