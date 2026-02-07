# Agent Readiness Score (ARS)

[![Go Reference](https://pkg.go.dev/badge/github.com/ingo/agent-readyness.svg)](https://pkg.go.dev/github.com/ingo/agent-readyness) [![Go Report Card](https://goreportcard.com/badge/github.com/ingo-eichhorst/agent-readyness)](https://goreportcard.com/report/github.com/ingo-eichhorst/agent-readyness) [![License](https://img.shields.io/github/license/ingo-eichhorst/agent-readyness)](https://github.com/ingo-eichhorst/agent-readyness/blob/main/LICENSE) [![Release](https://img.shields.io/github/release/ingo-eichhorst/agent-readyness)](https://github.com/ingo-eichhorst/agent-readyness/releases)

[![ARS](https://img.shields.io/badge/ARS-Agent--Assisted%206.6%2F10-yellow)](https://github.com/ingo-eichhorst/agent-readyness)

**Measure how ready your codebase is for AI agents.**

---

## 🎯 Why This Matters

AI agents are already writing, refactoring, and debugging code at scale. But unlike human developers who slow down when confused, **agents break catastrophically** when encountering poor code quality.

The bottom line is simple:

> **Code quality isn't just about maintainability anymore—it's about whether AI agents can function in your codebase at all.**

### The Research is Clear

- ✅ Clean code reduces agent break rates by **7-15 percentage points**
- ✅ Modular architecture enables **4.5x better context retrieval**
- ✅ Documentation boosts success rates by **32.8%**
- ✅ Test-driven workflows achieve **82.8% task completion**

**You could spend $10M/year on the best LLM API credits, or you could refactor your God Classes, add architecture docs, and improve test coverage—and get better results with cheaper models.**

📖 **[Read the detailed research evidence →](RESEARCH.md)**

---

## 🚀 Quick Start

### One-Line Install

**macOS / Linux:**
```bash
curl -fsSL https://raw.githubusercontent.com/ingo-eichhorst/agent-readyness/main/install.sh | bash
```

**Windows (PowerShell):**
```powershell
irm https://raw.githubusercontent.com/ingo-eichhorst/agent-readyness/main/install.ps1 | iex
```

**Or via Go:**
```bash
go install github.com/ingo-eichhorst/agent-readyness@latest
```

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

## ✨ Features

<table>
<tr>
<td width="50%">

### 📊 **Research-Backed Analysis**
7 categories, 38+ metrics, all grounded in peer-reviewed research with inline citations

### 🎨 **Beautiful Reports**
Interactive HTML reports with charts, expandable sections, and mobile-responsive design

### 🧠 **AI-Powered Insights**
Optional LLM analysis for documentation quality and live agent evaluation

</td>
<td width="50%">

### 🌍 **Multi-Language Support**
Auto-detects and analyzes Go, Python, and TypeScript codebases

### 🎯 **Actionable Recommendations**
Ranked improvement suggestions with impact scores and effort estimates

### 📈 **Baseline Comparison**
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

## 📥 Installation

### Method 1: Go Install (Recommended)

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

# Disable LLM features (enabled automatically when Claude CLI is detected)
ars scan . --no-llm

# Enable C7 agent evaluation (requires claude CLI)
ars scan . --enable-c7
```

**Supported languages:** Go, Python, TypeScript (auto-detected)

### LLM Features

ARS includes optional LLM-powered analysis for documentation quality (C4) and agent evaluation (C7). These features use the [Claude Code CLI](https://docs.claude.ai/docs/claude-code-overview) and are **automatically enabled** when the CLI is detected:

```bash
# LLM features auto-enabled when Claude CLI is available
ars scan .
# Output: "Claude CLI detected (claude 2.x.x) - LLM features enabled"

# Explicitly disable LLM features
ars scan . --no-llm
# Output: "LLM features disabled (--no-llm flag)"

# CLI not installed
ars scan .
# Output: "Claude CLI not found - LLM features disabled"
```

**Install Claude Code CLI:**
```bash
# macOS/Linux
curl -fsSL https://claude.ai/install.sh | bash

# Or via Homebrew
brew install --cask claude-code

# Or via npm
npm install -g @anthropic-ai/claude-code
```

No API key configuration needed - the CLI handles authentication.

### C7 Debug Mode

When investigating C7 Agent Evaluation scores, use debug mode to inspect what the agent sees and how responses are scored:

```bash
# Show debug output on stderr (normal output unchanged on stdout)
ars scan . --debug-c7

# Pipe normal output to file while viewing debug on terminal
ars scan . --debug-c7 --json > results.json 2>debug.log

# Save responses for offline analysis
ars scan . --debug-c7 --debug-dir ./c7-debug

# Replay saved responses (fast, no Claude CLI calls)
ars scan . --debug-c7 --debug-dir ./c7-debug
```

Debug output includes:
- Per-metric, per-sample prompt text (truncated)
- Full agent response (truncated in terminal, full in saved files)
- Score breakdown with heuristic indicator traces
- Timing data per sample and per metric

The `--debug-dir` flag enables response persistence:
- **First run**: Executes Claude CLI normally and saves all responses as JSON files
- **Subsequent runs**: Loads saved responses instead of calling Claude CLI (replay mode)
- Replay mode enables fast iteration on heuristic scoring without API costs

Debug output goes exclusively to stderr, so JSON output (`--json`) remains valid on stdout.

## Contributing

We welcome contributions from both humans and AI agents! 🤝

**For human contributors:**
- Read [CONTRIBUTING.md](CONTRIBUTING.md) for setup, workflow, and guidelines
- Check [Issues](https://github.com/ingo-eichhorst/agent-readyness/issues) for tasks labeled `good first issue`
- Join discussions in [GitHub Discussions](https://github.com/ingo-eichhorst/agent-readyness/discussions)

**For AI coding agents:**
- Read [AGENTS.md](AGENTS.md) for precise technical instructions
- Follow the exact commands, code patterns, and boundaries specified
- Complement this with [CLAUDE.md](CLAUDE.md) for detailed architecture

All contributions must:
- Include tests that pass (`go test ./...`)
- Follow Go conventions (`gofmt`)
- Use [Conventional Commits](https://www.conventionalcommits.org/) format
- Maintain code quality standards

## Test

go test ./... -coverprofile=cover.out

