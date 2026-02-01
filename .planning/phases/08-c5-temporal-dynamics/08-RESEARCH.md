# Phase 8: C5 Temporal Dynamics - Research

**Researched:** 2026-02-01
**Domain:** Git log parsing, temporal code analysis, churn metrics
**Confidence:** HIGH

## Summary

C5 Temporal Dynamics requires parsing git history to extract code churn, temporal coupling, author fragmentation, commit stability, and hotspot concentration metrics. The implementation uses native `git log` CLI (not go-git) for 10-100x faster performance, parsing the structured output via Go's `os/exec` with streamed `bufio.Scanner` reading.

The existing codebase has a clear Analyzer interface pattern (`pipeline.Analyzer`) with per-category metric structs, extractor functions for scoring, and breakpoint-based interpolation. C5 fits cleanly into this architecture: a new `C5Analyzer` implementing `Analyzer`, a `C5Metrics` struct in `pkg/types`, an `extractC5` function in `internal/scoring`, and C5 entries in `DefaultConfig()`.

**Primary recommendation:** Implement a single `git log --pretty=format:'%H|%ae|%at' --numstat --since='6 months ago' --no-merges` command, parse the output in a single streaming pass to build per-file statistics, then compute all 5 metrics from the accumulated data. The C5 analyzer is unique because it operates on the repository root (not per-file targets) -- it needs `RootDir` from any `AnalysisTarget` but ignores individual file contents.

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `os/exec` | stdlib | Execute `git log` CLI | No dependencies needed, project already uses native git decision |
| `bufio` | stdlib | Stream-parse git output line by line | Memory-efficient for large repos |
| `strings` | stdlib | Parse delimited git log format | Simple field splitting |
| `strconv` | stdlib | Parse unix timestamps, line counts | Standard numeric parsing |
| `time` | stdlib | Time window calculations (90-day, 6-month) | Date arithmetic |
| `sort` | stdlib | Rank files for hotspot concentration | Top-10% calculation |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `path/filepath` | stdlib | Normalize file paths from git output | Always -- git paths use forward slashes |
| `context` | stdlib | Timeout enforcement for 30-second budget | Wrap exec.CommandContext |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Native git CLI | go-git library | go-git is 10-100x slower on large repos (decided against in STATE.md) |
| Single git log call | Multiple git commands | Single call is faster, all data comes from one parse pass |
| `--numstat` | `--shortstat` | `--numstat` gives per-file detail needed for coupling/hotspots; `--shortstat` only gives totals |

**Installation:** No new dependencies needed -- all stdlib.

## Architecture Patterns

### Recommended Project Structure
```
internal/analyzer/
    c5_temporal.go       # C5Analyzer + git log parsing + all metric computations
    c5_temporal_test.go  # Unit tests with fixture git log output
pkg/types/
    types.go             # Add C5Metrics struct
internal/scoring/
    scorer.go            # Add extractC5 + RegisterExtractor("C5", extractC5)
    config.go            # Add C5 to DefaultConfig()
internal/recommend/
    recommend.go         # Add C5 metric impact descriptions + action templates
```

### Pattern 1: Single-Pass Git Log Parsing

**What:** Execute one `git log` command and parse the entire output in a single streaming pass, building all intermediate data structures (per-file stats, per-commit file lists, per-file author sets) simultaneously.

**When to use:** Always -- this is the only efficient approach for meeting the 30-second performance budget.

**The git command:**
```bash
git log --pretty=format:'%H|%ae|%at' --numstat --since='6 months ago' --no-merges
```

**Output format (one commit block):**
```
<commit-hash>|<author-email>|<unix-timestamp>
<added>\t<deleted>\t<filepath>
<added>\t<deleted>\t<filepath>
<blank-line>
```

**Parsing pseudocode:**
```go
type commitInfo struct {
    Hash      string
    Author    string
    Timestamp int64
    Files     []fileChange
}

type fileChange struct {
    Added   int
    Deleted int
    Path    string
}

// Parse streaming output
scanner := bufio.NewScanner(stdout)
var current *commitInfo
var commits []commitInfo

for scanner.Scan() {
    line := scanner.Text()
    if line == "" {
        if current != nil {
            commits = append(commits, *current)
            current = nil
        }
        continue
    }
    if strings.Contains(line, "|") && len(strings.Split(line, "|")) == 3 {
        // Header line: hash|author|timestamp
        parts := strings.Split(line, "|")
        ts, _ := strconv.ParseInt(parts[2], 10, 64)
        current = &commitInfo{Hash: parts[0], Author: parts[1], Timestamp: ts}
    } else if current != nil {
        // Numstat line: added\tdeleted\tfilepath
        parts := strings.Split(line, "\t")
        if len(parts) == 3 {
            added, _ := strconv.Atoi(parts[0])
            deleted, _ := strconv.Atoi(parts[1])
            current.Files = append(current.Files, fileChange{
                Added: added, Deleted: deleted, Path: parts[2],
            })
        }
    }
}
```

### Pattern 2: C5 Analyzer as Repo-Level (Not File-Level) Analyzer

**What:** Unlike C1/C2/C3/C6 which analyze individual source files, C5 operates on the entire repository's git history. It needs `RootDir` but not individual `AnalysisTarget.Files`.

**When to use:** Always -- this is fundamental to C5's design.

**Implementation:**
```go
type C5Analyzer struct{}

func (a *C5Analyzer) Name() string { return "C5: Temporal Dynamics" }

func (a *C5Analyzer) Analyze(targets []*types.AnalysisTarget) (*types.AnalysisResult, error) {
    if len(targets) == 0 {
        return nil, fmt.Errorf("no targets provided")
    }
    rootDir := targets[0].RootDir

    // Check for .git directory first (C5-07)
    if _, err := os.Stat(filepath.Join(rootDir, ".git")); os.IsNotExist(err) {
        // Return result with unavailable metrics, NOT an error
        return &types.AnalysisResult{
            Name:     "C5: Temporal Dynamics",
            Category: "C5",
            Metrics:  map[string]interface{}{"c5": &types.C5Metrics{Available: false}},
        }, nil
    }

    // Execute git log and parse
    metrics, err := analyzeGitHistory(rootDir, 6) // 6-month default window
    if err != nil {
        return nil, err
    }

    return &types.AnalysisResult{
        Name:     "C5: Temporal Dynamics",
        Category: "C5",
        Metrics:  map[string]interface{}{"c5": metrics},
    }, nil
}
```

### Pattern 3: Timeout-Protected Git Execution

**What:** Use `exec.CommandContext` with a timeout to ensure git commands respect the 30-second performance budget.

**Example:**
```go
func runGitLog(rootDir string, months int) (io.ReadCloser, *exec.Cmd, error) {
    since := fmt.Sprintf("--since=%d months ago", months)
    ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second) // leave 5s headroom
    cmd := exec.CommandContext(ctx, "git", "log",
        "--pretty=format:%H|%ae|%at",
        "--numstat",
        since,
        "--no-merges",
    )
    cmd.Dir = rootDir
    stdout, err := cmd.StdoutPipe()
    if err != nil {
        cancel()
        return nil, nil, err
    }
    if err := cmd.Start(); err != nil {
        cancel()
        return nil, nil, err
    }
    // Caller must call cancel() and cmd.Wait() after reading
    return stdout, cmd, nil
}
```

### Anti-Patterns to Avoid
- **Buffering entire git output into memory:** Use streaming `bufio.Scanner` with `StdoutPipe`, not `cmd.Output()` or `bytes.Buffer`. Large repos can produce hundreds of MB of git log output.
- **Multiple git commands for different metrics:** Parse everything from a single `git log --numstat` call. Multiple invocations multiply I/O cost.
- **Using go-git:** Decision was made in STATE.md to use native git CLI for 10-100x performance advantage.
- **Processing merge commits:** Use `--no-merges` to avoid inflated coupling from merge commits that touch many files.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Git log parsing | Custom recursive descent parser | Line-by-line scanner with field splitting | Git's `--pretty=format:` gives perfectly delimited output |
| Temporal coupling calculation | Custom graph algorithms | Simple co-occurrence counting with support/confidence from association rule mining | Well-established algorithm, CodeScene uses similar approach |
| Percentile calculation | Custom stats library | `sort.Float64s` + index math | Only need top-10% threshold, not full percentile library |
| Git availability check | Complex git version detection | `os.Stat(filepath.Join(rootDir, ".git"))` | Simple existence check is sufficient |

**Key insight:** All C5 metrics derive from the same parsed git log data. The hard part is efficient parsing; the metric calculations are straightforward arithmetic on accumulated maps.

## Common Pitfalls

### Pitfall 1: Binary Files in --numstat Output
**What goes wrong:** `git log --numstat` outputs `-\t-\tfilepath` for binary files (dashes instead of numbers).
**Why it happens:** Binary files don't have meaningful line counts.
**How to avoid:** Check for `-` in added/deleted fields before parsing as integer. Skip binary file entries.
**Warning signs:** `strconv.Atoi` errors during parsing.

### Pitfall 2: Renamed Files in --numstat
**What goes wrong:** Renamed files appear as `{old => new}` in the path field, e.g., `internal/{old_name.go => new_name.go}`.
**Why it happens:** Git tracks renames with `-M` flag (default behavior).
**How to avoid:** Parse the `{...}` rename syntax and use the NEW path for file statistics. Or strip the rename notation.
**Warning signs:** File paths containing `{`, `=>`, `}` characters.

### Pitfall 3: Large Commits Polluting Coupling Data
**What goes wrong:** Refactoring commits that touch 100+ files create false temporal coupling between unrelated files.
**Why it happens:** All files in a commit are considered co-changed.
**How to avoid:** Skip commits that modify more than 50 files (CodeScene's default threshold). This filters bulk refactors, large renames, and initial commits.
**Warning signs:** Unexpectedly high coupling percentages across the board.

### Pitfall 4: Empty Repositories or No History in Window
**What goes wrong:** `git log --since='6 months ago'` returns no output for repos with no recent activity.
**Why it happens:** The time window is too narrow for the repo's activity.
**How to avoid:** Return neutral scores (5.0) when insufficient data exists. Set `Available: false` on individual metrics that lack data.
**Warning signs:** Empty stdout from git command.

### Pitfall 5: Race Condition Reading StdoutPipe
**What goes wrong:** Calling `cmd.Wait()` before finishing reading from the pipe, or reading in a goroutine incorrectly.
**Why it happens:** Go's os/exec has specific ordering requirements.
**How to avoid:** Read ALL output from the pipe first, THEN call `cmd.Wait()`. Do not read in a separate goroutine unless using `cmd.Stdout = &buffer` pattern.
**Warning signs:** Intermittent test failures, truncated output.

### Pitfall 6: File Path Normalization
**What goes wrong:** Git outputs forward-slash paths; `AnalysisTarget.Files` may use OS-specific paths.
**Why it happens:** Git always uses `/` regardless of OS. Go's `filepath` uses `\` on Windows.
**How to avoid:** Normalize all paths to forward slashes for comparison, or use `filepath.ToSlash()`.
**Warning signs:** Files not matching between git output and target files.

## Code Examples

### C5Metrics Struct
```go
// C5Metrics holds Temporal & Operational Dynamics metric results.
type C5Metrics struct {
    Available            bool    // false if .git missing or insufficient history
    ChurnRate            float64 // avg lines changed per commit (90-day window)
    TemporalCouplingPct  float64 // % of file pairs with >70% co-change rate
    AuthorFragmentation  float64 // avg distinct authors per file (90-day window)
    CommitStability      float64 // median days between changes per file
    HotspotConcentration float64 // % of total changes in top 10% of files
    // Detail fields for verbose output / recommendations
    TopHotspots          []FileChurn    // top churning files
    CoupledPairs         []CoupledPair  // detected temporal couplings
    TotalCommits         int            // commits analyzed
    TimeWindowDays       int            // actual window used
}

// FileChurn holds churn data for a single file.
type FileChurn struct {
    Path         string
    TotalChanges int     // lines added + deleted
    CommitCount  int     // number of commits touching this file
    AuthorCount  int     // distinct authors
    LastChanged  int64   // unix timestamp of last change
}

// CoupledPair holds a pair of files with temporal coupling.
type CoupledPair struct {
    FileA       string
    FileB       string
    Coupling    float64 // percentage (0-100) of co-change frequency
    SharedCommits int   // number of commits where both changed
}
```

### Metric Calculations

**C5-02: Churn Rate (lines changed per commit, 90-day window)**
```go
func calcChurnRate(commits []commitInfo, windowDays int) float64 {
    cutoff := time.Now().Add(-time.Duration(windowDays) * 24 * time.Hour).Unix()
    totalLines := 0
    commitCount := 0
    for _, c := range commits {
        if c.Timestamp < cutoff {
            continue
        }
        commitCount++
        for _, f := range c.Files {
            totalLines += f.Added + f.Deleted
        }
    }
    if commitCount == 0 {
        return 0
    }
    return float64(totalLines) / float64(commitCount)
}
```

**C5-03: Temporal Coupling (files co-changed >70% of time)**
```go
func calcTemporalCoupling(commits []commitInfo, minCommits int, maxFilesPerCommit int) (float64, []CoupledPair) {
    // Count per-file commit frequency
    fileCommitCount := make(map[string]int)
    // Count co-occurrence for file pairs
    pairCount := make(map[[2]string]int)

    for _, c := range commits {
        if len(c.Files) > maxFilesPerCommit { // skip bulk commits (default: 50)
            continue
        }
        paths := uniquePaths(c.Files)
        for _, p := range paths {
            fileCommitCount[p]++
        }
        // Count co-occurrences (only for files with sufficient history)
        for i := 0; i < len(paths); i++ {
            for j := i + 1; j < len(paths); j++ {
                key := sortedPair(paths[i], paths[j])
                pairCount[key]++
            }
        }
    }

    // Calculate coupling strength: shared_commits / min(commits_A, commits_B)
    var coupled []CoupledPair
    totalPairs := 0
    coupledPairs := 0
    for pair, shared := range pairCount {
        countA := fileCommitCount[pair[0]]
        countB := fileCommitCount[pair[1]]
        if countA < minCommits || countB < minCommits { // default: 5
            continue
        }
        totalPairs++
        minCount := countA
        if countB < minCount {
            minCount = countB
        }
        strength := float64(shared) / float64(minCount) * 100
        if strength > 70 { // threshold from PRD
            coupledPairs++
            coupled = append(coupled, CoupledPair{
                FileA: pair[0], FileB: pair[1],
                Coupling: strength, SharedCommits: shared,
            })
        }
    }

    if totalPairs == 0 {
        return 0, nil
    }
    return float64(coupledPairs) / float64(totalPairs) * 100, coupled
}
```

**C5-04: Author Fragmentation (avg authors per file, 90-day window)**
```go
func calcAuthorFragmentation(commits []commitInfo, windowDays int) float64 {
    cutoff := time.Now().Add(-time.Duration(windowDays) * 24 * time.Hour).Unix()
    fileAuthors := make(map[string]map[string]bool) // file -> set of authors

    for _, c := range commits {
        if c.Timestamp < cutoff {
            continue
        }
        for _, f := range c.Files {
            if fileAuthors[f.Path] == nil {
                fileAuthors[f.Path] = make(map[string]bool)
            }
            fileAuthors[f.Path][c.Author] = true
        }
    }

    if len(fileAuthors) == 0 {
        return 0
    }
    totalAuthors := 0
    for _, authors := range fileAuthors {
        totalAuthors += len(authors)
    }
    return float64(totalAuthors) / float64(len(fileAuthors))
}
```

**C5-05: Commit Stability (median days between changes)**
```go
func calcCommitStability(commits []commitInfo) float64 {
    // Group commits by file, sorted by timestamp
    fileTimestamps := make(map[string][]int64)
    for _, c := range commits {
        for _, f := range c.Files {
            fileTimestamps[f.Path] = append(fileTimestamps[f.Path], c.Timestamp)
        }
    }

    var allIntervals []float64
    for _, timestamps := range fileTimestamps {
        sort.Slice(timestamps, func(i, j int) bool { return timestamps[i] < timestamps[j] })
        for i := 1; i < len(timestamps); i++ {
            days := float64(timestamps[i]-timestamps[i-1]) / 86400.0
            allIntervals = append(allIntervals, days)
        }
    }

    if len(allIntervals) == 0 {
        return 30 // default: stable if no repeat changes
    }
    sort.Float64s(allIntervals)
    return allIntervals[len(allIntervals)/2] // median
}
```

**C5-06: Hotspot Concentration (% changes in top 10% of files)**
```go
func calcHotspotConcentration(commits []commitInfo) (float64, []FileChurn) {
    fileChanges := make(map[string]int) // file -> total lines changed
    for _, c := range commits {
        for _, f := range c.Files {
            fileChanges[f.Path] += f.Added + f.Deleted
        }
    }

    if len(fileChanges) == 0 {
        return 0, nil
    }

    // Sort files by change volume
    type fc struct {
        path    string
        changes int
    }
    var sorted []fc
    totalChanges := 0
    for path, changes := range fileChanges {
        sorted = append(sorted, fc{path, changes})
        totalChanges += changes
    }
    sort.Slice(sorted, func(i, j int) bool { return sorted[i].changes > sorted[j].changes })

    // Top 10% of files
    top10Pct := len(sorted) / 10
    if top10Pct < 1 {
        top10Pct = 1
    }
    top10Changes := 0
    var hotspots []FileChurn
    for i := 0; i < top10Pct && i < len(sorted); i++ {
        top10Changes += sorted[i].changes
        hotspots = append(hotspots, FileChurn{Path: sorted[i].path, TotalChanges: sorted[i].changes})
    }

    return float64(top10Changes) / float64(totalChanges) * 100, hotspots
}
```

### Scoring Configuration for C5
```go
// Add to DefaultConfig() in config.go
"C5": {
    Name:   "Temporal Dynamics",
    Weight: 0.10,
    Metrics: []MetricThresholds{
        {
            Name:   "churn_rate",
            Weight: 0.20,
            Breakpoints: []Breakpoint{
                {Value: 50, Score: 10},   // <50 lines/commit avg = excellent
                {Value: 100, Score: 8},
                {Value: 300, Score: 6},
                {Value: 600, Score: 3},
                {Value: 1000, Score: 1},  // >1000 lines/commit = very churny
            },
        },
        {
            Name:   "temporal_coupling_pct",
            Weight: 0.25,
            Breakpoints: []Breakpoint{
                {Value: 0, Score: 10},
                {Value: 5, Score: 8},     // <5% coupled pairs = good
                {Value: 15, Score: 6},
                {Value: 25, Score: 3},
                {Value: 30, Score: 1},    // >30% = highly coupled
            },
        },
        {
            Name:   "author_fragmentation",
            Weight: 0.20,
            Breakpoints: []Breakpoint{
                {Value: 1, Score: 10},    // 1 author = strong ownership
                {Value: 2, Score: 8},
                {Value: 4, Score: 6},
                {Value: 6, Score: 3},
                {Value: 8, Score: 1},     // >8 authors avg = fragmented
            },
        },
        {
            Name:   "commit_stability",
            Weight: 0.15,
            Breakpoints: []Breakpoint{
                {Value: 0.5, Score: 1},   // <0.5 days between changes = unstable
                {Value: 1, Score: 3},
                {Value: 3, Score: 6},
                {Value: 7, Score: 8},     // >7 days = stable
                {Value: 14, Score: 10},
            },
        },
        {
            Name:   "hotspot_concentration",
            Weight: 0.20,
            Breakpoints: []Breakpoint{
                {Value: 20, Score: 10},   // <20% in top 10% = well distributed
                {Value: 30, Score: 8},
                {Value: 50, Score: 6},
                {Value: 70, Score: 3},
                {Value: 80, Score: 1},    // >80% = concentrated
            },
        },
    },
},
```

### Extractor Function
```go
func extractC5(ar *types.AnalysisResult) (map[string]float64, map[string]bool) {
    raw, ok := ar.Metrics["c5"]
    if !ok {
        return nil, nil
    }
    m, ok := raw.(*types.C5Metrics)
    if !ok {
        return nil, nil
    }

    if !m.Available {
        // All metrics unavailable
        unavailable := map[string]bool{
            "churn_rate":             true,
            "temporal_coupling_pct":  true,
            "author_fragmentation":   true,
            "commit_stability":       true,
            "hotspot_concentration":  true,
        }
        return map[string]float64{}, unavailable
    }

    return map[string]float64{
        "churn_rate":             m.ChurnRate,
        "temporal_coupling_pct":  m.TemporalCouplingPct,
        "author_fragmentation":   m.AuthorFragmentation,
        "commit_stability":       m.CommitStability,
        "hotspot_concentration":  m.HotspotConcentration,
    }, nil
}
```

### Pipeline Integration
```go
// In pipeline.go New() function, add to analyzers slice:
analyzers: []Analyzer{
    analyzer.NewC1Analyzer(tsParser),
    c2Analyzer,
    analyzer.NewC3Analyzer(tsParser),
    analyzer.NewC5Analyzer(), // NEW: no tsParser needed
    analyzer.NewC6Analyzer(tsParser),
},
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| go-git library | Native git CLI via os/exec | Project decision | 10-100x faster for large repos |
| Full history analysis | Time-windowed (6-month default, 90-day for some metrics) | CodeScene best practice | Meets 30-second budget |
| All commits included | Skip merge commits + large commits (>50 files) | CodeScene algorithm | Reduces false positives |

**Key performance insight:** The `--since` flag is handled by git internally and is very efficient -- it uses the commit graph to avoid walking ancient history. Combined with `--no-merges`, this dramatically reduces the data volume for parsing.

## Open Questions

1. **Bot author filtering**
   - What we know: Some repos have bot commits (dependabot, renovate) that inflate metrics
   - What's unclear: Whether to filter known bot patterns from author fragmentation
   - Recommendation: For v1, don't filter bots. Add as future enhancement. Bot commits are usually small and won't significantly skew churn or coupling metrics.

2. **Mean Time to Restore (C5 PRD metric)**
   - What we know: PRD lists MTTR as a C5 metric, but requirements C5-01 through C5-08 don't include it
   - What's unclear: Whether MTTR should be implemented
   - Recommendation: Skip MTTR -- it requires CI/CD integration data that's out of scope for git-only analysis. The 5 implemented metrics (churn, coupling, fragmentation, stability, hotspots) cover the git forensics domain completely.

3. **Monorepo handling**
   - What we know: Monorepos may have many files per commit naturally
   - What's unclear: Whether the 50-file commit threshold should be configurable
   - Recommendation: Use 50 as default (matching CodeScene). Could be made configurable via `.arsrc.yml` in a future phase.

## Sources

### Primary (HIGH confidence)
- Existing codebase: `internal/analyzer/c6_testing.go`, `internal/scoring/scorer.go`, `internal/scoring/config.go` -- verified analyzer pattern, extractor pattern, breakpoint configuration
- Existing codebase: `pkg/types/types.go`, `pkg/types/scoring.go` -- verified metric struct pattern, scoring types
- Existing codebase: `internal/pipeline/pipeline.go` -- verified pipeline integration point
- Git documentation: `--pretty=format`, `--numstat`, `--since`, `--no-merges` flags verified via local git execution

### Secondary (MEDIUM confidence)
- [CodeScene Temporal Coupling Docs (v3.4.0)](https://docs.enterprise.codescene.io/versions/3.4.0/guides/technical/temporal-coupling.html) -- algorithm thresholds (50 file max, 10 commit minimum, 70% strength threshold)
- [DoltHub: Go os/exec patterns](https://www.dolthub.com/blog/2022-11-28-go-os-exec-patterns/) -- streaming stdout with bufio.Scanner
- [Git Log Documentation](https://git-scm.com/docs/git-log) -- `--numstat` output format, date filtering

### Tertiary (LOW confidence)
- PRD C5 breakpoint values (Score 10/1 thresholds) -- reasonable but need validation against real repos

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- all stdlib, no external dependencies needed
- Architecture: HIGH -- directly follows existing C1/C2/C3/C6 analyzer pattern in codebase
- Git parsing: HIGH -- verified output format via local execution, well-documented git flags
- Metric algorithms: MEDIUM -- based on CodeScene docs + PRD requirements, formulas are standard but breakpoint calibration may need tuning
- Pitfalls: HIGH -- common issues well-documented across sources

**Research date:** 2026-02-01
**Valid until:** 2026-03-01 (stable domain, unlikely to change)
