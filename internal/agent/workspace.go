package agent

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

// CreateWorkspace creates an isolated directory for agent execution.
// It attempts to use git worktree for efficient isolation. If the project
// is not a git repository, it falls back to read-only mode using the original
// directory (agent tasks use read-only tools, so this is safe).
//
// Returns:
//   - workDir: the directory path for agent execution
//   - cleanup: function to call when done (removes worktree if created)
//   - err: error if workspace creation fails
func CreateWorkspace(projectDir string) (workDir string, cleanup func(), err error) {
	// Create temp directory for worktree
	worktreeDir, err := os.MkdirTemp("", "ars-c7-*")
	if err != nil {
		return "", nil, fmt.Errorf("create temp dir: %w", err)
	}

	// Check if projectDir is a git repository
	gitDir := filepath.Join(projectDir, ".git")
	if _, statErr := os.Stat(gitDir); os.IsNotExist(statErr) {
		// Not a git repo - fall back to read-only mode
		os.RemoveAll(worktreeDir) // Clean up unused temp dir
		log.Printf("[C7] Warning: %s is not a git repository, using read-only mode", projectDir)
		return projectDir, func() {}, nil
	}

	// Attempt to create git worktree
	cmd := exec.Command("git", "worktree", "add", worktreeDir, "HEAD", "--detach")
	cmd.Dir = projectDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Worktree failed (maybe git issues) - fall back to read-only mode
		os.RemoveAll(worktreeDir)
		log.Printf("[C7] Warning: git worktree failed (%v), using read-only mode. Output: %s",
			err, string(output))
		return projectDir, func() {}, nil
	}

	// Worktree created successfully
	cleanup = func() {
		// Remove worktree from git
		removeCmd := exec.Command("git", "worktree", "remove", worktreeDir, "--force")
		removeCmd.Dir = projectDir
		if err := removeCmd.Run(); err != nil {
			// If git worktree remove fails, try direct removal
			log.Printf("[C7] Warning: git worktree remove failed: %v", err)
		}
		// Clean up the directory
		os.RemoveAll(worktreeDir)
	}

	return worktreeDir, cleanup, nil
}
