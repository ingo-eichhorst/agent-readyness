package output

import (
	"bytes"
	"strings"
	"testing"

	"github.com/ingo/agent-readyness/internal/recommend"
	"github.com/ingo/agent-readyness/pkg/types"
)

func newTestResult() *types.ScanResult {
	return &types.ScanResult{
		RootDir:        "/test/project",
		TotalFiles:     5,
		SourceCount:    2,
		TestCount:      1,
		GeneratedCount: 1,
		VendorCount:    1,
		GitignoreCount: 0,
		Files: []types.DiscoveredFile{
			{Path: "/test/project/main.go", RelPath: "main.go", Class: types.ClassSource},
			{Path: "/test/project/util.go", RelPath: "util.go", Class: types.ClassSource},
			{Path: "/test/project/main_test.go", RelPath: "main_test.go", Class: types.ClassTest},
			{Path: "/test/project/gen.go", RelPath: "gen.go", Class: types.ClassGenerated},
			{Path: "/test/project/vendor/dep.go", RelPath: "vendor/dep.go", Class: types.ClassExcluded, ExcludeReason: "vendor"},
		},
	}
}

func newTestAnalysisResults() []*types.AnalysisResult {
	return []*types.AnalysisResult{
		{
			Name:     "C1: Code Health",
			Category: "C1",
			Metrics: map[string]interface{}{
				"c1": &types.C1Metrics{
					CyclomaticComplexity: types.MetricSummary{Avg: 3.5, Max: 8, MaxEntity: "main.handleRequest"},
					FunctionLength:       types.MetricSummary{Avg: 15.2, Max: 42, MaxEntity: "main.handleRequest"},
					FileSize:             types.MetricSummary{Avg: 120, Max: 250, MaxEntity: "main.go"},
					DuplicationRate:      2.5,
					Functions: []types.FunctionMetric{
						{Package: "main", Name: "handleRequest", File: "main.go", Line: 10, Complexity: 8, LineCount: 42},
						{Package: "main", Name: "init", File: "main.go", Line: 5, Complexity: 1, LineCount: 3},
					},
				},
			},
		},
		{
			Name:     "C3: Architecture",
			Category: "C3",
			Metrics: map[string]interface{}{
				"c3": &types.C3Metrics{
					MaxDirectoryDepth: 3,
					AvgDirectoryDepth: 1.5,
					ModuleFanout:      types.MetricSummary{Avg: 2.0, Max: 4},
					CircularDeps:      nil,
					DeadExports: []types.DeadExport{
						{Package: "util", Name: "Unused", File: "util.go", Line: 15, Kind: "func"},
					},
				},
			},
		},
		{
			Name:     "C6: Testing",
			Category: "C6",
			Metrics: map[string]interface{}{
				"c6": &types.C6Metrics{
					TestFileCount:   1,
					SourceFileCount: 2,
					TestToCodeRatio: 0.45,
					CoveragePercent: -1,
					CoverageSource:  "none",
					TestIsolation:   100,
					AssertionDensity: types.MetricSummary{Avg: 2.0, Max: 3},
					TestFunctions: []types.TestFunctionMetric{
						{Package: "main", Name: "TestHandle", File: "main_test.go", Line: 5, AssertionCount: 3, HasExternalDep: false},
					},
				},
			},
		},
	}
}

func TestRenderSummary(t *testing.T) {
	var buf bytes.Buffer
	result := newTestResult()

	RenderSummary(&buf, result, nil, false)
	out := buf.String()

	checks := []string{
		"ARS Scan: /test/project",
		"Go files discovered: 5",
		"Source files:",
		"Test files:",
		"Generated (excluded):",
		"Vendor (excluded):",
	}

	for _, check := range checks {
		if !strings.Contains(out, check) {
			t.Errorf("output missing %q\nGot:\n%s", check, out)
		}
	}

	// Gitignored should NOT appear (count is 0)
	if strings.Contains(out, "Gitignored") {
		t.Error("output should not contain 'Gitignored' when count is 0")
	}

	// Verbose content should NOT appear
	if strings.Contains(out, "Discovered files:") {
		t.Error("non-verbose output should not contain 'Discovered files:'")
	}
}

func TestRenderSummaryVerbose(t *testing.T) {
	var buf bytes.Buffer
	result := newTestResult()

	RenderSummary(&buf, result, nil, true)
	out := buf.String()

	// Should have file listing
	if !strings.Contains(out, "Discovered files:") {
		t.Error("verbose output missing 'Discovered files:' header")
	}

	// Individual files should appear
	fileChecks := []string{
		"[source] main.go",
		"[test] main_test.go",
		"[generated] gen.go",
		"[excluded] vendor/dep.go (vendor)",
	}

	for _, check := range fileChecks {
		if !strings.Contains(out, check) {
			t.Errorf("verbose output missing %q\nGot:\n%s", check, out)
		}
	}
}

func TestRenderSummaryWithMetrics(t *testing.T) {
	var buf bytes.Buffer
	result := newTestResult()
	analysisResults := newTestAnalysisResults()

	RenderSummary(&buf, result, analysisResults, false)
	out := buf.String()

	// C1 metrics should appear
	c1Checks := []string{
		"C1: Code Health",
		"Complexity avg:",
		"Complexity max:",
		"Func length avg:",
		"Func length max:",
		"File size avg:",
		"File size max:",
		"Duplication rate:",
	}
	for _, check := range c1Checks {
		if !strings.Contains(out, check) {
			t.Errorf("output missing C1 metric %q\nGot:\n%s", check, out)
		}
	}

	// C3 metrics should appear
	c3Checks := []string{
		"C3: Architecture",
		"Max directory depth:",
		"Avg directory depth:",
		"Avg module fanout:",
		"Circular deps:",
		"Dead exports:",
	}
	for _, check := range c3Checks {
		if !strings.Contains(out, check) {
			t.Errorf("output missing C3 metric %q\nGot:\n%s", check, out)
		}
	}

	// C6 metrics should appear
	c6Checks := []string{
		"C6: Testing",
		"Test-to-code ratio:",
		"Coverage:",
		"Test isolation:",
		"Assertion density:",
	}
	for _, check := range c6Checks {
		if !strings.Contains(out, check) {
			t.Errorf("output missing C6 metric %q\nGot:\n%s", check, out)
		}
	}

	// Non-verbose: should NOT show top functions or dead export details
	if strings.Contains(out, "Top complex functions:") {
		t.Error("non-verbose output should not contain 'Top complex functions:'")
	}
	if strings.Contains(out, "Test functions:") {
		t.Error("non-verbose output should not contain 'Test functions:'")
	}
}

func TestRenderSummaryWithMetricsVerbose(t *testing.T) {
	var buf bytes.Buffer
	result := newTestResult()
	analysisResults := newTestAnalysisResults()

	RenderSummary(&buf, result, analysisResults, true)
	out := buf.String()

	// Verbose: should show top functions
	verboseChecks := []string{
		"Top complex functions:",
		"Top longest functions:",
		"Dead exports:",
		"Test functions:",
		"main.handleRequest",
		"util.Unused",
		"main.TestHandle",
	}
	for _, check := range verboseChecks {
		if !strings.Contains(out, check) {
			t.Errorf("verbose output missing %q\nGot:\n%s", check, out)
		}
	}
}

func TestRenderRecommendations(t *testing.T) {
	recs := []recommend.Recommendation{
		{
			Rank:             1,
			Category:         "C6",
			MetricName:       "coverage_percent",
			CurrentValue:     30,
			CurrentScore:     3.0,
			TargetValue:      50,
			ScoreImprovement: 0.8,
			Effort:           "Medium",
			Summary:          "Improve test coverage from 30.0 to 50.0 -- Without test coverage data, agents cannot assess change safety",
			Action:           "Increase test coverage from 30% to 50%",
		},
		{
			Rank:             2,
			Category:         "C1",
			MetricName:       "complexity_avg",
			CurrentValue:     18,
			CurrentScore:     4.5,
			TargetValue:      10,
			ScoreImprovement: 0.3,
			Effort:           "High",
			Summary:          "Improve average complexity from 18.0 to 10.0 -- High complexity makes functions harder for agents",
			Action:           "Refactor functions with cyclomatic complexity > 18",
		},
	}

	var buf bytes.Buffer
	RenderRecommendations(&buf, recs)
	out := buf.String()

	checks := []string{
		"Top Recommendations",
		"1.",
		"2.",
		"Improve test coverage",
		"Impact: +0.8 points",
		"Effort: Medium",
		"Effort: High",
		"Increase test coverage",
		"Refactor functions",
		"Impact: +0.3 points",
	}

	for _, check := range checks {
		if !strings.Contains(out, check) {
			t.Errorf("output missing %q\nGot:\n%s", check, out)
		}
	}
}

func TestRenderRecommendationsEmpty(t *testing.T) {
	var buf bytes.Buffer
	RenderRecommendations(&buf, nil)
	out := buf.String()

	if !strings.Contains(out, "No recommendations") {
		t.Errorf("empty recommendations should show excellent message\nGot:\n%s", out)
	}
	if !strings.Contains(out, "excellent") {
		t.Errorf("empty recommendations should contain 'excellent'\nGot:\n%s", out)
	}
}
