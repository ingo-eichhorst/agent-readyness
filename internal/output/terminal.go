package output

import (
	"fmt"
	"io"
	"sort"

	"github.com/fatih/color"

	"github.com/ingo/agent-readyness/internal/recommend"
	"github.com/ingo/agent-readyness/pkg/types"
)

// RenderSummary prints a formatted scan summary to w.
// Color is automatically disabled when w is not a TTY (e.g., piped output).
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
		case "C5":
			renderC5(w, ar, verbose)
		case "C6":
			renderC6(w, ar, verbose)
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

func renderC1(w io.Writer, ar *types.AnalysisResult, verbose bool) {
	bold := color.New(color.Bold)

	raw, ok := ar.Metrics["c1"]
	if !ok {
		return
	}
	m, ok := raw.(*types.C1Metrics)
	if !ok {
		return
	}

	fmt.Fprintln(w)
	bold.Fprintln(w, "C1: Code Health")
	fmt.Fprintln(w, "────────────────────────────────────────")

	// Complexity
	cc := colorForFloat(m.CyclomaticComplexity.Avg, 10, 20)
	cc.Fprintf(w, "  Complexity avg:      %.1f\n", m.CyclomaticComplexity.Avg)
	cm := colorForInt(m.CyclomaticComplexity.Max, 15, 30)
	cm.Fprintf(w, "  Complexity max:      %d", m.CyclomaticComplexity.Max)
	if m.CyclomaticComplexity.MaxEntity != "" {
		fmt.Fprintf(w, " (%s)", m.CyclomaticComplexity.MaxEntity)
	}
	fmt.Fprintln(w)

	// Function length
	fl := colorForFloat(m.FunctionLength.Avg, 30, 60)
	fl.Fprintf(w, "  Func length avg:     %.1f lines\n", m.FunctionLength.Avg)
	flm := colorForInt(m.FunctionLength.Max, 50, 100)
	flm.Fprintf(w, "  Func length max:     %d lines", m.FunctionLength.Max)
	if m.FunctionLength.MaxEntity != "" {
		fmt.Fprintf(w, " (%s)", m.FunctionLength.MaxEntity)
	}
	fmt.Fprintln(w)

	// File size
	fs := colorForFloat(m.FileSize.Avg, 300, 500)
	fs.Fprintf(w, "  File size avg:       %.0f lines\n", m.FileSize.Avg)
	fsm := colorForInt(m.FileSize.Max, 500, 1000)
	fsm.Fprintf(w, "  File size max:       %d lines", m.FileSize.Max)
	if m.FileSize.MaxEntity != "" {
		fmt.Fprintf(w, " (%s)", m.FileSize.MaxEntity)
	}
	fmt.Fprintln(w)

	// Duplication
	dc := colorForFloat(m.DuplicationRate, 5, 15)
	dc.Fprintf(w, "  Duplication rate:    %.1f%%\n", m.DuplicationRate)

	// Verbose: top 5 most complex and longest functions
	if verbose && len(m.Functions) > 0 {
		fmt.Fprintln(w)
		bold.Fprintln(w, "  Top complex functions:")
		byComplexity := make([]types.FunctionMetric, len(m.Functions))
		copy(byComplexity, m.Functions)
		sort.Slice(byComplexity, func(i, j int) bool {
			return byComplexity[i].Complexity > byComplexity[j].Complexity
		})
		limit := 5
		if len(byComplexity) < limit {
			limit = len(byComplexity)
		}
		for _, f := range byComplexity[:limit] {
			fmt.Fprintf(w, "    %s.%s  complexity=%d  (%s:%d)\n", f.Package, f.Name, f.Complexity, f.File, f.Line)
		}

		fmt.Fprintln(w)
		bold.Fprintln(w, "  Top longest functions:")
		byLength := make([]types.FunctionMetric, len(m.Functions))
		copy(byLength, m.Functions)
		sort.Slice(byLength, func(i, j int) bool {
			return byLength[i].LineCount > byLength[j].LineCount
		})
		limit = 5
		if len(byLength) < limit {
			limit = len(byLength)
		}
		for _, f := range byLength[:limit] {
			fmt.Fprintf(w, "    %s.%s  lines=%d  (%s:%d)\n", f.Package, f.Name, f.LineCount, f.File, f.Line)
		}
	}
}

func renderC2(w io.Writer, ar *types.AnalysisResult, verbose bool) {
	bold := color.New(color.Bold)

	raw, ok := ar.Metrics["c2"]
	if !ok {
		return
	}
	m, ok := raw.(*types.C2Metrics)
	if !ok || m.Aggregate == nil {
		return
	}

	agg := m.Aggregate

	fmt.Fprintln(w)
	bold.Fprintln(w, "C2: Semantic Explicitness")
	fmt.Fprintln(w, "────────────────────────────────────────")

	tc := colorForFloatInverse(agg.TypeAnnotationCoverage, 50, 80)
	tc.Fprintf(w, "  Type annotation:     %.1f%%\n", agg.TypeAnnotationCoverage)

	nc := colorForFloatInverse(agg.NamingConsistency, 70, 90)
	nc.Fprintf(w, "  Naming consistency:  %.1f%%\n", agg.NamingConsistency)

	mr := colorForFloat(agg.MagicNumberRatio, 5, 15)
	mr.Fprintf(w, "  Magic numbers:       %.1f per kLOC\n", agg.MagicNumberRatio)

	if agg.TypeStrictness >= 1 {
		color.New(color.FgGreen).Fprintf(w, "  Type strictness:     on\n")
	} else {
		color.New(color.FgYellow).Fprintf(w, "  Type strictness:     off\n")
	}

	ns := colorForFloatInverse(agg.NullSafety, 30, 60)
	ns.Fprintf(w, "  Null safety:         %.0f%%\n", agg.NullSafety)

	// Verbose: per-language C2 breakdown
	if verbose && len(m.PerLanguage) > 0 {
		fmt.Fprintln(w)
		bold.Fprintln(w, "  Per-language C2 breakdown:")
		for lang, lm := range m.PerLanguage {
			strict := "off"
			if lm.TypeStrictness >= 1 {
				strict = "on"
			}
			fmt.Fprintf(w, "    %-12s type=%.0f%%  naming=%.0f%%  magic=%.1f/kLOC  strict=%s  null=%.0f%%  LOC=%d\n",
				string(lang)+":", lm.TypeAnnotationCoverage, lm.NamingConsistency,
				lm.MagicNumberRatio, strict, lm.NullSafety, lm.LOC)
		}
	}
}

func renderC3(w io.Writer, ar *types.AnalysisResult, verbose bool) {
	bold := color.New(color.Bold)

	raw, ok := ar.Metrics["c3"]
	if !ok {
		return
	}
	m, ok := raw.(*types.C3Metrics)
	if !ok {
		return
	}

	fmt.Fprintln(w)
	bold.Fprintln(w, "C3: Architecture")
	fmt.Fprintln(w, "────────────────────────────────────────")

	dd := colorForInt(m.MaxDirectoryDepth, 4, 7)
	dd.Fprintf(w, "  Max directory depth: %d\n", m.MaxDirectoryDepth)
	fmt.Fprintf(w, "  Avg directory depth: %.1f\n", m.AvgDirectoryDepth)

	fo := colorForFloat(m.ModuleFanout.Avg, 5, 10)
	fo.Fprintf(w, "  Avg module fanout:   %.1f\n", m.ModuleFanout.Avg)

	circCount := len(m.CircularDeps)
	cc := colorForInt(circCount, 0, 2)
	cc.Fprintf(w, "  Circular deps:       %d\n", circCount)

	deadCount := len(m.DeadExports)
	dc := colorForInt(deadCount, 5, 20)
	dc.Fprintf(w, "  Dead exports:        %d\n", deadCount)

	// Verbose: coupling details + dead exports
	if verbose {
		if circCount > 0 {
			fmt.Fprintln(w)
			bold.Fprintln(w, "  Circular dependencies:")
			for i, cycle := range m.CircularDeps {
				fmt.Fprintf(w, "    %d. %s\n", i+1, joinCycle(cycle))
			}
		}

		if deadCount > 0 {
			fmt.Fprintln(w)
			bold.Fprintln(w, "  Dead exports:")
			for _, de := range m.DeadExports {
				fmt.Fprintf(w, "    %s %s.%s  (%s:%d)\n", de.Kind, de.Package, de.Name, de.File, de.Line)
			}
		}
	}
}

func renderC5(w io.Writer, ar *types.AnalysisResult, verbose bool) {
	bold := color.New(color.Bold)

	raw, ok := ar.Metrics["c5"]
	if !ok {
		return
	}
	m, ok := raw.(*types.C5Metrics)
	if !ok {
		return
	}

	fmt.Fprintln(w)
	bold.Fprintln(w, "C5: Temporal Dynamics")
	fmt.Fprintln(w, "────────────────────────────────────────")

	if !m.Available {
		fmt.Fprintln(w, "  Not available (no .git directory)")
		return
	}

	fmt.Fprintf(w, "  Total commits:       %d (%d-day window)\n", m.TotalCommits, m.TimeWindowDays)

	cr := colorForFloat(m.ChurnRate, 100, 300)
	cr.Fprintf(w, "  Churn rate:          %.1f lines/commit\n", m.ChurnRate)

	tc := colorForFloat(m.TemporalCouplingPct, 10, 30)
	tc.Fprintf(w, "  Temporal coupling:   %.1f%%\n", m.TemporalCouplingPct)

	af := colorForFloat(m.AuthorFragmentation, 2, 4)
	af.Fprintf(w, "  Author fragmentation: %.2f avg authors/file\n", m.AuthorFragmentation)

	cs := colorForFloatInverse(m.CommitStability, 3, 7)
	cs.Fprintf(w, "  Commit stability:    %.1f days median\n", m.CommitStability)

	hc := colorForFloat(m.HotspotConcentration, 50, 75)
	hc.Fprintf(w, "  Hotspot concentration: %.1f%%\n", m.HotspotConcentration)

	// Verbose: show top hotspots and coupled pairs
	if verbose && len(m.TopHotspots) > 0 {
		fmt.Fprintln(w)
		bold.Fprintln(w, "  Top hotspots:")
		for _, h := range m.TopHotspots {
			fmt.Fprintf(w, "    %s  changes=%d commits=%d authors=%d\n", h.Path, h.TotalChanges, h.CommitCount, h.AuthorCount)
		}
	}
	if verbose && len(m.CoupledPairs) > 0 {
		fmt.Fprintln(w)
		bold.Fprintln(w, "  Coupled pairs (>70%% co-change):")
		limit := 10
		if len(m.CoupledPairs) < limit {
			limit = len(m.CoupledPairs)
		}
		for _, cp := range m.CoupledPairs[:limit] {
			fmt.Fprintf(w, "    %s <-> %s  %.0f%% (%d shared commits)\n", cp.FileA, cp.FileB, cp.Coupling, cp.SharedCommits)
		}
	}
}

func renderC6(w io.Writer, ar *types.AnalysisResult, verbose bool) {
	bold := color.New(color.Bold)

	raw, ok := ar.Metrics["c6"]
	if !ok {
		return
	}
	m, ok := raw.(*types.C6Metrics)
	if !ok {
		return
	}

	fmt.Fprintln(w)
	bold.Fprintln(w, "C6: Testing")
	fmt.Fprintln(w, "────────────────────────────────────────")

	tr := colorForFloatInverse(m.TestToCodeRatio, 0.2, 0.5)
	tr.Fprintf(w, "  Test-to-code ratio:  %.2f\n", m.TestToCodeRatio)

	if m.CoveragePercent >= 0 {
		cov := colorForFloatInverse(m.CoveragePercent, 40, 70)
		cov.Fprintf(w, "  Coverage:            %.1f%% (%s)\n", m.CoveragePercent, m.CoverageSource)
	} else {
		fmt.Fprintf(w, "  Coverage:            n/a (no coverage data found)\n")
	}

	iso := colorForFloatInverse(m.TestIsolation, 50, 80)
	iso.Fprintf(w, "  Test isolation:      %.0f%%\n", m.TestIsolation)

	ad := colorForFloat(m.AssertionDensity.Avg, 999, 999) // no yellow/red thresholds for assertion density
	ad.Fprintf(w, "  Assertion density:   %.1f avg\n", m.AssertionDensity.Avg)

	// Verbose: per-test function details
	if verbose && len(m.TestFunctions) > 0 {
		fmt.Fprintln(w)
		bold.Fprintln(w, "  Test functions:")
		for _, tf := range m.TestFunctions {
			extDep := ""
			if tf.HasExternalDep {
				extDep = " [external-dep]"
			}
			fmt.Fprintf(w, "    %s.%s  assertions=%d%s  (%s:%d)\n", tf.Package, tf.Name, tf.AssertionCount, extDep, tf.File, tf.Line)
		}
	}
}

// categoryDisplayNames maps category identifiers to human-readable labels.
var categoryDisplayNames = map[string]string{
	"C1": "Code Health",
	"C2": "Semantic Explicitness",
	"C3": "Architecture",
	"C5": "Temporal Dynamics",
	"C6": "Testing",
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
}

// RenderScores prints a formatted scoring section showing per-category scores,
// composite score, and tier rating. When verbose is true, per-metric sub-score
// breakdowns are shown beneath each category.
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
	if score >= 8.0 {
		return color.New(color.FgGreen)
	}
	if score >= 6.0 {
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
// Each recommendation shows rank, summary, impact, effort, and action.
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

		impactColor := scoreColor(rec.ScoreImprovement * 20) // map 0.5->10 for green threshold
		if rec.ScoreImprovement >= 0.5 {
			impactColor = color.New(color.FgGreen)
		} else if rec.ScoreImprovement >= 0.2 {
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
	result := ""
	for i, pkg := range cycle {
		if i > 0 {
			result += " -> "
		}
		result += pkg
	}
	result += " -> " + cycle[0]
	return result
}
