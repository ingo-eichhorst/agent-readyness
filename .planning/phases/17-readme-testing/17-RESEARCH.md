# Phase 17: README & Testing - Research

**Researched:** 2026-02-04
**Domain:** README badges, Go test coverage documentation
**Confidence:** HIGH

## Summary

This phase adds standard status badges to the README and ensures test documentation includes coverage profiling. The research confirms established patterns for Go project badges using shields.io and pkg.go.dev, with specific syntax verified from official sources and high-profile Go projects (Gin, Cobra).

Key finding: The current documentation uses `-coverprofile=coverage.out` but the C6 analyzer expects `cover.out`. This must be corrected to enable C6 self-analysis.

Badge placement follows the user-decided order: Go Reference, Go Report Card, License, Release - all inline after the H1 title, using flat style.

**Primary recommendation:** Use verified badge syntax from official sources (pkg.go.dev, goreportcard.com, shields.io), fix coverage filename to `cover.out` for C6 compatibility.

## Standard Stack

This phase requires no libraries - only markdown syntax and Go tooling.

### Badge Services
| Service | URL Pattern | Purpose | Why Standard |
|---------|-------------|---------|--------------|
| pkg.go.dev | `pkg.go.dev/badge/{module}.svg` | Go Reference badge | Official Go package registry |
| goreportcard.com | `goreportcard.com/badge/{repo}` | Code quality badge | De facto standard for Go quality |
| shields.io | `img.shields.io/github/{type}/{owner}/{repo}` | License & Release badges | Industry standard badge service |

### Go Coverage Tools
| Tool | Command | Purpose |
|------|---------|---------|
| go test | `go test -coverprofile=cover.out ./...` | Generate coverage profile |
| go tool cover | `go tool cover -html=cover.out` | View coverage report |

## Architecture Patterns

### Badge Placement Pattern

**What:** Badges immediately after H1 title, single inline row
**When to use:** All Go projects
**Example:**
```markdown
# Project Name

[![Go Reference](https://pkg.go.dev/badge/github.com/owner/repo.svg)](https://pkg.go.dev/github.com/owner/repo)
[![Go Report Card](https://goreportcard.com/badge/github.com/owner/repo)](https://goreportcard.com/report/github.com/owner/repo)
[![License](https://img.shields.io/github/license/owner/repo)](https://github.com/owner/repo/blob/main/LICENSE)
[![Release](https://img.shields.io/github/release/owner/repo)](https://github.com/owner/repo/releases)
```
Source: Verified from [Gin](https://github.com/gin-gonic/gin), [Cobra](https://github.com/spf13/cobra) READMEs

### Badge Order Pattern (User Decision - Locked)

Per CONTEXT.md, the order is:
1. Go Reference - links to pkg.go.dev documentation
2. Go Report Card - links to goreportcard.com report
3. License - links to LICENSE file (requires LICENSE file to exist)
4. Release - links to GitHub releases

### Anti-Patterns to Avoid
- **Stacked badges (vertical):** User decided inline/horizontal layout
- **Custom colors/styles:** User decided flat style with standard colors
- **Badge without link:** All badges must link to their source

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Go Reference badge | Custom SVG | `pkg.go.dev/badge/{module}.svg` | Official, always current |
| Go Report Card badge | Custom score display | `goreportcard.com/badge/{repo}` | Dynamic, updates automatically |
| License detection | Parse LICENSE file | `shields.io/github/license/{owner}/{repo}` | GitHub API provides this |
| Release version | Parse go.mod/git tags | `shields.io/github/release/{owner}/{repo}` | GitHub API provides this |

**Key insight:** All these services provide dynamic badges that update automatically - no maintenance needed.

## Common Pitfalls

### Pitfall 1: Coverage Filename Mismatch
**What goes wrong:** Documentation says `coverage.out` but C6 analyzer expects `cover.out`
**Why it happens:** Different conventions in Go ecosystem (`cover.out` vs `coverage.out`)
**How to avoid:** Use `cover.out` consistently - this matches C6 analyzer's expected filename
**Warning signs:** C6 reports "coverage: not available" despite running tests with coverage

**CRITICAL:** Current README.md and CLAUDE.md use `coverage.out`. The C6 analyzer at `internal/analyzer/c6_testing/testing.go:268` searches for `cover.out`. Must be corrected to `cover.out`.

### Pitfall 2: Hyphen Escaping in shields.io
**What goes wrong:** Badge text containing hyphens displays incorrectly
**Why it happens:** shields.io interprets `-` as separator, must escape with `--`
**How to avoid:** Double hyphens `--` render as single `-` in badge text
**Warning signs:** Badge text splits unexpectedly

Source: [shields.io documentation](https://shields.io/badges)

### Pitfall 3: License Badge Without LICENSE File
**What goes wrong:** License badge shows "Not found" or fails
**Why it happens:** shields.io GitHub license badge requires LICENSE file in repo root
**How to avoid:** Ensure LICENSE file exists before adding license badge
**Warning signs:** Badge displays generic error or "Not found"

**Note:** This repository currently has no LICENSE file. The license badge will not work until a LICENSE file is added.

### Pitfall 4: Module Path vs GitHub Path
**What goes wrong:** Badges show "Not found" for pkg.go.dev
**Why it happens:** pkg.go.dev uses Go module path, GitHub badges use owner/repo
**How to avoid:**
- pkg.go.dev: Use module path from go.mod (`github.com/ingo/agent-readyness`)
- GitHub badges: Use owner/repo (`ingo-eichhorst/agent-readyness`)
**Warning signs:** 404 on badge images

**Note:** The go.mod module path (`github.com/ingo/agent-readyness`) differs from the actual GitHub URL (`github.com/ingo-eichhorst/agent-readyness`). The pkg.go.dev badge should use the module path for consistency with Go imports.

## Code Examples

### Go Reference Badge
```markdown
[![Go Reference](https://pkg.go.dev/badge/github.com/ingo/agent-readyness.svg)](https://pkg.go.dev/github.com/ingo/agent-readyness)
```
Source: [pkg.go.dev badge documentation](https://pkg.go.dev/about#adding-a-badge)

### Go Report Card Badge
```markdown
[![Go Report Card](https://goreportcard.com/badge/github.com/ingo-eichhorst/agent-readyness)](https://goreportcard.com/report/github.com/ingo-eichhorst/agent-readyness)
```
Source: [goreportcard.com](https://github.com/gojp/goreportcard)

### GitHub License Badge
```markdown
[![License](https://img.shields.io/github/license/ingo-eichhorst/agent-readyness)](https://github.com/ingo-eichhorst/agent-readyness/blob/main/LICENSE)
```
Source: [shields.io GitHub license badge](https://shields.io/badges/git-hub-license)

### GitHub Release Badge
```markdown
[![Release](https://img.shields.io/github/release/ingo-eichhorst/agent-readyness)](https://github.com/ingo-eichhorst/agent-readyness/releases)
```
Source: [shields.io GitHub release badge](https://shields.io/badges/git-hub-release)

### Test Command with Coverage
```bash
# Run all tests with coverage profile
go test ./... -coverprofile=cover.out

# View coverage report
go tool cover -html=cover.out
```
Source: [go help testflag](https://pkg.go.dev/cmd/go#hdr-Test_packages)

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| godoc.org badges | pkg.go.dev badges | 2019 | godoc.org redirects to pkg.go.dev |
| img.shields.io/badge (static) | Dynamic GitHub badges | N/A | Auto-updates, no manual maintenance |

**Current standard:**
- pkg.go.dev for Go documentation (official)
- shields.io for GitHub-sourced dynamic badges
- goreportcard.com for Go quality metrics

## Open Questions

1. **LICENSE File Missing**
   - What we know: The repository has no LICENSE file
   - What's unclear: What license should be used (MIT, Apache 2.0, etc.)
   - Recommendation: Add LICENSE file before or during this phase, or skip license badge (but user decided to include it)

2. **Module Path Discrepancy**
   - What we know: go.mod says `github.com/ingo/agent-readyness`, GitHub URL is `github.com/ingo-eichhorst/agent-readyness`
   - What's unclear: Whether pkg.go.dev will resolve correctly
   - Recommendation: Use module path for pkg.go.dev badge, GitHub path for shields.io badges

## Sources

### Primary (HIGH confidence)
- [pkg.go.dev badge documentation](https://pkg.go.dev/about#adding-a-badge) - Official badge syntax
- [shields.io documentation](https://shields.io/badges) - Character escaping, URL format
- [shields.io GitHub license](https://shields.io/badges/git-hub-license) - License badge syntax
- [shields.io GitHub release](https://shields.io/badges/git-hub-release) - Release badge syntax
- [goreportcard GitHub](https://github.com/gojp/goreportcard) - Badge URL format
- `go help testflag` - Coverage flag documentation (local)
- C6 analyzer source (`internal/analyzer/c6_testing/testing.go`) - Expected coverage filename

### Secondary (MEDIUM confidence)
- [Gin README](https://github.com/gin-gonic/gin) - Real-world badge patterns
- [Cobra README](https://github.com/spf13/cobra) - Real-world badge patterns
- [daily.dev badge best practices](https://daily.dev/blog/readme-badges-github-best-practices) - Placement recommendations

### Tertiary (LOW confidence)
- None - all findings verified with official sources

## Metadata

**Confidence breakdown:**
- Badge syntax: HIGH - Official documentation from pkg.go.dev, shields.io, goreportcard.com
- Coverage filename: HIGH - Verified directly in C6 analyzer source code
- Badge order/placement: HIGH - User decision locked in CONTEXT.md
- License badge viability: MEDIUM - Depends on LICENSE file being added

**Research date:** 2026-02-04
**Valid until:** 2026-03-04 (stable domain, 30 days)

---

## Critical Findings for Planner

1. **Coverage filename MUST change** from `coverage.out` to `cover.out` in both README.md and CLAUDE.md
2. **LICENSE file missing** - License badge will fail without it; consider adding as part of this phase or noting as prerequisite
3. **Two different paths:**
   - Module path (go.mod): `github.com/ingo/agent-readyness`
   - GitHub path: `github.com/ingo-eichhorst/agent-readyness`
   - Use module path for pkg.go.dev, GitHub path for shields.io
