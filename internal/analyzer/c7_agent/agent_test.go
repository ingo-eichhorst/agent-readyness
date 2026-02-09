package c7

import (
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ingo/agent-readyness/internal/agent"
	"github.com/ingo/agent-readyness/internal/agent/metrics"
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

func TestC7Analyzer_SetDebug(t *testing.T) {
	a := NewC7Analyzer()

	// Default state: debug off, writer is io.Discard
	if a.debug {
		t.Error("debug should be false by default")
	}
	if a.debugWriter != io.Discard {
		t.Error("debugWriter should be io.Discard by default")
	}

	// Enable debug with os.Stderr
	a.SetDebug(true, os.Stderr)

	if !a.debug {
		t.Error("debug should be true after SetDebug(true, ...)")
	}
	if a.debugWriter != os.Stderr {
		t.Error("debugWriter should be os.Stderr after SetDebug(true, os.Stderr)")
	}
}

func TestC7Analyzer_DebugWriterNeverNil(t *testing.T) {
	a := NewC7Analyzer()

	// Even without SetDebug being called, debugWriter should be io.Discard (not nil)
	if a.debugWriter == nil {
		t.Fatal("debugWriter must never be nil -- should default to io.Discard")
	}

	// Writing to io.Discard should not panic
	_, err := a.debugWriter.Write([]byte("test"))
	if err != nil {
		t.Errorf("writing to default debugWriter should not error: %v", err)
	}
}

// mockParallelResult creates a test ParallelResult with populated debug fields.
func mockParallelResult() agent.ParallelResult {
	return agent.ParallelResult{
		Results: []metrics.MetricResult{
			{
				MetricID:   "code_behavior_comprehension",
				MetricName: "Code Behavior Comprehension",
				Score:      7,
				Duration:   5 * time.Second,
				Samples: []metrics.SampleResult{
					{
						Sample:   metrics.Sample{FilePath: "pkg/handler.go", Description: "HTTP handler dispatch logic"},
						Score:    8,
						Response: "The function dispatches HTTP requests based on method.",
						Prompt:   "Explain what this function does:\nfunc dispatch(w, r) { ... }",
						ScoreTrace: metrics.ScoreTrace{
							BaseScore:  5,
							FinalScore: 8,
							Indicators: []metrics.IndicatorMatch{
								{Name: "positive:returns", Matched: true, Delta: 1},
								{Name: "positive:describes_behavior", Matched: true, Delta: 2},
								{Name: "negative:unclear", Matched: false, Delta: 0},
							},
						},
						Duration: 2 * time.Second,
					},
					{
						Sample:   metrics.Sample{FilePath: "pkg/store.go", Description: "Database query builder"},
						Score:    6,
						Response: "This builds SQL queries from filter parameters.",
						Prompt:   "Explain what this function does:\nfunc buildQuery(filters) { ... }",
						ScoreTrace: metrics.ScoreTrace{
							BaseScore:  5,
							FinalScore: 6,
							Indicators: []metrics.IndicatorMatch{
								{Name: "positive:returns", Matched: true, Delta: 1},
								{Name: "negative:vague", Matched: false, Delta: 0},
							},
						},
						Duration: 3 * time.Second,
					},
				},
			},
		},
		TotalTokens: 1500,
	}
}

func TestBuildMetrics_AlwaysPopulatesDebugSamples(t *testing.T) {
	a := NewC7Analyzer()
	// debug is false by default -- DebugSamples should still be populated
	// (debug flag only controls terminal output, not data capture)

	result := a.buildMetrics(mockParallelResult(), time.Now())

	if len(result.MetricResults) != 1 {
		t.Fatalf("expected 1 metric result, got %d", len(result.MetricResults))
	}

	mr := result.MetricResults[0]
	if len(mr.Samples) != 2 {
		t.Errorf("expected 2 sample descriptions, got %d", len(mr.Samples))
	}
	if len(mr.DebugSamples) != 2 {
		t.Errorf("expected 2 DebugSamples even when debug off, got %d", len(mr.DebugSamples))
	}
}

// TestBuildMetrics_DebugOn_PopulatesDebugSamples verifies debug samples are populated
// when debug is on. Note: debug flag now only controls terminal output, not data capture.
// DebugSamples are always populated regardless of debug flag.
func TestBuildMetrics_DebugOn_PopulatesDebugSamples(t *testing.T) {
	a := NewC7Analyzer()
	a.SetDebug(true, io.Discard)

	result := a.buildMetrics(mockParallelResult(), time.Now())

	if len(result.MetricResults) != 1 {
		t.Fatalf("expected 1 metric result, got %d", len(result.MetricResults))
	}

	mr := result.MetricResults[0]
	if len(mr.DebugSamples) != 2 {
		t.Fatalf("expected 2 debug samples, got %d", len(mr.DebugSamples))
	}

	// Verify first sample
	ds := mr.DebugSamples[0]
	if ds.FilePath != "pkg/handler.go" {
		t.Errorf("expected FilePath 'pkg/handler.go', got %q", ds.FilePath)
	}
	if ds.Description != "HTTP handler dispatch logic" {
		t.Errorf("expected Description 'HTTP handler dispatch logic', got %q", ds.Description)
	}
	if ds.Prompt != "Explain what this function does:\nfunc dispatch(w, r) { ... }" {
		t.Errorf("expected Prompt to match, got %q", ds.Prompt)
	}
	if ds.Response != "The function dispatches HTTP requests based on method." {
		t.Errorf("expected Response to match, got %q", ds.Response)
	}
	if ds.Score != 8 {
		t.Errorf("expected Score 8, got %d", ds.Score)
	}
	if ds.Duration != 2.0 {
		t.Errorf("expected Duration 2.0s, got %f", ds.Duration)
	}

	// Verify score trace
	if ds.ScoreTrace.BaseScore != 5 {
		t.Errorf("expected BaseScore 5, got %d", ds.ScoreTrace.BaseScore)
	}
	if ds.ScoreTrace.FinalScore != 8 {
		t.Errorf("expected FinalScore 8, got %d", ds.ScoreTrace.FinalScore)
	}
	if len(ds.ScoreTrace.Indicators) != 3 {
		t.Fatalf("expected 3 indicators, got %d", len(ds.ScoreTrace.Indicators))
	}
	if ds.ScoreTrace.Indicators[0].Name != "positive:returns" {
		t.Errorf("expected first indicator 'positive:returns', got %q", ds.ScoreTrace.Indicators[0].Name)
	}
	if !ds.ScoreTrace.Indicators[0].Matched {
		t.Error("expected first indicator to be matched")
	}
	if ds.ScoreTrace.Indicators[0].Delta != 1 {
		t.Errorf("expected first indicator Delta 1, got %d", ds.ScoreTrace.Indicators[0].Delta)
	}

	// Verify second sample
	ds2 := mr.DebugSamples[1]
	if ds2.FilePath != "pkg/store.go" {
		t.Errorf("expected FilePath 'pkg/store.go', got %q", ds2.FilePath)
	}
	if ds2.Score != 6 {
		t.Errorf("expected Score 6, got %d", ds2.Score)
	}
}

func TestC7Analyzer_Analyze_NoTargets(t *testing.T) {
	analyzer := NewC7Analyzer()
	// Enable with a mock evaluator
	analyzer.Enable(&agent.Evaluator{})

	// Analyze with empty targets
	_, err := analyzer.Analyze([]*types.AnalysisTarget{})
	if err == nil {
		t.Error("expected error for empty targets, got nil")
	}
	if !strings.Contains(err.Error(), "no targets provided") {
		t.Errorf("expected 'no targets provided' error, got: %v", err)
	}
}

func TestC7Analyzer_Analyze_EvaluatorNil(t *testing.T) {
	analyzer := NewC7Analyzer()
	// Don't enable - evaluator remains nil

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

	c7, ok := result.Metrics["c7"].(*types.C7Metrics)
	if !ok {
		t.Fatal("expected c7 metrics in result")
	}

	if c7.Available {
		t.Error("expected Available=false when evaluator is nil")
	}
}

func TestC7Analyzer_DisabledResult(t *testing.T) {
	analyzer := NewC7Analyzer()
	result := analyzer.disabledResult()

	if result.Name != "C7: Agent Evaluation" {
		t.Errorf("expected name 'C7: Agent Evaluation', got %q", result.Name)
	}
	if result.Category != "C7" {
		t.Errorf("expected category 'C7', got %q", result.Category)
	}

	c7, ok := result.Metrics["c7"].(*types.C7Metrics)
	if !ok {
		t.Fatal("expected c7 metrics in result")
	}
	if c7.Available {
		t.Error("disabled result should have Available=false")
	}
}

func TestC7Analyzer_SetDebugDir(t *testing.T) {
	analyzer := NewC7Analyzer()

	// Initially empty
	if analyzer.debugDir != "" {
		t.Error("debugDir should be empty by default")
	}

	analyzer.SetDebugDir("/tmp/debug")

	if analyzer.debugDir != "/tmp/debug" {
		t.Errorf("debugDir = %q, want '/tmp/debug'", analyzer.debugDir)
	}
}

func TestC7Analyzer_SetEvaluator(t *testing.T) {
	analyzer := NewC7Analyzer()

	// Initially nil
	if analyzer.evaluator != nil {
		t.Error("evaluator should be nil by default")
	}

	eval := &agent.Evaluator{}
	analyzer.SetEvaluator(eval)

	if analyzer.evaluator != eval {
		t.Error("evaluator should be set")
	}

	// Should also set enabled flag
	if !analyzer.enabled {
		t.Error("SetEvaluator should enable the analyzer")
	}
}

func TestBuildMetrics_EmptyResults(t *testing.T) {
	analyzer := NewC7Analyzer()
	startTime := time.Now()

	emptyResult := agent.ParallelResult{
		Results: []metrics.MetricResult{},
	}

	m := analyzer.buildMetrics(emptyResult, startTime)

	if !m.Available {
		t.Error("metrics should be available even with empty results")
	}
	if len(m.MetricResults) != 0 {
		t.Errorf("expected 0 metric results, got %d", len(m.MetricResults))
	}
	if m.TotalDuration <= 0 {
		t.Error("total duration should be > 0")
	}
}

func TestBuildMetrics_WithErrors(t *testing.T) {
	analyzer := NewC7Analyzer()
	startTime := time.Now()

	// Simulate some metrics succeeding, some failing
	parallelResult := agent.ParallelResult{
		Results: []metrics.MetricResult{
			{
				MetricID:   "code_navigation",
				MetricName: "Code Navigation",
				Score:      8,
				Duration:   2 * time.Second,
				Samples: []metrics.SampleResult{
					{
						Sample: metrics.Sample{FilePath: "test.go"},
						Score:  8,
					},
				},
			},
			{
				MetricID:   "task_breakdown",
				MetricName: "Task Breakdown",
				Score:      0,   // failed metric
				Duration:   100 * time.Millisecond,
				Samples:    nil, // no samples due to failure
			},
		},
	}

	metrics := analyzer.buildMetrics(parallelResult, startTime)

	if !metrics.Available {
		t.Error("metrics should be available even with some failures")
	}
	if len(metrics.MetricResults) != 2 {
		t.Errorf("expected 2 metric results, got %d", len(metrics.MetricResults))
	}

	// Check that both results are present
	foundNav := false
	foundTask := false
	for _, mr := range metrics.MetricResults {
		if mr.MetricID == "code_navigation" {
			foundNav = true
			if mr.Score != 8 {
				t.Errorf("code_navigation score = %d, want 8", mr.Score)
			}
		}
		if mr.MetricID == "task_breakdown" {
			foundTask = true
			if mr.Score != 0 {
				t.Errorf("task_breakdown score = %d, want 0", mr.Score)
			}
		}
	}
	if !foundNav || !foundTask {
		t.Error("missing expected metric results")
	}
}

func TestC7MetricResult_DebugSamples_OmitEmpty_JSON(t *testing.T) {
	// Case 1: nil DebugSamples should be omitted from JSON
	mr := types.C7MetricResult{
		MetricID:   "test_metric",
		MetricName: "Test Metric",
		Score:      7,
		Status:     "completed",
	}

	data, err := json.Marshal(mr)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	jsonStr := string(data)
	if strings.Contains(jsonStr, "debug_samples") {
		t.Errorf("JSON should NOT contain 'debug_samples' when DebugSamples is nil, got: %s", jsonStr)
	}

	// Case 2: populated DebugSamples should appear in JSON
	mr.DebugSamples = []types.C7DebugSample{
		{
			FilePath:    "test.go",
			Description: "test sample",
			Prompt:      "test prompt",
			Response:    "test response",
			Score:       7,
			Duration:    1.5,
			ScoreTrace: types.C7ScoreTrace{
				BaseScore:  5,
				FinalScore: 7,
				Indicators: []types.C7IndicatorMatch{
					{Name: "positive:test", Matched: true, Delta: 1},
				},
			},
		},
	}

	data, err = json.Marshal(mr)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	jsonStr = string(data)
	if !strings.Contains(jsonStr, "debug_samples") {
		t.Errorf("JSON should contain 'debug_samples' when DebugSamples is populated, got: %s", jsonStr)
	}
	if !strings.Contains(jsonStr, "test prompt") {
		t.Errorf("JSON should contain prompt data, got: %s", jsonStr)
	}
	if !strings.Contains(jsonStr, "positive:test") {
		t.Errorf("JSON should contain indicator name, got: %s", jsonStr)
	}
}
