package agent

import (
	"context"
	"sync"

	"golang.org/x/sync/errgroup"

	"github.com/ingo-eichhorst/agent-readyness/internal/agent/metrics"
	"github.com/ingo-eichhorst/agent-readyness/pkg/types"
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
	result := initResult(len(allMetrics))
	if executor == nil {
		executor = newCLIExecutorAdapter(workDir)
	}

	g, ctx := errgroup.WithContext(ctx)
	var mu sync.Mutex

	for i, m := range allMetrics {
		i, m := i, m
		g.Go(func() error {
			mr := runSingleMetric(ctx, m, workDir, targets, executor, progress)
			mu.Lock()
			result.Results[i] = mr
			updateProgress(progress, m.ID(), mr)
			mu.Unlock()
			return nil
		})
	}
	_ = g.Wait()

	for _, r := range result.Results {
		result.TotalTokens += r.TokensUsed
	}
	return result
}

// RunMetricsSequential executes all metrics sequentially (fallback/debugging).
func RunMetricsSequential(
	ctx context.Context,
	workDir string,
	targets []*types.AnalysisTarget,
	progress *C7Progress,
	executor metrics.Executor,
) ParallelResult {
	allMetrics := metrics.AllMetrics()
	result := initResult(len(allMetrics))
	if executor == nil {
		executor = newCLIExecutorAdapter(workDir)
	}

	for i, m := range allMetrics {
		mr := runSingleMetric(ctx, m, workDir, targets, executor, progress)
		result.Results[i] = mr
		result.TotalTokens += mr.TokensUsed
		updateProgress(progress, m.ID(), mr)
		if ctx.Err() != nil {
			break
		}
	}
	return result
}

// initResult creates an empty ParallelResult with pre-allocated slices.
func initResult(count int) ParallelResult {
	return ParallelResult{
		Results: make([]metrics.MetricResult, count),
		Errors:  make([]error, 0),
	}
}

// runSingleMetric selects samples, reports progress, and executes a metric.
func runSingleMetric(
	ctx context.Context,
	m metrics.Metric,
	workDir string,
	targets []*types.AnalysisTarget,
	executor metrics.Executor,
	progress *C7Progress,
) metrics.MetricResult {
	samples := m.SelectSamples(targets)
	if progress != nil {
		progress.SetMetricRunning(m.ID(), len(samples))
	}
	return m.Execute(ctx, workDir, samples, executor)
}

// updateProgress reports metric completion or failure to the progress display.
func updateProgress(progress *C7Progress, id string, mr metrics.MetricResult) {
	if progress == nil {
		return
	}
	if mr.Error != "" {
		progress.SetMetricFailed(id, mr.Error)
	} else {
		progress.SetMetricComplete(id, mr.Score)
		progress.AddTokens(mr.TokensUsed)
	}
}
