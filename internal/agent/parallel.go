package agent

import (
	"context"
	"sync"

	"golang.org/x/sync/errgroup"

	"github.com/ingo/agent-readyness/internal/agent/metrics"
	"github.com/ingo/agent-readyness/pkg/types"
)

// ParallelResult holds the complete outcome of parallel metric execution.
type ParallelResult struct {
	Results     []metrics.MetricResult
	TotalTokens int
	Errors      []error
}

// RunMetricsParallel executes all metrics concurrently with progress updates.
// It does not abort on individual metric failures - all metrics run to completion.
// If executor is nil, a default CLIExecutorAdapter is created for live CLI execution.
func RunMetricsParallel(
	ctx context.Context,
	workDir string,
	targets []*types.AnalysisTarget,
	progress *C7Progress,
	executor metrics.Executor,
) ParallelResult {
	allMetrics := metrics.AllMetrics()
	result := ParallelResult{
		Results: make([]metrics.MetricResult, len(allMetrics)),
		Errors:  make([]error, 0),
	}

	// Use provided executor or create default CLI adapter
	if executor == nil {
		executor = NewCLIExecutorAdapter(workDir)
	}

	// Use errgroup for concurrent execution
	g, ctx := errgroup.WithContext(ctx)
	var mu sync.Mutex // Protect results slice and errors

	for i, m := range allMetrics {
		i, m := i, m // Capture loop variables

		g.Go(func() error {
			// Select samples
			samples := m.SelectSamples(targets)

			// Update progress: starting
			if progress != nil {
				progress.SetMetricRunning(m.ID(), len(samples))
			}

			// Execute metric with progress callback
			metricResult := executeMetricWithProgress(ctx, m, workDir, samples, executor, progress)

			// Store result
			mu.Lock()
			result.Results[i] = metricResult
			if metricResult.Error != "" {
				// Don't return error - we want all metrics to complete
				// Just track that this one failed
				if progress != nil {
					progress.SetMetricFailed(m.ID(), metricResult.Error)
				}
			} else {
				if progress != nil {
					progress.SetMetricComplete(m.ID(), metricResult.Score)
					progress.AddTokens(metricResult.TokensUsed)
				}
			}
			mu.Unlock()

			// Return nil - we don't want errgroup to cancel other goroutines
			return nil
		})
	}

	// Wait for all metrics to complete
	_ = g.Wait()

	// Sum up total tokens
	for _, r := range result.Results {
		result.TotalTokens += r.TokensUsed
	}

	return result
}

// executeMetricWithProgress runs a single metric and updates progress for each sample.
func executeMetricWithProgress(
	ctx context.Context,
	m metrics.Metric,
	workDir string,
	samples []metrics.Sample,
	executor metrics.Executor,
	progress *C7Progress,
) metrics.MetricResult {
	// Execute the metric
	result := m.Execute(ctx, workDir, samples, executor)

	// Progress updates happen inside Execute() via callbacks,
	// or we can track sample progress here if needed.
	// For now, the metric handles its own sample iteration.

	return result
}

// RunMetricsSequential executes all metrics sequentially (fallback/debugging).
// If executor is nil, a default CLIExecutorAdapter is created for live CLI execution.
func RunMetricsSequential(
	ctx context.Context,
	workDir string,
	targets []*types.AnalysisTarget,
	progress *C7Progress,
	executor metrics.Executor,
) ParallelResult {
	allMetrics := metrics.AllMetrics()
	result := ParallelResult{
		Results: make([]metrics.MetricResult, len(allMetrics)),
		Errors:  make([]error, 0),
	}

	// Use provided executor or create default CLI adapter
	if executor == nil {
		executor = NewCLIExecutorAdapter(workDir)
	}

	for i, m := range allMetrics {
		samples := m.SelectSamples(targets)

		if progress != nil {
			progress.SetMetricRunning(m.ID(), len(samples))
		}

		metricResult := m.Execute(ctx, workDir, samples, executor)
		result.Results[i] = metricResult

		if metricResult.Error != "" {
			if progress != nil {
				progress.SetMetricFailed(m.ID(), metricResult.Error)
			}
		} else {
			if progress != nil {
				progress.SetMetricComplete(m.ID(), metricResult.Score)
				progress.AddTokens(metricResult.TokensUsed)
			}
		}

		result.TotalTokens += metricResult.TokensUsed

		// Check for context cancellation between metrics
		if ctx.Err() != nil {
			break
		}
	}

	return result
}
