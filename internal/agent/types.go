// Package agent provides C7 agent evaluation infrastructure for headless Claude Code execution.
package agent

import "time"

// TaskStatus represents the completion status of an agent task.
type TaskStatus string

const (
	// StatusCompleted indicates the task finished successfully.
	StatusCompleted TaskStatus = "completed"
	// StatusTimeout indicates the task exceeded its time limit.
	StatusTimeout TaskStatus = "timeout"
	// StatusError indicates the task failed with an error.
	StatusError TaskStatus = "error"
	// StatusCLINotFound indicates the Claude CLI is not installed.
	StatusCLINotFound TaskStatus = "cli_not_found"
)

// Task defines a standardized agent evaluation task.
type Task struct {
	ID             string // Unique identifier (e.g., "intent_clarity")
	Name           string // Human-readable name
	Description    string // What this task measures
	Prompt         string // The prompt sent to Claude CLI
	ToolsAllowed   string // Comma-separated list (e.g., "Read,Glob,Grep")
	TimeoutSeconds int    // Per-task timeout (default 300)
}

// TaskResult holds the outcome of executing a single task.
type TaskResult struct {
	TaskID    string        // Which task was executed
	Status    TaskStatus    // Completion status
	Response  string        // Agent's text response (if completed)
	SessionID string        // Claude session ID (for debugging)
	StartTime time.Time     // When execution began
	EndTime   time.Time     // When execution finished
	Duration  time.Duration // EndTime - StartTime
	Error     string        // Error message (if status is error)
}

// C7EvaluationResult holds the complete C7 evaluation outcome.
type C7EvaluationResult struct {
	Tasks         []TaskResult  // Results for each task
	TotalDuration time.Duration // Total wall-clock time
	CLIAvailable  bool          // Whether Claude CLI was found
}
