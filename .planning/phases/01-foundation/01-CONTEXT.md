# Phase 1: Foundation - Context

**Gathered:** 2026-01-31
**Status:** Ready for planning

<domain>
## Phase Boundary

Build a CLI tool that discovers and classifies Go source files in a repository, validates the input path, and reports file counts (source vs test). Establish the scanning pipeline architecture that later phases will extend with analysis capabilities.

</domain>

<decisions>
## Implementation Decisions

### CLI Interface Design
- Command structure: Single command `ars <path>` (no subcommands in Phase 1)
- Path argument: Required positional argument, no default to current directory
- Core flags: `--help`, `--version`, and `--verbose` for detailed output
- Help text style: Concise, Go-style (terse and technical, like `go tool help`)

### File Discovery Behavior
- Auto-exclusions: Standard Go conventions only (vendor/, testdata/, hidden directories like .git)
- Symlink handling: Follow symlinks (traverse into symlinked directories)
- .gitignore: Respect .gitignore patterns (skip ignored files)
- Test detection: Claude's discretion (choose between _test.go suffix only OR include *_test packages)

### Error Messaging
- Non-Go path validation: Claude's discretion (choose between fast fail or guided error messages)
- Validation error style: Simple error lines (one line per error, no categorization)
- Permission errors: Fail immediately on first permission error (don't produce incomplete results)
- Error verbosity: Include context by default (show what failed + why, e.g., "cannot read /path: permission denied")

### Output Format
- Discovery progress: Show summary counts as files are discovered (reassure user work is happening)
- Final summary: Include counts + exclusion stats (Total/Source/Tests + vendor/gitignore exclusions)
- Verbose mode: Show discovered files + exclusion reasons (why things were skipped)
- Formatting: Detect TTY and colorize output with ANSI colors (plain text when piped)

### Claude's Discretion
- Test file detection logic (choose most robust approach for Go testing patterns)
- Exact wording of error messages for non-Go paths
- Progress display implementation details
- Exit code conventions

</decisions>

<specifics>
## Specific Ideas

No specific requirements — open to standard Go CLI patterns and conventions.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 01-foundation*
*Context gathered: 2026-01-31*
