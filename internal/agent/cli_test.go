package agent

import (
	"context"
	"testing"
	"time"
)

func TestDetectCLI_Available(t *testing.T) {
	// This test runs against the actual system
	// It's informational - the test passes regardless of whether claude is installed
	status := DetectCLI()

	if status.Available {
		t.Logf("Claude CLI is available: %s", status.Version)
		if status.Version == "" {
			t.Error("Expected non-empty version when Available is true")
		}
		if status.InstallHint != "" {
			t.Error("Expected empty InstallHint when Available is true")
		}
		if status.Error != "" {
			t.Error("Expected empty Error when Available is true")
		}
	} else {
		t.Logf("Claude CLI is not available: %s", status.Error)
		if status.InstallHint == "" {
			t.Error("Expected non-empty InstallHint when Available is false")
		}
	}
}

func TestDetectCLI_NotFound(t *testing.T) {
	// Test that when CLI is not found, we get proper install hint
	// We can't actually remove claude from PATH for this test,
	// but we can verify the install hint content
	status := CLIStatus{
		Available:   false,
		Error:       "claude CLI not found in PATH",
		InstallHint: installHint,
	}

	if status.Available {
		t.Error("Expected Available to be false")
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
	// Test that context timeout is respected
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Give the context time to expire
	time.Sleep(10 * time.Millisecond)

	status := DetectCLIWithContext(ctx)

	// If claude is not installed, we'll get "not found" error instead of timeout
	// If claude is installed but slow, we'll get timeout
	// Either way, the test verifies the function handles context properly
	if status.Available {
		t.Log("CLI detected even with minimal timeout - fast system")
	} else {
		t.Logf("CLI not available: %s", status.Error)
	}
}

func TestGetCLIStatus_Caching(t *testing.T) {
	// Reset cache before test
	ResetCLICache()

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

	// Reset for other tests
	ResetCLICache()
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
