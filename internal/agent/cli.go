package agent

import (
	"context"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// CLIStatus represents the availability and version of the Claude CLI.
type CLIStatus struct {
	Available   bool   // whether CLI is usable
	Version     string // CLI version string (e.g., "claude 2.1.12")
	Error       string // error message if not available
	InstallHint string // installation instructions
}

const installHint = `Claude CLI not found. Install using one of:
  curl -fsSL https://claude.ai/install.sh | bash
  brew install --cask claude-code
  npm install -g @anthropic-ai/claude-code`

// lookPathFunc and runVersionCmd are package-level variables for testing injection.
var (
	lookPathFunc  = exec.LookPath
	runVersionCmd = func(ctx context.Context, path string) ([]byte, error) {
		return exec.CommandContext(ctx, path, "--version").CombinedOutput()
	}
)

var (
	cliStatusOnce   sync.Once
	cachedCLIStatus CLIStatus
)

// cliVersionTimeout is the maximum time to wait for the Claude CLI version check.
const cliVersionTimeout = 5 * time.Second

// DetectCLI checks if the Claude CLI is installed and returns its status.
// This is a convenience wrapper around DetectCLIWithContext using a 5-second timeout.
func DetectCLI() CLIStatus {
	ctx, cancel := context.WithTimeout(context.Background(), cliVersionTimeout)
	defer cancel()
	return DetectCLIWithContext(ctx)
}

// DetectCLIWithContext checks if the Claude CLI is installed and returns its status.
// The context controls the timeout for the version check.
func DetectCLIWithContext(ctx context.Context) CLIStatus {
	// Check if CLI is in PATH
	path, err := lookPathFunc("claude")
	if err != nil {
		return CLIStatus{
			Available:   false,
			Error:       "claude CLI not found in PATH",
			InstallHint: installHint,
		}
	}

	// Run `claude --version` to get version
	output, err := runVersionCmd(ctx, path)
	if err != nil {
		// Check for context timeout
		if ctx.Err() == context.DeadlineExceeded {
			return CLIStatus{
				Available:   false,
				Error:       "timeout checking claude CLI version",
				InstallHint: installHint,
			}
		}
		return CLIStatus{
			Available:   false,
			Error:       "failed to get claude CLI version: " + err.Error(),
			InstallHint: installHint,
		}
	}

	// Parse version output (typically "claude 2.1.12" or similar)
	version := strings.TrimSpace(string(output))
	if version == "" {
		version = "unknown"
	}

	return CLIStatus{
		Available: true,
		Version:   version,
	}
}

// GetCLIStatus returns cached CLI status, detecting on first call.
// This is efficient for repeated checks within a single process.
func GetCLIStatus() CLIStatus {
	cliStatusOnce.Do(func() {
		cachedCLIStatus = DetectCLI()
	})
	return cachedCLIStatus
}

// ResetCLICache clears the cached CLI status (mainly for testing).
func ResetCLICache() {
	cliStatusOnce = sync.Once{}
	cachedCLIStatus = CLIStatus{}
}
