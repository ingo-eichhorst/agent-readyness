package metrics

import (
	"context"
	"fmt"
	"time"

	"github.com/ingo/agent-readyness/internal/agent"
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
func (a *CLIExecutorAdapter) ExecutePrompt(ctx context.Context, workDir, prompt, tools string, timeout time.Duration) (string, error) {
	// Use the provided workDir if specified, otherwise fall back to adapter's default
	dir := workDir
	if dir == "" {
		dir = a.workDir
	}

	// Create a task-like structure for the executor
	task := agent.Task{
		ID:             "metric_eval",
		Name:           "Metric Evaluation",
		Prompt:         prompt,
		ToolsAllowed:   tools,
		TimeoutSeconds: int(timeout.Seconds()),
	}

	executor := agent.NewExecutor(dir)
	result := executor.ExecuteTask(ctx, task)

	if result.Status != agent.StatusCompleted {
		if result.Error != "" {
			return "", fmt.Errorf("execution failed: %s", result.Error)
		}
		return "", fmt.Errorf("execution status: %s", result.Status)
	}

	return result.Response, nil
}
