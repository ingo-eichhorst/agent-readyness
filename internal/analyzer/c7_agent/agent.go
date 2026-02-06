package c7

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/ingo/agent-readyness/internal/agent"
	"github.com/ingo/agent-readyness/internal/agent/metrics"
	"github.com/ingo/agent-readyness/pkg/types"
)

// C7Analyzer implements the pipeline.Analyzer interface for C7: Agent Evaluation.
type C7Analyzer struct {
	evaluator   *agent.Evaluator
	enabled     bool      // only runs if explicitly enabled
	debug       bool      // debug mode flag
	debugWriter io.Writer // where debug output goes (io.Discard or os.Stderr)
}

// NewC7Analyzer creates a C7Analyzer. It's disabled by default.
// debugWriter defaults to io.Discard to prevent nil writer if SetDebug is never called.
func NewC7Analyzer() *C7Analyzer {
	return &C7Analyzer{
		enabled:     false,
		debugWriter: io.Discard,
	}
}

// Enable activates C7 analysis with the given CLI evaluator.
func (a *C7Analyzer) Enable(evaluator *agent.Evaluator) {
	a.evaluator = evaluator
	a.enabled = true
}

// SetDebug enables debug mode with the given writer for diagnostic output.
func (a *C7Analyzer) SetDebug(enabled bool, w io.Writer) {
	a.debug = enabled
	a.debugWriter = w
}

// Name returns the analyzer display name.
func (a *C7Analyzer) Name() string {
	return "C7: Agent Evaluation"
}

// Analyze runs C7 agent evaluation using 5 MECE metrics in parallel.
func (a *C7Analyzer) Analyze(targets []*types.AnalysisTarget) (*types.AnalysisResult, error) {
	if !a.enabled {
		return a.disabledResult(), nil
	}

	if len(targets) == 0 {
		return nil, fmt.Errorf("no targets provided")
	}
	rootDir := targets[0].RootDir

	// Check for Claude CLI
	if err := agent.CheckClaudeCLI(); err != nil {
		return a.disabledResult(), nil
	}

	// Create isolated workspace
	workDir, cleanup, err := agent.CreateWorkspace(rootDir)
	if err != nil {
		return nil, fmt.Errorf("create workspace: %w", err)
	}
	defer cleanup()

	// Initialize metrics
	allMetrics := metrics.AllMetrics()
	metricIDs := make([]string, len(allMetrics))
	metricNames := make([]string, len(allMetrics))
	for i, m := range allMetrics {
		metricIDs[i] = m.ID()
		metricNames[i] = m.Name()
	}

	// Create progress display
	progress := agent.NewC7Progress(os.Stderr, metricIDs, metricNames)
	progress.Start()
	defer progress.Stop()

	// Run metrics in parallel
	ctx := context.Background()
	startTime := time.Now()

	result := agent.RunMetricsParallel(ctx, workDir, targets, progress)

	// Build C7Metrics from results
	c7metrics := a.buildMetrics(result, startTime)

	return &types.AnalysisResult{
		Name:     "C7: Agent Evaluation",
		Category: "C7",
		Metrics:  map[string]interface{}{"c7": c7metrics},
	}, nil
}

// disabledResult returns a C7 result indicating the analyzer is disabled.
func (a *C7Analyzer) disabledResult() *types.AnalysisResult {
	return &types.AnalysisResult{
		Name:     "C7: Agent Evaluation",
		Category: "C7",
		Metrics:  map[string]interface{}{"c7": &types.C7Metrics{Available: false}},
	}
}

// buildMetrics constructs C7Metrics from parallel execution results.
func (a *C7Analyzer) buildMetrics(result agent.ParallelResult, startTime time.Time) *types.C7Metrics {
	m := &types.C7Metrics{
		Available:     true,
		MetricResults: make([]types.C7MetricResult, 0, len(result.Results)),
	}

	// Process each metric result
	for _, mr := range result.Results {
		// Add to MetricResults
		metricResult := types.C7MetricResult{
			MetricID:   mr.MetricID,
			MetricName: mr.MetricName,
			Score:      mr.Score,
			Status:     "completed",
			Duration:   mr.Duration.Seconds(),
			Reasoning:  "", // Will be populated from samples
		}
		if mr.Error != "" {
			metricResult.Status = "error"
			metricResult.Reasoning = mr.Error
		}

		// Extract sample descriptions
		for _, s := range mr.Samples {
			metricResult.Samples = append(metricResult.Samples, s.Sample.Description)
		}

		m.MetricResults = append(m.MetricResults, metricResult)

		// Set individual metric scores
		switch mr.MetricID {
		case "task_execution_consistency":
			m.TaskExecutionConsistency = mr.Score
		case "code_behavior_comprehension":
			m.CodeBehaviorComprehension = mr.Score
		case "cross_file_navigation":
			m.CrossFileNavigation = mr.Score
		case "identifier_interpretability":
			m.IdentifierInterpretability = mr.Score
		case "documentation_accuracy_detection":
			m.DocumentationAccuracyDetection = mr.Score
		}
	}

	// Calculate MECE score (weighted average)
	m.MECEScore = a.calculateWeightedScore(m)

	// Token and cost tracking
	m.TokensUsed = result.TotalTokens
	m.TotalDuration = time.Since(startTime).Seconds()
	// Sonnet 4.5 blended rate ~$5/MTok
	m.CostUSD = float64(m.TokensUsed) / 1_000_000 * 5.0

	return m
}

// calculateWeightedScore computes MECE score using research-based weights.
// NOTE: These weights are duplicated from internal/scoring/config.go (C7 category).
// This is intentional - the analyzer computes a quick weighted average for display,
// while the scoring package uses the same weights for formal scoring with breakpoints.
// If weights change, update both locations.
func (a *C7Analyzer) calculateWeightedScore(m *types.C7Metrics) float64 {
	// Weights from scoring config (internal/scoring/config.go):
	// M1: 0.20, M2: 0.25, M3: 0.25, M4: 0.15, M5: 0.15
	weights := map[string]float64{
		"task_execution_consistency":       0.20,
		"code_behavior_comprehension":      0.25,
		"cross_file_navigation":            0.25,
		"identifier_interpretability":      0.15,
		"documentation_accuracy_detection": 0.15,
	}

	scores := map[string]int{
		"task_execution_consistency":       m.TaskExecutionConsistency,
		"code_behavior_comprehension":      m.CodeBehaviorComprehension,
		"cross_file_navigation":            m.CrossFileNavigation,
		"identifier_interpretability":      m.IdentifierInterpretability,
		"documentation_accuracy_detection": m.DocumentationAccuracyDetection,
	}

	totalWeight := 0.0
	weightedSum := 0.0

	for id, score := range scores {
		if score > 0 { // Only include completed metrics
			weight := weights[id]
			weightedSum += float64(score) * weight
			totalWeight += weight
		}
	}

	if totalWeight == 0 {
		return 0.0
	}
	return weightedSum / totalWeight
}

// estimateResponseTokens estimates token count from response length.
// Kept for potential backward compatibility or utility functions.
func estimateResponseTokens(response string) int {
	return len(response) / 4 // ~4 chars per token
}
