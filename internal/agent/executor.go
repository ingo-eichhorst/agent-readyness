package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"time"
)

// Executor manages Claude CLI subprocess invocation for agent tasks.
type Executor struct {
	workDir string // Isolated workspace path for agent execution
}

// NewExecutor creates an Executor that runs tasks in the given work directory.
func NewExecutor(workDir string) *Executor {
	return &Executor{workDir: workDir}
}

// CheckClaudeCLI verifies that the Claude CLI is installed and accessible.
// Returns nil if available, or a descriptive error with installation instructions.
func CheckClaudeCLI() error {
	_, err := exec.LookPath("claude")
	if err != nil {
		return fmt.Errorf("claude CLI not found. Install from: https://claude.ai/install.sh\n" +
			"Alternative methods:\n" +
			"  - brew install --cask claude-code\n" +
			"  - npm install -g @anthropic-ai/claude-code")
	}
	return nil
}

// CLIResponse represents the JSON output from Claude CLI in headless mode.
type CLIResponse struct {
	Type      string `json:"type"`       // "result"
	SessionID string `json:"session_id"` // Session identifier
	Result    string `json:"result"`     // Agent's text response
}

// ExecuteTask runs a single task against the Claude CLI.
// Uses graceful timeout handling with SIGINT before force-kill.
func (e *Executor) ExecuteTask(ctx context.Context, task Task) TaskResult {
	result := TaskResult{
		TaskID:    task.ID,
		StartTime: time.Now(),
	}

	// Default timeout if not specified
	timeout := task.TimeoutSeconds
	if timeout <= 0 {
		timeout = 300 // 5 minutes default
	}

	// Create timeout context
	taskCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	// Build command with JSON output
	args := []string{
		"-p", task.Prompt,
		"--output-format", "json",
	}
	if task.ToolsAllowed != "" {
		args = append(args, "--allowedTools", task.ToolsAllowed)
	}

	cmd := exec.CommandContext(taskCtx, "claude", args...)
	cmd.Dir = e.workDir

	// Graceful cancellation: send SIGINT first (Go 1.20+)
	cmd.Cancel = func() error {
		return cmd.Process.Signal(os.Interrupt)
	}
	// Grace period before force-kill after SIGINT
	cmd.WaitDelay = 10 * time.Second

	// Capture both stdout and stderr for error diagnosis
	output, err := cmd.CombinedOutput()
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	// Handle execution errors
	if err != nil {
		if taskCtx.Err() == context.DeadlineExceeded {
			result.Status = StatusTimeout
			result.Error = fmt.Sprintf("task timed out after %d seconds", timeout)
			return result
		}

		// Check if CLI was not found
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.Status = StatusError
			result.Error = fmt.Sprintf("exit code %d: %s", exitErr.ExitCode(), string(output))
		} else if _, lookErr := exec.LookPath("claude"); lookErr != nil {
			result.Status = StatusCLINotFound
			result.Error = "claude CLI not found"
		} else {
			result.Status = StatusError
			result.Error = err.Error()
		}
		return result
	}

	// Parse JSON response
	parsed, parseErr := parseJSONOutput(output)
	if parseErr != nil {
		result.Status = StatusError
		result.Error = fmt.Sprintf("failed to parse CLI output: %v", parseErr)
		return result
	}

	result.Status = StatusCompleted
	result.Response = parsed.Result
	result.SessionID = parsed.SessionID
	return result
}

// parseJSONOutput extracts the response from Claude CLI JSON output.
func parseJSONOutput(output []byte) (*CLIResponse, error) {
	if len(output) == 0 {
		return nil, fmt.Errorf("empty output")
	}

	var resp CLIResponse
	if err := json.Unmarshal(output, &resp); err != nil {
		// Try to provide helpful context on parse failure
		preview := string(output)
		if len(preview) > 200 {
			preview = preview[:200] + "..."
		}
		return nil, fmt.Errorf("invalid JSON: %w (got: %s)", err, preview)
	}

	return &resp, nil
}
