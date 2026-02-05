package agent

import (
	"context"
	"fmt"
	"time"

	"github.com/ingo/agent-readyness/internal/agent/metrics"
)

// CLIExecutorAdapter adapts the real Claude CLI executor to the metrics.Executor interface.
type CLIExecutorAdapter struct {
	workDir string
}

// NewCLIExecutorAdapter creates an adapter for the given workspace directory.
func NewCLIExecutorAdapter(workDir string) *CLIExecutorAdapter {
	return &CLIExecutorAdapter{workDir: workDir}
}

// ExecutePrompt runs a prompt via Claude CLI and returns the response.
// Implements metrics.Executor interface.
func (a *CLIExecutorAdapter) ExecutePrompt(ctx context.Context, workDir, prompt, tools string, timeout time.Duration) (string, error) {
	// Use the provided workDir if specified, otherwise fall back to adapter's default
	dir := workDir
	if dir == "" {
		dir = a.workDir
	}

	// Create a task-like structure for the executor
	task := Task{
		ID:             "metric_eval",
		Name:           "Metric Evaluation",
		Prompt:         prompt,
		ToolsAllowed:   tools,
		TimeoutSeconds: int(timeout.Seconds()),
	}

	executor := NewExecutor(dir)
	result := executor.ExecuteTask(ctx, task)

	if result.Status != StatusCompleted {
		if result.Error != "" {
			return "", fmt.Errorf("execution failed: %s", result.Error)
		}
		return "", fmt.Errorf("execution status: %s", result.Status)
	}

	return result.Response, nil
}

// Compile-time check that CLIExecutorAdapter implements metrics.Executor.
var _ metrics.Executor = (*CLIExecutorAdapter)(nil)
