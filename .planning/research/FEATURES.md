# Feature Research

**Domain:** CLI static analysis tool (Go codebase scoring for AI agent readiness)
**Researched:** 2026-01-31
**Confidence:** HIGH (well-established domain with many reference implementations)

## Feature Landscape

### Table Stakes (Users Expect These)

Features users assume exist. Missing these = product feels incomplete or unusable.

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| Directory path as input | Every CLI analysis tool accepts a target path; `ars scan <dir>` is the minimum | LOW | Support `.` for current directory. Must validate path exists and contains Go files. |
| Auto-detection of Go projects | Users should not need to tell the tool what language they are scanning. golangci-lint, staticcheck, and goreportcard all auto-detect. | LOW | Look for `go.mod` and `*.go` files. Warn clearly if directory is not a Go project. |
| Non-zero exit codes for CI | CI pipelines need machine-readable pass/fail. golangci-lint exits non-zero on findings; Veracode and Salesforce Code Analyzer use `--severity-threshold` flags. This is the standard pattern. | LOW | Exit 0 = success, 1 = error, 2 = below threshold (when `--threshold` specified). Matches PROJECT.md spec. |
| Per-category score breakdown | SonarQube, Code Climate, and goreportcard all show per-dimension scores (maintainability, reliability, etc.). A single opaque number is not actionable. | MEDIUM | Show C1, C3, C6 individual scores alongside composite. Each category needs metric-level detail. |
| Composite score with clear methodology | Users need to understand WHY they got a 6/10. SonarQube documents its SQALE methodology; Code Climate explains its GPA calculation. Opaque scores destroy trust. | MEDIUM | Document the weighting (C1: 25%, C3: 20%, C6: 15%) and how each metric maps to the score. Print weights in verbose mode. |
| Human-readable terminal output | golangci-lint defaults to colored text with source lines. Every successful CLI tool has good default terminal output. | MEDIUM | File locations, metric values, and the composite score. Use color for tier ratings (green=Agent-Ready, red=Agent-Hostile). Degrade gracefully when piped (no color). |
| Actionable recommendations | SonarQube shows "fix this because X." Users expect to know what to improve, not just what's wrong. Without this, the score is just a number. | MEDIUM | Top 5 ranked by impact. Each recommendation should say: what to fix, where (file/function), why it matters, and estimated impact on score. |
| Reasonable performance on real codebases | golangci-lint runs in seconds on large codebases via parallelism and caching. A tool that takes 30 minutes on a medium repo is dead on arrival. | MEDIUM | Target: <30s for a typical Go project (50k LOC), <5min for very large projects (10k+ files per PROJECT.md constraint). |
| `--help` and `--version` flags | Universal CLI convention. Every tool has these. Missing them signals amateur-hour. | LOW | Use Go's `cobra` or `flag` package. Include usage examples in help text. |
| Error messages that point to root cause | When scanning fails (no Go files, parse errors, permission issues), the error must tell the user what went wrong and what to do about it. | LOW | "No go.mod found in /path. Are you pointing at a Go project?" not "error: nil pointer dereference". |

### Differentiators (Competitive Advantage)

Features that set ARS apart. These are not expected in a generic linter but are the unique value of an agent-readiness scoring tool.

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| Agent-readiness framing (tier rating) | No existing tool scores codebases for AI agent readiness. golangci-lint finds bugs; SonarQube measures maintainability. ARS answers "will an AI agent succeed here?" -- a novel and timely question. | LOW | Tier labels (Agent-Ready / Agent-Assisted / Agent-Limited / Agent-Hostile) are the single most memorable output. This is the headline feature. |
| Research-backed scoring model | ARS scoring is grounded in published research (Borg et al., SWE-bench, RepoGraph). Competitors use arbitrary thresholds. Research backing creates credibility with engineering leaders. | MEDIUM | Cite the research in docs and verbose output. "Test coverage is weighted at 15% because SWE-bench shows 47% correlation with agent task completion." |
| Improvement recommendations ranked by agent impact | Generic linters say "reduce complexity." ARS should say "reducing complexity in pkg/parser will have the highest impact on agent success because agents struggle with functions over 50 LOC." | MEDIUM | Requires mapping metric deltas to score impact. Frame recommendations in terms of agent workflows, not abstract quality. |
| Circular dependency detection | Most linters check syntax/style. Circular dependencies are an architectural issue that specifically hinders agent navigation (RepoGraph finding). golangci-lint does not check this natively. | MEDIUM | Build import graph, detect cycles using DFS. Report the cycle path. This is a C3 metric that directly predicts agent confusion. |
| Dead code detection | Unused functions/types increase cognitive load for agents navigating codebases. `deadcode` exists as a standalone tool but framing it as agent-readiness is novel. | MEDIUM | Use `golang.org/x/tools` packages or build a reachability analysis from exported entry points. |
| Threshold flag for CI gating | `ars scan --threshold 7 .` exits non-zero if score < 7. Turns the tool into a CI quality gate specifically for agent readiness. Salesforce Code Analyzer and Veracode use the same pattern. | LOW | Simple: compare composite score to threshold, set exit code accordingly. Very high value-to-effort ratio. |
| Score trend tracking (future) | Compare current score to previous run. "Your agent-readiness improved from 5.2 to 6.8." No competitor does this for agent readiness. | HIGH | Requires persisting historical scores (local file or CI artifact). Defer to post-MVP but design score output to be machine-parseable from day 1. |

### Anti-Features (Commonly Requested, Often Problematic)

Features that seem good but create problems. Deliberately NOT building these.

| Feature | Why Requested | Why Problematic | Alternative |
|---------|---------------|-----------------|-------------|
| Auto-fix / code mutation | "If you know what's wrong, just fix it." Tempting for complexity or style issues. | Automated fixes without human review are dangerous. They can break semantics, introduce bugs, and undermine trust. SonarQube and golangci-lint explicitly separate analysis from mutation. ARS is a diagnostic tool, not a surgeon. | Provide specific, actionable recommendations with file/line references. Let humans (or their AI agents) decide how to fix. |
| HTML report in v1 | "I want to share results with my manager." Visual reports feel polished. | HTML generation adds complexity (templating, CSS, asset management), testing burden, and maintenance cost. It's a presentation layer concern that should not delay core analysis accuracy. | Ship terminal text first. Add `--format json` in v1.x so users can build their own dashboards. HTML in Phase 2. |
| Multi-language support in v1 | "We have Python and TypeScript too." Natural request for polyglot teams. | Each language needs its own parser, AST, complexity calculator, and test detection. Doing one language well is hard; doing three poorly is useless. golangci-lint is Go-only and is the most popular Go tool precisely because of that focus. | Ship Go-only. Validate the scoring model works. Add languages only after proving the methodology on one language. |
| Granular per-file scoring | "Show me the score for every file." Feels comprehensive. | Per-file scores are noisy, overwhelming, and often misleading (a 10-line utility file with no tests is fine; a 2000-line file with no tests is not, but per-file scoring treats them the same). Cognitive overload kills adoption. | Score at the project and package level. Call out specific files only in recommendations ("the worst offenders"). |
| Plugin/extension system | "Let users add their own analyzers." Extensibility sounds great. | Plugin systems are architectural commitments that constrain your core design. They require stable APIs, documentation, versioning, and support. golangci-lint has one, and it's their biggest maintenance burden. | Keep the analyzer interface clean internally (for your own development velocity) but do not expose it publicly in v1. |
| Real-time watch mode | "Re-run analysis on file save." IDE-like experience. | Static analysis of an entire codebase is not a sub-second operation. Watch mode creates expectations of instant feedback that analysis tools cannot meet. It also adds filesystem watching complexity. | Focus on fast single-run performance. Let users integrate with their editor's save hooks if they want re-runs. |
| Comparing against external benchmarks | "How does my repo compare to Kubernetes?" Benchmarking sounds valuable. | Requires maintaining a database of scores for public repos. Different repos have different valid architectures. Comparisons without context are misleading. | Provide tier labels (Agent-Ready, etc.) as the benchmark. "You are Agent-Limited" is more actionable than "You are worse than Kubernetes." |
| LLM-based evaluation in v1 | "Use GPT to assess code quality." Cutting-edge appeal. | Adds API costs, latency, non-determinism, and a runtime dependency on external services. Makes the tool unusable offline and in air-gapped environments. Scoring should be deterministic and free. | Pure static analysis first. LLM evaluation (C7) is explicitly out of scope per PROJECT.md. Revisit when the static scoring model is validated. |

## Feature Dependencies

```
[Go Project Detection]
    |
    +--requires--> [AST Parsing / File Walking]
    |                   |
    |                   +--requires--> [C1: Code Health Metrics]
    |                   |                   - cyclomatic complexity
    |                   |                   - function length
    |                   |                   - file size
    |                   |                   - coupling (import analysis)
    |                   |                   - duplication detection
    |                   |
    |                   +--requires--> [C3: Architectural Navigability]
    |                   |                   - directory depth
    |                   |                   - module fanout
    |                   |                   - circular dependencies (import graph)
    |                   |                   - import complexity
    |                   |                   - dead code detection
    |                   |
    |                   +--requires--> [C6: Testing Infrastructure]
    |                                       - test file detection (*_test.go)
    |                                       - test-to-code ratio
    |                                       - coverage (via `go test -cover`)
    |                                       - assertion density
    |
    +--all three categories feed into-->
    |
[Composite Score Calculation] (weighted: C1 25%, C3 20%, C6 15%)
    |
    +--requires--> [Tier Rating] (maps score ranges to labels)
    |
    +--requires--> [Recommendations Engine]
    |                   - identifies lowest-scoring metrics
    |                   - maps to actionable improvements
    |                   - ranks by score impact
    |
    +--requires--> [Terminal Output Renderer]
    |                   - formatted text with color
    |                   - respects --no-color / pipe detection
    |
    +--optional--> [Threshold Gate] (--threshold flag, exit code 2)
```

### Dependency Notes

- **AST Parsing is the foundation:** All three analysis categories (C1, C3, C6) depend on Go source file parsing. Invest in a solid file walker and parser first. Use `go/parser` and `go/ast` from the standard library.
- **C1, C3, C6 are independent of each other:** They can be developed and tested in parallel once the parser layer exists. This enables parallel development or incremental delivery.
- **Composite score requires all categories:** Cannot produce a meaningful composite until at least the core metrics from each category are working. However, individual category scores can be shown earlier.
- **Recommendations depend on score calculation:** The engine needs to know which metrics dragged the score down to generate useful recommendations. Build scoring first, recommendations second.
- **Terminal output is the final layer:** Do not over-invest in output formatting until the underlying analysis is correct. Pretty output on wrong numbers is worse than ugly output on right numbers.
- **Threshold gate is a thin wrapper:** Once composite score exists, threshold comparison is trivial. Low effort, high CI value.

## MVP Definition

### Launch With (v1)

Minimum viable product -- what's needed to validate the scoring model and be useful.

- [x] `ars scan <directory>` command with Go project auto-detection
- [x] C1: Code Health -- cyclomatic complexity (gocyclo-style), function length, file size. Skip coupling and duplication for MVP if they slow you down.
- [x] C3: Architectural Navigability -- directory depth, import graph analysis, circular dependency detection. Dead code detection can be v1.x.
- [x] C6: Testing Infrastructure -- test file detection, test-to-code ratio, coverage via `go test -cover`. Assertion density can be v1.x.
- [x] Composite score (1-10) with tier rating label
- [x] Top 5 improvement recommendations (even if simple: "reduce complexity in X")
- [x] Terminal text output with color
- [x] Exit codes (0/1/2) with `--threshold` flag
- [x] `--help` and `--version`

### Add After Validation (v1.x)

Features to add once core scoring is working and validated on real repos.

- [ ] JSON output (`--format json`) -- enables CI dashboards and custom tooling
- [ ] Verbose mode (`-v`) showing per-metric scores and methodology
- [ ] Coupling analysis (afferent/efferent coupling per package)
- [ ] Code duplication detection (token-based or AST-based)
- [ ] Dead code detection (unreachable exported functions)
- [ ] Assertion density in tests
- [ ] Test isolation scoring
- [ ] `--quiet` mode (only score and tier, for scripting)
- [ ] Config file (`.ars.yml`) for custom thresholds and weights

### Future Consideration (v2+)

Features to defer until product-market fit is established.

- [ ] HTML report generation -- presentation layer, defer until analysis is solid
- [ ] JSON/SARIF output for IDE integration
- [ ] Multi-language support (Python, TypeScript) -- requires new parsers per language
- [ ] C2 (Semantic Explicitness), C4 (Documentation), C5 (Temporal Dynamics) categories
- [ ] C7 (LLM Judge) -- high cost, non-deterministic, needs careful design
- [ ] Score trend tracking and historical comparison
- [ ] GitHub Action for automated PR checks
- [ ] Package-level score breakdown (score per Go package)
- [ ] Incremental scanning / caching for large repos
- [ ] Baseline / diff mode (only show changes since last run)

## Feature Prioritization Matrix

| Feature | User Value | Implementation Cost | Priority |
|---------|------------|---------------------|----------|
| Go project auto-detection | HIGH | LOW | P1 |
| Cyclomatic complexity (C1) | HIGH | LOW | P1 |
| Function length / file size (C1) | HIGH | LOW | P1 |
| Import graph / circular deps (C3) | HIGH | MEDIUM | P1 |
| Directory depth (C3) | MEDIUM | LOW | P1 |
| Test detection and ratio (C6) | HIGH | LOW | P1 |
| Coverage via `go test` (C6) | HIGH | MEDIUM | P1 |
| Composite score + tier | HIGH | LOW | P1 |
| Top 5 recommendations | HIGH | MEDIUM | P1 |
| Terminal text output | HIGH | LOW | P1 |
| Exit codes + threshold | HIGH | LOW | P1 |
| JSON output | MEDIUM | LOW | P2 |
| Verbose mode | MEDIUM | LOW | P2 |
| Coupling analysis (C1) | MEDIUM | MEDIUM | P2 |
| Duplication detection (C1) | MEDIUM | HIGH | P2 |
| Dead code detection (C3) | MEDIUM | MEDIUM | P2 |
| Config file | MEDIUM | MEDIUM | P2 |
| HTML reports | LOW | HIGH | P3 |
| Multi-language | LOW | HIGH | P3 |
| LLM Judge (C7) | LOW | HIGH | P3 |
| Score trend tracking | MEDIUM | HIGH | P3 |

**Priority key:**
- P1: Must have for launch -- core analysis, scoring, and basic output
- P2: Should have, add in v1.x -- enhanced analysis and output formats
- P3: Nice to have, future consideration -- presentation and expansion

## Competitor Feature Analysis

| Feature | golangci-lint | goreportcard | SonarQube (Go) | staticcheck | ARS (Our Approach) |
|---------|---------------|--------------|-----------------|-------------|-------------------|
| Cyclomatic complexity | Via gocyclo linter | Via gocyclo | Built-in | No | Core C1 metric |
| Cognitive complexity | Via gocognit linter | No | Built-in | No | Consider for v1.x (better than cyclomatic for agent readiness) |
| Circular deps | No | No | No (Java only) | No | Core C3 metric -- key differentiator |
| Dead code | Via unused linter | No | Built-in | Via U1000 check | C3 metric |
| Test coverage | No (linter only) | No | Built-in | No | Core C6 metric via `go test` |
| Composite score | No (lint/no-lint) | Letter grade (A-F) | Multi-dimensional | No | 1-10 score with research-backed weights |
| Agent-readiness framing | N/A | N/A | N/A | N/A | Core differentiator -- unique in market |
| Recommendations | No (just findings) | No | Yes (generic) | No | Top 5 ranked by agent impact |
| CI exit codes | Yes (on findings) | N/A (web only) | Yes (quality gate) | Yes (on findings) | Yes (threshold-based) |
| Output formats | 10+ formats | Web/badge | Web dashboard | Text/JSON | Terminal text (v1), JSON (v1.x) |
| Performance | Excellent (parallel) | N/A (web service) | Slow (heavy) | Good | Target: fast single-pass |

### Key Competitive Insight

ARS does not compete with golangci-lint or staticcheck. Those tools find bugs and style violations. ARS answers a different question: "How ready is this codebase for AI agent workflows?" The competitive landscape is essentially empty for this specific question. The closest analogs are:

1. **goreportcard** -- produces a letter grade, but uses simple linter checks without architectural analysis or agent-readiness framing
2. **SonarQube** -- comprehensive quality platform, but heavyweight, expensive, and not agent-readiness-focused
3. **Code Climate** -- maintainability scoring, but SaaS-only and not Go-specialized

ARS's niche: lightweight, CLI-first, Go-focused, research-backed, agent-readiness-specific. This is defensible because the scoring model is the product, not the analysis engine.

## Sources

- [golangci-lint GitHub](https://github.com/golangci/golangci-lint) -- de facto Go linter, v2.8.0 (Jan 2026), 100+ linters [HIGH confidence]
- [golangci-lint docs: output formats](https://golangci-lint.run/docs/configuration/file/) -- SARIF, JSON, text, HTML, etc. [HIGH confidence]
- [goreportcard](https://goreportcard.com/) -- Go code quality scoring using gofmt, go vet, golint, gocyclo [HIGH confidence]
- [goreportcard GitHub](https://github.com/gojp/goreportcard) -- open source, Apache v2 [HIGH confidence]
- [staticcheck](https://staticcheck.dev/) -- deep Go static analysis, 150+ checks [HIGH confidence]
- [gocyclo](https://github.com/fzipp/gocyclo) -- cyclomatic complexity for Go [HIGH confidence]
- [gocognit](https://github.com/uudashr/gocognit) -- cognitive complexity for Go, based on SonarSource whitepaper [HIGH confidence]
- [SonarQube metrics docs](https://docs.sonarsource.com/sonarqube-server/user-guide/code-metrics/metrics-definition) -- SQALE methodology, quality gates [HIGH confidence]
- [Go `analysis` framework](https://pkg.go.dev/golang.org/x/tools/go/analysis) -- standard interface for Go analyzers [HIGH confidence]
- [JetBrains Go Ecosystem 2025](https://blog.jetbrains.com/go/2025/11/10/go-language-trends-ecosystem-2025/) -- golangci-lint is de facto standard [MEDIUM confidence]
- [Anthropic Agentic Coding Trends 2026](https://resources.anthropic.com/hubfs/2026%20Agentic%20Coding%20Trends%20Report.pdf) -- AI agents amplify existing code quality [MEDIUM confidence]
- [Code Quality in 2026](https://www.getpanto.ai/blog/code-quality) -- agent readiness depends on codebase maturity [MEDIUM confidence]
- [Salesforce Code Analyzer CI docs](https://developer.salesforce.com/docs/platform/salesforce-code-analyzer/guide/ci-cd.html) -- severity threshold / exit code pattern [MEDIUM confidence]

---
*Feature research for: CLI static analysis tool (Agent Readiness Score)*
*Researched: 2026-01-31*
