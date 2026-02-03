# Agent Readiness Score (ARS)

**Measure how ready your codebase is for AI agents.**

---

## Why Agent Readiness?

AI agents are already writing, refactoring, and debugging code at scale. But they don't fail gracefully like human developers—they **fail catastrophically**. The properties that make code "AI-friendly" are similar to those that make it "human-friendly," but **agents have zero tolerance for deviation** ([Borg et al., 2026](https://arxiv.org/abs/2601.02200)).

Humans compensate for bad code with intuition, tribal knowledge, and pattern recognition. Agents cannot. Where a senior developer slows down, an agent **breaks**.

**The bottom line:** Code quality isn't just about maintainability anymore—it's about whether AI agents can function in your codebase at all.

**[→ Read the detailed research evidence](RESEARCH.md)**

---

## The Bottom Line

**Investing in code quality is the highest-leverage action you can take to enable AI agent productivity.**

You could spend $10M/year on the best LLM API credits. Or you could refactor your God Classes, add architecture docs, and improve test coverage—and get **better results with cheaper models**.

The research is clear:

- ✅ Clean code reduces agent break rates by **7-15 percentage points**
- ✅ Modular architecture enables **4.5x better context retrieval**
- ✅ Documentation boosts success rates by **32.8%**
- ✅ Test-driven workflows achieve **82.8% task completion**

**Agent Readiness isn't a nice-to-have—it's the difference between an agent that ships code and one that creates busywork.**

---

## What's Next?

This repository provides:

1. **The Agent Readiness Score (ARS)** — a research-backed metric for measuring codebase AI-readiness
2. **Actionable improvement patterns** — concrete refactoring strategies with before/after examples
3. **Measurement tools** — scripts to calculate ARS for your codebase

## Installation

```bash
go install github.com/ingo-eichhorst/agent-readyness@latest
```

Or build from source:

```bash
git clone https://github.com/ingo-eichhorst/agent-readyness.git
cd agent-readyness
go build -o ars .
# or
go run . scan .
```

## Usage

```bash
# Scan current directory
ars scan .

# Scan with JSON output
ars scan . --json

# Generate HTML report
ars scan . --output-html report.html

# Set minimum score threshold (exits with code 2 if below)
ars scan . --threshold 6.0

# Compare against baseline
ars scan . --baseline previous.json

# Enable LLM-based documentation analysis (requires ANTHROPIC_API_KEY)
ars scan . --enable-c4-llm

# Enable agent evaluation (requires claude CLI)
ars scan . --enable-c7
```

**Supported languages:** Go, Python, TypeScript (auto-detected)

## Test

go test ./... -coverprofile=coverage.out

