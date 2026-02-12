package agent

import (
	"context"
	"fmt"
	"time"

	"github.com/ingo-eichhorst/agent-readyness/internal/agent/metrics"
)

// cliExecutorAdapter adapts the real Claude CLI executor to the metrics.Executor interface.
type cliExecutorAdapter struct {
	workDir string
}

// newCLIExecutorAdapter creates an adapter for the given workspace directory.
func newCLIExecutorAdapter(workDir string) *cliExecutorAdapter {
	return &cliExecutorAdapter{workDir: workDir}
}

// ExecutePrompt runs a prompt via Claude CLI and returns the response.
// Implements metrics.Executor interface.
func (a *cliExecutorAdapter) ExecutePrompt(ctx context.Context, workDir, prompt, tools string, timeout time.Duration) (string, error) {
	// Use the provided workDir if specified, otherwise fall back to adapter's default
	dir := workDir
	if dir == "" {
		dir = a.workDir
	}

	// Create a task-like structure for the executor
	t := task{
		ID:             "metric_eval",
		Name:           "Metric Evaluation",
		Prompt:         prompt,
		ToolsAllowed:   tools,
		TimeoutSeconds: int(timeout.Seconds()),
	}

	exec := newExecutor(dir)
	result := exec.ExecuteTask(ctx, t)

	if result.Status != statusCompleted {
		if result.Error != "" {
			return "", fmt.Errorf("execution failed: %s", result.Error)
		}
		return "", fmt.Errorf("execution status: %s", result.Status)
	}

	return result.Response, nil
}

// Compile-time check that cliExecutorAdapter implements metrics.Executor.
var _ metrics.Executor = (*cliExecutorAdapter)(nil)
