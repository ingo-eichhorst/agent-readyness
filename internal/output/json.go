package output

import (
	"encoding/json"
	"io"

	"github.com/ingo/agent-readyness/internal/recommend"
	"github.com/ingo/agent-readyness/pkg/types"
)

// JSONReport is the top-level JSON output structure.
type JSONReport struct {
	Version         string               `json:"version"`
	CompositeScore  float64              `json:"composite_score"`
	Tier            string               `json:"tier"`
	Categories      []jsonCategory       `json:"categories"`
	Recommendations []JSONRecommendation `json:"recommendations"`
	BadgeURL        string               `json:"badge_url,omitempty"`
	BadgeMarkdown   string               `json:"badge_markdown,omitempty"`
}

// jsonCategory represents a scoring category in JSON output.
type jsonCategory struct {
	Name      string       `json:"name"`
	Score     float64      `json:"score"`     // -1.0 when unavailable
	Weight    float64      `json:"weight"`
	Available bool         `json:"available"` // whether category is available
	SubScores []jsonMetric `json:"sub_scores"`
}

// jsonMetric represents a single metric within a category in JSON output.
type jsonMetric struct {
	Name      string               `json:"name"`
	RawValue  float64              `json:"raw_value"`
	Score     float64              `json:"score"`
	Weight    float64              `json:"weight"`
	Available bool                 `json:"available"`
	Evidence  []types.EvidenceItem `json:"evidence"`
}

// JSONRecommendation represents a single recommendation in JSON output.
type JSONRecommendation struct {
	Rank             int     `json:"rank"`
	Category         string  `json:"category"`
	MetricName       string  `json:"metric_name"`
	CurrentValue     float64 `json:"current_value"`
	CurrentScore     float64 `json:"current_score"`
	TargetValue      float64 `json:"target_value"`
	ScoreImprovement float64 `json:"score_improvement"`
	Effort           string  `json:"effort"`
	Summary          string  `json:"summary"`
	Action           string  `json:"action"`
}

// BuildJSONReport converts a ScoredResult and recommendations into a JSONReport.
// The verbose parameter is deprecated; sub_scores are always included.
// When includeBadge is true, badge URL and markdown are included.
func BuildJSONReport(scored *types.ScoredResult, recs []recommend.Recommendation, verbose bool, includeBadge bool) *JSONReport {
	report := &JSONReport{
		Version:        "3",
		CompositeScore: scored.Composite,
		Tier:           scored.Tier,
	}

	for _, cat := range scored.Categories {
		jc := jsonCategory{
			Name:      cat.Name,
			Score:     cat.Score,
			Weight:    cat.Weight,
			Available: cat.Score >= 0, // Infer from score
			SubScores: make([]jsonMetric, 0, len(cat.SubScores)),
		}

		for _, ss := range cat.SubScores {
			ev := ss.Evidence
			if ev == nil {
				ev = make([]types.EvidenceItem, 0)
			}
			jc.SubScores = append(jc.SubScores, jsonMetric{
				Name:      ss.MetricName,
				RawValue:  ss.RawValue,
				Score:     ss.Score,
				Weight:    ss.Weight,
				Available: ss.Available,
				Evidence:  ev,
			})
		}

		report.Categories = append(report.Categories, jc)
	}

	for _, rec := range recs {
		report.Recommendations = append(report.Recommendations, JSONRecommendation{
			Rank:             rec.Rank,
			Category:         rec.Category,
			MetricName:       rec.MetricName,
			CurrentValue:     rec.CurrentValue,
			CurrentScore:     rec.CurrentScore,
			TargetValue:      rec.TargetValue,
			ScoreImprovement: rec.ScoreImprovement,
			Effort:           rec.Effort,
			Summary:          rec.Summary,
			Action:           rec.Action,
		})
	}

	// Add badge information if requested
	if includeBadge && scored != nil {
		badge := GenerateBadge(scored)
		report.BadgeURL = badge.URL
		report.BadgeMarkdown = badge.Markdown
	}

	return report
}

// RenderJSON writes the JSON report to w with pretty-printed indentation.
func RenderJSON(w io.Writer, report *JSONReport) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(report)
}
