package output

import (
	"github.com/ingo/agent-readyness/pkg/types"
)

// renderC7Trace renders trace HTML for a C7 metric.
// Returns empty string if no matching metric or no DebugSamples.
func renderC7Trace(metricID string, metricResults []types.C7MetricResult) string {
	// Stub: full implementation in Task 2
	_ = metricID
	_ = metricResults
	return ""
}
