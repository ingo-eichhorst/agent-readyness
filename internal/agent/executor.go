package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"time"
)

// executor configuration constants.
const (
	defaultTaskTimeout  = 300               // Default task timeout in seconds (5 minutes)
	taskCommandGraceSec = 10 * time.Second  // Grace period after SIGINT before force-kill
	outputPreviewMax    = 200               // Max characters in error output preview
)

// execLookPath is a package-level variable wrapping exec.LookPath for testing injection.
var execLookPath = exec.LookPath

// executor manages Claude CLI subprocess invocation for agent tasks.
type executor struct {
	workDir string // Isolated workspace path for agent execution
}

// newExecutor creates an executor that runs tasks in the given work directory.
func newExecutor(workDir string) *executor {
	return &executor{workDir: workDir}
}

// CheckClaudeCLI verifies that the Claude CLI is installed and accessible.
// Returns nil if available, or a descriptive error with installation instructions.
func CheckClaudeCLI() error {
	_, err := execLookPath("claude")
	if err != nil {
		return fmt.Errorf("claude CLI not found. Install from: https://claude.ai/install.sh\n" +
			"Alternative methods:\n" +
			"  - brew install --cask claude-code\n" +
			"  - npm install -g @anthropic-ai/claude-code")
	}
	return nil
}

// cliResponse represents the JSON output from Claude CLI in headless mode.
type cliResponse struct {
	Type      string `json:"type"`       // "result"
	SessionID string `json:"session_id"` // Session identifier
	Result    string `json:"result"`     // Agent's text response
}

// ExecuteTask runs a single task against the Claude CLI.
// Uses graceful timeout handling with SIGINT before force-kill.
func (e *executor) ExecuteTask(ctx context.Context, t task) taskResult {
	result := taskResult{TaskID: t.ID, StartTime: time.Now()}

	timeout := t.TimeoutSeconds
	if timeout <= 0 {
		timeout = defaultTaskTimeout
	}

	taskCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	cmd := e.buildCommand(taskCtx, t)
	output, err := cmd.CombinedOutput()
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	if err != nil {
		classifyExecError(&result, taskCtx, err, output, timeout)
		return result
	}

	parsed, parseErr := parseJSONOutput(output)
	if parseErr != nil {
		result.Status = statusError
		result.Error = fmt.Sprintf("failed to parse CLI output: %v", parseErr)
		return result
	}

	result.Status = statusCompleted
	result.Response = parsed.Result
	result.SessionID = parsed.SessionID
	return result
}

// buildCommand constructs the Claude CLI exec.Cmd with graceful cancellation.
func (e *executor) buildCommand(ctx context.Context, t task) *exec.Cmd {
	args := []string{"-p", t.Prompt, "--output-format", "json"}
	if t.ToolsAllowed != "" {
		args = append(args, "--allowedTools", t.ToolsAllowed)
	}
	cmd := exec.CommandContext(ctx, "claude", args...)
	cmd.Dir = e.workDir
	cmd.Cancel = func() error { return cmd.Process.Signal(os.Interrupt) }
	cmd.WaitDelay = taskCommandGraceSec
	return cmd
}

// classifyExecError sets the appropriate status and error on a task result.
func classifyExecError(result *taskResult, taskCtx context.Context, err error, output []byte, timeout int) {
	if taskCtx.Err() == context.DeadlineExceeded {
		result.Status = statusTimeout
		result.Error = fmt.Sprintf("task timed out after %d seconds", timeout)
		return
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		result.Status = statusError
		result.Error = fmt.Sprintf("exit code %d: %s", exitErr.ExitCode(), string(output))
	} else if _, lookErr := execLookPath("claude"); lookErr != nil {
		result.Status = statusCLINotFound
		result.Error = "claude CLI not found"
	} else {
		result.Status = statusError
		result.Error = err.Error()
	}
}

// parseJSONOutput extracts the response from Claude CLI JSON output.
func parseJSONOutput(output []byte) (*cliResponse, error) {
	if len(output) == 0 {
		return nil, fmt.Errorf("empty output")
	}

	var resp cliResponse
	if err := json.Unmarshal(output, &resp); err != nil {
		// Try to provide helpful context on parse failure
		preview := string(output)
		if len(preview) > outputPreviewMax {
			preview = preview[:outputPreviewMax] + "..."
		}
		return nil, fmt.Errorf("invalid JSON: %w (got: %s)", err, preview)
	}

	return &resp, nil
}
