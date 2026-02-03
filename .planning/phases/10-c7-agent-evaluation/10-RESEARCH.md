# Phase 10: C7 Agent Evaluation - Research

**Researched:** 2026-02-03
**Domain:** Headless Claude Code integration, subprocess management, agent evaluation scoring
**Confidence:** MEDIUM (CLI behavior verified from official docs, subprocess patterns from Go stdlib)

## Summary

Phase 10 implements genuine agent-in-the-loop evaluation by invoking Claude Code CLI in headless mode (`claude -p`) as a subprocess. The agent executes standardized tasks against the user's codebase, and its responses are scored against rubrics to produce C7 metrics measuring intent clarity, modification confidence, cross-file coherence, and semantic completeness.

The research identified three key technical domains:
1. **Claude Code CLI headless invocation** - Using `-p` flag with `--output-format json` for programmatic responses
2. **Go subprocess management** - Using `exec.CommandContext` with Go 1.20+ `Cancel` and `WaitDelay` for graceful timeout handling
3. **Rubric-based scoring** - LLM-as-a-judge pattern using the existing Anthropic SDK client to score agent outputs

**Primary recommendation:** Invoke Claude Code via `exec.CommandContext` with 5-minute timeout per task, capture JSON output, then use the existing LLM client to score agent responses against task-specific rubrics.

## Standard Stack

The established libraries/tools for this domain:

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `os/exec` | Go stdlib | Subprocess invocation | Native Go support, Go 1.20+ has graceful cancellation |
| `context` | Go stdlib | Timeout and cancellation | Standard Go context pattern |
| `claude` CLI | latest | Headless agent execution | Official Anthropic tool, same tools as Claude Code |
| `anthropic-sdk-go` | existing | Rubric scoring via LLM | Already used in C4 for LLM-as-a-judge |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `encoding/json` | Go stdlib | Parse CLI JSON output | Extract result, session_id from responses |
| `os` | Go stdlib | Git worktree/temp dir | Create isolated workspace for agent |
| `time` | Go stdlib | Timeout configuration | 5-minute per-task timeout |
| `sort` | Go stdlib | Stratified sampling | Select functions by complexity buckets |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| CLI subprocess | Agent SDK (Python/TS) | SDK more powerful but adds language dependency; CLI is Go-native |
| Git worktree | Shallow clone | Worktree faster for local repos, clone better for remote |
| LLM scoring | Deterministic regex | LLM more accurate for nuanced rubrics, but adds cost |

**Installation:**
```bash
# Claude Code CLI (required on user's machine)
curl -fsSL https://claude.ai/install.sh | bash
# Or: brew install --cask claude-code
# Or: npm install -g @anthropic-ai/claude-code

# No additional Go dependencies needed - all stdlib + existing SDK
```

## Architecture Patterns

### Recommended Project Structure
```
internal/
├── agent/                    # C7 agent evaluation package
│   ├── executor.go           # Claude CLI subprocess management
│   ├── tasks.go              # Task definitions and prompts
│   ├── scorer.go             # Rubric-based LLM scoring
│   ├── sampler.go            # Stratified function sampling
│   └── types.go              # C7-specific types
├── llm/                      # Existing LLM client (reuse)
│   └── client.go             # Anthropic SDK wrapper
└── analyzer/
    └── c7_agent.go           # C7Analyzer integrating agent evaluation
```

### Pattern 1: Subprocess Invocation with Graceful Timeout
**What:** Use `exec.CommandContext` with custom `Cancel` function for graceful termination
**When to use:** Any long-running subprocess that needs timeout control
**Example:**
```go
// Source: https://pkg.go.dev/os/exec (Go 1.20+)
func executeTask(ctx context.Context, prompt string, workDir string) (*TaskResult, error) {
    // 5-minute timeout per task
    ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
    defer cancel()

    cmd := exec.CommandContext(ctx, "claude", "-p", prompt,
        "--output-format", "json",
        "--allowedTools", "Read,Glob,Grep,Bash",
    )
    cmd.Dir = workDir

    // Graceful cancellation: send SIGINT first, then kill after WaitDelay
    cmd.Cancel = func() error {
        return cmd.Process.Signal(os.Interrupt)
    }
    cmd.WaitDelay = 10 * time.Second // Grace period after SIGINT

    output, err := cmd.Output()
    if err != nil {
        if ctx.Err() == context.DeadlineExceeded {
            return &TaskResult{Status: "timeout"}, nil
        }
        return &TaskResult{Status: "error", Error: err.Error()}, nil
    }

    return parseJSONOutput(output)
}
```

### Pattern 2: JSON Output Parsing
**What:** Parse Claude CLI JSON response to extract result and metadata
**When to use:** Processing headless mode output
**Example:**
```go
// Source: https://code.claude.com/docs/en/headless
type CLIResponse struct {
    Type           string `json:"type"`      // "result"
    SessionID      string `json:"session_id"`
    Result         string `json:"result"`    // Agent's text response
    StructuredOutput json.RawMessage `json:"structured_output,omitempty"`
}

func parseJSONOutput(output []byte) (*TaskResult, error) {
    var resp CLIResponse
    if err := json.Unmarshal(output, &resp); err != nil {
        return nil, fmt.Errorf("parse CLI output: %w", err)
    }
    return &TaskResult{
        Status:    "completed",
        Response:  resp.Result,
        SessionID: resp.SessionID,
    }, nil
}
```

### Pattern 3: LLM-as-a-Judge Rubric Scoring
**What:** Use LLM to score agent responses against task-specific rubrics
**When to use:** Evaluating nuanced quality that can't be measured deterministically
**Example:**
```go
// Source: https://docs.ragas.io/en/latest/concepts/metrics/available_metrics/rubrics_based/
// Pattern adapted from existing C4 LLM client usage

const IntentClarityRubric = `You are evaluating an AI agent's response to a code understanding task.

Task: The agent was asked to find a function and explain what it does.

Score the response from 0-100 based on these criteria:
- Correct identification (40%): Did the agent find the right function?
- Accuracy of explanation (40%): Is the explanation correct and clear?
- Use of codebase context (20%): Did the agent reference related code appropriately?

Respond with JSON: {"score": N, "reason": "brief explanation"}`

func scoreResponse(ctx context.Context, llm *llm.Client, task *Task, response string) (int, string, error) {
    content := fmt.Sprintf("Task prompt: %s\n\nAgent response:\n%s", task.Prompt, response)
    eval, err := llm.EvaluateContent(ctx, task.Rubric, content)
    if err != nil {
        return 0, "", err
    }
    // Scale from 1-10 to 0-100
    return eval.Score * 10, eval.Reasoning, nil
}
```

### Pattern 4: Stratified Function Sampling
**What:** Select representative functions across complexity buckets for evaluation
**When to use:** Large codebases where evaluating all functions is impractical
**Example:**
```go
// Reuse complexity data from C1 analyzer
func sampleFunctions(functions []types.FunctionMetric, targetCount int) []types.FunctionMetric {
    if len(functions) <= targetCount {
        return functions
    }

    // Sort by complexity
    sort.Slice(functions, func(i, j int) bool {
        return functions[i].Complexity < functions[j].Complexity
    })

    // Stratify into buckets: low (1-3), medium (4-10), high (11+)
    var low, medium, high []types.FunctionMetric
    for _, f := range functions {
        switch {
        case f.Complexity <= 3:
            low = append(low, f)
        case f.Complexity <= 10:
            medium = append(medium, f)
        default:
            high = append(high, f)
        }
    }

    // Sample proportionally from each bucket
    result := make([]types.FunctionMetric, 0, targetCount)
    buckets := [][]types.FunctionMetric{low, medium, high}
    perBucket := targetCount / 3

    for _, bucket := range buckets {
        if len(bucket) <= perBucket {
            result = append(result, bucket...)
        } else {
            // Random sample from bucket
            step := len(bucket) / perBucket
            for i := 0; i < perBucket && i*step < len(bucket); i++ {
                result = append(result, bucket[i*step])
            }
        }
    }

    return result
}
```

### Pattern 5: Isolated Workspace Creation
**What:** Create isolated directory for agent writes without affecting user's codebase
**When to use:** Tasks that involve modification attempts
**Example:**
```go
// Source: https://git-scm.com/docs/git-worktree
func createIsolatedWorkspace(projectDir string) (string, func(), error) {
    // Option 1: Git worktree (faster, shares objects)
    worktreeDir, err := os.MkdirTemp("", "ars-c7-*")
    if err != nil {
        return "", nil, err
    }

    cmd := exec.Command("git", "worktree", "add", worktreeDir, "HEAD", "--detach")
    cmd.Dir = projectDir
    if err := cmd.Run(); err != nil {
        // Fall back to temp copy if not a git repo
        os.RemoveAll(worktreeDir)
        return createTempCopy(projectDir)
    }

    cleanup := func() {
        exec.Command("git", "worktree", "remove", worktreeDir, "--force").Run()
        os.RemoveAll(worktreeDir)
    }

    return worktreeDir, cleanup, nil
}
```

### Anti-Patterns to Avoid
- **Running agent in user's actual directory:** Always use isolated workspace to prevent unintended modifications
- **Parallel task execution:** Sequential execution avoids state conflicts and context leakage
- **Hard kill on timeout:** Use graceful SIGINT first, then SIGKILL after grace period
- **Ignoring CLI exit codes:** Check both exit code and JSON response for accurate error detection
- **Unbounded token usage:** Set reasonable prompt limits and truncate large files

## Don't Hand-Roll

Problems that look simple but have existing solutions:

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| CLI subprocess | Custom process fork | `exec.CommandContext` | Go stdlib handles signals, I/O, and timeouts correctly |
| Timeout handling | Manual timer goroutine | `context.WithTimeout` + `cmd.Cancel` | Go 1.20+ provides graceful cancellation |
| LLM scoring | Regex/keyword matching | Existing `llm.Client.EvaluateContent` | Already battle-tested in C4, handles retries |
| Function complexity | Re-analyze AST | C1 analyzer `FunctionMetric` data | Already computed and available in pipeline |
| JSON parsing | String manipulation | `encoding/json.Unmarshal` | Type-safe, handles edge cases |
| Git worktree | Shallow clone | `git worktree add` | Faster, shares objects, no network |

**Key insight:** The existing ARS infrastructure already provides function metrics (C1), LLM client (C4), and cost estimation patterns. C7 should compose these rather than rebuild them.

## Common Pitfalls

### Pitfall 1: CLI Not Found at Runtime
**What goes wrong:** User runs `ars scan --enable-c7` but doesn't have `claude` CLI installed
**Why it happens:** CLI is optional dependency, not bundled with ARS
**How to avoid:** Check for CLI at start, return clear error before any work begins
**Warning signs:** Exit code 127 (command not found) or error containing "not found"
```go
func checkClaudeCLI() error {
    _, err := exec.LookPath("claude")
    if err != nil {
        return fmt.Errorf("claude CLI not found. Install from: https://claude.ai/install.sh")
    }
    return nil
}
```

### Pitfall 2: Timeout Orphan Processes
**What goes wrong:** Timeout kills parent process but child subprocesses keep running
**Why it happens:** Go 1.20 default kills only the direct process, not process group
**How to avoid:** Use `cmd.Cancel` with SIGINT and `cmd.WaitDelay` for graceful shutdown
**Warning signs:** Zombie processes, resource leaks after timeout

### Pitfall 3: Non-JSON Output on Errors
**What goes wrong:** `json.Unmarshal` fails because CLI returned plain text error
**Why it happens:** CLI may output errors to stderr or as plain text even with `--output-format json`
**How to avoid:** Capture both stdout and stderr, check exit code first, handle non-JSON gracefully
**Warning signs:** "invalid character" in JSON parse errors

### Pitfall 4: ANTHROPIC_API_KEY Confusion
**What goes wrong:** C7 agent uses user's API key directly (if they have one set), but ARS expects to use its own calls
**Why it happens:** Claude CLI reads `ANTHROPIC_API_KEY` from environment
**How to avoid:** Document that C7 uses the user's API key via Claude CLI, not ARS's LLM client
**Warning signs:** Unexpected costs on user's API key

### Pitfall 5: Large Codebase Task Explosion
**What goes wrong:** Agent takes forever or times out on large files
**Why it happens:** No limits on content size in task prompts
**How to avoid:** Truncate large files, sample functions, set explicit file size limits in prompts
**Warning signs:** Tasks consistently timing out, high token costs

### Pitfall 6: Modification Tasks on Read-Only Workspace
**What goes wrong:** Agent fails modification tasks because worktree is incomplete
**Why it happens:** Git worktree doesn't include untracked files
**How to avoid:** Use git stash or copy untracked files, or limit modification tasks to tracked files only
**Warning signs:** Agent can't find files that exist in main repo

## Code Examples

Verified patterns from official sources:

### Claude CLI Headless Invocation
```go
// Source: https://code.claude.com/docs/en/headless
// Basic headless invocation with JSON output
cmd := exec.CommandContext(ctx, "claude",
    "-p", "What does the auth module do?",
    "--output-format", "json",
)
output, err := cmd.Output()
```

### Claude CLI with Tool Permissions
```go
// Source: https://code.claude.com/docs/en/headless
// Allow specific tools without prompting
cmd := exec.CommandContext(ctx, "claude",
    "-p", "Find and fix the bug in auth.py",
    "--allowedTools", "Read,Edit,Bash",
    "--output-format", "json",
)
```

### Go 1.20+ Graceful Cancellation
```go
// Source: https://pkg.go.dev/os/exec
cmd := exec.CommandContext(ctx, "claude", "-p", prompt)

// Send SIGINT on context cancellation instead of SIGKILL
cmd.Cancel = func() error {
    return cmd.Process.Signal(os.Interrupt)
}

// Wait up to 10 seconds for graceful shutdown before force-killing
cmd.WaitDelay = 10 * time.Second
```

### Cost Estimation Pattern
```go
// Adapted from existing llm/cost.go pattern
func EstimateC7Cost(taskCount int) CostEstimate {
    // Claude CLI uses Sonnet by default
    // ~10k tokens per task (prompt + response)
    tokensPerTask := 10000
    totalTokens := taskCount * tokensPerTask

    // Sonnet pricing: $3/MTok input, $15/MTok output
    // Assume 70% input, 30% output
    inputCost := float64(totalTokens) * 0.7 / 1_000_000 * 3.0
    outputCost := float64(totalTokens) * 0.3 / 1_000_000 * 15.0

    return CostEstimate{
        InputTokens:  int(float64(totalTokens) * 0.7),
        OutputTokens: int(float64(totalTokens) * 0.3),
        MinCost:      inputCost + outputCost,
        MaxCost:      (inputCost + outputCost) * 1.5,
    }
}
```

### Rubric Definition Pattern
```go
// Source: https://docs.ragas.io/en/latest/concepts/metrics/available_metrics/rubrics_based/
type TaskRubric struct {
    Name        string
    Description string
    Criteria    map[string]int // criterion name -> weight (total 100)
    Prompt      string         // LLM prompt with scoring instructions
}

var IntentClarityRubric = TaskRubric{
    Name:        "Intent Clarity",
    Description: "Measures agent's ability to understand code purpose",
    Criteria: map[string]int{
        "correct_identification":   40,
        "accuracy_of_explanation": 40,
        "use_of_codebase_context": 20,
    },
    Prompt: `Score the agent's code understanding response (0-100):
- Correct identification (40%): Did the agent find the right function?
- Accuracy of explanation (40%): Is the explanation correct and clear?
- Use of codebase context (20%): Did the agent reference related code?

Respond with JSON: {"score": N, "reason": "brief explanation"}`,
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Custom agent frameworks | Claude Agent SDK / CLI | 2025 | Standardized agentic patterns |
| Manual tool loops | Built-in tool execution | 2025 | No need to implement tool dispatch |
| `os.Process.Kill` | `cmd.Cancel` + `WaitDelay` | Go 1.20 (2023) | Graceful subprocess termination |
| Binary pass/fail | LLM-as-a-judge rubrics | 2024-2025 | Nuanced scoring, better accuracy |

**Deprecated/outdated:**
- Claude Code "headless mode" terminology: Renamed to "Agent SDK CLI" but same `-p` flag works
- Go 1.19 and earlier: Lacks `cmd.Cancel` and `cmd.WaitDelay` for graceful termination

## Open Questions

Things that couldn't be fully resolved:

1. **Exact JSON response schema**
   - What we know: Response includes `type`, `session_id`, `result` fields
   - What's unclear: Full schema with all possible fields, error response format
   - Recommendation: Test with actual CLI to capture sample responses, handle unknown fields gracefully

2. **Claude CLI exit codes**
   - What we know: Exit code 1 reported for various errors, 127 for not found
   - What's unclear: Official exit code documentation, timeout-specific codes
   - Recommendation: Check context.DeadlineExceeded first, treat all non-zero as error, parse stderr for details

3. **Token usage in CLI output**
   - What we know: Interactive mode shows `/cost` command, SDK has usage metrics
   - What's unclear: Whether `--output-format json` includes token counts
   - Recommendation: Estimate costs conservatively, consider adding `--verbose` flag investigation

4. **Agent behavior consistency**
   - What we know: Same prompt can produce different agent actions
   - What's unclear: How deterministic task results will be across runs
   - Recommendation: Focus rubric scoring on quality of final output, not exact steps taken

## Sources

### Primary (HIGH confidence)
- [Claude Code Headless Documentation](https://code.claude.com/docs/en/headless) - CLI flags, output formats, tool permissions
- [Go os/exec Package](https://pkg.go.dev/os/exec) - CommandContext, Cancel, WaitDelay API
- [Claude Agent SDK Overview](https://platform.claude.com/docs/en/agent-sdk/overview) - SDK capabilities, session management

### Secondary (MEDIUM confidence)
- [Git Worktree Documentation](https://git-scm.com/docs/git-worktree) - Isolated workspace creation
- [Ragas Rubric-Based Evaluation](https://docs.ragas.io/en/latest/concepts/metrics/available_metrics/rubrics_based/) - LLM-as-a-judge patterns
- [Shipyard Claude Code Cheatsheet](https://shipyard.build/blog/claude-code-cheat-sheet/) - CLI usage patterns

### Tertiary (LOW confidence)
- [GitHub Issue #5615](https://github.com/anthropics/claude-code/issues/5615) - Timeout configuration (community-verified)
- [GitHub Issue #8557](https://github.com/anthropics/claude-code/issues/8557) - Exit code behavior (user reports)
- WebSearch results for subprocess patterns and agent evaluation - general guidance, needs validation

## Metadata

**Confidence breakdown:**
- Standard stack: MEDIUM - CLI patterns verified in official docs, subprocess patterns from Go stdlib
- Architecture: MEDIUM - Patterns adapted from existing ARS code and official examples
- Pitfalls: MEDIUM - Combination of official docs, community issues, and general subprocess knowledge

**Research date:** 2026-02-03
**Valid until:** 2026-03-03 (30 days - Claude Code CLI may update)
