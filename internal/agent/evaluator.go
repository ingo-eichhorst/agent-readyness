package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"time"
)

// EvaluationResult holds the result of content quality evaluation.
type EvaluationResult struct {
	Score  int    `json:"score"`  // 1-10
	Reason string `json:"reason"` // Brief explanation
}

// Evaluator performs content quality evaluation using the Claude CLI.
type Evaluator struct {
	timeout time.Duration
}

// NewEvaluator creates an Evaluator with the specified timeout.
// If timeout is 0, a default of 60 seconds is used.
func NewEvaluator(timeout time.Duration) *Evaluator {
	if timeout == 0 {
		timeout = 60 * time.Second
	}
	return &Evaluator{timeout: timeout}
}

// EvaluateContent runs content evaluation using the Claude CLI.
// The systemPrompt provides evaluation criteria, and content is the material to evaluate.
func (e *Evaluator) EvaluateContent(ctx context.Context, systemPrompt, content string) (EvaluationResult, error) {
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

	cmd := exec.CommandContext(evalCtx, "claude", args...)

	// Graceful cancellation: send SIGINT first
	cmd.Cancel = func() error {
		return cmd.Process.Signal(os.Interrupt)
	}
	// Grace period before force-kill after SIGINT
	cmd.WaitDelay = 10 * time.Second

	// Execute and capture output
	output, err := cmd.CombinedOutput()
	if err != nil {
		if evalCtx.Err() == context.DeadlineExceeded {
			return EvaluationResult{}, fmt.Errorf("evaluation timed out after %v", e.timeout)
		}
		// Include output in error for debugging
		preview := string(output)
		if len(preview) > 200 {
			preview = preview[:200] + "..."
		}
		return EvaluationResult{}, fmt.Errorf("CLI execution failed: %w (output: %s)", err, preview)
	}

	// Parse JSON response
	// Claude CLI returns: {"session_id": "...", "result": "...", "structured_output": {...}}
	var resp struct {
		SessionID        string           `json:"session_id"`
		Result           string           `json:"result"`
		StructuredOutput EvaluationResult `json:"structured_output"`
	}

	if err := json.Unmarshal(output, &resp); err != nil {
		preview := string(output)
		if len(preview) > 200 {
			preview = preview[:200] + "..."
		}
		return EvaluationResult{}, fmt.Errorf("failed to parse CLI response: %w (got: %s)", err, preview)
	}

	// Validate score range
	if resp.StructuredOutput.Score < 1 || resp.StructuredOutput.Score > 10 {
		return EvaluationResult{}, fmt.Errorf("score out of range (1-10): %d", resp.StructuredOutput.Score)
	}

	return resp.StructuredOutput, nil
}

// EvaluateWithRetry runs EvaluateContent with one retry on failure.
func (e *Evaluator) EvaluateWithRetry(ctx context.Context, systemPrompt, content string) (EvaluationResult, error) {
	result, err := e.EvaluateContent(ctx, systemPrompt, content)
	if err == nil {
		return result, nil
	}

	// Check if context is already canceled
	if ctx.Err() != nil {
		return EvaluationResult{}, ctx.Err()
	}

	// Wait before retry
	select {
	case <-ctx.Done():
		return EvaluationResult{}, ctx.Err()
	case <-time.After(2 * time.Second):
	}

	// Retry once
	result, err = e.EvaluateContent(ctx, systemPrompt, content)
	if err != nil {
		return EvaluationResult{}, fmt.Errorf("evaluation failed after retry: %w", err)
	}

	return result, nil
}
