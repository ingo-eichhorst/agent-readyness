# Phase 5: Hardening - Research

**Researched:** 2026-02-01
**Domain:** Edge case handling, performance optimization, progress indicators (Go CLI tool)
**Confidence:** HIGH

## Summary

This phase hardens the ARS tool for real-world use across three domains: (1) graceful handling of edge cases like symlinks, syntax errors, and Unicode paths; (2) performance ensuring 50k LOC repos scan in under 30 seconds; and (3) progress indicators for long-running operations.

The current codebase uses `filepath.WalkDir` for discovery, which does NOT follow symlinks by default -- this is the correct behavior and just needs explicit detection/logging. The `go/packages` parser already handles syntax errors with partial results. Unicode paths work natively in Go since all strings are UTF-8. Progress indicators should be a simple stderr spinner (not a full progress bar library), since the tool already depends on `mattn/go-isatty` transitively via `fatih/color`. Performance optimization should focus on profiling the `go/packages.Load` call which is the known bottleneck, and parallelizing analyzers using `golang.org/x/sync/errgroup` which is already in the dependency tree.

**Primary recommendation:** The hardening work is mostly about adding resilient error handling wrappers around existing code, a lightweight stderr spinner, and benchmarking with a profiler -- not rewriting the pipeline architecture.

## Standard Stack

The established libraries/tools for this domain:

### Core (Already in go.mod)
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `mattn/go-isatty` | v0.0.20 | TTY detection for progress indicators | Already a transitive dependency via fatih/color; the standard Go TTY detection library |
| `golang.org/x/sync/errgroup` | v0.19.0 | Bounded parallel goroutine execution | Already in go.sum; standard Go library for parallel work with error propagation |
| `fatih/color` | v1.18.0 | Terminal coloring (already used) | Auto-disables when not TTY; spinner text uses this |

### Supporting (No new dependencies needed)
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `os.Lstat` / `d.Type()` | stdlib | Symlink detection in walker | Checking `fs.ModeSymlink` on DirEntry type bits |
| `testing.B` + `go test -bench` | stdlib | Performance benchmarking | Validating the 30-second target on large repos |
| `runtime/pprof` | stdlib | CPU/memory profiling | Identifying bottlenecks if 30s target is not met |
| `time.NewTicker` | stdlib | Spinner animation timing | 100-200ms tick for spinner frame rotation |
| `sync.Mutex` | stdlib | Thread-safe spinner state | Protecting spinner output from concurrent writes |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Hand-rolled spinner | `briandowns/spinner` or `schollz/progressbar` | Full libraries add dependency weight for a feature that needs ~50 lines of code; the tool already has go-isatty |
| `errgroup` for analyzer parallelism | Sequential analyzers (current) | Sequential is simpler but leaves performance on the table; analyzers are independent and parallelize trivially |
| `filepath.WalkDir` | `facebookgo/symwalk` | symwalk follows symlinks but risks infinite loops; WalkDir's default skip-symlinks behavior is safer and correct |

**Installation:**
```bash
# No new dependencies required. All needed packages are already in go.mod/go.sum.
# go-isatty is transitive via fatih/color
# errgroup is in golang.org/x/sync (already present)
```

## Architecture Patterns

### Recommended Changes to Project Structure
```
internal/
  discovery/
    walker.go          # ADD: symlink detection, error resilience in WalkDir callback
  parser/
    parser.go          # MODIFY: better syntax error handling, partial result collection
  pipeline/
    pipeline.go        # MODIFY: parallel analyzer execution, progress callback integration
    progress.go        # NEW: spinner/progress indicator implementation
  output/
    terminal.go        # No changes needed (already handles TTY via fatih/color)
```

### Pattern 1: Resilient Walker with Symlink Detection
**What:** Modify the `filepath.WalkDir` callback to detect symlinks via `d.Type()&fs.ModeSymlink != 0`, log them as warnings, and skip them. Handle permission errors and other walk errors without aborting the entire scan.
**When to use:** Always -- this is the primary edge case handler.
**Example:**
```go
// Source: Go stdlib filepath.WalkDir documentation + go/issues/4759
err = filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
    if err != nil {
        // Permission denied, broken symlink target, etc.
        // Log warning and skip this entry instead of aborting scan
        log.Printf("warning: skipping %s: %v", path, err)
        if d != nil && d.IsDir() {
            return fs.SkipDir
        }
        return nil // skip file, continue walk
    }

    // Detect symlinks (WalkDir does not follow them, but we should log)
    if d.Type()&fs.ModeSymlink != 0 {
        log.Printf("warning: skipping symlink: %s", path)
        return nil // skip symlinks
    }

    // ... rest of existing walk logic
    return nil
})
```

### Pattern 2: Parallel Analyzer Execution with errgroup
**What:** Run the three analyzers (C1, C3, C6) concurrently since they are independent read-only operations on the same parsed packages.
**When to use:** When analyzers are stateless and do not share mutable state.
**Example:**
```go
// Source: golang.org/x/sync/errgroup docs
import "golang.org/x/sync/errgroup"

g, ctx := errgroup.WithContext(context.Background())

var mu sync.Mutex
for _, a := range p.analyzers {
    a := a // capture loop variable
    g.Go(func() error {
        ar, err := a.Analyze(pkgs)
        if err != nil {
            fmt.Fprintf(p.writer, "Warning: %s analyzer error: %v\n", a.Name(), err)
            return nil // don't abort other analyzers
        }
        mu.Lock()
        p.results = append(p.results, ar)
        mu.Unlock()
        return nil
    })
}
if err := g.Wait(); err != nil {
    // handle unexpected error
}
```

### Pattern 3: Stderr Spinner with TTY Detection
**What:** A lightweight spinner that writes to stderr, auto-detects TTY, and replaces itself with a completion message. Uses carriage return (`\r`) to overwrite the current line.
**When to use:** When a pipeline stage takes longer than a configurable threshold (e.g., 500ms).
**Example:**
```go
// Source: mattn/go-isatty API + briandowns/spinner pattern
import "github.com/mattn/go-isatty"

type Spinner struct {
    mu      sync.Mutex
    frames  []string
    current int
    message string
    active  bool
    writer  *os.File
    ticker  *time.Ticker
    done    chan struct{}
}

func NewSpinner(w *os.File) *Spinner {
    return &Spinner{
        frames: []string{"|", "/", "-", "\\"},
        writer: w,
        done:   make(chan struct{}),
    }
}

func (s *Spinner) Start(message string) {
    if !isatty.IsTerminal(s.writer.Fd()) {
        return // no spinner when not a TTY
    }
    s.mu.Lock()
    s.message = message
    s.active = true
    s.mu.Unlock()

    s.ticker = time.NewTicker(100 * time.Millisecond)
    go func() {
        for {
            select {
            case <-s.ticker.C:
                s.mu.Lock()
                if s.active {
                    frame := s.frames[s.current%len(s.frames)]
                    fmt.Fprintf(s.writer, "\r%s %s", frame, s.message)
                    s.current++
                }
                s.mu.Unlock()
            case <-s.done:
                return
            }
        }
    }()
}

func (s *Spinner) Stop(finalMessage string) {
    s.mu.Lock()
    s.active = false
    s.mu.Unlock()
    if s.ticker != nil {
        s.ticker.Stop()
    }
    close(s.done)
    if isatty.IsTerminal(s.writer.Fd()) {
        fmt.Fprintf(s.writer, "\r%s\n", finalMessage)
    }
}
```

### Pattern 4: Progress Callback in Pipeline
**What:** Pipeline stages report progress via a callback function, allowing the spinner to update its message as stages complete.
**When to use:** To provide per-phase progress without coupling pipeline to UI.
**Example:**
```go
type ProgressFunc func(stage string, detail string)

func (p *Pipeline) Run(dir string, onProgress ProgressFunc) error {
    if onProgress == nil {
        onProgress = func(string, string) {} // no-op default
    }

    onProgress("discover", "Scanning files...")
    result, err := walker.Discover(dir)
    // ...

    onProgress("parse", "Parsing packages...")
    pkgs, err := p.parser.Parse(dir)
    // ...

    onProgress("analyze", "Analyzing code...")
    // run analyzers...

    onProgress("score", "Computing scores...")
    // score...
}
```

### Anti-Patterns to Avoid
- **Aborting on first edge case:** The walker currently returns errors that abort the entire scan. Edge cases (permission denied, broken symlinks) should log warnings and continue.
- **Spinner on stdout:** Progress output MUST go to stderr so it does not corrupt stdout (especially when `--json` output is piped).
- **Unbounded goroutines:** Use `errgroup.SetLimit()` if parallelizing file I/O; for 3 analyzers this is not an issue but establish the pattern.
- **Ignoring partial parse results:** The parser already handles this well (skips packages with errors but includes partial results). Do not change this behavior.

## Don't Hand-Roll

Problems that look simple but have existing solutions:

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| TTY detection | Custom `ioctl` calls or `os.Stat` heuristics | `mattn/go-isatty` (already in deps) | Cross-platform edge cases (Cygwin, MSYS2, Windows Console) |
| Parallel goroutine orchestration | Manual `sync.WaitGroup` + error channels | `errgroup` from `golang.org/x/sync` (already in deps) | `SetLimit`, context cancellation, first-error propagation are tricky to get right |
| Symlink resolution | Custom `os.Readlink` chains | `filepath.EvalSymlinks` (stdlib) | Handles chains of symlinks, relative targets, platform differences |
| Unicode path normalization | Manual NFC/NFD handling | Go stdlib (native UTF-8 strings) | Go paths are just byte sequences; OS handles encoding |

**Key insight:** Go's standard library and the project's existing dependencies already solve every hard problem in this phase. The hardening work is about wiring existing capabilities together correctly, not about adding new heavy dependencies.

## Common Pitfalls

### Pitfall 1: Walker Errors Abort the Entire Scan
**What goes wrong:** The current `filepath.WalkDir` callback returns errors from `d.Info()`, generated file checks, or relative path computation, which causes `WalkDir` to abort.
**Why it happens:** The default pattern is to propagate errors, but in a scanning tool, individual file failures should not stop the overall scan.
**How to avoid:** Catch errors in the WalkDir callback, log warnings, and return `nil` (or `fs.SkipDir` for directory errors) instead of the error. Count skipped files for summary reporting.
**Warning signs:** Users report "walk error: permission denied" on repos with a single unreadable file.

### Pitfall 2: Spinner Corrupts JSON Output
**What goes wrong:** If progress indicators write to stdout, they interleave with JSON output, producing invalid JSON.
**Why it happens:** Spinner writes to same stream as program output.
**How to avoid:** Always write spinner to `os.Stderr`. When `--json` flag is set, also consider disabling spinner entirely or ensuring it only uses stderr.
**Warning signs:** `ars scan . --json | jq .` fails with parse errors.

### Pitfall 3: go/packages.Load is the Real Bottleneck
**What goes wrong:** Developers optimize the walker or analyzers when the actual time is spent in `packages.Load` which shells out to `go list`.
**Why it happens:** `go list` does module resolution, downloads, and compilation. For 50k LOC repos, this can take 10-20 seconds on first run (cached runs are much faster).
**How to avoid:** Profile first with `go test -bench`. Accept that first-run performance depends on Go's build cache. The 30-second target should assume a warm build cache. Parallelizing analyzers helps but won't fix the packages.Load bottleneck.
**Warning signs:** Profiling shows 80%+ time in `packages.Load`.

### Pitfall 4: Symlink Detection Only Catches File Symlinks
**What goes wrong:** `d.Type()&fs.ModeSymlink` in WalkDir only fires for file symlinks. Directory symlinks are skipped silently by WalkDir (it never recurses into them) but the DirEntry IS still visited.
**Why it happens:** `filepath.WalkDir` does not follow directory symlinks (documented behavior since Go 1.0). The callback still receives the directory entry but with the symlink type bit set.
**How to avoid:** Check `d.Type()&fs.ModeSymlink` for both file and directory entries. Log both cases. This is correct -- do NOT follow directory symlinks (infinite loop risk).
**Warning signs:** Users with symlinked `pkg/` directories get incomplete scans with no warning.

### Pitfall 5: Unicode Paths Are a Non-Issue in Go (But Test Anyway)
**What goes wrong:** Developers spend time on Unicode normalization when Go handles this natively.
**Why it happens:** Experience from other languages (Python 2, Java) where encoding was a constant battle.
**How to avoid:** Go strings are UTF-8 by default. `filepath.WalkDir`, `os.Open`, and `os.Stat` all work with Unicode paths. The only real risk is on macOS where HFS+ uses NFD normalization (e.g., "cafe\u0301" vs "caf\u00e9"). This affects path comparison, not path traversal. Add a test with Unicode paths but do not add normalization logic.
**Warning signs:** None expected in practice. Test with `testdata/unicod\u00e9-path/` directory.

### Pitfall 6: Race Condition in Parallel Analyzer Output
**What goes wrong:** Concurrent analyzers write warnings to `p.writer` simultaneously, producing garbled output.
**Why it happens:** `fmt.Fprintf` to an `io.Writer` is not atomic for concurrent goroutines.
**How to avoid:** Use a mutex around warning output, or collect warnings and display them after `errgroup.Wait()`.
**Warning signs:** Interleaved warning messages in terminal output.

## Code Examples

### Resilient WalkDir Callback (Edge Case Handling)
```go
// Handles: symlinks, permission errors, broken paths
// Returns nil to continue walk even on errors
err = filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
    if err != nil {
        // Count and log, but don't abort
        result.SkippedCount++
        if p.verbose {
            fmt.Fprintf(os.Stderr, "warning: %s: %v\n", path, err)
        }
        if d != nil && d.IsDir() {
            return fs.SkipDir
        }
        return nil
    }

    // Symlink detection
    if d.Type()&fs.ModeSymlink != 0 {
        result.SymlinkCount++
        if p.verbose {
            fmt.Fprintf(os.Stderr, "warning: skipping symlink: %s\n", path)
        }
        return nil
    }

    // ... existing classification logic ...
    return nil
})
```

### TTY-Aware Spinner on Stderr
```go
// Check TTY on the stream you write to (stderr for spinner)
import "github.com/mattn/go-isatty"

func shouldShowProgress() bool {
    return isatty.IsTerminal(os.Stderr.Fd()) ||
           isatty.IsCygwinTerminal(os.Stderr.Fd())
}
```

### Benchmark Test for Performance Target
```go
// Source: Go testing.B documentation
func BenchmarkFullScan(b *testing.B) {
    dir := os.Getenv("ARS_BENCH_DIR")
    if dir == "" {
        b.Skip("set ARS_BENCH_DIR to a 50k LOC Go repo")
    }
    for i := 0; i < b.N; i++ {
        p := pipeline.New(io.Discard, false, nil, 0, false)
        if err := p.Run(dir); err != nil {
            b.Fatal(err)
        }
    }
}
```

### Parallel Analyzers with errgroup
```go
import (
    "golang.org/x/sync/errgroup"
    "sync"
)

g := new(errgroup.Group)
var (
    mu      sync.Mutex
    results []*types.AnalysisResult
)

for _, a := range p.analyzers {
    a := a
    g.Go(func() error {
        ar, err := a.Analyze(pkgs)
        if err != nil {
            mu.Lock()
            warnings = append(warnings, fmt.Sprintf("%s: %v", a.Name(), err))
            mu.Unlock()
            return nil
        }
        mu.Lock()
        results = append(results, ar)
        mu.Unlock()
        return nil
    })
}
_ = g.Wait()
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `filepath.Walk` | `filepath.WalkDir` | Go 1.16 (2021) | WalkDir avoids `os.Lstat` per entry, 2-3x faster on large trees |
| Manual WaitGroup + channels | `errgroup` with `SetLimit` | errgroup v0.19.0 | Cleaner bounded concurrency |
| Custom TTY detection | `mattn/go-isatty` | stable since 2020 | Cross-platform, handles Cygwin/MSYS2 |
| `go list -json` parsing | `go/packages.Load` | Go 1.11+ (modules era) | Handles modules, type info, build constraints natively |

**Deprecated/outdated:**
- `filepath.Walk`: Use `filepath.WalkDir` instead (already done in this codebase)
- Manual `os.Stat` for TTY: Use `go-isatty` (already in dependency tree)

## Open Questions

1. **Exact 30-second benchmark target corpus**
   - What we know: 50k LOC Go repo target
   - What's unclear: Should this be measured cold-cache or warm-cache? First run of `go/packages.Load` on a fresh clone can be significantly slower due to module downloads and compilation.
   - Recommendation: Benchmark with warm cache (second run). Document that first-run performance depends on Go toolchain cache. This is the standard approach for Go tool benchmarks.

2. **Progress threshold: when to show spinner**
   - What we know: User wants progress "for long-running scans"
   - What's unclear: What constitutes "long-running"? 1 second? 2 seconds?
   - Recommendation: Start spinner immediately for any scan (it will be invisible on fast scans since it completes quickly). Alternatively, delay spinner start by 500ms using a timer -- if the scan completes before 500ms, no spinner is shown.

3. **Analyzer ordering after parallelization**
   - What we know: Results are appended to a slice concurrently
   - What's unclear: Does output ordering matter (C1 before C3 before C6)?
   - Recommendation: Sort results by analyzer name after `g.Wait()` to ensure deterministic output ordering regardless of which analyzer finishes first.

## Sources

### Primary (HIGH confidence)
- Go stdlib `filepath.WalkDir` documentation -- symlink behavior, `fs.DirEntry.Type()` for symlink detection
- Go stdlib `filepath` package -- `EvalSymlinks` for symlink resolution
- `golang.org/x/sync/errgroup` v0.19.0 -- `Go()`, `SetLimit()`, `WithContext()` API verified via pkg.go.dev
- `mattn/go-isatty` v0.0.20 -- `IsTerminal(fd)`, `IsCygwinTerminal(fd)` verified via pkg.go.dev
- Project `go.mod` -- confirmed `golang.org/x/sync v0.19.0` and `mattn/go-isatty v0.0.20` already in dependency tree
- Project source code -- walker.go, pipeline.go, parser.go, terminal.go analyzed for current architecture

### Secondary (MEDIUM confidence)
- [golang/go#4759](https://github.com/golang/go/issues/4759) -- filepath.Walk symlink policy rationale
- [golang/go#29758](https://github.com/golang/go/issues/29758) -- go/packages parallelization status
- [incident.io blog](https://incident.io/blog/go-build-faster) -- go/packages performance characteristics in large codebases
- [briandowns/spinner](https://github.com/briandowns/spinner) -- stderr writer pattern for spinner libraries
- [schollz/progressbar](https://github.com/schollz/progressbar) -- spinner-as-unknown-length-bar pattern

### Tertiary (LOW confidence)
- General Go performance optimization guides -- common patterns but not specific to this codebase

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - all libraries already in go.mod, APIs verified on pkg.go.dev
- Architecture: HIGH - patterns derived from direct analysis of existing codebase + Go stdlib docs
- Pitfalls: HIGH - derived from Go documentation and known behaviors of filepath.WalkDir, go/packages
- Performance: MEDIUM - 30-second target feasibility depends on `go/packages.Load` which is hard to optimize externally

**Research date:** 2026-02-01
**Valid until:** 2026-03-01 (stable domain, no fast-moving dependencies)
