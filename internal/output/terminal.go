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
	"sort"
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

// Display thresholds for C7 score coloring (0-100 scale).
const (
	c7ScoreGreenMin  = 70
	c7ScoreYellowMin = 40
	c7ScoreScale     = 10 // Multiplier to convert 1-10 scores to 0-100 for color
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

// C1 metric color thresholds for terminal display (green = good, yellow = caution).
const (
	c1ComplexityAvgGreen  = 10.0
	c1ComplexityAvgYellow = 20.0
	c1ComplexityMaxGreen  = 15
	c1ComplexityMaxYellow = 30
	c1FuncLenAvgGreen     = 30.0
	c1FuncLenAvgYellow    = 60.0
	c1FuncLenMaxGreen     = 50
	c1FuncLenMaxYellow    = 100
	c1FileSizeAvgGreen    = 300.0
	c1FileSizeAvgYellow   = 500.0
	c1FileSizeMaxGreen    = 500
	c1FileSizeMaxYellow   = 1000
	c1DuplicationGreen    = 5.0
	c1DuplicationYellow   = 15.0
)

// C2 metric color thresholds (inverse: higher is better for coverage/consistency).
const (
	c2TypeAnnotationRed    = 50.0
	c2TypeAnnotationYellow = 80.0
	c2NamingRed            = 70.0
	c2NamingYellow         = 90.0
	c2MagicNumGreen        = 5.0
	c2MagicNumYellow       = 15.0
)

// C3 metric color thresholds.
const (
	c3DirDepthGreen     = 4
	c3DirDepthYellow    = 7
	c3FanoutGreen       = 5.0
	c3FanoutYellow      = 10.0
	c3CircularDepsMax   = 2
	c3DeadExportsGreen  = 5
	c3DeadExportsYellow = 20
)

// C4 metric color thresholds (inverse: higher is better).
const (
	c4CommentDensityRed    = 5.0
	c4CommentDensityYellow = 15.0
	c4APIDocRed            = 30.0
	c4APIDocYellow         = 60.0
	c4LLMScoreRed          = 4
	c4LLMScoreYellow       = 7
)

// C5 metric color thresholds.
const (
	c5ChurnGreen            = 100.0
	c5ChurnYellow           = 300.0
	c5TemporalCouplingGreen = 10.0
	c5TemporalCouplingYellow = 30.0
	c5AuthorFragGreen       = 2.0
	c5AuthorFragYellow      = 4.0
	c5CommitStabilityRed    = 3.0
	c5CommitStabilityYellow = 7.0
	c5HotspotGreen          = 50.0
	c5HotspotYellow         = 75.0
)

// C6 metric color thresholds.
const (
	c6TestRatioRed        = 0.2
	c6TestRatioYellow     = 0.5
	c6CoverageRed         = 40.0
	c6CoverageYellow      = 70.0
	c6IsolationRed        = 50.0
	c6IsolationYellow     = 80.0
	c6AssertionNoThreshold = 999.0 // Sentinel: no meaningful yellow/red threshold
)

// Recommendation impact thresholds.
const (
	recImpactScaleToScore = 20.0 // Scales 0.5 point improvement to score 10.0
	recHighImpact         = 0.5  // Score improvement >= this is high impact (green)
	recModerateImpact     = 0.2  // Score improvement >= this is moderate (yellow)
)

// RenderSummary prints a formatted scan summary to w.
//
// Color is automatically disabled when w is not a TTY (e.g., piped output).
// This prevents ANSI escape codes from corrupting piped data while preserving
// the visual hierarchy when output goes to a terminal. The layout uses a fixed
// hierarchy: header → file counts → per-language breakdown → exclusion counts
// → verbose file listing → category-specific metrics (C1-C7).
//
// Each category renderer (renderC1-C7) follows the same pattern: bold headers,
// color-coded metrics using score thresholds, and optional verbose sections
// for detailed breakdowns (top offenders, per-item details, traces).
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
	//
	// File listing shows classification tags (SOURCE, TEST, EXCLUDED) with exclusion
	// reasons when applicable. This helps users audit discovery logic and understand
	// why certain files were skipped. Common exclusion reasons: .gitignore rules,
	// vendor/ directories, generated code markers, or unsupported file extensions.
	//
	// Relative paths are displayed to keep output compact and repository-agnostic
	// (absolute paths would leak local filesystem structure and make diffs noisy).
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
	cc := colorForFloat(m.CyclomaticComplexity.Avg, c1ComplexityAvgGreen, c1ComplexityAvgYellow)
	cc.Fprintf(w, "  Complexity avg:      %.1f\n", m.CyclomaticComplexity.Avg)
	cm := colorForInt(m.CyclomaticComplexity.Max, c1ComplexityMaxGreen, c1ComplexityMaxYellow)
	cm.Fprintf(w, "  Complexity max:      %d", m.CyclomaticComplexity.Max)
	if m.CyclomaticComplexity.MaxEntity != "" {
		fmt.Fprintf(w, " (%s)", m.CyclomaticComplexity.MaxEntity)
	}
	fmt.Fprintln(w)

	// Function length
	fl := colorForFloat(m.FunctionLength.Avg, c1FuncLenAvgGreen, c1FuncLenAvgYellow)
	fl.Fprintf(w, "  Func length avg:     %.1f lines\n", m.FunctionLength.Avg)
	flm := colorForInt(m.FunctionLength.Max, c1FuncLenMaxGreen, c1FuncLenMaxYellow)
	flm.Fprintf(w, "  Func length max:     %d lines", m.FunctionLength.Max)
	if m.FunctionLength.MaxEntity != "" {
		fmt.Fprintf(w, " (%s)", m.FunctionLength.MaxEntity)
	}
	fmt.Fprintln(w)

	// File size
	fs := colorForFloat(m.FileSize.Avg, c1FileSizeAvgGreen, c1FileSizeAvgYellow)
	fs.Fprintf(w, "  File size avg:       %.0f lines\n", m.FileSize.Avg)
	fsm := colorForInt(m.FileSize.Max, c1FileSizeMaxGreen, c1FileSizeMaxYellow)
	fsm.Fprintf(w, "  File size max:       %d lines", m.FileSize.Max)
	if m.FileSize.MaxEntity != "" {
		fmt.Fprintf(w, " (%s)", m.FileSize.MaxEntity)
	}
	fmt.Fprintln(w)

	// Duplication
	dc := colorForFloat(m.DuplicationRate, c1DuplicationGreen, c1DuplicationYellow)
	dc.Fprintf(w, "  Duplication rate:    %.1f%%\n", m.DuplicationRate)

	// Verbose: top 5 most complex and longest functions
	//
	// "Top offenders" lists help developers prioritize refactoring work by identifying
	// the specific functions dragging down scores. Without this drill-down, users only
	// see aggregate metrics (e.g., "avg complexity: 12") but don't know which functions
	// to target. File:line references allow quick navigation via IDE jump-to-line features.
	//
	// Why limit to 5? Beyond 5 items, lists become overwhelming and users stop reading.
	// The Pareto principle suggests 20% of functions cause 80% of problems - top 5
	// captures the highest-leverage improvements. Users can always rerun with full
	// evidence details in HTML reports if they need comprehensive listings.
	if verbose && len(m.Functions) > 0 {
		fmt.Fprintln(w)
		bold.Fprintln(w, "  Top complex functions:")
		byComplexity := make([]types.FunctionMetric, len(m.Functions))
		copy(byComplexity, m.Functions)
		sort.Slice(byComplexity, func(i, j int) bool {
			return byComplexity[i].Complexity > byComplexity[j].Complexity
		})
		limit := verboseTopN
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
		limit = verboseTopN
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

	tc := colorForFloatInverse(agg.TypeAnnotationCoverage, c2TypeAnnotationRed, c2TypeAnnotationYellow)
	tc.Fprintf(w, "  Type annotation:     %.1f%%\n", agg.TypeAnnotationCoverage)

	nc := colorForFloatInverse(agg.NamingConsistency, c2NamingRed, c2NamingYellow)
	nc.Fprintf(w, "  Naming consistency:  %.1f%%\n", agg.NamingConsistency)

	mr := colorForFloat(agg.MagicNumberRatio, c2MagicNumGreen, c2MagicNumYellow)
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

	dd := colorForInt(m.MaxDirectoryDepth, c3DirDepthGreen, c3DirDepthYellow)
	dd.Fprintf(w, "  Max directory depth: %d\n", m.MaxDirectoryDepth)
	fmt.Fprintf(w, "  Avg directory depth: %.1f\n", m.AvgDirectoryDepth)

	fo := colorForFloat(m.ModuleFanout.Avg, c3FanoutGreen, c3FanoutYellow)
	fo.Fprintf(w, "  Avg module fanout:   %.1f\n", m.ModuleFanout.Avg)

	circCount := len(m.CircularDeps)
	cc := colorForInt(circCount, 0, c3CircularDepsMax)
	cc.Fprintf(w, "  Circular deps:       %d\n", circCount)

	deadCount := len(m.DeadExports)
	dc := colorForInt(deadCount, c3DeadExportsGreen, c3DeadExportsYellow)
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

	cr := colorForFloat(m.ChurnRate, c5ChurnGreen, c5ChurnYellow)
	cr.Fprintf(w, "  Churn rate:          %.1f lines/commit\n", m.ChurnRate)

	tc := colorForFloat(m.TemporalCouplingPct, c5TemporalCouplingGreen, c5TemporalCouplingYellow)
	tc.Fprintf(w, "  Temporal coupling:   %.1f%%\n", m.TemporalCouplingPct)

	af := colorForFloat(m.AuthorFragmentation, c5AuthorFragGreen, c5AuthorFragYellow)
	af.Fprintf(w, "  Author fragmentation: %.2f avg authors/file\n", m.AuthorFragmentation)

	cs := colorForFloatInverse(m.CommitStability, c5CommitStabilityRed, c5CommitStabilityYellow)
	cs.Fprintf(w, "  Commit stability:    %.1f days median\n", m.CommitStability)

	hc := colorForFloat(m.HotspotConcentration, c5HotspotGreen, c5HotspotYellow)
	hc.Fprintf(w, "  Hotspot concentration: %.1f%%\n", m.HotspotConcentration)

	// Verbose: show top hotspots and coupled pairs
	//
	// Hotspots reveal files under high change pressure - prime refactoring candidates.
	// High commit counts + high author counts = knowledge fragmentation (no single owner).
	// Agents struggle with hotspots because the code's behavior is a moving target,
	// increasing the likelihood of merge conflicts and stale context during modifications.
	//
	// Temporally coupled pairs (files that change together >70% of the time) indicate
	// hidden dependencies that aren't visible in static import graphs. For example,
	// a config file and its consumer may be coupled despite no direct import. Agents
	// miss these implicit dependencies, leading to incomplete changes (update config
	// but not consumer). Limit: 10 pairs to keep output manageable while surfacing
	// the strongest coupling relationships.
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
		limit := coupledPairsTopN
		if len(m.CoupledPairs) < limit {
			limit = len(m.CoupledPairs)
		}
		for _, cp := range m.CoupledPairs[:limit] {
			fmt.Fprintf(w, "    %s <-> %s  %.0f%% (%d shared commits)\n", cp.FileA, cp.FileB, cp.Coupling, cp.SharedCommits)
		}
	}
}

func renderC4(w io.Writer, ar *types.AnalysisResult, verbose bool) {
	bold := color.New(color.Bold)
	green := color.New(color.FgGreen)
	red := color.New(color.FgRed)

	raw, ok := ar.Metrics["c4"]
	if !ok {
		return
	}
	m, ok := raw.(*types.C4Metrics)
	if !ok {
		return
	}

	fmt.Fprintln(w)
	bold.Fprintln(w, "C4: Documentation Quality")
	fmt.Fprintln(w, "────────────────────────────────────────")

	if !m.Available {
		fmt.Fprintln(w, "  Not available")
		return
	}

	// README
	if m.ReadmePresent {
		green.Fprintf(w, "  README:              present (%d words)\n", m.ReadmeWordCount)
	} else {
		red.Fprintln(w, "  README:              absent")
	}

	// Comment density
	cd := colorForFloatInverse(m.CommentDensity, c4CommentDensityRed, c4CommentDensityYellow)
	cd.Fprintf(w, "  Comment density:     %.1f%%\n", m.CommentDensity)

	// API doc coverage
	ad := colorForFloatInverse(m.APIDocCoverage, c4APIDocRed, c4APIDocYellow)
	ad.Fprintf(w, "  API doc coverage:    %.1f%%\n", m.APIDocCoverage)

	// CHANGELOG
	if m.ChangelogPresent {
		green.Fprintln(w, "  CHANGELOG:           present")
	} else {
		red.Fprintln(w, "  CHANGELOG:           absent")
	}

	// Examples
	if m.ExamplesPresent {
		green.Fprintln(w, "  Examples:            present")
	} else {
		red.Fprintln(w, "  Examples:            absent")
	}

	// CONTRIBUTING
	if m.ContributingPresent {
		green.Fprintln(w, "  CONTRIBUTING:        present")
	} else {
		red.Fprintln(w, "  CONTRIBUTING:        absent")
	}

	// Diagrams
	if m.DiagramsPresent {
		green.Fprintln(w, "  Diagrams:            present")
	} else {
		color.New(color.FgYellow).Fprintln(w, "  Diagrams:            absent")
	}

	// LLM-based metrics (if enabled)
	fmt.Fprintln(w)
	bold.Fprintln(w, "  LLM Analysis:")
	if m.LLMEnabled {
		rc := colorForIntInverse(m.ReadmeClarity, c4LLMScoreRed, c4LLMScoreYellow)
		rc.Fprintf(w, "    README clarity:      %d/10\n", m.ReadmeClarity)
		eq := colorForIntInverse(m.ExampleQuality, c4LLMScoreRed, c4LLMScoreYellow)
		eq.Fprintf(w, "    Example quality:     %d/10\n", m.ExampleQuality)
		cp := colorForIntInverse(m.Completeness, c4LLMScoreRed, c4LLMScoreYellow)
		cp.Fprintf(w, "    Completeness:        %d/10\n", m.Completeness)
		cr := colorForIntInverse(m.CrossRefCoherence, c4LLMScoreRed, c4LLMScoreYellow)
		cr.Fprintf(w, "    Cross-ref coherence: %d/10\n", m.CrossRefCoherence)
		fmt.Fprintf(w, "    LLM cost:            $%.4f (%d tokens)\n", m.LLMCostUSD, m.LLMTokensUsed)
	} else {
		color.New(color.FgHiBlack).Fprintln(w, "    README clarity:      n/a (Claude CLI not detected)")
		color.New(color.FgHiBlack).Fprintln(w, "    Example quality:     n/a")
		color.New(color.FgHiBlack).Fprintln(w, "    Completeness:        n/a")
		color.New(color.FgHiBlack).Fprintln(w, "    Cross-ref coherence: n/a")
	}

	// Verbose: show counts
	//
	// These raw counts help developers understand the denominators behind
	// percentages (e.g., "15% comment density" becomes concrete: "469 comment
	// lines out of 3,127 total"). Concrete numbers make improvement targets
	// actionable ("add 78 comment lines" vs "increase density 2.5%").
	if verbose {
		fmt.Fprintln(w)
		bold.Fprintln(w, "  Detailed metrics:")
		fmt.Fprintf(w, "    Total source lines:  %d\n", m.TotalSourceLines)
		fmt.Fprintf(w, "    Comment lines:       %d\n", m.CommentLines)
		fmt.Fprintf(w, "    Public APIs:         %d\n", m.PublicAPIs)
		fmt.Fprintf(w, "    Documented APIs:     %d\n", m.DocumentedAPIs)
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

	tr := colorForFloatInverse(m.TestToCodeRatio, c6TestRatioRed, c6TestRatioYellow)
	tr.Fprintf(w, "  Test-to-code ratio:  %.2f\n", m.TestToCodeRatio)

	if m.CoveragePercent >= 0 {
		cov := colorForFloatInverse(m.CoveragePercent, c6CoverageRed, c6CoverageYellow)
		cov.Fprintf(w, "  Coverage:            %.1f%% (%s)\n", m.CoveragePercent, m.CoverageSource)
	} else {
		fmt.Fprintf(w, "  Coverage:            n/a (no coverage data found)\n")
	}

	iso := colorForFloatInverse(m.TestIsolation, c6IsolationRed, c6IsolationYellow)
	iso.Fprintf(w, "  Test isolation:      %.0f%%\n", m.TestIsolation)

	ad := colorForFloat(m.AssertionDensity.Avg, c6AssertionNoThreshold, c6AssertionNoThreshold)
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

// c7ScoreColor returns a color based on C7 score (0-100, higher is better).
func c7ScoreColor(score int) *color.Color {
	if score >= c7ScoreGreenMin {
		return color.New(color.FgGreen)
	}
	if score >= c7ScoreYellowMin {
		return color.New(color.FgYellow)
	}
	return color.New(color.FgRed)
}

func renderC7(w io.Writer, ar *types.AnalysisResult, verbose bool) {
	bold := color.New(color.Bold)

	raw, ok := ar.Metrics["c7"]
	if !ok {
		return
	}
	m, ok := raw.(*types.C7Metrics)
	if !ok {
		return
	}

	fmt.Fprintln(w)
	bold.Fprintln(w, "C7: Agent Evaluation")
	fmt.Fprintln(w, "────────────────────────────────────────")

	if !m.Available {
		fmt.Fprintln(w, "  Not available (LLM features disabled)")
		return
	}

	// New MECE metrics (1-10 scale)
	if m.TaskExecutionConsistency > 0 || m.CodeBehaviorComprehension > 0 ||
		m.CrossFileNavigation > 0 || m.IdentifierInterpretability > 0 ||
		m.DocumentationAccuracyDetection > 0 {
		// Show new MECE metrics
		m1c := c7ScoreColor(m.TaskExecutionConsistency * c7ScoreScale)
		m1c.Fprintf(w, "  M1 Exec Consistency:  %d/10\n", m.TaskExecutionConsistency)

		m2c := c7ScoreColor(m.CodeBehaviorComprehension * c7ScoreScale)
		m2c.Fprintf(w, "  M2 Comprehension:     %d/10\n", m.CodeBehaviorComprehension)

		m3c := c7ScoreColor(m.CrossFileNavigation * c7ScoreScale)
		m3c.Fprintf(w, "  M3 Navigation:        %d/10\n", m.CrossFileNavigation)

		m4c := c7ScoreColor(m.IdentifierInterpretability * c7ScoreScale)
		m4c.Fprintf(w, "  M4 Identifiers:       %d/10\n", m.IdentifierInterpretability)

		m5c := c7ScoreColor(m.DocumentationAccuracyDetection * c7ScoreScale)
		m5c.Fprintf(w, "  M5 Documentation:     %d/10\n", m.DocumentationAccuracyDetection)
	} else {
		// Fallback to legacy metrics for backward compatibility
		ic := c7ScoreColor(m.IntentClarity)
		ic.Fprintf(w, "  Intent clarity:       %d/100\n", m.IntentClarity)

		mc := c7ScoreColor(m.ModificationConfidence)
		mc.Fprintf(w, "  Modification conf:    %d/100\n", m.ModificationConfidence)

		cfc := c7ScoreColor(m.CrossFileCoherence)
		cfc.Fprintf(w, "  Cross-file coherence: %d/100\n", m.CrossFileCoherence)

		sc := c7ScoreColor(m.SemanticCompleteness)
		sc.Fprintf(w, "  Semantic complete:    %d/100\n", m.SemanticCompleteness)
	}

	// Summary metrics
	fmt.Fprintln(w, "  ─────────────────────────────────────")
	if m.MECEScore > 0 {
		// Show MECE score (weighted average of 5 metrics, 1-10 scale)
		os := c7ScoreColor(int(m.MECEScore * c7ScoreScale))
		os.Fprintf(w, "  MECE Score:           %.1f/10\n", m.MECEScore)
	} else {
		// Show legacy overall score (0-100 scale)
		os := c7ScoreColor(int(m.OverallScore))
		os.Fprintf(w, "  Overall score:        %.1f/100\n", m.OverallScore)
	}
	fmt.Fprintf(w, "  Duration:             %.1fs\n", m.TotalDuration)
	fmt.Fprintf(w, "  Estimated cost:       $%.4f\n", m.CostUSD)

	// Verbose: per-task breakdown
	if verbose && len(m.TaskResults) > 0 {
		fmt.Fprintln(w)
		bold.Fprintln(w, "  Per-task results:")
		for _, tr := range m.TaskResults {
			fmt.Fprintf(w, "    %s: score=%d status=%s (%.1fs)\n", tr.TaskName, tr.Score, tr.Status, tr.Duration)
			if tr.Reasoning != "" {
				fmt.Fprintf(w, "      Reasoning: %s\n", tr.Reasoning)
			}
		}
	}
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
// composite score, and tier rating. When verbose is true, per-metric sub-score
// breakdowns are shown beneath each category.
//
// Score rendering uses color-coded thresholds aligned with tier boundaries:
// - Green (≥8.0): Agent-Ready tier, agents can work autonomously
// - Yellow (≥6.0): Agent-Assisted tier, agents need human guidance
// - Red (<6.0): Agent-Limited or Agent-Hostile, agents struggle significantly
//
// This color mapping helps users quickly identify categories dragging down the
// composite score and prioritize improvements for tier advancement.
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
//
// Evidence-based display format: raw value → interpolated score → weight percentage.
// This shows users exactly how each metric contributes to the category score and
// reveals which metrics have the most improvement leverage (high weight + low score).
//
// Zero-weight metrics are filtered out to avoid confusing users with deprecated or
// inactive metrics. For example, C7's overall_score (0-100 scale) was replaced by
// MECE-based scoring but kept for backward compatibility with weight=0.
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
//
// Threshold rationale:
// - 8.0 (green): Agent-Ready tier boundary - autonomous agent operation possible
// - 6.0 (yellow): Agent-Assisted tier boundary - agents need human oversight
// - <6.0 (red): Agent-Limited/Hostile - agents struggle or fail frequently
//
// These thresholds were empirically derived from agent success rates across codebases
// and align with the project's research on agent-readiness factors (see agent.md).
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
//
// Bold formatting emphasizes tier classification since it's the primary outcome
// users care about (single summary of overall agent-readiness). Color semantics:
// green = positive (ready), yellow = caution (needs help), red = negative (hostile).
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
//
// Each recommendation shows rank, summary, impact, effort, and action.
// Recommendations are pre-sorted by ROI (impact/effort ratio) in the recommend
// package, so displaying them in order guides users toward the most efficient
// improvements. High-impact, low-effort changes appear first (quick wins).
//
// Impact visualization uses color thresholds: ≥0.5 points = green (significant),
// ≥0.2 points = yellow (moderate), <0.2 = red (marginal). This helps users
// gauge whether a recommendation is worth pursuing given their tier goals.
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

// RenderBadge prints the shields.io badge markdown to w.
func RenderBadge(w io.Writer, scored *types.ScoredResult) {
	if scored == nil {
		return
	}

	bold := color.New(color.Bold)
	badge := generateBadge(scored)

	fmt.Fprintln(w)
	bold.Fprintln(w, "Badge")
	fmt.Fprintln(w, "\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500")
	fmt.Fprintln(w, badge.Markdown)
}

// RenderC7Debug renders detailed C7 debug data (prompts, responses, scores, traces)
// to the provided writer. This is called only when --debug-c7 is active. The writer
// is typically os.Stderr so debug output never mixes with normal stdout output.
//
// Debug rendering helps developers:
// 1. Understand why C7 scores are low (inspect actual agent responses)
// 2. Validate C7 scoring heuristics (see score trace breakdowns)
// 3. Debug Claude CLI integration issues (capture raw prompts/responses)
// 4. Replay saved responses without hitting the Claude API (--debug-dir)
//
// Output goes to stderr by convention to keep stdout clean for JSON pipelines
// and avoid contaminating production output with debug data.
func RenderC7Debug(w io.Writer, analysisResults []*types.AnalysisResult) {
	// Find the C7 result
	var c7Result *types.AnalysisResult
	for _, ar := range analysisResults {
		if ar.Category == "C7" {
			c7Result = ar
			break
		}
	}
	if c7Result == nil {
		return
	}

	raw, ok := c7Result.Metrics["c7"]
	if !ok {
		return
	}
	m, ok := raw.(*types.C7Metrics)
	if !ok || !m.Available {
		return
	}
	if len(m.MetricResults) == 0 {
		return
	}

	bold := color.New(color.Bold)
	dim := color.New(color.FgHiBlack)
	red := color.New(color.FgRed)

	// Header
	fmt.Fprintln(w)
	bold.Fprintln(w, "C7 Debug: Agent Evaluation Details")
	fmt.Fprintln(w, strings.Repeat("=", separatorWide))

	for _, mr := range m.MetricResults {
		fmt.Fprintln(w)
		bold.Fprintf(w, "[%s] %s  score=%d/10  (%.1fs)\n", mr.MetricID, mr.MetricName, mr.Score, mr.Duration)
		fmt.Fprintln(w, strings.Repeat("-", separatorNarrow))

		if len(mr.DebugSamples) == 0 {
			dim.Fprintln(w, "  No debug samples captured")
			continue
		}

		for i, ds := range mr.DebugSamples {
			fmt.Fprintf(w, "  Sample %d: %s\n", i+1, ds.Description)
			fmt.Fprintf(w, "  File:     %s\n", ds.FilePath)
			fmt.Fprintf(w, "  Score:    %d/10  Duration: %.1fs\n", ds.Score, ds.Duration)

			// Prompt (truncated, dim)
			prompt := truncateString(ds.Prompt, truncateShort)
			dim.Fprintf(w, "  Prompt:   %s\n", prompt)

			// Response (truncated)
			response := truncateString(ds.Response, truncateLong)
			fmt.Fprintf(w, "  Response: %s\n", response)

			// Score trace
			renderScoreTrace(w, ds.ScoreTrace)

			// Error (red, if present)
			if ds.Error != "" {
				red.Fprintf(w, "  Error: %s\n", ds.Error)
			}

			// Blank line between samples (but not after the last)
			if i < len(mr.DebugSamples)-1 {
				fmt.Fprintln(w)
			}
		}
	}
}

// renderScoreTrace prints the score trace breakdown for a single debug sample.
func renderScoreTrace(w io.Writer, trace types.C7ScoreTrace) {
	var parts []string
	for _, ind := range trace.Indicators {
		if ind.Matched {
			sign := "+"
			if ind.Delta < 0 {
				sign = ""
			}
			parts = append(parts, fmt.Sprintf("%s(%s%d)", ind.Name, sign, ind.Delta))
		}
	}
	indicators := strings.Join(parts, " ")
	if indicators != "" {
		indicators = " " + indicators + " "
	} else {
		indicators = " "
	}
	fmt.Fprintf(w, "  Trace:    base=%d%s-> final=%d\n", trace.BaseScore, indicators, trace.FinalScore)
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
