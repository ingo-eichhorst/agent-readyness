package output

import (
	"testing"

	"github.com/ingo/agent-readyness/pkg/types"
)

func TestGenerateRadarChart(t *testing.T) {
	categories := []types.CategoryScore{
		{Name: "C1", Score: 7.5},
		{Name: "C2", Score: 8.2},
		{Name: "C3", Score: 6.1},
		{Name: "C4", Score: 7.8},
		{Name: "C5", Score: 5.5},
		{Name: "C6", Score: 8.9},
	}

	svg, err := generateRadarChart(categories)
	if err != nil {
		t.Fatalf("generateRadarChart() error = %v", err)
	}

	if svg == "" {
		t.Error("generateRadarChart() returned empty string")
	}

	// Verify SVG structure
	if len(svg) < 100 {
		t.Error("generateRadarChart() SVG too short, expected substantial content")
	}

	// SVG should contain radar-like elements
	if !containsString(svg, "<svg") {
		t.Error("generateRadarChart() SVG missing <svg tag")
	}
}

func TestGenerateRadarChart_Empty(t *testing.T) {
	svg, err := generateRadarChart(nil)
	if err != nil {
		t.Fatalf("generateRadarChart(nil) error = %v", err)
	}
	if svg != "" {
		t.Error("generateRadarChart(nil) should return empty string")
	}
}

func TestGenerateTrendChart(t *testing.T) {
	current := &types.ScoredResult{
		Categories: []types.CategoryScore{
			{Name: "C1", Score: 7.5},
			{Name: "C2", Score: 8.2},
			{Name: "C3", Score: 6.1},
		},
	}
	baseline := &types.ScoredResult{
		Categories: []types.CategoryScore{
			{Name: "C1", Score: 6.0},
			{Name: "C2", Score: 7.5},
			{Name: "C3", Score: 5.5},
		},
	}

	svg, err := generateTrendChart(current, baseline)
	if err != nil {
		t.Fatalf("generateTrendChart() error = %v", err)
	}

	if svg == "" {
		t.Error("generateTrendChart() returned empty string")
	}

	if !containsString(svg, "<svg") {
		t.Error("generateTrendChart() SVG missing <svg tag")
	}
}

func TestGenerateTrendChart_NilBaseline(t *testing.T) {
	current := &types.ScoredResult{
		Categories: []types.CategoryScore{
			{Name: "C1", Score: 7.5},
		},
	}

	svg, err := generateTrendChart(current, nil)
	if err != nil {
		t.Fatalf("generateTrendChart(nil baseline) error = %v", err)
	}
	if svg != "" {
		t.Error("generateTrendChart(nil baseline) should return empty string")
	}
}

// containsString checks if s contains substr
func containsString(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && len(s) >= len(substr) && findSubstring(s, substr)
}

func findSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
