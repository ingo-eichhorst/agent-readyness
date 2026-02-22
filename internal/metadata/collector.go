// Package metadata collects high-level repository metadata such as LOC,
// commit history, contributor count, and language breakdown.
package metadata

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/ingo-eichhorst/agent-readyness/pkg/types"
)

// Collect gathers repository metadata from the given directory.
// It uses git commands for commit/contributor data and the scan result for LOC/language info.
func Collect(rootDir string, scanResult *types.ScanResult) *types.RepoMetadata {
	meta := &types.RepoMetadata{
		LanguageBreakdown: make(map[types.Language]float64),
	}

	computeLOCAndLanguages(meta, scanResult)

	if !isGitRepo(rootDir) {
		return meta
	}
	meta.Available = true

	collectGitMetadata(meta, rootDir)
	return meta
}

func isGitRepo(rootDir string) bool {
	_, err := os.Stat(filepath.Join(rootDir, ".git"))
	return err == nil
}

func computeLOCAndLanguages(meta *types.RepoMetadata, scanResult *types.ScanResult) {
	langLOC := make(map[types.Language]int)

	for _, f := range scanResult.Files {
		if f.Class != types.ClassSource {
			continue
		}
		loc := countLines(f.Path)
		meta.LinesOfCode += loc
		langLOC[f.Language] += loc
	}

	if meta.LinesOfCode > 0 {
		for lang, loc := range langLOC {
			meta.LanguageBreakdown[lang] = float64(loc) / float64(meta.LinesOfCode) * 100.0
		}
	}
}

func countLines(path string) int {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	if len(data) == 0 {
		return 0
	}
	count := 0
	for _, b := range data {
		if b == '\n' {
			count++
		}
	}
	// If file doesn't end with newline, count the last line
	if data[len(data)-1] != '\n' {
		count++
	}
	return count
}

func collectGitMetadata(meta *types.RepoMetadata, rootDir string) {
	meta.TotalCommits = gitInt(rootDir, "rev-list", "--count", "HEAD")
	meta.ContributorCount = gitInt(rootDir, "shortlog", "-sn", "--all", "--no-merges")
	meta.FirstCommitDate = gitTime(rootDir, "log", "--reverse", "--format=%aI", "--max-count=1")
	meta.LastCommitDate = gitTime(rootDir, "log", "--format=%aI", "--max-count=1")
}

// gitInt runs a git command and returns the count of output lines or the first integer found.
func gitInt(rootDir string, args ...string) int {
	out := gitOutput(rootDir, args...)
	if out == "" {
		return 0
	}

	// For rev-list --count, output is a single number
	if n, err := strconv.Atoi(strings.TrimSpace(out)); err == nil {
		return n
	}

	// For shortlog -sn, count non-empty lines
	lines := strings.Split(strings.TrimSpace(out), "\n")
	count := 0
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			count++
		}
	}
	return count
}

// gitTime runs a git command and parses the first line as an ISO 8601 timestamp.
func gitTime(rootDir string, args ...string) time.Time {
	out := gitOutput(rootDir, args...)
	out = strings.TrimSpace(out)
	if out == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, out)
	if err != nil {
		return time.Time{}
	}
	return t
}

func gitOutput(rootDir string, args ...string) string {
	cmd := exec.Command("git", args...)
	cmd.Dir = rootDir
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return string(out)
}
