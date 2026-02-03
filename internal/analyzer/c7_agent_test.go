package analyzer

import (
	"testing"

	"github.com/ingo/agent-readyness/pkg/types"
)

func TestC7Analyzer_DisabledByDefault(t *testing.T) {
	analyzer := NewC7Analyzer()

	targets := []*types.AnalysisTarget{
		{
			Language: types.LangGo,
			RootDir:  "/tmp/test",
		},
	}

	result, err := analyzer.Analyze(targets)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should return result with Available: false when disabled
	c7, ok := result.Metrics["c7"].(*types.C7Metrics)
	if !ok {
		t.Fatal("expected c7 metrics in result")
	}

	if c7.Available {
		t.Error("expected Available to be false when analyzer is disabled")
	}
}

func TestC7Analyzer_Name(t *testing.T) {
	analyzer := NewC7Analyzer()
	expected := "C7: Agent Evaluation"
	if analyzer.Name() != expected {
		t.Errorf("expected name %q, got %q", expected, analyzer.Name())
	}
}

func TestC7Analyzer_ResultCategory(t *testing.T) {
	analyzer := NewC7Analyzer()

	targets := []*types.AnalysisTarget{
		{
			Language: types.LangGo,
			RootDir:  "/tmp/test",
		},
	}

	result, err := analyzer.Analyze(targets)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Category != "C7" {
		t.Errorf("expected category C7, got %s", result.Category)
	}
}

func TestC7Analyzer_Enable(t *testing.T) {
	analyzer := NewC7Analyzer()

	// Before enabling, it should be disabled
	if analyzer.enabled {
		t.Error("analyzer should be disabled by default")
	}

	// Enable with nil client (just testing the flag)
	analyzer.Enable(nil)

	if !analyzer.enabled {
		t.Error("analyzer should be enabled after Enable()")
	}
}

func TestEstimateResponseTokens(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"", 0},
		{"test", 1},        // 4 chars = 1 token
		{"testtest", 2},    // 8 chars = 2 tokens
		{"12345678901234567890", 5}, // 20 chars = 5 tokens
	}

	for _, tt := range tests {
		got := estimateResponseTokens(tt.input)
		if got != tt.expected {
			t.Errorf("estimateResponseTokens(%q) = %d, want %d", tt.input, got, tt.expected)
		}
	}
}
