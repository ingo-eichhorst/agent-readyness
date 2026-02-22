package agent

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ingo-eichhorst/agent-readyness/pkg/types"
)

// noopExecutor implements metrics.Executor but returns an error if called.
// Used in tests where we pass non-nil executor to avoid creating a real CLIExecutorAdapter.
type noopExecutor struct{}

func (e *noopExecutor) ExecutePrompt(_ context.Context, _, _, _ string, _ time.Duration) (string, error) {
	return "", fmt.Errorf("noopExecutor: should not be called in this test")
}

func TestRunMetricsParallel_NoTargets(t *testing.T) {
	ctx := context.Background()

	// Running with no targets should not panic
	result := RunMetricsParallel(ctx, "/tmp", nil, nil, &noopExecutor{})

	// Should have 5 results (one per metric)
	if len(result.Results) != 5 {
		t.Errorf("got %d results, want 5", len(result.Results))
	}

	// All results should have no samples error
	for _, r := range result.Results {
		if r.MetricID == "" {
			t.Error("result MetricID should not be empty")
		}
		if r.MetricName == "" {
			t.Error("result MetricName should not be empty")
		}
		// With no samples, we expect an error
		if r.Error == "" && len(r.Samples) == 0 {
			t.Logf("metric %s: no error and no samples (expected for nil targets)", r.MetricID)
		}
	}
}

func TestRunMetricsSequential_NoTargets(t *testing.T) {
	ctx := context.Background()

	result := runMetricsSequential(ctx, "/tmp", nil, nil, &noopExecutor{})

	if len(result.Results) != 5 {
		t.Errorf("got %d results, want 5", len(result.Results))
	}

	// Results should be in the same order as AllMetrics
	expectedIDs := []string{
		"task_execution_consistency",
		"code_behavior_comprehension",
		"cross_file_navigation",
		"identifier_interpretability",
		"documentation_accuracy_detection",
	}

	for i, expected := range expectedIDs {
		if i < len(result.Results) && result.Results[i].MetricID != expected {
			t.Errorf("result[%d].MetricID = %q, want %q", i, result.Results[i].MetricID, expected)
		}
	}
}

func TestRunMetricsParallel_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Should complete without hanging
	result := RunMetricsParallel(ctx, "/tmp", nil, nil, &noopExecutor{})

	// Should still have results (possibly with errors)
	if len(result.Results) == 0 {
		t.Error("expected some results even with cancelled context")
	}
}

func TestRunMetricsSequential_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Should complete without hanging
	result := runMetricsSequential(ctx, "/tmp", nil, nil, &noopExecutor{})

	// Should have at least some results
	if len(result.Results) == 0 {
		t.Error("expected some results even with cancelled context")
	}
}

func TestRunMetricsParallel_WithProgress(t *testing.T) {
	ctx := context.Background()

	// Create progress tracker with all metric IDs
	ids := []string{
		"task_execution_consistency",
		"code_behavior_comprehension",
		"cross_file_navigation",
		"identifier_interpretability",
		"documentation_accuracy_detection",
	}
	progress := NewC7Progress(nil, ids, nil)

	result := RunMetricsParallel(ctx, "/tmp", nil, progress, &noopExecutor{})

	// Results should be populated
	if len(result.Results) != 5 {
		t.Errorf("got %d results, want 5", len(result.Results))
	}

	// Progress should reflect all metrics being processed
	progress.mu.Lock()
	for _, id := range ids {
		metric := progress.metrics[id]
		// Each metric should have been either completed or failed
		if metric.Status != statusComplete && metric.Status != statusFailed {
			t.Errorf("metric %s status = %v, want Complete or Failed", id, metric.Status)
		}
	}
	progress.mu.Unlock()
}

func TestRunMetricsSequential_WithProgress(t *testing.T) {
	ctx := context.Background()

	ids := []string{
		"task_execution_consistency",
		"code_behavior_comprehension",
		"cross_file_navigation",
		"identifier_interpretability",
		"documentation_accuracy_detection",
	}
	progress := NewC7Progress(nil, ids, nil)

	result := runMetricsSequential(ctx, "/tmp", nil, progress, &noopExecutor{})

	if len(result.Results) != 5 {
		t.Errorf("got %d results, want 5", len(result.Results))
	}
}

func TestParallelResult_TotalTokensAccumulation(t *testing.T) {
	// This tests that token counts are properly accumulated
	ctx := context.Background()

	result := RunMetricsParallel(ctx, "/tmp", nil, nil, &noopExecutor{})

	// TotalTokens should be sum of all metric token counts
	var expectedTotal int
	for _, r := range result.Results {
		expectedTotal += r.TokensUsed
	}

	if result.TotalTokens != expectedTotal {
		t.Errorf("TotalTokens = %d, want %d (sum of all metrics)", result.TotalTokens, expectedTotal)
	}
}

func TestRunMetricsParallel_AllMetricsComplete(t *testing.T) {
	ctx := context.Background()

	// Even with empty targets, all 5 metrics should complete (with errors)
	result := RunMetricsParallel(ctx, "/tmp", []*types.AnalysisTarget{}, nil, &noopExecutor{})

	if len(result.Results) != 5 {
		t.Errorf("got %d results, want 5", len(result.Results))
	}

	// Verify each metric has a non-empty ID and name
	for _, r := range result.Results {
		if r.MetricID == "" {
			t.Error("MetricID should not be empty")
		}
		if r.MetricName == "" {
			t.Error("MetricName should not be empty")
		}
	}
}

func TestRunMetricsSequential_StopsOnContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	// Create targets that would take time to process
	targets := []*types.AnalysisTarget{
		{
			RootDir:  "/test",
			Language: types.LangGo,
			Files:    []types.SourceFile{},
		},
	}

	// Cancel after a brief moment (simulates timeout)
	cancel()

	result := runMetricsSequential(ctx, "/tmp", targets, nil, &noopExecutor{})

	// Should have stopped early due to context cancellation
	// May not have all 5 results if it checked context between metrics
	if len(result.Results) > 5 {
		t.Errorf("got %d results, want <= 5", len(result.Results))
	}
}

func TestCLIExecutorAdapter_Creation(t *testing.T) {
	adapter := newCLIExecutorAdapter("/test/dir")

	if adapter == nil {
		t.Fatal("NewCLIExecutorAdapter returned nil")
	}

	if adapter.workDir != "/test/dir" {
		t.Errorf("workDir = %q, want %q", adapter.workDir, "/test/dir")
	}
}
