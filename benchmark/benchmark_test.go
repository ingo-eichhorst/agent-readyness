//go:build benchmark

package benchmark

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/ingo-eichhorst/agent-readyness/internal/pipeline"
	"github.com/ingo-eichhorst/agent-readyness/internal/scoring"
)

var update = flag.Bool("update", false, "regenerate golden files")

// BenchmarkRepo represents a single repo entry in benchmark.yaml.
type BenchmarkRepo struct {
	Name                string    `yaml:"name"`
	Language            string    `yaml:"language"`
	URL                 string    `yaml:"url"`
	Commit              string    `yaml:"commit"`
	Size                string    `yaml:"size"`
	ExpectedQuality     string    `yaml:"expected_quality"`
	ExpectedScoreRange  [2]float64 `yaml:"expected_score_range"`
}

// BenchmarkConfig is the top-level structure of benchmark.yaml.
type BenchmarkConfig struct {
	Repos []BenchmarkRepo `yaml:"repos"`
}

// GoldenFile is the minimal, stable structure stored in golden files.
type GoldenFile struct {
	CompositeScore float64        `json:"composite_score"`
	Tier           string         `json:"tier"`
	Categories     []GoldenCategory `json:"categories"`
}

// GoldenCategory holds category name and score for golden comparison.
type GoldenCategory struct {
	Name  string  `json:"name"`
	Score float64 `json:"score"`
}

const scoreTolerance = 0.01

func TestBenchmarkRepos(t *testing.T) {
	flag.Parse()

	// Load benchmark config
	cfg, err := loadConfig(filepath.Join("benchmark.yaml"))
	if err != nil {
		t.Fatalf("load benchmark.yaml: %v", err)
	}

	reposDir := filepath.Join("repos")
	goldenDir := filepath.Join("golden")

	// Clone all repos in parallel
	fmt.Fprintf(os.Stderr, "Cloning %d repos...\n", len(cfg.Repos))
	cloneRepos(t, cfg.Repos, reposDir)

	// Results table for summary
	type result struct {
		name    string
		lang    string
		score   float64
		tier    string
		pass    bool
		errMsg  string
	}
	total := len(cfg.Repos)
	results := make([]result, total)

	var mu sync.Mutex
	var wg sync.WaitGroup
	var startCount int
	var completedCount int

	fmt.Fprintf(os.Stderr, "\nAnalyzing %d repos...\n", total)

	for i, repo := range cfg.Repos {
		i, repo := i, repo
		wg.Add(1)
		go func() {
			defer wg.Done()

			mu.Lock()
			startCount++
			startIdx := startCount
			mu.Unlock()
			fmt.Fprintf(os.Stderr, "[%d/%d] analyzing %s (%s, %s)...\n", startIdx, total, repo.Name, repo.Language, repo.Size)

			start := time.Now()
			repoDir := filepath.Join(reposDir, repo.Language, repo.Name)
			scored, scanErr := scanRepo(repoDir)

			r := result{name: repo.Name, lang: repo.Language}

			if scanErr != nil {
				r.errMsg = scanErr.Error()
			} else {
				golden := scored
				r.score = golden.CompositeScore
				r.tier = golden.Tier

				goldenPath := filepath.Join(goldenDir, repo.Language, repo.Name+".json")

				if *update {
					if writeErr := writeGolden(goldenPath, golden); writeErr != nil {
						r.errMsg = fmt.Sprintf("write golden: %v", writeErr)
					} else {
						r.pass = true
					}
				} else {
					if cmpErr := compareGolden(goldenPath, golden); cmpErr != nil {
						r.errMsg = cmpErr.Error()
					} else {
						r.pass = true
					}
				}
			}

			elapsed := time.Since(start)

			mu.Lock()
			completedCount++
			idx := completedCount
			results[i] = r
			mu.Unlock()

			if r.errMsg != "" {
				fmt.Fprintf(os.Stderr, "[%d/%d] %s: %.2f FAIL (expected %.0f–%.0f) [%s]\n",
					idx, total, repo.Name, r.score,
					repo.ExpectedScoreRange[0], repo.ExpectedScoreRange[1],
					elapsed.Round(time.Millisecond))
			} else {
				fmt.Fprintf(os.Stderr, "[%d/%d] %s: %.2f %s ✓ [%s]\n",
					idx, total, repo.Name, r.score, r.tier,
					elapsed.Round(time.Millisecond))
			}
		}()
	}

	wg.Wait()

	// Print summary table
	fmt.Fprintln(os.Stderr)
	fmt.Fprintf(os.Stderr, "%-20s %-12s %8s %-16s %s\n", "REPO", "LANGUAGE", "SCORE", "TIER", "STATUS")
	fmt.Fprintln(os.Stderr, strings.Repeat("-", 75))
	anyFail := false
	for _, r := range results {
		status := "PASS"
		if r.errMsg != "" {
			status = "FAIL: " + r.errMsg
			anyFail = true
		}
		fmt.Fprintf(os.Stderr, "%-20s %-12s %8.2f %-16s %s\n", r.name, r.lang, r.score, r.tier, status)
	}
	fmt.Fprintln(os.Stderr)

	if anyFail {
		t.Fail()
	}

	// Generate HTML report from golden files.
	fmt.Fprintln(os.Stderr, "Generating report...")
	cmd := exec.Command("go", "run", "./report")
	cmd.Dir = filepath.Dir(goldenDir) // benchmark/
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Logf("report generation failed: %v\n%s", err, out)
	} else {
		fmt.Fprintf(os.Stderr, "%s", out)
	}
}

// loadConfig reads and parses benchmark.yaml.
func loadConfig(path string) (*BenchmarkConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg BenchmarkConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// cloneRepos clones repos in parallel, skipping if already at the correct commit.
func cloneRepos(t *testing.T, repos []BenchmarkRepo, reposDir string) {
	t.Helper()
	var wg sync.WaitGroup
	var mu sync.Mutex
	var errs []string
	var completedCount int
	total := len(repos)

	for _, repo := range repos {
		repo := repo
		wg.Add(1)
		go func() {
			defer wg.Done()
			start := time.Now()
			dir := filepath.Join(reposDir, repo.Language, repo.Name)
			err := ensureRepo(dir, repo.URL, repo.Commit)
			elapsed := time.Since(start)

			mu.Lock()
			completedCount++
			idx := completedCount
			if err != nil {
				errs = append(errs, fmt.Sprintf("%s: %v", repo.Name, err))
			}
			mu.Unlock()

			status := "ok"
			if err != nil {
				status = "FAIL"
			}
			fmt.Fprintf(os.Stderr, "[%d/%d] cloned  %-20s  %8s  %s\n", idx, total, repo.Name, elapsed.Round(time.Millisecond), status)
		}()
	}

	wg.Wait()
	if len(errs) > 0 {
		t.Fatalf("clone errors:\n%s", strings.Join(errs, "\n"))
	}
}

// ensureRepo clones or verifies a repo at the given commit using shallow fetch.
func ensureRepo(dir, url, commit string) error {
	// Check if already cloned at correct commit
	if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
		head, err := runGit(dir, "rev-parse", "HEAD")
		if err == nil && strings.TrimSpace(head) != "" {
			// Repo exists; check if commit matches (prefix match for short hashes)
			head = strings.TrimSpace(head)
			if strings.HasPrefix(head, commit) || strings.HasPrefix(commit, head) {
				return nil
			}
			// Also check by resolving the tag/ref
			resolved, err := runGit(dir, "rev-list", "-n1", commit)
			if err == nil && strings.TrimSpace(resolved) == head {
				return nil
			}
		}
	}

	// Create directory
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}

	// Remove any partial clone
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		_ = os.RemoveAll(filepath.Join(dir, e.Name()))
	}

	// Init, fetch the specific ref
	if _, err := runGit(dir, "init"); err != nil {
		return fmt.Errorf("git init: %w", err)
	}

	// Try shallow fetch of the commit/tag
	if _, err := runGit(dir, "fetch", "--depth=500", url, commit); err != nil {
		// If that fails (e.g., short hash), try fetching default branch
		if _, err2 := runGit(dir, "fetch", "--depth=500", url); err2 != nil {
			return fmt.Errorf("git fetch %s at %s: %w", url, commit, err)
		}
	}

	if _, err := runGit(dir, "checkout", "FETCH_HEAD"); err != nil {
		return fmt.Errorf("git checkout FETCH_HEAD: %w", err)
	}

	return nil
}

// runGit runs a git command in the given directory and returns stdout.
func runGit(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}

// scanRepo runs the ARS pipeline on a directory with LLM disabled.
func scanRepo(dir string) (*GoldenFile, error) {
	cfg, err := scoring.LoadConfig("")
	if err != nil {
		return nil, fmt.Errorf("load scoring config: %w", err)
	}

	var buf bytes.Buffer
	p := pipeline.New(&buf, false, cfg, 0, true, nil)
	p.DisableLLM()

	if err := p.Run(dir); err != nil {
		return nil, fmt.Errorf("pipeline run: %w", err)
	}

	// Parse JSON output
	var report jsonReport
	if err := json.Unmarshal(buf.Bytes(), &report); err != nil {
		return nil, fmt.Errorf("parse pipeline JSON output: %w", err)
	}

	return &GoldenFile{
		CompositeScore: report.CompositeScore,
		Tier:           report.Tier,
		Categories:     extractCategories(report),
	}, nil
}

// jsonReport mirrors the pipeline's JSON output for unmarshalling.
type jsonReport struct {
	CompositeScore float64        `json:"composite_score"`
	Tier           string         `json:"tier"`
	Categories     []jsonCategory `json:"categories"`
}

type jsonCategory struct {
	Name  string  `json:"name"`
	Score float64 `json:"score"`
}

func extractCategories(r jsonReport) []GoldenCategory {
	cats := make([]GoldenCategory, 0, len(r.Categories))
	for _, c := range r.Categories {
		cats = append(cats, GoldenCategory{Name: c.Name, Score: c.Score})
	}
	return cats
}

// writeGolden writes a GoldenFile to disk, creating parent dirs as needed.
func writeGolden(path string, g *GoldenFile) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(g, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0644)
}

// compareGolden loads the golden file and compares it to the current scan.
func compareGolden(path string, current *GoldenFile) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("golden file missing (run with -update to generate): %w", err)
	}

	var golden GoldenFile
	if err := json.Unmarshal(data, &golden); err != nil {
		return fmt.Errorf("parse golden file: %w", err)
	}

	var errs []string

	// Compare composite score
	if math.Abs(current.CompositeScore-golden.CompositeScore) > scoreTolerance {
		errs = append(errs, fmt.Sprintf(
			"composite_score: got %.4f, want %.4f (diff %.4f > tolerance %.4f)",
			current.CompositeScore, golden.CompositeScore,
			math.Abs(current.CompositeScore-golden.CompositeScore), scoreTolerance,
		))
	}

	// Compare tier (exact)
	if current.Tier != golden.Tier {
		errs = append(errs, fmt.Sprintf("tier: got %q, want %q", current.Tier, golden.Tier))
	}

	// Compare category scores (bidirectional)
	goldenMap := make(map[string]float64, len(golden.Categories))
	for _, c := range golden.Categories {
		goldenMap[c.Name] = c.Score
	}
	currentMap := make(map[string]float64, len(current.Categories))
	for _, c := range current.Categories {
		currentMap[c.Name] = c.Score
	}
	for _, c := range current.Categories {
		want, ok := goldenMap[c.Name]
		if !ok {
			errs = append(errs, fmt.Sprintf("category %q: present in current but not in golden", c.Name))
			continue
		}
		if math.Abs(c.Score-want) > scoreTolerance {
			errs = append(errs, fmt.Sprintf(
				"category %s score: got %.4f, want %.4f (diff %.4f > tolerance %.4f)",
				c.Name, c.Score, want, math.Abs(c.Score-want), scoreTolerance,
			))
		}
	}
	for _, c := range golden.Categories {
		if _, ok := currentMap[c.Name]; !ok {
			errs = append(errs, fmt.Sprintf("category %q: present in golden but not in current", c.Name))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("golden mismatch:\n  %s", strings.Join(errs, "\n  "))
	}
	return nil
}
