// Package analyzer provides code analysis implementations for the ARS pipeline.
package analyzer

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
			Metrics:  map[string]interface{}{"c5": &types.C5Metrics{Available: false}},
		}, nil
	}

	metrics, err := analyzeGitHistory(rootDir, 6) // 6-month window
	if err != nil {
		return nil, err
	}

	return &types.AnalysisResult{
		Name:     "C5: Temporal Dynamics",
		Category: "C5",
		Metrics:  map[string]interface{}{"c5": metrics},
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
func runGitLog(rootDir string, months int) ([]commitInfo, error) {
	since := fmt.Sprintf("--since=%d months ago", months)
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
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
		if parts := strings.SplitN(line, "|", 3); len(parts) == 3 && len(parts[0]) >= 40 {
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
			parts := strings.SplitN(line, "\t", 3)
			if len(parts) == 3 {
				// Skip binary files (added/deleted shown as "-")
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

	churnRate := calcChurnRate(commits, 90)
	couplingPct, coupledPairs := calcTemporalCoupling(commits, 5, 50)
	authorFrag := calcAuthorFragmentation(commits, 90)
	stability := calcCommitStability(commits)
	hotspotPct, _ := calcHotspotConcentration(commits)

	// Build TopHotspots (up to 10) with full detail
	topHotspots := buildTopHotspots(commits, 10)

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
		TimeWindowDays:       months * 30,
	}, nil
}

// calcChurnRate computes average lines changed per commit within a time window.
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
		strength := float64(shared) / float64(minCount) * 100
		if strength > 70 {
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
	return float64(coupledPairs) / float64(totalPairs) * 100, coupled
}

// calcAuthorFragmentation computes the average distinct authors per file within a time window.
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
			days := float64(timestamps[i]-timestamps[i-1]) / 86400.0
			allIntervals = append(allIntervals, days)
		}
	}

	if len(allIntervals) == 0 {
		return 30 // default: stable if no repeat changes
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

	top10Pct := len(sorted) / 10
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

	return float64(top10Changes) / float64(totalChanges) * 100, hotspots
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
