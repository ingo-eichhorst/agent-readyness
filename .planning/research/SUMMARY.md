# Project Research Summary: ARS v0.0.3

**Project:** Agent-Ready Score (ARS) v0.0.3
**Domain:** CLI tool enhancements — Claude Code migration, badge generation, HTML improvements
**Researched:** 2026-02-03
**Confidence:** HIGH

## Executive Summary

ARS v0.0.3 focuses on architectural improvements and enhanced output capabilities rather than new analysis categories. The milestone involves four major changes: replacing the Anthropic SDK with Claude Code CLI for all LLM operations (unifying authentication and reducing dependencies), generating shields.io-compatible badges for README visibility, reorganizing the flat analyzer directory into category-based subdirectories (31 files across 7 categories), and enhancing HTML reports with collapsible sections using native HTML5 elements.

The recommended approach is to extend the existing `internal/agent/executor.go` pattern (already used for C7) to handle C4 documentation evaluation, eliminating the SDK entirely. For badges, use shields.io URL generation rather than local SVG creation to avoid dependency bloat. The analyzer reorganization should use subdirectories with root-level re-exports to maintain backward compatibility. HTML enhancements should leverage CSS-only collapsible sections (`<details>/<summary>`) to avoid Content Security Policy issues.

Key risks center on CLI subprocess management (orphaned processes, JSON schema changes) and the analyzer reorganization affecting 90+ import paths. The critical decision point is whether to fully remove the Anthropic SDK or retain it for C4's cost-sensitive evaluation needs. The research shows prompt caching (90% cost reduction) is not available via CLI, making SDK removal potentially expensive for C4 LLM analysis.

## Key Findings

### Recommended Stack

**Stack changes are subtractive, not additive.** v0.0.3 requires zero new Go dependencies. The key change is removing `github.com/anthropics/anthropic-sdk-go` and extending the existing `os/exec` + `claude` CLI integration already used for C7.

**Core technologies (changes only):**
- **REMOVE:** `github.com/anthropics/anthropic-sdk-go` v1.20.0 — replaced by CLI invocation
- **EXTEND:** `internal/agent/executor.go` — add `EvaluateContent()` method for C4
- **USE:** shields.io URL generation — pure string formatting, no dependencies
- **ENHANCE:** `html/template` with `<details>/<summary>` — native HTML5, no JavaScript

**Net effect:** One dependency removed, zero added. The existing codebase has all required infrastructure.

### Expected Features

The research documents analyzed v0.0.3 features against the GitHub issues:

**Must have (v0.0.3 scope):**
- **Issue #6:** Claude Code CLI for C4 LLM analysis (remove direct API calls)
- **Issue #5:** Badge URL generation with CLI flag `--badge` (url/markdown/html formats)
- **Issue #3:** Analyzer reorganization into category subdirectories (c1/, c2/, ..., c7/)
- **Issue #7:** HTML collapsible sections with metric descriptions

**Should have (enhancements):**
- Claude CLI version detection and validation
- Process group management for proper subprocess cleanup
- Root-level re-exports in `internal/analyzer/analyzer.go` for backward compatibility
- Smart defaults for collapsible sections (expand categories scoring <6.0)

**Defer (out of scope for v0.0.3):**
- Issue #2: Test coverage flag (separate issue, not researched)
- Issue #4: README badges (separate issue, not researched)
- Multi-agent debate for C7 (future enhancement)
- Headless Claude Code task execution for genuine agent-in-the-loop testing (C7 stretch goal)

### Architecture Approach

The ARS codebase has a clean pipeline architecture that supports all v0.0.3 changes with minimal disruption. The pipeline flows: discovery → parse → analyze → score → recommend → output. Each stage is isolated in its own package under `internal/`.

**Major components affected:**
1. **internal/agent/executor.go** — Extend with `EvaluateContent()` for C4, reuse for C7 scoring
2. **internal/analyzer/** — Reorganize 31 files into 7 subdirectories (c1/ through c7/), add re-exports
3. **internal/output/** — Add `badge.go` for URL generation, parallel to `json.go`/`html.go`/`terminal.go`
4. **internal/output/templates/** — Wrap categories in `<details>/<summary>`, update CSS

**Integration points:**
- C4 and C7 both switch from `internal/llm.Client` (SDK) to `agent.Executor` (CLI)
- Badge generation slots into pipeline Stage 4 (render output) alongside existing formats
- HTML templates gain collapsible sections without JavaScript (CSP-safe)
- Analyzer reorganization uses re-exports to avoid breaking pipeline imports

### Critical Pitfalls

1. **Claude CLI JSON Schema Instability** — The CLI's JSON output format may change between versions. Prevention: pin minimum CLI version, use defensive parsing with detailed error messages, test against multiple versions in CI.

2. **Subprocess Timeout Not Killing Child Processes** — `exec.CommandContext` kills only the direct child, orphaning Node.js workers and language servers. Prevention: use process groups (`SysProcAttr.Setpgid`) to kill entire process tree, set `WaitDelay > 0` to avoid blocking on orphaned I/O pipes.

3. **Import Path Breakage During Reorganization** — Moving 31 analyzer files affects 90+ import paths across the codebase. Prevention: create subdirectories with root-level re-exports, move one category at a time with full test passes, use long-form import paths.

4. **Removing Anthropic SDK Breaks C4 Cost Management** — The SDK provides prompt caching (90% cost reduction), token tracking, and retry logic that the CLI doesn't expose. Prevention: DECISION POINT — either keep SDK for C4 evaluation or accept 5-10x cost increase and estimate tokens rather than measure them.

5. **Inline JavaScript Breaks Content Security Policy** — HTML reports may be blocked by CSP when opened from `file://` URLs or shared via email. Prevention: use CSS-only interactive features (`<details>/<summary>`, `:target`, checkbox toggles), progressive enhancement.

## Implications for Roadmap

Based on dependencies, risk analysis, and architectural integration points, the recommended phase structure prioritizes low-risk additive changes before structural changes:

### Phase 1: Badge Generation (Low Risk, Fully Additive)

**Rationale:** This requires no changes to existing code paths. It's a pure addition to the output stage. Shields.io URL generation is simple string formatting with no external dependencies.

**Delivers:** CLI flag `--badge <format>` outputting url/markdown/html badge strings

**Implementation:**
- Create `internal/output/badge.go` with `GenerateBadgeURL()` and `FormatBadge()` functions
- Add flag to `cmd/scan.go`: `--badge url|markdown|html`
- Wire into pipeline Stage 4 alongside JSON/HTML/Terminal output

**Avoids:** Network dependency pitfall by using shields.io URLs (not local SVG generation)

**Duration estimate:** Low complexity, well-isolated

---

### Phase 2: HTML Enhancements (Low Risk, Template Only)

**Rationale:** Template-only changes with no Go code modifications. Uses native HTML5 `<details>/<summary>` elements which work without JavaScript and have 96%+ browser support.

**Delivers:** Collapsible category sections in HTML reports with metric descriptions

**Implementation:**
- Modify `internal/output/templates/report.html` to wrap categories in `<details>` elements
- Update `internal/output/templates/styles.css` with details/summary styling
- Add smart defaults: `open` attribute for categories scoring <6.0

**Avoids:** CSP and JavaScript pitfalls by using CSS-only interactive features

**Duration estimate:** Low complexity, easy to preview and iterate

---

### Phase 3: Claude Code Integration (Medium Risk, Behavior Change)

**Rationale:** Most complex change but critical for SDK removal. Builds on existing `agent.Executor` pattern from C7. Should come after feature work to allow thorough testing of the new integration before structural reorganization.

**Delivers:** Unified Claude Code CLI for C4 and C7, removal of Anthropic SDK dependency

**Implementation:**
- Add `EvaluateContent()` method to `internal/agent/executor.go`
- Update C4 analyzer (`c4_documentation.go`) to use `agent.Executor` instead of `llm.Client`
- Update C7 scorer to use `agent.Executor` for response scoring
- Add Claude CLI version detection with minimum version requirement
- Implement process group cleanup to prevent orphaned subprocesses
- Remove `internal/llm/client.go` or mark deprecated
- Update flag handling (no `ANTHROPIC_API_KEY` required)

**Critical decision:** Should C4 keep SDK for cost-sensitive evaluation? The CLI lacks prompt caching (90% cost reduction), token tracking, and Haiku model selection.

**Avoids:** Subprocess timeout pitfall via process groups, JSON schema pitfall via version checking

**Duration estimate:** Medium complexity, requires integration testing with Claude CLI

---

### Phase 4: Analyzer Reorganization (Medium Risk, Structural)

**Rationale:** Most invasive change but purely structural. Do this last after feature work is stable. The existing flat structure with naming conventions (`c1_python.go`) already provides logical separation. Reorganization improves navigability but requires careful import path management.

**Delivers:** Category-based subdirectories (c1/ through c7/) with root-level re-exports

**Implementation:**
- Create subdirectory structure: `internal/analyzer/{c1,c2,c3,c4,c5,c6,c7}/`
- Move files in order of increasing complexity: C5/C7/C4 (no language variants) first, then C1/C2/C3/C6
- Create `internal/analyzer/analyzer.go` with re-exported constructors (`NewC1Analyzer()`, etc.)
- Keep `helpers.go` at root level
- Update package declarations in moved files
- Run full test suite after each category move

**Avoids:** Import breakage pitfall via re-exports maintaining backward compatibility

**Duration estimate:** Medium complexity, requires careful execution and testing

---

### Phase Ordering Rationale

- **Badge generation first:** Zero risk, fully isolated, delivers immediate user value
- **HTML enhancements second:** Template-only, no code changes, easy to preview
- **CLI integration third:** Complex but needed before reorganization to establish stable interfaces
- **Reorganization last:** Most invasive, do after feature work stabilizes

**Critical path:** Phase 3 (CLI integration) is the blocker for SDK removal. Phases 1 and 2 can proceed independently and even ship before Phase 3 completes.

### Research Flags

Phases likely needing deeper research during planning:
- **Phase 3 (CLI integration):** Need to verify Claude CLI JSON output stability across versions, test subprocess lifecycle edge cases, decide SDK retention strategy for C4

Phases with standard patterns (skip research-phase):
- **Phase 1 (badges):** URL generation is straightforward string formatting, shields.io API is stable
- **Phase 2 (HTML):** Native HTML5 elements, well-documented, no compatibility concerns
- **Phase 4 (reorganization):** Go package reorganization is a known pattern, tooling exists

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | Verified against official Go docs, Claude CLI reference, existing codebase |
| Features | HIGH | Directly mapped to GitHub issues #3, #5, #6, #7 |
| Architecture | HIGH | Based on direct codebase analysis of 10+ files totaling 2,500+ lines |
| Pitfalls | MEDIUM-HIGH | Verified with official Go docs and GitHub issues, some are inferred from patterns |

**Overall confidence:** HIGH

### Gaps to Address

- **Claude CLI version stability:** The JSON output schema is documented but may change. Need to establish minimum version requirement and test against multiple versions.

- **C4 SDK decision:** Research shows prompt caching saves 90% cost but CLI doesn't support it. Need product decision: accept 5-10x cost increase or keep SDK for C4 evaluation.

- **Analyzer reorganization scope:** Current flat structure with naming conventions already works well. Validate that subdirectory reorganization provides enough value to justify the risk and effort.

- **Process group portability:** The `syscall.SysProcAttr{Setpgid: true}` approach works on Unix but needs Windows equivalent (`CreationFlags: CREATE_NEW_PROCESS_GROUP`). Cross-platform testing required.

## Sources

### Primary (HIGH confidence)
- **STACK.md** — Official Claude Code CLI reference, shields.io documentation, Go stdlib docs
- **ARCHITECTURE.md** — Direct codebase analysis of 10+ source files including pipeline.go, executor.go, analyzer structure
- **PITFALLS.md** — Go Issue #22485, Claude Code GitHub issues #9058 and #14442, official Go exec package docs

### Secondary (MEDIUM confidence)
- **FEATURES.md** — This was actually v2 feature research (C2/C4/C5/C7 categories), not v0.0.3 features. However, C4 and C7 implementation details informed the Claude Code integration approach.

### Tertiary (LOW confidence)
- None — all research is based on official sources or direct codebase analysis

---
*Research completed: 2026-02-03*
*Ready for roadmap: yes*
