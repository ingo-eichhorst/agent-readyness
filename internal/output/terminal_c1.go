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

	renderC1Complexity(w, m)
	renderC1FuncLength(w, m)
	renderC1FileSize(w, m)
	renderC1Duplication(w, m)

	if verbose && len(m.Functions) > 0 {
		renderC1VerboseFunctions(w, m, bold)
	}
}

// renderC1Complexity renders cyclomatic complexity avg/max.
func renderC1Complexity(w io.Writer, m *types.C1Metrics) {
	cc := colorForFloat(m.CyclomaticComplexity.Avg, c1ComplexityAvgGreen, c1ComplexityAvgYellow)
	cc.Fprintf(w, "  Complexity avg:      %.1f\n", m.CyclomaticComplexity.Avg)
	cm := colorForInt(m.CyclomaticComplexity.Max, c1ComplexityMaxGreen, c1ComplexityMaxYellow)
	cm.Fprintf(w, "  Complexity max:      %d", m.CyclomaticComplexity.Max)
	if m.CyclomaticComplexity.MaxEntity != "" {
		fmt.Fprintf(w, " (%s)", m.CyclomaticComplexity.MaxEntity)
	}
	fmt.Fprintln(w)
}

// renderC1FuncLength renders function length avg/max.
func renderC1FuncLength(w io.Writer, m *types.C1Metrics) {
	fl := colorForFloat(m.FunctionLength.Avg, c1FuncLenAvgGreen, c1FuncLenAvgYellow)
	fl.Fprintf(w, "  Func length avg:     %.1f lines\n", m.FunctionLength.Avg)
	flm := colorForInt(m.FunctionLength.Max, c1FuncLenMaxGreen, c1FuncLenMaxYellow)
	flm.Fprintf(w, "  Func length max:     %d lines", m.FunctionLength.Max)
	if m.FunctionLength.MaxEntity != "" {
		fmt.Fprintf(w, " (%s)", m.FunctionLength.MaxEntity)
	}
	fmt.Fprintln(w)
}

// renderC1FileSize renders file size avg/max.
func renderC1FileSize(w io.Writer, m *types.C1Metrics) {
	fs := colorForFloat(m.FileSize.Avg, c1FileSizeAvgGreen, c1FileSizeAvgYellow)
	fs.Fprintf(w, "  File size avg:       %.0f lines\n", m.FileSize.Avg)
	fsm := colorForInt(m.FileSize.Max, c1FileSizeMaxGreen, c1FileSizeMaxYellow)
	fsm.Fprintf(w, "  File size max:       %d lines", m.FileSize.Max)
	if m.FileSize.MaxEntity != "" {
		fmt.Fprintf(w, " (%s)", m.FileSize.MaxEntity)
	}
	fmt.Fprintln(w)
}

// renderC1Duplication renders duplication rate.
func renderC1Duplication(w io.Writer, m *types.C1Metrics) {
	dc := colorForFloat(m.DuplicationRate, c1DuplicationGreen, c1DuplicationYellow)
	dc.Fprintf(w, "  Duplication rate:    %.1f%%\n", m.DuplicationRate)
}

// renderC1VerboseFunctions renders top complex and longest functions.
func renderC1VerboseFunctions(w io.Writer, m *types.C1Metrics, bold *color.Color) {
	fmt.Fprintln(w)
	bold.Fprintln(w, "  Top complex functions:")
	renderC1TopFunctions(w, m.Functions, func(a, b types.FunctionMetric) bool {
		return a.Complexity > b.Complexity
	}, func(f types.FunctionMetric) string {
		return fmt.Sprintf("    %s.%s  complexity=%d  (%s:%d)\n", f.Package, f.Name, f.Complexity, f.File, f.Line)
	})

	fmt.Fprintln(w)
	bold.Fprintln(w, "  Top longest functions:")
	renderC1TopFunctions(w, m.Functions, func(a, b types.FunctionMetric) bool {
		return a.LineCount > b.LineCount
	}, func(f types.FunctionMetric) string {
		return fmt.Sprintf("    %s.%s  lines=%d  (%s:%d)\n", f.Package, f.Name, f.LineCount, f.File, f.Line)
	})
}

// renderC1TopFunctions sorts and renders the top N functions by the given comparator.
func renderC1TopFunctions(w io.Writer, funcs []types.FunctionMetric, less func(a, b types.FunctionMetric) bool, format func(types.FunctionMetric) string) {
	sorted := make([]types.FunctionMetric, len(funcs))
	copy(sorted, funcs)
	sort.Slice(sorted, func(i, j int) bool { return less(sorted[i], sorted[j]) })
	limit := verboseTopN
	if len(sorted) < limit {
		limit = len(sorted)
	}
	for _, f := range sorted[:limit] {
		fmt.Fprint(w, format(f))
	}
}
