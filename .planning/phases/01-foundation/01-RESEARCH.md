# Phase 1: Foundation - Research

**Researched:** 2026-01-31
**Domain:** Go CLI tool -- file discovery, classification, and pipeline architecture
**Confidence:** HIGH

## Summary

Phase 1 builds the CLI skeleton (`ars scan <dir>`), file discovery engine, Go file classifier, and the stub pipeline architecture that later phases extend. The primary technical challenges are: (1) correct Go file classification (test, generated, platform-specific, vendor), (2) respecting `.gitignore` patterns during directory traversal, (3) setting up the Cobra CLI with proper argument validation and error handling, and (4) establishing the pipeline architecture that `go/packages` will power in Phase 2.

The standard approach uses `spf13/cobra` for CLI scaffolding, `filepath.WalkDir` for filesystem traversal (fast, no extra stat calls), `sabhiram/go-gitignore` for gitignore pattern matching, `fatih/color` for TTY-aware colored output, and Go's standard `regexp` for generated code detection. The pipeline is structured as Discover -> Parse (stub) -> Analyze (stub) -> Output, with clear interface boundaries so Phase 2 can plug in real parsers and analyzers.

A key architectural decision from the roadmap is "use `go/packages` from day one." For Phase 1, this means designing the pipeline interfaces around `go/packages` data types, but the actual file discovery still uses filesystem walking (needed for gitignore exclusion, vendor detection, and project validation before `go/packages` is invoked). The `go/packages` integration becomes the parsing layer -- stub in Phase 1, real in Phase 2.

**Primary recommendation:** Build a filesystem walker that discovers and classifies Go files, wire it through a pipeline with stub parse/analyze stages, and output summary counts with colored TTY-aware formatting.

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `spf13/cobra` | v1.10.2 | CLI framework (commands, flags, help, version) | De facto standard for Go CLIs. 184k+ importers. Used by kubectl, hugo, gh. Provides `--help`, `--version`, subcommands, arg validation out of the box. |
| `fatih/color` | v1.18.0 | TTY-aware colored terminal output | Auto-disables on non-TTY (piped output). Honors `NO_COLOR` env var. Simple API. 7k+ stars. |
| `filepath.WalkDir` | stdlib (Go 1.16+) | Directory traversal | Faster than `filepath.Walk` (no extra `os.Stat` per entry). Standard library, zero dependencies. |
| `regexp` | stdlib | Generated code detection pattern matching | Standard library. The `// Code generated .* DO NOT EDIT\.$` pattern is a simple regex. |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `sabhiram/go-gitignore` | v1.0.2 | Parse and match `.gitignore` patterns | During file discovery to skip gitignored files/dirs. Supports `**` patterns, negation. 1,251 dependents. |
| `go/parser` | stdlib | Parse Go files for build tag extraction | Reading `//go:build` directives from file headers to classify build-constrained files. |
| `go/build` | stdlib | `MatchFile` for platform-specific file detection | Classifying `_linux.go`, `_amd64.go` suffix files using `go/build.Default.MatchFile()`. |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `sabhiram/go-gitignore` | `go-git/go-git` gitignore package | go-git's gitignore is more complete but pulls in the entire go-git dependency tree (~heavy). sabhiram is lightweight and sufficient. |
| `sabhiram/go-gitignore` | `denormal/go-gitignore` | denormal has better fnmatch compliance but fewer users. sabhiram is more battle-tested with 1,251 dependents. |
| `filepath.WalkDir` | `charlievieth/fastwalk` | fastwalk provides parallel traversal (~4x faster) but adds a dependency. Not needed for Phase 1; consider in Phase 5 (Hardening) if performance is an issue. |
| `fatih/color` | Raw ANSI codes | fatih/color handles TTY detection, NO_COLOR, and Windows automatically. Not worth hand-rolling. |
| `spf13/cobra` | stdlib `flag` | Cobra provides subcommand support, auto-generated help, version flag, and shell completion. ARS uses `ars scan <dir>` which is a subcommand pattern. |

**Installation:**
```bash
go mod init github.com/yourorg/ars

go get github.com/spf13/cobra@v1.10.2
go get github.com/fatih/color@v1.18.0
go get github.com/sabhiram/go-gitignore@latest
```

## Architecture Patterns

### Recommended Project Structure (Phase 1)

```
ars/
├── main.go                    # Entry point: cmd.Execute()
├── cmd/
│   ├── root.go                # Root cobra command, --version, --verbose flags
│   └── scan.go                # `ars scan <dir>` command, arg validation
├── internal/
│   ├── discovery/
│   │   ├── walker.go          # Filesystem traversal, gitignore, vendor exclusion
│   │   ├── classifier.go      # Go file classification (test, generated, platform)
│   │   └── walker_test.go
│   ├── pipeline/
│   │   ├── pipeline.go        # Pipeline orchestrator: discover -> parse -> analyze -> output
│   │   ├── interfaces.go      # Parser, Analyzer interfaces (stubs in Phase 1)
│   │   └── pipeline_test.go
│   └── output/
│       ├── terminal.go        # TTY-aware summary output with color
│       └── terminal_test.go
├── pkg/
│   └── types/
│       └── types.go           # Shared types: DiscoveredFile, ScanResult, FileClass
└── testdata/
    ├── valid-go-project/      # Has go.mod, .go files, _test.go files
    ├── non-go-project/        # No go.mod, no .go files
    ├── with-vendor/           # Has vendor/ directory
    ├── with-generated/        # Has generated code files
    └── with-gitignore/        # Has .gitignore with patterns
```

### Pattern 1: File Discovery with Classification

**What:** Walk the filesystem, classify each `.go` file by type (source, test, generated, platform-specific), and exclude vendor/gitignored paths. Return a structured result with counts and file lists.

**When to use:** Phase 1 discovery stage. This runs before any parsing.

**Example:**
```go
// internal/discovery/classifier.go

type FileClass int

const (
    ClassSource     FileClass = iota // Regular Go source file
    ClassTest                        // _test.go file
    ClassGenerated                   // Contains "Code generated" header
    ClassExcluded                    // Vendor, gitignored, etc.
)

type DiscoveredFile struct {
    Path         string
    RelPath      string    // Relative to scan root
    Class        FileClass
    ExcludeReason string   // Why excluded (vendor, gitignore, generated)
}
```

### Pattern 2: Pipeline with Stub Stages

**What:** Define the full pipeline interface (Discover -> Parse -> Analyze -> Output) but implement only Discovery and Output in Phase 1. Parse and Analyze are stubs that pass through data unchanged.

**When to use:** Phase 1. Establishes the architecture for Phase 2 to fill in.

**Example:**
```go
// internal/pipeline/interfaces.go

// Parser parses discovered files into ASTs (stub in Phase 1)
type Parser interface {
    Parse(files []types.DiscoveredFile) ([]types.ParsedFile, error)
}

// Analyzer runs analysis on parsed files (stub in Phase 1)
type Analyzer interface {
    Name() string
    Analyze(files []types.ParsedFile) (*types.AnalysisResult, error)
}

// StubParser passes through files without parsing (Phase 1)
type StubParser struct{}

func (s *StubParser) Parse(files []types.DiscoveredFile) ([]types.ParsedFile, error) {
    result := make([]types.ParsedFile, len(files))
    for i, f := range files {
        result[i] = types.ParsedFile{Path: f.Path, Class: f.Class}
    }
    return result, nil
}
```

### Pattern 3: Cobra Subcommand with Positional Arg

**What:** Structure the CLI as `ars scan <directory>` with exactly one required positional argument.

**Example:**
```go
// cmd/scan.go

var scanCmd = &cobra.Command{
    Use:   "scan <directory>",
    Short: "Scan a Go project for agent readiness",
    Long:  `Discovers and classifies all Go source files in the specified directory.`,
    Args:  cobra.ExactArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        dir := args[0]
        verbose, _ := cmd.Flags().GetBool("verbose")

        // Validate directory exists and is a Go project
        if err := validateGoProject(dir); err != nil {
            return err
        }

        // Run pipeline
        p := pipeline.New(verbose)
        result, err := p.Run(dir)
        if err != nil {
            return err
        }

        // Output results
        output.RenderSummary(cmd.OutOrStdout(), result, verbose)
        return nil
    },
}
```

### Anti-Patterns to Avoid

- **Mixing discovery and parsing:** Keep filesystem walking separate from Go parsing. Discovery produces file paths and classifications; parsing (Phase 2) produces ASTs. Do not parse files during the walk.
- **Hardcoding exclusion patterns:** Use a config struct for excluded directories (`vendor`, `.git`, `testdata`, `node_modules`). Makes it testable and extensible.
- **Ignoring the error return in WalkDir:** Always handle `fs.SkipDir` returns properly. Return `fs.SkipDir` from the walk function to skip entire directory subtrees (vendor, .git), not just individual files.
- **Using `filepath.Walk` instead of `filepath.WalkDir`:** Walk calls `os.Stat` on every entry. WalkDir uses `fs.DirEntry` which avoids the extra stat call. Measurably faster on large repos.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| CLI arg parsing, help text, version flag | Custom flag parsing | `spf13/cobra` | Help formatting, subcommands, shell completion, version injection, arg validators -- all solved. |
| Gitignore pattern matching | Custom glob matcher | `sabhiram/go-gitignore` | Gitignore semantics are surprisingly complex: negation patterns (`!`), directory-only patterns (`dir/`), nested `.gitignore` files, `**` recursive globs. |
| TTY detection for color | `os.Getenv("TERM")` checks | `fatih/color` | Handles TTY detection via `go-isatty`, `NO_COLOR` env var, Windows console, and programmatic disable. |
| Generated code detection regex | Simple string contains | `regexp.MustCompile("^// Code generated .* DO NOT EDIT\\.$")` | The official Go convention (golang/go#13560) specifies a regex pattern. String contains would match comments in the middle of files. |
| Platform-specific file detection | Parsing `_linux`, `_amd64` suffixes manually | `go/build.Default.MatchFile(dir, name)` | The `go/build` package knows all valid GOOS/GOARCH values and handles the suffix stripping logic (including `_test` suffix interaction). |

**Key insight:** File discovery and classification look simple but have many edge cases. Gitignore semantics alone have ~15 edge cases. Platform-specific file classification interacts with `_test.go` suffixes in non-obvious ways. Use proven libraries.

## Common Pitfalls

### Pitfall 1: Not Validating Go Project Before Walking

**What goes wrong:** The tool walks a non-Go directory (e.g., `/usr/bin`) and either produces confusing empty output or takes a long time traversing irrelevant files.
**Why it happens:** Developers focus on the happy path (valid Go project) and forget the error case.
**How to avoid:** Before starting the filesystem walk, check for `go.mod` presence OR the existence of `.go` files in the root or immediate subdirectories. Fail fast with an actionable error message: "No Go project found at /path. Expected go.mod file or .go source files."
**Warning signs:** Users report "tool hangs" or "empty output" when pointed at wrong directory.

### Pitfall 2: Generated Code Detection Only Checking First Line

**What goes wrong:** The `// Code generated ... DO NOT EDIT.` comment may appear after a copyright header or package comment, not on the first line.
**Why it happens:** The official Go spec says the comment must appear "before the first non-comment, non-blank text" but does NOT require it to be the first line. Copyright headers commonly precede it.
**How to avoid:** Scan all comment lines before the `package` declaration, not just the first line. Use `go/parser.ParseFile` with `parser.ParseComments` mode and check `ast.File.Comments` for the pattern, OR scan raw bytes up to the `package` keyword.
**Warning signs:** Generated protobuf files with copyright headers are incorrectly classified as source.

### Pitfall 3: Gitignore Path Relativity

**What goes wrong:** Gitignore patterns use paths relative to the `.gitignore` file location. Passing absolute paths or paths relative to a different root produces incorrect matches.
**Why it happens:** `filepath.WalkDir` provides paths relative to the walk root, but `.gitignore` expects paths relative to the gitignore file's directory. In nested `.gitignore` files, this gets especially confusing.
**How to avoid:** Always convert walk paths to be relative to the directory containing the `.gitignore` before matching. For Phase 1, support only root-level `.gitignore` (same directory as the scan target). Document that nested `.gitignore` support is deferred.
**Warning signs:** Files that should be ignored are not, or files that should be included are incorrectly excluded.

### Pitfall 4: Build Tag Files That Don't Compile on Current Platform

**What goes wrong:** Files with `//go:build linux` on macOS are still discovered and classified but cannot be parsed by `go/packages` (which respects build constraints). This creates inconsistency between discovery counts and later parsing counts.
**Why it happens:** Filesystem walking sees all files regardless of build constraints. `go/packages` only sees files matching the current GOOS/GOARCH.
**How to avoid:** For Phase 1 (discovery only), classify build-constrained files but include them in counts. Note the build constraint in the file metadata. When `go/packages` is integrated in Phase 2, reconcile the difference. The discovery count is "all Go files in the repo" while `go/packages` gives "files that compile on this platform."
**Warning signs:** File counts change between discovery and parsing phases.

### Pitfall 5: Symlink Cycles in Directory Walking

**What goes wrong:** The CONTEXT.md says "follow symlinks." Following symlinks with `filepath.WalkDir` can cause infinite loops if a symlink points to a parent directory.
**Why it happens:** `filepath.WalkDir` does NOT follow symlinks by default. If you add custom symlink following, you must handle cycles.
**How to avoid:** Track visited directories by device+inode (on Unix) or by resolved absolute path. If a symlinked directory resolves to an already-visited path, skip it. Use `os.Stat` (follows symlinks) on directory entries and maintain a visited set.
**Warning signs:** Tool hangs on repos with circular symlinks. OOM from infinite directory expansion.

### Pitfall 6: Exit Code 2 Conflicts

**What goes wrong:** FOUND-06 specifies exit code 2 for "below threshold." But `--threshold` is not a Phase 1 feature (it's Phase 4). If Phase 1 uses exit code 2 for something else, it creates a conflict.
**How to avoid:** In Phase 1, only use exit codes 0 (success) and 1 (error). Reserve exit code 2 for Phase 4's threshold feature. Document the exit code contract in code.
**Warning signs:** CI scripts break when exit codes change between phases.

## Code Examples

### Go Project Validation

```go
// Source: Go convention (go.mod presence)
func validateGoProject(dir string) error {
    // Check directory exists
    info, err := os.Stat(dir)
    if err != nil {
        if os.IsNotExist(err) {
            return fmt.Errorf("directory not found: %s", dir)
        }
        return fmt.Errorf("cannot access %s: %w", dir, err)
    }
    if !info.IsDir() {
        return fmt.Errorf("not a directory: %s", dir)
    }

    // Check for go.mod
    goModPath := filepath.Join(dir, "go.mod")
    if _, err := os.Stat(goModPath); err == nil {
        return nil // go.mod found, valid Go project
    }

    // Fallback: check for any .go files
    entries, err := os.ReadDir(dir)
    if err != nil {
        return fmt.Errorf("cannot read directory %s: %w", dir, err)
    }
    for _, e := range entries {
        if !e.IsDir() && strings.HasSuffix(e.Name(), ".go") {
            return nil // .go file found
        }
    }

    return fmt.Errorf("not a Go project: %s\nNo go.mod file or .go source files found. "+
        "Please specify a directory containing a Go project.", dir)
}
```

### Generated Code Detection

```go
// Source: golang/go#13560 -- official generated code convention
var generatedPattern = regexp.MustCompile(`^// Code generated .* DO NOT EDIT\.$`)

func isGeneratedFile(path string) (bool, error) {
    f, err := os.Open(path)
    if err != nil {
        return false, err
    }
    defer f.Close()

    scanner := bufio.NewScanner(f)
    for scanner.Scan() {
        line := scanner.Text()
        // Stop at package declaration (generated comment must appear before it)
        if strings.HasPrefix(line, "package ") {
            return false, nil
        }
        if generatedPattern.MatchString(line) {
            return true, nil
        }
    }
    return false, scanner.Err()
}
```

### File Classification

```go
// Source: go/build package documentation
func classifyGoFile(dir, name string) FileClass {
    // Test files
    if strings.HasSuffix(name, "_test.go") {
        return ClassTest
    }

    // Files starting with _ or . are ignored by Go tool
    if strings.HasPrefix(name, "_") || strings.HasPrefix(name, ".") {
        return ClassExcluded
    }

    return ClassSource
}
```

### Directory Walker with Gitignore and Vendor Exclusion

```go
// Source: filepath.WalkDir + sabhiram/go-gitignore
func (w *Walker) Discover(rootDir string) ([]DiscoveredFile, error) {
    var files []DiscoveredFile

    // Load .gitignore if present
    gitignorePath := filepath.Join(rootDir, ".gitignore")
    var ignorer *ignore.GitIgnore
    if _, err := os.Stat(gitignorePath); err == nil {
        ignorer, err = ignore.CompileIgnoreFile(gitignorePath)
        if err != nil {
            return nil, fmt.Errorf("parse .gitignore: %w", err)
        }
    }

    // Directories to always skip
    skipDirs := map[string]bool{
        "vendor":      true,
        ".git":        true,
        "node_modules": true,
        "testdata":    true,
    }

    err := filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
        if err != nil {
            return err // Permission error: fail immediately (per CONTEXT.md)
        }

        relPath, _ := filepath.Rel(rootDir, path)

        // Skip hidden and excluded directories
        if d.IsDir() {
            if skipDirs[d.Name()] {
                return fs.SkipDir
            }
            if strings.HasPrefix(d.Name(), ".") && d.Name() != "." {
                return fs.SkipDir
            }
            return nil
        }

        // Only process .go files
        if !strings.HasSuffix(d.Name(), ".go") {
            return nil
        }

        // Check gitignore
        if ignorer != nil && ignorer.MatchesPath(relPath) {
            files = append(files, DiscoveredFile{
                Path: path, RelPath: relPath,
                Class: ClassExcluded, ExcludeReason: "gitignore",
            })
            return nil
        }

        // Classify the file
        class := classifyGoFile(filepath.Dir(path), d.Name())
        files = append(files, DiscoveredFile{
            Path: path, RelPath: relPath, Class: class,
        })

        return nil
    })

    return files, err
}
```

### Cobra Root Command with Version

```go
// cmd/root.go
var (
    version = "dev" // Set via -ldflags at build time
    verbose bool
)

var rootCmd = &cobra.Command{
    Use:   "ars",
    Short: "Agent Readiness Score - analyze Go codebases for AI agent compatibility",
    Long: `ARS scans Go codebases and produces a composite score (1-10) measuring
how well the repository supports AI agent workflows.`,
    Version: version,
}

func init() {
    rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false,
        "Show detailed output including discovered files and exclusion reasons")
}

func Execute() {
    if err := rootCmd.Execute(); err != nil {
        os.Exit(1)
    }
}
```

### TTY-Aware Output

```go
// internal/output/terminal.go
func RenderSummary(w io.Writer, result *types.ScanResult, verbose bool) {
    bold := color.New(color.Bold)
    green := color.New(color.FgGreen)
    yellow := color.New(color.FgYellow)

    bold.Fprintf(w, "\nARS Scan: %s\n", result.RootDir)
    fmt.Fprintf(w, "─────────────────────────────\n")
    fmt.Fprintf(w, "Go files discovered: %d\n", result.TotalFiles)
    green.Fprintf(w, "  Source files:      %d\n", result.SourceCount)
    yellow.Fprintf(w, "  Test files:        %d\n", result.TestCount)
    if result.GeneratedCount > 0 {
        fmt.Fprintf(w, "  Generated (excluded): %d\n", result.GeneratedCount)
    }
    if result.VendorCount > 0 {
        fmt.Fprintf(w, "  Vendor (excluded):    %d\n", result.VendorCount)
    }
    if result.GitignoreCount > 0 {
        fmt.Fprintf(w, "  Gitignored (excluded): %d\n", result.GitignoreCount)
    }

    if verbose {
        // List individual files with their classification
        fmt.Fprintf(w, "\nDiscovered files:\n")
        for _, f := range result.Files {
            fmt.Fprintf(w, "  [%s] %s\n", f.Class, f.RelPath)
        }
    }
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `filepath.Walk` | `filepath.WalkDir` | Go 1.16 (Feb 2021) | ~1.5x faster; no extra stat calls. Always use WalkDir. |
| `// +build` tag syntax | `//go:build` directive | Go 1.17 (Aug 2021) | New syntax is boolean expressions. `go fix` converts automatically. Support both when reading. |
| `go/build` for package loading | `golang.org/x/tools/go/packages` | ~2019 | `go/packages` handles modules correctly. `go/build` is legacy. ARS will use `go/packages` in Phase 2. |
| `go list -json` for file discovery | `go/packages` with `NeedFiles` mode | ~2019 | Programmatic API. `NeedName \| NeedFiles` mode is fast, no network needed. |

**Deprecated/outdated:**
- `filepath.Walk`: Still works but `WalkDir` is strictly better. No reason to use Walk.
- `// +build` syntax: Still accepted but `//go:build` is the standard. gofmt auto-adds `// +build` for backwards compat.

## Open Questions

1. **Symlink following implementation details**
   - What we know: CONTEXT.md says "follow symlinks." `filepath.WalkDir` does NOT follow symlinks by default.
   - What's unclear: The exact cycle detection strategy. Should we use device+inode or resolved paths?
   - Recommendation: Use `os.Stat` (follows symlinks) on directory entries. Maintain a `map[string]bool` of resolved absolute paths. Skip directories already visited. This is simpler than device+inode and portable across platforms.

2. **Nested `.gitignore` support**
   - What we know: Real Git repos can have `.gitignore` files in subdirectories. `sabhiram/go-gitignore` loads one file at a time.
   - What's unclear: Whether Phase 1 needs nested gitignore support.
   - Recommendation: Support only root-level `.gitignore` in Phase 1. Defer nested gitignore to Phase 5 (Hardening). Most Go projects have a single root `.gitignore`.

3. **`go/packages` integration timing**
   - What we know: Roadmap says "use go/packages from day one." Phase 1 is discovery only.
   - What's unclear: Whether to actually call `go/packages.Load` in Phase 1 or just design interfaces for it.
   - Recommendation: Design the `Parser` interface to accept `go/packages` data types, but use a stub implementation in Phase 1 that does not call `go/packages.Load`. The filesystem walker provides discovery; `go/packages` provides parsing in Phase 2. This avoids requiring `go mod download` in Phase 1's pipeline.

4. **`testdata/` exclusion scope**
   - What we know: Go tooling ignores `testdata/` directories. CONTEXT.md lists it in "standard Go conventions."
   - What's unclear: Whether to exclude ALL `testdata/` directories at any depth, or only root-level.
   - Recommendation: Exclude `testdata/` at any depth (matching Go tool behavior). This is what `go build` does.

## Sources

### Primary (HIGH confidence)
- [spf13/cobra on pkg.go.dev](https://pkg.go.dev/github.com/spf13/cobra) -- v1.10.2, CLI framework API, arg validators
- [golang/go#13560](https://github.com/golang/go/issues/13560) -- Official generated code header convention
- [go/build package docs](https://pkg.go.dev/go/build) -- File naming conventions, build constraints, MatchFile
- [filepath.WalkDir docs](https://pkg.go.dev/path/filepath#WalkDir) -- Directory traversal API
- [fatih/color on GitHub](https://github.com/fatih/color) -- v1.18.0, TTY detection, NO_COLOR support
- [golang.org/x/tools/go/packages](https://pkg.go.dev/golang.org/x/tools/go/packages) -- NeedFiles/NeedSyntax modes

### Secondary (MEDIUM confidence)
- [sabhiram/go-gitignore](https://pkg.go.dev/github.com/sabhiram/go-gitignore) -- Gitignore pattern matching library, 1,251 dependents
- [Cobra user guide](https://github.com/spf13/cobra/blob/main/site/content/user_guide.md) -- Positional args, custom validators

### Tertiary (LOW confidence)
- Nested gitignore handling strategy -- Based on reasoning, not verified against a reference implementation

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- All libraries verified via pkg.go.dev, well-established in Go ecosystem
- Architecture: HIGH -- Pipeline pattern verified against golangci-lint, standard Go project layout
- File classification: HIGH -- Based on official Go specification and go/build documentation
- Gitignore handling: MEDIUM -- Library verified but integration pattern is based on reasoning
- Symlink handling: MEDIUM -- Strategy is sound but edge cases need testing against real repos

**Research date:** 2026-01-31
**Valid until:** 2026-03-01 (stable domain, libraries are mature)
