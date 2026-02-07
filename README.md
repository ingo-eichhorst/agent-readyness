<div align="center">

# 🤖 Agent Readiness Score (ARS)

### *Measure how ready your codebase is for AI agents*

[![Go Reference](https://pkg.go.dev/badge/github.com/ingo/agent-readyness.svg)](https://pkg.go.dev/github.com/ingo/agent-readyness)
[![Go Report Card](https://goreportcard.com/badge/github.com/ingo-eichhorst/agent-readyness)](https://goreportcard.com/report/github.com/ingo-eichhorst/agent-readyness)
[![Coverage](https://img.shields.io/badge/coverage-76.2%25-brightgreen)](https://github.com/ingo-eichhorst/agent-readyness)
[![License](https://img.shields.io/github/license/ingo-eichhorst/agent-readyness)](https://github.com/ingo-eichhorst/agent-readyness/blob/main/LICENSE)
[![Release](https://img.shields.io/github/release/ingo-eichhorst/agent-readyness)](https://github.com/ingo-eichhorst/agent-readyness/releases)

[![ARS](https://img.shields.io/badge/ARS-Agent--Assisted%206.6%2F10-yellow)](https://github.com/ingo-eichhorst/agent-readyness)

[Quick Start](#-quick-start) •
[Features](#-features) •
[Installation](#-installation) •
[Usage](#-usage) •
[Documentation](#-documentation) •
[Contributing](#-contributing)

</div>

---

AI agents are already writing, refactoring, and debugging code at scale. But they don't fail gracefully like human developers—they fail catastrophically. The properties that make code "AI-friendly" are similar to those that make it "human-friendly," but agents have zero tolerance for deviation (Borg et al., 2026).

Humans compensate for bad code with intuition, tribal knowledge, and pattern recognition. Agents cannot. Where a senior developer slows down, an agent breaks.

**The Bottom Line:**
> Investing in code quality is the highest-leverage action you can take to enable AI agent productivity.

You could spend $10M/year on the best LLM API credits. Or you could refactor your God Classes, add architecture docs, and improve test coverage—and get better results with cheaper models.

The research is clear:

✅ Clean code reduces agent break rates by 7-15 percentage points
✅ Modular architecture enables 4.5x better context retrieval
✅ Documentation boosts success rates by 32.8%
✅ Test-driven workflows achieve 82.8% task completion

Agent Readiness isn't a nice-to-have—it's the difference between an agent that ships code and one that creates busywork.

📖 **[Read the detailed research evidence →](RESEARCH.md)**

---

## Quick Start

### Install

```bash
go install github.com/ingo-eichhorst/agent-readyness@latest
```

Make sure `$GOPATH/bin` (usually `~/go/bin`) is in your PATH.

### Run Your First Scan

```bash
# Scan current directory
ars scan .

# Generate beautiful HTML report
ars scan . --output-html report.html

# Enable full AI agent evaluation (requires Claude CLI)
ars scan . --enable-c7
```

**That's it!** 🎉 You'll get a comprehensive analysis of your codebase's agent-readiness across 7 research-backed categories.

---

## Features

<table>
<tr>
<td width="50%">

### **Research-Backed Analysis**
7 categories, 38+ metrics, all grounded in peer-reviewed research with inline citations

### **Beautiful Reports**
Interactive HTML reports with charts, expandable sections, and mobile-responsive design

### **AI-Powered Insights**
Optional LLM analysis for documentation quality and live agent evaluation

</td>
<td width="50%">

### **Multi-Language Support**
Auto-detects and analyzes Go, Python, and TypeScript codebases

### **Actionable Recommendations**
Ranked improvement suggestions with impact scores and effort estimates

### **Baseline Comparison**
Track progress over time by comparing against previous scans

</td>
</tr>
</table>

---

## 📋 Prerequisites

### Required

- **[Go](https://go.dev/doc/install) 1.21+** - The programming language runtime

  <details>
  <summary>📦 Install Go</summary>

  **macOS:**
  ```bash
  brew install go
  # OR download from: https://go.dev/dl/
  ```

  **Linux:**
  ```bash
  wget https://go.dev/dl/go1.22.5.linux-amd64.tar.gz
  sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf go1.22.5.linux-amd64.tar.gz
  export PATH=$PATH:/usr/local/go/bin
  ```

  **Windows:**
  ```powershell
  # Download and run installer from: https://go.dev/dl/
  # Or via Chocolatey:
  choco install golang
  ```

  Verify installation: `go version`

  </details>

### Optional (for LLM Features)

- **[Claude Code CLI](https://code.claude.com/docs/en/quickstart)** - For advanced documentation analysis (C4) and live agent evaluation (C7)

  <details>
  <summary>🤖 Install Claude Code CLI</summary>

  **macOS / Linux:**
  ```bash
  curl -fsSL https://claude.ai/install.sh | bash
  ```

  **Windows (PowerShell):**
  ```powershell
  irm https://claude.ai/install.ps1 | iex
  ```

  **Setup:**
  ```bash
  # Complete one-time OAuth authentication
  claude auth login

  # Verify installation
  claude --version
  ```

  **Note:** No API key configuration needed - the CLI handles authentication automatically.

  </details>

---

## Installation

### Via Go Install (Recommended)

```bash
go install github.com/ingo-eichhorst/agent-readyness@latest
```

The binary will be installed to `$GOPATH/bin` (usually `~/go/bin`). Make sure this is in your PATH.

### Build from Source

```bash
git clone https://github.com/ingo-eichhorst/agent-readyness.git
cd agent-readyness
go build -o ars .
```

### Pre-built Binaries

Pre-built binaries will be available on the [releases page](https://github.com/ingo-eichhorst/agent-readyness/releases) for future releases.

---

## Usage

### Basic Commands

```bash
# Scan current directory
ars scan .

# Scan specific directory
ars scan /path/to/project

# Generate interactive HTML report
ars scan . --output-html report.html

# JSON output for CI/CD integration
ars scan . --json > results.json
```

### LLM Features

ARS includes **optional AI-powered analysis** that automatically enables when Claude CLI is detected:

```bash
# LLM features auto-enabled when Claude CLI is available
ars scan .
# Output: "Claude CLI detected (claude 2.x.x) - LLM features enabled"

# Explicitly disable LLM features
ars scan . --no-llm
# Output: "LLM features disabled (--no-llm flag)"

# Enable C7 agent evaluation (requires Claude CLI)
ars scan . --enable-c7
# Performs live agent tasks to measure real-world performance
```

### Debug Mode

When investigating C7 Agent Evaluation scores, use debug mode:

```bash
# Show debug output (prompts, responses, scoring traces)
ars scan . --debug-c7

# Save responses for offline analysis (no Claude CLI calls on replay)
ars scan . --debug-c7 --debug-dir ./c7-debug

# Pipe output to files
ars scan . --debug-c7 --json > results.json 2>debug.log
```

---

## 🔍 What Gets Analyzed

Agent Readiness Score evaluates your codebase across **7 research-backed categories**:

<table>
<tr>
<th width="20%">Category</th>
<th width="40%">What It Measures</th>
<th width="40%">Key Metrics</th>
</tr>

<tr>
<td><strong>C1</strong><br/>Code Quality</td>
<td>Structural complexity and maintainability patterns that affect agent comprehension</td>
<td>
• Cyclomatic complexity<br/>
• Function length<br/>
• Code duplication<br/>
• Coupling metrics
</td>
</tr>

<tr>
<td><strong>C2</strong><br/>Semantics</td>
<td>Explicitness of types, names, and intentions that help agents understand purpose</td>
<td>
• Type annotation coverage<br/>
• Naming consistency<br/>
• Magic number detection<br/>
• Interface clarity
</td>
</tr>

<tr>
<td><strong>C3</strong><br/>Architecture</td>
<td>Structural organization and dependency patterns that enable navigation</td>
<td>
• Directory depth<br/>
• Module coupling<br/>
• Circular dependencies<br/>
• Dead code detection
</td>
</tr>

<tr>
<td><strong>C4</strong><br/>Documentation</td>
<td>Quality and completeness of human and machine-readable documentation</td>
<td>
• README presence & clarity<br/>
• Comment density<br/>
• API documentation<br/>
• Example quality (AI-evaluated)
</td>
</tr>

<tr>
<td><strong>C5</strong><br/>Temporal Dynamics</td>
<td>Change patterns and stability indicators from git history</td>
<td>
• Code churn rate<br/>
• Temporal coupling<br/>
• Hotspot identification<br/>
• Change frequency
</td>
</tr>

<tr>
<td><strong>C6</strong><br/>Testing</td>
<td>Test infrastructure that enables safe agent modifications</td>
<td>
• Test-to-source ratio<br/>
• Code coverage<br/>
• Test isolation<br/>
• Assertion quality
</td>
</tr>

<tr>
<td><strong>C7</strong><br/>Agent Evaluation</td>
<td>Live AI agent performance on real tasks (requires Claude CLI)</td>
<td>
• Task execution consistency<br/>
• Code comprehension<br/>
• Cross-file navigation<br/>
• Documentation accuracy detection
</td>
</tr>
</table>

### Score Tiers

| Score | Tier | Meaning |
|-------|------|---------|
| **8.0-10.0** | 🟢 **Agent-Ready** | Agents work efficiently with minimal supervision |
| **6.0-7.9** | 🟡 **Agent-Assisted** | Agents are productive with human oversight |
| **4.0-5.9** | 🟠 **Agent-Limited** | Agents struggle and require significant guidance |
| **0.0-3.9** | 🔴 **Agent-Hostile** | Agents fail frequently or produce incorrect results |

---

## Documentation

- **[RESEARCH.md](RESEARCH.md)** - Detailed academic evidence and citations
- **[CHANGELOG.md](CHANGELOG.md)** - Release history and upgrade guidance
- **[CONTRIBUTING.md](CONTRIBUTING.md)** - How to contribute to this project
- **[CLAUDE.md](CLAUDE.md)** - Detailed architecture for AI agents
- **[AGENTS.md](AGENTS.md)** - Precise instructions for AI coding agents

---

## 🤝 Contributing

We welcome contributions from both humans and AI agents! 🤖🤝👥

### For Human Contributors

1. Check out issues labeled [`good first issue`](https://github.com/ingo-eichhorst/agent-readyness/issues?q=is%3Aissue+is%3Aopen+label%3A%22good+first+issue%22)
2. Read [CONTRIBUTING.md](CONTRIBUTING.md) for setup and workflow
3. Submit PRs following [Conventional Commits](https://www.conventionalcommits.org/)

### For AI Coding Agents

1. Read [AGENTS.md](AGENTS.md) for precise technical boundaries
2. Follow exact commands and code patterns specified
3. Complement with [CLAUDE.md](CLAUDE.md) for architecture details

### Development Setup

```bash
# Clone and build
git clone https://github.com/ingo-eichhorst/agent-readyness.git
cd agent-readyness
go build -o ars .

# Run tests
go test ./...

# Run tests with coverage
go test ./... -coverprofile=cover.out
go tool cover -html=cover.out

# Update coverage badge (run this script to see current coverage)
./scripts/update-coverage-badge.sh

# Format code
gofmt -w .

# Run scan on the project itself
./ars scan .
```

---

## 📊 Example Output

```
Agent Readiness Score: 6.6 / 10
Tier: Agent-Assisted 🟡
────────────────────────────────────────
C1: Code Quality             7.2 / 10
C2: Semantic Explicitness    8.1 / 10
C3: Architectural Design     5.4 / 10
C4: Documentation Quality    4.8 / 10
C5: Temporal Dynamics        7.3 / 10
C6: Testing Infrastructure   9.1 / 10
C7: Agent Evaluation         8.9 / 10
────────────────────────────────────────

Top Recommendations
══════════════════════════════════════
  1. Improve Documentation Coverage
     Impact: +1.2 points
     Effort: Medium
     Action: Add missing README sections and API docs

  2. Reduce Architectural Complexity
     Impact: +0.8 points
     Effort: High
     Action: Break down large modules and reduce coupling
```

---

## Star History

If you find this project useful, please consider giving it a star! ⭐

---

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

## Acknowledgments

Built with research from leading institutions and grounded in peer-reviewed publications. See [RESEARCH.md](RESEARCH.md) for 58+ academic citations spanning:

- Software Engineering (McCabe, Fowler, Martin, Gamma)
- Programming Language Theory (Pierce, Cardelli, Wright)
- Empirical Software Studies (Nagappan, Bird, Hassan, Mockus)
- AI & LLM Research (Jimenez, Kapoor, Ouyang, Haroon, Borg)

---

<div align="center">

**Made with ❤️ for the future of AI-assisted development**

[Report Bug](https://github.com/ingo-eichhorst/agent-readyness/issues) •
[Request Feature](https://github.com/ingo-eichhorst/agent-readyness/issues)

</div>
