package agent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ingo-eichhorst/agent-readyness/internal/agent/metrics"
)

func TestSaveLoadResponses(t *testing.T) {
	tmpDir := t.TempDir()

	// Build mock MetricResult slice with 2 metrics, 1 sample each
	results := []metrics.MetricResult{
		{
			MetricID:   "code_behavior_comprehension",
			MetricName: "Code Behavior Comprehension",
			Score:      7,
			Samples: []metrics.SampleResult{
				{
					Sample: metrics.Sample{
						FilePath:    "/test/main.go",
						Description: "Main entry point",
					},
					Prompt:   "Read the file at /test/main.go and explain what the code does.",
					Response: "This code initializes the application.",
					Score:    7,
					Duration: 2500 * time.Millisecond,
					Error:    "",
				},
			},
			Duration: 3 * time.Second,
		},
		{
			MetricID:   "cross_file_navigation",
			MetricName: "Cross-File Navigation",
			Score:      5,
			Samples: []metrics.SampleResult{
				{
					Sample: metrics.Sample{
						FilePath:    "/test/handler.go",
						Description: "HTTP handler",
					},
					Prompt:   "Examine the file at /test/handler.go and trace its dependencies.",
					Response: "This file depends on net/http and the service package.",
					Score:    5,
					Duration: 3200 * time.Millisecond,
					Error:    "",
				},
			},
			Duration: 4 * time.Second,
		},
	}

	// Save
	if err := SaveResponses(tmpDir, results); err != nil {
		t.Fatalf("SaveResponses failed: %v", err)
	}

	// Verify files exist
	f1 := filepath.Join(tmpDir, "code_behavior_comprehension_0.json")
	f2 := filepath.Join(tmpDir, "cross_file_navigation_0.json")
	if _, err := os.Stat(f1); err != nil {
		t.Errorf("expected file %s to exist", f1)
	}
	if _, err := os.Stat(f2); err != nil {
		t.Errorf("expected file %s to exist", f2)
	}

	// Load
	loaded, err := LoadResponses(tmpDir)
	if err != nil {
		t.Fatalf("LoadResponses failed: %v", err)
	}

	if len(loaded) != 2 {
		t.Fatalf("loaded %d entries, want 2", len(loaded))
	}

	// Check keys
	m2, ok := loaded["code_behavior_comprehension_0"]
	if !ok {
		t.Fatal("missing key code_behavior_comprehension_0")
	}
	if m2.MetricID != "code_behavior_comprehension" {
		t.Errorf("MetricID = %q, want %q", m2.MetricID, "code_behavior_comprehension")
	}
	if m2.FilePath != "/test/main.go" {
		t.Errorf("FilePath = %q, want %q", m2.FilePath, "/test/main.go")
	}
	if m2.Response != "This code initializes the application." {
		t.Errorf("Response = %q, want %q", m2.Response, "This code initializes the application.")
	}
	if m2.Duration != 2.5 {
		t.Errorf("Duration = %f, want 2.5", m2.Duration)
	}

	m3, ok := loaded["cross_file_navigation_0"]
	if !ok {
		t.Fatal("missing key cross_file_navigation_0")
	}
	if m3.MetricID != "cross_file_navigation" {
		t.Errorf("MetricID = %q, want %q", m3.MetricID, "cross_file_navigation")
	}
}

func TestReplayExecutor(t *testing.T) {
	responses := map[string]debugResponse{
		"code_behavior_comprehension_0": {
			MetricID:    "code_behavior_comprehension",
			SampleIndex: 0,
			Response:    "The function processes HTTP requests.",
		},
	}

	executor := NewReplayExecutor(responses)
	ctx := context.Background()

	// First call should match M2 and return sample_index 0
	resp, err := executor.ExecutePrompt(ctx, "/tmp", "Read the file and explain what the code does.", "", 30*time.Second)
	if err != nil {
		t.Fatalf("ExecutePrompt failed: %v", err)
	}
	if resp != "The function processes HTTP requests." {
		t.Errorf("response = %q, want %q", resp, "The function processes HTTP requests.")
	}

	// Second call should try code_behavior_comprehension_1 which doesn't exist
	_, err = executor.ExecutePrompt(ctx, "/tmp", "Read the file and explain what the code does.", "", 30*time.Second)
	if err == nil {
		t.Fatal("expected error for missing replay data")
	}
	if !strings.Contains(err.Error(), "no replay data") {
		t.Errorf("error = %q, want to contain 'no replay data'", err.Error())
	}
}

func TestReplayExecutor_NotFound(t *testing.T) {
	executor := NewReplayExecutor(make(map[string]debugResponse))
	ctx := context.Background()

	_, err := executor.ExecutePrompt(ctx, "/tmp", "some unknown prompt", "", 30*time.Second)
	if err == nil {
		t.Fatal("expected error for empty executor")
	}
	if !strings.Contains(err.Error(), "no replay data") {
		t.Errorf("error = %q, want to contain 'no replay data'", err.Error())
	}
}

func TestReplayExecutor_ReplayedError(t *testing.T) {
	responses := map[string]debugResponse{
		"unknown_0": {
			MetricID:    "unknown",
			SampleIndex: 0,
			Response:    "",
			Error:       "CLI timeout",
		},
	}

	executor := NewReplayExecutor(responses)
	ctx := context.Background()

	_, err := executor.ExecutePrompt(ctx, "/tmp", "some unknown prompt", "", 30*time.Second)
	if err == nil {
		t.Fatal("expected error for replayed error")
	}
	if !strings.Contains(err.Error(), "replayed error") {
		t.Errorf("error = %q, want to contain 'replayed error'", err.Error())
	}
	if !strings.Contains(err.Error(), "CLI timeout") {
		t.Errorf("error = %q, want to contain 'CLI timeout'", err.Error())
	}
}

func TestIdentifyMetricFromPrompt(t *testing.T) {
	tests := []struct {
		name     string
		prompt   string
		expected string
	}{
		{
			name:     "M1 list function names",
			prompt:   "Read the file at /test/main.go and list all function names defined in it.",
			expected: "task_execution_consistency",
		},
		{
			name:     "M1 list exported function",
			prompt:   "List all exported function signatures from the package.",
			expected: "task_execution_consistency",
		},
		{
			name:     "M2 explain what the code does",
			prompt:   "Read the file at /test/handler.go and explain what the code does.",
			expected: "code_behavior_comprehension",
		},
		{
			name:     "M3 trace the dependencies",
			prompt:   "Examine the file at /test/service.go and trace the dependencies of this module.",
			expected: "cross_file_navigation",
		},
		{
			name:     "M3 trace its dependencies",
			prompt:   "Examine the file at /test/service.go and trace its dependencies.",
			expected: "cross_file_navigation",
		},
		{
			name:     "M4 interpret what the identifier",
			prompt:   `Without reading the file, interpret what the identifier "processPayment" means based ONLY on its name.`,
			expected: "identifier_interpretability",
		},
		{
			name:     "M4 interpret what each identifier",
			prompt:   "Interpret what each identifier in this list represents.",
			expected: "identifier_interpretability",
		},
		{
			name:     "M5 documentation accuracy",
			prompt:   "Analyze the documentation accuracy in /test/docs.go.",
			expected: "documentation_accuracy_detection",
		},
		{
			name:     "M5 review the documentation",
			prompt:   "Please review the documentation in this file for accuracy.",
			expected: "documentation_accuracy_detection",
		},
		{
			name:     "M5 identify any inaccuracies",
			prompt:   "Read the comments and identify any inaccuracies in the documentation.",
			expected: "documentation_accuracy_detection",
		},
		{
			name:     "unknown prompt",
			prompt:   "Hello, how are you?",
			expected: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := identifyMetricFromPrompt(tt.prompt)
			if got != tt.expected {
				t.Errorf("identifyMetricFromPrompt(%q) = %q, want %q", tt.prompt, got, tt.expected)
			}
		})
	}
}

func TestLoadResponses_NonexistentDir(t *testing.T) {
	_, err := LoadResponses("/nonexistent/dir/path")
	if err == nil {
		t.Fatal("expected error for nonexistent directory")
	}
}

func TestSaveResponses_CreatesDir(t *testing.T) {
	tmpDir := t.TempDir()
	nestedDir := filepath.Join(tmpDir, "nested", "debug", "out")

	results := []metrics.MetricResult{
		{
			MetricID: "task_execution_consistency",
			Samples: []metrics.SampleResult{
				{
					Sample:   metrics.Sample{FilePath: "/test/main.go"},
					Prompt:   "list all function names",
					Response: "func main()",
					Duration: time.Second,
				},
			},
		},
	}

	if err := SaveResponses(nestedDir, results); err != nil {
		t.Fatalf("SaveResponses failed to create nested dir: %v", err)
	}

	// Verify directory was created and file exists
	f := filepath.Join(nestedDir, "task_execution_consistency_0.json")
	if _, err := os.Stat(f); err != nil {
		t.Errorf("expected file %s to exist after SaveResponses", f)
	}
}

