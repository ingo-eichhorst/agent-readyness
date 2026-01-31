package recommend

import (
	"fmt"
	"sort"

	"github.com/ingo/agent-readyness/internal/scoring"
	"github.com/ingo/agent-readyness/pkg/types"
)

// Recommendation represents a single improvement recommendation.
type Recommendation struct {
	Rank             int     // 1-based rank
	Category         string  // e.g., "C1"
	MetricName       string  // e.g., "complexity_avg"
	CurrentValue     float64 // raw metric value
	CurrentScore     float64 // 1-10 metric score
	TargetValue      float64 // next breakpoint value that improves score
	TargetScore      float64 // what metric score would be at target
	ScoreImprovement float64 // estimated composite improvement
	Effort           string  // "Low", "Medium", "High"
	Summary          string  // agent-readiness framed description
	Action           string  // concrete improvement action
}

// agentImpact maps metric names to agent-readiness-focused impact descriptions.
var agentImpact = map[string]string{
	"complexity_avg":        "High complexity makes functions harder for agents to reason about and modify safely",
	"func_length_avg":       "Long functions exceed agent context windows, forcing partial understanding",
	"file_size_avg":         "Large files make it harder for agents to locate and navigate relevant code",
	"duplication_rate":      "Duplicated code means agents must find and update multiple locations",
	"afferent_coupling_avg": "High incoming coupling means agent changes risk breaking many dependents",
	"efferent_coupling_avg": "High outgoing coupling means agents must understand many dependencies",
	"max_dir_depth":         "Deep directory nesting makes project navigation harder for agents",
	"module_fanout_avg":     "High module coupling means agent changes ripple across many packages",
	"circular_deps":         "Circular dependencies confuse agent dependency analysis",
	"import_complexity_avg": "Complex imports make it harder for agents to understand module boundaries",
	"dead_exports":          "Dead exports clutter the API surface agents must understand",
	"test_to_code_ratio":    "Low test coverage means agents cannot verify their changes",
	"coverage_percent":      "Without test coverage data, agents cannot assess change safety",
	"test_isolation":        "Non-isolated tests create flaky failures that block agent workflows",
	"assertion_density_avg": "Low assertion density means tests may pass despite broken behavior",
	"test_file_ratio":       "Few test files means agents lack verification for most code paths",
}

// actionTemplates maps metric names to concrete improvement actions.
var actionTemplates = map[string]string{
	"complexity_avg":        "Refactor functions with cyclomatic complexity > %.0f into smaller units",
	"func_length_avg":       "Break up functions longer than %.0f lines into focused helpers",
	"file_size_avg":         "Split files larger than %.0f lines into cohesive modules",
	"duplication_rate":      "Extract duplicated code blocks (currently %.1f%% duplication) into shared functions",
	"afferent_coupling_avg": "Reduce incoming dependencies by introducing interfaces or facade patterns",
	"efferent_coupling_avg": "Reduce outgoing dependencies by applying dependency inversion",
	"max_dir_depth":         "Flatten directory structure from depth %d to at most %d",
	"module_fanout_avg":     "Reduce module fan-out by consolidating related imports",
	"circular_deps":         "Break circular dependencies by extracting shared interfaces",
	"import_complexity_avg": "Simplify imports by reducing average import count per file",
	"dead_exports":          "Remove %.0f unused exported symbols to reduce API surface",
	"test_to_code_ratio":    "Add tests to improve test-to-code ratio from %.2f to %.2f",
	"coverage_percent":      "Increase test coverage from %.0f%% to %.0f%%",
	"test_isolation":        "Improve test isolation from %.0f%% to %.0f%%",
	"assertion_density_avg": "Add meaningful assertions (current avg: %.1f per test)",
	"test_file_ratio":       "Add test files to cover more source files (current ratio: %.2f)",
}

// hardMetrics are metrics that get a +1 effort level bump because they are
// inherently harder to improve.
var hardMetrics = map[string]bool{
	"complexity_avg":   true,
	"duplication_rate": true,
}

// Generate analyzes scored results and returns up to 5 improvement
// recommendations ranked by composite score impact.
func Generate(scored *types.ScoredResult, cfg *scoring.ScoringConfig) []Recommendation {
	if cfg == nil {
		cfg = scoring.DefaultConfig()
	}
	if len(scored.Categories) == 0 {
		return nil
	}

	var candidates []Recommendation

	for _, cat := range scored.Categories {
		catCfg := getCategoryConfig(cfg, cat.Name)
		if catCfg == nil {
			continue
		}

		for _, ss := range cat.SubScores {
			if !ss.Available || ss.Score >= 9.0 {
				continue
			}

			mt := findMetric(catCfg, ss.MetricName)
			if mt == nil {
				continue
			}

			targetValue, targetScore := findTargetBreakpoint(mt.Breakpoints, ss.Score)
			if targetScore <= ss.Score {
				continue // no improvement possible
			}

			impact := simulateComposite(scored, cfg, cat.Name, ss.MetricName, targetValue) - scored.Composite
			if impact <= 0 {
				continue
			}

			effort := effortLevel(targetScore-ss.Score, ss.MetricName)

			rec := Recommendation{
				Category:         cat.Name,
				MetricName:       ss.MetricName,
				CurrentValue:     ss.RawValue,
				CurrentScore:     ss.Score,
				TargetValue:      targetValue,
				TargetScore:      targetScore,
				ScoreImprovement: impact,
				Effort:           effort,
				Summary:          buildSummary(ss.MetricName, ss.RawValue, targetValue),
				Action:           buildAction(ss.MetricName, ss.RawValue, targetValue),
			}
			candidates = append(candidates, rec)
		}
	}

	// Sort by impact descending
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].ScoreImprovement > candidates[j].ScoreImprovement
	})

	// Cap at 5
	if len(candidates) > 5 {
		candidates = candidates[:5]
	}

	// Assign ranks
	for i := range candidates {
		candidates[i].Rank = i + 1
	}

	return candidates
}

// getCategoryConfig returns the CategoryConfig for a given category name.
func getCategoryConfig(cfg *scoring.ScoringConfig, name string) *scoring.CategoryConfig {
	switch name {
	case "C1":
		return &cfg.C1
	case "C3":
		return &cfg.C3
	case "C6":
		return &cfg.C6
	}
	return nil
}

// findMetric finds a MetricThresholds by name in a category config.
func findMetric(cat *scoring.CategoryConfig, name string) *scoring.MetricThresholds {
	for i := range cat.Metrics {
		if cat.Metrics[i].Name == name {
			return &cat.Metrics[i]
		}
	}
	return nil
}

// findTargetBreakpoint finds the next breakpoint that would give a higher score
// than the current score. Returns the breakpoint's value and score.
// Breakpoints are sorted by Value ascending; Score direction varies.
func findTargetBreakpoint(breakpoints []scoring.Breakpoint, currentScore float64) (float64, float64) {
	if len(breakpoints) == 0 {
		return 0, currentScore
	}

	// We need the breakpoint whose score is the next step above currentScore.
	// Collect unique score levels from breakpoints, find the next one above current.
	type target struct {
		value float64
		score float64
	}

	// Find all breakpoints with score > currentScore, pick the one with the
	// lowest score (the minimal improvement step).
	var best *target
	for _, bp := range breakpoints {
		if bp.Score > currentScore {
			if best == nil || bp.Score < best.score {
				best = &target{value: bp.Value, score: bp.Score}
			}
		}
	}

	if best == nil {
		// No breakpoint offers a higher score
		return 0, currentScore
	}

	return best.value, best.score
}

// simulateComposite deep-copies the categories, patches one sub-score with a
// simulated improvement, recomputes the category score, and returns the new
// composite score.
func simulateComposite(scored *types.ScoredResult, cfg *scoring.ScoringConfig,
	catName, metricName string, newRawValue float64) float64 {

	cats := make([]types.CategoryScore, len(scored.Categories))
	for i, c := range scored.Categories {
		cats[i] = c
		cats[i].SubScores = make([]types.SubScore, len(c.SubScores))
		copy(cats[i].SubScores, c.SubScores)
	}

	for ci := range cats {
		if cats[ci].Name != catName {
			continue
		}
		catCfg := getCategoryConfig(cfg, catName)
		if catCfg == nil {
			break
		}
		for si := range cats[ci].SubScores {
			if cats[ci].SubScores[si].MetricName != metricName {
				continue
			}
			mt := findMetric(catCfg, metricName)
			if mt == nil {
				break
			}
			cats[ci].SubScores[si].Score = scoring.Interpolate(mt.Breakpoints, newRawValue)
			cats[ci].SubScores[si].RawValue = newRawValue
		}
		// Recompute category score
		cats[ci].Score = categoryScore(cats[ci].SubScores)
	}

	return computeComposite(cats)
}

// categoryScore computes weighted average of available sub-scores.
func categoryScore(subScores []types.SubScore) float64 {
	totalWeight := 0.0
	weightedSum := 0.0
	for _, ss := range subScores {
		if !ss.Available {
			continue
		}
		weightedSum += ss.Score * ss.Weight
		totalWeight += ss.Weight
	}
	if totalWeight == 0 {
		return 5.0
	}
	return weightedSum / totalWeight
}

// computeComposite calculates the weighted composite from category scores.
func computeComposite(categories []types.CategoryScore) float64 {
	totalWeight := 0.0
	weightedSum := 0.0
	for _, cat := range categories {
		if cat.Score < 0 {
			continue
		}
		weightedSum += cat.Score * cat.Weight
		totalWeight += cat.Weight
	}
	if totalWeight == 0 {
		return 0
	}
	return weightedSum / totalWeight
}

// effortLevel estimates improvement effort based on score gap and metric difficulty.
// Gap < 1.0: Low, gap < 2.5: Medium, gap >= 2.5: High.
// Hard metrics (complexity_avg, duplication_rate) get bumped up one level.
func effortLevel(scoreGap float64, metricName string) string {
	level := 0 // 0=Low, 1=Medium, 2=High
	if scoreGap >= 2.5 {
		level = 2
	} else if scoreGap >= 1.0 {
		level = 1
	}

	if hardMetrics[metricName] {
		level++
		if level > 2 {
			level = 2
		}
	}

	switch level {
	case 0:
		return "Low"
	case 1:
		return "Medium"
	default:
		return "High"
	}
}

// displayName returns a human-friendly name for a metric.
var displayNames = map[string]string{
	"complexity_avg":        "average complexity",
	"func_length_avg":       "average function length",
	"file_size_avg":         "average file size",
	"duplication_rate":      "duplication rate",
	"afferent_coupling_avg": "average afferent coupling",
	"efferent_coupling_avg": "average efferent coupling",
	"max_dir_depth":         "max directory depth",
	"module_fanout_avg":     "average module fan-out",
	"circular_deps":         "circular dependencies",
	"import_complexity_avg": "average import complexity",
	"dead_exports":          "dead exports",
	"test_to_code_ratio":    "test-to-code ratio",
	"coverage_percent":      "test coverage",
	"test_isolation":        "test isolation",
	"assertion_density_avg": "average assertion density",
	"test_file_ratio":       "test file ratio",
}

// buildSummary creates an agent-readiness framed summary for a recommendation.
func buildSummary(metricName string, currentValue, targetValue float64) string {
	dn := displayNames[metricName]
	if dn == "" {
		dn = metricName
	}
	impact := agentImpact[metricName]
	if impact == "" {
		impact = "Improving this metric enhances agent effectiveness"
	}
	return fmt.Sprintf("Improve %s from %.1f to %.1f -- %s", dn, currentValue, targetValue, impact)
}

// buildAction creates a concrete improvement action string.
func buildAction(metricName string, currentValue, targetValue float64) string {
	tmpl, ok := actionTemplates[metricName]
	if !ok {
		return fmt.Sprintf("Improve %s from %.1f to %.1f", metricName, currentValue, targetValue)
	}

	switch metricName {
	case "complexity_avg", "func_length_avg", "file_size_avg":
		return fmt.Sprintf(tmpl, currentValue)
	case "duplication_rate":
		return fmt.Sprintf(tmpl, currentValue)
	case "max_dir_depth":
		return fmt.Sprintf(tmpl, int(currentValue), int(targetValue))
	case "dead_exports":
		return fmt.Sprintf(tmpl, currentValue)
	case "test_to_code_ratio":
		return fmt.Sprintf(tmpl, currentValue, targetValue)
	case "coverage_percent", "test_isolation":
		return fmt.Sprintf(tmpl, currentValue, targetValue)
	case "assertion_density_avg":
		return fmt.Sprintf(tmpl, currentValue)
	case "test_file_ratio":
		return fmt.Sprintf(tmpl, currentValue)
	default:
		return fmt.Sprintf(tmpl, currentValue, targetValue)
	}
}
