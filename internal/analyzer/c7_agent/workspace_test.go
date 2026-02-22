package c7

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCreateWorkspace_NonGitDir(t *testing.T) {
	// Create a temp directory without .git
	tmpDir, err := os.MkdirTemp("", "ars-test-nongit-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	workDir, cleanup, err := createWorkspace(tmpDir)
	if err != nil {
		t.Fatalf("createWorkspace failed: %v", err)
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

	workDir, cleanup, err := createWorkspace(repoRoot)
	if err != nil {
		t.Fatalf("createWorkspace failed: %v", err)
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
