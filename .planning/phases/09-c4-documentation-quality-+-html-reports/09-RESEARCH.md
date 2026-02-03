# Phase 9: C4 Documentation Quality + HTML Reports - Research

**Researched:** 2026-02-03
**Domain:** Documentation metrics analysis, LLM-as-Judge evaluation, HTML report generation
**Confidence:** HIGH

## Summary

Phase 9 implements two major features: (1) C4 Documentation Quality analysis with both static metrics (README presence, comment density, API docs) and optional LLM-based content evaluation (README clarity, example quality), and (2) HTML report generation with self-contained radar charts, metric breakdowns, and research citations.

The C4 static metrics (C4-01 through C4-07) require file system inspection and comment parsing using existing AST patterns. The LLM-based content quality metrics (C4-08 through C4-14) require integrating the official Anthropic Go SDK (`github.com/anthropics/anthropic-sdk-go`) with Claude Haiku for cost-effective evaluation, implementing prompt caching to reduce costs, and providing cost estimation before execution.

For HTML reports (HTML-01 through HTML-10), Go's `html/template` package provides automatic contextual XSS escaping. The `vicanso/go-charts/v2` library generates SVG radar charts natively in Go with no external dependencies. Reports are self-contained by inlining all CSS/JS and using the Go `embed` directive for templates.

**Primary recommendation:** Implement C4 static metrics as a new analyzer following the C5 pattern (repo-level analysis), add LLM client abstraction with cost estimation for opt-in content evaluation, and generate HTML reports using `html/template` with embedded SVG charts from `go-charts`.

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `html/template` | stdlib | HTML generation with XSS escaping | Go standard, automatic contextual escaping |
| `embed` | stdlib (Go 1.16+) | Embed CSS/templates into binary | Self-contained single-binary deployment |
| `github.com/anthropics/anthropic-sdk-go` | latest | Anthropic Claude API | Official SDK, supports prompt caching |
| `github.com/vicanso/go-charts/v2` | v2.x | SVG radar chart generation | Pure Go, <20ms generation, no JS dependencies |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `go/ast`, `go/parser` | stdlib | Go comment extraction | Analyzing Go docstrings |
| `regexp` | stdlib | Pattern matching for docs detection | Markdown heading detection, pattern matching |
| `path/filepath` | stdlib | File path operations | Scanning docs/, README.md |
| `strings`, `unicode` | stdlib | Text processing | Word counting, naming convention checks |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| vicanso/go-charts | Hand-rolled SVG | go-charts has radar chart support, themes, and is battle-tested |
| html/template | text/template | text/template lacks XSS protection -- never use for HTML |
| Anthropic SDK | Generic HTTP client | SDK handles auth, retries, streaming, prompt caching API |
| Claude Haiku | Claude Sonnet/Opus | Haiku is 10x cheaper ($1/$5 per MTok) for simple evaluation tasks |

**Installation:**
```bash
go get github.com/anthropics/anthropic-sdk-go
go get github.com/vicanso/go-charts/v2
```

## Architecture Patterns

### Recommended Project Structure
```
internal/analyzer/
    c4_documentation.go       # C4Analyzer for static metrics
    c4_documentation_test.go  # Unit tests with fixture files
internal/llm/
    client.go                 # LLM client abstraction
    client_test.go
    cost.go                   # Token counting and cost estimation
    prompts.go                # Evaluation prompts for C4 content quality
internal/output/
    html.go                   # HTML report generator
    html_test.go
    templates/
        report.html           # Main HTML template (embedded)
        styles.css            # Inline styles (embedded)
pkg/types/
    types.go                  # Add C4Metrics, C4LLMMetrics structs
```

### Pattern 1: C4 Analyzer as Repo-Level Analyzer

**What:** Like C5, C4 operates at the repository level, scanning for documentation files rather than analyzing individual source files.

**When to use:** Always for C4 -- documentation metrics are repo-wide, not per-file.

**Example:**
```go
// Source: existing C5Analyzer pattern in codebase
type C4Analyzer struct{}

func (a *C4Analyzer) Name() string { return "C4: Documentation Quality" }

func (a *C4Analyzer) Analyze(targets []*types.AnalysisTarget) (*types.AnalysisResult, error) {
    if len(targets) == 0 {
        return nil, fmt.Errorf("no targets provided")
    }
    rootDir := targets[0].RootDir

    metrics := &types.C4Metrics{}

    // C4-01: README presence and word count
    metrics.ReadmePresent, metrics.ReadmeWordCount = analyzeReadme(rootDir)

    // C4-02: Comment density (aggregate across all source files)
    metrics.CommentDensity = analyzeCommentDensity(targets)

    // C4-03: API documentation coverage
    metrics.APIDocCoverage = analyzeAPIDocs(targets)

    // ... remaining static metrics

    return &types.AnalysisResult{
        Name:     "C4: Documentation Quality",
        Category: "C4",
        Metrics:  map[string]interface{}{"c4": metrics},
    }, nil
}
```

### Pattern 2: LLM Client Abstraction with Cost Estimation

**What:** A thin abstraction over the Anthropic SDK that handles prompt caching, cost estimation, and sampling strategy for C4 LLM evaluation.

**When to use:** For C4-08 through C4-14 (opt-in LLM analysis with --enable-c4-llm).

**Example:**
```go
// Source: Anthropic SDK documentation + PRD requirements
type LLMClient struct {
    client *anthropic.Client
    model  anthropic.Model
}

// EstimateCost calculates the cost before running analysis
func (c *LLMClient) EstimateCost(files []string, promptTokens int) CostEstimate {
    // Count files to sample (max 50-100 per PRD)
    sampleSize := min(len(files), 100)

    // Estimate tokens per file (avg ~500 tokens of context)
    inputTokens := sampleSize * (promptTokens + 500)
    outputTokens := sampleSize * 100 // ~100 tokens response per file

    // Haiku 4.5 pricing: $1/MTok input, $5/MTok output
    // With prompt caching: writes 1.25x, reads 0.1x
    inputCost := float64(inputTokens) / 1_000_000 * 1.0
    outputCost := float64(outputTokens) / 1_000_000 * 5.0

    return CostEstimate{
        InputTokens:  inputTokens,
        OutputTokens: outputTokens,
        TotalCost:    inputCost + outputCost,
        FilesCount:   sampleSize,
    }
}

// EvaluateContent runs the LLM judge evaluation with prompt caching
func (c *LLMClient) EvaluateContent(ctx context.Context, content string, rubric string) (int, error) {
    response, err := c.client.Messages.New(ctx, anthropic.MessageNewParams{
        Model:     c.model,
        MaxTokens: 256,
        System: []anthropic.SystemContentBlock{
            anthropic.NewTextBlock(rubric),
            {
                Type: "text",
                Text: evaluationRubric, // ~2000 token system prompt
                CacheControl: &anthropic.CacheControlParam{
                    Type: "ephemeral", // 5-minute cache
                },
            },
        },
        Messages: []anthropic.MessageParam{
            anthropic.NewUserMessage(anthropic.NewTextBlock(content)),
        },
    })
    if err != nil {
        return 0, err
    }
    // Parse 1-10 score from response
    return parseScore(response.Content[0].Text)
}
```

### Pattern 3: Prompt Caching for C4 LLM Analysis

**What:** Use Anthropic's prompt caching to cache the evaluation rubric (system prompt) across multiple file evaluations, reducing costs by 90%.

**When to use:** Always for C4 LLM analysis -- the rubric is identical across all evaluations.

**API usage:**
```go
// Source: https://platform.claude.com/docs/en/build-with-claude/prompt-caching
// Mark the end of cacheable content with cache_control
System: []anthropic.SystemContentBlock{
    {
        Type: "text",
        Text: evaluationRubric,
        CacheControl: &anthropic.CacheControlParam{
            Type: "ephemeral", // 5-minute TTL, auto-refreshed on use
        },
    },
},
```

**Minimum cacheable tokens:**
- Claude Haiku 4.5: 4096 tokens minimum (per official docs)
- For shorter rubrics, pad with detailed examples to reach threshold

**Pricing multipliers:**
- Cache write: 1.25x base input price
- Cache read: 0.1x base input price (90% savings)
- Haiku 4.5: $1/MTok input, $5/MTok output

### Pattern 4: Self-Contained HTML with Embedded SVG

**What:** Generate a single HTML file with all CSS inline and charts as inline SVG, no external dependencies.

**When to use:** Always for --output-html to ensure offline rendering.

**Example:**
```go
// Source: Go embed directive + html/template docs
//go:embed templates/report.html templates/styles.css
var templateFS embed.FS

func RenderHTML(w io.Writer, data *HTMLReportData) error {
    // Load embedded template
    tmpl, err := template.ParseFS(templateFS, "templates/report.html")
    if err != nil {
        return err
    }

    // Generate radar chart as inline SVG
    radarSVG, err := generateRadarChart(data.Categories)
    if err != nil {
        return err
    }
    data.RadarChartSVG = template.HTML(radarSVG) // Safe: we generate this

    // Render template (html/template escapes user data automatically)
    return tmpl.Execute(w, data)
}
```

### Pattern 5: SVG Radar Chart with go-charts

**What:** Use vicanso/go-charts to generate radar charts as SVG strings for embedding in HTML.

**When to use:** For HTML reports with composite + per-category score visualization.

**Example:**
```go
// Source: github.com/vicanso/go-charts documentation
func generateRadarChart(categories []types.CategoryScore) (string, error) {
    // Prepare indicator names and max values
    var names []string
    var maxValues []float64
    var values [][]float64

    categoryValues := make([]float64, len(categories))
    for i, cat := range categories {
        names = append(names, cat.Name)
        maxValues = append(maxValues, 10.0) // All scores are 1-10
        categoryValues[i] = cat.Score
    }
    values = append(values, categoryValues)

    p, err := charts.RadarRender(
        values,
        charts.SVGTypeOption(),
        charts.TitleTextOptionFunc("Agent Readiness Score"),
        charts.RadarIndicatorOptionFunc(names, maxValues),
        charts.ThemeOptionFunc("light"),
        charts.WidthOptionFunc(400),
        charts.HeightOptionFunc(400),
    )
    if err != nil {
        return "", err
    }

    buf, err := p.Bytes()
    return string(buf), err
}
```

### Anti-Patterns to Avoid
- **Using template.HTML with user data:** NEVER use `template.HTML`, `template.JS`, or `template.CSS` on any data that could come from analyzed code. Only use for content YOU generate (like SVG charts).
- **External CSS/JS dependencies:** Don't link to CDNs -- reports must render offline.
- **text/template for HTML:** Always use html/template for automatic XSS escaping.
- **Calling LLM for every file:** Use sampling strategy (50-100 files max per C4-12).
- **Skipping cost estimation:** Always show estimated cost before running LLM analysis (C4-14).

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Radar chart rendering | Custom SVG path math | `vicanso/go-charts/v2` | Complex polar coordinate math, go-charts handles labels, scales, themes |
| HTML escaping | Manual escaping functions | `html/template` | Contextual escaping is hard; template handles HTML, JS, CSS, URL contexts |
| LLM API integration | Raw HTTP calls | `anthropic-sdk-go` | Handles auth, retries, streaming, error types, prompt caching API |
| Token counting | Regex-based estimation | tiktoken or API response | Tokenization is model-specific; use response `usage` field |
| Word counting | `strings.Split(" ")` | `unicode.IsSpace` + FSM | Handles multiple spaces, newlines, unicode properly |

**Key insight:** HTML report generation and LLM integration both have subtle edge cases (XSS, token limits, caching) that standard libraries handle correctly.

## Common Pitfalls

### Pitfall 1: XSS via template.HTML
**What goes wrong:** Using `template.HTML(userContent)` bypasses escaping, allowing XSS from code content.
**Why it happens:** Developers want to render code blocks without escaping.
**How to avoid:** NEVER use template.HTML on content from analyzed files. Let html/template escape it. Use `<pre><code>` for code display with proper escaping.
**Warning signs:** Any `template.HTML()` call on file content.

### Pitfall 2: Prompt Caching Minimum Token Threshold
**What goes wrong:** Cache writes fail silently when prompt is below 4096 tokens (Haiku 4.5).
**Why it happens:** Anthropic requires minimum cacheable length.
**How to avoid:** Ensure system prompt with rubric is >= 4096 tokens. Pad with examples if needed. Check `cache_creation_input_tokens` in response to verify caching.
**Warning signs:** `cache_read_input_tokens: 0` on subsequent requests.

### Pitfall 3: LLM Cost Explosion Without Sampling
**What goes wrong:** Running LLM analysis on thousands of files creates huge API bills.
**Why it happens:** No sampling strategy limits file count.
**How to avoid:** Implement sampling (C4-12): max 50-100 files. Sample stratified by complexity or file size.
**Warning signs:** Cost estimate > $1 for a single scan.

### Pitfall 4: Comment Density False Positives
**What goes wrong:** Counting license headers, auto-generated comments as "inline comments."
**Why it happens:** Naive comment counting includes all comments.
**How to avoid:** Filter license headers (first N lines), generated markers. Focus on function-level docstrings and inline explanatory comments.
**Warning signs:** 100% comment density on generated files.

### Pitfall 5: Trend Chart Without Baseline Normalization
**What goes wrong:** Baseline comparison shows meaningless changes when metrics aren't normalized.
**Why it happens:** Raw values have different scales (0-100% vs 0-50 complexity).
**How to avoid:** Compare SCORES (1-10 normalized), not raw values. Show delta in score points.
**Warning signs:** Trend chart with values like "42 -> 38" (what does that mean?).

### Pitfall 6: Blocking on LLM Calls
**What goes wrong:** Sequential LLM calls take minutes for large repos.
**Why it happens:** Running evaluations one-by-one without concurrency.
**How to avoid:** Use `errgroup` with limited concurrency (e.g., 5 parallel). Respect rate limits.
**Warning signs:** LLM analysis takes > 30 seconds.

## Code Examples

### C4Metrics Struct
```go
// Source: PRD requirements C4-01 through C4-14
// C4Metrics holds Documentation Quality metric results.
type C4Metrics struct {
    // Static metrics (always available)
    ReadmePresent     bool
    ReadmeWordCount   int
    CommentDensity    float64          // % lines with comments (0-100)
    APIDocCoverage    float64          // % public APIs with docstrings (0-100)
    ChangelogPresent  bool
    ChangelogDaysOld  int              // -1 if not present
    DiagramsPresent   bool
    ExamplesPresent   bool
    ContributingPresent bool

    // LLM metrics (only if --enable-c4-llm)
    LLMEnabled        bool
    ReadmeClarity     int              // 1-10 scale
    ExampleQuality    int              // 1-10 scale
    Completeness      int              // 1-10 scale
    CrossRefCoherence int              // 1-10 scale
    LLMCost           float64          // Actual cost in USD
    LLMTokensUsed     int
    FilesSampled      int
}
```

### README Analysis
```go
// Source: PRD C4-01
func analyzeReadme(rootDir string) (bool, int) {
    // Check common README locations
    readmePaths := []string{
        filepath.Join(rootDir, "README.md"),
        filepath.Join(rootDir, "README"),
        filepath.Join(rootDir, "readme.md"),
        filepath.Join(rootDir, "Readme.md"),
    }

    for _, path := range readmePaths {
        content, err := os.ReadFile(path)
        if err != nil {
            continue
        }
        wordCount := countWords(string(content))
        return true, wordCount
    }

    return false, 0
}

func countWords(text string) int {
    words := 0
    inWord := false
    for _, r := range text {
        if unicode.IsSpace(r) {
            inWord = false
        } else if !inWord {
            inWord = true
            words++
        }
    }
    return words
}
```

### Comment Density Calculation
```go
// Source: PRD C4-02
func analyzeCommentDensity(targets []*types.AnalysisTarget) float64 {
    totalLines := 0
    commentLines := 0

    for _, target := range targets {
        for _, file := range target.Files {
            if file.Class != types.ClassSource {
                continue
            }

            lines, comments := countCommentLines(file.Content, target.Language)
            totalLines += lines
            commentLines += comments
        }
    }

    if totalLines == 0 {
        return 0
    }
    return float64(commentLines) / float64(totalLines) * 100
}

func countCommentLines(content []byte, lang types.Language) (int, int) {
    lines := strings.Split(string(content), "\n")
    total := len(lines)
    comments := 0

    inBlockComment := false
    for _, line := range lines {
        trimmed := strings.TrimSpace(line)

        switch lang {
        case types.LangGo:
            if strings.HasPrefix(trimmed, "//") {
                comments++
            } else if strings.HasPrefix(trimmed, "/*") {
                inBlockComment = true
                comments++
            } else if inBlockComment {
                comments++
                if strings.Contains(trimmed, "*/") {
                    inBlockComment = false
                }
            }
        case types.LangPython:
            if strings.HasPrefix(trimmed, "#") {
                comments++
            }
            // TODO: Handle """ docstrings
        case types.LangTypeScript:
            if strings.HasPrefix(trimmed, "//") {
                comments++
            } else if strings.HasPrefix(trimmed, "/*") || strings.HasPrefix(trimmed, "/**") {
                inBlockComment = true
                comments++
            } else if inBlockComment {
                comments++
                if strings.Contains(trimmed, "*/") {
                    inBlockComment = false
                }
            }
        }
    }

    return total, comments
}
```

### LLM Evaluation Prompt (README Clarity)
```go
// Source: LLM-as-Judge best practices + PRD C4-08
const readmeClarityPrompt = `You are an expert code documentation evaluator. Your task is to rate the clarity of a README file on a scale of 1-10.

Evaluation criteria:
- PURPOSE (2 points): Does it clearly explain what the project does?
- QUICKSTART (2 points): Can a developer start using it within 5 minutes?
- STRUCTURE (2 points): Is it well-organized with clear sections?
- EXAMPLES (2 points): Does it include practical usage examples?
- COMPLETENESS (2 points): Does it cover installation, usage, and configuration?

Scoring guide:
- 9-10: Excellent. All criteria met with high quality.
- 7-8: Good. Most criteria met, minor gaps.
- 5-6: Adequate. Basic information present but could improve.
- 3-4: Poor. Significant gaps in clarity or content.
- 1-2: Very poor. Barely usable as documentation.

Respond with ONLY a JSON object: {"score": <1-10>, "reason": "<brief explanation>"}`
```

### HTML Template Structure
```html
<!-- Source: html/template best practices -->
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>ARS Report: {{.ProjectName}}</title>
    <style>
        /* Inline all CSS for self-containment */
        :root {
            --score-excellent: #22c55e;
            --score-good: #eab308;
            --score-poor: #ef4444;
        }
        body { font-family: system-ui, sans-serif; max-width: 1200px; margin: 0 auto; padding: 2rem; }
        .radar-chart { display: flex; justify-content: center; margin: 2rem 0; }
        .metric-table { width: 100%; border-collapse: collapse; }
        .metric-table th, .metric-table td { padding: 0.75rem; border-bottom: 1px solid #e5e7eb; }
        .citation { font-size: 0.875rem; color: #6b7280; }
        /* ... more styles ... */
    </style>
</head>
<body>
    <header>
        <h1>Agent Readiness Score: {{.ProjectName}}</h1>
        <div class="composite-score">
            <span class="score">{{printf "%.1f" .Composite}}</span>
            <span class="tier tier-{{.TierClass}}">{{.Tier}}</span>
        </div>
    </header>

    <section class="radar-chart">
        <!-- Inline SVG from go-charts -->
        {{.RadarChartSVG}}
    </section>

    <section class="categories">
        {{range .Categories}}
        <div class="category">
            <h2>{{.Name}}</h2>
            <table class="metric-table">
                <thead>
                    <tr><th>Metric</th><th>Value</th><th>Score</th><th>Threshold</th></tr>
                </thead>
                <tbody>
                    {{range .Metrics}}
                    <tr>
                        <td>{{.DisplayName}}</td>
                        <td>{{.FormattedValue}}</td>
                        <td class="score-cell">{{printf "%.1f" .Score}}</td>
                        <td class="citation">{{.Threshold}} <a href="{{.CitationURL}}">[{{.CitationShort}}]</a></td>
                    </tr>
                    {{end}}
                </tbody>
            </table>
        </div>
        {{end}}
    </section>

    {{if .HasBaseline}}
    <section class="trends">
        <h2>Score Trends</h2>
        {{.TrendChartSVG}}
    </section>
    {{end}}

    <section class="recommendations">
        <h2>Top Recommendations</h2>
        {{range .Recommendations}}
        <div class="recommendation">
            <h3>{{.Rank}}. {{.Summary}}</h3>
            <p><strong>Impact:</strong> +{{printf "%.1f" .ScoreImprovement}} points</p>
            <p><strong>Effort:</strong> {{.Effort}}</p>
            <p><strong>Action:</strong> {{.Action}}</p>
        </div>
        {{end}}
    </section>

    <footer>
        <p>Generated by ARS v{{.Version}} on {{.GeneratedAt}}</p>
        <h3>Research Citations</h3>
        <ul class="citations">
            {{range .Citations}}
            <li><a href="{{.URL}}">{{.Title}}</a> - {{.Description}}</li>
            {{end}}
        </ul>
    </footer>
</body>
</html>
```

### Scoring Configuration for C4
```go
// Add to DefaultConfig() in config.go
"C4": {
    Name:   "Documentation Quality",
    Weight: 0.15,
    Metrics: []MetricThresholds{
        {
            Name:   "readme_word_count",
            Weight: 0.15,
            Breakpoints: []Breakpoint{
                {Value: 0, Score: 1},
                {Value: 100, Score: 3},
                {Value: 300, Score: 6},
                {Value: 500, Score: 8},
                {Value: 1000, Score: 10},
            },
        },
        {
            Name:   "comment_density",
            Weight: 0.20,
            Breakpoints: []Breakpoint{
                {Value: 0, Score: 1},
                {Value: 5, Score: 3},
                {Value: 10, Score: 6},
                {Value: 15, Score: 8},
                {Value: 25, Score: 10},  // 15-25% is ideal
                {Value: 40, Score: 6},   // Too many comments is also bad
            },
        },
        {
            Name:   "api_doc_coverage",
            Weight: 0.25,
            Breakpoints: []Breakpoint{
                {Value: 0, Score: 1},
                {Value: 30, Score: 3},
                {Value: 50, Score: 6},
                {Value: 80, Score: 8},
                {Value: 100, Score: 10},
            },
        },
        {
            Name:   "changelog_present",
            Weight: 0.10,
            Breakpoints: []Breakpoint{
                {Value: 0, Score: 3},  // Not present
                {Value: 1, Score: 10}, // Present
            },
        },
        {
            Name:   "examples_present",
            Weight: 0.15,
            Breakpoints: []Breakpoint{
                {Value: 0, Score: 3},
                {Value: 1, Score: 10},
            },
        },
        {
            Name:   "contributing_present",
            Weight: 0.10,
            Breakpoints: []Breakpoint{
                {Value: 0, Score: 3},
                {Value: 1, Score: 10},
            },
        },
        {
            Name:   "diagrams_present",
            Weight: 0.05,
            Breakpoints: []Breakpoint{
                {Value: 0, Score: 5},
                {Value: 1, Score: 10},
            },
        },
    },
},
```

### CLI Flag Integration
```go
// Source: PRD CLI requirements
var (
    enableC4LLM bool
    outputHTML  bool
    baseline    string
)

func init() {
    scanCmd.Flags().BoolVar(&enableC4LLM, "enable-c4-llm", false,
        "Enable LLM-based C4 content quality evaluation (costs ~$0.05-0.50)")
    scanCmd.Flags().BoolVar(&outputHTML, "output-html", false,
        "Generate self-contained HTML report")
    scanCmd.Flags().StringVar(&baseline, "baseline", "",
        "Path to previous JSON output for trend comparison")
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Beta prompt caching API | GA prompt caching (no beta prefix) | 2025 | SDK simplified: `client.Messages.New()` not `client.beta.promptCaching.messages.create()` |
| Separate CSS files | Go `embed` directive | Go 1.16 (2021) | Single-binary deployment, no file system dependencies |
| External charting JS | Pure Go SVG generation | go-charts matured 2024 | No CDN dependencies, offline rendering |
| Manual XSS escaping | html/template auto-escaping | Always in Go | Contextual escaping for HTML, JS, CSS, URL |

**Key API change:** Anthropic prompt caching is now GA with workspace-level isolation (changed Feb 5, 2026). No beta flags needed.

## Open Questions

1. **Multi-language docstring parsing**
   - What we know: Need to detect JSDoc, godoc, Python docstrings
   - What's unclear: Exact patterns for "meaningful" vs "trivial" docstrings
   - Recommendation: Start with presence detection (has/hasn't docstring), not quality. Quality is LLM's job.

2. **Diagram detection accuracy**
   - What we know: Look for .png/.svg in docs/, mermaid blocks in README
   - What's unclear: False positive rate on unrelated images
   - Recommendation: Check for common diagram keywords in filenames (architecture, diagram, flow, sequence).

3. **Cost estimation precision**
   - What we know: Can estimate based on file sizes and sample count
   - What's unclear: Actual token count varies by content
   - Recommendation: Show estimate with "typically $X-$Y" range. Actual cost shown in output.

4. **Trend chart visualization**
   - What we know: Need to show score changes over time
   - What's unclear: Best chart type (line, bar, delta bars)
   - Recommendation: Use go-charts line chart showing composite + per-category scores. Simple is better.

## Sources

### Primary (HIGH confidence)
- [Anthropic Go SDK](https://github.com/anthropics/anthropic-sdk-go) - Official SDK, verified API patterns
- [Anthropic Prompt Caching Docs](https://platform.claude.com/docs/en/build-with-claude/prompt-caching) - Exact API syntax, pricing, minimum tokens
- [Go html/template](https://pkg.go.dev/html/template) - XSS escaping behavior, contextual encoding
- [vicanso/go-charts](https://pkg.go.dev/github.com/vicanso/go-charts/v2) - RadarRender API, SVG output
- Existing codebase: `internal/analyzer/c5_temporal.go` - repo-level analyzer pattern
- Existing codebase: `internal/output/terminal.go`, `json.go` - output renderer patterns
- Existing codebase: `internal/recommend/recommend.go` - impact descriptions, action templates

### Secondary (MEDIUM confidence)
- [Semgrep Go XSS Cheatsheet](https://semgrep.dev/docs/cheat-sheets/go-xss) - XSS prevention patterns
- [LLM-as-a-Judge Guide](https://www.evidentlyai.com/llm-guide/llm-as-a-judge) - Prompt design best practices
- [Langfuse LLM Evaluation](https://langfuse.com/docs/evaluation/evaluation-methods/llm-as-a-judge) - Rubric design patterns

### Tertiary (LOW confidence)
- PRD C4 breakpoint values - Reasonable starting points, may need tuning
- Cost estimates for Haiku 4.5 - Based on published pricing, actual may vary

## Metadata

**Confidence breakdown:**
- C4 static metrics: HIGH - standard file system operations, AST parsing
- C4 LLM integration: HIGH - official SDK, documented API, verified prompt caching
- HTML generation: HIGH - Go stdlib, well-documented patterns
- SVG charts: HIGH - go-charts is mature, tested RadarRender API
- Scoring config: MEDIUM - breakpoints are reasonable but need validation

**Research date:** 2026-02-03
**Valid until:** 2026-03-03 (stable domain, SDK/API unlikely to change)
