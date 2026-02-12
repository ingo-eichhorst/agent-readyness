package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/ingo-eichhorst/agent-readyness/internal/recommend"
	"github.com/ingo-eichhorst/agent-readyness/pkg/types"
)

func newTestScoredResult() *types.ScoredResult {
	return &types.ScoredResult{
		Composite: 7.2,
		Tier:      "Agent-Assisted",
		Categories: []types.CategoryScore{
			{
				Name:   "C1",
				Score:  8.1,
				Weight: 0.25,
				SubScores: []types.SubScore{
					{MetricName: "complexity_avg", RawValue: 5.0, Score: 8.5, Weight: 0.30, Available: true, Evidence: []types.EvidenceItem{
						{FilePath: "pkg/foo.go", Line: 42, Value: 15, Description: "highComplexity has complexity 15"},
					}},
					{MetricName: "func_length_avg", RawValue: 20.0, Score: 7.8, Weight: 0.25, Available: true, Evidence: []types.EvidenceItem{}},
				},
			},
			{
				Name:   "C3",
				Score:  6.5,
				Weight: 0.15,
				SubScores: []types.SubScore{
					{MetricName: "max_dir_depth", RawValue: 5.0, Score: 6.0, Weight: 0.20, Available: true, Evidence: []types.EvidenceItem{}},
				},
			},
			{
				Name:   "C6",
				Score:  6.8,
				Weight: 0.20,
				SubScores: []types.SubScore{
					{MetricName: "coverage_percent", RawValue: 55.0, Score: 6.5, Weight: 0.30, Available: true, Evidence: []types.EvidenceItem{}},
					{MetricName: "test_isolation", RawValue: -1, Score: -1, Weight: 0.15, Available: false, Evidence: []types.EvidenceItem{}},
				},
			},
		},
	}
}

func newTestRecommendations() []recommend.Recommendation {
	return []recommend.Recommendation{
		{
			Rank:             1,
			Category:         "C6",
			MetricName:       "coverage_percent",
			CurrentValue:     55.0,
			CurrentScore:     6.5,
			TargetValue:      70.0,
			ScoreImprovement: 0.6,
			Effort:           "Medium",
			Summary:          "Improve test coverage from 55.0 to 70.0",
			Action:           "Increase test coverage from 55% to 70%",
		},
		{
			Rank:             2,
			Category:         "C3",
			MetricName:       "max_dir_depth",
			CurrentValue:     5.0,
			CurrentScore:     6.0,
			TargetValue:      4.0,
			ScoreImprovement: 0.2,
			Effort:           "Low",
			Summary:          "Reduce max directory depth from 5 to 4",
			Action:           "Flatten directory structure from depth 5 to at most 4",
		},
	}
}

func TestJSONOutputValid(t *testing.T) {
	scored := newTestScoredResult()
	recs := newTestRecommendations()
	report := BuildJSONReport(scored, recs, false, false)

	var buf bytes.Buffer
	err := RenderJSON(&buf, report)
	if err != nil {
		t.Fatalf("RenderJSON error: %v", err)
	}

	if !json.Valid(buf.Bytes()) {
		t.Errorf("output is not valid JSON:\n%s", buf.String())
	}
}

func TestJSONNoANSI(t *testing.T) {
	scored := newTestScoredResult()
	recs := newTestRecommendations()
	report := BuildJSONReport(scored, recs, true, false)

	var buf bytes.Buffer
	err := RenderJSON(&buf, report)
	if err != nil {
		t.Fatalf("RenderJSON error: %v", err)
	}

	out := buf.String()
	if strings.Contains(out, "\x1b") {
		t.Error("JSON output contains ANSI escape sequences")
	}
}

func TestJSONVersion(t *testing.T) {
	scored := newTestScoredResult()
	report := BuildJSONReport(scored, nil, false, false)

	var buf bytes.Buffer
	if err := RenderJSON(&buf, report); err != nil {
		t.Fatalf("RenderJSON error: %v", err)
	}

	var parsed JSONReport
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if parsed.Version != "3" {
		t.Errorf("version = %q, want %q", parsed.Version, "3")
	}
}

func TestJSONAlwaysIncludesSubScores(t *testing.T) {
	scored := newTestScoredResult()
	report := BuildJSONReport(scored, nil, false, false)
	var buf bytes.Buffer
	if err := RenderJSON(&buf, report); err != nil {
		t.Fatalf("RenderJSON error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, `"sub_scores"`) {
		t.Error("JSON output should always include sub_scores field")
	}
	if strings.Contains(out, `"metrics"`) {
		t.Error("JSON output should use sub_scores, not metrics")
	}
}

func TestJSONSubScoresIncludeMetricFields(t *testing.T) {
	scored := newTestScoredResult()
	report := BuildJSONReport(scored, nil, true, false)

	var buf bytes.Buffer
	if err := RenderJSON(&buf, report); err != nil {
		t.Fatalf("RenderJSON error: %v", err)
	}

	var parsed JSONReport
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if len(parsed.Categories) == 0 {
		t.Fatal("no categories in output")
	}
	if len(parsed.Categories[0].SubScores) == 0 {
		t.Error("sub_scores should always be present in categories")
	}

	// Check metric fields
	m := parsed.Categories[0].SubScores[0]
	if m.Name != "complexity_avg" {
		t.Errorf("metric name = %q, want %q", m.Name, "complexity_avg")
	}
	if m.RawValue != 5.0 {
		t.Errorf("metric raw_value = %v, want 5.0", m.RawValue)
	}
}

func TestJSONIncludesRecommendations(t *testing.T) {
	scored := newTestScoredResult()
	recs := newTestRecommendations()
	report := BuildJSONReport(scored, recs, false, false)

	var buf bytes.Buffer
	if err := RenderJSON(&buf, report); err != nil {
		t.Fatalf("RenderJSON error: %v", err)
	}

	var parsed JSONReport
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if len(parsed.Recommendations) != 2 {
		t.Fatalf("recommendations count = %d, want 2", len(parsed.Recommendations))
	}

	r := parsed.Recommendations[0]
	if r.Rank != 1 {
		t.Errorf("rank = %d, want 1", r.Rank)
	}
	if r.Category != "C6" {
		t.Errorf("category = %q, want %q", r.Category, "C6")
	}
	if r.ScoreImprovement != 0.6 {
		t.Errorf("score_improvement = %v, want 0.6", r.ScoreImprovement)
	}
	if r.Effort != "Medium" {
		t.Errorf("effort = %q, want %q", r.Effort, "Medium")
	}
}

func TestJSONCompositeAndTier(t *testing.T) {
	scored := newTestScoredResult()
	report := BuildJSONReport(scored, nil, false, false)

	var buf bytes.Buffer
	if err := RenderJSON(&buf, report); err != nil {
		t.Fatalf("RenderJSON error: %v", err)
	}

	var parsed JSONReport
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if parsed.CompositeScore != 7.2 {
		t.Errorf("composite_score = %v, want 7.2", parsed.CompositeScore)
	}
	if parsed.Tier != "Agent-Assisted" {
		t.Errorf("tier = %q, want %q", parsed.Tier, "Agent-Assisted")
	}
}

func TestJSONEmptyRecommendations(t *testing.T) {
	scored := newTestScoredResult()
	report := BuildJSONReport(scored, nil, false, false)

	var buf bytes.Buffer
	if err := RenderJSON(&buf, report); err != nil {
		t.Fatalf("RenderJSON error: %v", err)
	}

	var parsed JSONReport
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	// Nil recommendations should marshal as null, which unmarshals as nil
	if parsed.Recommendations != nil {
		t.Errorf("recommendations should be nil for empty input, got %v", parsed.Recommendations)
	}
}

func TestJSONIncludesBadge(t *testing.T) {
	scored := newTestScoredResult()
	report := BuildJSONReport(scored, nil, false, true)

	var buf bytes.Buffer
	if err := RenderJSON(&buf, report); err != nil {
		t.Fatalf("RenderJSON error: %v", err)
	}

	var parsed JSONReport
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	// Badge should be present
	if parsed.BadgeURL == "" {
		t.Error("badge_url should be present when includeBadge is true")
	}
	if parsed.BadgeMarkdown == "" {
		t.Error("badge_markdown should be present when includeBadge is true")
	}

	// Check badge URL format
	if !strings.Contains(parsed.BadgeURL, "img.shields.io/badge/ARS-") {
		t.Errorf("badge_url should contain shields.io URL, got %q", parsed.BadgeURL)
	}
	if !strings.Contains(parsed.BadgeURL, "yellow") {
		t.Errorf("badge_url should contain color 'yellow' for Agent-Assisted tier, got %q", parsed.BadgeURL)
	}

	// Check markdown format
	if !strings.HasPrefix(parsed.BadgeMarkdown, "[![ARS](") {
		t.Errorf("badge_markdown should start with markdown image syntax, got %q", parsed.BadgeMarkdown)
	}
}

func TestJSONOmitsBadgeByDefault(t *testing.T) {
	scored := newTestScoredResult()
	report := BuildJSONReport(scored, nil, false, false)

	var buf bytes.Buffer
	if err := RenderJSON(&buf, report); err != nil {
		t.Fatalf("RenderJSON error: %v", err)
	}

	out := buf.String()
	// With omitempty, badge fields should not appear
	if strings.Contains(out, `"badge_url"`) {
		t.Error("badge_url should be omitted when includeBadge is false")
	}
	if strings.Contains(out, `"badge_markdown"`) {
		t.Error("badge_markdown should be omitted when includeBadge is false")
	}
}

func TestJSONEvidenceNotNull(t *testing.T) {
	scored := newTestScoredResult()
	report := BuildJSONReport(scored, nil, false, false)
	var buf bytes.Buffer
	if err := RenderJSON(&buf, report); err != nil {
		t.Fatalf("RenderJSON error: %v", err)
	}
	out := buf.String()
	// Evidence should be [] not null
	if strings.Contains(out, `"evidence": null`) {
		t.Error("evidence should be empty array [], not null")
	}
	// At least one evidence array should be present
	if !strings.Contains(out, `"evidence"`) {
		t.Error("evidence field should be present in JSON output")
	}
}

func TestJSONEvidenceWithData(t *testing.T) {
	scored := newTestScoredResult()
	report := BuildJSONReport(scored, nil, false, false)
	var buf bytes.Buffer
	if err := RenderJSON(&buf, report); err != nil {
		t.Fatalf("RenderJSON error: %v", err)
	}
	var parsed JSONReport
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	// Find the complexity_avg metric which has test evidence
	c1 := parsed.Categories[0]
	if len(c1.SubScores) == 0 {
		t.Fatal("C1 should have sub_scores")
	}
	found := false
	for _, m := range c1.SubScores {
		if m.Name == "complexity_avg" && len(m.Evidence) > 0 {
			found = true
			ev := m.Evidence[0]
			if ev.FilePath != "pkg/foo.go" {
				t.Errorf("evidence file_path = %q, want %q", ev.FilePath, "pkg/foo.go")
			}
			if ev.Line != 42 {
				t.Errorf("evidence line = %d, want 42", ev.Line)
			}
		}
	}
	if !found {
		t.Error("complexity_avg should have evidence data")
	}
}

func TestJSONBaselineBackwardCompatibility(t *testing.T) {
	// Simulate v0.0.5 JSON with "metrics" field name
	oldJSON := `{
		"version": "1",
		"composite_score": 7.5,
		"tier": "Agent-Assisted",
		"categories": [
			{"name": "C1", "score": 8.0, "weight": 0.25, "metrics": [
				{"name": "complexity_avg", "raw_value": 5.0, "score": 8.5, "weight": 0.30, "available": true}
			]}
		]
	}`
	var report JSONReport
	if err := json.Unmarshal([]byte(oldJSON), &report); err != nil {
		t.Fatalf("unmarshal old JSON: %v", err)
	}
	// Category-level fields must load correctly
	if report.Categories[0].Name != "C1" {
		t.Errorf("category name = %q, want C1", report.Categories[0].Name)
	}
	if report.Categories[0].Score != 8.0 {
		t.Errorf("category score = %v, want 8.0", report.Categories[0].Score)
	}
	// SubScores will be empty (old json tag "metrics" doesn't match new "sub_scores")
	// This is fine -- loadBaseline never reads sub-scores
	// Verify we don't crash
	if report.CompositeScore != 7.5 {
		t.Errorf("composite = %v, want 7.5", report.CompositeScore)
	}
}

func TestJSONBaselineV1FullRoundTrip(t *testing.T) {
	// Complete v1-era JSON with all 7 categories using old "metrics" field name
	v1JSON := `{
		"version": "1",
		"composite_score": 6.8,
		"tier": "Agent-Assisted",
		"categories": [
			{"name": "C1", "score": 7.5, "weight": 0.25, "metrics": [
				{"name": "complexity_avg", "raw_value": 5.0, "score": 8.0, "weight": 0.30, "available": true}
			]},
			{"name": "C2", "score": 6.0, "weight": 0.10, "metrics": [
				{"name": "type_annotation_coverage", "raw_value": 80.0, "score": 7.0, "weight": 0.25, "available": true}
			]},
			{"name": "C3", "score": 7.0, "weight": 0.20, "metrics": [
				{"name": "max_dir_depth", "raw_value": 4.0, "score": 7.5, "weight": 0.20, "available": true}
			]},
			{"name": "C4", "score": 5.5, "weight": 0.10, "metrics": [
				{"name": "readme_word_count", "raw_value": 300.0, "score": 6.0, "weight": 0.15, "available": true}
			]},
			{"name": "C5", "score": 6.2, "weight": 0.10, "metrics": [
				{"name": "churn_rate", "raw_value": 45.0, "score": 6.5, "weight": 0.25, "available": true}
			]},
			{"name": "C6", "score": 7.8, "weight": 0.15, "metrics": [
				{"name": "coverage_percent", "raw_value": 70.0, "score": 8.0, "weight": 0.30, "available": true}
			]},
			{"name": "C7", "score": 6.5, "weight": 0.10, "metrics": [
				{"name": "task_execution_consistency", "raw_value": 7.0, "score": 7.0, "weight": 0.20, "available": true}
			]}
		]
	}`

	var report JSONReport
	if err := json.Unmarshal([]byte(v1JSON), &report); err != nil {
		t.Fatalf("unmarshal v1 JSON with all 7 categories: %v", err)
	}

	// Verify top-level fields
	if report.Version != "1" {
		t.Errorf("version = %q, want %q", report.Version, "1")
	}
	if report.CompositeScore != 6.8 {
		t.Errorf("composite_score = %v, want 6.8", report.CompositeScore)
	}
	if report.Tier != "Agent-Assisted" {
		t.Errorf("tier = %q, want %q", report.Tier, "Agent-Assisted")
	}

	// Verify all 7 categories loaded
	if len(report.Categories) != 7 {
		t.Fatalf("categories count = %d, want 7", len(report.Categories))
	}

	expectedCats := []struct {
		name   string
		score  float64
		weight float64
	}{
		{"C1", 7.5, 0.25},
		{"C2", 6.0, 0.10},
		{"C3", 7.0, 0.20},
		{"C4", 5.5, 0.10},
		{"C5", 6.2, 0.10},
		{"C6", 7.8, 0.15},
		{"C7", 6.5, 0.10},
	}

	for i, want := range expectedCats {
		got := report.Categories[i]
		if got.Name != want.name {
			t.Errorf("categories[%d].name = %q, want %q", i, got.Name, want.name)
		}
		if got.Score != want.score {
			t.Errorf("categories[%d].score = %v, want %v", i, got.Score, want.score)
		}
		if got.Weight != want.weight {
			t.Errorf("categories[%d].weight = %v, want %v", i, got.Weight, want.weight)
		}
		// SubScores must be empty (old "metrics" tag doesn't match new "sub_scores")
		// This is expected -- baseline loading only reads category-level scores
		if len(got.SubScores) != 0 {
			t.Errorf("categories[%d].sub_scores should be empty for v1 JSON (old 'metrics' tag), got %d", i, len(got.SubScores))
		}
	}

	// Verify v2 marshal uses "sub_scores" not "metrics"
	t.Run("v2 output uses sub_scores", func(t *testing.T) {
		scored := newTestScoredResult()
		report := BuildJSONReport(scored, nil, false, false)

		var buf bytes.Buffer
		if err := RenderJSON(&buf, report); err != nil {
			t.Fatalf("RenderJSON error: %v", err)
		}
		out := buf.String()
		if !strings.Contains(out, `"sub_scores"`) {
			t.Error("v2 JSON should use 'sub_scores' field name")
		}
		if strings.Contains(out, `"metrics"`) {
			t.Error("v2 JSON should NOT use 'metrics' field name")
		}
	})
}
