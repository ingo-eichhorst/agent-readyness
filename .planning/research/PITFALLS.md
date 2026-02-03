# Pitfalls Research: ARS v0.0.3

**Milestone:** v0.0.3 (Claude Code CLI integration, analyzer reorganization, badges, HTML enhancements)
**Researched:** 2026-02-03
**Confidence:** MEDIUM-HIGH (verified with official Go docs, shields.io docs, and Claude Code CLI reference)

This document catalogs pitfalls specific to the v0.0.3 milestone changes:
1. Replacing direct Anthropic API calls with CLI subprocess invocation
2. Reorganizing Go package structure mid-project
3. Generating shields.io badges
4. Adding interactive HTML elements to static reports

> **Note:** This document supersedes the previous v2 expansion pitfalls for this milestone.
> The v2 document remains valid for broader concerns (Tree-sitter parsing, C5 git forensics, etc.).

---

## Critical Pitfalls

These mistakes can cause rewrites, major delays, or user-facing failures.

### 1. Claude CLI JSON Output Schema Instability

**Risk:** The Claude Code CLI is under active development. The JSON output format (`--output-format json`) may change between versions, breaking the `parseJSONOutput` function in `internal/agent/executor.go`. The current code assumes a fixed structure:
```json
{"type":"result","session_id":"abc123","result":"..."}
```

A [GitHub issue](https://github.com/anthropics/claude-code/issues/9058) confirms that Claude Code cannot guarantee output matches a specific JSON schema. The `--output-format json` returns Claude Code's wrapper structure, with actual data nested inside. Recent changes (November 2025) moved the `output_format` parameter to `output_config.format`.

**Warning signs:**
- Tests pass locally but fail in CI (different Claude CLI versions)
- Users report "failed to parse CLI output" errors after updating Claude CLI
- JSON unmarshal errors with unexpected fields
- Existing `parseJSONOutput` function returns errors on valid CLI output

**Prevention:**
1. Pin Claude CLI version in documentation and CI (minimum version requirement)
2. Add version compatibility check: run `claude --version` and compare against known-good versions
3. Use defensive parsing that ignores unknown fields (Go's `json.Unmarshal` does this by default, but validate expected fields exist)
4. Add a version-mismatch warning message with upgrade instructions
5. Wrap JSON parsing with clear error messages including the actual output received (already partially implemented with `preview` in `parseJSONOutput`)
6. Consider using `--output-format stream-json` with `--include-partial-messages` for more robust streaming, but note this is not backwards-compatible

**Phase impact:** Phase 1 (CLI integration) - Must establish version checking and defensive parsing from the start.

**Sources:**
- [Claude Code CLI reference](https://code.claude.com/docs/en/cli-reference)
- [GitHub Issue: JSON Schema Compliance](https://github.com/anthropics/claude-code/issues/9058)
- [GitHub Issue: JSON Parse Error on Windows](https://github.com/anthropics/claude-code/issues/14442)

---

### 2. Subprocess Timeout Not Killing Child Processes

**Risk:** The current `exec.CommandContext` usage in `internal/agent/executor.go` will kill the Claude CLI process when context times out, but Claude CLI spawns child processes (Node.js workers, language servers) that become orphaned. These zombie processes can:
- Consume resources indefinitely
- Hold locks on files in the workspace
- Prevent cleanup of temporary worktrees
- Leak entries in the process table

The existing code has partial mitigation:
```go
cmd.Cancel = func() error {
    return cmd.Process.Signal(os.Interrupt)
}
cmd.WaitDelay = 10 * time.Second
```

However, [Go Issue #22485](https://github.com/golang/go/issues/22485) documents that `CommandContext` with timeout does not kill subprocesses - only the direct child is killed.

**Warning signs:**
- Increasing memory/CPU usage over time when running C7 repeatedly
- "directory not empty" errors during worktree cleanup
- Orphaned `node` or `claude` processes visible in `ps aux`
- Tests hanging in CI after timeout
- `(*Cmd).Wait` blocks forever waiting for orphaned grandchildren

**Prevention:**
1. Use process groups (PGID) to kill entire process tree on timeout:
   ```go
   import "syscall"

   cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
   // On timeout: syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
   ```
2. Add a cleanup function that explicitly hunts orphaned Claude processes
3. Consider running Claude CLI in a container/sandbox for full isolation
4. Add integration tests that verify no orphaned processes after timeout
5. Document that Docker/container environments need `dumb-init` or similar on PID 1 to reap zombies
6. Use `WaitDelay` > 0 to avoid blocking forever on orphaned subprocess I/O pipes

**Phase impact:** Phase 1 (CLI integration) - Process management must be robust before expanding CLI usage to C4.

**Sources:**
- [Go Issue #22485: CommandContext with timeout with multiple subprocesses isn't canceled](https://github.com/golang/go/issues/22485)
- [Killing a process and all of its descendants in Go](https://sigmoid.at/post/2023/08/kill_process_descendants_golang/)
- [Go exec package documentation](https://pkg.go.dev/os/exec) - WaitDelay field documentation

---

### 3. Import Path Breakage During Analyzer Reorganization

**Risk:** The `internal/analyzer/` directory has 33 files with cross-dependencies. Reorganizing into subdirectories (e.g., `internal/analyzer/go/`, `internal/analyzer/python/`) will break:
- All internal imports across the codebase (90+ Go files)
- The `pipeline/` package which imports analyzers
- Test files that import analyzer types
- The `cmd/scan.go` which wires up analyzers

Go modules do not have a built-in tool for batch import path updates. The `gopls` and `goimports` tools help but are not atomic. Manual changes across 90+ files are error-prone.

Current import structure at risk:
```go
import "github.com/ingo/agent-readyness/internal/analyzer"
```

**Warning signs:**
- "package X is not in GOROOT" or "cannot find package" errors after moving files
- Circular import errors introduced by incorrect reorganization
- Tests fail to compile after reorganization
- `go build ./...` fails with import path errors

**Prevention:**
1. Create the reorganization plan BEFORE moving any files:
   - Map every import relationship (`go list -m -json all`)
   - Identify which types need to be in a shared package
   - Check for circular dependency risks
2. Keep shared interfaces/types in `internal/analyzer/analyzer.go` or `internal/analyzer/types.go`
3. Move files atomically in a single commit with all import updates
4. Use `gopls rename` or IDE refactoring tools with care
5. Run `go build ./...` and `go test ./...` after each logical move
6. Keep the `pipeline.Analyzer` interface in `pipeline/` to avoid circular imports
7. Consider a flat structure with naming conventions instead of subdirectories (e.g., `c1_go.go`, `c1_python.go` - which already exists!)

**Alternative approach:** The existing naming convention (`c1_python.go`, `c1_typescript.go`) already provides language separation without import path changes. Consider keeping this flat structure and using only naming conventions.

**Phase impact:** Phase 2 (code reorganization) - This is the highest-risk phase. Plan extensively before executing.

**Sources:**
- [Go modules documentation](https://go.dev/ref/mod)
- Current codebase structure analysis

---

### 4. Removing Anthropic SDK Breaks C4 LLM Analysis

**Risk:** The milestone states "REMOVING direct Anthropic SDK, switching to Claude Code CLI." However, C4 (documentation quality) in `internal/llm/client.go` uses the Anthropic SDK with features the CLI does not expose:
- **Prompt caching:** `CacheControl: anthropic.NewCacheControlEphemeralParam()` for 90% cost reduction
- **Token tracking:** `message.Usage.InputTokens`, `message.Usage.OutputTokens`
- **Retry logic:** Custom exponential backoff with `isRetryableError()`
- **Model selection:** `anthropic.ModelClaudeHaiku4_5` for cost-effective evaluation

Claude Code CLI does not expose:
- Prompt caching control (no equivalent flag)
- Token usage statistics (not in JSON output)
- Fine-grained retry behavior (CLI handles internally but not configurable)
- Haiku model selection (CLI uses Sonnet by default, `--model` accepts limited options)

Naively replacing SDK calls with CLI invocations will:
- **Increase costs significantly** (no prompt caching = 10x more expensive for repeated evaluations)
- **Lose cost tracking accuracy** (`metrics.LLMTokensUsed` and `metrics.LLMCostUSD` become estimates)
- **Change error handling behavior** (429s handled differently)
- **Lose model flexibility** (can't use Haiku for bulk evaluation)

**Warning signs:**
- C4 LLM costs increase 5-10x after migration
- `metrics.LLMTokensUsed` and `metrics.LLMCostUSD` become zero or inaccurate
- Rate limiting errors without proper backoff
- C4 evaluations become noticeably slower

**Prevention:**
1. **DECISION POINT: Should C4 keep using SDK for evaluation?**
   - Option A: Keep SDK for C4 (cost-sensitive, many small calls), use CLI only for C7 (agent evaluation)
   - Option B: Accept higher costs and reduced observability with CLI
   - Option C: Use CLI with `--max-budget-usd` flag for cost control
2. If keeping SDK for C4: document that `ANTHROPIC_API_KEY` is still required for `--enable-c4-llm`
3. If migrating C4 to CLI:
   - Implement token estimation based on input size (already exists in `c4_documentation.go`: `estimateTokens`)
   - Use `--fallback-model` flag for rate limit resilience
   - Remove prompt caching expectations from cost estimates
   - Use `--max-budget-usd` to prevent cost overruns
4. Document the cost implications clearly for users enabling `--enable-c4-llm`

**Phase impact:** Phase 1 - Decide SDK vs CLI strategy before implementation. This affects architecture.

**Sources:**
- Current `internal/llm/client.go` implementation
- Current `internal/analyzer/c4_documentation.go` LLM integration
- [Claude Code CLI flags documentation](https://code.claude.com/docs/en/cli-reference)

---

## Moderate Pitfalls

These mistakes cause delays or technical debt but are recoverable.

### 5. Shields.io Badge Network Dependency

**Risk:** Generating badges using `https://img.shields.io/badge/...` URLs requires network access at report viewing time. This fails for:
- Air-gapped environments
- Offline documentation viewing
- Badge CDN outages (shields.io processes 1.6B images/month - outages happen)
- Corporate firewalls blocking external resources

**Warning signs:**
- Broken badge images in HTML reports opened offline
- Badge loading delays in slow network environments
- Users complaining about network requests from "local" HTML reports
- Security reviews flagging external resource dependencies

**Prevention:**
1. Use `badge-maker` npm package to generate SVG badges locally at report generation time:
   ```javascript
   const { makeBadge } = require('badge-maker');
   const svg = makeBadge({
     label: 'ARS Score',
     message: '7.5',
     color: 'brightgreen'
   });
   ```
2. Since ARS is a Go tool, either:
   - Embed a pre-built badge template SVG and parameterize it in Go
   - Use `go:embed` to include a small badge-generation library
   - Generate SVG strings directly in Go (SVG is just XML)
3. Embed SVG directly in HTML rather than using `<img src="...">` URLs
4. If using external URLs, provide a fallback `alt` text that shows the score
5. Consider offering both modes: `--badge-mode=embedded` (default) vs `--badge-mode=shields.io`

**Phase impact:** Phase 3 (badge generation) - Design for offline-first from the start.

**Sources:**
- [shields.io GitHub](https://github.com/badges/shields)
- [badge-maker npm package](https://github.com/badges/shields/tree/master/badge-maker)
- [Common Mistakes When Using Shields.io Badges](https://infinitejs.com/posts/common-mistakes-shields-io-badges/)

---

### 6. SVG Badge ID Collision in HTML Reports

**Risk:** When embedding multiple SVG badges inline in the same HTML document, element IDs can collide. SVG gradients, filters, and clip paths use IDs internally. Multiple badges will have:
- `id="g"` for gradients
- `id="s"` for shadows
- CSS cross-contamination between badges

This causes badges to display incorrectly - wrong colors, missing elements, or visual artifacts.

**Warning signs:**
- Badges display with wrong colors
- Some badges appear invisible or clipped incorrectly
- Browser console shows "duplicate ID" warnings
- Second badge always looks like the first badge

**Prevention:**
1. If using badge-maker, use the `idSuffix` parameter to ensure unique IDs per badge:
   ```javascript
   makeBadge({ ..., idSuffix: 'c1-score' })
   ```
2. If generating SVG in Go, post-process to add unique prefixes to all IDs:
   ```go
   svg = strings.ReplaceAll(svg, `id="`, fmt.Sprintf(`id="%s-`, uniqueID))
   svg = strings.ReplaceAll(svg, `url(#`, fmt.Sprintf(`url(#%s-`, uniqueID))
   ```
3. Test HTML reports with ALL 7 category badges visible simultaneously
4. Use CSS `isolation: isolate` on badge containers as defense-in-depth
5. Consider using flat-style badges (no gradients) which have fewer ID-dependent elements

**Phase impact:** Phase 3 (badge generation) - Must implement ID uniqueness from the start.

**Sources:**
- [badge-maker documentation](https://github.com/badges/shields/tree/master/badge-maker) - idSuffix parameter

---

### 7. Inline JavaScript Breaks Content Security Policy

**Risk:** Adding interactive elements to HTML reports (collapsible sections, charts, sorting tables) requires JavaScript. The current HTML report uses inline CSS (`template.CSS`) which works. But inline JavaScript (`<script>` blocks or `onclick` handlers) may be blocked by:
- Browser CSP defaults for `file://` URLs
- Corporate proxy CSP headers
- User browser extensions (NoScript, uBlock Origin)
- Email clients blocking scripts in attachments

**Warning signs:**
- Interactive features work locally but fail when report is shared
- Browser console shows "Refused to execute inline script because it violates the following Content Security Policy directive"
- Features work in Chrome but not Firefox/Safari/Edge
- Reports work in browser but not in email/Slack previews

**Prevention:**
1. **Progressive enhancement:** HTML reports must be useful without JavaScript
   - Tables should be readable without sorting
   - All content should be visible without collapsing
   - Scores and recommendations should be clear in static HTML
2. **CSS-only alternatives:**
   - Use `<details>/<summary>` for collapsible sections (native HTML5, no JS)
   - Use CSS `:target` for navigation
   - Use `<input type="checkbox">` + CSS for toggles
3. **If JavaScript is necessary:**
   - Put all JavaScript in external files and reference via `<script src>`
   - Use nonce-based CSP and include the meta tag:
     ```html
     <meta http-equiv="Content-Security-Policy" content="script-src 'nonce-{random}'">
     ```
   - Use hash-based allowlisting for specific script blocks
4. Test reports in "strict CSP" browser configurations
5. Test reports opened from `file://` URLs

**Phase impact:** Phase 4 (HTML enhancements) - Design interactive features with CSP compatibility in mind.

**Sources:**
- [MDN Content Security Policy](https://developer.mozilla.org/en-US/docs/Web/HTTP/Guides/CSP)
- [OWASP CSP Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Content_Security_Policy_Cheat_Sheet.html)
- [web.dev: Strict CSP](https://web.dev/articles/strict-csp)

---

### 8. Self-Contained HTML Grows Too Large

**Risk:** The HTML report is designed to be self-contained (embedded CSS, embedded charts via `embed.FS`). Adding:
- Inline SVG badges (7 badges x ~2KB = 14KB)
- Interactive JavaScript libraries
- More detailed category breakdowns
- Per-file metric tables

Could push the HTML file from ~50KB to 500KB+, causing:
- Slow loading from disk
- Email attachment size limits exceeded (10MB is common limit)
- Memory pressure when generating reports
- Browser performance issues rendering large DOM

**Warning signs:**
- HTML report generation becomes slow (>1 second)
- Reports fail to attach to emails
- Memory usage spikes during report generation
- Browser tab becomes sluggish when viewing report
- `html.go` template execution takes noticeable time

**Prevention:**
1. **Set a budget:** HTML reports should stay under 200KB
2. Minify embedded CSS/JS (the current `styles.css` is already embedded via `embed.FS`)
3. Use SVG optimization on badges (remove metadata, minimize paths)
4. Consider lazy-loading detail sections via CSS `content-visibility: auto`
5. Offer a "compact" mode without verbose details
6. **Track report size in tests:**
   ```go
   func TestHTMLReportSize(t *testing.T) {
       var buf bytes.Buffer
       generator.GenerateReport(&buf, scored, recs, nil)
       if buf.Len() > 200*1024 {
           t.Errorf("HTML report too large: %d bytes", buf.Len())
       }
   }
   ```
7. For very large repos, consider pagination or summary-only mode

**Phase impact:** Phase 4 (HTML enhancements) - Monitor report size throughout development.

---

### 9. Inconsistent CLI Flag Parsing Between Claude CLI Versions

**Risk:** Different Claude CLI versions may:
- Add new flags (breaking position-based argument parsing)
- Remove deprecated flags
- Change flag behavior silently
- Change short-flag mappings

The current `executor.go` constructs commands as:
```go
args := []string{
    "-p", task.Prompt,
    "--output-format", "json",
}
if task.ToolsAllowed != "" {
    args = append(args, "--allowedTools", task.ToolsAllowed)
}
```

The `-p` flag is short for `--print` but short flags can be ambiguous if new flags are added.

**Warning signs:**
- "unknown flag" errors after user updates Claude CLI
- Arguments interpreted incorrectly (prompt treated as flag value)
- Silent behavioral changes in agent evaluation
- Different behavior between macOS and Linux Claude CLI

**Prevention:**
1. Always use long-form flags (`--print` not `-p`) for clarity and stability
2. Quote prompt arguments to handle special characters
3. Test against minimum supported Claude CLI version in CI
4. Add runtime version detection:
   ```go
   func CheckClaudeCLIVersion() (string, error) {
       out, err := exec.Command("claude", "--version").Output()
       // Parse and validate version
   }
   ```
5. Warn if version is unsupported or untested
6. Handle flag deprecation warnings in stderr parsing

**Phase impact:** Phase 1 - Establish version checking and flag best practices early.

**Sources:**
- [Claude Code CLI reference](https://code.claude.com/docs/en/cli-reference) - flag documentation

---

## Minor Pitfalls

These cause annoyance but are easily fixable.

### 10. Badge Color Inconsistency Across Themes

**Risk:** Badge colors that look good on light backgrounds may be invisible or hard to read on dark backgrounds (GitHub dark mode, VS Code dark theme, terminal dark themes). Default shields.io colors like `brightgreen` (#4c1) have fixed hex values that don't adapt.

**Warning signs:**
- Users report badges are hard to read on dark mode
- Screenshots show badges with poor contrast
- Accessibility tools flag contrast issues (WCAG violations)
- Badge text disappears against certain backgrounds

**Prevention:**
1. Test badge visibility on both light and dark backgrounds
2. Use colors with sufficient contrast ratio (WCAG 2.1 AA requires 4.5:1):
   - Avoid light colors on potentially light backgrounds
   - Consider dual-tone badges with contrasting label/message
3. Test with color blindness simulators (deuteranopia affects red-green perception)
4. Provide `alt` text with numeric score for screen readers
5. Consider using badge styles that work universally:
   - `flat-square` style has better contrast than `plastic`
   - Avoid very light message backgrounds

**Phase impact:** Phase 3 - Test badge visibility across environments.

---

### 11. Worktree Cleanup Race Condition

**Risk:** The `CreateWorkspace` function in `internal/agent/workspace.go` creates a git worktree and returns a cleanup function. If cleanup races with the executor (e.g., executor still writing files when cleanup starts), files may be left behind or cleanup may fail with "directory not empty."

The current test shows this is already a known edge case:
```go
// TestCreateWorkspace_WithGitRepo
if workDir != repoRoot {
    if _, err := os.Stat(workDir); !os.IsNotExist(err) {
        t.Errorf("worktree dir %q still exists after cleanup", workDir)
    }
}
```

**Warning signs:**
- Occasional test failures with "directory not empty"
- Leftover worktree directories in `/tmp`
- Git complains about stale worktrees (`git worktree list` shows orphaned entries)
- Disk space slowly consumed by abandoned worktrees

**Prevention:**
1. Ensure executor fully terminates before cleanup runs (wait for process, not just context cancel)
2. Add a grace period after process termination before cleanup
3. Use `git worktree remove --force` if normal removal fails
4. Add retry logic to cleanup function with exponential backoff
5. Log warnings (not errors) for cleanup failures to avoid masking real issues
6. Periodically clean stale worktrees: `git worktree prune`

**Phase impact:** Phase 1 - Improve cleanup robustness when enhancing CLI usage.

---

### 12. Test Mocking Strategy for CLI Subprocess

**Risk:** The existing tests skip when Claude CLI isn't installed:
```go
if _, err := exec.LookPath("claude"); err != nil {
    t.Skip("claude CLI not installed, skipping availability test")
}
```

This means:
- CI environments without Claude CLI have no test coverage for executor logic
- JSON parsing tests exist but not execution flow tests
- Error paths are not exercised in CI

**Warning signs:**
- High test coverage locally (with Claude CLI), low in CI (without)
- Bugs in executor logic discovered only in production
- Difficulty reproducing user-reported issues
- Changes to executor code have no test feedback in CI

**Prevention:**
1. Implement re-exec testing pattern for subprocess testing:
   ```go
   // In test, spawn test binary itself with special env var
   if os.Getenv("TEST_MOCK_CLAUDE") == "1" {
       // Act as mock claude CLI
       fmt.Println(`{"type":"result","session_id":"test","result":"mocked"}`)
       os.Exit(0)
   }
   ```
2. Create a mock `claude` shell script for testing:
   ```bash
   #!/bin/bash
   echo '{"type":"result","session_id":"mock","result":"test response"}'
   ```
3. Use environment variable to inject mock behavior: `CLAUDE_CLI_PATH`
4. At minimum, test all error paths with mocked commands
5. Consider using `httptest.Server` for SDK-based tests if keeping SDK for C4

**Phase impact:** Phase 1 - Establish testable patterns before expanding CLI usage.

**Sources:**
- [Re-exec testing Go subprocesses](https://rednafi.com/go/test_subprocesses/)

---

### 13. Breaking User Scripts That Parse JSON Output

**Risk:** If ARS JSON output format (`--output json`) changes during this milestone (new fields, reorganized structure, removed fields), user scripts that parse the output will break. The JSON output is effectively a public API.

**Warning signs:**
- GitHub issues/complaints about broken CI pipelines after upgrade
- User scripts fail with JSON parsing errors
- Grep-based scripts stop working due to format changes

**Prevention:**
1. Treat JSON output as a public API contract
2. Document JSON output schema as part of release notes
3. **Additive changes only:** Add new fields without removing existing ones
4. Version the JSON schema in the output itself:
   ```json
   {"version": "2", "composite": 7.5, ...}
   ```
5. Maintain backward compatibility for at least one major version
6. Run integration tests against example user scripts
7. Provide a `--output-format-version` flag to request specific schema version

**Phase impact:** Phase 2 (if output structure changes) - Treat JSON output as public API.

---

## Integration Pitfalls (Cross-Cutting)

### 14. Circular Dependency Between Phases

**Risk:** Phase dependencies create potential deadlocks:
- Phase 2 (reorganization) may want CLI abstractions from Phase 1
- Phase 3 (badges) needs scoring types that may move in Phase 2
- Phase 4 (HTML) needs badge generation from Phase 3

If phases aren't cleanly separated, merge conflicts and rework occur.

**Warning signs:**
- Phases can't be developed independently
- Merge conflicts between feature branches
- "I need to undo Phase 2 changes to finish Phase 1"
- Features from later phases needed to complete earlier phases

**Prevention:**
1. Define stable interfaces BEFORE phases begin
2. Phase 1 completes fully and merges before Phase 2 starts
3. Create adapter/wrapper types to insulate between phases
4. Use feature flags to enable new code paths incrementally
5. Keep phase scope small and focused (don't let scope creep across phases)

**Phase impact:** All phases - Plan phase boundaries carefully.

---

### 15. SDK Removal Breaks Existing User Workflows

**Risk:** Users currently using `--enable-c4-llm` with `ANTHROPIC_API_KEY` may find this breaks if SDK is removed without migration path. The CLI uses different authentication:
- SDK: `ANTHROPIC_API_KEY` environment variable
- CLI: OAuth or Claude Max subscription

This creates a breaking change for existing users.

**Warning signs:**
- Documentation says "set ANTHROPIC_API_KEY" but it's no longer used
- Users with API keys can't use LLM features
- Different authentication requirements between C4 and C7
- Users complain features worked in v0.0.2 but not v0.0.3

**Prevention:**
1. Document authentication requirements clearly in release notes
2. Support BOTH SDK and CLI as backends during transition period
3. Provide migration guide: "If using API key, do X; if using Claude CLI, do Y"
4. Consider keeping SDK as optional dependency for users who prefer it
5. Emit deprecation warnings before removing SDK support entirely:
   ```
   Warning: ANTHROPIC_API_KEY-based LLM analysis is deprecated.
   Please install Claude CLI for LLM features. See: https://...
   ```

**Phase impact:** Phase 1 - Decide on authentication/SDK strategy before implementation.

---

## Phase-Specific Risk Summary

| Phase | Likely Pitfall | Mitigation |
|-------|---------------|------------|
| Phase 1: CLI Integration | JSON schema changes (#1), orphaned processes (#2), SDK removal breaks C4 (#4), flag inconsistency (#9) | Version checking, process groups, decide SDK strategy, use long-form flags |
| Phase 2: Reorganization | Import breakage (#3), circular dependencies (#14), breaking JSON output (#13) | Plan imports before moving, use automation tools, version JSON schema |
| Phase 3: Badges | Network dependency (#5), ID collision (#6), color contrast (#10) | Offline-first design, unique IDs, contrast testing |
| Phase 4: HTML | CSP blocking JS (#7), report size (#8) | Progressive enhancement, CSS-only features, size budget |

---

## Verification Checklist

Before completing each phase, verify:

### Phase 1: CLI Integration
- [ ] Claude CLI version is detected and validated
- [ ] Process groups are used for proper timeout handling
- [ ] No orphaned processes after timeout (verified with `ps aux`)
- [ ] C4 LLM functionality decision is documented (SDK vs CLI)
- [ ] Long-form CLI flags used exclusively
- [ ] Mock testing strategy implemented for CI

### Phase 2: Reorganization
- [ ] `go build ./...` passes after all moves
- [ ] `go test ./...` passes after all moves
- [ ] No circular imports introduced
- [ ] JSON output schema is versioned
- [ ] All import paths updated atomically

### Phase 3: Badges
- [ ] Badges work offline (no external URLs required)
- [ ] SVG IDs are unique across multiple badges
- [ ] Badges readable on both light and dark backgrounds
- [ ] Badge generation adds <5KB to report size

### Phase 4: HTML Enhancements
- [ ] Report is useful without JavaScript enabled
- [ ] Report works when opened from `file://` URL
- [ ] Report size stays under 200KB for typical repos
- [ ] CSP meta tag included if inline scripts used
- [ ] All interactive features have CSS-only fallbacks

---

## Sources

### Official Documentation
- [Go exec package](https://pkg.go.dev/os/exec) - Subprocess management, WaitDelay, process groups
- [Claude Code CLI reference](https://code.claude.com/docs/en/cli-reference) - CLI flags and output formats
- [MDN Content Security Policy](https://developer.mozilla.org/en-US/docs/Web/HTTP/Guides/CSP) - CSP for HTML

### GitHub Issues
- [Go Issue #22485](https://github.com/golang/go/issues/22485) - CommandContext subprocess timeout
- [Claude Code JSON Schema Compliance](https://github.com/anthropics/claude-code/issues/9058)
- [Claude Code JSON Parse Error](https://github.com/anthropics/claude-code/issues/14442)
- [shields.io GitHub](https://github.com/badges/shields) - Badge generation

### Libraries and Tools
- [badge-maker](https://github.com/badges/shields/tree/master/badge-maker) - Offline badge generation
- [shields.io Static Badges](https://shields.io/docs/static-badges)

### Community Resources
- [Killing process descendants in Go](https://sigmoid.at/post/2023/08/kill_process_descendants_golang/)
- [Re-exec testing Go subprocesses](https://rednafi.com/go/test_subprocesses/)
- [Running External Programs in Go](https://medium.com/@caring_smitten_gerbil_914/running-external-programs-in-go-the-right-way-38b11d272cd1)
- [Command PATH security in Go](https://go.dev/blog/path-security)
- [Common Mistakes When Using Shields.io Badges](https://infinitejs.com/posts/common-mistakes-shields-io-badges/)
- [OWASP CSP Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Content_Security_Policy_Cheat_Sheet.html)
- [web.dev: Strict CSP](https://web.dev/articles/strict-csp)
