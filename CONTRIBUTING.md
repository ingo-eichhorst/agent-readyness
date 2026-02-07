# Contributing to Agent Readiness Score (ARS)

Thank you for your interest in contributing to ARS! This guide will help you get started with contributing to the project.

## Table of Contents

- [About the Project](#about-the-project)
- [Getting Started](#getting-started)
- [Development Workflow](#development-workflow)
- [Code Quality Standards](#code-quality-standards)
- [Testing Requirements](#testing-requirements)
- [Submitting Changes](#submitting-changes)
- [AI Agent Guidelines](#ai-agent-guidelines)
- [Community Guidelines](#community-guidelines)

## About the Project

Agent Readiness Score (ARS) is a CLI tool that measures how ready a codebase is for AI agents. It analyzes code across 7 categories (C1-C7) and produces a composite score with tier classification.

**Project Goals:**
- Provide research-backed metrics for measuring AI agent readiness
- Support multiple languages (Go, Python, TypeScript)
- Deliver actionable recommendations for improvement
- Maintain scientific rigor with proper citations

## Getting Started

### Prerequisites

- Go 1.21 or later
- Git
- (Optional) Claude CLI for C7 agent evaluation features

### Setup

1. **Fork and clone the repository:**
   ```bash
   git clone https://github.com/YOUR_USERNAME/agent-readyness.git
   cd agent-readyness
   ```

2. **Install dependencies:**
   ```bash
   go mod download
   ```

3. **Build the project:**
   ```bash
   go build -o ars .
   ```

4. **Run tests:**
   ```bash
   go test ./...
   ```

5. **Try scanning the project itself:**
   ```bash
   ./ars scan .
   ```

## Development Workflow

### 1. Understanding the Architecture

Before making changes, familiarize yourself with the architecture:

- **Pipeline Flow:** `internal/pipeline/` orchestrates the scan stages
- **Analyzers:** `internal/analyzer/{category}/` contains category-specific analyzers
- **Scoring:** `internal/scoring/` implements the scoring model
- **Output:** `internal/output/` handles terminal, JSON, and HTML rendering

See `CLAUDE.md` for detailed architecture documentation.

### 2. Finding Work

- Check [Issues](https://github.com/ingo-eichhorst/agent-readyness/issues) for open tasks
- Look for issues labeled `good first issue` or `help wanted`
- Review the [Project Roadmap](.planning/ROADMAP.md) for planned features

### 3. Creating a Branch

Use descriptive branch names:

```bash
git checkout -b feature/add-rust-support
git checkout -b fix/c3-circular-dep-detection
git checkout -b docs/improve-c7-examples
```

### 4. Making Changes

**Keep changes focused:**
- One feature/fix per pull request
- Address a single issue or implement a specific feature
- Avoid bundling unrelated changes

**Follow the project's patterns:**
- New analyzers follow `internal/analyzer/{category}/{language}.go` structure
- Language-specific implementations use interfaces like `GoAwareAnalyzer`
- Tests are colocated with implementation (`*_test.go`)

## Code Quality Standards

### Go Code Style

- Follow standard Go conventions (`gofmt`, `golint`)
- Keep functions focused and under 50 lines when possible
- Use meaningful variable names (no single-letter names except loop counters)
- Add comments for exported functions and complex logic

**Example of good code style:**

```go
// extractComplexity calculates cyclomatic complexity from the AST
func extractComplexity(node ast.Node) int {
    complexity := 1
    ast.Inspect(node, func(n ast.Node) bool {
        switch n.(type) {
        case *ast.IfStmt, *ast.ForStmt, *ast.CaseClause:
            complexity++
        }
        return true
    })
    return complexity
}
```

### Commit Messages

Use [Conventional Commits](https://www.conventionalcommits.org/) format:

```
feat(c1): add duplication detection for Python
fix(scoring): correct threshold interpolation edge case
docs(readme): add installation instructions for Windows
test(c6): add coverage for test isolation metric
```

**Types:** `feat`, `fix`, `docs`, `test`, `refactor`, `perf`, `chore`

**Scopes:** Use phase numbers (e.g., `26-01`) during active development, or category names (`c1`, `c2`, etc.)

## Testing Requirements

### Test Coverage

- All new features must include tests
- Aim for >80% coverage for new code
- Use table-driven tests for multiple scenarios

**Example test structure:**

```go
func TestC1Analyzer_Complexity(t *testing.T) {
    tests := []struct {
        name     string
        code     string
        expected float64
    }{
        {
            name:     "simple function",
            code:     "func Add(a, b int) int { return a + b }",
            expected: 1.0,
        },
        {
            name:     "function with if statement",
            code:     "func Max(a, b int) int { if a > b { return a } return b }",
            expected: 2.0,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := analyzeComplexity(tt.code)
            if result != tt.expected {
                t.Errorf("got %v, want %v", result, tt.expected)
            }
        })
    }
}
```

### Test Fixtures

- Use `testdata/` directories for fixture projects
- Document fixture structure in comments
- Keep fixtures minimal but representative

### Running Tests

```bash
# All tests
go test ./...

# With coverage
go test ./... -coverprofile=cover.out

# Specific package
go test ./internal/analyzer/c1_code_quality/...

# Specific test
go test ./internal/analyzer/c1_code_quality -run TestComplexity

# Verbose output
go test ./... -v
```

## Submitting Changes

### Pull Request Process

1. **Update your branch:**
   ```bash
   git fetch origin
   git rebase origin/main
   ```

2. **Ensure all tests pass:**
   ```bash
   go test ./...
   go build ./...
   ```

3. **Push your changes:**
   ```bash
   git push origin feature/your-feature
   ```

4. **Create a Pull Request:**
   - Use a clear, descriptive title
   - Reference related issues (e.g., "Closes #42")
   - Provide context on what changed and why
   - Include screenshots for UI changes

### PR Template

```markdown
## Description
Brief description of changes

## Related Issues
Closes #42

## Changes Made
- Added X feature
- Fixed Y bug
- Refactored Z module

## Testing
- [ ] All tests pass
- [ ] Added tests for new functionality
- [ ] Manually tested on sample projects

## Checklist
- [ ] Code follows project style guidelines
- [ ] Documentation updated (if needed)
- [ ] Commit messages follow conventional format
```

### Review Process

- Maintainers will review your PR within a few days
- Address review feedback promptly
- Be open to suggestions and iterative improvements
- Once approved, maintainers will merge your PR

## AI Agent Guidelines

**For AI coding agents (Claude, Copilot, Cursor, etc.):**

This project includes **AGENTS.md** with detailed instructions for AI agents. If you're an AI agent contributing to this project:

1. Read and follow `AGENTS.md` for technical specifics
2. Respect the boundaries defined in the "Never Touch" section
3. Use exact commands with flags from `CLAUDE.md`
4. Follow the code style examples shown above

## Community Guidelines

### Code of Conduct

We are committed to providing a welcoming and inclusive environment:

- **Be respectful:** Treat everyone with respect and consideration
- **Be collaborative:** Work together and help others learn
- **Be patient:** Everyone was a beginner once
- **Be constructive:** Provide helpful feedback and suggestions

### Communication Channels

- **GitHub Issues:** Bug reports, feature requests, questions
- **Pull Requests:** Code contributions and discussions
- **GitHub Discussions:** General questions and community chat

### Getting Help

- Check existing [Issues](https://github.com/ingo-eichhorst/agent-readyness/issues) and [Discussions](https://github.com/ingo-eichhorst/agent-readyness/discussions)
- Read `CLAUDE.md` for architecture details
- Review `.planning/PROJECT.md` for project context
- Ask questions in GitHub Discussions

## Recognition

Contributors are recognized in:
- Git commit history
- Release notes for significant contributions
- Project README (for major features)

## Project Development Methodology

This project uses **GSD (Get Stuff Done)** methodology:

- Work is organized into phases documented in `.planning/phases/`
- Each phase has research, plans, and verification documents
- See `.planning/PROJECT.md` for the overall project roadmap

**For phase-based work:**
- Check `.planning/ROADMAP.md` for current milestone
- Review phase context before starting work
- Follow the atomic commit pattern established in GSD

### Updating the Changelog

When completing a phase or making user-visible changes:

1. **During development:**
   - Add entry to `## [Unreleased]` section in CHANGELOG.md
   - Use appropriate subsection (Added/Changed/Fixed/Removed/Deprecated/Security)
   - Focus on user-visible changes (not internal refactors or test additions)
   - Use clear, concise language with examples where helpful
   - Commit with: `docs: update CHANGELOG for Phase XX`

2. **When releasing a version:**
   - Move `[Unreleased]` entries to new version section
   - Add version number and date: `## [X.Y.Z] - YYYY-MM-DD`
   - Create git tag: `git tag vX.Y.Z`
   - Update comparison links at bottom of file
   - Mark breaking changes prominently in a **BREAKING CHANGES** section

**Example changelog entries:**

```markdown
### Added
- C7 scoring bug fixed - M2/M3/M4 metrics now produce non-zero scores
- Debug infrastructure with `--debug-c7` flag for score transparency

### Changed
- Python and TypeScript language support added

### Fixed
- Circular dependency detection now handles import cycles correctly
```

**What NOT to include:**
- Internal refactors with no user impact
- Test additions (unless they signal new feature coverage)
- Implementation details ("Updated MetricExtractor signature")
- TODOs or planned work

## License

By contributing to ARS, you agree that your contributions will be licensed under the [MIT License](LICENSE).

---

## Resources

- [Open Source Guides](https://opensource.guide/)
- [How to Contribute to Open Source](https://opensource.guide/how-to-contribute/)
- [Conventional Commits](https://www.conventionalcommits.org/)
- [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments)

**Thank you for contributing to Agent Readiness Score!** 🚀
