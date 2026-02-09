package agent

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestCheckClaudeCLI_Available(t *testing.T) {
	origLookPath := execLookPath
	defer func() { execLookPath = origLookPath }()

	execLookPath = func(file string) (string, error) {
		return "/usr/local/bin/claude", nil
	}

	err := CheckClaudeCLI()
	if err != nil {
		t.Errorf("CheckClaudeCLI returned error when claude is available: %v", err)
	}
}

func TestCheckClaudeCLI_ErrorMessage(t *testing.T) {
	origLookPath := execLookPath
	defer func() { execLookPath = origLookPath }()

	execLookPath = func(file string) (string, error) {
		return "", fmt.Errorf("%w: claude", exec.ErrNotFound)
	}

	testErr := CheckClaudeCLI()
	if testErr == nil {
		t.Fatal("Expected error when claude is not found")
	}

	expectedSubstrings := []string{
		"claude CLI not found",
		"https://claude.ai/install.sh",
		"brew install",
		"npm install",
	}

	errMsg := testErr.Error()
	for _, substr := range expectedSubstrings {
		if !containsStr(errMsg, substr) {
			t.Errorf("error message missing expected substring %q: got %q", substr, errMsg)
		}
	}
}

func TestAllTasksDefined(t *testing.T) {
	tasks := allTasks()

	if len(tasks) != 4 {
		t.Errorf("expected 4 tasks, got %d", len(tasks))
	}

	expectedIDs := []string{
		"intent_clarity",
		"modification_confidence",
		"cross_file_coherence",
		"semantic_completeness",
	}

	for i, task := range tasks {
		// Verify non-empty fields
		if task.ID == "" {
			t.Errorf("task %d has empty ID", i)
		}
		if task.Name == "" {
			t.Errorf("task %d (%s) has empty Name", i, task.ID)
		}
		if task.Description == "" {
			t.Errorf("task %d (%s) has empty Description", i, task.ID)
		}
		if task.Prompt == "" {
			t.Errorf("task %d (%s) has empty Prompt", i, task.ID)
		}
		if task.ToolsAllowed == "" {
			t.Errorf("task %d (%s) has empty ToolsAllowed", i, task.ID)
		}

		// Verify reasonable timeout
		if task.TimeoutSeconds < 60 {
			t.Errorf("task %d (%s) timeout too short: %d seconds", i, task.ID, task.TimeoutSeconds)
		}
		if task.TimeoutSeconds > 600 {
			t.Errorf("task %d (%s) timeout too long: %d seconds", i, task.ID, task.TimeoutSeconds)
		}

		// Verify expected ID
		if i < len(expectedIDs) && task.ID != expectedIDs[i] {
			t.Errorf("task %d expected ID %q, got %q", i, expectedIDs[i], task.ID)
		}
	}
}

func TestTaskResultStatuses(t *testing.T) {
	// Verify status constants are defined correctly
	statuses := []taskStatus{
		statusCompleted,
		statusTimeout,
		statusError,
		statusCLINotFound,
	}

	expectedValues := []string{
		"completed",
		"timeout",
		"error",
		"cli_not_found",
	}

	for i, status := range statuses {
		if string(status) != expectedValues[i] {
			t.Errorf("status %d expected %q, got %q", i, expectedValues[i], status)
		}
	}
}

func TestCreateWorkspace_NonGitDir(t *testing.T) {
	// Create a temp directory without .git
	tmpDir, err := os.MkdirTemp("", "ars-test-nongit-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	workDir, cleanup, err := CreateWorkspace(tmpDir)
	if err != nil {
		t.Fatalf("CreateWorkspace failed: %v", err)
	}

	// Should return original dir in fallback mode
	if workDir != tmpDir {
		t.Errorf("expected workDir to be original dir %q, got %q", tmpDir, workDir)
	}

	// Cleanup should be callable (no-op)
	cleanup()
}

func TestCreateWorkspace_WithGitRepo(t *testing.T) {
	// Use the actual ARS repo for this test
	// Find repo root by looking for .git
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}

	// Walk up to find repo root
	repoRoot := cwd
	for {
		if _, err := os.Stat(filepath.Join(repoRoot, ".git")); err == nil {
			break
		}
		parent := filepath.Dir(repoRoot)
		if parent == repoRoot {
			t.Skip("not running in a git repository")
		}
		repoRoot = parent
	}

	workDir, cleanup, err := CreateWorkspace(repoRoot)
	if err != nil {
		t.Fatalf("CreateWorkspace failed: %v", err)
	}

	// Worktree should be different from original
	if workDir == repoRoot {
		// This could happen if worktree creation failed silently
		t.Log("worktree creation fell back to read-only mode")
	} else {
		// Verify worktree directory exists
		if _, err := os.Stat(workDir); os.IsNotExist(err) {
			t.Errorf("worktree dir %q does not exist", workDir)
		}

		// Verify it looks like a git worktree (has .git file or dir)
		gitPath := filepath.Join(workDir, ".git")
		if _, err := os.Stat(gitPath); os.IsNotExist(err) {
			t.Errorf("worktree %q missing .git", workDir)
		}
	}

	// Cleanup
	cleanup()

	// Verify worktree was removed (if it was created)
	if workDir != repoRoot {
		if _, err := os.Stat(workDir); !os.IsNotExist(err) {
			t.Errorf("worktree dir %q still exists after cleanup", workDir)
		}
	}
}

func TestExecutor_JSONParsing_Valid(t *testing.T) {
	validJSON := `{"type":"result","session_id":"abc123","result":"The main function is in cmd/main.go"}`

	resp, err := parseJSONOutput([]byte(validJSON))
	if err != nil {
		t.Fatalf("parseJSONOutput failed on valid JSON: %v", err)
	}

	if resp.Type != "result" {
		t.Errorf("expected type 'result', got %q", resp.Type)
	}
	if resp.SessionID != "abc123" {
		t.Errorf("expected session_id 'abc123', got %q", resp.SessionID)
	}
	if resp.Result != "The main function is in cmd/main.go" {
		t.Errorf("unexpected result: %q", resp.Result)
	}
}

func TestExecutor_JSONParsing_Malformed(t *testing.T) {
	malformedCases := []struct {
		name  string
		input string
	}{
		{"empty", ""},
		{"not json", "Error: authentication failed"},
		{"partial json", `{"type":"result"`},
		{"html error", "<html>Error</html>"},
	}

	for _, tc := range malformedCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseJSONOutput([]byte(tc.input))
			if err == nil {
				t.Errorf("expected error for malformed input %q", tc.input)
			}
		})
	}
}

func TestExecutor_JSONParsing_EmptyFields(t *testing.T) {
	// Valid JSON but with empty fields
	emptyResult := `{"type":"result","session_id":"","result":""}`

	resp, err := parseJSONOutput([]byte(emptyResult))
	if err != nil {
		t.Fatalf("parseJSONOutput failed: %v", err)
	}

	// Empty fields are valid - the agent might have nothing to say
	if resp.Result != "" {
		t.Errorf("expected empty result, got %q", resp.Result)
	}
}

// containsStr is a simple string contains check
func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
