package metrics

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/ingo/agent-readyness/pkg/types"
)

// loadFixture reads a test fixture file from testdata/c7_responses/{subdir}/{name}.
func loadFixture(t *testing.T, subdir, name string) string {
	t.Helper()
	_, thisFile, _, _ := runtime.Caller(0)
	dir := filepath.Dir(thisFile)
	path := filepath.Join(dir, "testdata", "c7_responses", subdir, name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("loadFixture(%s/%s): %v", subdir, name, err)
	}
	return string(data)
}

// mockExecutor implements the Executor interface returning canned responses.
type mockExecutor struct {
	response string
}

func (m *mockExecutor) ExecutePrompt(ctx context.Context, workDir, prompt, tools string, timeout time.Duration) (string, error) {
	return m.response, nil
}

func TestAllMetricsReturns5(t *testing.T) {
	metrics := AllMetrics()
	if len(metrics) != 5 {
		t.Errorf("AllMetrics() returned %d metrics, want 5", len(metrics))
	}
}

func TestMetricIDsAreUnique(t *testing.T) {
	metrics := AllMetrics()
	seen := make(map[string]bool)
	for _, m := range metrics {
		if seen[m.ID()] {
			t.Errorf("duplicate metric ID: %s", m.ID())
		}
		seen[m.ID()] = true
	}
}

func TestMetricNamesAreUnique(t *testing.T) {
	metrics := AllMetrics()
	seen := make(map[string]bool)
	for _, m := range metrics {
		if seen[m.Name()] {
			t.Errorf("duplicate metric Name: %s", m.Name())
		}
		seen[m.Name()] = true
	}
}

func TestGetMetricByID(t *testing.T) {
	tests := []struct {
		id   string
		want string
	}{
		{"task_execution_consistency", "Task Execution Consistency"},
		{"code_behavior_comprehension", "Code Behavior Comprehension"},
		{"cross_file_navigation", "Cross-File Navigation"},
		{"identifier_interpretability", "Identifier Interpretability"},
		{"documentation_accuracy_detection", "Documentation Accuracy Detection"},
	}

	for _, tc := range tests {
		m := getMetric(tc.id)
		if m == nil {
			t.Errorf("getMetric(%q) returned nil", tc.id)
			continue
		}
		if m.Name() != tc.want {
			t.Errorf("getMetric(%q).Name() = %q, want %q", tc.id, m.Name(), tc.want)
		}
	}
}

func TestGetMetricUnknown(t *testing.T) {
	m := getMetric("unknown_metric")
	if m != nil {
		t.Errorf("getMetric(unknown) = %v, want nil", m)
	}
}

func TestM1Consistency_SelectSamples(t *testing.T) {
	m := newM1Consistency()

	// Empty targets should return empty samples
	samples := m.SelectSamples(nil)
	if len(samples) > m.SampleCount() {
		t.Errorf("SelectSamples returned %d samples, max should be %d", len(samples), m.SampleCount())
	}
}

func TestM2Comprehension_SelectSamples(t *testing.T) {
	m := newM2Comprehension()

	// With targets
	targets := []*types.AnalysisTarget{
		{
			RootDir:  "/test",
			Language: "go",
			Files: []types.SourceFile{
				{Path: "/test/main.go", Lines: 100, Class: types.ClassSource},
				{Path: "/test/util.go", Lines: 50, Class: types.ClassSource},
			},
		},
	}

	samples := m.SelectSamples(targets)
	if len(samples) > m.SampleCount() {
		t.Errorf("SelectSamples returned %d samples, max should be %d", len(samples), m.SampleCount())
	}
}

func TestM3Navigation_SelectSamples(t *testing.T) {
	m := newM3Navigation()

	// Create targets with files containing imports
	content := []byte(`package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	fmt.Println("hello")
}
`)

	targets := []*types.AnalysisTarget{
		{
			RootDir:  "/test",
			Language: types.LangGo,
			Files: []types.SourceFile{
				{Path: "/test/main.go", Lines: 15, Class: types.ClassSource, Content: content},
			},
		},
	}

	samples := m.SelectSamples(targets)
	if len(samples) > m.SampleCount() {
		t.Errorf("SelectSamples returned %d samples, max should be %d", len(samples), m.SampleCount())
	}
}

func TestM4Identifiers_SelectSamples(t *testing.T) {
	m := newM4Identifiers()

	// Create targets with exported identifiers
	content := []byte(`package analyzer

type CodeAnalyzer struct {
	config AnalyzerConfig
}

func NewCodeAnalyzer(cfg AnalyzerConfig) *CodeAnalyzer {
	return &CodeAnalyzer{config: cfg}
}

func (a *CodeAnalyzer) AnalyzeDirectory(path string) error {
	return nil
}

var GlobalConfiguration = Config{}
`)

	targets := []*types.AnalysisTarget{
		{
			RootDir:  "/test",
			Language: types.LangGo,
			Files: []types.SourceFile{
				{Path: "/test/analyzer.go", Lines: 20, Class: types.ClassSource, Content: content},
			},
		},
	}

	samples := m.SelectSamples(targets)
	if len(samples) > m.SampleCount() {
		t.Errorf("SelectSamples returned %d samples, max should be %d", len(samples), m.SampleCount())
	}

	// Check that samples have identifier info
	for _, sample := range samples {
		if sample.FunctionName == "" {
			t.Errorf("Sample should have FunctionName set for identifier: %+v", sample)
		}
	}
}

func TestM5Documentation_SelectSamples(t *testing.T) {
	m := newM5Documentation()

	// Create targets with comments
	content := []byte(`package main

// main is the entry point for the application.
// It initializes the config and starts the server.
func main() {
	// Load configuration from environment
	cfg := loadConfig()

	// Start the HTTP server
	// This will block until shutdown
	startServer(cfg)
}

// loadConfig reads configuration from environment variables.
func loadConfig() Config {
	return Config{}
}

// startServer starts the HTTP server on the configured port.
func startServer(cfg Config) {
	// Implementation here
}
`)

	targets := []*types.AnalysisTarget{
		{
			RootDir:  "/test",
			Language: types.LangGo,
			Files: []types.SourceFile{
				{Path: "/test/main.go", Lines: 30, Class: types.ClassSource, Content: content},
			},
		},
	}

	samples := m.SelectSamples(targets)
	if len(samples) > m.SampleCount() {
		t.Errorf("SelectSamples returned %d samples, max should be %d", len(samples), m.SampleCount())
	}
}

func TestMetricTimeoutsArePositive(t *testing.T) {
	for _, m := range AllMetrics() {
		if m.Timeout() <= 0 {
			t.Errorf("%s.Timeout() = %v, want > 0", m.ID(), m.Timeout())
		}
	}
}

func TestMetricSampleCountsArePositive(t *testing.T) {
	for _, m := range AllMetrics() {
		if m.SampleCount() <= 0 {
			t.Errorf("%s.SampleCount() = %d, want > 0", m.ID(), m.SampleCount())
		}
	}
}

func TestMetricDescriptionsNonEmpty(t *testing.T) {
	for _, m := range AllMetrics() {
		if m.Description() == "" {
			t.Errorf("%s.Description() is empty", m.ID())
		}
	}
}

func TestCalculateVariance(t *testing.T) {
	tests := []struct {
		name     string
		scores   []int
		expected float64
	}{
		{"empty", []int{}, 0},
		{"single", []int{5}, 0},
		{"identical", []int{5, 5, 5}, 0},
		{"varied", []int{2, 4, 6}, 2.6666666666666665}, // variance = ((2-4)^2 + (4-4)^2 + (6-4)^2) / 3 = 8/3
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := calculateVariance(tc.scores)
			if got != tc.expected {
				t.Errorf("calculateVariance(%v) = %v, want %v", tc.scores, got, tc.expected)
			}
		})
	}
}

func TestCountIdentifierWords(t *testing.T) {
	tests := []struct {
		name     string
		expected int
	}{
		{"simple", 1},
		{"CamelCase", 2},
		{"PascalCaseWord", 3},
		{"snake_case", 2},
		{"SCREAMING_SNAKE_CASE", 3},
		{"NewHTTPServer", 6}, // N-ew-H-T-T-P-S-erver counts all uppercase transitions
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := countIdentifierWords(tc.name)
			if got != tc.expected {
				t.Errorf("countIdentifierWords(%q) = %d, want %d", tc.name, got, tc.expected)
			}
		})
	}
}

func TestAbs(t *testing.T) {
	tests := []struct {
		input    int
		expected int
	}{
		{0, 0},
		{5, 5},
		{-5, 5},
		{-100, 100},
	}

	for _, tc := range tests {
		got := abs(tc.input)
		if got != tc.expected {
			t.Errorf("abs(%d) = %d, want %d", tc.input, got, tc.expected)
		}
	}
}

func TestMin(t *testing.T) {
	tests := []struct {
		a, b     int
		expected int
	}{
		{1, 2, 1},
		{2, 1, 1},
		{5, 5, 5},
		{-1, 1, -1},
	}

	for _, tc := range tests {
		got := min(tc.a, tc.b)
		if got != tc.expected {
			t.Errorf("min(%d, %d) = %d, want %d", tc.a, tc.b, got, tc.expected)
		}
	}
}

// Test scoring heuristics for M2 (Comprehension)
func TestM2_ScoreComprehensionResponse(t *testing.T) {
	m := newM2Comprehension().(*m2Comprehension)

	tests := []struct {
		name     string
		response string
		minScore int
		maxScore int
	}{
		{
			name:     "empty response",
			response: "",
			minScore: 1,
			maxScore: 5,
		},
		{
			name:     "good response with indicators",
			response: "The function returns the result after handling errors. It validates input and checks conditions in a loop. It iterates through items for each element and ensures edge cases are handled.",
			minScore: 7,
			maxScore: 10,
		},
		{
			name:     "uncertain response",
			response: "I'm not sure what this does. It might process data, probably returns something. The behavior is unclear to me.",
			minScore: 1,
			maxScore: 5,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			score, _ := m.scoreComprehensionResponse(tc.response)
			if score < tc.minScore || score > tc.maxScore {
				t.Errorf("scoreComprehensionResponse() = %d, want between %d and %d", score, tc.minScore, tc.maxScore)
			}
		})
	}
}

// Test scoring heuristics for M3 (Navigation)
func TestM3_ScoreNavigationResponse(t *testing.T) {
	m := newM3Navigation().(*m3Navigation)

	tests := []struct {
		name     string
		response string
		minScore int
		maxScore int
	}{
		{
			name:     "empty response",
			response: "",
			minScore: 1,
			maxScore: 5,
		},
		{
			name:     "good navigation trace",
			response: "Imports: import fmt, import os/path/filepath. The module exports a function that calls another package. Data Flow: main() -> handler() in /src/handlers/user.go -> database.Query() in /src/db/query.go",
			minScore: 7,
			maxScore: 10,
		},
		{
			name:     "failed navigation",
			response: "Cannot find the file. Unable to trace dependencies. File not found in the project.",
			minScore: 1,
			maxScore: 4,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			score, _ := m.scoreNavigationResponse(tc.response)
			if score < tc.minScore || score > tc.maxScore {
				t.Errorf("scoreNavigationResponse() = %d, want between %d and %d", score, tc.minScore, tc.maxScore)
			}
		})
	}
}

// Test scoring heuristics for M4 (Identifiers)
func TestM4_ScoreIdentifierResponse(t *testing.T) {
	m := newM4Identifiers().(*m4Identifiers)

	tests := []struct {
		name     string
		response string
		minScore int
		maxScore int
	}{
		{
			name:     "empty response",
			response: "",
			minScore: 1,
			maxScore: 5,
		},
		{
			name:     "accurate interpretation",
			response: "Interpretation: This function creates a new database connection. It handles connection pooling. Type: function. Verification: Confirmed accurate - the code does exactly that. Accuracy: Correct.",
			minScore: 7,
			maxScore: 10,
		},
		{
			name:     "wrong interpretation",
			response: "I got it wrong.",
			minScore: 1,
			maxScore: 6,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			score, _ := m.scoreIdentifierResponse(tc.response)
			if score < tc.minScore || score > tc.maxScore {
				t.Errorf("scoreIdentifierResponse() = %d, want between %d and %d", score, tc.minScore, tc.maxScore)
			}
		})
	}
}

// Test scoring heuristics for M5 (Documentation)
func TestM5_ScoreDocumentationResponse(t *testing.T) {
	m := newM5Documentation().(*m5Documentation)

	tests := []struct {
		name     string
		response string
		minScore int
		maxScore int
	}{
		{
			name:     "empty response",
			response: "",
			minScore: 1,
			maxScore: 5,
		},
		{
			name:     "thorough analysis",
			response: "## Summary\nOverall documentation is good.\n\n## Accurate Documentation\nThe main function comment correctly describes its behavior.\n\n## Potential Mismatches\nLocation: line 45\nComment says: returns nil on success\nCode does: returns error on failure\nIssue: Documentation is outdated",
			minScore: 8,
			maxScore: 10,
		},
		{
			name:     "failed analysis",
			response: "Cannot analyze the file. Error reading content. No comments found.",
			minScore: 1,
			maxScore: 4,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			score, _ := m.scoreDocumentationResponse(tc.response)
			if score < tc.minScore || score > tc.maxScore {
				t.Errorf("scoreDocumentationResponse() = %d, want between %d and %d", score, tc.minScore, tc.maxScore)
			}
		})
	}
}

// TestScoreTrace_SumsCorrectly verifies that for each metric (M2-M5), the
// ScoreTrace is the source of truth: BaseScore + sum(Deltas) == FinalScore
// (before clamping). Also checks that non-empty responses produce indicators.
func TestScoreTrace_SumsCorrectly(t *testing.T) {
	tests := []struct {
		name     string
		scoreFn  func(string) (int, ScoreTrace)
		response string
	}{
		{
			name:     "M2 good response",
			scoreFn:  newM2ComprehensionMetric().scoreComprehensionResponse,
			response: "The function returns the result after handling errors. It validates input and checks conditions.",
		},
		{
			name:     "M2 empty response",
			scoreFn:  newM2ComprehensionMetric().scoreComprehensionResponse,
			response: "",
		},
		{
			name:     "M3 good response",
			scoreFn:  newM3NavigationMetric().scoreNavigationResponse,
			response: "Imports: import fmt, import os. The module exports a function that calls -> /src/handler.go",
		},
		{
			name:     "M3 empty response",
			scoreFn:  newM3NavigationMetric().scoreNavigationResponse,
			response: "",
		},
		{
			name:     "M4 good response",
			scoreFn:  newM4IdentifiersMetric().scoreIdentifierResponse,
			response: "Interpretation: This function creates a database connection. Verification: Confirmed accurate. Accuracy: Correct.",
		},
		{
			name:     "M4 empty response",
			scoreFn:  newM4IdentifiersMetric().scoreIdentifierResponse,
			response: "",
		},
		{
			name:     "M5 good response",
			scoreFn:  newM5DocumentationMetric().scoreDocumentationResponse,
			response: "## Summary\nGood documentation.\n## Accurate Documentation\nComment correctly describes behavior.\n## Potential Mismatches\nLocation: line 10\nComment says: returns nil\nCode does: returns error\nIssue: outdated",
		},
		{
			name:     "M5 empty response",
			scoreFn:  newM5DocumentationMetric().scoreDocumentationResponse,
			response: "",
		},
	}

	// Expected base scores after grouped indicator refactor
	expectedBase := map[string]int{
		"M2 good response":  2,
		"M2 empty response": 2,
		"M3 good response":  2,
		"M3 empty response": 2,
		"M4 good response":  1,
		"M4 empty response": 1,
		"M5 good response":  3,
		"M5 empty response": 3,
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			score, trace := tc.scoreFn(tc.response)

			// Verify base score matches expected for this metric
			wantBase, ok := expectedBase[tc.name]
			if !ok {
				t.Fatalf("no expected base score for %q", tc.name)
			}
			if trace.BaseScore != wantBase {
				t.Errorf("BaseScore = %d, want %d", trace.BaseScore, wantBase)
			}

			// Verify that indicators exist
			if len(trace.Indicators) == 0 {
				t.Errorf("expected at least 1 indicator, got 0")
			}

			// Compute expected score from trace
			expected := trace.BaseScore
			for _, ind := range trace.Indicators {
				expected += ind.Delta
			}
			if expected < 1 {
				expected = 1
			}
			if expected > 10 {
				expected = 10
			}

			if trace.FinalScore != expected {
				t.Errorf("FinalScore = %d, but BaseScore(%d) + sum(Deltas) clamped = %d",
					trace.FinalScore, trace.BaseScore, expected)
			}

			if score != trace.FinalScore {
				t.Errorf("returned score %d != trace.FinalScore %d", score, trace.FinalScore)
			}

			// Verify Delta is 0 when Matched is false
			for _, ind := range trace.Indicators {
				if !ind.Matched && ind.Delta != 0 {
					t.Errorf("indicator %q: Matched=false but Delta=%d (want 0)", ind.Name, ind.Delta)
				}
			}
		})
	}
}

// TestAllMetrics_CapturePrompt verifies that all 5 metrics populate sr.Prompt
// when Execute() is called with a mock executor.
func TestAllMetrics_CapturePrompt(t *testing.T) {
	executor := &mockExecutor{
		response: `["funcA", "funcB", "funcC"]`,
	}

	// Build source file content that satisfies selection criteria for all metrics
	content := []byte(`package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"encoding/json"
)

// main is the entry point.
// It initializes config and starts the server.
func main() {
	// Load config
	cfg := loadConfig()
	if cfg.Port == 0 {
		cfg.Port = 8080
	}
	startServer(cfg)
}

// loadConfig reads environment variables.
func loadConfig() Config {
	return Config{}
}

// startServer starts the HTTP server.
func startServer(cfg Config) {}

// ProcessRequest handles incoming HTTP requests.
// It validates and routes to the appropriate handler.
func ProcessRequest(r Request) Response {
	if r.Valid {
		switch r.Method {
		case "GET":
			return handleGet(r)
		case "POST":
			return handlePost(r)
		default:
			return errorResponse()
		}
	}
	return errorResponse()
}

// ValidateInput checks request validity.
func ValidateInput(input string) bool {
	if len(input) == 0 {
		return false
	}
	for _, c := range input {
		if c == ';' {
			return false
		}
	}
	return true
}

// BatchProcess handles multiple items.
func BatchProcess(items []string) []string {
	var results []string
	for _, item := range items {
		if len(item) > 0 {
			results = append(results, item)
		} else {
			results = append(results, "empty")
		}
	}
	return results
}

type Config struct{ Port int }
type Request struct{ Valid bool; Method string }
type Response struct{}

func handleGet(r Request) Response { return Response{} }
func handlePost(r Request) Response { return Response{} }
func handle(r Request) Response { return Response{} }
func errorResponse() Response { return Response{} }
`)

	targets := []*types.AnalysisTarget{
		{
			RootDir:  "/test",
			Language: types.LangGo,
			Files: []types.SourceFile{
				{
					Path:    "/test/main.go",
					RelPath: "main.go",
					Lines:   80,
					Class:   types.ClassSource,
					Content: content,
				},
			},
		},
	}

	ctx := context.Background()

	tests := []struct {
		name   string
		metric Metric
	}{
		{"M1", newM1Consistency()},
		{"M2", newM2Comprehension()},
		{"M3", newM3Navigation()},
		{"M4", newM4Identifiers()},
		{"M5", newM5Documentation()},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			samples := tc.metric.SelectSamples(targets)
			if len(samples) == 0 {
				t.Skip("no samples selected (file may not meet selection criteria)")
			}

			result := tc.metric.Execute(ctx, "/test", samples, executor)

			// Check that at least one successful sample has a non-empty Prompt
			foundPrompt := false
			for _, sr := range result.Samples {
				if sr.Error == "" && sr.Prompt != "" {
					foundPrompt = true
					break
				}
			}
			if !foundPrompt {
				t.Errorf("%s: no successful SampleResult has a non-empty Prompt", tc.name)
			}
		})
	}
}

// --- Fixture-based scoring tests ---
// These test scoring functions against real LLM response fixtures captured in 28-01.
// Good responses should score 6-8, weaker responses should score 4-6.

func TestM2_Score_Fixtures(t *testing.T) {
	m := newM2ComprehensionMetric()

	tests := []struct {
		name     string
		fixture  string
		minScore int
		maxScore int
	}{
		{
			name:     "good Go explanation",
			fixture:  "good_go_explanation.txt",
			minScore: 6,
			maxScore: 8,
		},
		{
			name:     "minimal explanation",
			fixture:  "minimal_explanation.txt",
			minScore: 4,
			maxScore: 6,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			response := loadFixture(t, "m2_comprehension", tc.fixture)
			score, trace := m.scoreComprehensionResponse(response)

			if score < tc.minScore || score > tc.maxScore {
				t.Errorf("scoreComprehensionResponse(%s) = %d, want %d-%d\nBaseScore=%d, matched indicators:",
					tc.fixture, score, tc.minScore, tc.maxScore, trace.BaseScore)
				for _, ind := range trace.Indicators {
					if ind.Matched {
						t.Errorf("  %s: delta=%+d", ind.Name, ind.Delta)
					}
				}
			}
		})
	}
}

func TestM3_Score_Fixtures(t *testing.T) {
	m := newM3NavigationMetric()

	tests := []struct {
		name     string
		fixture  string
		minScore int
		maxScore int
	}{
		{
			name:     "good dependency trace",
			fixture:  "good_dependency_trace.txt",
			minScore: 6,
			maxScore: 8,
		},
		{
			name:     "shallow trace",
			fixture:  "shallow_trace.txt",
			minScore: 4,
			maxScore: 6,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			response := loadFixture(t, "m3_navigation", tc.fixture)
			score, trace := m.scoreNavigationResponse(response)

			if score < tc.minScore || score > tc.maxScore {
				t.Errorf("scoreNavigationResponse(%s) = %d, want %d-%d\nBaseScore=%d, matched indicators:",
					tc.fixture, score, tc.minScore, tc.maxScore, trace.BaseScore)
				for _, ind := range trace.Indicators {
					if ind.Matched {
						t.Errorf("  %s: delta=%+d", ind.Name, ind.Delta)
					}
				}
			}
		})
	}
}

func TestM4_Score_Fixtures(t *testing.T) {
	m := newM4IdentifiersMetric()

	tests := []struct {
		name     string
		fixture  string
		minScore int
		maxScore int
	}{
		{
			name:     "accurate interpretation",
			fixture:  "accurate_interpretation.txt",
			minScore: 6,
			maxScore: 8,
		},
		{
			name:     "partial interpretation",
			fixture:  "partial_interpretation.txt",
			minScore: 4,
			maxScore: 6,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			response := loadFixture(t, "m4_identifiers", tc.fixture)
			score, trace := m.scoreIdentifierResponse(response)

			if score < tc.minScore || score > tc.maxScore {
				t.Errorf("scoreIdentifierResponse(%s) = %d, want %d-%d\nBaseScore=%d, matched indicators:",
					tc.fixture, score, tc.minScore, tc.maxScore, trace.BaseScore)
				for _, ind := range trace.Indicators {
					if ind.Matched {
						t.Errorf("  %s: delta=%+d", ind.Name, ind.Delta)
					}
				}
			}
		})
	}
}
