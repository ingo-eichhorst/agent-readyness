package output

import (
	"embed"
	"fmt"
	"html/template"
	"io"
	"strings"
	"time"

	"github.com/ingo/agent-readyness/internal/recommend"
	"github.com/ingo/agent-readyness/pkg/types"
	"github.com/ingo/agent-readyness/pkg/version"
)

//go:embed templates/report.html templates/styles.css
var templateFS embed.FS

// HTMLGenerator generates HTML reports from scored results.
type HTMLGenerator struct {
	tmpl *template.Template
}

// HTMLReportData holds all data for HTML report rendering.
type HTMLReportData struct {
	ProjectName     string
	Composite       float64
	Tier            string
	TierClass       string // "ready", "assisted", "limited", "hostile"
	GeneratedAt     string
	Version         string
	RadarChartSVG   template.HTML // Safe: we generate this
	TrendChartSVG   template.HTML // Safe: we generate this
	HasTrend        bool
	Categories      []HTMLCategory
	Recommendations []HTMLRecommendation
	Citations       []Citation
	InlineCSS       template.CSS // Safe: from our template
	BadgeMarkdown   string       // Badge markdown for copy section
	BadgeURL        string       // Badge URL for preview
}

// HTMLCategory represents a category for HTML display.
type HTMLCategory struct {
	Name              string
	DisplayName       string
	Score             float64
	ScoreClass        string // "ready", "assisted", "limited"
	SubScores         []HTMLSubScore
	ImpactDescription string
	Citations         []Citation // Per-category citations
}

// HTMLSubScore represents a metric sub-score for HTML display.
type HTMLSubScore struct {
	Key                 string        // Unique key like "complexity_avg"
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
}

// HTMLRecommendation represents a recommendation for HTML display.
type HTMLRecommendation struct {
	Rank             int
	Summary          string
	ScoreImprovement float64
	Effort           string
	Action           string
}

// NewHTMLGenerator creates a generator with embedded templates.
func NewHTMLGenerator() (*HTMLGenerator, error) {
	funcMap := template.FuncMap{
		"mul": func(a, b float64) float64 { return a * b },
	}

	tmpl, err := template.New("report.html").Funcs(funcMap).ParseFS(templateFS, "templates/report.html")
	if err != nil {
		return nil, fmt.Errorf("parse template: %w", err)
	}

	return &HTMLGenerator{tmpl: tmpl}, nil
}

// GenerateReport renders an HTML report to the provided writer.
func (g *HTMLGenerator) GenerateReport(w io.Writer, scored *types.ScoredResult, recs []recommend.Recommendation, baseline *types.ScoredResult) error {
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
	data := HTMLReportData{
		ProjectName:     scored.ProjectName,
		Composite:       scored.Composite,
		Tier:            scored.Tier,
		TierClass:       tierToClass(scored.Tier),
		GeneratedAt:     time.Now().Format("2006-01-02 15:04:05"),
		Version:         version.Version,
		RadarChartSVG:   template.HTML(radarSVG), // Safe: we generated it
		TrendChartSVG:   template.HTML(trendSVG), // Safe: we generated it
		HasTrend:        baseline != nil && trendSVG != "",
		Categories:      buildHTMLCategories(scored.Categories, researchCitations),
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
	if score >= 8.0 {
		return "ready"
	}
	if score >= 6.0 {
		return "assisted"
	}
	return "limited"
}

// buildHTMLCategories converts scored categories to HTML display format.
func buildHTMLCategories(categories []types.CategoryScore, citations []Citation) []HTMLCategory {
	result := make([]HTMLCategory, 0, len(categories))

	for _, cat := range categories {
		hc := HTMLCategory{
			Name:              cat.Name,
			DisplayName:       categoryDisplayName(cat.Name),
			Score:             cat.Score,
			ScoreClass:        scoreToClass(cat.Score),
			SubScores:         buildHTMLSubScores(cat.SubScores),
			ImpactDescription: categoryImpact(cat.Name),
			Citations:         filterCitationsByCategory(citations, cat.Name),
		}
		result = append(result, hc)
	}

	return result
}

// filterCitationsByCategory returns citations matching the given category name.
func filterCitationsByCategory(citations []Citation, categoryName string) []Citation {
	var filtered []Citation
	for _, c := range citations {
		if c.Category == categoryName {
			filtered = append(filtered, c)
		}
	}
	return filtered
}

// buildHTMLSubScores converts sub-scores to HTML display format.
func buildHTMLSubScores(subScores []types.SubScore) []HTMLSubScore {
	result := make([]HTMLSubScore, 0, len(subScores))

	for _, ss := range subScores {
		// Skip metrics with zero weight (e.g., deprecated overall_score in C7)
		if ss.Weight == 0.0 {
			continue
		}

		desc := getMetricDescription(ss.MetricName)
		hss := HTMLSubScore{
			Key:                 ss.MetricName,
			MetricName:          ss.MetricName,
			DisplayName:         metricDisplayName(ss.MetricName),
			RawValue:            ss.RawValue,
			FormattedValue:      formatMetricValue(ss.MetricName, ss.RawValue, ss.Available),
			Score:               ss.Score,
			ScoreClass:          scoreToClass(ss.Score),
			WeightPct:           ss.Weight * 100,
			Available:           ss.Available,
			BriefDescription:    desc.Brief,
			DetailedDescription: desc.Detailed,
			ShouldExpand:        ss.Score < desc.Threshold,
		}
		result = append(result, hss)
	}

	return result
}

// buildHTMLRecommendations converts recommendations to HTML display format.
func buildHTMLRecommendations(recs []recommend.Recommendation) []HTMLRecommendation {
	result := make([]HTMLRecommendation, 0, len(recs))

	for _, rec := range recs {
		hr := HTMLRecommendation{
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
		"complexity_avg":            "Complexity avg",
		"func_length_avg":           "Func length avg",
		"file_size_avg":             "File size avg",
		"afferent_coupling_avg":     "Afferent coupling",
		"efferent_coupling_avg":     "Efferent coupling",
		"duplication_rate":          "Duplication rate",
		"max_dir_depth":             "Max dir depth",
		"module_fanout_avg":         "Module fanout avg",
		"circular_deps":             "Circular deps",
		"import_complexity_avg":     "Import complexity",
		"dead_exports":              "Dead exports",
		"test_to_code_ratio":        "Test-to-code ratio",
		"coverage_percent":          "Coverage",
		"test_isolation":            "Test isolation",
		"assertion_density_avg":     "Assertion density",
		"test_file_ratio":           "Test file ratio",
		"type_annotation_coverage":  "Type annotations",
		"naming_consistency":        "Naming consistency",
		"magic_number_ratio":        "Magic numbers",
		"type_strictness":           "Type strictness",
		"null_safety":               "Null safety",
		"churn_rate":                "Churn rate",
		"temporal_coupling_pct":     "Temporal coupling",
		"author_fragmentation":      "Author fragmentation",
		"commit_stability":          "Commit stability",
		"hotspot_concentration":     "Hotspot concentration",
		"readme_word_count":         "README word count",
		"comment_density":           "Comment density",
		"api_doc_coverage":          "API doc coverage",
		"changelog_present":         "CHANGELOG",
		"examples_present":          "Examples",
		"contributing_present":      "CONTRIBUTING",
		"diagrams_present":          "Diagrams",
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
