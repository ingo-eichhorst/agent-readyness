# Stack Research

**Domain:** Go CLI static analysis tool (codebase quality scorer)
**Researched:** 2026-01-31
**Confidence:** HIGH

## Recommended Stack

### Core Technologies

| Technology | Version | Purpose | Why Recommended |
|------------|---------|---------|-----------------|
| Go | 1.24+ (latest: 1.25.6) | Language runtime | Go 1.24 added tool directives in go.mod for managing tool dependencies. Go 1.25 is current stable. Target 1.24+ for broadest compatibility while getting modern features. |
| `go/ast` + `go/parser` + `go/token` | stdlib | AST parsing of Go source files | Standard library. Zero dependencies. Battle-tested. This is what every Go static analysis tool uses under the hood. No reason to use anything else. |
| `go/types` | stdlib | Type checking and type information | Required for resolving identifiers, understanding imports, and deeper semantic analysis beyond syntax. Part of the standard toolchain. |
| `golang.org/x/tools/go/packages` | v0.41.0+ | Package loading and dependency resolution | The official way to load Go packages with full type information and dependency graphs. Replaces the old `go/build` loader. Handles modules, build tags, and environments correctly. Essential for analyzing multi-package projects. |
| `golang.org/x/tools/go/ast/inspector` | (part of x/tools) | Optimized AST traversal | Faster than manual `ast.Walk` for analysis tools that check multiple node types. Pre-computes traversal information. Use this instead of rolling your own walker. |
| `spf13/cobra` | v1.10.2 | CLI framework | De facto standard for Go CLIs. Used by kubectl, hugo, gh. 184k+ importers. Provides subcommands, flags, help generation, shell completion. Well-maintained (released Dec 2024). |

### Supporting Libraries

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `fzipp/gocyclo` | v0.6.0+ | Cyclomatic complexity calculation | Use as a library (not CLI) via `gocyclo.AnalyzeASTFile()` to compute per-function complexity from already-parsed AST files. Avoids re-parsing. Lightweight, focused, BSD-licensed. |
| `golang.org/x/tools/cover` | (part of x/tools) | Coverage profile parsing | Use `cover.ParseProfiles()` to parse `go test -coverprofile` output files. Official Go tooling. Returns structured `Profile` and `ProfileBlock` data. |
| `golang.org/x/tools/refactor/importgraph` | (part of x/tools) | Import dependency graph construction | Builds forward and reverse import dependency graphs for all packages in a workspace. Use for circular dependency detection and coupling analysis. |
| `fatih/color` | v1.18.0 | Colored terminal output | Simple ANSI color output. Respects `NO_COLOR` env var. Auto-disables on non-TTY. 7k+ stars. Use for score display, warnings, pass/fail indicators. |
| `encoding/json` | stdlib | JSON output | For machine-readable output format. Use stdlib json -- no need for third-party JSON libraries for this use case. |

### Development Tools

| Tool | Purpose | Notes |
|------|---------|-------|
| `golangci-lint` v2.8.0 | Linting and code quality | Use v2 (major rewrite with simplified config). Install via `go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest`. Configure in `.golangci.yml`. |
| `go test -race -coverprofile` | Testing with race detection and coverage | Built into Go toolchain. Run with `-race` in CI. TDD approach means tests from day one. |
| `go vet` | Static analysis | Built-in. Catches common mistakes. Already integrated into `golangci-lint` but good to run standalone too. |
| `goreleaser` | Binary distribution | If/when distributing binaries. Cross-compilation for Linux/macOS/Windows. Not needed for MVP. |

## What to Build vs What to Import

**Critical architecture decision:** For a scoring tool that computes metrics, most analysis should be built on top of Go's standard `go/ast` + `go/types` + `go/packages` primitives rather than importing heavy analysis frameworks. Here is why:

1. **Cyclomatic complexity** -- Use `gocyclo` as a library. It already does AST-level complexity counting well. No need to reimplement.

2. **Coupling metrics (afferent/efferent)** -- Build this yourself using `go/packages` import graphs. The logic is straightforward: count inbound and outbound package imports. Existing tools like `spm-go` (68 stars, limited maintenance) are CLI-only and not designed as importable libraries. The computation is ~50 lines of code on top of `go/packages`.

3. **Code duplication** -- Use `mibk/dupl` as a library if its token-based approach is sufficient, or build a simpler heuristic (e.g., AST subtree hashing). For MVP, a line-count-based heuristic may be enough. `dupl` has not been updated since 2016 (v1.0.0) but the algorithm is sound and Go-native.

4. **Dead code detection** -- Use `golang.org/x/tools/cmd/deadcode` concepts (RTA algorithm via `go/ssa` and `go/callgraph`). For MVP, a simpler heuristic based on exported-but-unreferenced functions via import graph analysis is sufficient and avoids the SSA dependency.

5. **Test detection** -- Build this yourself. It is trivial: scan for `*_test.go` files, parse to find `Test*` functions. No library needed.

6. **Coverage parsing** -- Use `golang.org/x/tools/cover`. Official, well-maintained, does exactly this.

## Installation

```bash
# Initialize module
go mod init github.com/yourorg/ars

# Core dependencies
go get github.com/spf13/cobra@v1.10.2
go get golang.org/x/tools@latest
go get github.com/fzipp/gocyclo@latest
go get github.com/fatih/color@v1.18.0

# Duplication detection (optional, evaluate at implementation time)
go get github.com/mibk/dupl@latest

# Dev tools (go.mod tool directives, Go 1.24+)
go get -tool github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
```

## Alternatives Considered

| Recommended | Alternative | When to Use Alternative |
|-------------|-------------|-------------------------|
| `spf13/cobra` | `urfave/cli` | If you prefer a flatter API without subcommands. Cobra is better for our case because `ars scan`, `ars report`, etc. map naturally to subcommands. |
| `spf13/cobra` | No framework (just `flag` stdlib) | If the CLI is truly single-command with 2-3 flags. ARS is simple enough that this could work for MVP, but Cobra future-proofs for subcommands at negligible cost. |
| `go/packages` | `go/build` | Never. `go/build` is the old API. `go/packages` is the modern replacement that handles modules correctly. |
| `fatih/color` | `charmbracelet/lipgloss` | If you need rich TUI layouts, tables, or interactive terminal UI. Lipgloss v2 is in alpha (unstable API). For simple colored text output, `fatih/color` is simpler and stable. |
| `fatih/color` | No color (plain `fmt`) | If machine-readable output is primary. We want human-friendly terminal output, so color is worth the small dependency. |
| `fzipp/gocyclo` | `ichiban/cyclomatic` | If you want integration with `go/analysis` framework. For our use case (direct AST analysis), gocyclo's library API is simpler. |
| Build coupling metrics | `fdaines/spm-go` | Never as a library -- spm-go is CLI-only, 68 stars, not designed for programmatic use. Build the ~50 lines yourself. |
| `mibk/dupl` | `kucherenko/jscpd` | If analyzing multi-language codebases. jscpd is Node.js-based (external dependency). Since ARS analyzes Go only, dupl or a custom solution is better. |

## What NOT to Use

| Avoid | Why | Use Instead |
|-------|-----|-------------|
| `charmbracelet/bubbletea` | Full TUI framework -- massive overkill for a CLI that prints a score report. Adds complexity, interactive mode we do not need. | `fmt.Println` + `fatih/color` for output |
| `charmbracelet/lipgloss` v2 | API still in alpha (v2.0.0-alpha.2). Unstable for production use. Also overkill for colored text output. | `fatih/color` v1.18.0 (stable) |
| `spf13/viper` | Config file management framework. ARS does not need config files for MVP. Flags and env vars via Cobra are sufficient. Viper adds heavy transitive dependencies. | Cobra's built-in flag handling |
| `go/build` | Deprecated package loading API. Does not handle Go modules correctly. | `golang.org/x/tools/go/packages` |
| `go/ssa` (for MVP) | SSA intermediate representation is powerful but complex. Needed for precise dead code analysis (RTA) but overkill for MVP scoring heuristics. | Simple import graph analysis + unexported function detection for MVP. Add SSA in a later phase if precision matters. |
| `golangci-lint` as a library | golangci-lint is designed as a CLI tool, not an importable library. Do not try to embed it. | Use individual analysis packages directly (`gocyclo`, `go/analysis`, etc.) |
| External linters as subprocesses | Shelling out to `go vet`, `golangci-lint`, etc. makes the tool fragile, slow, and hard to test. | Use Go's analysis packages programmatically for all metrics |

## Stack Patterns by Variant

**If targeting Go 1.25+ only:**
- Use `go tool` directives (stabilized in 1.24) for dev dependencies
- Can use experimental `encoding/json/v2` via `GOEXPERIMENT=jsonv2` for better JSON output (but probably not worth the risk)

**If maximum repo compatibility needed:**
- Target Go 1.22+ minimum (two versions back from current)
- Avoid 1.24+ features like generic type aliases
- `go/packages` works fine on 1.22+

**If performance becomes an issue on very large repos (50k+ files):**
- Use `go/ast/inspector` for batched AST traversal (already recommended)
- Consider `golang.org/x/sync/errgroup` for parallel file processing
- Profile before optimizing -- `go/packages` loading is usually the bottleneck, not analysis

## Version Compatibility

| Package | Compatible With | Notes |
|---------|-----------------|-------|
| `spf13/cobra` v1.10.2 | Go 1.18+ | Cobra supports older Go versions. No issues. |
| `golang.org/x/tools` v0.41.0 | Go 1.23+ | x/tools typically requires N-2 Go versions. Check go.mod. |
| `fzipp/gocyclo` v0.6.0 | Go 1.18+ | Minimal dependencies. Uses only `go/ast` and `go/token`. |
| `fatih/color` v1.18.0 | Go 1.17+ | Very stable, minimal dependencies. |
| `mibk/dupl` v1.0.0 | Go 1.11+ | Old but functional. Uses `go/ast` internals. Test before relying on it. |

**Recommended minimum Go version for ARS:** Go 1.24+. This gets us tool directives in go.mod and is the current supported release (1.24.12 and 1.25.6 are both receiving security patches as of Jan 2026).

## Dependency Count Philosophy

ARS should have **minimal dependencies**. The recommended stack adds only:

- `spf13/cobra` (CLI framework -- 2 transitive deps: `spf13/pflag`, `inconshreveable/mousetrap`)
- `golang.org/x/tools` (official Go extended stdlib -- already needed for `go/packages`)
- `fzipp/gocyclo` (single-purpose, ~0 transitive deps)
- `fatih/color` (1 transitive dep: `mattn/go-isatty`)
- `mibk/dupl` (optional, ~0 transitive deps)

Total: ~5 direct dependencies, ~8 transitive. This is very lean for a Go project.

## Sources

- [spf13/cobra on pkg.go.dev](https://pkg.go.dev/github.com/spf13/cobra) -- v1.10.2, 184k+ importers (HIGH confidence)
- [cobra releases on GitHub](https://github.com/spf13/cobra/releases) -- v1.10.2 released Dec 4, 2024 (HIGH confidence)
- [golang.org/x/tools on pkg.go.dev](https://pkg.go.dev/golang.org/x/tools) -- v0.41.0, published Jan 12, 2026 (HIGH confidence)
- [go/analysis framework docs](https://pkg.go.dev/golang.org/x/tools/go/analysis) -- official analysis framework (HIGH confidence)
- [go/packages docs](https://pkg.go.dev/golang.org/x/tools/go/packages) -- package loading API (HIGH confidence)
- [golang.org/x/tools/cover](https://pkg.go.dev/golang.org/x/tools/cover) -- coverage profile parsing (HIGH confidence)
- [fzipp/gocyclo on pkg.go.dev](https://pkg.go.dev/github.com/fzipp/gocyclo) -- cyclomatic complexity (HIGH confidence)
- [fzipp/gocyclo on GitHub](https://github.com/fzipp/gocyclo) -- library API with AnalyzeASTFile (HIGH confidence)
- [mibk/dupl on GitHub](https://github.com/mibk/dupl) -- code clone detection, v1.0.0, last updated 2016 (MEDIUM confidence -- old but functional)
- [fatih/color on GitHub](https://github.com/fatih/color) -- v1.18.0, colored terminal output (HIGH confidence)
- [Go deadcode blog post](https://go.dev/blog/deadcode) -- official dead code detection approach using RTA (HIGH confidence)
- [fdaines/spm-go on GitHub](https://github.com/fdaines/spm-go) -- coupling metrics tool, 68 stars (LOW confidence -- small project, CLI-only)
- [golang.org/x/tools/refactor/importgraph](https://pkg.go.dev/golang.org/x/tools/refactor/importgraph) -- import graph construction (HIGH confidence)
- [Go 1.25 release notes](https://go.dev/doc/go1.25) -- current Go version features (HIGH confidence)
- [Go 1.24 release notes](https://go.dev/doc/go1.24) -- tool directives in go.mod (HIGH confidence)
- [golangci-lint v2](https://golangci-lint.run/) -- v2.8.0 released Jan 7, 2026 (HIGH confidence)
- [golangci-lint v2 announcement](https://ldez.github.io/blog/2025/03/23/golangci-lint-v2/) -- revamped config structure (MEDIUM confidence)

---
*Stack research for: ARS (Agent Readiness Score) -- Go CLI static analysis tool*
*Researched: 2026-01-31*
