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
	Categories      []JSONCategory       `json:"categories"`
	Recommendations []JSONRecommendation `json:"recommendations"`
	BadgeURL        string               `json:"badge_url,omitempty"`
	BadgeMarkdown   string               `json:"badge_markdown,omitempty"`
}

// JSONCategory represents a scoring category in JSON output.
type JSONCategory struct {
	Name    string       `json:"name"`
	Score   float64      `json:"score"`
	Weight  float64      `json:"weight"`
	Metrics []JSONMetric `json:"metrics,omitempty"`
}

// JSONMetric represents a single metric within a category in JSON output.
type JSONMetric struct {
	Name      string  `json:"name"`
	RawValue  float64 `json:"raw_value"`
	Score     float64 `json:"score"`
	Weight    float64 `json:"weight"`
	Available bool    `json:"available"`
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
// When verbose is true, per-metric sub-scores are included in each category.
// When includeBadge is true, badge URL and markdown are included.
func BuildJSONReport(scored *types.ScoredResult, recs []recommend.Recommendation, verbose bool, includeBadge bool) *JSONReport {
	report := &JSONReport{
		Version:        "1",
		CompositeScore: scored.Composite,
		Tier:           scored.Tier,
	}

	for _, cat := range scored.Categories {
		jc := JSONCategory{
			Name:   cat.Name,
			Score:  cat.Score,
			Weight: cat.Weight,
		}

		if verbose {
			for _, ss := range cat.SubScores {
				jc.Metrics = append(jc.Metrics, JSONMetric{
					Name:      ss.MetricName,
					RawValue:  ss.RawValue,
					Score:     ss.Score,
					Weight:    ss.Weight,
					Available: ss.Available,
				})
			}
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
