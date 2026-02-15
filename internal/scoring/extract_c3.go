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

	if m.ModuleFanout.MaxEntity != "" {
		evidence["module_fanout_avg"] = []types.EvidenceItem{{
			FilePath: m.ModuleFanout.MaxEntity, Value: float64(m.ModuleFanout.Max),
			Description: fmt.Sprintf("highest fanout: %d references", m.ModuleFanout.Max),
		}}
	}

	if len(m.CircularDeps) > 0 {
		evidence["circular_deps"] = topNEvidence(len(m.CircularDeps), func(i int) types.EvidenceItem {
			cycle := m.CircularDeps[i]
			filePath := ""
			if len(cycle) > 0 {
				filePath = cycle[0]
			}
			return types.EvidenceItem{
				FilePath: filePath, Value: float64(len(cycle)),
				Description: fmt.Sprintf("cycle: %s", strings.Join(cycle, " -> ")),
			}
		})
	}

	if m.ImportComplexity.MaxEntity != "" {
		evidence["import_complexity_avg"] = []types.EvidenceItem{{
			FilePath: m.ImportComplexity.MaxEntity, Value: float64(m.ImportComplexity.Max),
			Description: fmt.Sprintf("most complex imports: %d segments", m.ImportComplexity.Max),
		}}
	}

	if len(m.DeadExports) > 0 {
		evidence["dead_exports"] = topNEvidence(len(m.DeadExports), func(i int) types.EvidenceItem {
			de := m.DeadExports[i]
			return types.EvidenceItem{
				FilePath: de.File, Line: de.Line, Value: 1,
				Description: fmt.Sprintf("unused %s: %s", de.Kind, de.Name),
			}
		})
	}

	ensureEvidenceKeys(evidence, []string{
		"max_dir_depth", "module_fanout_avg", "circular_deps",
		"import_complexity_avg", "dead_exports",
	})

	return map[string]float64{
		"max_dir_depth":        float64(m.MaxDirectoryDepth),
		"module_fanout_avg":    m.ModuleFanout.Avg,
		"circular_deps":        float64(len(m.CircularDeps)),
		"import_complexity_avg": m.ImportComplexity.Avg,
		"dead_exports":          float64(len(m.DeadExports)),
	}, nil, evidence
}
