# Phase 15: Claude Code Integration - Research

**Researched:** 2026-02-03
**Domain:** Claude Code CLI integration, Go process execution, LLM-based analysis migration
**Confidence:** HIGH

## Summary

This phase migrates C4 documentation quality analysis from the Anthropic SDK to Claude Code CLI (`claude -p`), unifies all LLM features under CLI execution, removes the SDK dependency, and implements auto-detection of CLI availability. The existing C7 agent evaluation already uses Claude CLI and serves as the implementation pattern.

The codebase already has a working Claude CLI integration in `internal/agent/executor.go` that handles JSON output parsing, timeout management, and graceful shutdown. The C4 analyzer in `internal/analyzer/c4_documentation.go` uses `internal/llm/client.go` which wraps the Anthropic SDK. The migration requires creating a CLI-based evaluator that matches the existing `llm.Client.EvaluateContent()` interface.

The key implementation approach is to extend the existing `agent.Executor` pattern to support C4's evaluation prompts, replacing direct SDK calls with `claude -p` invocations that return structured JSON responses.

**Primary recommendation:** Extend `internal/agent/executor.go` with a new `EvaluateContent` method that uses `claude -p --output-format json --json-schema` to get structured evaluation responses, then refactor C4 and C7 analyzers to use this unified CLI-based evaluator.

## Standard Stack

The established libraries/tools for this domain:

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Claude Code CLI | 2.1.x+ | LLM execution and evaluation | Official Anthropic tool, handles auth, rate limiting, model selection |
| os/exec (Go stdlib) | Go 1.25 | Process execution | Already used in codebase, well-tested patterns |
| context (Go stdlib) | Go 1.25 | Timeout and cancellation | Standard Go pattern for async operations |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| encoding/json (stdlib) | Go 1.25 | JSON parsing of CLI output | Parse `--output-format json` responses |
| os/signal (stdlib) | Go 1.25 | Graceful shutdown | SIGINT handling for long-running CLI calls |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Claude CLI | Anthropic SDK | SDK requires ANTHROPIC_API_KEY, CLI uses existing auth; CLI is simpler for this use case |
| exec.CommandContext | Third-party process libs | stdlib is sufficient, no external deps needed |

**Installation:**
```bash
# Claude CLI (user must have installed)
curl -fsSL https://claude.ai/install.sh | bash
# or
brew install --cask claude-code
```

## Architecture Patterns

### Recommended Project Structure
```
internal/
├── agent/
│   ├── executor.go      # CLI execution (extend for C4)
│   ├── evaluator.go     # NEW: unified content evaluation via CLI
│   ├── types.go         # Task types (already exists)
│   └── cli.go           # NEW: CLI detection and version checking
├── analyzer/
│   ├── c4_documentation.go  # Refactor to use agent.Evaluator
│   └── c7_agent.go          # Already uses agent.Executor
└── llm/                     # REMOVE: entire package after migration
```

### Pattern 1: CLI Detection and Caching
**What:** Check CLI availability once at startup, cache result for scan duration
**When to use:** At pipeline initialization, before any LLM analysis
**Example:**
```go
// Source: Based on existing agent.CheckClaudeCLI() pattern
type CLIStatus struct {
    Available   bool
    Version     string
    Error       string
    InstallHint string
}

func DetectCLI() CLIStatus {
    path, err := exec.LookPath("claude")
    if err != nil {
        return CLIStatus{
            Available:   false,
            Error:       "claude CLI not found in PATH",
            InstallHint: "Install from: https://claude.ai/install.sh",
        }
    }

    // Verify CLI responds (not just exists)
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    cmd := exec.CommandContext(ctx, path, "--version")
    output, err := cmd.Output()
    if err != nil {
        return CLIStatus{
            Available:   false,
            Error:       fmt.Sprintf("claude CLI failed version check: %v", err),
            InstallHint: "Reinstall from: https://claude.ai/install.sh",
        }
    }

    version := strings.TrimSpace(string(output))
    return CLIStatus{
        Available: true,
        Version:   version,
    }
}
```

### Pattern 2: Unified Content Evaluation via CLI
**What:** Replace SDK-based evaluation with CLI-based execution using JSON schema for structured output
**When to use:** For C4 documentation quality analysis (and potentially C7 scoring)
**Example:**
```go
// Source: Based on CLI docs at https://code.claude.com/docs/en/headless
type Evaluator struct {
    timeout time.Duration
}

type EvaluationResult struct {
    Score  int    `json:"score"`
    Reason string `json:"reason"`
}

func (e *Evaluator) EvaluateContent(ctx context.Context, systemPrompt, content string) (EvaluationResult, error) {
    // JSON schema for structured output
    schema := `{"type":"object","properties":{"score":{"type":"integer","minimum":1,"maximum":10},"reason":{"type":"string"}},"required":["score","reason"]}`

    args := []string{
        "-p", content,
        "--system-prompt", systemPrompt,
        "--output-format", "json",
        "--json-schema", schema,
    }

    taskCtx, cancel := context.WithTimeout(ctx, e.timeout)
    defer cancel()

    cmd := exec.CommandContext(taskCtx, "claude", args...)
    cmd.Cancel = func() error {
        return cmd.Process.Signal(os.Interrupt)
    }
    cmd.WaitDelay = 10 * time.Second

    output, err := cmd.CombinedOutput()
    if err != nil {
        if taskCtx.Err() == context.DeadlineExceeded {
            return EvaluationResult{}, fmt.Errorf("evaluation timed out after %v", e.timeout)
        }
        return EvaluationResult{}, fmt.Errorf("CLI execution failed: %w (stderr: %s)", err, string(output))
    }

    // Parse JSON response
    var resp struct {
        Result          string `json:"result"`
        StructuredOutput EvaluationResult `json:"structured_output"`
    }
    if err := json.Unmarshal(output, &resp); err != nil {
        return EvaluationResult{}, fmt.Errorf("failed to parse CLI response: %w", err)
    }

    return resp.StructuredOutput, nil
}
```

### Pattern 3: Retry with Exponential Backoff
**What:** Retry CLI failures once before graceful degradation
**When to use:** When CLI calls fail due to transient errors
**Example:**
```go
// Source: Existing pattern in llm/client.go, adapted for CLI
func (e *Evaluator) EvaluateWithRetry(ctx context.Context, systemPrompt, content string) (EvaluationResult, error) {
    result, err := e.EvaluateContent(ctx, systemPrompt, content)
    if err == nil {
        return result, nil
    }

    // Single retry after brief delay
    select {
    case <-ctx.Done():
        return EvaluationResult{}, ctx.Err()
    case <-time.After(2 * time.Second):
    }

    return e.EvaluateContent(ctx, systemPrompt, content)
}
```

### Anti-Patterns to Avoid
- **Polling CLI availability repeatedly:** Check once at startup, cache result
- **Blocking on stdin for confirmation:** Auto-detect, no user prompts for CLI features
- **Ignoring exit codes:** CLI exit codes indicate specific failure modes
- **Not setting WaitDelay:** Process may hang if SIGINT ignored without force-kill

## Don't Hand-Roll

Problems that look simple but have existing solutions:

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| CLI output parsing | Custom text parsing | `--output-format json` with `--json-schema` | CLI provides structured JSON, no regex needed |
| Authentication | API key management | Claude CLI's built-in auth | CLI handles OAuth/subscription automatically |
| Rate limiting | Custom backoff | CLI's built-in handling | Claude CLI manages API rate limits internally |
| Model selection | Hardcoded model IDs | CLI's model aliases (`sonnet`, `opus`) | CLI keeps models current |

**Key insight:** The Claude CLI abstracts away SDK complexities (auth, rate limits, model versions). Using the CLI simplifies the codebase significantly.

## Common Pitfalls

### Pitfall 1: Large Input Truncation
**What goes wrong:** Claude CLI may return empty output with very large stdin input (~7000+ characters)
**Why it happens:** Known CLI bug with large inputs in headless mode
**How to avoid:** Truncate content before sending (existing code already does this: 20000 chars for README, 10000 for examples)
**Warning signs:** Empty response from CLI despite valid input

### Pitfall 2: JSON Schema Validation Failures
**What goes wrong:** CLI returns error if LLM response doesn't match schema
**Why it happens:** LLM may not strictly follow schema, especially edge cases
**How to avoid:** Use simple schemas, provide clear instructions in system prompt, parse `result` field as fallback
**Warning signs:** Parse errors mentioning "structured_output"

### Pitfall 3: Timeout Without Graceful Shutdown
**What goes wrong:** Process killed without cleanup, may leave orphan processes
**Why it happens:** Using Kill instead of SIGINT, no WaitDelay
**How to avoid:** Set `cmd.Cancel` to send SIGINT, set `cmd.WaitDelay` for grace period (existing executor.go does this correctly)
**Warning signs:** Zombie processes, resource leaks

### Pitfall 4: Not Checking CLI Version
**What goes wrong:** Old CLI versions may not support required flags (`--json-schema`)
**Why it happens:** User hasn't updated CLI
**How to avoid:** Run `claude --version` and warn if version is too old (2.1.x+ required for `--json-schema`)
**Warning signs:** Unknown flag errors from CLI

### Pitfall 5: Auth Conflict Detection
**What goes wrong:** Unexpected behavior if both ANTHROPIC_API_KEY and OAuth token set
**Why it happens:** CLI prioritizes API key over OAuth
**How to avoid:** Don't set ANTHROPIC_API_KEY when using CLI; ignore it if set (per context decisions)
**Warning signs:** Unexpected API charges, different model behavior

## Code Examples

Verified patterns from official sources:

### CLI Detection with Version Check
```go
// Source: https://code.claude.com/docs/en/cli-reference
func DetectAndValidateCLI() (CLIStatus, error) {
    // Check PATH
    path, err := exec.LookPath("claude")
    if err != nil {
        return CLIStatus{
            Available:   false,
            InstallHint: "Install Claude Code CLI:\n  curl -fsSL https://claude.ai/install.sh | bash\n  or: brew install --cask claude-code",
        }, nil
    }

    // Get version
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    cmd := exec.CommandContext(ctx, path, "--version")
    output, err := cmd.Output()
    if err != nil {
        return CLIStatus{
            Available: false,
            Error:     fmt.Sprintf("version check failed: %v", err),
        }, nil
    }

    version := strings.TrimSpace(string(output))
    // Version format: "claude 2.1.12" or similar
    return CLIStatus{
        Available: true,
        Version:   version,
    }, nil
}
```

### Headless Evaluation with JSON Schema
```go
// Source: https://code.claude.com/docs/en/headless
func evaluateWithSchema(ctx context.Context, systemPrompt, content string, timeout time.Duration) (EvaluationResult, error) {
    schema := `{"type":"object","properties":{"score":{"type":"integer"},"reason":{"type":"string"}},"required":["score","reason"]}`

    args := []string{
        "-p", content,
        "--system-prompt", systemPrompt,
        "--output-format", "json",
        "--json-schema", schema,
    }

    taskCtx, cancel := context.WithTimeout(ctx, timeout)
    defer cancel()

    cmd := exec.CommandContext(taskCtx, "claude", args...)
    cmd.Cancel = func() error {
        return cmd.Process.Signal(os.Interrupt)
    }
    cmd.WaitDelay = 10 * time.Second

    output, err := cmd.CombinedOutput()
    if err != nil {
        if taskCtx.Err() == context.DeadlineExceeded {
            return EvaluationResult{}, fmt.Errorf("timeout after %v", timeout)
        }
        return EvaluationResult{}, fmt.Errorf("execution failed (exit: %v): %s", err, truncate(string(output), 200))
    }

    var response struct {
        SessionID        string           `json:"session_id"`
        Result           string           `json:"result"`
        StructuredOutput EvaluationResult `json:"structured_output"`
    }

    if err := json.Unmarshal(output, &response); err != nil {
        return EvaluationResult{}, fmt.Errorf("JSON parse failed: %w (got: %s)", err, truncate(string(output), 100))
    }

    if response.StructuredOutput.Score < 1 || response.StructuredOutput.Score > 10 {
        return EvaluationResult{}, fmt.Errorf("score out of range: %d", response.StructuredOutput.Score)
    }

    return response.StructuredOutput, nil
}
```

### Flag Removal Pattern
```go
// Source: Cobra documentation - unknown flags cause error by default
// When removing --enable-c4-llm flag, users get:
// "Error: unknown flag: --enable-c4-llm"
// This is the desired behavior per context decisions

// Old code to remove from cmd/scan.go:
// var enableC4LLM bool
// scanCmd.Flags().BoolVar(&enableC4LLM, "enable-c4-llm", false, "...")

// Add new --no-llm flag instead:
var noLLM bool
scanCmd.Flags().BoolVar(&noLLM, "no-llm", false, "disable LLM features even when Claude CLI is available")
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Anthropic SDK for LLM calls | Claude CLI `claude -p` | Phase 15 | Simpler auth, no API key needed |
| `--enable-c4-llm` flag | Auto-detect CLI availability | Phase 15 | Better UX, features enabled automatically |
| ANTHROPIC_API_KEY required | CLI handles auth via OAuth | Phase 15 | No env vars needed, uses subscription |
| SDK prompt caching | CLI handles caching internally | Phase 15 | Simplified code |

**Deprecated/outdated:**
- `--enable-c4-llm` flag: Removed entirely
- `ANTHROPIC_API_KEY` requirement: No longer needed (CLI uses subscription auth)
- `internal/llm/client.go`: Entire package removed after migration
- `github.com/anthropics/anthropic-sdk-go` dependency: Removed from go.mod

## Open Questions

Things that couldn't be fully resolved:

1. **Minimum CLI version for `--json-schema`**
   - What we know: `--json-schema` flag exists in current CLI (2.1.x)
   - What's unclear: Exact minimum version where this flag was introduced
   - Recommendation: Check for flag support at runtime by testing `--help` output or catching error

2. **CLI auth state detection**
   - What we know: CLI returns errors if not authenticated
   - What's unclear: Exact error message/code for unauthenticated state
   - Recommendation: Run `claude --version` as smoke test; if that fails, CLI is likely not configured

3. **Exact JSON output format for `--json-schema` failures**
   - What we know: Schema validation failures return errors
   - What's unclear: Exact error format when LLM output doesn't match schema
   - Recommendation: Implement fallback to parse `result` field as text if `structured_output` missing

## Sources

### Primary (HIGH confidence)
- [Claude Code CLI Reference](https://code.claude.com/docs/en/cli-reference) - All CLI flags and options
- [Claude Code Headless Mode](https://code.claude.com/docs/en/headless) - Programmatic usage with `-p`
- [Claude Code Setup](https://code.claude.com/docs/en/setup) - Installation and authentication
- Existing codebase: `internal/agent/executor.go` - Working CLI integration pattern

### Secondary (MEDIUM confidence)
- [Go Graceful Shutdown Patterns](https://victoriametrics.com/blog/go-graceful-shutdown/) - Signal handling best practices
- [Claude Code GitHub Issues](https://github.com/anthropics/claude-code/issues) - Known bugs and limitations

### Tertiary (LOW confidence)
- WebSearch results for CLI version requirements - Version numbering not officially documented

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - Official Claude CLI docs, existing codebase patterns
- Architecture: HIGH - Clear migration path from existing executor.go to unified evaluator
- Pitfalls: MEDIUM - Some based on GitHub issues, others from codebase analysis

**Research date:** 2026-02-03
**Valid until:** 2026-03-03 (CLI updates frequently, check for breaking changes)
