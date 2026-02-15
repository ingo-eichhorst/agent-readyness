package scoring

import (
	"fmt"
	"strings"

	"github.com/ingo-eichhorst/agent-readyness/pkg/types"
)

// extractC3 extracts C3 (Architecture) metrics from an AnalysisResult and collects evidence.
func extractC3(ar *types.AnalysisResult) (map[string]float64, map[string]bool, map[string][]types.EvidenceItem) {
	raw, ok := ar.Metrics["c3"]
	if !ok {
		return nil, nil, nil
	}
	m, ok := raw.(*types.C3Metrics)
	if !ok {
		return nil, nil, nil
	}

	evidence := make(map[string][]types.EvidenceItem)

	// module_fanout_avg: single worst module if available
	if m.ModuleFanout.MaxEntity != "" {
		evidence["module_fanout_avg"] = []types.EvidenceItem{{
			FilePath:    m.ModuleFanout.MaxEntity,
			Line:        0,
			Value:       float64(m.ModuleFanout.Max),
			Description: fmt.Sprintf("highest fanout: %d references", m.ModuleFanout.Max),
		}}
	}

	// circular_deps: first 5 cycles
	if len(m.CircularDeps) > 0 {
		limit := evidenceTopN
		if len(m.CircularDeps) < limit {
			limit = len(m.CircularDeps)
		}
		items := make([]types.EvidenceItem, limit)
		for i := 0; i < limit; i++ {
			cycle := m.CircularDeps[i]
			filePath := ""
			if len(cycle) > 0 {
				filePath = cycle[0]
			}
			items[i] = types.EvidenceItem{
				FilePath:    filePath,
				Line:        0,
				Value:       float64(len(cycle)),
				Description: fmt.Sprintf("cycle: %s", strings.Join(cycle, " -> ")),
			}
		}
		evidence["circular_deps"] = items
	}

	// import_complexity_avg: single worst if available
	if m.ImportComplexity.MaxEntity != "" {
		evidence["import_complexity_avg"] = []types.EvidenceItem{{
			FilePath:    m.ImportComplexity.MaxEntity,
			Line:        0,
			Value:       float64(m.ImportComplexity.Max),
			Description: fmt.Sprintf("most complex imports: %d segments", m.ImportComplexity.Max),
		}}
	}

	// dead_exports: first 5 unused exports
	if len(m.DeadExports) > 0 {
		limit := evidenceTopN
		if len(m.DeadExports) < limit {
			limit = len(m.DeadExports)
		}
		items := make([]types.EvidenceItem, limit)
		for i := 0; i < limit; i++ {
			de := m.DeadExports[i]
			items[i] = types.EvidenceItem{
				FilePath:    de.File,
				Line:        de.Line,
				Value:       1,
				Description: fmt.Sprintf("unused %s: %s", de.Kind, de.Name),
			}
		}
		evidence["dead_exports"] = items
	}

	// Ensure all 5 keys have at least empty arrays
	for _, key := range []string{"max_dir_depth", "module_fanout_avg", "circular_deps", "import_complexity_avg", "dead_exports"} {
		if evidence[key] == nil {
			evidence[key] = []types.EvidenceItem{}
		}
	}

	return map[string]float64{
		"max_dir_depth":        float64(m.MaxDirectoryDepth),
		"module_fanout_avg":    m.ModuleFanout.Avg,
		"circular_deps":        float64(len(m.CircularDeps)),
		"import_complexity_avg": m.ImportComplexity.Avg,
		"dead_exports":          float64(len(m.DeadExports)),
	}, nil, evidence
}
