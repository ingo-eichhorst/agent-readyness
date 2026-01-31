# Properties That Make Codebases AI Agent-Friendly: A Research Synthesis

**Human-friendly code is AI-friendly code.** This striking finding from Borg et al.'s January 2026 study (arxiv 2601.02200) anchors a growing body of research on what makes codebases conducive to AI coding agents. Analysis of peer-reviewed papers from ICSE, FSE, NeurIPS, EMNLP, and arxiv reveals that existing software engineering quality metrics—originally calibrated for human comprehension—predict AI agent success with **92%+ accuracy** on code generation tasks. The research identifies six distinct, measurable categories of codebase properties that systematically affect whether AI agents can reliably read, modify, and extend code.

---

## 1. Code health metrics are the strongest predictors of AI success

The seed paper "Code for Machines, Not Just Humans" (arxiv 2601.02200, Borg et al., January 2026) provides the most direct empirical evidence linking code quality to AI performance. Testing GPT-4o, Claude Sonnet, and Qwen on **5,000 Python files**, researchers found a clear negative correlation between CodeHealth scores and AI refactoring failures—healthier code (scores **≥8 on a 1-10 scale**) breaks significantly less often during AI modification.

CodeHealth aggregates **25+ individual metrics** including: cyclomatic complexity, cognitive complexity, nested block depth, lines of code, LCOM4 cohesion, DRY violations, and "brain method" detection (functions with too much concentrated behavior). Separately, Sepidband et al. (arxiv 2505.23953) validated that **Halstead Effort** (a composite of code vocabulary, length, and difficulty) predicts LLM Pass@1 rates with 92.1% accuracy on HumanEval using logistic regression with Shapley values.

| Metric | Measurement Method | Evidence Source | AI Impact |
|--------|-------------------|-----------------|-----------|
| **CodeHealth** | Static analysis + git history | arxiv 2601.02200 | CH≥8 threshold for safe AI refactoring |
| **Cyclomatic Complexity** | Static (control flow paths) | arxiv 2505.23953 | Strong Pass@1 predictor |
| **Halstead Effort** | Static (operators/operands) | arxiv 2505.23953 | 94.89 vs 42.79 for fail/pass |
| **Nested Block Depth** | Static (AST analysis) | arxiv 2601.02200 | More failures with deeper nesting |
| **Lines of Code** | Static count | Multiple papers | Shapley importance confirmed |

The business case is compelling: files with alert-level CodeHealth contain **15× more defects** and require **124% more development time** (Code Red study, CodeScene). For AI adoption, this means organizations can predict where AI interventions are lower-risk using existing quality dashboards.

---

## 2. Repository structure and architecture documentation enable agent navigation

A study of **328 Claude Code configuration files** (arxiv 2511.09268, Santos et al.) reveals that architecture documentation appears in **72.6%** of successful AI agent projects—and in every single top-5 configuration pattern. The most effective Claude.md files combine architecture information with development guidelines, testing procedures, and dependency specifications.

| Configuration Content | Frequency | Importance |
|----------------------|-----------|------------|
| Software Architecture | 72.6% | Essential (in all top-5 patterns) |
| Development Guidelines | 44.8% | High |
| Project Overview | 39.0% | High |
| Testing Guidelines | 35.4% | Moderate-High |
| Commands | 33.2% | Moderate |
| Dependencies | 30.8% | Moderate |

Research on modular software design (MIT, November 2025) demonstrates that code organized into single-responsibility modules with explicit synchronization rules is "easier for tools like LLMs to generate correctly." The MASAI framework (2024) found **40% improvement** in AI-generated fixes when architectural constraints were embedded in system design. Dependency-aware planning tools like CodePlan (arxiv 2309.12499, FSE 2024) achieve 5/7 valid repository-level changes versus 0/7 for baselines without dependency graph analysis.

**Key structural properties**: explicit module boundaries, documented interfaces, dependency graphs, semantic file naming, and logical folder organization. These can be measured through AST analysis, import graph construction, and configuration file completeness checks.

---

## 3. Documentation quality critically affects comprehension, but examples matter most

API documentation research (arxiv 2503.15231) reveals a counterintuitive hierarchy: **removing code examples causes the largest performance drop** (pass rates fall from 0.66 to ~0.39), while removing parameter descriptions surprisingly yields slight improvement, and removing prose descriptions shows minimal effect. This finding has direct implications for documentation investment.

Comment quality studies (arxiv 2506.11007) show that **comment prevalence correlates positively with LLM comprehension** as measured by multiple-choice question answering accuracy. Minor comment inaccuracies have negligible effects, but major inaccuracies cause significant degradation. The Agent READMEs study (arxiv 2511.12884) identifies that **security and performance non-functional requirements are frequently missing** from AI configuration files, leading to "functional yet vulnerable code."

For private/internal libraries, the ReadMe.LLM framework (arxiv 2504.09798) demonstrates that well-established libraries (like Pandas) produce reliable LLM output, while lesser-known libraries are "often misused or misrepresented in AI-generated code"—context tailoring to the target model improves quality.

| Documentation Component | Impact on AI | Measurement |
|------------------------|--------------|-------------|
| Code examples | Critical (largest drop when removed) | Count of executable snippets |
| Architecture diagrams | High | Presence/absence + coverage |
| Comment prevalence | Moderate-High | Comments/total lines ratio |
| Comment accuracy | High (when major errors exist) | Manual verification |
| NFR specifications | Critical for security/performance | Checklist coverage |

---

## 4. Cross-file context is the key bottleneck for repository-level tasks

CrossCodeEval (arxiv 2310.11248, NeurIPS 2024) demonstrates that models perform **dramatically worse** with only in-file context—StarCoder-15.5B achieves just 8.82% exact match in Python without cross-file context, improving up to **4.5× with oracle context** retrieval. The Repository-Centric Learning paradigm (arxiv 2601.21649, SWE-Spot) proposes that small models must internalize "the physics of a target software environment" through training on repository-specific patterns rather than task-generic examples.

The SERA paper (arxiv 2601.20789) shows that repository specialization through fine-tuning is now practical—a 32B model specialized to Django matches teacher model performance with only **8,000 samples** at a cost of ~$1,300. This validates that repository-specific patterns, conventions, and domain knowledge can be encoded in model weights.

| Technique | Performance Gain | Source |
|-----------|-----------------|--------|
| Cross-file context retrieval | 3-4.5× exact match | CrossCodeEval |
| Repository-centric training | Outperforms 8× larger models | SWE-Spot |
| Iterative retrieval-generation | Matches GPT-3.5 with 350M params | RepoCoder |
| Repository specialization | Matches teacher at $1,300 | SERA |

**Measurable properties**: import graph density, API invocation patterns, class hierarchy depth, cross-file dependency ratio. These can be extracted through static analysis tools that construct dependency graphs from ASTs.

---

## 5. Task complexity thresholds reveal agent limitations

SWE-Bench Pro (arxiv 2509.16941) establishes that AI agent performance drops from **~70% on simple tasks to ~23% on enterprise-complexity tasks**—a 77% decline. The critical thresholds identified across benchmarks:

- **Lines of code modified**: Performance degrades significantly above **100 lines per patch** (SWE-Bench Pro average: 107.4 lines across 4.1 files)
- **Files touched**: Sharp degradation with **3+ files** requiring coordinated changes
- **Context length**: Accuracy drops after **~32k tokens** (LongCodeBench shows Claude 3.5 Sonnet falling from 29% to 3%)
- **Programming language**: Python/Go outperform JavaScript/TypeScript by 2-4× (SWE-PolyBench)

IDE-Bench (arxiv 2601.20886) evaluates agents using IDE-native tools (read_file, edit_file, codebase_search) across **80 tasks in 8 never-published repositories**. Terminal-Bench (arxiv 2601.11868) shows frontier models score **<65%** on realistic long-horizon tasks. These benchmarks confirm that **multi-file coordination and long-horizon planning** remain fundamental capability gaps.

**Failure mode analysis** (arxiv 2601.15195, studying 33,000 agent PRs): bug-fix and performance optimization tasks perform worst; documentation and CI updates perform best. Not-merged PRs characteristically involve larger code changes touching more files.

---

## 6. Testing infrastructure enables verification loops that improve outcomes

Research consistently shows that **comprehensive test suites enable verification loops** critical for agent success. The original SWE-Bench (arxiv 2310.06770, Jimenez et al.) uses FAIL_TO_PASS and PASS_TO_PASS test execution as the primary evaluation mechanism—agents without test feedback show dramatically lower success rates.

The MANTRA multi-agent refactoring framework (arxiv 2503.14340) achieves **82.8% success rate** by combining LLMs with traditional software engineering tools including RefactoringMiner for verification. The critical insight: "Without feedback from external tools... MANTRA encounters challenges in generating valid refactored code."

Static analysis integration studies (arxiv 2508.14419) show LLMs resolve identified issues across quality dimensions when given tool feedback—security issues reduced to **13-15%**, readability convention issues reduced from 85% to **11-33%**. However, adoption of AI coding agents correlates with **30% increase in static analysis warnings** and **41% increase in cognitive complexity** (arxiv 2511.04427)—indicating agents may create technical debt without verification enforcement.

| Testing Property | Measurement | Impact |
|-----------------|-------------|--------|
| Test coverage of patched code | Coverage % | Higher enables verification |
| CI/CD pipeline integration | Pass/fail signals | Essential feedback loop |
| Static analysis warnings | Warning count | Quality gate metric |
| RefactoringMiner validity | Compilable + test-passing | 82.8% success with tool |

---

## Synthesis: Six MECE categories ranked by evidence strength

Based on the research synthesis, the following taxonomy organizes AI-friendly codebase properties into mutually exclusive, collectively exhaustive categories, ranked by empirical evidence strength:

| Rank | Category | Key Metrics | Measurement Method | Primary Sources |
|------|----------|-------------|-------------------|-----------------|
| **1** | **Code Health/Quality** | CodeHealth score, Halstead Effort, cyclomatic complexity | Static analysis | arxiv 2601.02200, 2505.23953 |
| **2** | **Cross-File Context** | Dependency graph density, import patterns, API invocations | Static analysis + graph construction | arxiv 2310.11248, 2601.21649 |
| **3** | **Repository Structure** | Architecture documentation presence, modularity ratio, interface definitions | Config file analysis, AST | arxiv 2511.09268, 2309.12499 |
| **4** | **Documentation Quality** | Code examples count, comment prevalence, NFR coverage | Content analysis | arxiv 2503.15231, 2506.11007 |
| **5** | **Complexity Constraints** | LOC per file, nested depth, files-per-change | Static counting | arxiv 2509.16941, 2601.20886 |
| **6** | **Testing Infrastructure** | Test coverage, CI integration, static analysis gates | Coverage tools, CI logs | arxiv 2503.14340, 2508.14419 |

**The overarching finding**: existing software engineering best practices—maintainable code, clear architecture, comprehensive documentation, modular design, and strong testing—already optimize codebases for AI agents. Organizations need not adopt new frameworks; they should **invest more deeply in established quality practices** that benefit both human and machine comprehension.

---

## Measurement guidance for practitioners

For organizations seeking to assess and improve AI-friendliness:

**Statically measurable** (automated tooling):
- CodeHealth via CodeScene or similar tools
- Cyclomatic/cognitive complexity via Radon, SonarQube
- Dependency graphs via language-specific AST tools
- File size, nesting depth, code duplication

**Git history-derived**:
- Change coupling (files frequently modified together)
- Developer congestion patterns
- Hotspot analysis (frequently-changing complex code)

**Agent-as-judge evaluation**:
- Refactoring semantic preservation rate
- Issue resolution success on held-out test suites
- Cross-file completion accuracy

The research indicates **CodeHealth ≥8** serves as a practical threshold for lower-risk AI deployment, while scores below 6-7 warrant "additional human oversight" before scaling AI interventions.