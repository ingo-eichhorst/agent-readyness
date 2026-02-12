package output

import (
	"strings"
	"testing"

	"github.com/ingo-eichhorst/agent-readyness/internal/scoring"
	"github.com/ingo-eichhorst/agent-readyness/pkg/types"
)

func TestRenderImprovementPrompt_C1Metric(t *testing.T) {
	params := promptParams{
		CategoryName:    "C1",
		CategoryDisplay: "C1: Code Health",
		CategoryImpact:  "Lower complexity and smaller functions help agents reason about and modify code safely.",
		MetricName:      "complexity_avg",
		MetricDisplay:   "Complexity avg",
		RawValue:        18.3,
		FormattedValue:  "18.3",
		Score:           4.2,
		TargetScore:     6.0,
		TargetValue:     10,
		HasBreakpoints:  true,
		Evidence: []types.EvidenceItem{
			{FilePath: "internal/parser/go.go", Line: 42, Value: 35.0, Description: "high complexity function"},
			{FilePath: "internal/analyzer/c1.go", Line: 100, Value: 28.0, Description: "complex switch statement"},
			{FilePath: "cmd/root.go", Line: 15, Value: 22.0, Description: "deeply nested conditionals"},
		},
		Language: "go",
	}

	result := renderImprovementPrompt(params)

	if result == "" {
		t.Fatal("expected non-empty result")
	}

	checks := []string{
		"## Context",
		"## Build &amp; Test Commands",
		"## Task",
		"## Verification",
		"4.2/10",
		"go test ./...",
		"internal/parser/go.go",
		"internal/analyzer/c1.go",
		"cmd/root.go",
	}
	for _, want := range checks {
		if !strings.Contains(result, want) {
			t.Errorf("result missing %q", want)
		}
	}
}

func TestRenderImprovementPrompt_NoEvidence(t *testing.T) {
	params := promptParams{
		CategoryName:    "C2",
		CategoryDisplay: "C2: Semantic Explicitness",
		CategoryImpact:  "Explicit types and consistent naming enable agents to understand code semantics without guessing.",
		MetricName:      "naming_consistency",
		MetricDisplay:   "Naming consistency",
		RawValue:        85.0,
		FormattedValue:  "85.0%",
		Score:           6.0,
		TargetScore:     8.0,
		TargetValue:     95,
		HasBreakpoints:  true,
		Evidence:        []types.EvidenceItem{},
		Language:        "python",
	}

	result := renderImprovementPrompt(params)

	if result == "" {
		t.Fatal("expected non-empty result")
	}

	// Must NOT contain "Files to Focus On" section
	if strings.Contains(result, "Files to Focus On") {
		t.Error("result should not contain 'Files to Focus On' when evidence is empty")
	}

	// Must still contain all 4 sections
	for _, section := range []string{"## Context", "## Build &amp; Test Commands", "## Task", "## Verification"} {
		if !strings.Contains(result, section) {
			t.Errorf("result missing section %q", section)
		}
	}
}

func TestRenderImprovementPrompt_C7Metric(t *testing.T) {
	params := promptParams{
		CategoryName:    "C7",
		CategoryDisplay: "C7: Agent Evaluation",
		CategoryImpact:  "Direct measurement of how well AI agents perform real-world coding tasks in your codebase.",
		MetricName:      "task_execution_consistency",
		MetricDisplay:   "Task Execution Consistency",
		RawValue:        6.0,
		FormattedValue:  "6.0",
		Score:           6.0,
		TargetScore:     0, // unused for C7
		TargetValue:     0, // unused for C7
		HasBreakpoints:  false,
		Evidence:        []types.EvidenceItem{},
		Language:        "go",
	}

	result := renderImprovementPrompt(params)

	if result == "" {
		t.Fatal("expected non-empty result")
	}

	// Should contain improvement guidance
	if !strings.Contains(result, "## Context") {
		t.Error("result missing Context section")
	}
	if !strings.Contains(result, "Improve score above") {
		t.Error("C7 prompt should contain 'Improve score above' target")
	}
}

func TestNextTarget_Descending(t *testing.T) {
	// Complexity breakpoints: higher value = lower score (descending)
	cfg := scoring.DefaultConfig()
	var breakpoints []scoring.Breakpoint
	for _, m := range cfg.Categories["C1"].Metrics {
		if m.Name == "complexity_avg" {
			breakpoints = m.Breakpoints
			break
		}
	}

	if len(breakpoints) == 0 {
		t.Fatal("complexity_avg breakpoints not found")
	}

	// Current score ~3 (between 20->3 and 40->1)
	targetValue, targetScore := nextTarget(3.0, breakpoints)

	if targetScore <= 3.0 {
		t.Errorf("expected target score > 3.0, got %.1f", targetScore)
	}
	// For descending breakpoints, target value should be lower (less complexity)
	if targetValue >= 20 {
		t.Errorf("expected target value < 20 for descending breakpoints, got %.4g", targetValue)
	}
}

func TestNextTarget_Ascending(t *testing.T) {
	// Coverage breakpoints: higher value = higher score (ascending)
	cfg := scoring.DefaultConfig()
	var breakpoints []scoring.Breakpoint
	for _, m := range cfg.Categories["C6"].Metrics {
		if m.Name == "coverage_percent" {
			breakpoints = m.Breakpoints
			break
		}
	}

	if len(breakpoints) == 0 {
		t.Fatal("coverage_percent breakpoints not found")
	}

	// Current score ~4 (between 30->4 and 50->6)
	targetValue, targetScore := nextTarget(4.0, breakpoints)

	if targetScore <= 4.0 {
		t.Errorf("expected target score > 4.0, got %.1f", targetScore)
	}
	// For ascending breakpoints, target value should be higher (more coverage)
	if targetValue <= 30 {
		t.Errorf("expected target value > 30 for ascending breakpoints, got %.4g", targetValue)
	}
}

func TestNextTarget_MaxScore(t *testing.T) {
	breakpoints := []scoring.Breakpoint{
		{Value: 0, Score: 1},
		{Value: 50, Score: 6},
		{Value: 100, Score: 10},
	}

	targetValue, targetScore := nextTarget(10.0, breakpoints)

	if targetScore != 10.0 {
		t.Errorf("expected target score 10.0, got %.1f", targetScore)
	}
	if targetValue != 100 {
		t.Errorf("expected target value 100, got %.4g", targetValue)
	}
}

func TestGetMetricTaskGuidance(t *testing.T) {
	result := getMetricTaskGuidance("complexity_avg", 18.3, 10, true)

	if result == "" {
		t.Fatal("expected non-empty guidance")
	}

	// Should contain improvement guidance (either from descriptions or generic)
	lower := strings.ToLower(result)
	if !strings.Contains(lower, "complexity") && !strings.Contains(lower, "improve") {
		t.Errorf("expected guidance to mention complexity or improvement, got: %s", result)
	}
}
