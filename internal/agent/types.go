// Package agent provides C7 agent evaluation infrastructure for headless Claude Code execution.
package agent

import "time"

// taskStatus represents the completion status of an agent task.
type taskStatus string

const (
	// statusCompleted indicates the task finished successfully.
	statusCompleted taskStatus = "completed"
	// statusTimeout indicates the task exceeded its time limit.
	statusTimeout taskStatus = "timeout"
	// statusError indicates the task failed with an error.
	statusError taskStatus = "error"
	// statusCLINotFound indicates the Claude CLI is not installed.
	statusCLINotFound taskStatus = "cli_not_found"
)

// task defines a standardized agent evaluation task.
type task struct {
	ID             string // Unique identifier (e.g., "intent_clarity")
	Name           string // Human-readable name
	Description    string // What this task measures
	Prompt         string // The prompt sent to Claude CLI
	ToolsAllowed   string // Comma-separated list (e.g., "Read,Glob,Grep")
	TimeoutSeconds int    // Per-task timeout (default 300)
}

// taskResult holds the outcome of executing a single task.
type taskResult struct {
	TaskID    string        // Which task was executed
	Status    taskStatus    // Completion status
	Response  string        // Agent's text response (if completed)
	SessionID string        // Claude session ID (for debugging)
	StartTime time.Time     // When execution began
	EndTime   time.Time     // When execution finished
	Duration  time.Duration // EndTime - StartTime
	Error     string        // Error message (if status is error)
}

// c7EvaluationResult holds the complete C7 evaluation outcome.
type c7EvaluationResult struct {
	Tasks         []taskResult  // Results for each task
	TotalDuration time.Duration // Total wall-clock time
	CLIAvailable  bool          // Whether Claude CLI was found
}
