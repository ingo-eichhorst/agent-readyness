package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"time"
)

// Evaluator configuration constants.
const (
	defaultEvalTimeout = 60 * time.Second  // Default evaluation timeout
	commandWaitDelay   = 10 * time.Second  // Grace period after SIGINT before force-kill
	retryDelay         = 2 * time.Second   // Wait before retrying failed evaluation
	errorPreviewMax    = 200               // Max characters in error output preview
	scoreMin           = 1                 // Minimum valid evaluation score
	scoreMax           = 10                // Maximum valid evaluation score
)

// evaluationResult holds the result of content quality evaluation.
type evaluationResult struct {
	Score  int    `json:"score"`  // 1-10
	Reason string `json:"reason"` // Brief explanation
}

// commandRunnerFunc executes a command and returns its combined output.
type commandRunnerFunc func(ctx context.Context, name string, args ...string) ([]byte, error)

// defaultCommandRunner runs a real CLI command with SIGINT graceful shutdown.
func defaultCommandRunner(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Cancel = func() error {
		return cmd.Process.Signal(os.Interrupt)
	}
	cmd.WaitDelay = commandWaitDelay
	return cmd.CombinedOutput()
}

// Evaluator performs content quality evaluation using the Claude CLI.
type Evaluator struct {
	timeout    time.Duration
	runCommand commandRunnerFunc
}

// NewEvaluator creates an Evaluator with the specified timeout.
// If timeout is 0, a default of 60 seconds is used.
func NewEvaluator(timeout time.Duration) *Evaluator {
	if timeout == 0 {
		timeout = defaultEvalTimeout
	}
	return &Evaluator{timeout: timeout, runCommand: defaultCommandRunner}
}

// SetCommandRunner replaces the command execution function (for testing).
func (e *Evaluator) SetCommandRunner(fn commandRunnerFunc) {
	e.runCommand = fn
}

// EvaluateContent runs content evaluation using the Claude CLI.
// The systemPrompt provides evaluation criteria, and content is the material to evaluate.
func (e *Evaluator) EvaluateContent(ctx context.Context, systemPrompt, content string) (evaluationResult, error) {
	// Build JSON schema for structured output
	schema := `{"type":"object","properties":{"score":{"type":"integer","minimum":1,"maximum":10},"reason":{"type":"string"}},"required":["score","reason"]}`

	// Build CLI args
	args := []string{
		"-p", content,
		"--system-prompt", systemPrompt,
		"--output-format", "json",
		"--json-schema", schema,
	}

	// Create command with timeout
	evalCtx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()

	// Execute and capture output
	output, err := e.runCommand(evalCtx, "claude", args...)
	if err != nil {
		if evalCtx.Err() == context.DeadlineExceeded {
			return evaluationResult{}, fmt.Errorf("evaluation timed out after %v", e.timeout)
		}
		// Include output in error for debugging
		preview := string(output)
		if len(preview) > errorPreviewMax {
			preview = preview[:errorPreviewMax] + "..."
		}
		return evaluationResult{}, fmt.Errorf("CLI execution failed: %w (output: %s)", err, preview)
	}

	// Parse JSON response
	// Claude CLI returns: {"session_id": "...", "result": "...", "structured_output": {...}}
	var resp struct {
		SessionID        string           `json:"session_id"`
		Result           string           `json:"result"`
		StructuredOutput evaluationResult `json:"structured_output"`
	}

	if err := json.Unmarshal(output, &resp); err != nil {
		preview := string(output)
		if len(preview) > errorPreviewMax {
			preview = preview[:errorPreviewMax] + "..."
		}
		return evaluationResult{}, fmt.Errorf("failed to parse CLI response: %w (got: %s)", err, preview)
	}

	// Validate score range
	if resp.StructuredOutput.Score < scoreMin || resp.StructuredOutput.Score > scoreMax {
		return evaluationResult{}, fmt.Errorf("score out of range (1-10): %d", resp.StructuredOutput.Score)
	}

	return resp.StructuredOutput, nil
}

// EvaluateWithRetry runs EvaluateContent with one retry on failure.
func (e *Evaluator) EvaluateWithRetry(ctx context.Context, systemPrompt, content string) (evaluationResult, error) {
	result, err := e.EvaluateContent(ctx, systemPrompt, content)
	if err == nil {
		return result, nil
	}

	// Check if context is already canceled
	if ctx.Err() != nil {
		return evaluationResult{}, ctx.Err()
	}

	// Wait before retry
	select {
	case <-ctx.Done():
		return evaluationResult{}, ctx.Err()
	case <-time.After(retryDelay):
	}

	// Retry once
	result, err = e.EvaluateContent(ctx, systemPrompt, content)
	if err != nil {
		return evaluationResult{}, fmt.Errorf("evaluation failed after retry: %w", err)
	}

	return result, nil
}
