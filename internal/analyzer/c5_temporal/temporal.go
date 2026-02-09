// Package analyzer provides code analysis implementations for the ARS pipeline.
package c5

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ingo/agent-readyness/pkg/types"
)

// Temporal analysis constants.
const (
	defaultAnalysisMonths     = 6
	gitLogTimeout             = 25 * time.Second
	gitSHAMinLength           = 40
	defaultChurnWindowDays    = 90
	minCommitsForCoupling     = 5
	maxFilesPerCommitCoupling = 50
	topHotspotsLimit          = 10
	approxDaysPerMonth        = 30
	couplingStrengthThreshold = 70.0
	toPercentC5               = 100.0
	secondsPerDay             = 86400.0
	defaultStabilityDays      = 30.0
	hotspotTopPercentDivisor  = 10
	numstatFieldCount         = 3
)

// C5Analyzer implements the pipeline.Analyzer interface for C5: Temporal Dynamics.
// It parses git log output and computes churn rate, temporal coupling,
// author fragmentation, commit stability, and hotspot concentration.
type C5Analyzer struct{}

// NewC5Analyzer creates a C5Analyzer. No dependencies needed -- git-based analysis.
func NewC5Analyzer() *C5Analyzer {
	return &C5Analyzer{}
}

// Name returns the analyzer display name.
func (a *C5Analyzer) Name() string {
	return "C5: Temporal Dynamics"
}

// Analyze runs the C5 temporal dynamics analysis on the repository.
func (a *C5Analyzer) Analyze(targets []*types.AnalysisTarget) (*types.AnalysisResult, error) {
	if len(targets) == 0 {
		return nil, fmt.Errorf("no targets provided")
	}
	rootDir := targets[0].RootDir

	// Check for .git directory -- return unavailable if missing (not an error)
	if _, err := os.Stat(filepath.Join(rootDir, ".git")); os.IsNotExist(err) {
		return &types.AnalysisResult{
			Name:     "C5: Temporal Dynamics",
			Category: "C5",
			Metrics:  map[string]types.CategoryMetrics{"c5": &types.C5Metrics{Available: false}},
		}, nil
	}

	metrics, err := analyzeGitHistory(rootDir, defaultAnalysisMonths)
	if err != nil {
		return nil, err
	}

	return &types.AnalysisResult{
		Name:     "C5: Temporal Dynamics",
		Category: "C5",
		Metrics:  map[string]types.CategoryMetrics{"c5": metrics},
	}, nil
}

// commitInfo holds parsed data for a single git commit.
type commitInfo struct {
	Hash      string
	Author    string
	Timestamp int64
	Files     []fileChange
}

// fileChange holds per-file change data from git numstat.
type fileChange struct {
	Added   int
	Deleted int
	Path    string
}

// runGitLog executes git log and parses the output into commit structs.
//
// Git output format:
// - --pretty=format:%H|%ae|%at : hash|author_email|unix_timestamp
// - --numstat: added\tdeleted\tpath for each file in commit
// - --no-merges: excludes merge commits to focus on authored changes
//
// Rename handling: Converts git's "{old => new}" notation to final path via resolveRenamePath.
// Timeout: 25s context timeout with graceful degradation (returns partial results on timeout).
// Binary files: Skipped (git shows "-" for added/deleted counts).
func runGitLog(rootDir string, months int) ([]commitInfo, error) {
	since := fmt.Sprintf("--since=%d months ago", months)
	ctx, cancel := context.WithTimeout(context.Background(), gitLogTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "log",
		"--pretty=format:%H|%ae|%at",
		"--numstat",
		since,
		"--no-merges",
	)
	cmd.Dir = rootDir

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("git log stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("git log start: %w", err)
	}

	var commits []commitInfo
	var current *commitInfo
	scanner := bufio.NewScanner(stdout)

	for scanner.Scan() {
		line := scanner.Text()

		if line == "" {
			if current != nil {
				commits = append(commits, *current)
				current = nil
			}
			continue
		}

		// Header line: hash|author|timestamp
		// Check hash length (>=40 chars) to distinguish from numstat lines with pipes
		if parts := strings.SplitN(line, "|", numstatFieldCount); len(parts) == numstatFieldCount && len(parts[0]) >= gitSHAMinLength {
			if current != nil {
				commits = append(commits, *current)
			}
			ts, _ := strconv.ParseInt(parts[2], 10, 64)
			current = &commitInfo{
				Hash:      parts[0],
				Author:    parts[1],
				Timestamp: ts,
			}
			continue
		}

		// Numstat line: added\tdeleted\tpath
		if current != nil && strings.Contains(line, "\t") {
			parts := strings.SplitN(line, "\t", numstatFieldCount)
			if len(parts) == numstatFieldCount {
				// Skip binary files: git numstat shows "-" for added/deleted counts on binary files
				// Binary files would skew churn metrics since we can't measure meaningful line changes
				if parts[0] == "-" || parts[1] == "-" {
					continue
				}

				added, err1 := strconv.Atoi(parts[0])
				deleted, err2 := strconv.Atoi(parts[1])
				if err1 != nil || err2 != nil {
					continue
				}

				path := parts[2]
				// Handle renames: {old => new} syntax
				path = resolveRenamePath(path)
				path = filepath.ToSlash(path)

				current.Files = append(current.Files, fileChange{
					Added:   added,
					Deleted: deleted,
					Path:    path,
				})
			}
		}
	}

	// Don't forget the last commit if no trailing blank line
	if current != nil {
		commits = append(commits, *current)
	}

	// Read all output first, THEN wait
	if err := cmd.Wait(); err != nil {
		// If context was cancelled (timeout), return what we have
		if ctx.Err() != nil {
			return commits, nil
		}
		// If git returned error but we got some commits, use them
		if len(commits) > 0 {
			return commits, nil
		}
		return nil, fmt.Errorf("git log: %w", err)
	}

	return commits, nil
}

// resolveRenamePath handles git rename notation: prefix{old => new}suffix -> prefix + new + suffix
func resolveRenamePath(path string) string {
	braceStart := strings.Index(path, "{")
	braceEnd := strings.Index(path, "}")
	arrowIdx := strings.Index(path, " => ")

	if braceStart < 0 || braceEnd < 0 || arrowIdx < 0 || arrowIdx < braceStart || arrowIdx > braceEnd {
		// No rename notation -- check for simple "old => new" without braces
		if arrowIdx >= 0 {
			parts := strings.SplitN(path, " => ", 2)
			if len(parts) == 2 {
				return parts[1]
			}
		}
		return path
	}

	prefix := path[:braceStart]
	suffix := path[braceEnd+1:]
	// Extract new name from between => and }
	inside := path[braceStart+1 : braceEnd]
	arrowInside := strings.Index(inside, " => ")
	if arrowInside < 0 {
		return path
	}
	newPart := inside[arrowInside+4:]
	return prefix + newPart + suffix
}

// analyzeGitHistory parses git log and computes all C5 metrics.
func analyzeGitHistory(rootDir string, months int) (*types.C5Metrics, error) {
	commits, err := runGitLog(rootDir, months)
	if err != nil {
		return nil, err
	}

	if len(commits) == 0 {
		return &types.C5Metrics{Available: false}, nil
	}

	churnRate := calcChurnRate(commits, defaultChurnWindowDays)
	couplingPct, coupledPairs := calcTemporalCoupling(commits, minCommitsForCoupling, maxFilesPerCommitCoupling)
	authorFrag := calcAuthorFragmentation(commits, defaultChurnWindowDays)
	stability := calcCommitStability(commits)
	hotspotPct, _ := calcHotspotConcentration(commits)

	// Build TopHotspots (up to 10) with full detail
	topHotspots := buildTopHotspots(commits, topHotspotsLimit)

	return &types.C5Metrics{
		Available:            true,
		ChurnRate:            churnRate,
		TemporalCouplingPct:  couplingPct,
		AuthorFragmentation:  authorFrag,
		CommitStability:      stability,
		HotspotConcentration: hotspotPct,
		TopHotspots:          topHotspots,
		CoupledPairs:         coupledPairs,
		TotalCommits:         len(commits),
		TimeWindowDays:       months * approxDaysPerMonth,
	}, nil
}

// calcChurnRate computes average lines changed per commit within a time window.
// Lines changed = added + deleted (measures total change activity).
// 90-day window focuses on recent development patterns, ignoring historical churn.
func calcChurnRate(commits []commitInfo, windowDays int) float64 {
	cutoff := time.Now().Add(-time.Duration(windowDays) * 24 * time.Hour).Unix()
	totalLines := 0
	commitCount := 0

	for _, c := range commits {
		if c.Timestamp < cutoff {
			continue
		}
		commitCount++
		for _, f := range c.Files {
			totalLines += f.Added + f.Deleted
		}
	}

	if commitCount == 0 {
		return 0
	}
	return float64(totalLines) / float64(commitCount)
}

// calcTemporalCoupling computes the percentage of file pairs with >70% co-change rate.
//
// Algorithm:
// - Counts how often each file pair appears together in commits
// - Coupling strength = (shared commits / min(commits_A, commits_B)) * 100
// - Reports pairs with strength > 70% as "temporally coupled"
//
// Thresholds:
// - 70% coupling: Indicates files frequently change together (strong dependency signal)
// - minCommits=5: Filters out rarely-changed files (statistical significance)
// - maxFilesPerCommit=50: Excludes mass refactors/renames that skew co-change data
//
// High temporal coupling suggests hidden architectural dependencies or missing abstractions.
func calcTemporalCoupling(commits []commitInfo, minCommits int, maxFilesPerCommit int) (float64, []types.CoupledPair) {
	// Count per-file commit frequency
	fileCommitCount := make(map[string]int)
	// Count co-occurrence for file pairs
	pairCount := make(map[[2]string]int)

	for _, c := range commits {
		if len(c.Files) > maxFilesPerCommit {
			continue
		}
		paths := uniquePaths(c.Files)
		for _, p := range paths {
			fileCommitCount[p]++
		}
		// Count co-occurrences
		for i := 0; i < len(paths); i++ {
			for j := i + 1; j < len(paths); j++ {
				key := sortedPair(paths[i], paths[j])
				pairCount[key]++
			}
		}
	}

	// Calculate coupling strength
	var coupled []types.CoupledPair
	totalPairs := 0
	coupledPairs := 0

	for pair, shared := range pairCount {
		countA := fileCommitCount[pair[0]]
		countB := fileCommitCount[pair[1]]
		if countA < minCommits || countB < minCommits {
			continue
		}
		totalPairs++
		minCount := countA
		if countB < minCount {
			minCount = countB
		}
		strength := float64(shared) / float64(minCount) * toPercentC5
		if strength > couplingStrengthThreshold {
			coupledPairs++
			coupled = append(coupled, types.CoupledPair{
				FileA:         pair[0],
				FileB:         pair[1],
				Coupling:      strength,
				SharedCommits: shared,
			})
		}
	}

	if totalPairs == 0 {
		return 0, nil
	}
	return float64(coupledPairs) / float64(totalPairs) * toPercentC5, coupled
}

// calcAuthorFragmentation computes the average distinct authors per file within a time window.
// High fragmentation (many authors per file) can indicate:
// - Lack of code ownership (everyone touches everything)
// - Unclear module boundaries
func calcAuthorFragmentation(commits []commitInfo, windowDays int) float64 {
	cutoff := time.Now().Add(-time.Duration(windowDays) * 24 * time.Hour).Unix()
	fileAuthors := make(map[string]map[string]bool)

	for _, c := range commits {
		if c.Timestamp < cutoff {
			continue
		}
		for _, f := range c.Files {
			if fileAuthors[f.Path] == nil {
				fileAuthors[f.Path] = make(map[string]bool)
			}
			fileAuthors[f.Path][c.Author] = true
		}
	}

	if len(fileAuthors) == 0 {
		return 0
	}
	totalAuthors := 0
	for _, authors := range fileAuthors {
		totalAuthors += len(authors)
	}
	return float64(totalAuthors) / float64(len(fileAuthors))
}

// calcCommitStability computes the median days between changes across all files.
// Median is used instead of mean to handle outliers (very long or short gaps).
// Files with only 1 commit default to 30 days (assumes monthly review cycle).
func calcCommitStability(commits []commitInfo) float64 {
	fileTimestamps := make(map[string][]int64)
	for _, c := range commits {
		for _, f := range c.Files {
			fileTimestamps[f.Path] = append(fileTimestamps[f.Path], c.Timestamp)
		}
	}

	var allIntervals []float64
	for _, timestamps := range fileTimestamps {
		sort.Slice(timestamps, func(i, j int) bool { return timestamps[i] < timestamps[j] })
		for i := 1; i < len(timestamps); i++ {
			days := float64(timestamps[i]-timestamps[i-1]) / secondsPerDay
			allIntervals = append(allIntervals, days)
		}
	}

	if len(allIntervals) == 0 {
		return defaultStabilityDays // default: stable if no repeat changes
	}
	sort.Float64s(allIntervals)
	return allIntervals[len(allIntervals)/2] // median
}

// calcHotspotConcentration computes the percentage of total changes in the top 10% of files.
func calcHotspotConcentration(commits []commitInfo) (float64, []types.FileChurn) {
	fileChanges := make(map[string]int)
	for _, c := range commits {
		for _, f := range c.Files {
			fileChanges[f.Path] += f.Added + f.Deleted
		}
	}

	if len(fileChanges) == 0 {
		return 0, nil
	}

	type fc struct {
		path    string
		changes int
	}
	var sorted []fc
	totalChanges := 0
	for path, changes := range fileChanges {
		sorted = append(sorted, fc{path, changes})
		totalChanges += changes
	}
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].changes > sorted[j].changes })

	top10Pct := len(sorted) / hotspotTopPercentDivisor
	if top10Pct < 1 {
		top10Pct = 1
	}
	top10Changes := 0
	var hotspots []types.FileChurn
	for i := 0; i < top10Pct && i < len(sorted); i++ {
		top10Changes += sorted[i].changes
		hotspots = append(hotspots, types.FileChurn{
			Path:         sorted[i].path,
			TotalChanges: sorted[i].changes,
		})
	}

	return float64(top10Changes) / float64(totalChanges) * toPercentC5, hotspots
}

// buildTopHotspots creates detailed FileChurn entries for the top N churning files.
func buildTopHotspots(commits []commitInfo, limit int) []types.FileChurn {
	type fileStats struct {
		totalChanges int
		commitCount  int
		authors      map[string]bool
	}
	stats := make(map[string]*fileStats)

	for _, c := range commits {
		for _, f := range c.Files {
			s, ok := stats[f.Path]
			if !ok {
				s = &fileStats{authors: make(map[string]bool)}
				stats[f.Path] = s
			}
			s.totalChanges += f.Added + f.Deleted
			s.commitCount++
			s.authors[c.Author] = true
		}
	}

	type ranked struct {
		path  string
		stats *fileStats
	}
	var all []ranked
	for path, s := range stats {
		all = append(all, ranked{path, s})
	}
	sort.Slice(all, func(i, j int) bool { return all[i].stats.totalChanges > all[j].stats.totalChanges })

	if len(all) > limit {
		all = all[:limit]
	}

	var result []types.FileChurn
	for _, r := range all {
		result = append(result, types.FileChurn{
			Path:         r.path,
			TotalChanges: r.stats.totalChanges,
			CommitCount:  r.stats.commitCount,
			AuthorCount:  len(r.stats.authors),
		})
	}
	return result
}

// uniquePaths extracts unique file paths from a commit's file changes.
func uniquePaths(files []fileChange) []string {
	seen := make(map[string]bool, len(files))
	var paths []string
	for _, f := range files {
		if !seen[f.Path] {
			seen[f.Path] = true
			paths = append(paths, f.Path)
		}
	}
	sort.Strings(paths)
	return paths
}

// sortedPair returns a canonically ordered pair of strings.
func sortedPair(a, b string) [2]string {
	if a < b {
		return [2]string{a, b}
	}
	return [2]string{b, a}
}
