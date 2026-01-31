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
