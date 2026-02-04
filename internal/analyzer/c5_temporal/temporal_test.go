package c5

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ingo/agent-readyness/pkg/types"
)

// findProjectRoot walks up from cwd to locate the .git directory.
func findProjectRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Skip("no .git directory found in parent chain")
		}
		dir = parent
	}
}

// makeTarget creates an AnalysisTarget for the given root directory.
func makeTarget(root string) []*types.AnalysisTarget {
	return []*types.AnalysisTarget{
		{RootDir: root, Language: types.LangGo},
	}
}

func TestC5Analyzer_Name(t *testing.T) {
	a := NewC5Analyzer()
	if got := a.Name(); got != "C5: Temporal Dynamics" {
		t.Errorf("Name() = %q, want %q", got, "C5: Temporal Dynamics")
	}
}

func TestC5Analyzer_EmptyTargets(t *testing.T) {
	a := NewC5Analyzer()

	_, err := a.Analyze(nil)
	if err == nil {
		t.Error("Analyze(nil) should return error")
	}

	_, err = a.Analyze([]*types.AnalysisTarget{})
	if err == nil {
		t.Error("Analyze([]) should return error")
	}
}

func TestC5Analyzer_NoGitDir(t *testing.T) {
	dir := t.TempDir() // no .git inside
	a := NewC5Analyzer()
	result, err := a.Analyze(makeTarget(dir))
	if err != nil {
		t.Fatalf("Analyze on non-git dir returned error: %v", err)
	}

	c5, ok := result.Metrics["c5"].(*types.C5Metrics)
	if !ok {
		t.Fatal("expected *types.C5Metrics in Metrics[\"c5\"]")
	}
	if c5.Available {
		t.Error("Available should be false for non-git directory")
	}
}

func TestC5Analyzer_Category(t *testing.T) {
	root := findProjectRoot(t)
	a := NewC5Analyzer()
	result, err := a.Analyze(makeTarget(root))
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}

	if result.Category != "C5" {
		t.Errorf("Category = %q, want %q", result.Category, "C5")
	}
	if result.Name != "C5: Temporal Dynamics" {
		t.Errorf("Name = %q, want %q", result.Name, "C5: Temporal Dynamics")
	}

	_, ok := result.Metrics["c5"].(*types.C5Metrics)
	if !ok {
		t.Fatal("expected *types.C5Metrics in Metrics[\"c5\"]")
	}
}

func TestC5Analyzer_RealRepo(t *testing.T) {
	root := findProjectRoot(t)
	a := NewC5Analyzer()
	result, err := a.Analyze(makeTarget(root))
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}

	c5, ok := result.Metrics["c5"].(*types.C5Metrics)
	if !ok {
		t.Fatal("expected *types.C5Metrics in Metrics[\"c5\"]")
	}

	if !c5.Available {
		t.Fatal("Available should be true for real git repo")
	}
	if c5.TotalCommits <= 0 {
		t.Errorf("TotalCommits = %d, want > 0", c5.TotalCommits)
	}
	if c5.ChurnRate <= 0 {
		t.Errorf("ChurnRate = %f, want > 0", c5.ChurnRate)
	}
	if c5.HotspotConcentration <= 0 {
		t.Errorf("HotspotConcentration = %f, want > 0", c5.HotspotConcentration)
	}
	if c5.TimeWindowDays <= 0 {
		t.Errorf("TimeWindowDays = %d, want > 0", c5.TimeWindowDays)
	}
	if len(c5.TopHotspots) == 0 {
		t.Error("TopHotspots should not be empty for real repo")
	}
}

func TestC5Analyzer_RealRepo_MetricRanges(t *testing.T) {
	root := findProjectRoot(t)
	a := NewC5Analyzer()
	result, err := a.Analyze(makeTarget(root))
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}

	c5 := result.Metrics["c5"].(*types.C5Metrics)

	// ChurnRate: lines per commit, should be reasonable (not negative, not millions)
	if c5.ChurnRate < 0 || c5.ChurnRate > 100000 {
		t.Errorf("ChurnRate = %f, expected 0-100000", c5.ChurnRate)
	}

	// TemporalCouplingPct: 0-100 percentage
	if c5.TemporalCouplingPct < 0 || c5.TemporalCouplingPct > 100 {
		t.Errorf("TemporalCouplingPct = %f, expected 0-100", c5.TemporalCouplingPct)
	}

	// AuthorFragmentation: >= 0 (avg authors per file)
	if c5.AuthorFragmentation < 0 {
		t.Errorf("AuthorFragmentation = %f, expected >= 0", c5.AuthorFragmentation)
	}

	// CommitStability: median days between changes, >= 0
	if c5.CommitStability < 0 {
		t.Errorf("CommitStability = %f, expected >= 0", c5.CommitStability)
	}

	// HotspotConcentration: 0-100 percentage
	if c5.HotspotConcentration < 0 || c5.HotspotConcentration > 100 {
		t.Errorf("HotspotConcentration = %f, expected 0-100", c5.HotspotConcentration)
	}
}

func TestResolveRenamePath(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"src/file.go", "src/file.go"},
		{"old.go => new.go", "new.go"},
		{"src/{old.go => new.go}", "src/new.go"},
		{"src/{subdir => other}/file.go", "src/other/file.go"},
		{"{old => new}/file.go", "new/file.go"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := resolveRenamePath(tc.input)
			if got != tc.want {
				t.Errorf("resolveRenamePath(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestUniquePaths(t *testing.T) {
	files := []fileChange{
		{Path: "b.go"},
		{Path: "a.go"},
		{Path: "b.go"},
		{Path: "c.go"},
	}
	got := uniquePaths(files)
	want := []string{"a.go", "b.go", "c.go"}
	if len(got) != len(want) {
		t.Fatalf("uniquePaths length = %d, want %d", len(got), len(want))
	}
	for i, g := range got {
		if g != want[i] {
			t.Errorf("uniquePaths[%d] = %q, want %q", i, g, want[i])
		}
	}
}

func TestSortedPair(t *testing.T) {
	p1 := sortedPair("b", "a")
	if p1 != [2]string{"a", "b"} {
		t.Errorf("sortedPair(b,a) = %v, want [a b]", p1)
	}
	p2 := sortedPair("a", "b")
	if p2 != [2]string{"a", "b"} {
		t.Errorf("sortedPair(a,b) = %v, want [a b]", p2)
	}
}
