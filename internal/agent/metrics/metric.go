// Package metrics provides 5 MECE agent evaluation metrics for C7.
//
// The metrics are:
//   - M1: Task Execution Consistency - measures reproducibility across runs
//   - M2: Code Behavior Comprehension - measures understanding of code semantics
//   - M3: Cross-File Navigation - measures dependency tracing ability
//   - M4: Identifier Interpretability - measures name-based purpose inference
//   - M5: Documentation Accuracy Detection - measures comment/code mismatch detection
package metrics

import (
	"context"
	"time"

	"github.com/ingo/agent-readyness/pkg/types"
)

// Metric defines a single MECE agent evaluation capability.
type Metric interface {
	ID() string                    // e.g., "task_execution_consistency"
	Name() string                  // e.g., "Task Execution Consistency"
	Description() string           // What this metric measures
	Timeout() time.Duration        // Per-metric timeout
	SampleCount() int              // Number of samples to evaluate (1-5)
	SelectSamples(targets []*types.AnalysisTarget) []Sample
	Execute(ctx context.Context, workDir string, samples []Sample, executor Executor) MetricResult
}

// Sample represents a code sample selected for metric evaluation.
type Sample struct {
	FilePath       string  // Absolute path to file
	FunctionName   string  // Optional: specific function/method
	StartLine      int     // Optional: line range start
	EndLine        int     // Optional: line range end
	SelectionScore float64 // Score used for deterministic selection
	Description    string  // Why this sample was selected
}

// SampleResult holds the outcome of evaluating one sample.
type SampleResult struct {
	Sample   Sample
	Score    int           // 1-10 scale
	Response string        // Agent's response
	Duration time.Duration // How long this sample took
	Error    string        // Empty if successful
}

// MetricResult holds the complete outcome of a metric evaluation.
type MetricResult struct {
	MetricID   string
	MetricName string
	Score      int           // 1-10 aggregate score
	Samples    []SampleResult
	TokensUsed int
	Duration   time.Duration
	Error      string // Empty if successful
}

// Executor abstracts Claude CLI execution for testability.
type Executor interface {
	ExecutePrompt(ctx context.Context, workDir, prompt, tools string, timeout time.Duration) (response string, err error)
}

// Stub constructors - replaced in Task 2 with real implementations.
// These exist so registry.go compiles before metric implementations exist.

// NewM1Consistency creates the Task Execution Consistency metric.
func NewM1Consistency() Metric {
	return &stubMetric{id: "task_execution_consistency", name: "Task Execution Consistency"}
}

// NewM2Comprehension creates the Code Behavior Comprehension metric.
func NewM2Comprehension() Metric {
	return &stubMetric{id: "code_behavior_comprehension", name: "Code Behavior Comprehension"}
}

// NewM3Navigation creates the Cross-File Navigation metric.
func NewM3Navigation() Metric {
	return &stubMetric{id: "cross_file_navigation", name: "Cross-File Navigation"}
}

// NewM4Identifiers creates the Identifier Interpretability metric.
func NewM4Identifiers() Metric {
	return &stubMetric{id: "identifier_interpretability", name: "Identifier Interpretability"}
}

// NewM5Documentation creates the Documentation Accuracy Detection metric.
func NewM5Documentation() Metric {
	return &stubMetric{id: "documentation_accuracy_detection", name: "Documentation Accuracy Detection"}
}

// stubMetric is a placeholder until real implementations exist.
type stubMetric struct {
	id   string
	name string
}

func (s *stubMetric) ID() string      { return s.id }
func (s *stubMetric) Name() string    { return s.name }
func (s *stubMetric) Description() string { return "Stub implementation" }
func (s *stubMetric) Timeout() time.Duration { return 60 * time.Second }
func (s *stubMetric) SampleCount() int { return 1 }
func (s *stubMetric) SelectSamples(_ []*types.AnalysisTarget) []Sample { return nil }
func (s *stubMetric) Execute(_ context.Context, _ string, _ []Sample, _ Executor) MetricResult {
	return MetricResult{MetricID: s.id, MetricName: s.name, Error: "stub implementation"}
}
