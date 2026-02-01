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

// --- categoryScore tests ---

func TestCategoryScore_AllAvailable(t *testing.T) {
	subs := []types.SubScore{
		{MetricName: "a", Score: 8, Weight: 0.5, Available: true},
		{MetricName: "b", Score: 6, Weight: 0.5, Available: true},
	}
	got := categoryScore(subs)
	// (8*0.5 + 6*0.5) / 1.0 = 7.0
	if math.Abs(got-7.0) > 0.01 {
		t.Errorf("categoryScore = %v, want 7.0", got)
	}
}

func TestCategoryScore_SkipUnavailable(t *testing.T) {
	subs := []types.SubScore{
		{MetricName: "a", Score: 8, Weight: 0.5, Available: true},
		{MetricName: "b", Score: 0, Weight: 0.5, Available: false},
	}
	got := categoryScore(subs)
	// Only 'a' contributes: 8*0.5 / 0.5 = 8.0
	if math.Abs(got-8.0) > 0.01 {
		t.Errorf("categoryScore = %v, want 8.0", got)
	}
}

func TestCategoryScore_Empty(t *testing.T) {
	got := categoryScore(nil)
	if got != 5.0 {
		t.Errorf("categoryScore(nil) = %v, want 5.0", got)
	}
}

func TestCategoryScore_AllUnavailable(t *testing.T) {
	subs := []types.SubScore{
		{MetricName: "a", Score: 0, Weight: 0.5, Available: false},
		{MetricName: "b", Score: 0, Weight: 0.5, Available: false},
	}
	got := categoryScore(subs)
	if got != 5.0 {
		t.Errorf("categoryScore(all unavailable) = %v, want 5.0", got)
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
