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
