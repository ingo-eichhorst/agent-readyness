package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/ingo/agent-readyness/internal/recommend"
	"github.com/ingo/agent-readyness/pkg/types"
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
					{MetricName: "complexity_avg", RawValue: 5.0, Score: 8.5, Weight: 0.30, Available: true},
					{MetricName: "func_length_avg", RawValue: 20.0, Score: 7.8, Weight: 0.25, Available: true},
				},
			},
			{
				Name:   "C3",
				Score:  6.5,
				Weight: 0.15,
				SubScores: []types.SubScore{
					{MetricName: "max_dir_depth", RawValue: 5.0, Score: 6.0, Weight: 0.20, Available: true},
				},
			},
			{
				Name:   "C6",
				Score:  6.8,
				Weight: 0.20,
				SubScores: []types.SubScore{
					{MetricName: "coverage_percent", RawValue: 55.0, Score: 6.5, Weight: 0.30, Available: true},
					{MetricName: "test_isolation", RawValue: -1, Score: -1, Weight: 0.15, Available: false},
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
	report := BuildJSONReport(scored, recs, false)

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
	report := BuildJSONReport(scored, recs, true)

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
	report := BuildJSONReport(scored, nil, false)

	var buf bytes.Buffer
	if err := RenderJSON(&buf, report); err != nil {
		t.Fatalf("RenderJSON error: %v", err)
	}

	var parsed JSONReport
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if parsed.Version != "1" {
		t.Errorf("version = %q, want %q", parsed.Version, "1")
	}
}

func TestJSONVerboseIncludesMetrics(t *testing.T) {
	scored := newTestScoredResult()
	report := BuildJSONReport(scored, nil, true)

	var buf bytes.Buffer
	if err := RenderJSON(&buf, report); err != nil {
		t.Fatalf("RenderJSON error: %v", err)
	}

	var parsed JSONReport
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	// Verbose mode should populate metrics
	if len(parsed.Categories) == 0 {
		t.Fatal("no categories in output")
	}
	if len(parsed.Categories[0].Metrics) == 0 {
		t.Error("verbose mode should include metrics in categories")
	}

	// Check metric fields
	m := parsed.Categories[0].Metrics[0]
	if m.Name != "complexity_avg" {
		t.Errorf("metric name = %q, want %q", m.Name, "complexity_avg")
	}
	if m.RawValue != 5.0 {
		t.Errorf("metric raw_value = %v, want 5.0", m.RawValue)
	}
}

func TestJSONNonVerboseOmitsMetrics(t *testing.T) {
	scored := newTestScoredResult()
	report := BuildJSONReport(scored, nil, false)

	var buf bytes.Buffer
	if err := RenderJSON(&buf, report); err != nil {
		t.Fatalf("RenderJSON error: %v", err)
	}

	out := buf.String()
	// With omitempty, the "metrics" key should not appear
	if strings.Contains(out, `"metrics"`) {
		t.Error("non-verbose mode should omit metrics field from JSON")
	}
}

func TestJSONIncludesRecommendations(t *testing.T) {
	scored := newTestScoredResult()
	recs := newTestRecommendations()
	report := BuildJSONReport(scored, recs, false)

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
	report := BuildJSONReport(scored, nil, false)

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
	report := BuildJSONReport(scored, nil, false)

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
