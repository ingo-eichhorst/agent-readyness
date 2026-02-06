# Phase 26: Debug Foundation - Research

**Researched:** 2026-02-06
**Domain:** CLI flag plumbing, debug output routing, Go `io.Writer` patterns
**Confidence:** HIGH

## Summary

Phase 26 adds a single `--debug-c7` CLI flag that (a) activates C7 evaluation automatically (DBG-02), (b) routes all future debug output exclusively to stderr (DBG-03), and (c) wires the debug state through the existing pipeline without changing any existing behavior when the flag is absent (DBG-01). This is pure plumbing -- no debug content is emitted yet (that is Phase 27-29).

The research reveals that the existing codebase already has every pattern needed. The `--enable-c7` flag in `cmd/scan.go` demonstrates the exact flag declaration, registration, and pipeline threading pattern. The `Pipeline.SetC7Enabled()` method shows how to pass boolean state from CLI to analyzer. The `Spinner` in `pipeline/progress.go` and `C7Progress` in `agent/progress.go` both demonstrate stderr-only diagnostic output with `isatty` TTY detection. The `io.Discard` stdlib type provides zero-cost disabling.

The primary risk is debug output accidentally reaching stdout and corrupting `--json` mode. The mitigation is architectural: establish a `debugWriter io.Writer` field initialized to `io.Discard` (normal mode) or `os.Stderr` (debug mode), and use this writer exclusively for all debug output in later phases.

**Primary recommendation:** Follow the exact `--enable-c7` pattern for flag plumbing. Add a `debugWriter io.Writer` field to `Pipeline` initialized to `io.Discard` or `os.Stderr` based on the flag. Thread through to `C7Analyzer`. Verify with the success criterion: `ars scan . --debug-c7 --json 2>/dev/null | jq` produces valid JSON.

## Standard Stack

The established libraries/tools for this domain:

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `spf13/cobra` | v1.10.2 | CLI flag registration (`BoolVar`) | Already used for all 8 existing scan flags |
| `io.Discard` | stdlib | Zero-cost no-op writer when debug disabled | Standard Go pattern for conditional output; zero allocations |
| `os.Stderr` | stdlib | Debug output destination | Unix convention: diagnostic output on stderr, data on stdout |
| `mattn/go-isatty` | v0.0.20 | TTY detection for debug writer | Already used by `Spinner` and `C7Progress` for the same purpose |
| `fmt.Fprintf` | stdlib | Writing debug output to `io.Writer` | Consistent with existing `Pipeline.writer` pattern throughout codebase |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `testing` | stdlib | Unit tests for flag behavior | Verify flag threading and auto-enable logic |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `io.Discard` | `log/slog` with disabled handler | slog adds structured logging paradigm inconsistent with existing `fmt.Fprintf` pattern; out of scope per requirements |
| Package-level `debugC7 bool` | Global `var` or env var | Global state makes testing hard; existing pattern threads state through method calls |
| New `DebugWriter` wrapper type | Bare `io.Writer` field | Wrapper adds abstraction for no gain; `io.Writer` interface is sufficient; future phases can wrap if needed |

**Installation:**
```bash
# No new dependencies. Zero go.mod changes.
```

## Architecture Patterns

### Recommended Change Topology

```
cmd/scan.go              (add debugC7 var + flag registration + auto-enable + pipeline call)
  |
  v
internal/pipeline/pipeline.go    (add debugC7 bool + debugWriter io.Writer fields + SetC7Debug method)
  |
  v
internal/analyzer/c7_agent/agent.go  (add debug bool field + SetDebug method -- consumed in later phases)
```

### Pattern 1: Flag Declaration and Registration (follow --enable-c7 exactly)

**What:** Declare package-level `var`, register with cobra `BoolVar` in `init()`, consume in `RunE` closure
**When to use:** Every time a new CLI flag is added
**Source:** Verified from `cmd/scan.go` lines 15-24 (declaration), 124-133 (registration), 89-96 (consumption)

**Example:**
```go
// cmd/scan.go -- declaration (alongside existing vars, line ~21)
var (
    // ... existing vars ...
    debugC7  bool   // Enable C7 debug mode
)

// cmd/scan.go -- registration (in init(), line ~133)
scanCmd.Flags().BoolVar(&debugC7, "debug-c7", false,
    "enable C7 debug mode (implies --enable-c7; debug output on stderr)")

// cmd/scan.go -- consumption (in RunE, after C7 handling block)
if debugC7 {
    enableC7 = true  // DBG-02: auto-enable C7
}
// Then the existing enableC7 block (lines 90-96) handles SetC7Enabled()

// After C7 is enabled, set debug mode
if debugC7 {
    p.SetC7Debug(true)
}
```

### Pattern 2: Pipeline State Threading (follow SetC7Enabled exactly)

**What:** Add a method on `Pipeline` that stores config, passes to relevant analyzer
**When to use:** When CLI state needs to reach an analyzer
**Source:** Verified from `pipeline.go` lines 119-124 (`SetC7Enabled`)

**Example:**
```go
// internal/pipeline/pipeline.go -- new fields on Pipeline struct
type Pipeline struct {
    // ... existing fields ...
    debugC7     bool       // C7 debug mode enabled
    debugWriter io.Writer  // io.Discard (normal) or os.Stderr (debug)
}

// internal/pipeline/pipeline.go -- in New(), initialize debugWriter
func New(...) *Pipeline {
    return &Pipeline{
        // ... existing fields ...
        debugWriter: io.Discard,  // default: zero-cost no-op
    }
}

// internal/pipeline/pipeline.go -- new method
func (p *Pipeline) SetC7Debug(enabled bool) {
    p.debugC7 = enabled
    if enabled {
        p.debugWriter = os.Stderr
    }
    // Also enable C7 if not already
    p.SetC7Enabled()
}
```

### Pattern 3: Analyzer Debug State (follow C7Analyzer.Enable exactly)

**What:** Add a method on the analyzer to receive debug configuration
**When to use:** When analyzer behavior changes based on CLI config
**Source:** Verified from `c7_agent/agent.go` lines 26-29 (`Enable`)

**Example:**
```go
// internal/analyzer/c7_agent/agent.go -- new field and method
type C7Analyzer struct {
    evaluator   *agent.Evaluator
    enabled     bool
    debug       bool       // NEW: debug mode flag
    debugWriter io.Writer  // NEW: where debug output goes
}

func (a *C7Analyzer) SetDebug(enabled bool, w io.Writer) {
    a.debug = enabled
    a.debugWriter = w
}
```

### Pattern 4: DebugWriter as io.Discard / os.Stderr toggle

**What:** Use `io.Discard` for zero-cost disabled mode, `os.Stderr` for enabled mode
**When to use:** Any conditional output that must have zero overhead when disabled
**Source:** `io.Discard` is stdlib; pattern used by `Spinner` and `C7Progress` for conditional output

**Example:**
```go
// Zero-cost when debug is off: io.Discard.Write() returns immediately
// No conditional checks needed in hot path
fmt.Fprintf(a.debugWriter, "[DEBUG] [C7] Starting evaluation\n")
// When debugWriter is io.Discard, this is effectively free
// When debugWriter is os.Stderr, this writes to stderr
```

### Pattern 5: Auto-enable C7 from debug flag (DBG-02)

**What:** `--debug-c7` implies `--enable-c7` so users don't need both flags
**When to use:** When a debug flag is meaningless without its parent feature enabled
**Source:** Principle of least surprise; mirrors how `--debug-c7` is described in requirements

**Implementation detail:** Set `enableC7 = true` BEFORE the existing `if enableC7 {}` block in `cmd/scan.go`. This reuses the existing C7 enablement logic including the Claude CLI availability check and error message. No duplication.

```go
// In RunE, before the existing enableC7 block (line 90):
if debugC7 {
    enableC7 = true
}

// Existing block at line 90-96 handles:
if enableC7 {
    if !cliStatus.Available {
        spinner.Stop("")
        return fmt.Errorf("--enable-c7 requires Claude Code CLI...")
    }
    p.SetC7Enabled()
}

// After C7 enabled, set debug mode:
if debugC7 {
    p.SetC7Debug(true)
}
```

### Anti-Patterns to Avoid

- **Global debug variable:** Don't use a package-level `var debug bool` that metrics check directly. Thread state through method calls, matching the existing `Pipeline.verbose` and `C7Analyzer.enabled` patterns.
- **Environment variable for debug:** Don't check `os.Getenv("ARS_DEBUG")`. CLI flags are the established interface. Environment variables may be added later for per-metric filtering, but the primary toggle must be a flag.
- **Modifying the Executor interface:** Don't add debug parameters to `metrics.Executor.ExecutePrompt()`. The executor is an abstraction boundary for testability. Debug context flows through the analyzer, not the executor.
- **Writing debug output to `p.writer` (stdout):** This will corrupt `--json` output. All debug output must go to `debugWriter` (which is always `io.Discard` or `os.Stderr`, never stdout).
- **Conditional check on every write:** Don't use `if p.debugC7 { fmt.Fprintf(os.Stderr, ...) }`. Use the `debugWriter` pattern instead -- the `io.Discard` receiver handles the "disabled" case with zero branching overhead.

## Don't Hand-Roll

Problems that look simple but have existing solutions:

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Zero-cost conditional output | Custom `debugf()` with `if enabled` check | `io.Discard` as `io.Writer` | stdlib, zero-alloc, no branching in hot path |
| TTY detection | Manual `os.Stat` or env var checks | `isatty.IsTerminal(os.Stderr.Fd())` | Already a dependency (`go-isatty`), already used in `Spinner` and `C7Progress` |
| Flag registration | Manual `os.Args` parsing | `cobra.Command.Flags().BoolVar()` | Already used for all 8 existing flags |
| Debug writer abstraction | Custom `DebugLogger` interface | Bare `io.Writer` field on structs | `io.Writer` is the universal Go output abstraction; anything more is premature |

**Key insight:** Phase 26 is pure plumbing. There is nothing to hand-roll because every component already exists as a pattern in the codebase. The work is connecting existing patterns to a new flag.

## Common Pitfalls

### Pitfall 1: Debug output reaches stdout, corrupts JSON

**What goes wrong:** Debug `fmt.Fprintf` calls use `p.writer` (which is `cmd.OutOrStdout()`) instead of `p.debugWriter` (stderr). The `--json` mode produces invalid JSON with debug lines interleaved.
**Why it happens:** `p.writer` is the most accessible writer in the pipeline. Developers instinctively write to the "output" writer.
**How to avoid:** The `debugWriter` field is a separate `io.Writer` that is NEVER `p.writer`. Name it distinctly. In code review, any `fmt.Fprintf(p.writer, "[DEBUG]")` is a bug.
**Warning signs:** `ars scan . --debug-c7 --json 2>/dev/null | jq` fails to parse.

### Pitfall 2: Auto-enable happens after the C7 availability check

**What goes wrong:** If `enableC7 = true` is set AFTER the existing `if enableC7 {}` block, the auto-enable is dead code. C7 is never actually enabled.
**Why it happens:** The developer adds the debug handling at the end of RunE, not before the existing enableC7 block.
**How to avoid:** Set `enableC7 = true` BEFORE line 90 in `cmd/scan.go` (the existing `if enableC7 {}` block). This reuses the CLI check, error handling, and `SetC7Enabled()` call.
**Warning signs:** `ars scan . --debug-c7` does not run C7 evaluation; output shows "C7: Agent Evaluation (disabled)".

### Pitfall 3: Performance regression when debug is disabled

**What goes wrong:** Adding `debugWriter io.Writer` fields or `debug bool` checks to the Pipeline adds overhead even when debug is off. Go evaluates function arguments before the `if` check.
**Why it happens:** `fmt.Sprintf("[DEBUG] metric %s scored %d", metricID, score)` allocates and formats even if the result is discarded by `io.Discard`.
**How to avoid:** In Phase 26, there are NO actual debug writes -- just the plumbing. In later phases, use the `debugWriter` pattern: writes to `io.Discard` are effectively free (the `Write` method returns immediately). The overhead is one function call and zero allocations. For complex format strings, defer to later phases where the pattern can be `fmt.Fprintf(w, "[DEBUG] [%s] ...", id, ...)` -- the `Fprintf` overhead to `io.Discard` is negligible.
**Warning signs:** Benchmark shows non-zero regression with `--debug-c7` absent.

### Pitfall 4: Forgetting to thread debugWriter to C7Analyzer

**What goes wrong:** The `Pipeline` has `debugWriter` but it never reaches `C7Analyzer`. Later phases try to use `a.debugWriter` in the analyzer but it is nil/unset.
**Why it happens:** The developer adds `SetC7Debug` to Pipeline but doesn't call `a.SetDebug(enabled, p.debugWriter)` on the C7Analyzer.
**How to avoid:** Pattern match exactly with `SetC7Enabled()`: the Pipeline method calls the Analyzer method. Add the threading in `SetC7Debug()` itself.
**Warning signs:** Phase 27 or 29 fails because `C7Analyzer.debugWriter` is nil.

### Pitfall 5: Flag interaction with --no-llm

**What goes wrong:** `--debug-c7 --no-llm` is contradictory but produces confusing behavior. Debug enables C7, but `--no-llm` disables the evaluator, so C7 returns disabled result with no debug output.
**How to avoid:** `--debug-c7` should take precedence: if debug is requested, the user wants C7 to run. Check: if `debugC7 && noLLM`, print a warning to stderr: "Warning: --debug-c7 overrides --no-llm for C7 evaluation". Or, more simply, the existing `enableC7` path already requires `cliStatus.Available` -- if CLI is present and `debugC7` is set, C7 runs regardless of `noLLM` because `noLLM` only affects C4 LLM features, not C7.
**Warning signs:** `ars scan . --debug-c7 --no-llm` silently produces no C7 output.

### Pitfall 6: Not testing flag interaction with --json

**What goes wrong:** Debug works in terminal mode but was never tested with `--json`. The debug write to stderr is fine, but some incidental stdout write (like the "Claude CLI detected" message) breaks JSON.
**Why it happens:** The existing "Claude CLI detected" message on line 80 of `cmd/scan.go` writes to `cmd.OutOrStdout()`. This is already a problem with `--json` mode but is tolerated. With `--debug-c7` implying `--enable-c7`, this message now appears and contaminates JSON.
**How to avoid:** Add a test that verifies: `ars scan . --debug-c7 --json 2>/dev/null` produces valid JSON. The "Claude CLI detected" message issue is pre-existing and may need fixing regardless.
**Warning signs:** `--json` output starts with "Claude CLI detected..." instead of `{`.

## Code Examples

Verified patterns from the existing codebase:

### Existing flag declaration pattern (cmd/scan.go lines 15-24)
```go
// Source: cmd/scan.go (verified direct read)
var (
    configPath   string
    threshold    float64
    jsonOutput   bool
    noLLM        bool
    enableC7     bool
    outputHTML   string
    baselinePath string
    badgeOutput  bool
)
```

### Existing flag registration pattern (cmd/scan.go lines 124-133)
```go
// Source: cmd/scan.go init() (verified direct read)
scanCmd.Flags().BoolVar(&enableC7, "enable-c7", false,
    "enable C7 agent evaluation using Claude Code CLI (requires claude CLI installed)")
```

### Existing pipeline method threading (pipeline.go lines 119-124)
```go
// Source: pipeline.go SetC7Enabled (verified direct read)
func (p *Pipeline) SetC7Enabled() {
    if p.c7Analyzer != nil && p.evaluator != nil {
        p.c7Analyzer.Enable(p.evaluator)
    }
}
```

### Existing C7Analyzer enable pattern (c7_agent/agent.go lines 26-29)
```go
// Source: agent.go Enable (verified direct read)
func (a *C7Analyzer) Enable(evaluator *agent.Evaluator) {
    a.evaluator = evaluator
    a.enabled = true
}
```

### Spinner stderr pattern (pipeline/progress.go lines 30-37)
```go
// Source: progress.go NewSpinner (verified direct read)
func NewSpinner(w *os.File) *Spinner {
    return &Spinner{
        frames: []string{"|", "/", "-", "\\"},
        writer: w,
        isTTY:  isatty.IsTerminal(w.Fd()) || isatty.IsCygwinTerminal(w.Fd()),
        done:   make(chan struct{}),
    }
}
```

### Pipeline constructor (pipeline.go line 71)
```go
// Source: pipeline.go New (verified direct read)
// The `w` parameter is cmd.OutOrStdout() -- this is stdout for structured output
p := pipeline.New(cmd.OutOrStdout(), verbose, cfg, threshold, jsonOutput, onProgress)
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `log.Printf` for debug | `io.Writer` field with `io.Discard` toggle | Go 1.16 (io.Discard added) | Zero-cost disable without conditional checks |
| Per-call `if debug {}` | Write to `io.Discard` always | Established Go idiom | No branching in hot path; cleaner code |
| `--verbose` for everything | Category-specific `--debug-c7` | ARS v0.0.5 design decision | Avoids noise: verbose affects all categories, debug is C7-only |

**Deprecated/outdated:**
- `ioutil.Discard`: Deprecated since Go 1.16, replaced by `io.Discard`. The codebase already uses `io` package.

## Open Questions

1. **Pre-existing stdout message contamination with --json**
   - What we know: Lines 78-85 of `cmd/scan.go` write "Claude CLI detected" and "LLM features disabled" to `cmd.OutOrStdout()`. This is stdout. When combined with `--json`, these messages appear before the JSON object.
   - What's unclear: Whether this is already a known issue or is tolerated by users.
   - Recommendation: Phase 26 should NOT fix this pre-existing issue. Test with `--debug-c7 --json` should validate JSON output, and if the pre-existing messages break it, note it as a pre-existing bug and file separately. The debug flag itself should not introduce NEW stdout contamination.

2. **Debug writer type for later phases**
   - What we know: Phase 26 establishes `io.Writer` field. Later phases (27-29) need to write structured debug output with metric ID prefixes and concurrent-safe access.
   - What's unclear: Whether a plain `io.Writer` is sufficient or whether a mutex-wrapped writer is needed from the start.
   - Recommendation: Start with plain `io.Writer` (either `io.Discard` or `os.Stderr`). `os.Stderr` writes are already safe for concurrent use in Go (file writes are atomic up to a certain size). If interleaving becomes a problem in Phase 27+, wrap at that point. Don't over-engineer in Phase 26.

## Sources

### Primary (HIGH confidence)

All findings verified by direct codebase reading:

- `cmd/scan.go` -- Flag declaration (lines 15-24), registration (lines 124-133), C7 enablement (lines 89-96)
- `cmd/root.go` -- `verbose` persistent flag pattern (line 23)
- `internal/pipeline/pipeline.go` -- Pipeline struct (lines 26-43), New() constructor (lines 49-100), SetC7Enabled() (lines 119-124), Run() (lines 140-288), writer usage throughout
- `internal/analyzer/c7_agent/agent.go` -- C7Analyzer struct (lines 15-18), Enable() (lines 26-29), Analyze() (lines 37-87)
- `internal/pipeline/progress.go` -- Spinner stderr pattern (lines 30-37), isatty check (line 34)
- `internal/agent/progress.go` -- C7Progress stderr pattern (lines 50-71), isatty check (line 67)
- `go.mod` -- cobra v1.10.2, go-isatty v0.0.20

### Secondary (HIGH confidence)

Prior milestone research (same codebase, same day):

- `.planning/research/ARCHITECTURE-debug.md` -- Debug architecture with complete data flow diagrams
- `.planning/research/FEATURES-c7-debug.md` -- Feature landscape and MVP definition
- `.planning/research/PITFALLS.md` -- 7 critical pitfalls with prevention strategies
- `.planning/research/SUMMARY.md` -- Consolidated findings and phase ordering rationale

### Tertiary (MEDIUM confidence)

Web-verified patterns:

- [cobra package docs](https://pkg.go.dev/github.com/spf13/cobra) -- BoolVar flag registration
- [cobra working with flags guide](https://cobra.dev/docs/how-to-guides/working-with-flags/) -- Local vs persistent flags
- [Go os.Stderr best practices](https://dev.to/wycliffealphus/leveraging-osstderr-in-go-best-practices-for-effective-error-handling-3iof) -- stderr for diagnostics
- [AWS CLI Issue #5187](https://github.com/aws/aws-cli/issues/5187) -- Industry precedent for debug-to-stderr

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - Zero new dependencies. All patterns verified in existing codebase with line references.
- Architecture: HIGH - Direct pattern matching against existing `--enable-c7` flow. Three files to modify, all read and analyzed.
- Pitfalls: HIGH - 6 specific pitfalls identified from codebase analysis, each with prevention strategy mapped to specific lines of code.

**Research date:** 2026-02-06
**Valid until:** 2026-03-06 (stable; no external dependencies, all internal patterns)
