// Package output renders analysis results to various output formats (terminal, JSON, HTML, badges).
//
// Terminal rendering uses hierarchical display with automatic color encoding (green/yellow/red)
// based on score thresholds. Colors convey metric health at a glance without requiring users to
// interpret numeric scores. NO_COLOR environment variable support ensures compatibility with
// screen readers, CI/CD pipelines, and accessibility tools per https://no-color.org standards.
//
// The package handles evidence formatting, section expansion/collapse for verbose output,
// and tier classification display with visual indicators (emojis, color, progress bars).
package output

import (
	"fmt"
	"io"
	"strings"

	"github.com/fatih/color"

	"github.com/ingo-eichhorst/agent-readyness/internal/recommend"
	"github.com/ingo-eichhorst/agent-readyness/pkg/types"
)

// Display thresholds for terminal score coloring.
const (
	scoreGreenMin  = 8.0 // Score at or above: green (Agent-Ready)
	scoreYellowMin = 6.0 // Score at or above: yellow (Agent-Assisted)
)

// Display limits for verbose terminal output.
const (
	verboseTopN      = 5   // Top offenders shown in verbose mode
	coupledPairsTopN = 10  // Max coupled pairs in C5 verbose output
	truncateShort    = 200 // Truncation limit for short text (prompts)
	truncateLong     = 500 // Truncation limit for long text (responses)
	separatorWide    = 60  // Wide separator width (C7 debug)
	separatorNarrow  = 50  // Narrow separator width (C7 debug)
)

// Recommendation impact thresholds.
const (
	recImpactScaleToScore = 20.0 // Scales 0.5 point improvement to score 10.0
	recHighImpact         = 0.5  // Score improvement >= this is high impact (green)
	recModerateImpact     = 0.2  // Score improvement >= this is moderate (yellow)
)

// RenderSummary prints a formatted scan summary to w.
func RenderSummary(w io.Writer, result *types.ScanResult, analysisResults []*types.AnalysisResult, verbose bool) {
	bold := color.New(color.Bold)
	green := color.New(color.FgGreen)
	yellow := color.New(color.FgYellow)

	// Header
	bold.Fprintf(w, "ARS Scan: %s\n", result.RootDir)
	fmt.Fprintln(w, "────────────────────────────────────────")

	// Total count
	fmt.Fprintf(w, "Files discovered: %d\n", result.TotalFiles)

	// Source and test counts (always shown)
	green.Fprintf(w, "  Source files:        %d\n", result.SourceCount)
	yellow.Fprintf(w, "  Test files:          %d\n", result.TestCount)

	// Per-language file counts (if multi-language)
	if len(result.PerLanguage) > 1 || (len(result.PerLanguage) == 1 && result.PerLanguage[types.LangGo] == 0) {
		fmt.Fprintln(w, "  Per-language source files:")
		for lang, count := range result.PerLanguage {
			fmt.Fprintf(w, "    %-12s %d\n", string(lang)+":", count)
		}
	}

	// Excluded categories (only shown if non-zero)
	if result.GeneratedCount > 0 {
		fmt.Fprintf(w, "  Generated (excluded): %d\n", result.GeneratedCount)
	}
	if result.VendorCount > 0 {
		fmt.Fprintf(w, "  Vendor (excluded):    %d\n", result.VendorCount)
	}
	if result.GitignoreCount > 0 {
		fmt.Fprintf(w, "  Gitignored (excluded): %d\n", result.GitignoreCount)
	}

	// Verbose: list individual files
	if verbose {
		fmt.Fprintln(w)
		bold.Fprintln(w, "Discovered files:")
		for _, f := range result.Files {
			tag := f.Class.String()
			suffix := ""
			if f.Class == types.ClassExcluded && f.ExcludeReason != "" {
				suffix = fmt.Sprintf(" (%s)", f.ExcludeReason)
			}
			fmt.Fprintf(w, "  [%s] %s%s\n", tag, f.RelPath, suffix)
		}
	}

	// Render analysis results
	for _, ar := range analysisResults {
		switch ar.Category {
		case "C1":
			renderC1(w, ar, verbose)
		case "C2":
			renderC2(w, ar, verbose)
		case "C3":
			renderC3(w, ar, verbose)
		case "C4":
			renderC4(w, ar, verbose)
		case "C5":
			renderC5(w, ar, verbose)
		case "C6":
			renderC6(w, ar, verbose)
		case "C7":
			renderC7(w, ar, verbose)
		}
	}
}

// colorForFloat returns a color function based on threshold values.
// Values <= greenMax are green, <= yellowMax are yellow, above are red.
func colorForFloat(val, greenMax, yellowMax float64) *color.Color {
	if val <= greenMax {
		return color.New(color.FgGreen)
	}
	if val <= yellowMax {
		return color.New(color.FgYellow)
	}
	return color.New(color.FgRed)
}

// colorForInt returns a color function based on threshold values.
func colorForInt(val, greenMax, yellowMax int) *color.Color {
	if val <= greenMax {
		return color.New(color.FgGreen)
	}
	if val <= yellowMax {
		return color.New(color.FgYellow)
	}
	return color.New(color.FgRed)
}

// colorForFloatInverse returns a color where higher is better (e.g., coverage).
func colorForFloatInverse(val, redBelow, yellowBelow float64) *color.Color {
	if val < redBelow {
		return color.New(color.FgRed)
	}
	if val < yellowBelow {
		return color.New(color.FgYellow)
	}
	return color.New(color.FgGreen)
}

// colorForIntInverse returns a color where higher is better (for 1-10 scale metrics).
func colorForIntInverse(val, redBelow, yellowBelow int) *color.Color {
	if val < redBelow {
		return color.New(color.FgRed)
	}
	if val < yellowBelow {
		return color.New(color.FgYellow)
	}
	return color.New(color.FgGreen)
}

// categoryDisplayNames maps category identifiers to human-readable labels.
var categoryDisplayNames = map[string]string{
	"C1": "Code Health",
	"C2": "Semantic Explicitness",
	"C3": "Architecture",
	"C4": "Documentation Quality",
	"C5": "Temporal Dynamics",
	"C6": "Testing",
	"C7": "Agent Evaluation",
}

// metricDisplayNames maps metric identifiers to human-readable labels.
var metricDisplayNames = map[string]string{
	"complexity_avg":        "Complexity avg",
	"func_length_avg":       "Func length avg",
	"file_size_avg":         "File size avg",
	"afferent_coupling_avg": "Afferent coupling",
	"efferent_coupling_avg": "Efferent coupling",
	"duplication_rate":      "Duplication rate",
	"max_dir_depth":         "Max dir depth",
	"module_fanout_avg":     "Module fanout avg",
	"circular_deps":         "Circular deps",
	"import_complexity_avg": "Import complexity",
	"dead_exports":          "Dead exports",
	"test_to_code_ratio":    "Test-to-code ratio",
	"coverage_percent":      "Coverage",
	"test_isolation":        "Test isolation",
	"assertion_density_avg": "Assertion density",
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
	// C4 metrics
	"readme_word_count":     "README word count",
	"comment_density":       "Comment density",
	"api_doc_coverage":      "API doc coverage",
	"changelog_present":     "CHANGELOG",
	"examples_present":      "Examples",
	"contributing_present":  "CONTRIBUTING",
	"diagrams_present":      "Diagrams",
	// C7 metrics
	"intent_clarity":          "Intent clarity",
	"modification_confidence": "Modification confidence",
	"cross_file_coherence":    "Cross-file coherence",
	"semantic_completeness":   "Semantic completeness",
}

// RenderScores prints a formatted scoring section showing per-category scores,
// composite score, and tier rating.
func RenderScores(w io.Writer, scored *types.ScoredResult, verbose bool) {
	bold := color.New(color.Bold)

	fmt.Fprintln(w)
	bold.Fprintln(w, "Agent Readiness Score")
	fmt.Fprintln(w, "\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550")

	for _, cat := range scored.Categories {
		displayName := categoryDisplayNames[cat.Name]
		if displayName == "" {
			displayName = cat.Name
		}
		label := fmt.Sprintf("%s: %-20s", cat.Name, displayName)

		// Check for unavailable category
		if cat.Score < 0 {
			color.New(color.FgHiBlack).Fprintf(w, "  %sn/a\n", label)
			continue
		}

		sc := scoreColor(cat.Score)
		sc.Fprintf(w, "  %s%.1f / 10\n", label, cat.Score)

		if verbose {
			renderSubScores(w, cat.SubScores)
		}
	}

	fmt.Fprintln(w, "\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500")

	// Composite score
	cc := scoreColor(scored.Composite)
	cc.Fprintf(w, "  Composite Score:          %.1f / 10\n", scored.Composite)

	// Tier rating
	tc := tierColor(scored.Tier)
	fmt.Fprintf(w, "  Rating:                   ")
	tc.Fprintln(w, scored.Tier)
}

// renderSubScores prints per-metric sub-score details indented beneath a category.
func renderSubScores(w io.Writer, subScores []types.SubScore) {
	for _, ss := range subScores {
		// Skip metrics with zero weight (e.g., deprecated overall_score in C7)
		if ss.Weight == 0.0 {
			continue
		}

		displayName := metricDisplayNames[ss.MetricName]
		if displayName == "" {
			displayName = ss.MetricName
		}

		if !ss.Available {
			fmt.Fprintf(w, "    %-22s n/a         (%.0f%%, excluded)\n",
				displayName+":", ss.Weight*100)
			continue
		}

		fmt.Fprintf(w, "    %-22s %7.1f  ->  %-4.1f  (%.0f%%)\n",
			displayName+":", ss.RawValue, ss.Score, ss.Weight*100)
	}
}

// scoreColor returns a color based on score thresholds: green >= 8, yellow >= 6, red < 6.
func scoreColor(score float64) *color.Color {
	if score >= scoreGreenMin {
		return color.New(color.FgGreen)
	}
	if score >= scoreYellowMin {
		return color.New(color.FgYellow)
	}
	return color.New(color.FgRed)
}

// tierColor returns a color for tier badge display.
func tierColor(tier string) *color.Color {
	switch tier {
	case "Agent-Ready":
		return color.New(color.FgGreen, color.Bold)
	case "Agent-Assisted":
		return color.New(color.FgYellow, color.Bold)
	default:
		return color.New(color.FgRed, color.Bold)
	}
}

// RenderRecommendations prints ranked improvement recommendations to w.
func RenderRecommendations(w io.Writer, recs []recommend.Recommendation) {
	bold := color.New(color.Bold)
	green := color.New(color.FgGreen)

	fmt.Fprintln(w)
	bold.Fprintln(w, "Top Recommendations")
	fmt.Fprintln(w, "\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550")

	if len(recs) == 0 {
		green.Fprintln(w, "  No recommendations -- all metrics are excellent!")
		return
	}

	for i, rec := range recs {
		bold.Fprintf(w, "  %d. %s\n", rec.Rank, rec.Summary)

		impactColor := scoreColor(rec.ScoreImprovement * recImpactScaleToScore)
		if rec.ScoreImprovement >= recHighImpact {
			impactColor = color.New(color.FgGreen)
		} else if rec.ScoreImprovement >= recModerateImpact {
			impactColor = color.New(color.FgYellow)
		} else {
			impactColor = color.New(color.FgRed)
		}
		impactColor.Fprintf(w, "     Impact: +%.1f points\n", rec.ScoreImprovement)

		fmt.Fprintf(w, "     Effort: %s\n", rec.Effort)
		fmt.Fprintf(w, "     Action: %s\n", rec.Action)

		if i < len(recs)-1 {
			fmt.Fprintln(w)
		}
	}
}

// joinCycle formats a dependency cycle as "A -> B -> C -> A".
func joinCycle(cycle []string) string {
	if len(cycle) == 0 {
		return ""
	}
	return strings.Join(cycle, " -> ") + " -> " + cycle[0]
}

// RenderBadge prints the shields.io badge markdown to w.
func RenderBadge(w io.Writer, scored *types.ScoredResult) {
	if scored == nil {
		return
	}

	bold := color.New(color.Bold)
	badge := GenerateBadge(scored)

	fmt.Fprintln(w)
	bold.Fprintln(w, "Badge")
	fmt.Fprintln(w, "\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500")
	fmt.Fprintln(w, badge.Markdown)
}

// truncateString truncates s to maxLen characters, appending "..." if truncated.
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
