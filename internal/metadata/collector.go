// Package metadata collects high-level repository metadata such as LOC,
// commit history, contributor count, and language breakdown.
package metadata

import (
	"bufio"
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
	f, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer f.Close()

	count := 0
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		count++
	}
	return count
}

func collectGitMetadata(meta *types.RepoMetadata, rootDir string) {
	meta.TotalCommits = gitCount(rootDir, "rev-list", "--count", "HEAD")
	meta.ContributorCount = gitLineCount(rootDir, "shortlog", "-sn", "--all", "--no-merges")
	meta.FirstCommitDate = gitTime(rootDir, "log", "--reverse", "--format=%aI", "--max-count=1")
	meta.LastCommitDate = gitTime(rootDir, "log", "--format=%aI", "--max-count=1")
}

// gitCount runs a git command that outputs a single integer and parses it.
func gitCount(rootDir string, args ...string) int {
	out := strings.TrimSpace(gitOutput(rootDir, args...))
	if out == "" {
		return 0
	}
	n, err := strconv.Atoi(out)
	if err != nil {
		return 0
	}
	return n
}

// gitLineCount runs a git command and returns the number of non-empty output lines.
func gitLineCount(rootDir string, args ...string) int {
	out := strings.TrimSpace(gitOutput(rootDir, args...))
	if out == "" {
		return 0
	}
	count := 0
	for _, line := range strings.Split(out, "\n") {
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
