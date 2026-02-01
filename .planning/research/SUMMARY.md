# Project Research Summary: ARS v2 Expansion

**Project:** Agent-Ready Score (ARS) v2
**Domain:** Static code analysis tool — multi-language expansion for agent-readiness evaluation
**Researched:** 2026-02-01
**Confidence:** HIGH (core expansion strategy), MEDIUM (LLM integration, C7 headless)

> **Note:** This summary supersedes the 2026-01-31 v1 research. It focuses exclusively on v2 expansion (multi-language support, C2/C4/C5/C7 categories). The v1 foundation (Go-only, C1/C3/C6) remains valid and unchanged.

## Executive Summary

ARS v2 expands from Go-only analysis to multi-language (Go/Python/TypeScript) with four new analysis categories (C2, C4, C5, C7) that directly measure code's fitness for AI coding agents. The research reveals a clear dual-parser strategy: preserve Go's existing `go/packages` infrastructure for rich type information, add Tree-sitter for Python/TypeScript syntax analysis, and introduce a unified `AnalysisTarget` abstraction that keeps the pipeline language-agnostic while allowing language-specific analyzers to access deep type data where available.

The recommended approach layers features by risk and dependency: start with deterministic, free, fast additions (C2 semantic analysis via AST, C5 git forensics via native git CLI), then add controlled-cost LLM features (C4 documentation quality with sampling and caching, C7 headless agent evaluation as explicit opt-in). The existing v1 architecture is well-structured for this expansion — the pipeline pattern (Discover → Parse → Analyze → Score → Recommend → Render) remains correct. The primary architectural change is generalizing the `Parser` and `Analyzer` interfaces from `[]*parser.ParsedPackage` (Go-specific) to `[]*types.AnalysisTarget` (language-agnostic with optional language extensions).

Key risks center on cost control and performance: LLM-based C4/C7 analysis can easily cost $50+ per scan without aggressive sampling and prompt caching; git forensics with go-git suffers 10-100x slowdowns on large repos requiring a switch to native git CLI; Tree-sitter CGO bindings require explicit `Close()` calls to prevent memory leaks. All three risks are mitigated through: (1) sampling 50-100 files for LLM evaluation instead of full scans, (2) shelling out to native git with bounded history windows, and (3) using the official `tree-sitter/go-tree-sitter` bindings with disciplined defer patterns. The research is backed by verified library APIs, competitor analysis (CodeScene for C5, SonarQube for C2), and academic research (CrossCodeEval for type coverage impact, SWE-bench for documentation quality correlation).

## Key Findings

### Recommended Stack

**v2 expands the proven v1 stack with minimal new dependencies while maintaining the pure-Go philosophy where practical.** The critical decision is Tree-sitter vs language-specific parsers: Tree-sitter wins because it provides a unified query language across Python/TypeScript, is battle-tested (Neovim, GitHub, Helix), and delivers 36x faster parsing than traditional approaches. However, Tree-sitter provides only syntax analysis (CST), not type resolution — this limitation shapes the C2 metrics.

**Core technologies (new for v2):**
- **`tree-sitter/go-tree-sitter` v0.25.0** (official bindings) + Python/TypeScript grammars — Multi-language AST parsing. Chosen over community fork `smacker/go-tree-sitter` because official bindings are modular (only include needed grammars), have cleaner memory management, and are maintained by the Tree-sitter organization. Requires CGO and explicit `Close()` calls.
- **`anthropics/anthropic-sdk-go` v1.20.0** — Claude API client for C4 content quality and C7 agent evaluation. Official SDK, supports prompt caching (90% cost reduction), structured output via JSON schema. Use Haiku model for cost efficiency ($0.25/MTok vs $3/MTok for Sonnet).
- **Native `git` CLI via `os/exec`** — C5 git forensics. Chosen over `go-git` v5 after research revealed 10-100x performance gaps on file-filtered log queries (35s vs <1s on 3k commits). go-git's tree diffing approach cannot compete with git's commit-graph optimization.
- **`html/template` (stdlib) + embedded Apache ECharts JS** — HTML report generation. Zero dependencies for template engine, self-contained reports with inline SVG charts (no external CDN). Rejected `go-echarts` as over-engineered for simple radar/bar charts.
- **`gopkg.in/yaml.v3` v3.0.1** (promoted from indirect) — `.arsrc.yml` config parsing. Already in dependency tree, promotion adds zero binary size.

**What NOT to add:**
- **go-git** — 35x slower than native git for blame, "extremely slow" for log with file filtering (multiple open issues)
- **Language runtimes (Python/TS)** — Tree-sitter parses without target language installed
- **Multiple LLM providers** — Single provider (Anthropic) simplifies auth, C7 already requires Claude Code
- **`go-echarts`** — Abstraction layer over ECharts adds dependency for features we don't need

**Stack impact:**
- Binary size: ~4-5 MB increase (from 15 MB to 20 MB) due to Tree-sitter CGO grammars
- CGO requirement: v2 requires `CGO_ENABLED=1` (v1 was pure Go) — biggest tradeoff
- New external dependency: `git` binary in PATH (gracefully degrade if missing)
- API dependency: Anthropic API for C4/C7 opt-in features

### Expected Features

Research divided features into four new categories (C2, C4, C5, C7) with clear table stakes vs differentiators.

**Must have (table stakes):**
- **C2 Type coverage percentage** — What fraction of functions/parameters have explicit type annotations (Python/TS) or avoid `any`/`interface{}` (Go/TS). Production tools exist for all three languages (typecoverage PyPI, type-coverage npm). This is the single most important semantic explicitness metric per CrossCodeEval research (23% accuracy improvement with typed code).
- **C2 Magic number density** — Numeric literals without named constants. Users expect this from SonarQube/ESLint. Low complexity via AST walk.
- **C4 README presence and API doc coverage** — Basic documentation hygiene. SWE-bench research shows documentation quality correlates with agent task success.
- **C5 Code churn rate and hotspot detection** — Commits per file over time window. CodeScene's core metric. High-churn + low-health files are highest-priority refactoring targets.
- **C5 Author fragmentation** — Number of authors per file. Files with no clear owner accumulate inconsistencies.
- **HTML report generation** — Users expect visual output for presentations and archives. Terminal/JSON exist in v1.

**Should have (competitive differentiators):**
- **C2 Agent-specific type coverage framing** — No existing tool frames type coverage as "will an agent understand this?" Reframe existing metric through agent-readiness lens.
- **C4 LLM-based documentation content quality** — Beyond presence: is the documentation actually useful? No production tool does this at scale. Requires opt-in flag and aggressive cost controls.
- **C5 Agent-impact hotspot ranking** — CodeScene ranks by maintenance cost. ARS ranks by agent-readiness impact: "changed constantly AND hard for agents to understand." Combines temporal data (C5) with agent-readiness scores (C1-C3).
- **C7 Headless Claude Code evaluation** — The most novel feature: spawn Claude Code programmatically, run standardized tasks, measure success rate. No competitor offers genuine agent-in-the-loop assessment.

**Defer (v2+):**
- **C2 Cross-file type propagation** — Track whether types "survive" across module boundaries. High complexity, requires type flow analysis. Interesting but not essential.
- **C4 Architecture documentation scoring** — ARCHITECTURE.md presence/quality. Simple but low priority.
- **C5 Churn-complexity trend analysis** — Is the codebase improving over time? High complexity (requires historical snapshots), nice-to-have.
- **C7 Multi-agent debate for scoring** — MAJ-EVAL pattern improves quality but multiplies cost. Premature optimization.

### Architecture Approach

**The v1 pipeline structure remains correct; only the abstraction boundary changes.** The discovery-parse-analyze-score-recommend-render flow works for multi-language. The coupling point is `parser.ParsedPackage` (Go-specific) being baked into the `Analyzer` interface. v2 introduces `AnalysisTarget` as a language-agnostic intermediate representation with optional language-specific extensions.

**Major architectural decisions:**

1. **Dual-parser strategy (not unified)**
   - Keep `go/packages` for Go — provides type information Tree-sitter cannot (dead export detection, cross-package references)
   - Add Tree-sitter for Python/TypeScript — syntax-only analysis, extremely fast
   - Both produce `AnalysisTarget` objects; Go-specific analyzers access `.GoPackage`, Python/TS analyzers use `.TreeSitterTree`

2. **Language-agnostic `AnalysisTarget` with extensions**
   ```go
   type AnalysisTarget struct {
       Language   Language
       Path       string
       Files      []SourceFile
       Functions  []FunctionInfo
       Imports    []ImportInfo

       // Language-specific extensions (type-assert when needed)
       GoPackage      *parser.ParsedPackage  // Non-nil only for Go
       TreeSitterTree *sitter.Tree           // Non-nil for Python/TS
   }
   ```
   This "adapter pattern" allows language-agnostic analyzers (C2, C4, C5) to work uniformly while preserving Go's rich type data for Go-specific features (C1 duplication, C3 dead exports).

3. **Tiered execution model for cost control**
   - **Tier 1 (default):** C1, C2, C3, C5, C6 — fast, local, deterministic, free
   - **Tier 2 (`--enable-llm`):** C4 with LLM content quality — moderate latency (~5-15s), moderate cost (~$0.50-$2.00 per scan with sampling)
   - **Tier 3 (`--enable-c7`):** C7 headless agent evaluation — high latency (60-300s), high cost (~$1.50 per evaluation)
   Cost estimation and explicit opt-in are mandatory for Tier 2/3.

4. **Git forensics as separate pre-stage**
   - C5 runs in parallel with source parsing (separate data source)
   - Uses native `git` CLI via `os/exec` for performance
   - Gracefully degrades if `.git` missing or `git` not in PATH
   - Bounded history window (default: 12 months) to control performance

5. **HTML generation via `html/template` + embedded assets**
   - Templates stored in `internal/output/templates/`, embedded via `//go:embed`
   - Inline SVG for charts (no external JS dependencies)
   - Self-contained single-file reports (work offline)
   - Auto-escaping prevents XSS from code snippets/file paths

**Major components:**
1. **Config system** — Load `.arsrc.yml` early in CLI, pass to pipeline. Controls enabled categories, scoring weights, LLM options.
2. **Discovery** — Extended to classify `.go`, `.py`, `.ts`, `.tsx`, `.js` files by language. Returns per-language file lists.
3. **Multi-parser stage** — Go and Tree-sitter parsers run in parallel, produce `[]*AnalysisTarget`. Git parser runs concurrently.
4. **Analyzer registry** — Replace hard-coded switch with registry pattern. Each category (C1-C7) registers scorer and renderer.
5. **LLM client abstraction** — Shared by C4 and C7. Handles rate limiting, prompt caching, cost tracking, graceful degradation.
6. **Output registry** — Terminal, JSON, HTML renderers. HTML uses templates + embedded CSS.

**Critical refactoring (Phase 1):**
- Define `AnalysisTarget` type
- Update `Parser` interface: `Parse(dir) ([]*AnalysisTarget, error)`
- Update `Analyzer` interface: `Analyze([]*AnalysisTarget) (*AnalysisResult, error)`
- Adapt existing C1/C3/C6 analyzers to extract `GoPackage` from targets
Once this is complete, new parsers and analyzers plug in without further interface changes.

### Critical Pitfalls

Research identified 7 critical pitfalls and 30+ moderate/minor pitfalls. Top 5 by impact:

1. **Coupling Tree-sitter to Go's ParsedPackage interface**
   **Risk:** Shoehorning Tree-sitter CST nodes into Go-centric struct creates semantic mismatch. Parallel pipelines per language duplicate code.
   **Avoidance:** Define language-agnostic `AnalysisTarget` interface from day one (Phase 1). Go-specific analyzers type-assert to `ParsedPackage`. Do NOT make Tree-sitter output look like `go/packages`.
   **Phase:** Phase 1 (Multi-language Foundation). Recovery cost: HIGH if wrong.

2. **Assuming Tree-sitter provides type information (it does not)**
   **Risk:** C2 type analysis for Python/TS measures annotation *coverage* (syntax), not type *correctness* (semantics). Teams expect go/packages-level depth and are disappointed.
   **Avoidance:** Design C2 metrics to accommodate language-specific capabilities. For Python/TS: "what percentage of functions have type annotations?" not "are the types correct?" Document this clearly. Do NOT shell out to pyright/tsc (massive latency/complexity).
   **Phase:** Phase 2 (C2 Implementation). Recovery cost: MEDIUM (metric redefinition damages trust).

3. **LLM cost blowup on C4 content quality**
   **Risk:** Naive approach sends every file to API. 10k-file repo = $18-$50+ per scan. CI runs become unaffordable.
   **Avoidance:** (a) Sample 50-100 representative files, not all files. (b) Use prompt caching (90% cost reduction on repeated scans). (c) Cache results by file content hash. (d) Use Haiku model, not Sonnet. (e) Set hard cost caps per scan. (f) Batch API for CI (50% discount).
   **Phase:** Phase 3 (C4 Implementation). Recovery cost: LOW-MEDIUM (retrofittable but trust damage).

4. **go-git performance collapse on large repos for C5**
   **Risk:** File-filtered log queries take 30s in go-git vs <1s in native git. On Kubernetes (100k commits), go-git must be aborted. Memory usage 2-8x higher.
   **Avoidance:** Shell out to native `git` for performance-critical C5 operations. Use `git log --format=...` with structured output. Handle missing git gracefully (C5 unavailable, not fatal). Bound history window (default: 12 months).
   **Phase:** Phase 2 (C5 Implementation). Recovery cost: HIGH (rewrite git interaction layer).

5. **LLM non-determinism breaking score reproducibility**
   **Risk:** Same repo, different scores across runs. Users expect deterministic results (v1 guarantee).
   **Avoidance:** (a) Cache LLM results keyed by file content hash. (b) Use structured output with JSON schema. (c) Pin model versions (not `latest`). (d) Mark C4/C7 scores as "LLM-evaluated" in output. (e) Average multiple evaluations during calibration.
   **Phase:** Phase 3 (C4 Implementation). Recovery cost: LOW (add caching) but trust damage if discovered late.

**Additional high-impact pitfalls:**
- **XSS in HTML reports** — Never use `template.HTML` with code-derived content. Syntax highlight client-side, not server-side.
- **C7 agent unreliability** — Headless Claude Code is non-deterministic, expensive, and can hang. Make opt-in only, set 60s timeouts, define narrow tasks, run multiple trials.
- **Tree-sitter CGO memory leaks** — Official bindings require explicit `Close()` on Parser/Tree/Query objects. Use `defer` immediately after creation.

## Implications for Roadmap

Based on research, v2 should be delivered in 4 phases with clear dependency ordering. The critical path is: generalize interfaces (Phase 1) → add deterministic categories (Phase 2) → add LLM tiers (Phase 3/4). Parallel work is possible after Phase 1 completes.

### Phase 1: Multi-Language Foundation + C2 Explicitness

**Rationale:** Interface generalization is prerequisite for everything else. C2 is pure static analysis (no LLM, no git) and can validate the new architecture immediately. Delivering C2 in Phase 1 proves multi-language analysis works before committing to more complex features.

**Delivers:**
- Language-agnostic `AnalysisTarget` abstraction
- Tree-sitter parser for Python/TypeScript
- Extended discovery for `.py`, `.ts`, `.tsx` files
- C2 analyzer (type coverage, magic numbers, naming quality, null safety)
- Config system (`.arsrc.yml` loader)
- Scoring expansion for C2 (7 categories total)

**Addresses features:**
- C2 type coverage (table stakes)
- C2 magic number detection (table stakes)
- C2 naming quality (table stakes)
- C2 agent-specific framing (differentiator)

**Avoids pitfalls:**
- Pitfall #1: Couples Tree-sitter correctly via AnalysisTarget from day one
- Pitfall #2: Designs C2 metrics with language-specific capabilities in mind
- Pitfall #8 (minor): Tree-sitter CGO memory leaks via disciplined Close() pattern

**Parallel work:** Config foundation can run concurrently with parser work.

### Phase 2: C5 Temporal Dynamics + Git Integration

**Rationale:** C5 is independent of multi-language parsing (git-based, not AST-based). Can run in parallel with Phase 1 completion. Delivers high-value feature (hotspot detection) quickly. Proves the git integration strategy before LLM complexity.

**Delivers:**
- Git analyzer using native `git` CLI
- C5 analyzer (churn, hotspots, author fragmentation, temporal coupling)
- History window configuration (default: 12 months)
- Graceful degradation for shallow clones / missing git
- Scoring expansion for C5

**Addresses features:**
- C5 churn rate (table stakes)
- C5 hotspot detection (table stakes)
- C5 author fragmentation (table stakes)
- C5 agent-impact hotspot ranking (differentiator — combines C5 churn with C1 health)

**Avoids pitfalls:**
- Pitfall #4: Uses native git CLI, not go-git (10-100x performance difference)
- Pitfall #5 (moderate): Handles shallow clones, missing git gracefully
- Pitfall #10 (moderate): Subprocess injection via proper `exec.Command` usage

**Uses stack:**
- Native `git` via `os/exec`
- Structured output parsing (`git log --format`)

### Phase 3: C4 Documentation Quality + LLM Integration

**Rationale:** C4 introduces LLM dependency, which is shared with C7. Build the LLM infrastructure once for both. C4 is lower-risk than C7 (simpler prompts, faster, cheaper). Proves cost control and caching strategies before C7.

**Delivers:**
- LLM client abstraction (Anthropic SDK wrapper)
- C4 analyzer with two tiers:
  - Tier 1 (always): README presence, API doc coverage, comment density
  - Tier 2 (opt-in): LLM-based content quality evaluation
- Prompt caching infrastructure (90% cost reduction)
- Content-hash-based result caching
- Cost estimation and budget caps
- Scoring expansion for C4
- HTML report generation (uses C4 results for documentation sections)

**Addresses features:**
- C4 README presence (table stakes)
- C4 API doc coverage (table stakes)
- C4 comment-to-code ratio (table stakes)
- C4 LLM content quality (differentiator — unique to ARS)

**Avoids pitfalls:**
- Pitfall #3: LLM cost blowup via sampling (50-100 files), caching, Haiku model, cost caps
- Pitfall #5: Non-determinism via content-hash caching, pinned models, structured output
- Pitfall #7: XSS in HTML via `html/template` auto-escaping, no `template.HTML` with user data

**Uses stack:**
- `anthropics/anthropic-sdk-go` v1.20.0
- `html/template` + embedded templates
- Apache ECharts (inline SVG)

**Research flag:** Standard LLM integration patterns. May need deeper research if Claude API changes behavior.

### Phase 4: C7 Agent Evaluation (Headless Claude Code)

**Rationale:** C7 is highest-risk, most expensive, most novel. Should be last because it benefits from C1/C5 scores for intelligent sampling, shares LLM infrastructure with C4, and is the most impressive feature (save for when foundation is solid).

**Delivers:**
- C7 analyzer using headless Claude Code (`claude -p`)
- Standardized evaluation tasks (based on codebase structure)
- Multi-trial execution (3 runs, median score)
- Cost estimation for agent runs
- Timeout handling (60s per task)
- Scoring expansion for C7

**Addresses features:**
- C7 headless agent evaluation (differentiator — entirely novel, no competitor)
- C7 intent clarity, modification confidence, coherence scores

**Avoids pitfalls:**
- Pitfall #6: C7 unreliability via narrow tasks, multiple trials, aggressive timeouts, circuit breaker
- Pitfall #3 (shared): LLM cost control via sampling, caching, cost warnings

**Uses stack:**
- `claude` CLI binary (external dependency)
- Anthropic SDK for cost estimation
- Shared LLM client from C4

**Research flag:** NEEDS RESEARCH. Headless Claude Code interface (`claude -p`) is documented but specifics of output parsing, task definition, and success criteria measurement need validation. Fast-moving area (Claude Code updated frequently).

### Phase Ordering Rationale

**Why this order:**
1. **Phase 1 first** — Interface generalization is prerequisite for all other work. C2 validates the architecture early.
2. **Phase 2 parallel/early** — C5 is independent of multi-language parsing. Can overlap with Phase 1 completion. High value (hotspots).
3. **Phase 3 before Phase 4** — C4 proves LLM integration strategy (caching, cost control) before more complex C7.
4. **Phase 4 last** — C7 is riskiest, needs intelligent sampling from C1/C5, benefits from C4 infrastructure.

**Grouping logic:**
- Phase 1: Core abstraction + first multi-language analyzer (proves it works)
- Phase 2: Git-based analysis (different data source, proves modular pipeline)
- Phase 3: LLM tier 1 (controlled complexity, proves cost control)
- Phase 4: LLM tier 2 (high complexity, high reward, builds on proven patterns)

**Dependency chains:**
- C2 depends on: multi-language parsing (Phase 1)
- C5 depends on: git integration (Phase 2)
- C4 depends on: LLM client (Phase 3)
- C7 depends on: C4 LLM infrastructure (Phase 4), C1/C5 scores for sampling
- HTML depends on: all categories to render (Phase 3, after C1-C5 exist)

**Parallel opportunities:**
- Config system (Phase 1) runs concurrently with parser work
- C5 (Phase 2) can start as soon as AnalysisTarget interface stabilizes (late Phase 1)
- HTML templates (Phase 3) can be prototyped before C4 completes

### Research Flags

**Phases needing deeper research during planning:**
- **Phase 4 (C7):** Headless Claude Code interface details need validation. The `claude -p` flag and `--output-format json` are documented but task definition, success measurement, and output schema need hands-on experimentation. Fast-moving area (Claude Code updated monthly). Allocate research time before implementation.

**Phases with standard patterns (skip research-phase):**
- **Phase 1 (Multi-language):** Tree-sitter integration is well-documented. Hundreds of projects use `go-tree-sitter`. Query patterns for Python/TS function extraction are standard.
- **Phase 2 (C5):** Git forensics patterns are established (CodeScene, Code Maat). Native git CLI parsing is straightforward.
- **Phase 3 (C4):** LLM integration patterns are mature. Claude API docs are comprehensive. Prompt caching is well-documented.

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | All recommended libraries verified via official docs and pkg.go.dev. Tree-sitter performance claims backed by Symflower benchmarks. go-git rejection backed by multiple GitHub issues documenting performance problems. Native git approach validated by Gitea switching for performance. |
| Features | HIGH (C2/C5), MEDIUM (C4/C7) | C2 metrics map to established tools (typecoverage PyPI, type-coverage npm). C5 metrics match CodeScene methodology (verified against published research). C4 structural metrics are standard; LLM content quality is novel but feasible. C7 is emerging paradigm — headless agent evaluation is unproven at scale. |
| Architecture | HIGH | Dual-parser strategy validated by SonarQube/Semgrep multi-language architecture (adapter pattern with language extensions). v1 codebase review confirms pipeline refactoring is straightforward. AnalysisTarget abstraction tested conceptually. |
| Pitfalls | HIGH (parsing/git/HTML), MEDIUM (LLM) | Tree-sitter CGO pitfalls documented in official bindings. go-git performance issues verified via multiple long-standing GitHub issues. XSS prevention patterns standard for Go templates. LLM cost/non-determinism pitfalls are general LLM concerns but ARS-specific impacts are inferred (MEDIUM confidence). C7 agent unreliability is based on general agent behavior, not ARS-specific testing (MEDIUM confidence). |

**Overall confidence:** HIGH for expansion feasibility, MEDIUM for specific LLM/agent features.

**Confidence by phase:**
- Phase 1 (Multi-language + C2): HIGH — proven patterns, verified libraries
- Phase 2 (C5 git forensics): HIGH — CodeScene methodology, native git validated
- Phase 3 (C4 + HTML): MEDIUM-HIGH — structural metrics HIGH, LLM quality MEDIUM (novel but feasible)
- Phase 4 (C7 agent eval): MEDIUM — headless Claude Code interface documented but unproven, agent evaluation paradigm emerging

### Gaps to Address

**Gaps requiring validation during implementation:**

1. **Tree-sitter query patterns for type annotations**
   - **Gap:** Research provides query examples for Python/TS type detection, but edge cases (nested types, generics, union types) need hands-on testing.
   - **Mitigation:** Allocate time in Phase 1 to build comprehensive test suite with real-world Python/TS repos. Test against popular projects (requests, pandas, react, vue) to validate query accuracy.

2. **LLM prompt engineering for C4 content quality**
   - **Gap:** Research suggests structured rubrics with chain-of-thought improve reliability, but specific prompt wording for documentation evaluation is untested.
   - **Mitigation:** Phase 3 should include prompt calibration phase. Evaluate 50-100 doc samples manually, compare with LLM scores, iterate on prompt until correlation is high.

3. **Headless Claude Code task definition**
   - **Gap:** What constitutes a "standardized task" that measures agent-readiness? Research suggests narrow, deterministic tasks but specifics are undefined.
   - **Mitigation:** Phase 4 needs upfront research phase. Experiment with 5-10 candidate tasks on diverse repos. Measure variance, success criteria measurability, and cost. Select 3-5 proven tasks for production.

4. **Multi-language scoring normalization**
   - **Gap:** Should a Go repo and a Python repo of identical quality produce similar composite scores? If Go C2 measures `any` usage and Python C2 measures annotation coverage, are these comparable?
   - **Mitigation:** Phase 1 must define normalization strategy. Options: (a) per-language score ranges, (b) cross-language calibration against known-good repos, (c) separate composite scores per language. Requires user research / design decision.

5. **HTML report performance at scale**
   - **Gap:** Research notes 10k-file repos could produce 5-10 MB HTML. At what repo size does the report become unusable?
   - **Mitigation:** Phase 3 includes performance testing with synthetic large repos (10k, 50k, 100k files). If >5 MB, implement summary-only mode or pagination.

6. **C5 temporal coupling performance**
   - **Gap:** Research notes temporal coupling is O(n²) file-pair analysis. What are practical thresholds for repo size before this becomes prohibitive?
   - **Mitigation:** Phase 2 includes performance testing on large repos (Kubernetes, Linux kernel). If >30s, reduce scope (e.g., analyze only top 1000 files by churn, or skip temporal coupling for repos >10k commits).

## Sources

### Primary (HIGH confidence)

**Stack:**
- [tree-sitter/go-tree-sitter on pkg.go.dev](https://pkg.go.dev/github.com/tree-sitter/go-tree-sitter) — Official Go bindings, memory management requirements
- [tree-sitter/go-tree-sitter on GitHub](https://github.com/tree-sitter/go-tree-sitter) — API docs, Close() requirements
- [anthropics/anthropic-sdk-go on GitHub](https://github.com/anthropics/anthropic-sdk-go) — v1.20.0, official SDK
- [Claude prompt caching docs](https://platform.claude.com/docs/en/build-with-claude/prompt-caching) — 90% cost reduction, cache TTL
- [Claude Code headless docs](https://code.claude.com/docs/en/headless) — `-p` flag, `--output-format json`, `--allowedTools`
- [go-git blame performance issue #14](https://github.com/go-git/go-git/issues/14) — 35x slower than CLI
- [go-git log filtering issue #137](https://github.com/go-git/go-git/issues/137) — filename filtering very slow
- [gopkg.in/yaml.v3 on pkg.go.dev](https://pkg.go.dev/gopkg.in/yaml.v3) — v3.0.1, standard YAML library

**Features:**
- [CrossCodeEval - NeurIPS 2023](https://crosscodeeval.github.io/) — Cross-file code completion, type context importance (23% improvement)
- [Meta Python Typing Survey 2025](https://engineering.fb.com/2025/12/22/developer-tools/python-typing-survey-2025-code-quality-flexibility-typing-adoption/) — 73% adoption, 41% CI enforcement
- [CodeScene Hotspots docs](https://codescene.io/docs/guides/technical/hotspots.html) — Hotspot methodology
- [SWE-bench Pro](https://scale.com/leaderboard/swe_bench_pro_public) — Documentation quality impact on agent success

**Architecture:**
- [Symflower Tree-sitter benchmarks](https://symflower.com/en/company/blog/2023/parsing-code-with-tree-sitter/) — 36x parsing speedup
- [html/template package docs](https://pkg.go.dev/html/template) — Auto-escaping, XSS prevention

**Pitfalls:**
- [Tree Sitter and the Complications of Parsing Languages - Mastering Emacs](https://www.masteringemacs.org/article/tree-sitter-complications-of-parsing-languages) — Cross-language grammar differences
- [Trail of Bits: Go parser security footguns](https://blog.trailofbits.com/2025/06/17/unexpected-security-footguns-in-gos-parsers/) — JSON case-insensitive matching, YAML risks
- [Kubernetes CVE-2019-11253](https://github.com/kubernetes/kubernetes/issues/83253) — YAML billion laughs DoS

### Secondary (MEDIUM confidence)

**Features:**
- [LLM-as-a-Judge Guide (Langfuse)](https://langfuse.com/docs/evaluation/evaluation-methods/llm-as-a-judge) — Framework for LLM evaluation
- [Agent-as-a-Judge Survey](https://arxiv.org/html/2508.02994v1) — Evolution of LLM judge paradigms
- [PRDBench](https://arxiv.org/html/2510.24358v1) — Agent-driven code evaluation benchmark

**Architecture:**
- [GoReporter HTML reports](https://github.com/360EntSecGroup-Skylar/goreporter) — Reference for html/template-based reports
- [Claude Code timeout issues](https://github.com/anthropics/claude-code/issues/5615) — 2-minute default timeout

### Tertiary (LOW confidence)

**Features:**
- [Multi-Agent-as-Judge](https://arxiv.org/abs/2507.21028) — Multi-dimensional LLM evaluation (emerging research, not production-ready)

**Pitfalls:**
- [Claude Code hanging during complex tasks](https://github.com/anthropics/claude-code/issues/4744) — Agent zombie processes (user reports, not official acknowledgment)

---
*Research completed: 2026-02-01*
*Ready for roadmap: yes*
