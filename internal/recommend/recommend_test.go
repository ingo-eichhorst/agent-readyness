package recommend

import (
	"math"
	"testing"

	"github.com/ingo/agent-readyness/internal/scoring"
	"github.com/ingo/agent-readyness/pkg/types"
)

// helper to build a scored result with known values for testing.
func buildScoredResult(cfg *scoring.ScoringConfig) *types.ScoredResult {
	// Build a mediocre project: complexity_avg=12 (score ~5.4), func_length_avg=35 (score ~5.5),
	// file_size_avg=200 (score ~7.3), afferent_coupling_avg=6 (score ~5.3),
	// efferent_coupling_avg=3 (score ~7.2), duplication_rate=10 (score ~4.3)
	c1Cfg := cfg.Categories["C1"]
	c1Sub := []types.SubScore{
		{MetricName: "complexity_avg", RawValue: 12, Score: scoring.Interpolate(c1Cfg.Metrics[0].Breakpoints, 12), Weight: 0.25, Available: true},
		{MetricName: "func_length_avg", RawValue: 35, Score: scoring.Interpolate(c1Cfg.Metrics[1].Breakpoints, 35), Weight: 0.20, Available: true},
		{MetricName: "file_size_avg", RawValue: 200, Score: scoring.Interpolate(c1Cfg.Metrics[2].Breakpoints, 200), Weight: 0.15, Available: true},
		{MetricName: "afferent_coupling_avg", RawValue: 6, Score: scoring.Interpolate(c1Cfg.Metrics[3].Breakpoints, 6), Weight: 0.15, Available: true},
		{MetricName: "efferent_coupling_avg", RawValue: 3, Score: scoring.Interpolate(c1Cfg.Metrics[4].Breakpoints, 3), Weight: 0.10, Available: true},
		{MetricName: "duplication_rate", RawValue: 10, Score: scoring.Interpolate(c1Cfg.Metrics[5].Breakpoints, 10), Weight: 0.15, Available: true},
	}

	// C3: max_dir_depth=4 (~7.0), module_fanout_avg=4 (~7.3), circular_deps=0 (10),
	// import_complexity_avg=3 (~7.0), dead_exports=8 (~7.6)
	c3Cfg := cfg.Categories["C3"]
	c3Sub := []types.SubScore{
		{MetricName: "max_dir_depth", RawValue: 4, Score: scoring.Interpolate(c3Cfg.Metrics[0].Breakpoints, 4), Weight: 0.20, Available: true},
		{MetricName: "module_fanout_avg", RawValue: 4, Score: scoring.Interpolate(c3Cfg.Metrics[1].Breakpoints, 4), Weight: 0.20, Available: true},
		{MetricName: "circular_deps", RawValue: 0, Score: scoring.Interpolate(c3Cfg.Metrics[2].Breakpoints, 0), Weight: 0.25, Available: true},
		{MetricName: "import_complexity_avg", RawValue: 3, Score: scoring.Interpolate(c3Cfg.Metrics[3].Breakpoints, 3), Weight: 0.15, Available: true},
		{MetricName: "dead_exports", RawValue: 8, Score: scoring.Interpolate(c3Cfg.Metrics[4].Breakpoints, 8), Weight: 0.20, Available: true},
	}

	// C6: test_to_code_ratio=0.3 (~4.8), coverage_percent=40 (score ~5.0),
	// test_isolation=50 (~5.0), assertion_density_avg=1.5 (~5.0), test_file_ratio=0.4 (~4.7)
	c6Cfg := cfg.Categories["C6"]
	c6Sub := []types.SubScore{
		{MetricName: "test_to_code_ratio", RawValue: 0.3, Score: scoring.Interpolate(c6Cfg.Metrics[0].Breakpoints, 0.3), Weight: 0.25, Available: true},
		{MetricName: "coverage_percent", RawValue: 40, Score: scoring.Interpolate(c6Cfg.Metrics[1].Breakpoints, 40), Weight: 0.30, Available: true},
		{MetricName: "test_isolation", RawValue: 50, Score: scoring.Interpolate(c6Cfg.Metrics[2].Breakpoints, 50), Weight: 0.15, Available: true},
		{MetricName: "assertion_density_avg", RawValue: 1.5, Score: scoring.Interpolate(c6Cfg.Metrics[3].Breakpoints, 1.5), Weight: 0.15, Available: true},
		{MetricName: "test_file_ratio", RawValue: 0.4, Score: scoring.Interpolate(c6Cfg.Metrics[4].Breakpoints, 0.4), Weight: 0.15, Available: true},
	}

	// Compute category scores as weighted avg of available sub-scores
	c1Score := weightedAvg(c1Sub)
	c3Score := weightedAvg(c3Sub)
	c6Score := weightedAvg(c6Sub)

	cats := []types.CategoryScore{
		{Name: "C1", Score: c1Score, Weight: 0.25, SubScores: c1Sub},
		{Name: "C3", Score: c3Score, Weight: 0.20, SubScores: c3Sub},
		{Name: "C6", Score: c6Score, Weight: 0.15, SubScores: c6Sub},
	}

	composite := computeTestComposite(cats)

	return &types.ScoredResult{
		Categories: cats,
		Composite:  composite,
		Tier:       "Agent-Limited",
	}
}

func weightedAvg(subs []types.SubScore) float64 {
	totalW := 0.0
	sum := 0.0
	for _, s := range subs {
		if !s.Available {
			continue
		}
		sum += s.Score * s.Weight
		totalW += s.Weight
	}
	if totalW == 0 {
		return 5.0
	}
	return sum / totalW
}

func computeTestComposite(cats []types.CategoryScore) float64 {
	totalW := 0.0
	sum := 0.0
	for _, c := range cats {
		if c.Score < 0 {
			continue
		}
		sum += c.Score * c.Weight
		totalW += c.Weight
	}
	if totalW == 0 {
		return 0
	}
	return sum / totalW
}

func approxEqual(a, b, tolerance float64) bool {
	return math.Abs(a-b) < tolerance
}

func TestGenerate_BasicRanking(t *testing.T) {
	cfg := scoring.DefaultConfig()
	scored := buildScoredResult(cfg)

	recs := Generate(scored, cfg)

	if len(recs) == 0 {
		t.Fatal("expected recommendations, got none")
	}
	if len(recs) > 5 {
		t.Errorf("expected at most 5 recommendations, got %d", len(recs))
	}

	// Recommendations must be sorted by ScoreImprovement descending
	for i := 1; i < len(recs); i++ {
		if recs[i].ScoreImprovement > recs[i-1].ScoreImprovement {
			t.Errorf("recommendations not sorted: rec[%d].ScoreImprovement=%.4f > rec[%d].ScoreImprovement=%.4f",
				i, recs[i].ScoreImprovement, i-1, recs[i-1].ScoreImprovement)
		}
	}

	// Ranks must be 1-based and sequential
	for i, rec := range recs {
		if rec.Rank != i+1 {
			t.Errorf("rec[%d].Rank = %d, want %d", i, rec.Rank, i+1)
		}
	}
}

func TestGenerate_ImpactAccuracy(t *testing.T) {
	cfg := scoring.DefaultConfig()
	scored := buildScoredResult(cfg)

	recs := Generate(scored, cfg)
	if len(recs) == 0 {
		t.Fatal("expected recommendations, got none")
	}

	// Each recommendation's ScoreImprovement must be positive
	for _, rec := range recs {
		if rec.ScoreImprovement <= 0 {
			t.Errorf("rec %q ScoreImprovement=%.4f, want > 0", rec.MetricName, rec.ScoreImprovement)
		}
	}

	// Manually verify the top recommendation's impact:
	// Simulate improving the top rec's metric to its target value
	top := recs[0]
	simulated := simulateManually(scored, cfg, top.Category, top.MetricName, top.TargetValue)
	expectedImprovement := simulated - scored.Composite

	if !approxEqual(top.ScoreImprovement, expectedImprovement, 0.01) {
		t.Errorf("top recommendation impact: got %.4f, want %.4f (manual simulation)",
			top.ScoreImprovement, expectedImprovement)
	}
}

// simulateManually recomputes composite with one metric improved.
func simulateManually(scored *types.ScoredResult, cfg *scoring.ScoringConfig,
	catName, metricName string, newRawValue float64) float64 {

	cats := make([]types.CategoryScore, len(scored.Categories))
	for i, c := range scored.Categories {
		cats[i] = c
		cats[i].SubScores = make([]types.SubScore, len(c.SubScores))
		copy(cats[i].SubScores, c.SubScores)
	}

	// Find the metric and update its score
	for ci := range cats {
		if cats[ci].Name != catName {
			continue
		}
		catCfg := getCatConfig(cfg, catName)
		for si := range cats[ci].SubScores {
			if cats[ci].SubScores[si].MetricName != metricName {
				continue
			}
			mt := findMetricCfg(catCfg, metricName)
			cats[ci].SubScores[si].Score = scoring.Interpolate(mt.Breakpoints, newRawValue)
			cats[ci].SubScores[si].RawValue = newRawValue
		}
		// Recompute category score
		cats[ci].Score = weightedAvg(cats[ci].SubScores)
	}

	return computeTestComposite(cats)
}

func getCatConfig(cfg *scoring.ScoringConfig, name string) *scoring.CategoryConfig {
	cat, ok := cfg.Categories[name]
	if !ok {
		return nil
	}
	return &cat
}

func findMetricCfg(cat *scoring.CategoryConfig, name string) *scoring.MetricThresholds {
	for i := range cat.Metrics {
		if cat.Metrics[i].Name == name {
			return &cat.Metrics[i]
		}
	}
	return nil
}

func TestGenerate_EffortEstimation(t *testing.T) {
	cfg := scoring.DefaultConfig()
	scored := buildScoredResult(cfg)

	recs := Generate(scored, cfg)
	if len(recs) == 0 {
		t.Fatal("expected recommendations")
	}

	for _, rec := range recs {
		switch rec.Effort {
		case "Low", "Medium", "High":
			// valid
		default:
			t.Errorf("rec %q has invalid effort %q", rec.MetricName, rec.Effort)
		}
	}
}

func TestGenerate_EffortDifficultyBump(t *testing.T) {
	// complexity_avg and duplication_rate should get a +1 level bump
	cfg := scoring.DefaultConfig()

	// Build a result where complexity_avg has a small gap (<1 point)
	// Normally small gap = "Low" but with bump should be "Medium"
	c1Sub := []types.SubScore{
		{MetricName: "complexity_avg", RawValue: 5, Score: 8.0, Weight: 0.25, Available: true},
		{MetricName: "func_length_avg", RawValue: 5, Score: 10.0, Weight: 0.20, Available: true},
		{MetricName: "file_size_avg", RawValue: 50, Score: 10.0, Weight: 0.15, Available: true},
		{MetricName: "afferent_coupling_avg", RawValue: 0, Score: 10.0, Weight: 0.15, Available: true},
		{MetricName: "efferent_coupling_avg", RawValue: 0, Score: 10.0, Weight: 0.10, Available: true},
		{MetricName: "duplication_rate", RawValue: 3, Score: 8.0, Weight: 0.15, Available: true},
	}
	c1Score := weightedAvg(c1Sub)
	cats := []types.CategoryScore{
		{Name: "C1", Score: c1Score, Weight: 0.25, SubScores: c1Sub},
	}
	scored := &types.ScoredResult{
		Categories: cats,
		Composite:  computeTestComposite(cats),
		Tier:       "Agent-Ready",
	}

	recs := Generate(scored, cfg)

	// Find the complexity_avg recommendation if present
	for _, rec := range recs {
		if rec.MetricName == "complexity_avg" {
			// Score gap to next breakpoint should be small, but difficulty bump
			// should push effort up by one level
			if rec.Effort == "Low" {
				t.Errorf("complexity_avg should have difficulty bump, got effort %q", rec.Effort)
			}
		}
		if rec.MetricName == "duplication_rate" {
			if rec.Effort == "Low" {
				t.Errorf("duplication_rate should have difficulty bump, got effort %q", rec.Effort)
			}
		}
	}
}

func TestGenerate_EmptyInput(t *testing.T) {
	cfg := scoring.DefaultConfig()

	// Empty scored result (no categories)
	scored := &types.ScoredResult{
		Categories: nil,
		Composite:  0,
		Tier:       "Agent-Hostile",
	}

	recs := Generate(scored, cfg)
	if len(recs) != 0 {
		t.Errorf("expected 0 recommendations for empty input, got %d", len(recs))
	}
}

func TestGenerate_AllExcellent(t *testing.T) {
	cfg := scoring.DefaultConfig()

	// All metrics score >= 9.0
	c1Sub := []types.SubScore{
		{MetricName: "complexity_avg", RawValue: 1, Score: 10.0, Weight: 0.25, Available: true},
		{MetricName: "func_length_avg", RawValue: 5, Score: 10.0, Weight: 0.20, Available: true},
		{MetricName: "file_size_avg", RawValue: 50, Score: 10.0, Weight: 0.15, Available: true},
		{MetricName: "afferent_coupling_avg", RawValue: 0, Score: 10.0, Weight: 0.15, Available: true},
		{MetricName: "efferent_coupling_avg", RawValue: 0, Score: 10.0, Weight: 0.10, Available: true},
		{MetricName: "duplication_rate", RawValue: 0, Score: 10.0, Weight: 0.15, Available: true},
	}

	cats := []types.CategoryScore{
		{Name: "C1", Score: 10.0, Weight: 0.25, SubScores: c1Sub},
	}

	scored := &types.ScoredResult{
		Categories: cats,
		Composite:  10.0,
		Tier:       "Agent-Ready",
	}

	recs := Generate(scored, cfg)
	if len(recs) != 0 {
		t.Errorf("expected 0 recommendations for all-excellent scores, got %d", len(recs))
	}
}

func TestGenerate_SkipsUnavailableMetrics(t *testing.T) {
	cfg := scoring.DefaultConfig()

	c6Sub := []types.SubScore{
		{MetricName: "test_to_code_ratio", RawValue: 0.1, Score: 2.0, Weight: 0.25, Available: true},
		{MetricName: "coverage_percent", RawValue: -1, Score: 0, Weight: 0.30, Available: false},
		{MetricName: "test_isolation", RawValue: 30, Score: 3.0, Weight: 0.15, Available: true},
		{MetricName: "assertion_density_avg", RawValue: 0.5, Score: 2.5, Weight: 0.15, Available: true},
		{MetricName: "test_file_ratio", RawValue: 0.2, Score: 3.0, Weight: 0.15, Available: true},
	}

	cats := []types.CategoryScore{
		{Name: "C6", Score: weightedAvg(c6Sub), Weight: 0.15, SubScores: c6Sub},
	}

	scored := &types.ScoredResult{
		Categories: cats,
		Composite:  computeTestComposite(cats),
		Tier:       "Agent-Hostile",
	}

	recs := Generate(scored, cfg)

	for _, rec := range recs {
		if rec.MetricName == "coverage_percent" {
			t.Error("should not recommend unavailable metric coverage_percent")
		}
	}
}

func TestGenerate_FewerThan5(t *testing.T) {
	cfg := scoring.DefaultConfig()

	// Only 2 improvable metrics (rest are excellent)
	c1Sub := []types.SubScore{
		{MetricName: "complexity_avg", RawValue: 12, Score: 5.4, Weight: 0.25, Available: true},
		{MetricName: "func_length_avg", RawValue: 35, Score: 5.5, Weight: 0.20, Available: true},
		{MetricName: "file_size_avg", RawValue: 50, Score: 10.0, Weight: 0.15, Available: true},
		{MetricName: "afferent_coupling_avg", RawValue: 0, Score: 10.0, Weight: 0.15, Available: true},
		{MetricName: "efferent_coupling_avg", RawValue: 0, Score: 10.0, Weight: 0.10, Available: true},
		{MetricName: "duplication_rate", RawValue: 0, Score: 10.0, Weight: 0.15, Available: true},
	}

	cats := []types.CategoryScore{
		{Name: "C1", Score: weightedAvg(c1Sub), Weight: 0.25, SubScores: c1Sub},
	}

	scored := &types.ScoredResult{
		Categories: cats,
		Composite:  computeTestComposite(cats),
		Tier:       "Agent-Assisted",
	}

	recs := Generate(scored, cfg)
	if len(recs) > 2 {
		t.Errorf("expected at most 2 recommendations, got %d", len(recs))
	}
	if len(recs) < 1 {
		t.Error("expected at least 1 recommendation")
	}
}

func TestGenerate_NilConfig(t *testing.T) {
	cfg := scoring.DefaultConfig()
	scored := buildScoredResult(cfg)

	// Should use DefaultConfig when nil passed
	recs := Generate(scored, nil)
	if len(recs) == 0 {
		t.Error("expected recommendations with nil config (should use defaults)")
	}
}

func TestGenerate_AgentReadinessSummary(t *testing.T) {
	cfg := scoring.DefaultConfig()
	scored := buildScoredResult(cfg)

	recs := Generate(scored, cfg)
	if len(recs) == 0 {
		t.Fatal("expected recommendations")
	}

	for _, rec := range recs {
		if rec.Summary == "" {
			t.Errorf("rec %q has empty Summary", rec.MetricName)
		}
		if rec.Action == "" {
			t.Errorf("rec %q has empty Action", rec.MetricName)
		}
		if rec.Category == "" {
			t.Errorf("rec %q has empty Category", rec.MetricName)
		}
	}
}

func TestGenerate_TargetValues(t *testing.T) {
	cfg := scoring.DefaultConfig()
	scored := buildScoredResult(cfg)

	recs := Generate(scored, cfg)
	if len(recs) == 0 {
		t.Fatal("expected recommendations")
	}

	for _, rec := range recs {
		// Target score should be higher than current score
		if rec.TargetScore <= rec.CurrentScore {
			t.Errorf("rec %q: target score %.2f should be > current score %.2f",
				rec.MetricName, rec.TargetScore, rec.CurrentScore)
		}
	}
}
