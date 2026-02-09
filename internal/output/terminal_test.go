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
		{
			Name:     "C7: Agent Evaluation",
			Category: "C7",
			Metrics: map[string]interface{}{
				"c7": &types.C7Metrics{
					Available:              true,
					IntentClarity:          75,
					ModificationConfidence: 68,
					CrossFileCoherence:     82,
					SemanticCompleteness:   71,
					OverallScore:           74.0,
					TotalDuration:          45.5,
					CostUSD:                0.0125,
					TaskResults: []types.C7TaskResult{
						{TaskID: "intent_clarity", TaskName: "Intent Clarity", Score: 75, Status: "completed", Duration: 12.3, Reasoning: "Clear function signatures"},
						{TaskID: "modification_confidence", TaskName: "Modification Confidence", Score: 68, Status: "completed", Duration: 11.2, Reasoning: "Good test coverage"},
					},
				},
			},
		},
		{
			Name:     "C4: Documentation Quality",
			Category: "C4",
			Metrics: map[string]interface{}{
				"c4": &types.C4Metrics{
					Available:           true,
					ReadmePresent:       true,
					ReadmeWordCount:     450,
					CommentDensity:      12.5,
					APIDocCoverage:      65.0,
					ChangelogPresent:    true,
					ExamplesPresent:     true,
					ContributingPresent: false,
					DiagramsPresent:     false,
					LLMEnabled:          false,
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
		"Files discovered: 5",
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

	// C7 metrics should appear
	c7Checks := []string{
		"C7: Agent Evaluation",
		"Intent clarity:",
		"Modification conf:",
		"Cross-file coherence:",
		"Semantic complete:",
		"Overall score:",
		"Duration:",
		"Estimated cost:",
	}
	for _, check := range c7Checks {
		if !strings.Contains(out, check) {
			t.Errorf("output missing C7 metric %q\nGot:\n%s", check, out)
		}
	}

	// C4 metrics should appear
	c4Checks := []string{
		"C4: Documentation Quality",
		"README:",
		"Comment density:",
		"API doc coverage:",
		"CHANGELOG:",
		"Examples:",
		"CONTRIBUTING:",
		"Diagrams:",
		"LLM Analysis:",
		"n/a",
	}
	for _, check := range c4Checks {
		if !strings.Contains(out, check) {
			t.Errorf("output missing C4 metric %q\nGot:\n%s", check, out)
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
		// C7 verbose: per-task results
		"Per-task results:",
		"Intent Clarity:",
		"score=75",
		"completed",
		"Clear function signatures",
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

func TestRenderC7Unavailable(t *testing.T) {
	var buf bytes.Buffer
	ar := &types.AnalysisResult{
		Name:     "C7: Agent Evaluation",
		Category: "C7",
		Metrics: map[string]interface{}{
			"c7": &types.C7Metrics{
				Available: false,
			},
		},
	}
	RenderSummary(&buf, newTestResult(), []*types.AnalysisResult{ar}, false)
	out := buf.String()

	if !strings.Contains(out, "C7: Agent Evaluation") {
		t.Error("output should contain C7 header")
	}
	if !strings.Contains(out, "Not available") {
		t.Error("output should indicate C7 not available")
	}
	// Should NOT contain metric values when unavailable
	if strings.Contains(out, "Intent clarity:") {
		t.Error("unavailable C7 should not show metric details")
	}
}

func TestRenderC4WithLLM(t *testing.T) {
	var buf bytes.Buffer
	ar := &types.AnalysisResult{
		Name:     "C4: Documentation Quality",
		Category: "C4",
		Metrics: map[string]interface{}{
			"c4": &types.C4Metrics{
				Available:           true,
				ReadmePresent:       true,
				ReadmeWordCount:     500,
				CommentDensity:      15.0,
				APIDocCoverage:      70.0,
				ChangelogPresent:    true,
				ExamplesPresent:     true,
				ContributingPresent: true,
				DiagramsPresent:     true,
				LLMEnabled:          true,
				ReadmeClarity:       8,
				ExampleQuality:      7,
				Completeness:        6,
				CrossRefCoherence:   7,
				LLMCostUSD:          0.0015,
				LLMTokensUsed:       5000,
			},
		},
	}
	RenderSummary(&buf, newTestResult(), []*types.AnalysisResult{ar}, false)
	out := buf.String()

	// LLM metrics should show actual values
	llmChecks := []string{
		"README clarity:",
		"8/10",
		"Example quality:",
		"7/10",
		"Completeness:",
		"6/10",
		"Cross-ref coherence:",
		"LLM cost:",
	}
	for _, check := range llmChecks {
		if !strings.Contains(out, check) {
			t.Errorf("output missing LLM metric %q\nGot:\n%s", check, out)
		}
	}

	// Should NOT contain n/a when LLM enabled
	if strings.Contains(out, "n/a (Claude CLI not detected)") {
		t.Error("LLM-enabled C4 should not show n/a")
	}
}

func TestRenderC4Unavailable(t *testing.T) {
	var buf bytes.Buffer
	ar := &types.AnalysisResult{
		Name:     "C4: Documentation Quality",
		Category: "C4",
		Metrics: map[string]interface{}{
			"c4": &types.C4Metrics{
				Available: false,
			},
		},
	}
	RenderSummary(&buf, newTestResult(), []*types.AnalysisResult{ar}, false)
	out := buf.String()

	if !strings.Contains(out, "C4: Documentation Quality") {
		t.Error("output should contain C4 header")
	}
	if !strings.Contains(out, "Not available") {
		t.Error("output should indicate C4 not available")
	}
	if strings.Contains(out, "README:") {
		t.Error("unavailable C4 should not show metric details")
	}
}

func TestRenderC7Debug(t *testing.T) {
	var buf bytes.Buffer

	results := []*types.AnalysisResult{
		{
			Name:     "C7: Agent Evaluation",
			Category: "C7",
			Metrics: map[string]interface{}{
				"c7": &types.C7Metrics{
					Available: true,
					MetricResults: []types.C7MetricResult{
						{
							MetricID:   "code_behavior_comprehension",
							MetricName: "Code Behavior Comprehension",
							Score:      7,
							Status:     "completed",
							Duration:   12.5,
							DebugSamples: []types.C7DebugSample{
								{
									FilePath:    "test.go",
									Description: "test sample",
									Prompt:      "Explain this code that processes input data",
									Response:    "The code implements a data processing pipeline",
									Score:       7,
									Duration:    12.5,
									ScoreTrace: types.C7ScoreTrace{
										BaseScore: 2,
										Indicators: []types.C7IndicatorMatch{
											{Name: "positive:returns", Matched: true, Delta: 2},
											{Name: "negative:unclear", Matched: false, Delta: 0},
										},
										FinalScore: 7,
									},
								},
							},
						},
					},
				},
			},
		},
	}

	RenderC7Debug(&buf, results)
	out := buf.String()

	checks := []string{
		"C7 Debug: Agent Evaluation Details",
		"code_behavior_comprehension",
		"score=7/10",
		"Sample 1: test sample",
		"File:     test.go",
		"base=2",
		"final=7",
		"positive:returns",
		"Explain this code",
		"The code implements",
	}
	for _, check := range checks {
		if !strings.Contains(out, check) {
			t.Errorf("output missing %q\nGot:\n%s", check, out)
		}
	}
}

func TestRenderC7Debug_NoDebugSamples(t *testing.T) {
	var buf bytes.Buffer

	results := []*types.AnalysisResult{
		{
			Name:     "C7: Agent Evaluation",
			Category: "C7",
			Metrics: map[string]interface{}{
				"c7": &types.C7Metrics{
					Available: true,
					MetricResults: []types.C7MetricResult{
						{
							MetricID:     "code_behavior_comprehension",
							MetricName:   "Code Behavior Comprehension",
							Score:        5,
							Duration:     8.0,
							DebugSamples: nil, // no debug samples
						},
					},
				},
			},
		},
	}

	// Should not panic
	RenderC7Debug(&buf, results)
	out := buf.String()

	// Should still render the metric header
	if !strings.Contains(out, "code_behavior_comprehension") {
		t.Errorf("output should contain metric header\nGot:\n%s", out)
	}
	if !strings.Contains(out, "No debug samples captured") {
		t.Errorf("output should indicate no debug samples\nGot:\n%s", out)
	}
}

func TestRenderC7Debug_NoC7Result(t *testing.T) {
	var buf bytes.Buffer

	// No C7 in the results at all
	results := []*types.AnalysisResult{
		{
			Name:     "C1: Code Health",
			Category: "C1",
			Metrics:  map[string]interface{}{},
		},
	}

	// Should not panic
	RenderC7Debug(&buf, results)
	out := buf.String()

	// Should produce no output
	if out != "" {
		t.Errorf("expected empty output when no C7 result, got:\n%s", out)
	}
}

func TestRenderC2(t *testing.T) {
	var buf bytes.Buffer
	ar := &types.AnalysisResult{
		Name:     "C2: Semantic Explicitness",
		Category: "C2",
		Metrics: map[string]interface{}{
			"c2": &types.C2Metrics{
				Aggregate: &types.C2LanguageMetrics{
					TypeAnnotationCoverage: 75.0,
					NamingConsistency:      85.0,
					MagicNumberRatio:       3.5,
					TypeStrictness:         1,
					NullSafety:             60.0,
				},
			},
		},
	}

	renderC2(&buf, ar, false)
	out := buf.String()

	checks := []string{
		"C2: Semantic Explicitness",
		"Type annotation:",
		"75.0%",
		"Naming consistency:",
		"85.0%",
		"Magic numbers:",
		"3.5 per kLOC",
		"Type strictness:",
		"on",
		"Null safety:",
		"60%",
	}

	for _, check := range checks {
		if !strings.Contains(out, check) {
			t.Errorf("output missing %q\nGot:\n%s", check, out)
		}
	}
}

func TestRenderC2_TypeStrictnessOff(t *testing.T) {
	var buf bytes.Buffer
	ar := &types.AnalysisResult{
		Name:     "C2: Semantic Explicitness",
		Category: "C2",
		Metrics: map[string]interface{}{
			"c2": &types.C2Metrics{
				Aggregate: &types.C2LanguageMetrics{
					TypeAnnotationCoverage: 50.0,
					NamingConsistency:      70.0,
					MagicNumberRatio:       10.0,
					TypeStrictness:         0, // off
					NullSafety:             30.0,
				},
			},
		},
	}

	renderC2(&buf, ar, false)
	out := buf.String()

	if !strings.Contains(out, "Type strictness:     off") {
		t.Errorf("output should show type strictness off\nGot:\n%s", out)
	}
}

func TestRenderC2_Verbose(t *testing.T) {
	var buf bytes.Buffer
	ar := &types.AnalysisResult{
		Name:     "C2: Semantic Explicitness",
		Category: "C2",
		Metrics: map[string]interface{}{
			"c2": &types.C2Metrics{
				Aggregate: &types.C2LanguageMetrics{
					TypeAnnotationCoverage: 75.0,
					NamingConsistency:      85.0,
					MagicNumberRatio:       3.5,
					TypeStrictness:         1,
					NullSafety:             60.0,
				},
				PerLanguage: map[types.Language]*types.C2LanguageMetrics{
					types.LangGo: {
						TypeAnnotationCoverage: 80,
						NamingConsistency:      90,
						MagicNumberRatio:       2.0,
						TypeStrictness:         1,
						NullSafety:             70,
						LOC:                    1000,
					},
				},
			},
		},
	}

	renderC2(&buf, ar, true)
	out := buf.String()

	if !strings.Contains(out, "Per-language C2 breakdown:") {
		t.Error("verbose output should show per-language breakdown")
	}
	if !strings.Contains(out, "go:") {
		t.Error("verbose output should show Go language metrics")
	}
	if !strings.Contains(out, "LOC=1000") {
		t.Error("verbose output should show LOC")
	}
}

func TestRenderC5(t *testing.T) {
	var buf bytes.Buffer
	ar := &types.AnalysisResult{
		Name:     "C5: Temporal Dynamics",
		Category: "C5",
		Metrics: map[string]interface{}{
			"c5": &types.C5Metrics{
				Available:            true,
				TotalCommits:         100,
				TimeWindowDays:       90,
				ChurnRate:            150.5,
				TemporalCouplingPct:  25.0,
				AuthorFragmentation:  3.2,
				CommitStability:      5.5,
				HotspotConcentration: 60.0,
			},
		},
	}

	renderC5(&buf, ar, false)
	out := buf.String()

	checks := []string{
		"C5: Temporal Dynamics",
		"Total commits:",
		"100",
		"90-day window",
		"Churn rate:",
		"150.5",
		"Temporal coupling:",
		"25.0%",
		"Author fragmentation:",
		"3.2",
		"Commit stability:",
		"5.5 days",
		"Hotspot concentration:",
		"60.0%",
	}

	for _, check := range checks {
		if !strings.Contains(out, check) {
			t.Errorf("output missing %q\nGot:\n%s", check, out)
		}
	}
}

func TestRenderC5_Unavailable(t *testing.T) {
	var buf bytes.Buffer
	ar := &types.AnalysisResult{
		Name:     "C5: Temporal Dynamics",
		Category: "C5",
		Metrics: map[string]interface{}{
			"c5": &types.C5Metrics{
				Available: false,
			},
		},
	}

	renderC5(&buf, ar, false)
	out := buf.String()

	if !strings.Contains(out, "Not available (no .git directory)") {
		t.Errorf("output should indicate unavailable\nGot:\n%s", out)
	}
}

func TestRenderC5_Verbose(t *testing.T) {
	var buf bytes.Buffer
	ar := &types.AnalysisResult{
		Name:     "C5: Temporal Dynamics",
		Category: "C5",
		Metrics: map[string]interface{}{
			"c5": &types.C5Metrics{
				Available:      true,
				TotalCommits:   50,
				TimeWindowDays: 90,
				TopHotspots: []types.FileChurn{
					{Path: "main.go", TotalChanges: 25, CommitCount: 20, AuthorCount: 3},
				},
				CoupledPairs: []types.CoupledPair{
					{FileA: "config.go", FileB: "handler.go", Coupling: 85.0, SharedCommits: 17},
				},
			},
		},
	}

	renderC5(&buf, ar, true)
	out := buf.String()

	if !strings.Contains(out, "Top hotspots:") {
		t.Error("verbose output should show top hotspots")
	}
	if !strings.Contains(out, "main.go") {
		t.Error("verbose output should show hotspot file")
	}
	if !strings.Contains(out, "Coupled pairs") {
		t.Error("verbose output should show coupled pairs")
	}
	if !strings.Contains(out, "config.go <-> handler.go") {
		t.Error("verbose output should show coupled pair details")
	}
}

func TestRenderScores(t *testing.T) {
	var buf bytes.Buffer
	scored := &types.ScoredResult{
		Composite: 7.5,
		Tier:      "Agent-Assisted",
		Categories: []types.CategoryScore{
			{Name: "C1", Score: 8.0, Weight: 0.25},
			{Name: "C3", Score: 7.0, Weight: 0.20},
			{Name: "C6", Score: 6.5, Weight: 0.15},
		},
	}

	RenderScores(&buf, scored, false)
	out := buf.String()

	checks := []string{
		"Agent Readiness Score",
		"C1:",
		"8.0 / 10",
		"C3:",
		"7.0 / 10",
		"C6:",
		"6.5 / 10",
		"Composite Score:",
		"7.5 / 10",
		"Rating:",
		"Agent-Assisted",
	}

	for _, check := range checks {
		if !strings.Contains(out, check) {
			t.Errorf("output missing %q\nGot:\n%s", check, out)
		}
	}
}

func TestRenderScores_UnavailableCategory(t *testing.T) {
	var buf bytes.Buffer
	scored := &types.ScoredResult{
		Composite: 7.0,
		Tier:      "Agent-Assisted",
		Categories: []types.CategoryScore{
			{Name: "C1", Score: 8.0, Weight: 0.25},
			{Name: "C5", Score: -1, Weight: 0.10}, // unavailable
		},
	}

	RenderScores(&buf, scored, false)
	out := buf.String()

	if !strings.Contains(out, "C5:") {
		t.Error("output should contain C5 category")
	}
	if !strings.Contains(out, "n/a") {
		t.Error("unavailable category should show n/a")
	}
}

func TestRenderScores_Verbose(t *testing.T) {
	var buf bytes.Buffer
	scored := &types.ScoredResult{
		Composite: 7.5,
		Tier:      "Agent-Assisted",
		Categories: []types.CategoryScore{
			{
				Name:   "C1",
				Score:  8.0,
				Weight: 0.25,
				SubScores: []types.SubScore{
					{MetricName: "complexity_avg", RawValue: 5.5, Score: 8.0, Weight: 0.3, Available: true},
					{MetricName: "function_length_avg", RawValue: 20.0, Score: 7.5, Weight: 0.25, Available: true},
				},
			},
		},
	}

	RenderScores(&buf, scored, true)
	out := buf.String()

	// Should show subscores in verbose mode (check for weight percentage)
	if !strings.Contains(out, "30%") {
		t.Errorf("verbose mode should show subscore weight\nGot:\n%s", out)
	}
	if !strings.Contains(out, "25%") {
		t.Errorf("verbose mode should show subscore weight\nGot:\n%s", out)
	}
}

func TestRenderSubScores(t *testing.T) {
	var buf bytes.Buffer
	subScores := []types.SubScore{
		{MetricName: "complexity_avg", RawValue: 5.5, Score: 8.0, Weight: 0.3, Available: true},
		{MetricName: "function_length_avg", RawValue: 20.0, Score: 7.5, Weight: 0.25, Available: true},
		{MetricName: "deprecated_metric", RawValue: 0, Score: 0, Weight: 0.0, Available: true}, // should be filtered
	}

	renderSubScores(&buf, subScores)
	out := buf.String()

	// Metric names may be displayed through metricDisplayNames map, so check for values instead
	if out == "" {
		t.Error("output should not be empty")
	}
	if !strings.Contains(out, "5.5") {
		t.Errorf("output should contain raw value\nGot:\n%s", out)
	}
	if !strings.Contains(out, "8.0") {
		t.Errorf("output should contain score\nGot:\n%s", out)
	}
	if !strings.Contains(out, "30%") {
		t.Errorf("output should contain weight percentage\nGot:\n%s", out)
	}
}

func TestRenderSubScores_Unavailable(t *testing.T) {
	var buf bytes.Buffer
	subScores := []types.SubScore{
		{MetricName: "coverage_percent", RawValue: 0, Score: 0, Weight: 0.4, Available: false},
	}

	renderSubScores(&buf, subScores)
	out := buf.String()

	if out == "" {
		t.Error("output should not be empty")
	}
	if !strings.Contains(out, "n/a") {
		t.Errorf("unavailable metric should show n/a\nGot:\n%s", out)
	}
	if !strings.Contains(out, "40%") {
		t.Errorf("output should show weight\nGot:\n%s", out)
	}
	if !strings.Contains(out, "excluded") {
		t.Errorf("unavailable metric should indicate excluded\nGot:\n%s", out)
	}
}

func TestTierColor(t *testing.T) {
	tests := []struct {
		tier string
		want string // We can't test exact colors, but verify it returns a color
	}{
		{"Agent-Ready", "green"},
		{"Agent-Assisted", "yellow"},
		{"Agent-Limited", "red"},
		{"Agent-Hostile", "red"},
	}

	for _, tt := range tests {
		t.Run(tt.tier, func(t *testing.T) {
			c := tierColor(tt.tier)
			if c == nil {
				t.Error("tierColor should not return nil")
			}
		})
	}
}

func TestJoinCycle(t *testing.T) {
	tests := []struct {
		name  string
		cycle []string
		want  string
	}{
		{
			name:  "simple cycle",
			cycle: []string{"A", "B", "C"},
			want:  "A -> B -> C -> A",
		},
		{
			name:  "two node cycle",
			cycle: []string{"pkg/a", "pkg/b"},
			want:  "pkg/a -> pkg/b -> pkg/a",
		},
		{
			name:  "empty cycle",
			cycle: []string{},
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := joinCycle(tt.cycle)
			if got != tt.want {
				t.Errorf("joinCycle(%v) = %q, want %q", tt.cycle, got, tt.want)
			}
		})
	}
}

func TestRenderBadge(t *testing.T) {
	var buf bytes.Buffer
	scored := &types.ScoredResult{
		Composite: 8.5,
		Tier:      "Agent-Ready",
	}

	RenderBadge(&buf, scored)
	out := buf.String()

	if !strings.Contains(out, "Badge") {
		t.Error("output should contain Badge header")
	}
	// Output should contain markdown badge (exact format depends on GenerateBadge)
	if out == "" {
		t.Error("output should not be empty")
	}
}

func TestRenderBadge_Nil(t *testing.T) {
	var buf bytes.Buffer

	// Should not panic with nil input
	RenderBadge(&buf, nil)
	out := buf.String()

	// Should produce no output
	if out != "" {
		t.Errorf("expected empty output for nil scored result, got:\n%s", out)
	}
}
