package agent

import (
	"context"
	"fmt"
	"os/exec"
	"testing"
)

func TestDetectCLI_Available(t *testing.T) {
	// Swap package-level functions with mocks
	origLookPath := lookPathFunc
	origRunVersion := runVersionCmd
	defer func() {
		lookPathFunc = origLookPath
		runVersionCmd = origRunVersion
	}()

	lookPathFunc = func(file string) (string, error) {
		return "/usr/local/bin/claude", nil
	}
	runVersionCmd = func(ctx context.Context, path string) ([]byte, error) {
		return []byte("claude 2.1.12"), nil
	}

	status := detectCLI()

	if !status.Available {
		t.Fatal("Expected Available to be true")
	}
	if status.Version != "claude 2.1.12" {
		t.Errorf("Expected version %q, got %q", "claude 2.1.12", status.Version)
	}
	if status.Error != "" {
		t.Errorf("Expected empty Error, got %q", status.Error)
	}
	if status.InstallHint != "" {
		t.Errorf("Expected empty InstallHint, got %q", status.InstallHint)
	}
}

func TestDetectCLI_NotFound(t *testing.T) {
	origLookPath := lookPathFunc
	defer func() { lookPathFunc = origLookPath }()

	lookPathFunc = func(file string) (string, error) {
		return "", fmt.Errorf("%w: claude", exec.ErrNotFound)
	}

	status := detectCLI()

	if status.Available {
		t.Error("Expected Available to be false")
	}

	if status.Error != "claude CLI not found in PATH" {
		t.Errorf("Expected error %q, got %q", "claude CLI not found in PATH", status.Error)
	}

	// Verify install hint contains expected installation methods
	if status.InstallHint == "" {
		t.Fatal("Expected non-empty InstallHint")
	}

	expectedMethods := []string{
		"curl -fsSL https://claude.ai/install.sh",
		"brew install --cask claude-code",
		"npm install -g @anthropic-ai/claude-code",
	}

	for _, method := range expectedMethods {
		if !containsSubstr(status.InstallHint, method) {
			t.Errorf("InstallHint should contain %q", method)
		}
	}
}

func TestDetectCLIWithContext_Timeout(t *testing.T) {
	origLookPath := lookPathFunc
	origRunVersion := runVersionCmd
	defer func() {
		lookPathFunc = origLookPath
		runVersionCmd = origRunVersion
	}()

	lookPathFunc = func(file string) (string, error) {
		return "/usr/local/bin/claude", nil
	}
	// Mock version command that blocks until context expires
	runVersionCmd = func(ctx context.Context, path string) ([]byte, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	}

	// Use an already-expired context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	status := detectCLIWithContext(ctx)

	if status.Available {
		t.Fatal("Expected Available to be false with expired context")
	}
}

func TestGetCLIStatus_Caching(t *testing.T) {
	origLookPath := lookPathFunc
	origRunVersion := runVersionCmd
	defer func() {
		lookPathFunc = origLookPath
		runVersionCmd = origRunVersion
		resetCLICache()
	}()

	var callCount int
	lookPathFunc = func(file string) (string, error) {
		callCount++
		return "/usr/local/bin/claude", nil
	}
	runVersionCmd = func(ctx context.Context, path string) ([]byte, error) {
		return []byte("claude 3.0.0"), nil
	}

	// Reset cache before test
	resetCLICache()

	// First call should populate cache
	status1 := GetCLIStatus()

	// Second call should return same result from cache
	status2 := GetCLIStatus()

	if status1.Available != status2.Available {
		t.Error("Cached status should be consistent")
	}
	if status1.Version != status2.Version {
		t.Error("Cached version should be consistent")
	}
	if !status1.Available {
		t.Error("Expected Available to be true")
	}

	// LookPath should only have been called once (cached)
	if callCount != 1 {
		t.Errorf("Expected lookPathFunc called once, got %d", callCount)
	}
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
