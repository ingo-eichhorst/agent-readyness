# Stack Research

**Domain:** Go CLI static analysis tool (codebase quality scorer) -- v2 expansion
**Researched:** 2026-02-01
**Confidence:** HIGH (core stack), MEDIUM (LLM integration, C7 headless)

## Context: v1 Stack (Validated, Unchanged)

The following technologies are already in use and validated. They are listed for reference only -- do NOT re-evaluate.

| Technology | Version | Purpose |
|------------|---------|---------|
| Go | 1.25.1 | Runtime |
| `go/ast` + `go/parser` + `go/token` | stdlib | Go source AST parsing |
| `go/types` | stdlib | Go type checking |
| `golang.org/x/tools/go/packages` | v0.41.0 | Go package loading |
| `spf13/cobra` | v1.10.2 | CLI framework |
| `fzipp/gocyclo` | v0.6.0 | Cyclomatic complexity |
| `fatih/color` | v1.18.0 | Terminal color output |
| `sabhiram/go-gitignore` | latest | Gitignore pattern matching |
| `encoding/json` | stdlib | JSON output |
| `gopkg.in/yaml.v3` | v3.0.1 | YAML parsing (already a transitive dep) |

---

## v2 New Stack Additions

### 1. Multi-Language Parsing: Tree-sitter

**Recommendation:** `github.com/tree-sitter/go-tree-sitter` (official bindings) + per-language grammar packages

| Technology | Version | Purpose | Why Recommended |
|------------|---------|---------|-----------------|
| `tree-sitter/go-tree-sitter` | v0.25.0 | Core tree-sitter Go bindings | **Official** bindings from tree-sitter org. Clean API with Parser, Tree, Node, Query, QueryCursor. Grammars imported separately so binary only includes what you need. Published Feb 2025, MIT licensed. |
| `tree-sitter/tree-sitter-python/bindings/go` | latest | Python grammar | Official Python grammar with Go bindings. Supports all Python 3.x syntax including type annotations, decorators, async/await. |
| `tree-sitter/tree-sitter-typescript/bindings/go` | latest | TypeScript grammar | Official TypeScript grammar with Go bindings. Exports both TypeScript and TSX language functions. |

**Why tree-sitter over alternatives:**

| Approach | Verdict | Reason |
|----------|---------|--------|
| tree-sitter (official Go bindings) | **USE THIS** | Unified query language across all languages. Same code structure for Python + TypeScript analysis. S-expression queries for type annotations, function defs, naming patterns. Battle-tested (used by Neovim, GitHub, Helix). |
| `smacker/go-tree-sitter` | Do not use | Community fork. Bundles all grammars (bloats binary). Uses `runtime.SetFinalizer` which has CGO bugs. Official bindings supersede this. |
| Language-specific parsers (e.g., Python AST via subprocess) | Do not use | Requires target language runtime installed. Subprocess overhead. Different APIs per language = more code to maintain. |
| Language server protocol | Do not use | Massive overkill. Requires running language servers. Designed for IDEs, not batch analysis. |

**Critical implementation note:** Official go-tree-sitter requires explicit `.Close()` calls on Parser, Tree, TreeCursor, Query, QueryCursor, and LookaheadIterator. Use `defer` immediately after creation. The `runtime.SetFinalizer` approach from smacker's fork has known CGO bugs -- the official library chose explicit cleanup for safety.

**Architecture integration:** Create a `internal/treesitter/` package that wraps tree-sitter with a language-agnostic interface. Each language gets its own query file. The existing `internal/parser/` package (which wraps `go/packages`) remains for Go -- tree-sitter is for Python and TypeScript only. Do NOT replace go/packages with tree-sitter for Go analysis; go/packages provides full type information that tree-sitter cannot.

**Tree-sitter query examples for C2 metrics:**

```scheme
;; Python: find functions WITHOUT return type annotations
(function_definition
  name: (identifier) @func.name
  !return_type) @unannotated

;; Python: find functions WITH return type annotations
(function_definition
  name: (identifier) @func.name
  return_type: (type) @return.type) @annotated

;; Python: find parameters without type annotations
(function_definition
  parameters: (parameters
    (identifier) @param.untyped))

;; TypeScript: find functions without return type
(function_declaration
  name: (identifier) @func.name
  !return_type)

;; TypeScript: find 'any' type usage
(type_identifier) @type
(#eq? @type "any")

;; Both: magic number detection (numeric literals not in const/enum)
(number) @magic
```

**Performance:** Tree-sitter parsing is extremely fast -- typically <100ms for files up to 100k lines. The C core is optimized for incremental parsing. Total parse time for a 50k LOC Python/TS codebase should be well under 5 seconds.

---

### 2. C2 Analysis (Explicitness) -- Per-Language Approach

C2 metrics (type coverage, naming conventions, magic numbers) require different strategies per language:

| Metric | Go Approach | Python Approach | TypeScript Approach |
|--------|-------------|-----------------|---------------------|
| Type coverage | N/A (Go is statically typed) | tree-sitter query: count functions with/without `return_type` and typed parameters | tree-sitter query: count `any` usage, missing return types, implicit `any` |
| Naming conventions | `go/ast` Ident nodes, check exported naming | tree-sitter `identifier` nodes, check PEP 8 patterns | tree-sitter `identifier` nodes, check camelCase patterns |
| Magic numbers | `go/ast` BasicLit nodes, filter by context | tree-sitter `(number)` captures outside const contexts | tree-sitter `(number)` captures outside const/enum contexts |
| Doc coverage | `go/ast` Comment groups on exported decls | tree-sitter `(expression_statement (string))` for docstrings | tree-sitter JSDoc `(comment)` nodes before declarations |

**No new libraries needed for C2.** This is pure analysis logic on top of existing go/ast (for Go) and tree-sitter (for Python/TypeScript). Build it yourself.

---

### 3. C4 Analysis (Documentation Quality) -- LLM Integration

**Recommendation:** `github.com/anthropics/anthropic-sdk-go` for LLM-based doc quality assessment

| Technology | Version | Purpose | Why Recommended |
|------------|---------|---------|-----------------|
| `anthropics/anthropic-sdk-go` | v1.20.0 | Claude API client | **Official** Anthropic Go SDK. Requires Go 1.22+. Supports all Claude models, streaming, system prompts. Clean API: `client.Messages.New()`. |

**C4 architecture:**

1. **Static metrics (no LLM):** README presence, doc comment coverage ratio, inline comment density. These are fast and free -- run always.
2. **LLM quality assessment (opt-in):** Send sampled doc comments + README to Claude for quality scoring. This is expensive and slow -- gate behind `--with-llm` flag.

**LLM integration pattern:**

```go
// C4 has two tiers:
// Tier 1: Static analysis (always runs, free, fast)
//   - README exists? Has sections (install, usage, API)?
//   - Doc comment coverage: % of exported symbols with comments
//   - Comment-to-code ratio
//
// Tier 2: LLM quality (opt-in via --with-llm flag)
//   - Sample N doc comments, send to Claude
//   - Ask: "Rate clarity, completeness, accuracy 1-10"
//   - Estimate cost before running, require --yes or prompt
```

**Cost control:**
- Use `claude-haiku-4-20250801` for C4 doc evaluation (cheapest, fast, good enough for quality ratings)
- Sample at most 20 doc comments + README per run
- Estimate tokens before calling API, display cost estimate
- Require explicit opt-in flag (`--with-llm`)

**Why not other LLM providers:**
- ARS already needs Claude for C7 (headless Claude Code). Single provider = simpler auth, fewer API keys.
- Anthropic SDK is official, well-maintained, Go-native.
- Claude Haiku is competitive on price for classification tasks.

---

### 4. C5 Analysis (Git Forensics) -- Hybrid Approach

**Recommendation:** `os/exec` + native `git` CLI for performance-critical operations. NOT go-git.

| Technology | Version | Purpose | Why Recommended |
|------------|---------|---------|-----------------|
| `os/exec` | stdlib | Execute git commands | Native `git` is 10-35x faster than go-git for log and blame on real repositories. go-git blame takes 35s where CLI takes <1s. go-git log with filename filtering is "extremely slow" per multiple open issues. |

**Why NOT go-git:**

go-git (v5.16.4) has well-documented, long-standing performance problems that directly impact C5:

| Operation | go-git | Native git | Impact |
|-----------|--------|------------|--------|
| `git log --follow <file>` | Extremely slow (issue #811, #137) | Fast | C5 churn analysis needs per-file commit history |
| `git blame <file>` | ~35s vs <1s (issue #14) | Fast | C5 ownership analysis needs blame data |
| Large repo (40k commits) | Memory + time issues (issue #447, #505) | Handles well | C5 must work on real-world repos |

**C5 implementation approach:**

```go
// Wrap git CLI calls in internal/gitcli/ package
// Parse structured output from git --format options

// Churn: git log --numstat --format="%H %at" -- <path>
// Hotspots: git log --format="" --name-only | sort | uniq -c | sort -rn
// Temporal coupling: git log --format="%H" --name-only (group co-changed files per commit)
// File age: git log --diff-filter=A --format="%at" -- <path>
// Ownership: git shortlog -sn -- <path>
```

**Prerequisite:** C5 requires `git` binary in PATH. If not found, C5 analysis fails gracefully with a clear error message. This is an acceptable constraint -- any codebase worth analyzing has git installed.

**Performance:** Native git operations on a 50k LOC repo with 10k commits should complete in <5 seconds total. Use `--since` flags to limit history depth if needed (default: 1 year of history).

**Alternative considered:** `go-git/go-git` v5.16.4 -- Pure Go, no external dependency. Rejected because blame and log performance on real repositories is unacceptable for a tool that must complete in <30s. The performance gap (10-35x) is too large to work around.

---

### 5. C7 Analysis (Agent Evaluation) -- Headless Claude Code

**Recommendation:** Shell out to `claude` CLI with `-p` flag (headless mode)

| Technology | Version | Purpose | Why Recommended |
|------------|---------|---------|-----------------|
| `claude` CLI | latest | Headless Claude Code execution | Official Agent SDK CLI. Use `-p` flag for non-interactive mode. Supports `--output-format json` for structured output, `--allowedTools` for permission control, `--json-schema` for structured responses. |

**C7 architecture:**

C7 evaluates how well a codebase supports AI coding agents. It does this by running a controlled Claude Code session against the target codebase and measuring outcomes.

```go
// C7 evaluation flow:
// 1. Pre-flight: Check 'claude' binary exists, check API key, estimate cost
// 2. Task generation: Create N standardized tasks based on codebase structure
//    - "Find all functions that handle authentication"
//    - "Add a test for function X"
//    - "Explain the data flow from A to B"
// 3. Execution: Run each task via claude -p with --output-format json
// 4. Scoring: Measure success rate, time-to-completion, tool usage patterns
```

**Implementation:**

```go
cmd := exec.CommandContext(ctx, "claude",
    "-p", taskPrompt,
    "--output-format", "json",
    "--json-schema", schemaJSON,
    "--allowedTools", "Read,Grep,Glob",  // Read-only tools for safety
    "--append-system-prompt", evalSystemPrompt,
)
cmd.Dir = targetRepoPath
output, err := cmd.Output()
```

**Key design decisions:**

1. **Read-only tools only:** C7 evaluation uses `--allowedTools "Read,Grep,Glob"` -- no Edit, no Bash. This prevents the evaluation from modifying the target codebase.
2. **Timeout per task:** Use `context.WithTimeout` -- 60 seconds per task, kill if exceeded.
3. **Cost estimation:** Before running, estimate total tokens (roughly N tasks * avg prompt size). Display estimate and require `--yes` flag or interactive confirmation.
4. **Opt-in only:** C7 never runs by default. Requires explicit `--with-c7` or `--full` flag.
5. **Session isolation:** Each task gets a fresh `claude -p` invocation (no `--continue`) so tasks don't leak context.

**Why shell out to CLI vs. using the SDK directly:**
- Claude Code's agent loop (tool usage, file reading, grep) is the point of C7 -- we need the full agent, not just the Messages API.
- The CLI `-p` mode gives us the full Claude Code agent with structured JSON output.
- No need to reimplement the agent loop in Go.

**Cost model (MEDIUM confidence):**
- 5 evaluation tasks per run
- ~2k input tokens + ~1k output tokens per task
- Using Claude Sonnet: ~$0.15-0.30 per full C7 evaluation
- Using Claude Haiku: ~$0.02-0.05 per full C7 evaluation
- Display cost estimate before running

---

### 6. HTML Report Generation

**Recommendation:** `html/template` (stdlib) + embedded Apache ECharts JS

| Technology | Version | Purpose | Why Recommended |
|------------|---------|---------|-----------------|
| `html/template` | stdlib | HTML report generation | Go stdlib. Zero dependencies. Supports template composition, auto-escaping. Sufficient for generating a single-file HTML report. |
| Apache ECharts | v5.6+ (JS, embedded) | Interactive charts in HTML | Embed the ECharts JS library directly in the HTML template as a `<script>` tag (CDN link or embedded via `embed.FS`). 25+ chart types, interactive tooltips, responsive. |
| `embed` | stdlib (Go 1.16+) | Embed static assets in binary | Use `//go:embed` to include HTML templates and optionally the ECharts JS file in the binary. Single binary distribution, no external file dependencies. |

**Why NOT go-echarts:**
- go-echarts (v2) is a Go wrapper that generates ECharts HTML. It adds a dependency and abstraction layer for something we can do directly.
- Our chart needs are simple: radar chart for C1-C7 scores, bar charts for sub-metrics, maybe a line chart for trends. Direct ECharts JS in a template is ~50 lines of JavaScript.
- go-echarts makes sense for a web dashboard with many dynamic charts. For a static HTML report generated once, templates + raw ECharts is simpler and dependency-free.

**Report architecture:**

```
internal/output/
  terminal.go     (existing)
  json.go         (existing)
  html.go         (NEW)
  templates/
    report.html.tmpl    (main template)
    partials/
      header.html.tmpl
      category.html.tmpl
      chart.html.tmpl
```

**Template approach:**
```go
//go:embed templates/*
var templateFS embed.FS

func RenderHTML(result *types.ScanResult, w io.Writer) error {
    tmpl := template.Must(template.ParseFS(templateFS, "templates/*.html.tmpl"))
    return tmpl.ExecuteTemplate(w, "report.html.tmpl", result)
}
```

**ECharts integration:** Embed chart data as JSON in a `<script>` block within the template. ECharts JS loaded via CDN (`https://cdn.jsdelivr.net/npm/echarts@5/dist/echarts.min.js`) with a fallback message if offline. For fully offline reports, embed ECharts (~800KB minified) via `embed.FS`.

---

### 7. Config Management: YAML Parsing

**Recommendation:** `gopkg.in/yaml.v3` -- already in go.mod as a transitive dependency

| Technology | Version | Purpose | Why Recommended |
|------------|---------|---------|-----------------|
| `gopkg.in/yaml.v3` | v3.0.1 | Parse `.arsrc.yml` config | Already a transitive dependency (via cobra/pflag chain). Stable, MIT + Apache 2.0 licensed. De facto standard Go YAML library. Supports struct tags, custom unmarshalling. |

**Why already available:** Check go.sum -- `gopkg.in/yaml.v3 v3.0.1` is already listed as an indirect dependency. Promoting it to a direct dependency adds zero new code to the binary.

**Config structure:**

```yaml
# .arsrc.yml
version: 1

categories:
  c1: { enabled: true }
  c2: { enabled: true }
  c3: { enabled: true }
  c4: { enabled: true, llm: false }  # llm: true requires --with-llm
  c5: { enabled: true }
  c6: { enabled: true }
  c7: { enabled: false }  # opt-in only

scoring:
  passing_threshold: 7.0

languages:
  - go
  - python
  - typescript

output:
  format: terminal  # terminal | json | html
  html_path: ./ars-report.html

llm:
  model: claude-haiku-4-20250801
  max_cost_usd: 1.00
```

**Implementation:**

```go
type Config struct {
    Version    int                `yaml:"version"`
    Categories map[string]CatCfg `yaml:"categories"`
    Scoring    ScoringCfg        `yaml:"scoring"`
    Languages  []string          `yaml:"languages"`
    Output     OutputCfg         `yaml:"output"`
    LLM        LLMCfg            `yaml:"llm"`
}

func LoadConfig(path string) (*Config, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return DefaultConfig(), nil  // missing config = use defaults
    }
    var cfg Config
    if err := yaml.Unmarshal(data, &cfg); err != nil {
        return nil, fmt.Errorf("invalid .arsrc.yml: %w", err)
    }
    return &cfg, nil
}
```

**Alternatives considered:**

| Recommended | Alternative | Why Not |
|-------------|-------------|---------|
| `yaml.v3` | `spf13/viper` | Viper is a full config framework (env vars, remote config, watching). Overkill. Adds heavy transitive deps. We just need to parse one YAML file. |
| `yaml.v3` | `goccy/go-yaml` | Better YAML spec compliance (passes 60 more test suite cases). But yaml.v3 is already in our dependency tree and handles all real-world YAML configs correctly. Not worth adding a new dep. |
| `yaml.v3` | TOML (`BurntSushi/toml`) | TOML is fine for config. But YAML is more common for `.xxxrc` files in the code quality tool space (eslint, prettier use JSON/YAML). YAML also already in our deps. |
| `yaml.v3` | JSON config | JSON lacks comments. Config files benefit enormously from inline comments explaining thresholds and options. YAML wins here. |

---

## Complete v2 Installation

```bash
# New dependencies for v2 (in addition to existing v1 deps)

# Tree-sitter: multi-language parsing
go get github.com/tree-sitter/go-tree-sitter@latest
go get github.com/tree-sitter/tree-sitter-python/bindings/go@latest
go get github.com/tree-sitter/tree-sitter-typescript/bindings/go@latest

# LLM integration: Anthropic SDK for C4 + C7
go get github.com/anthropics/anthropic-sdk-go@v1.20.0

# YAML: promote from indirect to direct (already in go.sum)
go get gopkg.in/yaml.v3@v3.0.1

# No new deps needed for:
# - C5 git forensics (os/exec + native git)
# - C7 headless Claude Code (os/exec + claude CLI)
# - HTML reports (html/template + embed stdlib)
# - C2 analysis (built on tree-sitter + go/ast)
```

## v2 Dependency Impact

| New Dependency | Transitive Deps | Binary Size Impact | Why Accept |
|----------------|-----------------|-------------------|------------|
| `tree-sitter/go-tree-sitter` | CGO (links libtree-sitter.a) | ~2-3 MB | Core requirement for multi-language support. No pure-Go alternative exists. |
| `tree-sitter-python/bindings/go` | Embeds C grammar | ~500 KB | Required for Python analysis. |
| `tree-sitter-typescript/bindings/go` | Embeds C grammar | ~800 KB | Required for TypeScript analysis. |
| `anthropics/anthropic-sdk-go` | HTTP client deps | ~1 MB | Required for C4 LLM + C7 cost estimation. Only used when --with-llm flag present. |
| `gopkg.in/yaml.v3` (promoted) | None new | 0 KB (already linked) | Already in binary via transitive dep. |

**Total new binary size:** ~4-5 MB increase (from ~15 MB to ~20 MB). Acceptable for the functionality gained.

**CGO requirement:** Tree-sitter requires CGO enabled (`CGO_ENABLED=1`). This is the default on macOS and Linux but must be explicitly set for cross-compilation. This is the biggest tradeoff of v2 -- v1 was pure Go. Document this in build instructions.

---

## What NOT to Add for v2

| Avoid | Why | Use Instead |
|-------|-----|-------------|
| `go-git/go-git` v5 | Blame is 35x slower than native git. Log with file filtering is "extremely slow." Multiple long-standing issues (#14, #137, #811). | `os/exec` + native `git` CLI |
| `smacker/go-tree-sitter` | Community fork superseded by official bindings. Bundles all grammars (binary bloat). Uses buggy SetFinalizer pattern. | `tree-sitter/go-tree-sitter` (official) |
| `go-echarts/go-echarts` v2 | Abstraction layer over ECharts that adds a dependency for simple chart needs. Our reports need 3-4 chart types, not 25+. | `html/template` + raw ECharts JS (CDN or embedded) |
| `spf13/viper` | Config framework overkill. We parse one YAML file. Viper adds 10+ transitive deps for features we don't need (env vars, remote config, watching). | `gopkg.in/yaml.v3` directly |
| Python/TypeScript runtime | Do NOT require target language runtimes for analysis. Tree-sitter parses without them. | Tree-sitter grammar packages (compiled C) |
| OpenAI/other LLM SDKs | Single LLM provider (Anthropic) keeps auth simple. Claude Code is already required for C7. | `anthropics/anthropic-sdk-go` for all LLM needs |
| `M1n9X/claude-agent-sdk-go` | Unofficial Go SDK for Claude Code agent. Published Jan 2026, unproven, single maintainer. | Shell out to `claude` CLI directly -- official, tested, maintained by Anthropic |

---

## Stack Patterns by Feature Category

**C2 (Explicitness):**
- Go: `go/ast` + `go/types` (existing) -- type info already available via packages
- Python: tree-sitter queries for type annotations, docstrings, naming
- TypeScript: tree-sitter queries for `any` usage, missing types, naming
- No new deps beyond tree-sitter

**C4 (Documentation):**
- Static tier: tree-sitter queries for doc comments + file I/O for README detection
- LLM tier: `anthropics/anthropic-sdk-go` with Haiku model
- Cost-gated behind `--with-llm` flag

**C5 (Temporal):**
- Pure `os/exec` + `git` CLI
- Parse git output with stdlib (`strings`, `strconv`, `time`)
- No new deps

**C7 (Agent Readiness):**
- Pure `os/exec` + `claude` CLI with `-p` flag
- Parse JSON output with `encoding/json`
- No new deps beyond Anthropic SDK (for cost estimation)

---

## Version Compatibility (v2 additions)

| Package | Compatible With | Notes |
|---------|-----------------|-------|
| `tree-sitter/go-tree-sitter` v0.25.0 | Go 1.11+ (needs CGO) | Requires C compiler. Default on macOS (clang) and Linux (gcc). |
| `tree-sitter-python/bindings/go` | Matches go-tree-sitter version | Install at same time as core bindings. |
| `tree-sitter-typescript/bindings/go` | Matches go-tree-sitter version | Exports both TS and TSX language functions. |
| `anthropics/anthropic-sdk-go` v1.20.0 | Go 1.22+ | Uses generics. Our minimum (Go 1.24+) exceeds this. |
| `gopkg.in/yaml.v3` v3.0.1 | Go 1.15+ | Very stable. No compatibility concerns. |

---

## Sources

### Tree-sitter
- [tree-sitter/go-tree-sitter on pkg.go.dev](https://pkg.go.dev/github.com/tree-sitter/go-tree-sitter) -- v0.25.0, official Go bindings (HIGH confidence)
- [tree-sitter/go-tree-sitter on GitHub](https://github.com/tree-sitter/go-tree-sitter) -- API docs, memory management requirements (HIGH confidence)
- [tree-sitter-typescript Go bindings](https://pkg.go.dev/github.com/tree-sitter/tree-sitter-typescript/bindings/go) -- official TS grammar (HIGH confidence)
- [smacker/go-tree-sitter on GitHub](https://github.com/smacker/go-tree-sitter) -- community fork, evaluated and rejected (HIGH confidence)
- [Tree-sitter query syntax docs](https://tree-sitter.github.io/tree-sitter/using-parsers/queries/1-syntax.html) -- query patterns for type annotations (HIGH confidence)

### Git Forensics
- [go-git/go-git on pkg.go.dev](https://pkg.go.dev/github.com/go-git/go-git/v5) -- v5.16.4, evaluated and rejected for performance (HIGH confidence)
- [go-git blame performance issue #14](https://github.com/go-git/go-git/issues/14) -- 35x slower than CLI (HIGH confidence)
- [go-git log filtering issue #137](https://github.com/go-git/go-git/issues/137) -- filename filtering very slow (HIGH confidence)
- [go-git log issue #811](https://github.com/go-git/go-git/issues/811) -- log extremely slow applied to file (HIGH confidence)

### LLM Integration
- [anthropics/anthropic-sdk-go on GitHub](https://github.com/anthropics/anthropic-sdk-go) -- v1.20.0, official Go SDK (HIGH confidence)
- [Claude Code headless docs](https://code.claude.com/docs/en/headless) -- -p flag, --output-format json, --allowedTools (HIGH confidence)
- [Claude Code Agent SDK blog](https://blog.promptlayer.com/building-agents-with-claude-codes-sdk/) -- SDK architecture overview (MEDIUM confidence)

### HTML Reports
- [go-echarts on GitHub](https://github.com/go-echarts/go-echarts) -- evaluated, rejected for simplicity (HIGH confidence)
- [Apache ECharts](https://echarts.apache.org/) -- JS charting library, embed in HTML template (HIGH confidence)

### YAML
- [gopkg.in/yaml.v3 on pkg.go.dev](https://pkg.go.dev/gopkg.in/yaml.v3) -- v3.0.1, de facto Go YAML library (HIGH confidence)
- [goccy/go-yaml on GitHub](https://github.com/goccy/go-yaml) -- alternative evaluated, rejected (MEDIUM confidence)

---
*Stack research for: ARS v2 -- Multi-language support, C2/C4/C5/C7, HTML reports, config*
*Researched: 2026-02-01*
