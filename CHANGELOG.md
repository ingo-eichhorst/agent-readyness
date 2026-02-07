# Changelog

All notable changes to Agent Readiness Score (ARS) are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.0.6] - 2026-02-07

### Added
- **Call Trace Modals** - Interactive transparency for all metric scores
  - "View Trace" button on every metric in HTML reports
  - C7 metrics show full Claude prompt, response, and indicator breakdown
  - C1-C6 metrics show raw values, scoring breakpoints, and top-5 worst offenders
  - Syntax highlighting for JSON and shell commands in trace content
  - Progressive enhancement with `<details>` fallback for non-JS environments
- **Improvement Prompt Modals** - AI-ready prompts for every metric
  - "Improve" button generates copy-paste prompts for AI agents
  - Research-backed 4-section structure: Context, Build/Test Commands, Task, Verification
  - Project-specific interpolation with evidence data (files, functions, scores)
  - Copy-to-clipboard with fallback for file:// protocol
  - All 7 categories (C1-C7) covered with tailored prompt templates
- **Evidence Data Flow** - Top-5 worst offenders for every metric
  - Evidence items with file paths, line numbers, values, and descriptions
  - Flows through entire pipeline from extraction to JSON output
  - Visible in JSON with `sub_scores[].evidence` field
  - Powers both trace modals and improvement prompts
- **Modal UI Infrastructure** - Native dialog component for HTML reports
  - Responsive design with mobile viewport support (375px+)
  - Accessibility support: keyboard navigation, focus trapping, ARIA attributes
  - Three close methods: Escape key, X button, backdrop click
  - Independent scrolling for long content
  - iOS scroll lock and progressive enhancement

### Fixed
- C5 ChurnRate test bound increased from 100k to 500k lines/commit
  - Accommodates legitimate bulk changes (documentation, refactoring)
  - Aligns with scoring config (tops out at 1,000 for score=1)
  - Still catches pathological cases (parser bugs, data corruption)
- CI workflow Go version upgraded from 1.21 to 1.25
  - Matches go.mod requirement (1.25.1)
  - Eliminates "go: no such tool 'covdata'" errors
  - Fixes coverage profiling for packages without test files

### Changed
- HTML report file size with C7 data stays under 500KB budget
- All 38 metrics now map to category-level improvement prompt templates
- JSON backward compatibility maintained with v0.0.5 baseline format

## [0.0.5] - 2026-02-06

### Fixed
- C7 scoring bug where M2/M3/M4/M5 metrics produced near-zero scores due to indicator saturation
  - Implemented grouped indicator logic to properly weight multiple-choice responses
  - All C7 metrics now produce meaningful scores across the 1-10 range

### Added
- Debug infrastructure with `--debug-c7` flag for C7 score transparency
  - Shows full prompts sent to Claude CLI
  - Displays complete responses with matched indicators
  - Reveals score calculation with indicator breakdown
- Response persistence with `--debug-dir` flag
  - Saves C7 responses to disk for replay
  - Enables analysis without re-running expensive Claude CLI calls
  - Supports iterative testing of scoring heuristics
- Verbose debug output in terminal with `--debug-c7` flag
- Fixture-based tests using real Claude responses captured from production runs

## [0.0.4] - 2026-02-05

### Added
- Research citations for all 7 analysis categories (58 citations total)
  - C1 Code Health: 7 citations (McCabe, Fowler, Borg, Parnas, Martin, Gamma, Chowdhury)
  - C2 Semantic Explicitness: 11 citations (Gao, Pierce, Cardelli, Wright, Meta, Hoare, Borg)
  - C3 Architectural Navigability: 13 citations (Parnas, Stevens, Gamma, Chidamber, Lakos, Martin, Borg)
  - C4 Documentation Quality: 14 citations (Knuth, Robillard, Garousi, Sadowski, Prana, Borg)
  - C5 Temporal Dynamics: 10 citations (Graves, Nagappan, Kim, Gall, Bird, Hassan, Tornhill, Borg)
  - C6 Testing Infrastructure: 8 citations (Beck, Mockus, Meszaros, Nagappan, Luo, Borg)
  - C7 Agent Evaluation: 9 citations (Jimenez, Kapoor, Ouyang, Haroon, Havare, Wen, Butler, Borg)
- MECE (Mutually Exclusive, Collectively Exhaustive) C7 metrics framework
  - M1: Task Execution Consistency (instruction following)
  - M2: Code Behavior Comprehension (program understanding)
  - M3: Cross-File Navigation (codebase traversal)
  - M4: Identifier Interpretability (naming clarity)
  - M5: Documentation Accuracy Detection (doc quality assessment)
- Parallel execution framework for C7 metrics with thread-safe coordination
- Real-time progress display for C7 evaluation showing current metric, token count, and cost
- Research-based scoring thresholds with inline documentation
- Citation quality protocols documented in `docs/CITATION-GUIDE.md`
- Expandable "Research Evidence" sections in HTML reports with inline citations

### Changed
- All 38 metric descriptions now include research citations with DOI/ArXiv/publisher links
- C7 metrics split from monolithic task-based approach to 5 focused, measurable dimensions

## [0.0.3] - 2026-02-04

### BREAKING CHANGES
- `--enable-c4-llm` flag removed - LLM features now auto-enabled when Claude CLI detected
  - Migration: Remove flag from scripts, use `--no-llm` to disable if needed
  - No ANTHROPIC_API_KEY environment variable needed - CLI handles authentication
  - LLM features (C4 content evaluation, C7 agent assessment) work out-of-box when `claude` command available

### Added
- Badge generation with `--badge` flag for README visibility
  - Shields.io format: `![ARS](https://img.shields.io/badge/...)`
  - Color-coded by tier: Agent-Ready (green), Agent-Assisted (yellow), Agent-Limited (orange), Agent-Hostile (red)
  - Shows tier name and score (e.g., "Agent-Assisted 6.6/10")
- Enhanced HTML reports with metric descriptions
  - Brief descriptions for quick understanding
  - Detailed descriptions with research context
  - CSS-only expandable sections (no JavaScript required)
  - Auto-expand for low-scoring metrics
- Analyzer reorganization with 7 category subdirectories
  - `internal/analyzer/c1_code_quality/` through `internal/analyzer/c7_agent/`
  - Shared utilities in `internal/analyzer/shared/` package
  - Type aliases in root `analyzer.go` for backward compatibility
- Standard README badges (Go Reference, Go Report Card, License, Release)
- Coverage file detection improvements

### Changed
- C4 and C7 now use `internal/agent/` package with unified `Evaluator` interface
- Switched from Anthropic API SDK to Claude CLI for all LLM features
  - Auto-detection of `claude` command availability
  - No configuration or API keys needed
  - Better handling of authentication and session management
- Improved test coverage documentation with `cover.out` examples

### Removed
- Anthropic SDK dependency (`github.com/anthropics/anthropic-sdk-go`)
- `ANTHROPIC_API_KEY` environment variable requirement
- `--enable-c4-llm` CLI flag (replaced by automatic detection)

## [0.0.2] - 2026-02-03

### Added
- Multi-language support for Python and TypeScript
  - Tree-sitter parsers for AST analysis
  - Language-specific metrics for C1, C2, C3, and C6
  - Mixed-language project support with per-language breakdowns
- C2 Semantic Explicitness category
  - Type annotation coverage
  - Naming consistency patterns
  - Magic number detection
  - Documentation string presence
  - Explicit vs implicit code patterns
- C4 Documentation Quality category with optional LLM analysis
  - README presence, structure, and length
  - Code comment density and quality
  - API documentation coverage
  - Example/tutorial presence
  - LLM-based content quality evaluation with `--enable-c4-llm` flag
- C5 Temporal Dynamics category (git-based analysis)
  - Code churn rate (files changed frequently)
  - Temporal coupling (files changed together)
  - Contributor ownership patterns
  - Commit stability metrics
  - Requires `.git` directory - gracefully degrades if unavailable
- C7 Agent Evaluation category with live Claude CLI interaction
  - Repository comprehension tasks
  - Bug diagnosis scenarios
  - Code generation challenges
  - Documentation usage tests
  - Requires Claude CLI with `--enable-c7` flag
- HTML report output with `--output-html` flag
  - Radar chart visualization of all 7 categories
  - Self-contained single file (no external dependencies)
  - Responsive design for desktop and mobile
  - Research citations and metric explanations
- Baseline comparison with `--baseline` flag
  - Compare current scan against previous JSON output
  - Show score deltas and trend indicators
  - Track improvement over time
- Configurable scoring via `.arsrc.yml`
  - Custom weights per category
  - Custom thresholds per metric
  - Project-specific tuning support

### Changed
- Composite scoring now weights all 7 categories (C1-C7)
- JSON output format expanded to include new categories and metrics
- Terminal output shows all 7 category scores when available

## [0.0.1] - 2026-02-03

### Added
- C7 Agent Evaluation category (initial implementation)
  - Live agent assessment using Claude CLI
  - Task-based evaluation across 4 dimensions
  - Cost estimation and user confirmation
  - Optional feature with `--enable-c7` flag

## [1.0.0] - 2026-02-01

Initial release of Agent Readiness Score (ARS) - a CLI tool that measures codebase readiness for AI agents.

### Added
- Go language support with AST-based analysis using `go/packages`
- C1 Code Health category
  - Cyclomatic complexity (average and maximum per function)
  - Function length (lines per function)
  - File size metrics
  - Afferent coupling (incoming dependencies)
  - Efferent coupling (outgoing dependencies)
  - Code duplication detection via AST hashing
- C3 Architectural Navigability category
  - Directory depth analysis
  - Module fanout (import complexity)
  - Circular dependency detection
  - Import path complexity
  - Dead code detection (unused exports)
- C6 Testing Infrastructure category
  - Test file detection and classification
  - Test-to-code ratio calculation
  - Coverage report parsing (go-cover, LCOV, Cobertura formats)
  - Test isolation metrics (external dependency usage)
  - Assertion density analysis
- Scoring model with piecewise linear interpolation
  - Per-category scores (1-10 scale)
  - Composite score weighted by category importance
  - Tier classification: Agent-Ready (8-10), Agent-Assisted (6-8), Agent-Limited (4-6), Agent-Hostile (1-4)
  - Configurable thresholds and weights
- Terminal output with ANSI colors
  - Composite score and tier rating
  - Per-category breakdown
  - Top 5 improvement recommendations
  - Verbose mode with per-metric details
- JSON output with `--json` flag for CI integration
- Configuration system via `.arsrc.yml`
  - Custom category weights
  - Custom metric thresholds
  - Project-specific overrides
- CLI threshold gating with `--threshold` flag
  - Exit code 0: Success
  - Exit code 1: Error
  - Exit code 2: Score below threshold
- File discovery with smart exclusions
  - Respects `.gitignore` patterns
  - Excludes `vendor/` and generated files
  - Handles symlinks, permissions, and Unicode paths
- Progress indicators with TTY detection
- Recommendation engine with impact ranking
  - Score improvement estimates
  - Effort level classification (Low/Medium/High)
  - Agent-readiness framing

[Unreleased]: https://github.com/ingo-eichhorst/agent-readyness/compare/v0.0.6...HEAD
[0.0.6]: https://github.com/ingo-eichhorst/agent-readyness/compare/v0.0.5...v0.0.6
[0.0.5]: https://github.com/ingo-eichhorst/agent-readyness/compare/v0.0.4...v0.0.5
[0.0.4]: https://github.com/ingo-eichhorst/agent-readyness/compare/v0.0.3...v0.0.4
[0.0.3]: https://github.com/ingo-eichhorst/agent-readyness/compare/v0.0.2...v0.0.3
[0.0.2]: https://github.com/ingo-eichhorst/agent-readyness/compare/v0.0.1...v0.0.2
[0.0.1]: https://github.com/ingo-eichhorst/agent-readyness/compare/v1...v0.0.1
[1.0.0]: https://github.com/ingo-eichhorst/agent-readyness/releases/tag/v1
