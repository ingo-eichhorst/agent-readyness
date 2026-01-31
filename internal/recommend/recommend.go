package recommend

import (
	"github.com/ingo/agent-readyness/internal/scoring"
	"github.com/ingo/agent-readyness/pkg/types"
)

// Recommendation represents a single improvement recommendation.
type Recommendation struct {
	Rank             int     // 1-based rank
	Category         string  // e.g., "C1"
	MetricName       string  // e.g., "complexity_avg"
	CurrentValue     float64 // raw metric value
	CurrentScore     float64 // 1-10 metric score
	TargetValue      float64 // next breakpoint value that improves score
	TargetScore      float64 // what metric score would be at target
	ScoreImprovement float64 // estimated composite improvement
	Effort           string  // "Low", "Medium", "High"
	Summary          string  // agent-readiness framed description
	Action           string  // concrete improvement action
}

// Generate analyzes scored results and returns up to 5 improvement
// recommendations ranked by composite score impact.
func Generate(scored *types.ScoredResult, cfg *scoring.ScoringConfig) []Recommendation {
	return nil
}
