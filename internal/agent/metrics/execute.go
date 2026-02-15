package metrics

import (
	"context"
	"time"
)

// promptFunc generates a prompt for a given sample.
type promptFunc func(sample Sample) string

// scoreFunc scores a response and returns the score and trace.
type scoreFunc func(response string) (int, ScoreTrace)

// executeConfig holds the configuration for a standard metric execution loop.
type executeConfig struct {
	metricID   string
	metricName string
	timeout    time.Duration
	tools      string
	buildPrompt promptFunc
	scoreResponse scoreFunc
}

// executeStandardMetric runs the common Execute loop shared by m2-m5.
// It iterates over samples, executes prompts, scores responses, and aggregates results.
func executeStandardMetric(ctx context.Context, workDir string, samples []Sample, executor Executor, cfg executeConfig) MetricResult {
	result := initializeMetricResult(cfg)
	startTime := time.Now()

	if len(samples) == 0 {
		return emptyMetricResult(result, startTime)
	}

	sampleResults, totalScore, successCount := executeSamples(ctx, workDir, samples, executor, cfg)
	return finalizeMetricResult(result, sampleResults, totalScore, successCount, startTime)
}

func initializeMetricResult(cfg executeConfig) MetricResult {
	return MetricResult{
		MetricID:   cfg.metricID,
		MetricName: cfg.metricName,
	}
}

func emptyMetricResult(result MetricResult, startTime time.Time) MetricResult {
	result.Error = "no samples available for evaluation"
	result.Duration = time.Since(startTime)
	return result
}

func executeSamples(ctx context.Context, workDir string, samples []Sample, executor Executor, cfg executeConfig) ([]SampleResult, int, int) {
	timePerSample := cfg.timeout / time.Duration(len(samples))
	var sampleResults []SampleResult
	var totalScore int
	successCount := 0

	for _, sample := range samples {
		sr := executeSingleSample(ctx, workDir, sample, executor, cfg, timePerSample)
		if sr.Error == "" {
			totalScore += sr.Score
			successCount++
		}
		sampleResults = append(sampleResults, sr)
	}

	return sampleResults, totalScore, successCount
}

func executeSingleSample(ctx context.Context, workDir string, sample Sample, executor Executor, cfg executeConfig, timePerSample time.Duration) SampleResult {
	sampleStart := time.Now()
	sampleCtx, cancel := context.WithTimeout(ctx, timePerSample)
	defer cancel()

	prompt := cfg.buildPrompt(sample)
	response, err := executor.ExecutePrompt(sampleCtx, workDir, prompt, cfg.tools, timePerSample)

	sr := SampleResult{
		Sample:   sample,
		Response: response,
		Prompt:   prompt,
		Duration: time.Since(sampleStart),
	}

	if err != nil {
		sr.Error = err.Error()
		sr.Score = 0
	} else {
		sr.Score, sr.ScoreTrace = cfg.scoreResponse(response)
	}

	return sr
}

func finalizeMetricResult(result MetricResult, sampleResults []SampleResult, totalScore int, successCount int, startTime time.Time) MetricResult {
	result.Samples = sampleResults
	result.Duration = time.Since(startTime)

	if successCount == 0 {
		result.Score = 0
		result.Error = "all samples failed"
		return result
	}

	result.Score = totalScore / successCount
	return result
}
