package output

import (
	"fmt"
	"io"
	"sort"

	"github.com/fatih/color"

	"github.com/ingo-eichhorst/agent-readyness/pkg/types"
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

func renderC1(w io.Writer, ar *types.AnalysisResult, verbose bool) {
	raw, ok := ar.Metrics["c1"]
	if !ok {
		return
	}
	m, ok := raw.(*types.C1Metrics)
	if !ok {
		return
	}

	bold := color.New(color.Bold)
	fmt.Fprintln(w)
	bold.Fprintln(w, "C1: Code Health")
	fmt.Fprintln(w, "────────────────────────────────────────")

	renderMetricAvgMax(w, "Complexity", m.CyclomaticComplexity, c1ComplexityAvgGreen, c1ComplexityAvgYellow, c1ComplexityMaxGreen, c1ComplexityMaxYellow, "")
	renderMetricAvgMax(w, "Func length", m.FunctionLength, c1FuncLenAvgGreen, c1FuncLenAvgYellow, c1FuncLenMaxGreen, c1FuncLenMaxYellow, " lines")
	renderMetricAvgMax(w, "File size", m.FileSize, c1FileSizeAvgGreen, c1FileSizeAvgYellow, c1FileSizeMaxGreen, c1FileSizeMaxYellow, " lines")

	dc := colorForFloat(m.DuplicationRate, c1DuplicationGreen, c1DuplicationYellow)
	dc.Fprintf(w, "  Duplication rate:    %.1f%%\n", m.DuplicationRate)

	if verbose && len(m.Functions) > 0 {
		renderC1VerboseFunctions(w, m, bold)
	}
}

// renderMetricAvgMax renders avg and max lines for a MetricSummary.
func renderMetricAvgMax(w io.Writer, label string, ms types.MetricSummary, avgGreen, avgYellow float64, maxGreen, maxYellow int, unit string) {
	ac := colorForFloat(ms.Avg, avgGreen, avgYellow)
	ac.Fprintf(w, "  %-22s%.1f%s\n", label+" avg:", ms.Avg, unit)
	mc := colorForInt(ms.Max, maxGreen, maxYellow)
	mc.Fprintf(w, "  %-22s%d%s", label+" max:", ms.Max, unit)
	if ms.MaxEntity != "" {
		fmt.Fprintf(w, " (%s)", ms.MaxEntity)
	}
	fmt.Fprintln(w)
}

// renderC1VerboseFunctions renders top complex and longest functions.
func renderC1VerboseFunctions(w io.Writer, m *types.C1Metrics, bold *color.Color) {
	fmt.Fprintln(w)
	bold.Fprintln(w, "  Top complex functions:")
	printTopFunctions(w, m.Functions, func(a, b types.FunctionMetric) bool {
		return a.Complexity > b.Complexity
	}, func(f types.FunctionMetric) string {
		return fmt.Sprintf("    %s.%s  complexity=%d  (%s:%d)", f.Package, f.Name, f.Complexity, f.File, f.Line)
	})

	fmt.Fprintln(w)
	bold.Fprintln(w, "  Top longest functions:")
	printTopFunctions(w, m.Functions, func(a, b types.FunctionMetric) bool {
		return a.LineCount > b.LineCount
	}, func(f types.FunctionMetric) string {
		return fmt.Sprintf("    %s.%s  lines=%d  (%s:%d)", f.Package, f.Name, f.LineCount, f.File, f.Line)
	})
}

// printTopFunctions sorts functions and prints the top N.
func printTopFunctions(w io.Writer, funcs []types.FunctionMetric, less func(a, b types.FunctionMetric) bool, format func(types.FunctionMetric) string) {
	sorted := make([]types.FunctionMetric, len(funcs))
	copy(sorted, funcs)
	sort.Slice(sorted, func(i, j int) bool { return less(sorted[i], sorted[j]) })
	limit := verboseTopN
	if len(sorted) < limit {
		limit = len(sorted)
	}
	for _, f := range sorted[:limit] {
		fmt.Fprintln(w, format(f))
	}
}
