package scoring

import (
	"math"
	"testing"

	"github.com/ingo/agent-readyness/pkg/types"
)

// --- Interpolate tests ---

func TestInterpolate_LowerIsBetter(t *testing.T) {
	// Complexity-style: low values score high
	bp := []Breakpoint{
		{Value: 1, Score: 10},
		{Value: 5, Score: 8},
		{Value: 10, Score: 6},
		{Value: 20, Score: 3},
		{Value: 40, Score: 1},
	}

	tests := []struct {
		name  string
		value float64
		want  float64
	}{
		{"clamp below first", 0, 10.0},
		{"exact first", 1, 10.0},
		{"midpoint 1-5", 3, 9.0},
		{"exact second", 5, 8.0},
		{"midpoint 5-10", 7.5, 7.0},
		{"exact third", 10, 6.0},
		{"midpoint 10-20", 15, 4.5},
		{"exact fourth", 20, 3.0},
		{"midpoint 20-40", 30, 2.0},
		{"exact last", 40, 1.0},
		{"clamp above last", 50, 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Interpolate(bp, tt.value)
			if math.Abs(got-tt.want) > 0.01 {
				t.Errorf("Interpolate(%v) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestInterpolate_HigherIsBetter(t *testing.T) {
	// Coverage-style: high values score high
	bp := []Breakpoint{
		{Value: 0, Score: 1},
		{Value: 30, Score: 4},
		{Value: 50, Score: 6},
		{Value: 70, Score: 8},
		{Value: 90, Score: 10},
	}

	tests := []struct {
		name  string
		value float64
		want  float64
	}{
		{"zero coverage", 0, 1.0},
		{"low coverage", 15, 2.5},
		{"mid coverage", 50, 6.0},
		{"good coverage", 80, 9.0},
		{"perfect coverage", 100, 10.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Interpolate(bp, tt.value)
			if math.Abs(got-tt.want) > 0.01 {
				t.Errorf("Interpolate(%v) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestInterpolate_EmptyBreakpoints(t *testing.T) {
	got := Interpolate(nil, 5.0)
	if got != 5.0 {
		t.Errorf("Interpolate(nil, 5) = %v, want 5.0", got)
	}

	got = Interpolate([]Breakpoint{}, 5.0)
	if got != 5.0 {
		t.Errorf("Interpolate([], 5) = %v, want 5.0", got)
	}
}

func TestInterpolate_SingleBreakpoint(t *testing.T) {
	bp := []Breakpoint{{Value: 5, Score: 7}}
	tests := []struct {
		name  string
		value float64
		want  float64
	}{
		{"below", 3, 7.0},
		{"exact", 5, 7.0},
		{"above", 10, 7.0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Interpolate(bp, tt.value)
			if math.Abs(got-tt.want) > 0.01 {
				t.Errorf("Interpolate(%v) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

// --- computeComposite tests ---

func TestComputeComposite_AllTens(t *testing.T) {
	cats := []types.CategoryScore{
		{Name: "C1", Score: 10, Weight: 0.25},
		{Name: "C3", Score: 10, Weight: 0.20},
		{Name: "C6", Score: 10, Weight: 0.15},
	}
	s := &Scorer{Config: DefaultConfig()}
	got := s.computeComposite(cats)
	if math.Abs(got-10.0) > 0.01 {
		t.Errorf("computeComposite(all 10s) = %v, want 10.0", got)
	}
}

func TestComputeComposite_Mixed(t *testing.T) {
	cats := []types.CategoryScore{
		{Name: "C1", Score: 8, Weight: 0.25},
		{Name: "C3", Score: 6, Weight: 0.20},
		{Name: "C6", Score: 7, Weight: 0.15},
	}
	s := &Scorer{Config: DefaultConfig()}
	got := s.computeComposite(cats)
	// (8*0.25 + 6*0.20 + 7*0.15) / 0.60 = (2.0 + 1.2 + 1.05) / 0.60 = 4.25 / 0.60 = 7.0833...
	want := 7.0833
	if math.Abs(got-want) > 0.01 {
		t.Errorf("computeComposite(mixed) = %v, want ~%v", got, want)
	}
}

func TestComputeComposite_SingleCategory(t *testing.T) {
	cats := []types.CategoryScore{
		{Name: "C1", Score: 8, Weight: 0.25},
	}
	s := &Scorer{Config: DefaultConfig()}
	got := s.computeComposite(cats)
	if math.Abs(got-8.0) > 0.01 {
		t.Errorf("computeComposite(single C1=8) = %v, want 8.0", got)
	}
}

func TestComputeComposite_Empty(t *testing.T) {
	s := &Scorer{Config: DefaultConfig()}
	got := s.computeComposite(nil)
	if got != 0 {
		t.Errorf("computeComposite(nil) = %v, want 0", got)
	}
}

func TestComputeComposite_SkipsNegativeScore(t *testing.T) {
	cats := []types.CategoryScore{
		{Name: "C1", Score: 8, Weight: 0.25},
		{Name: "C3", Score: -1, Weight: 0.20}, // unavailable
		{Name: "C6", Score: 6, Weight: 0.15},
	}
	s := &Scorer{Config: DefaultConfig()}
	got := s.computeComposite(cats)
	// (8*0.25 + 6*0.15) / (0.25 + 0.15) = (2.0 + 0.9) / 0.40 = 7.25
	want := 7.25
	if math.Abs(got-want) > 0.01 {
		t.Errorf("computeComposite(skip negative) = %v, want %v", got, want)
	}
}

// --- classifyTier tests ---

func TestClassifyTier(t *testing.T) {
	s := &Scorer{Config: DefaultConfig()}

	tests := []struct {
		name  string
		score float64
		want  string
	}{
		{"perfect", 10.0, "Agent-Ready"},
		{"boundary 8.0", 8.0, "Agent-Ready"},
		{"just below 8.0", 7.99, "Agent-Assisted"},
		{"boundary 6.0", 6.0, "Agent-Assisted"},
		{"just below 6.0", 5.99, "Agent-Limited"},
		{"boundary 4.0", 4.0, "Agent-Limited"},
		{"just below 4.0", 3.99, "Agent-Hostile"},
		{"minimum", 1.0, "Agent-Hostile"},
		{"near zero", 0.5, "Agent-Hostile"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := s.classifyTier(tt.score)
			if got != tt.want {
				t.Errorf("classifyTier(%v) = %q, want %q", tt.score, got, tt.want)
			}
		})
	}
}

// --- CategoryScore tests ---

func TestCategoryScore_AllAvailable(t *testing.T) {
	subs := []types.SubScore{
		{MetricName: "a", Score: 8, Weight: 0.5, Available: true},
		{MetricName: "b", Score: 6, Weight: 0.5, Available: true},
	}
	got := CategoryScore(subs)
	// (8*0.5 + 6*0.5) / 1.0 = 7.0
	if math.Abs(got-7.0) > 0.01 {
		t.Errorf("CategoryScore = %v, want 7.0", got)
	}
}

func TestCategoryScore_SkipUnavailable(t *testing.T) {
	subs := []types.SubScore{
		{MetricName: "a", Score: 8, Weight: 0.5, Available: true},
		{MetricName: "b", Score: 0, Weight: 0.5, Available: false},
	}
	got := CategoryScore(subs)
	// Only 'a' contributes: 8*0.5 / 0.5 = 8.0
	if math.Abs(got-8.0) > 0.01 {
		t.Errorf("CategoryScore = %v, want 8.0", got)
	}
}

func TestCategoryScore_Empty(t *testing.T) {
	got := CategoryScore(nil)
	if got != -1.0 {
		t.Errorf("CategoryScore(nil) = %v, want -1.0", got)
	}
}

func TestCategoryScore_AllUnavailable(t *testing.T) {
	subs := []types.SubScore{
		{MetricName: "a", Score: 0, Weight: 0.5, Available: false},
		{MetricName: "b", Score: 0, Weight: 0.5, Available: false},
	}
	got := CategoryScore(subs)
	if got != -1.0 {
		t.Errorf("CategoryScore(all unavailable) = %v, want -1.0", got)
	}
}

// --- Helper tests ---

func TestAvgMapValues(t *testing.T) {
	tests := []struct {
		name string
		m    map[string]int
		want float64
	}{
		{"nil map", nil, 0},
		{"empty map", map[string]int{}, 0},
		{"single entry", map[string]int{"a": 6}, 6.0},
		{"multiple entries", map[string]int{"a": 2, "b": 4, "c": 6}, 4.0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := avgMapValues(tt.m)
			if math.Abs(got-tt.want) > 0.01 {
				t.Errorf("avgMapValues(%v) = %v, want %v", tt.m, got, tt.want)
			}
		})
	}
}

func TestFindMetric(t *testing.T) {
	metrics := []MetricThresholds{
		{Name: "complexity_avg", Weight: 0.25},
		{Name: "func_length_avg", Weight: 0.20},
	}

	t.Run("found", func(t *testing.T) {
		got := findMetric(metrics, "complexity_avg")
		if got == nil {
			t.Fatal("expected non-nil")
		}
		if got.Name != "complexity_avg" {
			t.Errorf("got name %q, want complexity_avg", got.Name)
		}
	})

	t.Run("not found", func(t *testing.T) {
		got := findMetric(metrics, "nonexistent")
		if got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})
}

// --- Score C1 tests (via Score method) ---

func makeHealthyC1() *types.AnalysisResult {
	return &types.AnalysisResult{
		Name:     "code-health",
		Category: "C1",
		Metrics: map[string]interface{}{
			"c1": &types.C1Metrics{
				CyclomaticComplexity: types.MetricSummary{Avg: 3.0, Max: 8},
				FunctionLength:       types.MetricSummary{Avg: 12.0, Max: 30},
				FileSize:             types.MetricSummary{Avg: 100.0, Max: 250},
				AfferentCoupling:     map[string]int{"pkg/a": 1, "pkg/b": 3},
				EfferentCoupling:     map[string]int{"pkg/a": 2},
				DuplicationRate:      2.0,
			},
		},
	}
}

// scoreCategory is a test helper that scores a single analysis result and returns the category score.
func scoreCategory(s *Scorer, ar *types.AnalysisResult) types.CategoryScore {
	result, _ := s.Score([]*types.AnalysisResult{ar})
	if len(result.Categories) == 0 {
		catCfg := s.Config.Categories[ar.Category]
		return types.CategoryScore{Name: ar.Category, Weight: catCfg.Weight}
	}
	return result.Categories[0]
}

func TestScoreC1_Healthy(t *testing.T) {
	s := &Scorer{Config: DefaultConfig()}
	ar := makeHealthyC1()
	got := scoreCategory(s, ar)

	if got.Name != "C1" {
		t.Errorf("name = %q, want C1", got.Name)
	}
	if got.Weight != 0.25 {
		t.Errorf("weight = %v, want 0.25", got.Weight)
	}
	if len(got.SubScores) != 6 {
		t.Fatalf("subscore count = %d, want 6", len(got.SubScores))
	}
	// With low complexity (3.0), the score should be high (>8)
	if got.Score < 7.0 {
		t.Errorf("healthy C1 score = %v, want > 7.0", got.Score)
	}

	// Verify all sub-scores are available
	for _, ss := range got.SubScores {
		if !ss.Available {
			t.Errorf("sub-score %q should be available", ss.MetricName)
		}
	}
}

func TestScoreC1_PoorCodebase(t *testing.T) {
	s := &Scorer{Config: DefaultConfig()}
	ar := &types.AnalysisResult{
		Name:     "code-health",
		Category: "C1",
		Metrics: map[string]interface{}{
			"c1": &types.C1Metrics{
				CyclomaticComplexity: types.MetricSummary{Avg: 25.0, Max: 60},
				FunctionLength:       types.MetricSummary{Avg: 70.0, Max: 200},
				FileSize:             types.MetricSummary{Avg: 600.0, Max: 1500},
				AfferentCoupling:     map[string]int{"a": 12, "b": 8},
				EfferentCoupling:     map[string]int{"a": 15, "b": 5},
				DuplicationRate:      20.0,
			},
		},
	}
	got := scoreCategory(s, ar)
	if got.Score > 4.0 {
		t.Errorf("poor C1 score = %v, want < 4.0", got.Score)
	}
}

func TestScoreC1_EmptyCouplingMaps(t *testing.T) {
	s := &Scorer{Config: DefaultConfig()}
	ar := &types.AnalysisResult{
		Name:     "code-health",
		Category: "C1",
		Metrics: map[string]interface{}{
			"c1": &types.C1Metrics{
				CyclomaticComplexity: types.MetricSummary{Avg: 3.0},
				FunctionLength:       types.MetricSummary{Avg: 10.0},
				FileSize:             types.MetricSummary{Avg: 100.0},
				AfferentCoupling:     nil,
				EfferentCoupling:     map[string]int{},
				DuplicationRate:      0.0,
			},
		},
	}
	got := scoreCategory(s, ar)

	// Empty maps -> raw value 0 -> should get top score for coupling
	for _, ss := range got.SubScores {
		if ss.MetricName == "afferent_coupling_avg" || ss.MetricName == "efferent_coupling_avg" {
			if ss.RawValue != 0 {
				t.Errorf("%s raw value = %v, want 0", ss.MetricName, ss.RawValue)
			}
			if ss.Score < 9.5 {
				t.Errorf("%s score = %v, want >= 9.5 (top score for 0 coupling)", ss.MetricName, ss.Score)
			}
		}
	}
}

func TestScoreC1_InvalidMetricsType(t *testing.T) {
	s := &Scorer{Config: DefaultConfig()}
	ar := &types.AnalysisResult{
		Name:     "code-health",
		Category: "C1",
		Metrics:  map[string]interface{}{"c1": "not-a-struct"},
	}
	got := scoreCategory(s, ar)
	// Should return zero-value CategoryScore on type assertion failure
	if got.Score != 0 {
		t.Errorf("invalid metrics type score = %v, want 0", got.Score)
	}
}

func TestScoreC1_CouplingAverage(t *testing.T) {
	s := &Scorer{Config: DefaultConfig()}
	ar := &types.AnalysisResult{
		Name:     "code-health",
		Category: "C1",
		Metrics: map[string]interface{}{
			"c1": &types.C1Metrics{
				CyclomaticComplexity: types.MetricSummary{Avg: 5.0},
				FunctionLength:       types.MetricSummary{Avg: 15.0},
				FileSize:             types.MetricSummary{Avg: 150.0},
				AfferentCoupling:     map[string]int{"a": 2, "b": 4, "c": 6},
				EfferentCoupling:     map[string]int{"a": 3, "b": 9},
				DuplicationRate:      3.0,
			},
		},
	}
	got := scoreCategory(s, ar)

	for _, ss := range got.SubScores {
		if ss.MetricName == "afferent_coupling_avg" {
			// avg of {2,4,6} = 4.0
			if math.Abs(ss.RawValue-4.0) > 0.01 {
				t.Errorf("afferent_coupling_avg raw = %v, want 4.0", ss.RawValue)
			}
		}
		if ss.MetricName == "efferent_coupling_avg" {
			// avg of {3,9} = 6.0
			if math.Abs(ss.RawValue-6.0) > 0.01 {
				t.Errorf("efferent_coupling_avg raw = %v, want 6.0", ss.RawValue)
			}
		}
	}
}

// --- scoreC3 tests ---

func TestScoreC3_Healthy(t *testing.T) {
	s := &Scorer{Config: DefaultConfig()}
	ar := &types.AnalysisResult{
		Name:     "architecture",
		Category: "C3",
		Metrics: map[string]interface{}{
			"c3": &types.C3Metrics{
				MaxDirectoryDepth: 3,
				ModuleFanout:      types.MetricSummary{Avg: 2.5},
				CircularDeps:      nil,
				ImportComplexity:  types.MetricSummary{Avg: 1.5},
				DeadExports:       nil,
			},
		},
	}
	got := scoreCategory(s, ar)
	if got.Name != "C3" {
		t.Errorf("name = %q, want C3", got.Name)
	}
	if got.Weight != 0.20 {
		t.Errorf("weight = %v, want 0.20", got.Weight)
	}
	if len(got.SubScores) != 5 {
		t.Fatalf("subscore count = %d, want 5", len(got.SubScores))
	}
	if got.Score < 7.0 {
		t.Errorf("healthy C3 score = %v, want > 7.0", got.Score)
	}
}

func TestScoreC3_PoorArchitecture(t *testing.T) {
	s := &Scorer{Config: DefaultConfig()}
	ar := &types.AnalysisResult{
		Name:     "architecture",
		Category: "C3",
		Metrics: map[string]interface{}{
			"c3": &types.C3Metrics{
				MaxDirectoryDepth: 8,
				ModuleFanout:      types.MetricSummary{Avg: 12.0},
				CircularDeps:      [][]string{{"a", "b"}, {"c", "d"}, {"e", "f"}, {"g", "h"}},
				ImportComplexity:  types.MetricSummary{Avg: 7.0},
				DeadExports: []types.DeadExport{
					{Name: "a"}, {Name: "b"}, {Name: "c"}, {Name: "d"}, {Name: "e"},
					{Name: "f"}, {Name: "g"}, {Name: "h"}, {Name: "i"}, {Name: "j"},
					{Name: "k"}, {Name: "l"}, {Name: "m"}, {Name: "n"}, {Name: "o"},
					{Name: "p"}, {Name: "q"}, {Name: "r"}, {Name: "s"}, {Name: "t"},
					{Name: "u"}, {Name: "v"}, {Name: "w"}, {Name: "x"}, {Name: "y"},
					{Name: "z"}, {Name: "aa"}, {Name: "bb"}, {Name: "cc"}, {Name: "dd"},
					{Name: "ee"}, {Name: "ff"}, {Name: "gg"}, {Name: "hh"}, {Name: "ii"},
				},
			},
		},
	}
	got := scoreCategory(s, ar)
	if got.Score > 4.0 {
		t.Errorf("poor C3 score = %v, want < 4.0", got.Score)
	}
}

func TestScoreC3_MetricExtraction(t *testing.T) {
	s := &Scorer{Config: DefaultConfig()}
	ar := &types.AnalysisResult{
		Name:     "architecture",
		Category: "C3",
		Metrics: map[string]interface{}{
			"c3": &types.C3Metrics{
				MaxDirectoryDepth: 5,
				ModuleFanout:      types.MetricSummary{Avg: 4.0},
				CircularDeps:      [][]string{{"a", "b"}, {"c", "d"}},
				ImportComplexity:  types.MetricSummary{Avg: 3.0},
				DeadExports:       []types.DeadExport{{Name: "x"}, {Name: "y"}, {Name: "z"}},
			},
		},
	}
	got := scoreCategory(s, ar)

	expected := map[string]float64{
		"max_dir_depth":        5.0,
		"module_fanout_avg":    4.0,
		"circular_deps":        2.0,
		"import_complexity_avg": 3.0,
		"dead_exports":         3.0,
	}

	for _, ss := range got.SubScores {
		want, ok := expected[ss.MetricName]
		if !ok {
			t.Errorf("unexpected metric %q", ss.MetricName)
			continue
		}
		if math.Abs(ss.RawValue-want) > 0.01 {
			t.Errorf("%s raw = %v, want %v", ss.MetricName, ss.RawValue, want)
		}
	}
}

// --- scoreC6 tests ---

func TestScoreC6_Healthy(t *testing.T) {
	s := &Scorer{Config: DefaultConfig()}
	ar := &types.AnalysisResult{
		Name:     "testing",
		Category: "C6",
		Metrics: map[string]interface{}{
			"c6": &types.C6Metrics{
				TestFileCount:    20,
				SourceFileCount:  25,
				TestToCodeRatio:  0.9,
				CoveragePercent:  75.0,
				TestIsolation:    85.0,
				AssertionDensity: types.MetricSummary{Avg: 3.5},
			},
		},
	}
	got := scoreCategory(s, ar)
	if got.Name != "C6" {
		t.Errorf("name = %q, want C6", got.Name)
	}
	if got.Weight != 0.15 {
		t.Errorf("weight = %v, want 0.15", got.Weight)
	}
	if len(got.SubScores) != 5 {
		t.Fatalf("subscore count = %d, want 5", len(got.SubScores))
	}
	if got.Score < 7.0 {
		t.Errorf("healthy C6 score = %v, want > 7.0", got.Score)
	}
	// All sub-scores should be available
	for _, ss := range got.SubScores {
		if !ss.Available {
			t.Errorf("sub-score %q should be available", ss.MetricName)
		}
	}
}

func TestScoreC6_MissingCoverage(t *testing.T) {
	s := &Scorer{Config: DefaultConfig()}
	ar := &types.AnalysisResult{
		Name:     "testing",
		Category: "C6",
		Metrics: map[string]interface{}{
			"c6": &types.C6Metrics{
				TestFileCount:    10,
				SourceFileCount:  20,
				TestToCodeRatio:  0.5,
				CoveragePercent:  -1, // unavailable
				TestIsolation:    60.0,
				AssertionDensity: types.MetricSummary{Avg: 2.0},
			},
		},
	}
	got := scoreCategory(s, ar)

	// Coverage sub-score should be marked unavailable
	for _, ss := range got.SubScores {
		if ss.MetricName == "coverage_percent" {
			if ss.Available {
				t.Error("coverage_percent should be Available=false when CoveragePercent == -1")
			}
		} else {
			if !ss.Available {
				t.Errorf("%q should be available", ss.MetricName)
			}
		}
	}

	// Score should still be valid (weight redistributed)
	if got.Score < 1.0 || got.Score > 10.0 {
		t.Errorf("score with missing coverage = %v, want between 1 and 10", got.Score)
	}
}

func TestScoreC6_ZeroSourceFiles(t *testing.T) {
	s := &Scorer{Config: DefaultConfig()}
	ar := &types.AnalysisResult{
		Name:     "testing",
		Category: "C6",
		Metrics: map[string]interface{}{
			"c6": &types.C6Metrics{
				TestFileCount:    0,
				SourceFileCount:  0,
				TestToCodeRatio:  0,
				CoveragePercent:  -1,
				TestIsolation:    0,
				AssertionDensity: types.MetricSummary{Avg: 0},
			},
		},
	}
	// Must not panic on zero division
	got := scoreCategory(s, ar)

	for _, ss := range got.SubScores {
		if ss.MetricName == "test_file_ratio" {
			if ss.RawValue != 0 {
				t.Errorf("test_file_ratio with 0 source files = %v, want 0", ss.RawValue)
			}
		}
	}
}

func TestScoreC6_TestFileRatio(t *testing.T) {
	s := &Scorer{Config: DefaultConfig()}
	ar := &types.AnalysisResult{
		Name:     "testing",
		Category: "C6",
		Metrics: map[string]interface{}{
			"c6": &types.C6Metrics{
				TestFileCount:    15,
				SourceFileCount:  20,
				TestToCodeRatio:  0.8,
				CoveragePercent:  70.0,
				TestIsolation:    80.0,
				AssertionDensity: types.MetricSummary{Avg: 3.0},
			},
		},
	}
	got := scoreCategory(s, ar)

	for _, ss := range got.SubScores {
		if ss.MetricName == "test_file_ratio" {
			// 15/20 = 0.75
			if math.Abs(ss.RawValue-0.75) > 0.01 {
				t.Errorf("test_file_ratio raw = %v, want 0.75", ss.RawValue)
			}
		}
	}
}

// --- scoreC2 tests ---

func TestScoreC2_GoMetrics(t *testing.T) {
	s := &Scorer{Config: DefaultConfig()}
	ar := &types.AnalysisResult{
		Name:     "semantic-explicitness",
		Category: "C2",
		Metrics: map[string]interface{}{
			"c2": &types.C2Metrics{
				Aggregate: &types.C2LanguageMetrics{
					TypeAnnotationCoverage: 100,
					NamingConsistency:      92,
					MagicNumberRatio:       3.5,
					TypeStrictness:         1,
					NullSafety:             65,
				},
			},
		},
	}
	got := scoreCategory(s, ar)

	if got.Name != "C2" {
		t.Errorf("name = %q, want C2", got.Name)
	}
	if got.Weight != 0.10 {
		t.Errorf("weight = %v, want 0.10", got.Weight)
	}
	if len(got.SubScores) != 5 {
		t.Fatalf("subscore count = %d, want 5", len(got.SubScores))
	}
	// TypeAnnotationCoverage=100 -> 10, NamingConsistency=92 -> ~7.4, MagicNumberRatio=3.5 -> ~8.7,
	// TypeStrictness=1 -> 10, NullSafety=65 -> ~7.0
	// Expected weighted average should be high
	if got.Score < 7.0 {
		t.Errorf("C2 score with good Go metrics = %v, want > 7.0", got.Score)
	}

	// Verify all sub-scores are available
	for _, ss := range got.SubScores {
		if !ss.Available {
			t.Errorf("sub-score %q should be available", ss.MetricName)
		}
	}
}

func TestScoreC2_NilAggregate(t *testing.T) {
	s := &Scorer{Config: DefaultConfig()}
	ar := &types.AnalysisResult{
		Name:     "semantic-explicitness",
		Category: "C2",
		Metrics: map[string]interface{}{
			"c2": &types.C2Metrics{
				Aggregate: nil,
			},
		},
	}
	got := scoreCategory(s, ar)
	if got.Score != 0 {
		t.Errorf("C2 score with nil aggregate = %v, want 0", got.Score)
	}
}

// --- Score (full round-trip) tests ---

func TestScore_FullRoundTrip(t *testing.T) {
	s := &Scorer{Config: DefaultConfig()}
	results := []*types.AnalysisResult{
		makeHealthyC1(),
		{
			Name:     "architecture",
			Category: "C3",
			Metrics: map[string]interface{}{
				"c3": &types.C3Metrics{
					MaxDirectoryDepth: 3,
					ModuleFanout:      types.MetricSummary{Avg: 2.0},
					CircularDeps:      nil,
					ImportComplexity:  types.MetricSummary{Avg: 1.5},
					DeadExports:       nil,
				},
			},
		},
		{
			Name:     "testing",
			Category: "C6",
			Metrics: map[string]interface{}{
				"c6": &types.C6Metrics{
					TestFileCount:    20,
					SourceFileCount:  25,
					TestToCodeRatio:  0.8,
					CoveragePercent:  70.0,
					TestIsolation:    80.0,
					AssertionDensity: types.MetricSummary{Avg: 3.0},
				},
			},
		},
	}

	got, err := s.Score(results)
	if err != nil {
		t.Fatalf("Score() error: %v", err)
	}

	// Should have 3 category scores
	if len(got.Categories) != 3 {
		t.Fatalf("categories count = %d, want 3", len(got.Categories))
	}

	// Composite should be valid
	if got.Composite < 1.0 || got.Composite > 10.0 {
		t.Errorf("composite = %v, want between 1 and 10", got.Composite)
	}

	// Tier should be non-empty
	if got.Tier == "" {
		t.Error("tier should not be empty")
	}

	// With healthy metrics, should score well
	if got.Composite < 6.0 {
		t.Errorf("healthy composite = %v, want > 6.0", got.Composite)
	}

	// Verify category names
	names := map[string]bool{}
	for _, cat := range got.Categories {
		names[cat.Name] = true
	}
	for _, want := range []string{"C1", "C3", "C6"} {
		if !names[want] {
			t.Errorf("missing category %s", want)
		}
	}
}

func TestScore_EmptyResults(t *testing.T) {
	s := &Scorer{Config: DefaultConfig()}
	got, err := s.Score(nil)
	if err != nil {
		t.Fatalf("Score(nil) error: %v", err)
	}
	if len(got.Categories) != 0 {
		t.Errorf("categories count = %d, want 0", len(got.Categories))
	}
}

func TestScore_UnknownCategory(t *testing.T) {
	s := &Scorer{Config: DefaultConfig()}
	results := []*types.AnalysisResult{
		{Name: "unknown", Category: "C99", Metrics: map[string]interface{}{}},
	}
	// Unknown categories should be silently skipped
	got, err := s.Score(results)
	if err != nil {
		t.Fatalf("Score() error: %v", err)
	}
	if len(got.Categories) != 0 {
		t.Errorf("categories count = %d, want 0 (unknown skipped)", len(got.Categories))
	}
}

// --- extractC7 tests ---

func TestExtractC7_ReturnsAllMetrics(t *testing.T) {
	ar := &types.AnalysisResult{
		Metrics: map[string]interface{}{
			"c7": &types.C7Metrics{
				Available:                      true,
				OverallScore:                   75.0,
				TaskExecutionConsistency:       8,
				CodeBehaviorComprehension:      7,
				CrossFileNavigation:            6,
				IdentifierInterpretability:     7,
				DocumentationAccuracyDetection: 5,
			},
		},
	}

	rawValues, unavailable, _ := extractC7(ar)

	if unavailable != nil {
		t.Errorf("expected no unavailable metrics, got %v", unavailable)
	}

	expectedKeys := []string{
		"task_execution_consistency",
		"code_behavior_comprehension",
		"cross_file_navigation",
		"identifier_interpretability",
		"documentation_accuracy_detection",
	}

	for _, key := range expectedKeys {
		val, ok := rawValues[key]
		if !ok {
			t.Errorf("missing key %q in rawValues", key)
			continue
		}
		if val == 0 {
			t.Errorf("key %q has value 0, expected non-zero", key)
		}
	}

	// Verify specific values
	if rawValues["code_behavior_comprehension"] != 7.0 {
		t.Errorf("code_behavior_comprehension = %v, want 7.0", rawValues["code_behavior_comprehension"])
	}
	if rawValues["task_execution_consistency"] != 8.0 {
		t.Errorf("task_execution_consistency = %v, want 8.0", rawValues["task_execution_consistency"])
	}
	if rawValues["cross_file_navigation"] != 6.0 {
		t.Errorf("cross_file_navigation = %v, want 6.0", rawValues["cross_file_navigation"])
	}
	if rawValues["identifier_interpretability"] != 7.0 {
		t.Errorf("identifier_interpretability = %v, want 7.0", rawValues["identifier_interpretability"])
	}
	if rawValues["documentation_accuracy_detection"] != 5.0 {
		t.Errorf("documentation_accuracy_detection = %v, want 5.0", rawValues["documentation_accuracy_detection"])
	}

	// Verify count: exactly 5 keys
	if len(rawValues) != 5 {
		t.Errorf("expected 5 keys, got %d", len(rawValues))
	}
}

func TestExtractC7_UnavailableMarksAllMetrics(t *testing.T) {
	ar := &types.AnalysisResult{
		Metrics: map[string]interface{}{
			"c7": &types.C7Metrics{
				Available: false,
			},
		},
	}

	_, unavailable, _ := extractC7(ar)

	expectedUnavailable := []string{
		"task_execution_consistency",
		"code_behavior_comprehension",
		"cross_file_navigation",
		"identifier_interpretability",
		"documentation_accuracy_detection",
	}

	for _, key := range expectedUnavailable {
		if !unavailable[key] {
			t.Errorf("expected %q in unavailable set", key)
		}
	}

	// Verify count: exactly 5 unavailable
	if len(unavailable) != 5 {
		t.Errorf("expected 5 unavailable metrics, got %d", len(unavailable))
	}
}

func TestScoreC7_NonZeroSubScores(t *testing.T) {
	s := &Scorer{Config: DefaultConfig()}
	ar := &types.AnalysisResult{
		Name:     "agent-evaluation",
		Category: "C7",
		Metrics: map[string]interface{}{
			"c7": &types.C7Metrics{
				Available:                      true,
				OverallScore:                   75.0,
				TaskExecutionConsistency:       8,
				CodeBehaviorComprehension:      7,
				CrossFileNavigation:            6,
				IdentifierInterpretability:     7,
				DocumentationAccuracyDetection: 5,
			},
		},
	}
	got := scoreCategory(s, ar)

	if got.Name != "C7" {
		t.Errorf("name = %q, want C7", got.Name)
	}
	if got.Weight != 0.10 {
		t.Errorf("weight = %v, want 0.10", got.Weight)
	}
	if len(got.SubScores) != 5 {
		t.Fatalf("subscore count = %d, want 5", len(got.SubScores))
	}

	// All 5 MECE metrics should produce non-zero scores
	nonZero := 0
	for _, ss := range got.SubScores {
		if ss.Score > 0 {
			nonZero++
		}
	}
	if nonZero != 5 {
		t.Errorf("expected 5 non-zero sub-scores, got %d", nonZero)
	}

	// Category score should be non-zero (this was the original bug)
	if got.Score <= 0 {
		t.Errorf("C7 category score = %v, want > 0 (was the original bug)", got.Score)
	}

	// With values 5-8, score should be reasonable (not near zero or max)
	if got.Score < 4.0 || got.Score > 9.0 {
		t.Errorf("C7 score = %v, want between 4.0 and 9.0 for mid-range inputs", got.Score)
	}
}

// --- Evidence extraction tests ---

func TestExtractEvidence_AllCategories(t *testing.T) {
	tests := []struct {
		name     string
		category string
		ar       *types.AnalysisResult
		// metrics that should have non-empty evidence (file-level violations)
		nonEmptyMetrics []string
		// metrics that should have empty (but non-nil) evidence
		emptyMetrics []string
		// total expected metric keys in evidence map
		totalKeys int
	}{
		{
			name:     "C1 - Code Health with violations",
			category: "C1",
			ar: &types.AnalysisResult{
				Name:     "code-health",
				Category: "C1",
				Metrics: map[string]interface{}{
					"c1": &types.C1Metrics{
						CyclomaticComplexity: types.MetricSummary{Avg: 30.0, Max: 50, MaxEntity: "pkg/big.go"},
						FunctionLength:       types.MetricSummary{Avg: 100.0, Max: 300, MaxEntity: "pkg/big.go"},
						FileSize:             types.MetricSummary{Avg: 800.0, Max: 2000, MaxEntity: "pkg/huge.go"},
						AfferentCoupling:     map[string]int{"pkg/core": 10, "pkg/util": 5},
						EfferentCoupling:     map[string]int{"pkg/core": 8, "pkg/db": 12},
						DuplicationRate:      15.0,
						DuplicatedBlocks: []types.DuplicateBlock{
							{FileA: "a.go", StartA: 10, EndA: 20, FileB: "b.go", StartB: 30, EndB: 40, LineCount: 10},
						},
						Functions: []types.FunctionMetric{
							{Name: "bigFunc", File: "pkg/big.go", Line: 1, Complexity: 30, LineCount: 100},
							{Name: "anotherFunc", File: "pkg/other.go", Line: 50, Complexity: 20, LineCount: 80},
						},
					},
				},
			},
			nonEmptyMetrics: []string{"complexity_avg", "func_length_avg", "file_size_avg", "afferent_coupling_avg", "efferent_coupling_avg", "duplication_rate"},
			emptyMetrics:    []string{},
			totalKeys:       6,
		},
		{
			name:     "C2 - Semantic Explicitness with aggregate",
			category: "C2",
			ar: &types.AnalysisResult{
				Name:     "semantic-explicitness",
				Category: "C2",
				Metrics: map[string]interface{}{
					"c2": &types.C2Metrics{
						Aggregate: &types.C2LanguageMetrics{
							TypeAnnotationCoverage: 50.0,
							NamingConsistency:      60.0,
							MagicNumberRatio:       15.0,
							TypeStrictness:         0,
							NullSafety:             30.0,
						},
					},
				},
			},
			// C2 currently returns empty evidence for all metrics (no file-level detail)
			nonEmptyMetrics: []string{},
			emptyMetrics:    []string{"type_annotation_coverage", "naming_consistency", "magic_number_ratio", "type_strictness", "null_safety"},
			totalKeys:       5,
		},
		{
			name:     "C3 - Architecture with violations",
			category: "C3",
			ar: &types.AnalysisResult{
				Name:     "architecture",
				Category: "C3",
				Metrics: map[string]interface{}{
					"c3": &types.C3Metrics{
						MaxDirectoryDepth: 8,
						ModuleFanout:      types.MetricSummary{Avg: 10.0, Max: 25, MaxEntity: "pkg/hub"},
						CircularDeps:      [][]string{{"a", "b", "a"}, {"c", "d", "c"}},
						ImportComplexity:  types.MetricSummary{Avg: 5.0, Max: 12, MaxEntity: "pkg/deep/nested"},
						DeadExports: []types.DeadExport{
							{Name: "UnusedFunc", File: "pkg/unused.go", Line: 10, Kind: "func"},
							{Name: "UnusedType", File: "pkg/unused.go", Line: 20, Kind: "type"},
						},
					},
				},
			},
			nonEmptyMetrics: []string{"module_fanout_avg", "circular_deps", "import_complexity_avg", "dead_exports"},
			emptyMetrics:    []string{"max_dir_depth"},
			totalKeys:       5,
		},
		{
			name:     "C4 - Documentation (binary metrics produce empty evidence)",
			category: "C4",
			ar: &types.AnalysisResult{
				Name:     "documentation",
				Category: "C4",
				Metrics: map[string]interface{}{
					"c4": &types.C4Metrics{
						ReadmeWordCount:     200,
						CommentDensity:      10.0,
						APIDocCoverage:      50.0,
						ChangelogPresent:    true,
						ExamplesPresent:     false,
						ContributingPresent: true,
						DiagramsPresent:     false,
					},
				},
			},
			// C4 has all empty evidence -- binary/count metrics, no file-level detail
			nonEmptyMetrics: []string{},
			emptyMetrics:    []string{"readme_word_count", "comment_density", "api_doc_coverage", "changelog_present", "examples_present", "contributing_present", "diagrams_present"},
			totalKeys:       7,
		},
		{
			name:     "C5 - Temporal Dynamics with hotspots",
			category: "C5",
			ar: &types.AnalysisResult{
				Name:     "temporal",
				Category: "C5",
				Metrics: map[string]interface{}{
					"c5": &types.C5Metrics{
						Available:            true,
						ChurnRate:            50.0,
						TemporalCouplingPct:  10.0,
						AuthorFragmentation:  3.0,
						CommitStability:      5.0,
						HotspotConcentration: 40.0,
						TopHotspots: []types.FileChurn{
							{Path: "pkg/hot.go", TotalChanges: 100, CommitCount: 50, AuthorCount: 5},
							{Path: "pkg/warm.go", TotalChanges: 60, CommitCount: 30, AuthorCount: 3},
						},
						CoupledPairs: []types.CoupledPair{
							{FileA: "pkg/a.go", FileB: "pkg/b.go", Coupling: 85.0, SharedCommits: 20},
						},
					},
				},
			},
			nonEmptyMetrics: []string{"churn_rate", "temporal_coupling_pct", "author_fragmentation", "hotspot_concentration"},
			emptyMetrics:    []string{"commit_stability"},
			totalKeys:       5,
		},
		{
			name:     "C6 - Testing with violations",
			category: "C6",
			ar: &types.AnalysisResult{
				Name:     "testing",
				Category: "C6",
				Metrics: map[string]interface{}{
					"c6": &types.C6Metrics{
						TestFileCount:    5,
						SourceFileCount:  20,
						TestToCodeRatio:  0.3,
						CoveragePercent:  30.0,
						TestIsolation:    50.0,
						AssertionDensity: types.MetricSummary{Avg: 1.0},
						TestFunctions: []types.TestFunctionMetric{
							{Name: "TestA", File: "pkg/a_test.go", Line: 10, AssertionCount: 0, HasExternalDep: true},
							{Name: "TestB", File: "pkg/b_test.go", Line: 20, AssertionCount: 1, HasExternalDep: false},
						},
					},
				},
			},
			nonEmptyMetrics: []string{"test_isolation", "assertion_density_avg"},
			emptyMetrics:    []string{"test_to_code_ratio", "coverage_percent", "test_file_ratio"},
			totalKeys:       5,
		},
		{
			name:     "C7 - Agent Evaluation (score-based, no file-level evidence)",
			category: "C7",
			ar: &types.AnalysisResult{
				Name:     "agent-evaluation",
				Category: "C7",
				Metrics: map[string]interface{}{
					"c7": &types.C7Metrics{
						Available:                      true,
						TaskExecutionConsistency:       7,
						CodeBehaviorComprehension:      6,
						CrossFileNavigation:            5,
						IdentifierInterpretability:     8,
						DocumentationAccuracyDetection: 4,
					},
				},
			},
			// C7 is score-based: all evidence arrays are present but empty
			nonEmptyMetrics: []string{},
			emptyMetrics:    []string{"task_execution_consistency", "code_behavior_comprehension", "cross_file_navigation", "identifier_interpretability", "documentation_accuracy_detection"},
			totalKeys:       5,
		},
	}

	extractors := map[string]metricExtractor{
		"C1": extractC1,
		"C2": extractC2,
		"C3": extractC3,
		"C4": extractC4,
		"C5": extractC5,
		"C6": extractC6,
		"C7": extractC7,
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extractor := extractors[tt.category]
			_, _, evidence := extractor(tt.ar)

			// Evidence map must be non-nil
			if evidence == nil {
				t.Fatal("evidence map must not be nil")
			}

			// Verify total number of metric keys
			if len(evidence) != tt.totalKeys {
				t.Errorf("evidence map has %d keys, want %d; keys: %v", len(evidence), tt.totalKeys, evidenceKeys(evidence))
			}

			// Verify non-empty evidence metrics
			for _, metricKey := range tt.nonEmptyMetrics {
				items, ok := evidence[metricKey]
				if !ok {
					t.Errorf("evidence missing key %q", metricKey)
					continue
				}
				if items == nil {
					t.Errorf("evidence[%q] is nil, want non-nil slice", metricKey)
					continue
				}
				if len(items) == 0 {
					t.Errorf("evidence[%q] is empty, want non-empty for violated metric", metricKey)
					continue
				}
				// Validate each evidence item has required fields
				for i, item := range items {
					if item.FilePath == "" {
						t.Errorf("evidence[%q][%d].FilePath is empty", metricKey, i)
					}
					if item.Description == "" {
						t.Errorf("evidence[%q][%d].Description is empty", metricKey, i)
					}
				}
			}

			// Verify empty evidence metrics: non-nil but len == 0
			for _, metricKey := range tt.emptyMetrics {
				items, ok := evidence[metricKey]
				if !ok {
					t.Errorf("evidence missing key %q", metricKey)
					continue
				}
				if items == nil {
					t.Errorf("evidence[%q] is nil, want empty slice []EvidenceItem{}", metricKey)
					continue
				}
				if len(items) != 0 {
					t.Errorf("evidence[%q] has %d items, want 0 for metric without file-level violations", metricKey, len(items))
				}
			}
		})
	}
}

// evidenceKeys returns the keys of an evidence map for diagnostic output.
func evidenceKeys(evidence map[string][]types.EvidenceItem) []string {
	keys := make([]string, 0, len(evidence))
	for k := range evidence {
		keys = append(keys, k)
	}
	return keys
}

// --- Custom config test ---

func TestScoreC1_CustomConfig(t *testing.T) {
	// Create a config with very different C1 complexity breakpoints
	customCfg := DefaultConfig()
	// Replace complexity_avg breakpoints: now {1->1, 5->5, 10->10}
	// With default config, complexity 3.0 -> interpolates between (1,10) and (5,8) -> ~9.0
	// With custom config, complexity 3.0 -> interpolates between (1,1) and (5,5) -> 3.0
	c1 := customCfg.Categories["C1"]
	for i, m := range c1.Metrics {
		if m.Name == "complexity_avg" {
			c1.Metrics[i].Breakpoints = []Breakpoint{
				{Value: 1, Score: 1},
				{Value: 5, Score: 5},
				{Value: 10, Score: 10},
			}
			break
		}
	}
	customCfg.Categories["C1"] = c1

	defaultScorer := &Scorer{Config: DefaultConfig()}
	customScorer := &Scorer{Config: customCfg}

	ar := makeHealthyC1() // complexity avg = 3.0

	defaultResult := scoreCategory(defaultScorer, ar)
	customResult := scoreCategory(customScorer, ar)

	// Find the complexity sub-score in each
	var defaultComplexityScore, customComplexityScore float64
	for _, ss := range defaultResult.SubScores {
		if ss.MetricName == "complexity_avg" {
			defaultComplexityScore = ss.Score
		}
	}
	for _, ss := range customResult.SubScores {
		if ss.MetricName == "complexity_avg" {
			customComplexityScore = ss.Score
		}
	}

	// They must differ -- proves config wiring works
	if math.Abs(defaultComplexityScore-customComplexityScore) < 0.5 {
		t.Errorf("custom config should produce different complexity score: default=%v custom=%v",
			defaultComplexityScore, customComplexityScore)
	}

	// Overall category scores should also differ
	if math.Abs(defaultResult.Score-customResult.Score) < 0.1 {
		t.Errorf("custom config should produce different C1 score: default=%v custom=%v",
			defaultResult.Score, customResult.Score)
	}
}
