// Package output renders analysis results to various output formats (terminal, JSON, HTML, badges).
//
// HTML report generation uses Go's embedded template system (//go:embed) to bundle
// templates at compile time, producing self-contained binaries with no runtime dependencies.
// The generated HTML is fully standalone: all CSS, JavaScript, and chart rendering code
// is inlined, allowing reports to be viewed offline without external CDN dependencies.
//
// CRITICAL MAINTENANCE NOTE: Templates are embedded at compile time. After modifying
// templates/report.html or templates/styles.css, you MUST rebuild the binary with
// `go build -o ars .` for changes to take effect. Hot-reloading does NOT work.
//
// Security: All user-generated content (file paths, evidence, scores) is sanitized via
// template.JSEscape() before embedding in <script> tags to prevent XSS attacks.
package output

import (
	"embed"
	"fmt"
	"html/template"
	"io"
	"strings"
	"time"

	"github.com/ingo-eichhorst/agent-readyness/internal/recommend"
	"github.com/ingo-eichhorst/agent-readyness/internal/scoring"
	"github.com/ingo-eichhorst/agent-readyness/pkg/types"
	"github.com/ingo-eichhorst/agent-readyness/pkg/version"
)

// Constants for HTML report generation.
const (
	promptScoreThreshold = 9.0
	weightToPercent      = 100.0
)

//go:embed templates/report.html templates/styles.css
var templateFS embed.FS

// htmlGenerator generates HTML reports from scored results.
type htmlGenerator struct {
	tmpl *template.Template
}

// htmlReportData holds all data for HTML report rendering.
type htmlReportData struct {
	ProjectName     string
	Composite       float64
	Tier            string
	TierClass       string // "ready", "assisted", "limited", "hostile"
	GeneratedAt     string
	Version         string
	RadarChartSVG   template.HTML // Safe: we generate this
	TrendChartSVG   template.HTML // Safe: we generate this
	HasTrend        bool
	Categories      []htmlCategory
	Recommendations []htmlRecommendation
	Citations       []citation
	InlineCSS       template.CSS // Safe: from our template
	BadgeMarkdown   string       // Badge markdown for copy section
	BadgeURL        string       // Badge URL for preview
}

// htmlCategory represents a category for HTML display.
type htmlCategory struct {
	Name              string
	DisplayName       string
	Score             float64
	ScoreClass        string // "ready", "assisted", "limited"
	Available         bool   // whether category data is available
	SubScores         []htmlSubScore
	ImpactDescription string
	Citations         []citation // Per-category citations
}

// htmlSubScore represents a metric sub-score for HTML display.
type htmlSubScore struct {
	Key                 string // Unique key like "complexity_avg"
	MetricName          string
	DisplayName         string
	RawValue            float64
	FormattedValue      string
	Score               float64
	ScoreClass          string
	WeightPct           float64 // Weight as percentage (0-100)
	Available           bool
	BriefDescription    string        // Always visible, 1-2 sentences
	DetailedDescription template.HTML // Expandable content with sections
	ShouldExpand        bool          // true if score below threshold
	TraceHTML           template.HTML // Pre-rendered modal body content
	HasTrace            bool          // Whether trace data is available
	PromptHTML          template.HTML // Pre-rendered improvement prompt modal content
	HasPrompt           bool          // Whether prompt data is available
}

// TraceData holds analysis data needed for rendering call trace modals.
// Passed to GenerateReport; can be nil when trace rendering is not needed.
type TraceData struct {
	ScoringConfig   *scoring.ScoringConfig
	AnalysisResults []*types.AnalysisResult
	Languages       []string // Detected project languages for build/test commands
}

// htmlRecommendation represents a recommendation for HTML display.
type htmlRecommendation struct {
	Rank             int
	Summary          string
	ScoreImprovement float64
	Effort           string
	Action           string
}

// NewHTMLGenerator creates a generator with embedded templates.
func NewHTMLGenerator() (*htmlGenerator, error) {
	funcMap := template.FuncMap{
		"mul": func(a, b float64) float64 { return a * b },
	}

	tmpl, err := template.New("report.html").Funcs(funcMap).ParseFS(templateFS, "templates/report.html")
	if err != nil {
		return nil, fmt.Errorf("parse template: %w", err)
	}

	return &htmlGenerator{tmpl: tmpl}, nil
}

// GenerateReport renders an HTML report to the provided writer.
//
// trace can be nil when trace rendering is not needed (backward compatible).
//
// Rendering strategy: Generate all dynamic content (charts, trace modals, evidence)
// server-side and inline into the HTML template. This produces a self-contained
// single-file report that works offline and doesn't rely on external JavaScript
// frameworks or CDNs. Charts use vanilla Canvas 2D API, not heavyweight libraries.
//
// The report includes:
// - Radar chart (7-axis category scores visualization)
// - Trend chart (baseline comparison if --baseline provided)
// - Per-metric trace modals (breakpoint interpolation details)
// - Evidence tables (top offenders per metric)
// - Improvement prompts (actionable suggestions with build commands)
func (g *htmlGenerator) GenerateReport(w io.Writer, scored *types.ScoredResult, recs []recommend.Recommendation, baseline *types.ScoredResult, trace *TraceData) error {
	// Load CSS
	cssBytes, err := templateFS.ReadFile("templates/styles.css")
	if err != nil {
		return fmt.Errorf("read CSS: %w", err)
	}

	// Generate charts
	radarSVG, err := generateRadarChart(scored.Categories)
	if err != nil {
		return fmt.Errorf("generate radar chart: %w", err)
	}

	var trendSVG string
	if baseline != nil {
		trendSVG, err = generateTrendChart(scored, baseline)
		if err != nil {
			return fmt.Errorf("generate trend chart: %w", err)
		}
	}

	// Generate badge info
	badge := GenerateBadge(scored)

	// Build template data
	data := htmlReportData{
		ProjectName:     scored.ProjectName,
		Composite:       scored.Composite,
		Tier:            scored.Tier,
		TierClass:       tierToClass(scored.Tier),
		GeneratedAt:     time.Now().Format("2006-01-02 15:04:05"),
		Version:         version.Version,
		RadarChartSVG:   template.HTML(radarSVG), // Safe: we generated it
		TrendChartSVG:   template.HTML(trendSVG), // Safe: we generated it
		HasTrend:        baseline != nil && trendSVG != "",
		Categories:      buildHTMLCategories(scored.Categories, researchCitations, trace),
		Recommendations: buildHTMLRecommendations(recs),
		Citations:       researchCitations,
		InlineCSS:       template.CSS(string(cssBytes)), // Safe: from our template
		BadgeMarkdown:   badge.Markdown,
		BadgeURL:        badge.URL,
	}

	return g.tmpl.Execute(w, data)
}

// tierToClass converts tier string to CSS class name.
func tierToClass(tier string) string {
	switch tier {
	case "Agent-Ready":
		return "ready"
	case "Agent-Assisted":
		return "assisted"
	case "Agent-Limited":
		return "limited"
	default:
		return "hostile"
	}
}

// scoreToClass converts a score to a CSS class name.
func scoreToClass(score float64) string {
	if score >= scoreGreenMin {
		return "ready"
	}
	if score >= scoreYellowMin {
		return "assisted"
	}
	return "limited"
}

// buildHTMLCategories converts scored categories to HTML display format.
func buildHTMLCategories(categories []types.CategoryScore, citations []citation, trace *TraceData) []htmlCategory {
	result := make([]htmlCategory, 0, len(categories))

	for _, cat := range categories {
		hc := htmlCategory{
			Name:              cat.Name,
			DisplayName:       categoryDisplayName(cat.Name),
			Score:             cat.Score,
			ScoreClass:        scoreToClass(cat.Score),
			Available:         cat.Score >= 0, // Infer from score
			SubScores:         buildHTMLSubScores(cat.Name, cat.SubScores, trace),
			ImpactDescription: categoryImpact(cat.Name),
			Citations:         filterCitationsByCategory(citations, cat.Name),
		}
		result = append(result, hc)
	}

	return result
}

// filterCitationsByCategory returns citations matching the given category name.
func filterCitationsByCategory(citations []citation, categoryName string) []citation {
	var filtered []citation
	for _, c := range citations {
		if c.Category == categoryName {
			filtered = append(filtered, c)
		}
	}
	return filtered
}

// buildHTMLSubScores converts sub-scores to HTML display format.
//
// This function handles multiple rendering concerns in a single pass:
// 1. Basic metric display (raw value, score, weight)
// 2. Evidence attachment (top offenders per metric from scoring phase)
// 3. Trace modal generation (breakpoint interpolation details for debugging)
// 4. Improvement prompt generation (actionable suggestions with commands)
//
// Why consolidate? It minimizes passes over the data and keeps related display
// logic together. The alternative (separate functions for each concern) would
// require multiple iterations and make it harder to ensure consistency.
func buildHTMLSubScores(categoryName string, subScores []types.SubScore, trace *TraceData) []htmlSubScore {
	result := make([]htmlSubScore, 0, len(subScores))

	// Extract C7 metric results if this is the C7 category and trace data is available
	//
	// C7 metrics use live agent evaluation, so traces include actual prompts sent to
	// Claude CLI and the responses received. This differs from C1-C6 which use static
	// breakpoint interpolation. Extracting C7 results separately handles this distinction.
	var c7MetricResults []types.C7MetricResult
	if categoryName == "C7" && trace != nil && trace.AnalysisResults != nil {
		for _, ar := range trace.AnalysisResults {
			if ar.Category == "C7" {
				if c7Raw, ok := ar.Metrics["c7"]; ok {
					if c7m, ok := c7Raw.(*types.C7Metrics); ok {
						c7MetricResults = c7m.MetricResults
					}
				}
				break
			}
		}
	}

	for _, ss := range subScores {
		// Skip metrics with zero weight (e.g., deprecated overall_score in C7)
		//
		// Zero-weight metrics are kept in the data model for backward compatibility
		// (old JSON files may reference them) but hidden from HTML display to avoid
		// confusing users with inactive metrics. This is preferable to removing them
		// entirely, which would break JSON schema compatibility.
		if ss.Weight == 0.0 {
			continue
		}

		desc := getMetricDescription(ss.MetricName)
		hss := htmlSubScore{
			Key:                 ss.MetricName,
			MetricName:          ss.MetricName,
			DisplayName:         metricDisplayName(ss.MetricName),
			RawValue:            ss.RawValue,
			FormattedValue:      formatMetricValue(ss.MetricName, ss.RawValue, ss.Available),
			Score:               ss.Score,
			ScoreClass:          scoreToClass(ss.Score),
			WeightPct:           ss.Weight * weightToPercent,
			Available:           ss.Available,
			BriefDescription:    desc.Brief,
			DetailedDescription: desc.Detailed,
			ShouldExpand:        ss.Score < desc.Threshold,
		}

		// Populate C7 trace data if available
		if categoryName == "C7" && len(c7MetricResults) > 0 {
			traceHTML := renderC7Trace(ss.MetricName, c7MetricResults)
			if traceHTML != "" {
				hss.TraceHTML = template.HTML(traceHTML)
				hss.HasTrace = true
			}
		}

		// Populate C1-C6 breakpoint trace data
		var breakpoints []scoring.Breakpoint
		if categoryName != "C7" && trace != nil && trace.ScoringConfig != nil {
			catCfg := trace.ScoringConfig.Category(categoryName)
			for _, mt := range catCfg.Metrics {
				if mt.Name == ss.MetricName {
					breakpoints = mt.Breakpoints
					break
				}
			}
			traceHTML := renderBreakpointTrace(ss.MetricName, ss.RawValue, ss.Score, breakpoints, ss.Evidence)
			if traceHTML != "" {
				hss.TraceHTML = template.HTML(traceHTML)
				hss.HasTrace = true
			}
		}

		// Populate improvement prompt (for metrics scoring below 9.0)
		if ss.Available && ss.Score < promptScoreThreshold && trace != nil {
			// Determine language for build commands
			lang := ""
			if len(trace.Languages) > 0 {
				lang = trace.Languages[0]
			}

			// For C7 metrics, look up breakpoints separately
			if categoryName == "C7" && trace.ScoringConfig != nil && len(breakpoints) == 0 {
				catCfg := trace.ScoringConfig.Category(categoryName)
				for _, mt := range catCfg.Metrics {
					if mt.Name == ss.MetricName {
						breakpoints = mt.Breakpoints
						break
					}
				}
			}

			// Calculate target
			targetValue, targetScore := nextTarget(ss.Score, breakpoints)

			promptHTML := renderImprovementPrompt(promptParams{
				CategoryName:    categoryName,
				CategoryDisplay: categoryDisplayName(categoryName),
				CategoryImpact:  categoryImpact(categoryName),
				MetricName:      ss.MetricName,
				MetricDisplay:   metricDisplayName(ss.MetricName),
				RawValue:        ss.RawValue,
				FormattedValue:  formatMetricValue(ss.MetricName, ss.RawValue, ss.Available),
				Score:           ss.Score,
				TargetScore:     targetScore,
				TargetValue:     targetValue,
				HasBreakpoints:  len(breakpoints) > 0,
				Evidence:        ss.Evidence,
				Language:        lang,
			})
			if promptHTML != "" {
				hss.PromptHTML = template.HTML(promptHTML)
				hss.HasPrompt = true
			}
		}

		result = append(result, hss)
	}

	return result
}

// buildHTMLRecommendations converts recommendations to HTML display format.
func buildHTMLRecommendations(recs []recommend.Recommendation) []htmlRecommendation {
	result := make([]htmlRecommendation, 0, len(recs))

	for _, rec := range recs {
		hr := htmlRecommendation{
			Rank:             rec.Rank,
			Summary:          rec.Summary,
			ScoreImprovement: rec.ScoreImprovement,
			Effort:           rec.Effort,
			Action:           rec.Action,
		}
		result = append(result, hr)
	}

	return result
}

// categoryDisplayName returns human-readable category name.
func categoryDisplayName(name string) string {
	names := map[string]string{
		"C1": "C1: Code Health",
		"C2": "C2: Semantic Explicitness",
		"C3": "C3: Architecture",
		"C4": "C4: Documentation Quality",
		"C5": "C5: Temporal Dynamics",
		"C6": "C6: Testing",
		"C7": "C7: Agent Evaluation",
	}
	if dn, ok := names[name]; ok {
		return dn
	}
	return name
}

// metricDisplayName returns human-readable metric name.
func metricDisplayName(name string) string {
	// Reuse from terminal.go mapping
	names := map[string]string{
		"complexity_avg":                   "Complexity avg",
		"func_length_avg":                  "Func length avg",
		"file_size_avg":                    "File size avg",
		"afferent_coupling_avg":            "Afferent coupling",
		"efferent_coupling_avg":            "Efferent coupling",
		"duplication_rate":                 "Duplication rate",
		"max_dir_depth":                    "Max dir depth",
		"module_fanout_avg":                "Module fanout avg",
		"circular_deps":                    "Circular deps",
		"import_complexity_avg":            "Import complexity",
		"dead_exports":                     "Dead exports",
		"test_to_code_ratio":               "Test-to-code ratio",
		"coverage_percent":                 "Coverage",
		"test_isolation":                   "Test isolation",
		"assertion_density_avg":            "Assertion density",
		"test_file_ratio":                  "Test file ratio",
		"type_annotation_coverage":         "Type annotations",
		"naming_consistency":               "Naming consistency",
		"magic_number_ratio":               "Magic numbers",
		"type_strictness":                  "Type strictness",
		"null_safety":                      "Null safety",
		"churn_rate":                       "Churn rate",
		"temporal_coupling_pct":            "Temporal coupling",
		"author_fragmentation":             "Author fragmentation",
		"commit_stability":                 "Commit stability",
		"hotspot_concentration":            "Hotspot concentration",
		"readme_word_count":                "README word count",
		"comment_density":                  "Comment density",
		"api_doc_coverage":                 "API doc coverage",
		"changelog_present":                "CHANGELOG",
		"examples_present":                 "Examples",
		"contributing_present":             "CONTRIBUTING",
		"diagrams_present":                 "Diagrams",
		"task_execution_consistency":       "Task Execution Consistency",
		"code_behavior_comprehension":      "Code Behavior Comprehension",
		"cross_file_navigation":            "Cross-File Navigation",
		"identifier_interpretability":      "Identifier Interpretability",
		"documentation_accuracy_detection": "Documentation Accuracy Detection",
	}
	if dn, ok := names[name]; ok {
		return dn
	}
	return strings.ReplaceAll(name, "_", " ")
}

// formatMetricValue formats a metric value for display.
func formatMetricValue(name string, value float64, available bool) string {
	if !available {
		return "n/a"
	}

	switch name {
	case "duplication_rate", "type_annotation_coverage", "naming_consistency",
		"null_safety", "temporal_coupling_pct", "hotspot_concentration",
		"comment_density", "api_doc_coverage", "coverage_percent", "test_isolation":
		return fmt.Sprintf("%.1f%%", value)
	case "test_to_code_ratio":
		return fmt.Sprintf("%.2f", value)
	case "complexity_avg", "func_length_avg", "file_size_avg", "module_fanout_avg",
		"import_complexity_avg", "assertion_density_avg", "churn_rate",
		"author_fragmentation", "commit_stability":
		return fmt.Sprintf("%.1f", value)
	case "max_dir_depth", "circular_deps", "dead_exports", "readme_word_count":
		return fmt.Sprintf("%.0f", value)
	case "changelog_present", "examples_present", "contributing_present",
		"diagrams_present", "type_strictness":
		if value >= 1 {
			return "yes"
		}
		return "no"
	case "magic_number_ratio":
		return fmt.Sprintf("%.1f/kLOC", value)
	default:
		return fmt.Sprintf("%.1f", value)
	}
}

// categoryImpact returns agent-readiness impact description for a category.
//
// These descriptions are intentionally concise (max 80 characters) to fit in:
// - HTML report tooltips (desktop hover, mobile tap)
// - Terminal output summary lines
// - Badge alt-text for accessibility
//
// Each description explains WHY the category matters for AI agents specifically,
// not just general code quality. For example, C1 doesn't say "complexity is bad";
// it says "complexity prevents agents from reasoning safely" - the agent impact
// is the key insight. These descriptions must be updated when adding categories.
func categoryImpact(name string) string {
	impacts := map[string]string{
		"C1": "Lower complexity and smaller functions help agents reason about and modify code safely.",
		"C2": "Explicit types and consistent naming enable agents to understand code semantics without guessing.",
		"C3": "Clear module boundaries and low coupling allow agents to make isolated, safe changes.",
		"C4": "Quality documentation helps agents understand project context and API contracts.",
		"C5": "Stable, low-churn code reduces agent merge conflicts and increases change confidence.",
		"C6": "Comprehensive tests let agents verify their changes don't break existing functionality.",
		"C7": "Direct measurement of how well AI agents perform real-world coding tasks in your codebase.",
	}
	return impacts[name]
}
