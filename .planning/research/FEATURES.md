# Feature Research: v2 Analysis Categories (C2, C4, C5, C7)

**Domain:** Agent-readiness code analysis -- expansion from 3 to 7 analysis categories
**Researched:** 2026-02-01
**Confidence:** HIGH for C2/C5 (well-established tools exist), MEDIUM for C4 (metrics are contested), MEDIUM for C7 (emerging paradigm, fast-moving)

---

## Category C2: Semantic Explicitness

**What it measures:** How well the code communicates intent through types, naming, constants, and null safety -- directly predicting whether an AI agent can correctly infer semantics without guessing.

**Research basis:** CrossCodeEval (NeurIPS 2023) demonstrates that cross-file context with typed code significantly improves code completion accuracy. Meta's 2025 Python Typing Survey shows 73% of Python developers use type hints in production, but only 41% run type checkers in CI -- meaning type coverage is an actionable gap.

### Table Stakes

| Feature | Why Expected | Complexity | Language Notes |
|---------|--------------|------------|----------------|
| **Type coverage percentage** | The single most important semantic explicitness metric. Measures what fraction of identifiers/parameters/returns have explicit type annotations. Production tools exist for all three languages. | MEDIUM | **Go:** Fully typed by design -- measure `interface{}` / `any` usage instead (lower = better). **Python:** Use AST to count typed vs untyped function signatures, parameters, and return types (mirrors `typecoverage` PyPI package). **TypeScript:** Count `any` types vs total identifiers (mirrors `type-coverage` npm package). |
| **Magic number density** | Numeric literals without named constants make code opaque to agents. `go-mnd` (Go), `@typescript-eslint/no-magic-numbers`, and SonarQube all flag these. Users expect this from any "explicitness" analysis. | LOW | Exclude 0, 1, -1 by default. Count magic numbers per 1000 LOC. All three languages have established detection patterns -- AST walk for numeric literals not in const/enum declarations. |
| **Naming quality score** | Short/ambiguous identifier names (single-char variables outside loops, abbreviated names) reduce agent comprehension. This is what separates "semantic" analysis from pure type checking. | MEDIUM | **Go:** Check against Go naming conventions (no stuttering like `pkg.PkgName`, short receivers, descriptive exported names). **Python:** Check against PEP 8 naming (snake_case functions, PascalCase classes). **TypeScript:** Check against camelCase conventions. All languages: flag single-char non-loop variables, measure avg identifier length. |
| **Null/nil safety patterns** | Unchecked nil dereferences are the #1 runtime panic in Go. TypeScript's `strictNullChecks` eliminates an entire class of bugs. Agents operating on code with poor null safety produce more errors. | MEDIUM | **Go:** Ratio of pointer returns with nil checks vs without. Check for `if err != nil` patterns after fallible calls. **Python:** Ratio of `Optional[T]` annotations vs bare `None` returns. **TypeScript:** Check `strictNullChecks` in tsconfig.json, count `!` non-null assertions (lower = better). |

### Differentiators

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| **Agent-specific type coverage framing** | No existing tool measures type coverage through the lens of "will an agent understand this?" SonarQube measures maintainability; `type-coverage` measures TypeScript any-percentage. ARS frames it as: "agents are 23% more accurate on typed code" (CrossCodeEval). This reframing is the unique value. | LOW | Same underlying metric, different presentation and scoring context. |
| **Semantic density score** | Composite metric combining type coverage + naming quality + constant usage. One number that answers "how explicit is this code?" No production tool offers this composite. | LOW | Weighted combination of sub-metrics. The composite itself is the differentiator, not the individual metrics. |
| **Cross-file type propagation analysis** | Measure whether types "survive" across module boundaries. A function returning `interface{}` forces callers to do type assertions -- agent-hostile. Tracking type information loss at API boundaries is novel. | HIGH | Requires cross-package type flow analysis. Go's `go/types` package supports this. Python/TypeScript need type checker integration. Defer to post-MVP within C2 unless effort is manageable. |

### Anti-Features

| Anti-Feature | Why Avoid | What to Do Instead |
|--------------|-----------|-------------------|
| **Running mypy/pyright/tsc as a subprocess** | Adds massive external dependencies, unpredictable runtime, and version compatibility nightmares. Users may not have these tools installed. ARS should be self-contained. | Parse AST to count type annotations structurally. You do not need a full type checker -- you need to know whether annotations EXIST, not whether they are CORRECT. Correctness is the type checker's job. |
| **Style enforcement (gofmt compliance, etc.)** | ARS is not a linter. Style compliance is golangci-lint's domain. Duplicating it dilutes the agent-readiness focus. | Measure semantic properties (types, naming patterns, constants) not stylistic ones (whitespace, brace placement). |
| **Per-identifier naming suggestions** | "Rename `x` to `counter`" is noisy and subjective. Generates hundreds of findings that overwhelm users and drown the signal. | Report aggregate naming quality scores (avg identifier length, single-char variable ratio). Flag only the worst offenders in recommendations. |
| **Enforcing 100% type coverage** | Unrealistic for most codebases. Generates false urgency. Even TypeScript's `type-coverage` tool acknowledges catch blocks and callbacks make 100% impractical. | Use breakpoint scoring (0-10 scale) where 80% type coverage scores high. Diminishing returns above ~90%. |

### Complexity Notes

- **Type coverage (Go):** LOW -- Go is statically typed; the metric is `any`/`interface{}` usage count. AST walk is straightforward.
- **Type coverage (Python):** MEDIUM -- Walk AST for `def` nodes, check for type annotations on args and return. `ast` module handles this natively. No need for mypy.
- **Type coverage (TypeScript):** MEDIUM -- Requires TypeScript AST parsing (Tree-sitter). Count `any` keyword usage vs total type positions.
- **Magic numbers:** LOW across all languages -- simple AST pattern matching.
- **Naming quality:** MEDIUM -- need heuristics for "good" names which are inherently subjective. Keep it simple: length, case convention adherence, single-char ratio.
- **Null safety:** MEDIUM for Go (pattern matching nil checks), LOW for TypeScript (check tsconfig), MEDIUM for Python (check Optional annotations).

### Dependencies on v1

- Requires multi-language AST parsing infrastructure (Tree-sitter for Python/TypeScript, existing `go/ast` for Go)
- Scoring config needs new C2 category with weight allocation
- Recommendation engine needs new C2 improvement suggestions

---

## Category C4: Documentation Quality

**What it measures:** Whether the codebase has sufficient documentation for an agent to understand intent, API contracts, and architectural decisions -- from README presence to inline comment quality.

**Research basis:** SWE-bench research shows documentation quality correlates with agent task success, with well-documented repositories showing significantly higher resolution rates. SWE-bench Pro analysis confirms "codebase complexity, problem type, or documentation quality significantly impact an agent's ability to succeed."

### Table Stakes

| Feature | Why Expected | Complexity | Language Notes |
|---------|--------------|------------|----------------|
| **README presence and completeness** | A project without a README is immediately hostile to agents and humans alike. Check for existence, minimum length, and key sections (description, installation, usage). | LOW | Language-agnostic. Check for README.md (or README, README.rst). Score presence of sections: description, install/setup, usage/examples, API reference, contributing. |
| **Exported symbol documentation rate** | Percentage of public APIs (exported functions, types, classes) that have doc comments. Go enforces this culturally; Python has docstrings; TypeScript has JSDoc. | MEDIUM | **Go:** Check for comment above exported identifiers (standard `go/ast` comment mapping). **Python:** Check for docstrings on public functions/classes (first statement is string literal). **TypeScript:** Check for JSDoc comments on exported members. |
| **Comment-to-code ratio** | Basic density metric: what percentage of lines are comments? Too low means no documentation; too high can mean commented-out code (a different problem). | LOW | Language-agnostic LOC counting. Exclude blank lines. Sweet spot is roughly 10-25% for most codebases. Score both extremes low (under 5% and over 40%). |
| **API documentation presence** | For libraries/packages: do exported types have documented parameters, return values, and error conditions? Agents rely heavily on API docs for cross-file calls. | MEDIUM | **Go:** Check godoc format (param descriptions in prose). **Python:** Check docstring format (Google, NumPy, or Sphinx style) for Args/Returns/Raises sections. **TypeScript:** Check JSDoc `@param`, `@returns`, `@throws` tags. |

### Differentiators

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| **Documentation content quality (LLM-evaluated)** | Beyond presence: is the documentation actually USEFUL? An LLM judge can assess whether a docstring explains WHAT a function does vs just restating the signature. No production tool does this at scale. "31% higher agent success with good docs" is the pitch. | HIGH | **Cost implications:** Requires LLM API calls per documented symbol. At $3/MTok input, a 50k LOC repo with ~500 documented symbols costs ~$0.50-$2.00. Must be opt-in (`--deep-docs` flag or similar). Cache results aggressively. |
| **Stale documentation detection** | Doc comments that contradict the code signature (wrong param names, missing params, outdated descriptions). Stale docs are worse than no docs because they mislead agents. | MEDIUM | Compare parameter names in doc comments with actual function signature. Check for `@deprecated` without removal timeline. Detect TODO/FIXME in docs. No LLM needed -- purely structural. |
| **Architecture documentation scoring** | Presence and quality of ARCHITECTURE.md, design docs, ADRs. These high-level docs are the most valuable for agent context but rarely measured by tools. | LOW | File presence check + basic content analysis (word count, section headings). Simple but novel -- no production tool scores this. |

### Anti-Features

| Anti-Feature | Why Avoid | What to Do Instead |
|--------------|-----------|-------------------|
| **Readability scoring (Flesch-Kincaid, Gunning Fog)** | These metrics are designed for prose, not code documentation. Technical docs SHOULD use domain terminology. Penalizing "hard" words in API docs is counterproductive. A doc that says "serializes the struct to protobuf wire format" is better than "changes the thing to bytes" even though Flesch-Kincaid would prefer the latter. | Score documentation PRESENCE and STRUCTURE (does it have params? returns? examples?) not linguistic complexity. If using LLM evaluation for content quality, let the LLM judge usefulness, not grade level. |
| **Spell checking** | False positives on technical terms, library names, and domain jargon would be overwhelming. Maintaining a domain-specific dictionary is a maintenance burden. | Leave spell checking to dedicated tools (aspell, cspell). Focus on structural and semantic documentation quality. |
| **Auto-generating documentation** | "If docs are missing, just generate them" is tempting but produces generic, low-value docs that satisfy the metric without adding real value. ARS is a diagnostic tool, not a generator. | Score the current state honestly. Recommend where docs are needed. Let humans or their agents write the actual docs. |
| **Scoring inline implementation comments** | Comments explaining HOW code works are subjective and often indicate the code is too complex (the real fix is simplification). Measuring inline comment quality is a rabbit hole. | Focus on API-level documentation (what a function does, its contract) not implementation comments. The C1 complexity score already catches "code that needs explaining." |

### Complexity Notes

- **README analysis:** LOW -- file existence + regex for section headings.
- **Doc comment rate:** MEDIUM -- requires AST traversal with comment association. Go's `go/ast` has `CommentMap`; Python's `ast` has docstring detection; TypeScript needs Tree-sitter comment handling.
- **Comment ratio:** LOW -- line counting with comment detection.
- **LLM content quality:** HIGH -- requires LLM integration, prompt engineering, cost management, caching, and opt-in UX. This is the single most complex sub-feature across all four categories. Must be clearly separated as opt-in.
- **Stale doc detection:** MEDIUM -- requires matching doc content against actual code signatures.

### Dependencies on v1

- Multi-language parsing infrastructure (same as C2)
- LLM integration infrastructure needed for content quality (shared with C7)
- Scoring config extension
- New recommendation templates for documentation improvements

---

## Category C5: Temporal Dynamics

**What it measures:** How the codebase evolves over time -- code churn patterns, change coupling, author fragmentation, and hotspots. Based on CodeScene's behavioral code analysis methodology and Adam Tornhill's "Your Code as a Crime Scene."

**Research basis:** CodeScene research demonstrates a strong correlation between hotspots (high-churn + low-health code), maintenance costs, and software defects. Change frequency follows a power law -- most development activity is in a small fraction of the codebase. Code churn is the single most important metric for predicting quality issues (per CodeScene's published findings).

### Table Stakes

| Feature | Why Expected | Complexity | Language Notes |
|---------|--------------|------------|----------------|
| **Code churn rate** | Commits per file over a time window (default: 6 months). High-churn files that also score low on C1/C3 are the highest-priority refactoring targets. CodeScene's core metric. | MEDIUM | Language-agnostic (git log analysis). Parse `git log --numstat` for additions/deletions per file. Calculate relative churn (changes / file size) to normalize across file sizes. Requires `.git` directory -- fail with clear error if missing. |
| **Hotspot detection** | Files with both high churn AND low code health (from C1). This is CodeScene's signature analysis: prioritize technical debt by actual development activity, not just static quality. | MEDIUM | Combine C5 churn data with C1 scores per file. Rank by churn * (10 - health_score). Top hotspots are the highest-impact improvement targets. |
| **Author fragmentation** | Number of distinct authors per file. Files touched by many authors with no clear owner tend to accumulate inconsistencies. CodeScene calls this "diffusion of responsibility." | LOW | `git log --format='%aN' -- <file> | sort -u | wc -l` equivalent. Calculate author count per file and identify files with no primary owner (no author has >50% of commits). |
| **Temporal coupling** | Files that change together in the same commits, indicating hidden dependencies. CodeScene's temporal coupling analysis reveals architectural coupling invisible in the code itself. | HIGH | Analyze commit history for co-change patterns. For each pair of files changed in the same commit, count co-occurrences. Filter by minimum thresholds (min 5 shared commits, min 30% coupling degree). This is computationally expensive for large repos -- requires thoughtful thresholds. |

### Differentiators

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| **Agent-impact hotspot ranking** | CodeScene ranks hotspots by maintenance cost. ARS ranks by agent-readiness impact: "this file is changed constantly AND is hard for agents to understand." Combining temporal data with agent-readiness scores is novel. | LOW | Composite of existing metrics. The framing and scoring integration is the value, not new data collection. |
| **Churn-complexity trend** | Is the codebase getting better or worse over time? Track whether high-churn files are trending toward higher or lower complexity. Answers "are we making progress?" | HIGH | Requires analyzing git history at multiple time points. Computationally expensive. Consider: analyze last N commits in windows (e.g., monthly buckets) and report trend direction. Defer detailed trending to post-initial C5 delivery. |
| **Bus factor per module** | How many authors would need to leave before knowledge of a module is lost? Files with bus factor = 1 are high risk. | LOW | Count primary contributors per directory/package. Bus factor = number of authors contributing >10% of changes. Simple calculation from git log data. |

### Anti-Features

| Anti-Feature | Why Avoid | What to Do Instead |
|--------------|-----------|-------------------|
| **Full git history analysis by default** | Analyzing the entire git history of a large repo (10+ years, 100k+ commits) is prohibitively slow. CodeScene takes minutes for large repos even with optimized algorithms. | Default to 6-month window. Allow `--history-window` flag. Most recent changes are most relevant for agent-readiness anyway. |
| **Cross-repository temporal coupling** | CodeScene supports this for microservices, but it requires access to multiple repos and is an order of magnitude more complex. | Analyze single-repo only. Multi-repo analysis is a v3+ feature if ever. |
| **Individual developer attribution/scoring** | "Developer X writes the worst code" is toxic and will get ARS banned from organizations. CodeScene carefully avoids this. | Report at file/module level only. "This module has high author fragmentation" not "Author X's changes have low quality." Never rank individuals. |
| **Commit message quality analysis** | Tempting but subjective. "fix bug" vs "Fix race condition in connection pool by adding mutex" -- the latter is better, but scoring this reliably is hard and not directly agent-relevant. | Skip entirely. Commit messages affect git-blame-based understanding but are not a primary agent-readiness signal. Focus on code and doc quality. |

### Complexity Notes

- **Git integration:** MEDIUM -- `git log` parsing is well-understood but needs robust error handling (shallow clones, missing git, large repos).
- **Churn calculation:** MEDIUM -- `git log --numstat` gives additions/deletions per file per commit. Aggregate over time window.
- **Author fragmentation:** LOW -- simple aggregation from git log.
- **Temporal coupling:** HIGH -- O(n^2) file-pair analysis on commits. Need smart thresholds and capping. CodeMaat (Adam Tornhill's tool) solves this with configurable min-revisions and max-changeset-size filters.
- **Performance:** C5 is the only category that scales with repository history, not current code size. A repo with 50k LOC but 50k commits will take longer for C5 than a 500k LOC repo with 500 commits. The `--history-window` default is critical.

### Dependencies on v1

- No dependency on existing analyzers (git-based, not AST-based)
- Needs C1 file-level scores for hotspot correlation
- Requires `.git` directory -- must gracefully handle repos without git (skip C5 with warning)
- New pipeline stage: git analysis runs independently of AST parsing (can parallelize)

---

## Category C7: Agent Evaluation

**What it measures:** Direct LLM-based assessment of code's agent-friendliness -- using an AI judge to evaluate intent clarity, modification confidence, and overall coherence. The "ask an agent if it can work with this code" approach.

**Research basis:** AlpacaEval framework demonstrates LLM-as-judge achieves ~90% agreement with human preferences. PRDBench (2025) applies agent-as-a-judge specifically to code evaluation. The paradigm is evolving from single-model judges to multi-agent debate systems, but single-judge with chain-of-thought is the proven starting point.

### Table Stakes

| Feature | Why Expected | Complexity | Language Notes |
|---------|--------------|------------|----------------|
| **Intent clarity score** | Can an agent understand WHAT this code does from reading it? LLM evaluates a sample of functions and rates clarity of purpose. This is the core C7 metric -- the most direct measure of agent-readiness. | HIGH | Language-agnostic (LLM reads code as text). Sample selection matters: evaluate the most important files (entry points, public APIs, hotspots from C5) not random files. |
| **Modification confidence score** | Could an agent safely modify this code? LLM evaluates whether the code has clear boundaries, predictable side effects, and sufficient context for safe changes. | HIGH | Language-agnostic. Prompt engineering is critical: "Given this function, rate your confidence that you could modify it without breaking other parts of the system. Explain why." |
| **Coherence score** | Does the codebase follow consistent patterns? LLM evaluates whether similar operations are done the same way, whether naming is consistent, whether architecture is predictable. | HIGH | Requires sampling from multiple files. Compare patterns across the codebase. This is the most expensive metric because it needs cross-file context. |
| **Cost estimation and opt-in** | Users MUST know the cost before running C7. "This scan will make ~50 LLM calls, estimated cost: $1.50. Proceed? [y/N]" Without cost transparency, users will be surprised and angry. | MEDIUM | Calculate: (sampled files * prompts per file * avg tokens) * cost per token. Show estimate before execution. Always opt-in (`--agent-eval` flag), never default-on. |

### Differentiators

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| **Headless Claude Code evaluation** | Instead of a generic LLM call, use Claude Code (the agentic coding tool) to actually attempt a small modification task on the codebase and measure success. This is a genuine "agent-in-the-loop" test, not just an opinion poll. Per PROJECT.md: "headless agent evaluation using Claude Code for genuine agent-in-the-loop assessment." | VERY HIGH | This is the most ambitious feature in all of v2. Requires: spawning headless Claude Code, defining synthetic tasks, measuring completion, handling failures gracefully. Start with simpler LLM-as-judge, add headless evaluation as an advanced option. |
| **Structured evaluation rubric** | Rather than "rate this code 1-10," provide a detailed rubric with specific dimensions (naming, structure, documentation, error handling, testability). AlpacaEval research shows structured prompts with chain-of-thought improve reliability by 10-15%. | MEDIUM | Define rubric once, apply consistently. Chain-of-thought prompting: "First explain what this function does, then evaluate each dimension, then give a score." More tokens = more cost, but significantly more reliable. |
| **Evaluation calibration against known codebases** | Run C7 on well-known repos (standard library, popular open source) to establish baselines. "Your code scores 6/10 on intent clarity; the Go standard library scores 8.5/10." Calibration creates trust. | MEDIUM | One-time effort to generate baselines. Store as reference data. Update periodically. |

### Anti-Features

| Anti-Feature | Why Avoid | What to Do Instead |
|--------------|-----------|-------------------|
| **Default-on LLM evaluation** | API costs per scan, non-deterministic results, network dependency, latency (adds minutes to scan). Making this default-on would make ARS unusable in CI, offline environments, and cost-sensitive teams. | Always opt-in. C7 is a separate flag (`--agent-eval`). All other categories (C1-C6) remain deterministic, free, and fast. |
| **Evaluating every function** | A 50k LOC repo might have 2000+ functions. Evaluating each with an LLM is expensive ($10+) and slow (minutes). Diminishing returns after sampling the most important ones. | Smart sampling: evaluate entry points, public APIs, high-churn files (from C5), low-scoring files (from C1/C3). Target 20-50 functions for a representative score. |
| **Multi-agent debate for scoring** | The research shows multi-agent-as-judge (MAJ-EVAL) improves quality, but it multiplies cost by the number of agents. Premature optimization of evaluation quality at this stage. | Single-judge with chain-of-thought first. It achieves ~90% human agreement (AlpacaEval). Multi-agent is a future refinement when the single-judge baseline is established. |
| **Fine-tuning a custom judge model** | Training a specialized judge model would reduce per-call costs but requires training data, infrastructure, and ongoing maintenance. Massive upfront investment for marginal improvement. | Use frontier models (Claude Sonnet/Opus) as judges. The cost per scan is manageable ($1-5) and the quality is superior to any fine-tuned small model. |
| **Replacing C1-C6 metrics with LLM evaluation** | "Just ask the LLM to score everything" eliminates the need for static analysis. But LLM scores are non-deterministic, expensive, and opaque. Reproducibility is destroyed. | C1-C6 are deterministic and free. C7 adds a complementary perspective. The static metrics are the foundation; the LLM evaluation is the capstone. |

### Complexity Notes

- **LLM integration:** HIGH -- needs API client, prompt templates, response parsing, error handling, rate limiting, cost tracking.
- **Prompt engineering:** HIGH -- the quality of C7 scores depends entirely on prompt quality. Requires iteration and validation against human judgment.
- **Cost management:** MEDIUM -- token counting, cost estimation, budget limits, caching of results.
- **Sampling strategy:** MEDIUM -- selecting which files/functions to evaluate requires combining signals from C1, C3, C5.
- **Reproducibility:** C7 scores will vary between runs due to LLM non-determinism. Must clearly communicate this to users. Consider: run 3 evaluations and report median to reduce variance.
- **Headless Claude Code:** VERY HIGH -- spawning and controlling an agentic coding tool programmatically is uncharted territory for most tools. Start with LLM-as-judge API calls; headless agent evaluation is a stretch goal.

### Dependencies on v1

- Needs LLM API integration infrastructure (new dependency: Claude API client)
- Sampling strategy benefits from C1 scores (evaluate low-scoring functions) and C5 hotspots
- Scoring config needs C7 weight allocation (should be lower than deterministic categories given non-determinism)
- Must NOT block overall scan -- C7 failure should degrade gracefully (score other categories normally)
- Shared infrastructure with C4 content quality evaluation (LLM calls)

---

## Cross-Category Feature Dependencies

```
[Multi-language AST Parsing] (Tree-sitter for Python/TS, go/ast for Go)
    |
    +--required by--> [C2: Semantic Explicitness] (type annotations, naming, magic numbers)
    +--required by--> [C4: Documentation Quality] (doc comments, API docs)
    +--NOT required by--> [C5: Temporal Dynamics] (git-based, no AST needed)
    +--NOT required by--> [C7: Agent Evaluation] (LLM reads raw code text)

[Git Integration] (git log parsing)
    |
    +--required by--> [C5: Temporal Dynamics]
    +--enhances--> [C7: Agent Evaluation] (churn data informs sampling)

[LLM API Integration] (Claude API client)
    |
    +--required by--> [C7: Agent Evaluation]
    +--optional for--> [C4: Documentation Quality] (content quality is opt-in)

[C1: Code Health scores] (existing v1)
    |
    +--enhances--> [C5: Hotspot Detection] (churn * low-health = hotspot)
    +--enhances--> [C7: Sampling Strategy] (evaluate low-health code first)

[Scoring Config Extension]
    |
    +--required by--> ALL new categories (C2, C4, C5, C7 need weights and breakpoints)
```

### Dependency Notes

- **C2 and C4 share multi-language parsing:** Build the parsing infrastructure once (Tree-sitter integration for Python/TypeScript), then C2 and C4 both use it. This argues for delivering C2 and C4 in the same phase or sequential phases.
- **C5 is independent:** Git-based analysis has zero dependency on AST parsing. C5 can be built in parallel with C2/C4 or in any order.
- **C7 depends on everything else:** Optimal C7 sampling uses C1 health scores, C5 churn data, and C4 documentation gaps. Deliver C7 last.
- **LLM infrastructure is shared:** C4 content quality and C7 agent evaluation both need LLM API calls. Build the integration once for both.

---

## Implementation Priority Recommendation

### Phase 1: C2 (Semantic Explicitness) + C5 (Temporal Dynamics)

Rationale: C2 and C5 are the highest-value additions with established tooling patterns.

- C2 is pure static analysis (extends existing AST infrastructure)
- C5 is git-based analysis (independent pipeline, parallelizable)
- Together they fill the biggest gap: "what does the code mean?" (C2) and "how does it evolve?" (C5)
- Both are deterministic, free, and fast -- maintaining ARS's core value proposition

### Phase 2: C4 (Documentation Quality)

Rationale: C4 static metrics (doc presence, comment ratio) are straightforward. LLM content quality evaluation shares infrastructure with C7 and should be built in the same phase or immediately before.

- Static documentation metrics first (presence, rate, structure)
- LLM-based content quality as opt-in enhancement

### Phase 3: C7 (Agent Evaluation)

Rationale: C7 is the most complex, most expensive, and most dependent on other categories. It should be last because:

- Needs LLM infrastructure (can share with C4 if C4's LLM features come first)
- Benefits from C1/C5 scores for intelligent sampling
- Is the riskiest feature (non-deterministic, costly, novel)
- Is the most impressive feature -- save it for when the foundation is solid

---

## Scoring Weight Redistribution

Current v1 weights (total 60% -- space reserved for new categories):
- C1: 25%, C3: 20%, C6: 15%

Recommended v2 weights (total 100%):

| Category | Weight | Rationale |
|----------|--------|-----------|
| C1: Code Health | 20% | Slightly reduced; still foundational |
| C2: Semantic Explicitness | 15% | High agent-readiness impact (CrossCodeEval) |
| C3: Architecture | 15% | Slightly reduced from 20% |
| C4: Documentation | 12% | Important but partially subjective |
| C5: Temporal Dynamics | 13% | Strong predictive power (CodeScene research) |
| C6: Testing | 12% | Slightly reduced from 15% |
| C7: Agent Evaluation | 13% | Direct measurement but non-deterministic |

Note: When C7 is not run (opt-out), redistribute its weight proportionally across C1-C6. The scoring engine already handles unavailable metrics -- extend this to handle unavailable categories.

---

## Competitor Feature Analysis (v2 Categories)

| Feature | SonarQube | CodeScene | CodeClimate | ARS (Our Approach) |
|---------|-----------|-----------|-------------|-------------------|
| Type coverage | No (test coverage only) | No | No | Core C2 metric -- framed as agent-readiness |
| Magic numbers | Yes (rule-based) | No | No | C2 metric, scored not just flagged |
| Naming quality | Partial (conventions) | No | No | C2 metric with language-specific heuristics |
| Doc comment rate | Yes (rule-based) | No | No | C4 metric with quality assessment |
| Doc content quality | No | No | No | C4 differentiator via LLM evaluation |
| Code churn | No | Core feature | No | C5 metric, integrated with health scores |
| Temporal coupling | No | Core feature | No | C5 metric (simplified vs CodeScene) |
| Author fragmentation | No | Yes | No | C5 metric |
| Hotspot detection | No | Core feature | No | C5 metric combined with C1 health |
| LLM code evaluation | No | No | No | C7 -- entirely novel |
| Agent-readiness framing | No | No | No | Core differentiator across all categories |

### Key Competitive Insight for v2

With v2, ARS becomes the only tool that combines:
1. Static code quality analysis (like SonarQube)
2. Behavioral/temporal analysis (like CodeScene)
3. LLM-based evaluation (novel -- no competitor)
4. All framed through agent-readiness scoring (unique perspective)

No existing tool offers this combination. SonarQube is closest on static analysis but lacks temporal and LLM dimensions. CodeScene is closest on temporal analysis but lacks type coverage and LLM evaluation. Neither frames their analysis through agent-readiness.

---

## Sources

### C2: Semantic Explicitness
- [CrossCodeEval - NeurIPS 2023](https://crosscodeeval.github.io/) -- cross-file code completion benchmark demonstrating type context importance [HIGH confidence]
- [Meta Python Typing Survey 2025](https://engineering.fb.com/2025/12/22/developer-tools/python-typing-survey-2025-code-quality-flexibility-typing-adoption/) -- 73% adoption, 41% CI enforcement [HIGH confidence]
- [typecoverage PyPI](https://pypi.org/project/typecoverage/) -- Python type annotation coverage tool [HIGH confidence]
- [type-coverage npm](https://github.com/plantain-00/type-coverage) -- TypeScript type coverage, v2.29.7 [HIGH confidence]
- [go-mnd](https://github.com/tommy-muehle/go-mnd) -- Go magic number detector, integrated in golangci-lint [HIGH confidence]
- [@typescript-eslint/no-magic-numbers](https://typescript-eslint.io/rules/no-magic-numbers/) -- TypeScript magic number linting [HIGH confidence]
- [Go nillability proposal](https://github.com/golang/go/issues/49202) -- Go's nil safety gap [MEDIUM confidence]

### C4: Documentation Quality
- [SWE-bench Pro](https://scale.com/leaderboard/swe_bench_pro_public) -- documentation quality as factor in agent success [MEDIUM confidence]
- [Penify.dev README Analysis](https://blogs.penify.dev/docs/analyze-readme-readability.html) -- readability metrics for README files [MEDIUM confidence]
- [SonarQube metrics](https://docs.sonarsource.com/sonarqube-server/user-guide/code-metrics/metrics-definition) -- comment density and documentation rules [HIGH confidence]

### C5: Temporal Dynamics
- [CodeScene Hotspots](https://codescene.io/docs/guides/technical/hotspots.html) -- hotspot methodology and research backing [HIGH confidence]
- [CodeScene Hotspot Metrics](https://docs.enterprise.codescene.io/versions/1.7.0/configuration/hotspot-metrics.html) -- churn calculation methods [HIGH confidence]
- [Code Maat](https://github.com/adamtornhill/code-maat) -- open-source temporal coupling analysis tool [HIGH confidence]
- [CodeScene Architectural Analysis](https://docs.enterprise.codescene.io/versions/3.5.6/guides/architectural/architectural-analyses.html) -- temporal coupling at architecture level [HIGH confidence]
- [Swimm Code Churn Guide](https://swimm.io/learn/developer-experience/how-to-measure-code-churn-why-it-matters-and-4-ways-to-reduce-it) -- churn measurement methodology [MEDIUM confidence]

### C7: Agent Evaluation
- [LLM-as-a-Judge Guide (Langfuse)](https://langfuse.com/docs/evaluation/evaluation-methods/llm-as-a-judge) -- framework for LLM evaluation [MEDIUM confidence]
- [Agent-as-a-Judge Survey](https://arxiv.org/html/2508.02994v1) -- evolution of LLM judge paradigms [MEDIUM confidence]
- [LLM-as-Judge Best Practices (Monte Carlo)](https://www.montecarlodata.com/blog-llm-as-judge/) -- bias mitigation, chain-of-thought [MEDIUM confidence]
- [PRDBench](https://arxiv.org/html/2510.24358v1) -- agent-driven code evaluation benchmark [MEDIUM confidence]
- [Multi-Agent-as-Judge](https://arxiv.org/abs/2507.21028) -- multi-dimensional LLM evaluation [LOW confidence -- emerging research]
- [LLM-as-a-Judge 2026 Guide](https://labelyourdata.com/articles/llm-as-a-judge) -- comprehensive overview of patterns [MEDIUM confidence]

---
*Feature research for: ARS v2 Analysis Categories (C2, C4, C5, C7)*
*Researched: 2026-02-01*
