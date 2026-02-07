package c3

import (
	"github.com/ingo/agent-readyness/internal/analyzer/shared"
	"github.com/ingo/agent-readyness/internal/parser"
	arstypes "github.com/ingo/agent-readyness/pkg/types"
)

// languageAnalysis holds the results of language-specific C3 analysis.
type languageAnalysis struct {
	maxDepth      int
	avgDepth      float64
	fanout        arstypes.MetricSummary
	circularDeps  [][]string
	deadExports   []arstypes.DeadExport
}

// analyzeLanguageTarget performs C3 analysis for a Python or TypeScript target.
// It returns analyzed metrics that can be merged into the overall C3Metrics.
func (a *C3Analyzer) analyzeLanguageTarget(
	target *arstypes.AnalysisTarget,
	filterSourceFiles func([]*parser.ParsedTreeSitterFile) []*parser.ParsedTreeSitterFile,
	buildImportGraph func([]*parser.ParsedTreeSitterFile) *shared.ImportGraph,
	detectDeadCode func([]*parser.ParsedTreeSitterFile) []arstypes.DeadExport,
	analyzeDirectoryDepth func([]*parser.ParsedTreeSitterFile, string) (int, float64),
) (*languageAnalysis, error) {
	if a.tsParser == nil {
		return nil, nil
	}

	parsed, err := a.tsParser.ParseTargetFiles(target)
	if err != nil {
		return nil, err
	}
	defer parser.CloseAll(parsed)

	srcFiles := filterSourceFiles(parsed)
	graph := buildImportGraph(parsed)
	deadExports := detectDeadCode(parsed)
	maxDepth, avgDepth := analyzeDirectoryDepth(parsed, target.RootDir)

	// Calculate circular dependencies
	cycles := detectCircularDeps(graph)

	// Calculate module fanout
	var fanout arstypes.MetricSummary
	if len(srcFiles) > 0 && len(graph.Forward) > 0 {
		totalFanout := 0
		maxFanout := 0
		maxEntity := ""
		for file, deps := range graph.Forward {
			f := len(deps)
			totalFanout += f
			if f > maxFanout {
				maxFanout = f
				maxEntity = file
			}
		}
		fanout = arstypes.MetricSummary{
			Avg:       float64(totalFanout) / float64(len(graph.Forward)),
			Max:       maxFanout,
			MaxEntity: maxEntity,
		}
	}

	return &languageAnalysis{
		maxDepth:     maxDepth,
		avgDepth:     avgDepth,
		fanout:       fanout,
		circularDeps: cycles,
		deadExports:  deadExports,
	}, nil
}

// mergeLanguageAnalysis merges language-specific analysis into the main metrics.
func mergeLanguageAnalysis(metrics *arstypes.C3Metrics, analysis *languageAnalysis) {
	if analysis == nil {
		return
	}

	// Merge directory depth (take maximum)
	if analysis.maxDepth > metrics.MaxDirectoryDepth {
		metrics.MaxDirectoryDepth = analysis.maxDepth
	}
	if analysis.avgDepth > metrics.AvgDirectoryDepth {
		metrics.AvgDirectoryDepth = analysis.avgDepth
	}

	// Merge circular dependencies
	metrics.CircularDeps = append(metrics.CircularDeps, analysis.circularDeps...)

	// Merge module fanout (take higher max)
	if analysis.fanout.Max > metrics.ModuleFanout.Max {
		metrics.ModuleFanout = analysis.fanout
	}

	// Merge dead exports
	metrics.DeadExports = append(metrics.DeadExports, analysis.deadExports...)
}
