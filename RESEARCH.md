# Research Evidence: Why Code Quality Matters for AI Agents

This document contains the detailed research evidence supporting the Agent Readiness Score (ARS). All claims are backed by peer-reviewed studies and empirical measurements.

---

## 1. Code Quality Predicts Agent Success—Dramatically

In a controlled study of 5,000 Python files ([Borg et al., 2026](https://arxiv.org/abs/2601.02200)), researchers measured how code quality (CodeHealth ≥9 threshold) affected LLM break rates on refactoring tasks:

| Model | Healthy Code Break Rate | Unhealthy Code Break Rate | Difference |
|-------|-------------------------|---------------------------|------------|
| **Claude Sonnet 3.5** | 3.8% | 5.2% | +1.4pp |
| **Qwen 2.5 Coder** | 20.7% | 27.8% | **+7.1pp** |
| **GPT-4o** | 35.9% | 47.0% | **+11.1pp** |
| **GLM-4-Plus** | 39.9% | 50.0% | **+10.1pp** |
| **Gemma 2** | 44.3% | 59.4% | **+15.1pp** |

**Key insight:** Even the best model (Claude) degrades with unhealthy code. Weaker or cheaper models pay a **massive "AI Tax"**—break rates increase by 7-15 percentage points. That's the difference between a viable agent and one that constantly requires human intervention.

---

## 2. Complexity Kills Agent Performance

Agent performance doesn't degrade linearly—it **collapses** on complex tasks:

- **SWE-bench Pro** ([arXiv:2509.16941](https://arxiv.org/abs/2509.16941)): Performance drops from **~70% on simple tasks to ~23% on enterprise-complexity tasks**—a 77% decline
- Degradation thresholds:
  - Patches >100 lines
  - Changes spanning 3+ files
  - Context windows >32k tokens

**Why it matters:** If your codebase has 500-line God Classes, deeply nested control flow, or scattered cross-file dependencies, agents will struggle with even moderate changes.

---

## 3. What Makes Code "Complex" to an Agent?

Halstead Effort—a measure of code cognitive load based on operators and operands—predicts LLM success with **92.1% accuracy** ([Sepidband et al., 2025](https://arxiv.org/abs/2505.23953)):

- **Failed code:** Average Halstead Effort = 94.89
- **Passing code:** Average Halstead Effort = 42.79

Translation: Code that's hard for humans to reason about is **twice as hard** for agents. And unlike humans, agents can't "power through" with caffeine and Stack Overflow.

---

## 4. Cross-File Context Is Make-or-Break

Agents can't intuit implicit dependencies. Without explicit context:

- **CrossCodeEval** ([arXiv:2310.11248](https://arxiv.org/abs/2310.11248)): StarCoder-15.5B achieves **only 8.82% exact match** without cross-file context
- With oracle retrieval: **up to 4.5x improvement**

**The problem:** God Classes, hidden coupling, and missing architecture docs create "invisible" dependencies. Agents retrieve the wrong code, flood their context window with noise, and produce incorrect solutions.

**The fix:** Explicit dependency graphs, modular architecture, and clear entry points enable agents to navigate your codebase like a map instead of a maze.

---

## 5. Architecture Documentation Is a Navigation Signal

When researchers analyzed 60+ AI agent configurations on real-world tasks ([Santos et al., 2025](https://arxiv.org/abs/2511.09268)):

- **72.6%** of successful configurations included architecture documentation
- Present in **every single top-5 configuration pattern**

And when repo-level code graphs are provided ([arXiv:2410.14684](https://arxiv.org/abs/2410.14684)):

- Agent success rates increase by **32.8%** on SWE-bench

**Why:** Architecture docs and code graphs act as a "table of contents" for your codebase. Without them, agents waste context budget exploring dead ends.

---

## 6. Documentation Must Be Accurate—Or It's Worse Than Nothing

Code examples in documentation boost pass rates from **0.39 to 0.66** ([arXiv:2503.15231](https://arxiv.org/abs/2503.15231)). But:

- **Incorrect documentation performs worse than no documentation** ([arXiv:2404.03114](https://arxiv.org/abs/2404.03114))

Agents trust docs implicitly. Outdated API signatures or wrong usage examples will be faithfully reproduced—at scale.

---

## 7. Tests Are the Agent's Safety Net

The most successful agent systems combine LLMs with traditional SE tools:

- **MANTRA** ([arXiv:2503.14340](https://arxiv.org/abs/2503.14340)): **82.8% success rate** by using test feedback as a verification loop
- Without test-driven validation: dramatically lower success rates

But there's a catch—agents that lack guardrails introduce technical debt:

- **30% increase in static analysis warnings**
- **41% increase in cognitive complexity** ([arXiv:2511.04427](https://arxiv.org/abs/2511.04427))

**The lesson:** Comprehensive test coverage doesn't just catch bugs—it's the feedback mechanism that teaches agents what "correct" looks like.

---

## References

- Borg, M., et al. (2026). "Code Quality's Impact on LLM Refactoring Performance." *arXiv:2601.02200*. https://arxiv.org/abs/2601.02200
- Sepidband, M., et al. (2025). "Halstead Complexity Metrics Predict LLM Code Generation Success." *arXiv:2505.23953*. https://arxiv.org/abs/2505.23953
- "SWE-bench Pro: Enterprise-Complexity Performance Analysis." (2025). *arXiv:2509.16941*. https://arxiv.org/abs/2509.16941
- CrossCodeEval: Multi-File Context Evaluation. (2023). *arXiv:2310.11248*. https://arxiv.org/abs/2310.11248
- "RepoGraph: Repository-Level Code Understanding." (2024). *arXiv:2410.14684*. https://arxiv.org/abs/2410.14684
- Santos, E., et al. (2025). "AI Agent Configuration Patterns in Real-World Software Engineering." *arXiv:2511.09268*. https://arxiv.org/abs/2511.09268
- "Impact of Documentation Quality on LLM Code Generation." (2025). *arXiv:2503.15231*. https://arxiv.org/abs/2503.15231
- "Incorrect Documentation and LLM Performance." (2024). *arXiv:2404.03114*. https://arxiv.org/abs/2404.03114
- MANTRA: Test-Driven Agent Validation. (2025). *arXiv:2503.14340*. https://arxiv.org/abs/2503.14340
- "Technical Debt Accumulation in AI-Assisted Development." (2025). *arXiv:2511.04427*. https://arxiv.org/abs/2511.04427
