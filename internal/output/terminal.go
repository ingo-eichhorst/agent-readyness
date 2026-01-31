package output

import (
	"fmt"
	"io"
	"sort"

	"github.com/fatih/color"

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
	fmt.Fprintf(w, "Go files discovered: %d\n", result.TotalFiles)

	// Source and test counts (always shown)
	green.Fprintf(w, "  Source files:        %d\n", result.SourceCount)
	yellow.Fprintf(w, "  Test files:          %d\n", result.TestCount)

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
		case "C3":
			renderC3(w, ar, verbose)
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
