package c7

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/ingo-eichhorst/agent-readyness/internal/agent"
	"github.com/ingo-eichhorst/agent-readyness/internal/agent/metrics"
	"github.com/ingo-eichhorst/agent-readyness/pkg/types"
)

// C7 analysis constants.
const (
	c7CostRatePerMTok  = 5.0       // Sonnet 4.5 blended rate $/MTok
	c7TokensPerMillion = 1_000_000 // Token-to-millions conversion
	c7CharsPerToken    = 4         // Approximate characters per token

	// MECE metric weights (duplicated from scoring config for quick display).
	c7WeightM1 = 0.20 // Task Execution Consistency
	c7WeightM2 = 0.25 // Code Behavior Comprehension
	c7WeightM3 = 0.25 // Cross-File Navigation
	c7WeightM4 = 0.15 // Identifier Interpretability
	c7WeightM5 = 0.15 // Documentation Accuracy Detection
)

// C7Analyzer implements the pipeline.Analyzer interface for C7: Agent Evaluation.
type C7Analyzer struct {
	evaluator   *agent.Evaluator
	enabled     bool      // only runs if explicitly enabled
	debug       bool      // debug mode flag
	debugWriter io.Writer // where debug output goes (io.Discard or os.Stderr)
	debugDir    string    // directory for response persistence and replay
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

// SetEvaluator sets the evaluator for C7 analysis.
// Setting a non-nil evaluator auto-enables C7; setting nil disables it.
// This method matches C4's pattern for LLM control.
func (a *C7Analyzer) SetEvaluator(eval *agent.Evaluator) {
	a.evaluator = eval
	if eval != nil {
		a.enabled = true
	} else {
		a.enabled = false
	}
}

// SetDebug enables debug mode with the given writer for diagnostic output.
func (a *C7Analyzer) SetDebug(enabled bool, w io.Writer) {
	a.debug = enabled
	a.debugWriter = w
}

// SetDebugDir configures the directory for response persistence and replay.
func (a *C7Analyzer) SetDebugDir(dir string) {
	a.debugDir = dir
}

// Name returns the analyzer display name.
func (a *C7Analyzer) Name() string {
	return "C7: Agent Evaluation"
}

// Analyze runs C7 agent evaluation using 5 MECE metrics in parallel.
func (a *C7Analyzer) Analyze(targets []*types.AnalysisTarget) (*types.AnalysisResult, error) {
	// Check if LLM features are disabled (evaluator is nil)
	if a.evaluator == nil {
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

	// Determine executor: replay from files or live CLI
	var executor metrics.Executor
	if a.debugDir != "" {
		responses, loadErr := agent.LoadResponses(a.debugDir)
		if loadErr == nil && len(responses) > 0 {
			fmt.Fprintf(a.debugWriter, "[C7 DEBUG] Replay mode: loading %d responses from %s\n", len(responses), a.debugDir)
			executor = agent.NewReplayExecutor(responses)
		} else {
			fmt.Fprintf(a.debugWriter, "[C7 DEBUG] Capture mode: responses will be saved to %s\n", a.debugDir)
		}
	}

	result := agent.RunMetricsParallel(ctx, workDir, targets, progress, executor)

	// Save responses for future replay (only when in capture mode, i.e. executor was nil)
	if a.debugDir != "" && executor == nil {
		if saveErr := agent.SaveResponses(a.debugDir, result.Results); saveErr != nil {
			fmt.Fprintf(a.debugWriter, "[C7 DEBUG] Warning: failed to save responses: %v\n", saveErr)
		} else {
			fmt.Fprintf(a.debugWriter, "[C7 DEBUG] Saved %d metric responses to %s\n", len(result.Results), a.debugDir)
		}
	}

	// Build C7Metrics from results
	c7metrics := a.buildMetrics(result, startTime)

	return &types.AnalysisResult{
		Name:     "C7: Agent Evaluation",
		Category: "C7",
		Metrics:  map[string]types.CategoryMetrics{"c7": c7metrics},
	}, nil
}

// disabledResult returns a C7 result indicating the analyzer is disabled.
func (a *C7Analyzer) disabledResult() *types.AnalysisResult {
	return &types.AnalysisResult{
		Name:     "C7: Agent Evaluation",
		Category: "C7",
		Metrics:  map[string]types.CategoryMetrics{"c7": &types.C7Metrics{Available: false}},
	}
}

// buildMetrics constructs C7Metrics from parallel execution results.
func (a *C7Analyzer) buildMetrics(result agent.ParallelResult, startTime time.Time) *types.C7Metrics {
	m := &types.C7Metrics{
		Available:     true,
		MetricResults: make([]types.C7MetricResult, 0, len(result.Results)),
	}

	a.processMetricResults(m, result.Results)
	m.MECEScore = a.calculateWeightedScore(m)
	a.populateTokensAndCost(m, result, startTime)

	return m
}

// processMetricResults processes each metric result and populates C7Metrics fields.
func (a *C7Analyzer) processMetricResults(m *types.C7Metrics, results []metrics.MetricResult) {
	for _, mr := range results {
		metricResult := a.buildMetricResult(mr)
		m.MetricResults = append(m.MetricResults, metricResult)
		a.assignIndividualScore(m, mr)
	}
}

// buildMetricResult constructs a C7MetricResult from a MetricResult.
func (a *C7Analyzer) buildMetricResult(mr metrics.MetricResult) types.C7MetricResult {
	metricResult := types.C7MetricResult{
		MetricID:   mr.MetricID,
		MetricName: mr.MetricName,
		Score:      mr.Score,
		Status:     "completed",
		Duration:   mr.Duration.Seconds(),
		Reasoning:  "",
	}

	if mr.Error != "" {
		metricResult.Status = "error"
		metricResult.Reasoning = mr.Error
	}

	a.extractSampleData(&metricResult, mr.Samples)
	return metricResult
}

// extractSampleData extracts sample descriptions and debug data from samples.
func (a *C7Analyzer) extractSampleData(metricResult *types.C7MetricResult, samples []metrics.SampleResult) {
	for _, s := range samples {
		metricResult.Samples = append(metricResult.Samples, s.Sample.Description)

		metricResult.DebugSamples = append(metricResult.DebugSamples, types.C7DebugSample{
			FilePath:    s.Sample.FilePath,
			Description: s.Sample.Description,
			Prompt:      s.Prompt,
			Response:    s.Response,
			Score:       s.Score,
			Duration:    s.Duration.Seconds(),
			ScoreTrace:  convertScoreTrace(s.ScoreTrace),
			Error:       s.Error,
		})
	}
}

// assignIndividualScore assigns the score to the appropriate C7Metrics field.
func (a *C7Analyzer) assignIndividualScore(m *types.C7Metrics, mr metrics.MetricResult) {
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

// populateTokensAndCost calculates and populates token usage, duration, and cost.
func (a *C7Analyzer) populateTokensAndCost(m *types.C7Metrics, result agent.ParallelResult, startTime time.Time) {
	m.TokensUsed = result.TotalTokens
	m.TotalDuration = time.Since(startTime).Seconds()
	m.CostUSD = float64(m.TokensUsed) / c7TokensPerMillion * c7CostRatePerMTok
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
		"task_execution_consistency":       c7WeightM1,
		"code_behavior_comprehension":      c7WeightM2,
		"cross_file_navigation":            c7WeightM3,
		"identifier_interpretability":      c7WeightM4,
		"documentation_accuracy_detection": c7WeightM5,
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

// convertScoreTrace converts an internal metrics.ScoreTrace to the output types.C7ScoreTrace.
func convertScoreTrace(st metrics.ScoreTrace) types.C7ScoreTrace {
	trace := types.C7ScoreTrace{
		BaseScore:  st.BaseScore,
		FinalScore: st.FinalScore,
	}
	for _, ind := range st.Indicators {
		trace.Indicators = append(trace.Indicators, types.C7IndicatorMatch{
			Name:    ind.Name,
			Matched: ind.Matched,
			Delta:   ind.Delta,
		})
	}
	return trace
}

// estimateResponseTokens estimates token count from response length.
// Kept for potential backward compatibility or utility functions.
func estimateResponseTokens(response string) int {
	return len(response) / c7CharsPerToken
}
