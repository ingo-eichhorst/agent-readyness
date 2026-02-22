package output

import (
	"encoding/json"
	"io"
	"time"

	"github.com/ingo-eichhorst/agent-readyness/internal/recommend"
	"github.com/ingo-eichhorst/agent-readyness/pkg/types"
)

// jsonReport is the top-level JSON output structure.
type jsonReport struct {
	Version         string               `json:"version"`
	RepoMetadata    *jsonRepoMetadata    `json:"repo_metadata,omitempty"`
	CompositeScore  float64              `json:"composite_score"`
	Tier            string               `json:"tier"`
	Categories      []jsonCategory       `json:"categories"`
	Recommendations []jsonRecommendation `json:"recommendations"`
	BadgeURL        string               `json:"badge_url,omitempty"`
	BadgeMarkdown   string               `json:"badge_markdown,omitempty"`
}

// jsonRepoMetadata represents repository metadata in JSON output.
type jsonRepoMetadata struct {
	LinesOfCode       int                `json:"lines_of_code"`
	FirstCommitDate   string             `json:"first_commit_date,omitempty"`
	LastCommitDate    string             `json:"last_commit_date,omitempty"`
	ContributorCount  int                `json:"contributor_count"`
	TotalCommits      int                `json:"total_commits"`
	LanguageBreakdown map[string]float64 `json:"language_breakdown"`
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

// jsonRecommendation represents a single recommendation in JSON output.
type jsonRecommendation struct {
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

// BuildJSONReport converts a ScoredResult and recommendations into a jsonReport.
// The verbose parameter is deprecated; sub_scores are always included.
// When includeBadge is true, badge URL and markdown are included.
func BuildJSONReport(scored *types.ScoredResult, recs []recommend.Recommendation, verbose bool, includeBadge bool, meta *types.RepoMetadata) *jsonReport {
	report := initializejsonReport(scored)
	addRepoMetadata(report, meta)
	buildCategories(report, scored.Categories)
	buildRecommendations(report, recs)
	addBadgeIfRequested(report, scored, includeBadge)
	return report
}

func initializejsonReport(scored *types.ScoredResult) *jsonReport {
	return &jsonReport{
		Version:        "3",
		CompositeScore: scored.Composite,
		Tier:           scored.Tier,
	}
}

func addRepoMetadata(report *jsonReport, meta *types.RepoMetadata) {
	if meta == nil {
		return
	}
	jm := &jsonRepoMetadata{
		LinesOfCode:       meta.LinesOfCode,
		ContributorCount:  meta.ContributorCount,
		TotalCommits:      meta.TotalCommits,
		LanguageBreakdown: make(map[string]float64),
	}
	if !meta.FirstCommitDate.IsZero() {
		jm.FirstCommitDate = meta.FirstCommitDate.Format(time.RFC3339)
	}
	if !meta.LastCommitDate.IsZero() {
		jm.LastCommitDate = meta.LastCommitDate.Format(time.RFC3339)
	}
	for lang, pct := range meta.LanguageBreakdown {
		jm.LanguageBreakdown[string(lang)] = pct
	}
	report.RepoMetadata = jm
}

func buildCategories(report *jsonReport, categories []types.CategoryScore) {
	for _, cat := range categories {
		jc := jsonCategory{
			Name:      cat.Name,
			Score:     cat.Score,
			Weight:    cat.Weight,
			Available: cat.Score >= 0,
			SubScores: buildSubScores(cat.SubScores),
		}
		report.Categories = append(report.Categories, jc)
	}
}

func buildSubScores(subScores []types.SubScore) []jsonMetric {
	result := make([]jsonMetric, 0, len(subScores))
	for _, ss := range subScores {
		ev := ss.Evidence
		if ev == nil {
			ev = make([]types.EvidenceItem, 0)
		}
		result = append(result, jsonMetric{
			Name:      ss.MetricName,
			RawValue:  ss.RawValue,
			Score:     ss.Score,
			Weight:    ss.Weight,
			Available: ss.Available,
			Evidence:  ev,
		})
	}
	return result
}

func buildRecommendations(report *jsonReport, recs []recommend.Recommendation) {
	for _, rec := range recs {
		report.Recommendations = append(report.Recommendations, jsonRecommendation{
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
}

func addBadgeIfRequested(report *jsonReport, scored *types.ScoredResult, includeBadge bool) {
	if includeBadge && scored != nil {
		badge := generateBadge(scored)
		report.BadgeURL = badge.URL
		report.BadgeMarkdown = badge.Markdown
	}
}

// ParseJSONBaseline parses a JSON report byte slice and extracts the baseline
// scoring result needed for trend comparison. It returns a minimal ScoredResult
// containing composite score, tier, and per-category scores/weights.
func ParseJSONBaseline(data []byte) (*types.ScoredResult, error) {
	var report jsonReport
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, err
	}
	result := &types.ScoredResult{
		Composite: report.CompositeScore,
		Tier:      report.Tier,
	}
	for _, cat := range report.Categories {
		result.Categories = append(result.Categories, types.CategoryScore{
			Name:   cat.Name,
			Score:  cat.Score,
			Weight: cat.Weight,
		})
	}
	return result, nil
}

// RenderJSON writes the JSON report to w with pretty-printed indentation.
func RenderJSON(w io.Writer, report *jsonReport) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(report)
}
