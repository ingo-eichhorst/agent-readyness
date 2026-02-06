package output

import (
	"bytes"
	"strings"
	"testing"

	"github.com/ingo/agent-readyness/internal/recommend"
	"github.com/ingo/agent-readyness/internal/scoring"
	"github.com/ingo/agent-readyness/pkg/types"
)

func TestNewHTMLGenerator(t *testing.T) {
	gen, err := NewHTMLGenerator()
	if err != nil {
		t.Fatalf("NewHTMLGenerator() error = %v", err)
	}
	if gen == nil {
		t.Error("NewHTMLGenerator() returned nil")
	}
	if gen.tmpl == nil {
		t.Error("NewHTMLGenerator() template is nil")
	}
}

func TestHTMLGenerator_GenerateReport(t *testing.T) {
	gen, err := NewHTMLGenerator()
	if err != nil {
		t.Fatalf("NewHTMLGenerator() error = %v", err)
	}

	scored := &types.ScoredResult{
		ProjectName: "test-project",
		Composite:   7.5,
		Tier:        "Agent-Assisted",
		Categories: []types.CategoryScore{
			{
				Name:   "C1",
				Score:  7.5,
				Weight: 0.20,
				SubScores: []types.SubScore{
					{MetricName: "complexity_avg", RawValue: 8.5, Score: 7.0, Weight: 0.30, Available: true},
					{MetricName: "func_length_avg", RawValue: 25.0, Score: 8.0, Weight: 0.25, Available: true},
				},
			},
			{
				Name:   "C2",
				Score:  8.2,
				Weight: 0.15,
				SubScores: []types.SubScore{
					{MetricName: "type_annotation_coverage", RawValue: 85.0, Score: 8.5, Weight: 0.35, Available: true},
				},
			},
			{
				Name:   "C3",
				Score:  7.0,
				Weight: 0.15,
				SubScores: []types.SubScore{
					{MetricName: "max_dir_depth", RawValue: 4.0, Score: 8.0, Weight: 0.20, Available: true},
				},
			},
		},
	}

	recs := []recommend.Recommendation{
		{
			Rank:             1,
			Summary:          "Reduce average complexity",
			ScoreImprovement: 0.5,
			Effort:           "Medium",
			Action:           "Refactor complex functions",
		},
	}

	var buf bytes.Buffer
	err = gen.GenerateReport(&buf, scored, recs, nil, nil)
	if err != nil {
		t.Fatalf("GenerateReport() error = %v", err)
	}

	html := buf.String()

	// Basic structure checks
	if !strings.Contains(html, "<!DOCTYPE html>") {
		t.Error("GenerateReport() missing DOCTYPE")
	}
	if !strings.Contains(html, "test-project") {
		t.Error("GenerateReport() missing project name")
	}
	if !strings.Contains(html, "7.5") {
		t.Error("GenerateReport() missing composite score")
	}
	if !strings.Contains(html, "Agent-Assisted") {
		t.Error("GenerateReport() missing tier")
	}
	if !strings.Contains(html, "<svg") {
		t.Error("GenerateReport() missing radar chart SVG")
	}
	if !strings.Contains(html, "C1: Code Health") {
		t.Error("GenerateReport() missing category display name")
	}
	if !strings.Contains(html, "category-citations") {
		t.Error("GenerateReport() missing per-category citations")
	}
}

func TestHTMLReport_ContainsModalComponent(t *testing.T) {
	gen, err := NewHTMLGenerator()
	if err != nil {
		t.Fatalf("NewHTMLGenerator() error = %v", err)
	}

	scored := &types.ScoredResult{
		ProjectName: "modal-test",
		Composite:   7.0,
		Tier:        "Agent-Assisted",
		Categories: []types.CategoryScore{
			{
				Name:   "C1",
				Score:  7.0,
				Weight: 0.20,
				SubScores: []types.SubScore{
					{MetricName: "complexity_avg", RawValue: 8.0, Score: 7.0, Weight: 0.30, Available: true},
				},
			},
		},
	}

	var buf bytes.Buffer
	err = gen.GenerateReport(&buf, scored, nil, nil, nil)
	if err != nil {
		t.Fatalf("GenerateReport() error = %v", err)
	}

	html := buf.String()

	checks := []struct {
		substring string
		desc      string
	}{
		{`<dialog id="ars-modal"`, "generated HTML should contain dialog element"},
		{`class="ars-modal-close"`, "generated HTML should contain modal close button"},
		{"openModal", "generated HTML should define openModal function"},
		{"closeModal", "generated HTML should define closeModal function"},
		{"showModal()", "generated HTML should use showModal() for native dialog"},
		{"<noscript>", "generated HTML should contain noscript progressive enhancement"},
		{"ars-modal-trigger", "generated HTML should contain modal trigger button styles"},
	}

	for _, c := range checks {
		if !strings.Contains(html, c.substring) {
			t.Errorf("%s (missing %q)", c.desc, c.substring)
		}
	}
}

func TestHTMLGenerator_XSSPrevention(t *testing.T) {
	gen, err := NewHTMLGenerator()
	if err != nil {
		t.Fatalf("NewHTMLGenerator() error = %v", err)
	}

	// Malicious project name
	scored := &types.ScoredResult{
		ProjectName: "<script>alert('XSS')</script>",
		Composite:   7.5,
		Tier:        "Agent-Assisted",
		Categories: []types.CategoryScore{
			{Name: "C1", Score: 7.5, Weight: 0.20},
			{Name: "C2", Score: 8.0, Weight: 0.15},
			{Name: "C3", Score: 7.0, Weight: 0.15},
		},
	}

	var buf bytes.Buffer
	err = gen.GenerateReport(&buf, scored, nil, nil, nil)
	if err != nil {
		t.Fatalf("GenerateReport() error = %v", err)
	}

	html := buf.String()

	// Should NOT contain unescaped script tag
	if strings.Contains(html, "<script>alert") {
		t.Error("XSS vulnerability: script tag not escaped")
	}

	// Should contain escaped version
	if !strings.Contains(html, "&lt;script&gt;") {
		t.Error("XSS: script tag should be HTML-escaped")
	}
}

func TestHTMLGenerator_WithBaseline(t *testing.T) {
	gen, err := NewHTMLGenerator()
	if err != nil {
		t.Fatalf("NewHTMLGenerator() error = %v", err)
	}

	current := &types.ScoredResult{
		ProjectName: "test-project",
		Composite:   7.5,
		Tier:        "Agent-Assisted",
		Categories: []types.CategoryScore{
			{Name: "C1", Score: 7.5, Weight: 0.20},
			{Name: "C2", Score: 8.2, Weight: 0.15},
			{Name: "C3", Score: 7.0, Weight: 0.15},
		},
	}

	baseline := &types.ScoredResult{
		ProjectName: "test-project",
		Composite:   6.5,
		Tier:        "Agent-Limited",
		Categories: []types.CategoryScore{
			{Name: "C1", Score: 6.0, Weight: 0.20},
			{Name: "C2", Score: 7.0, Weight: 0.15},
			{Name: "C3", Score: 6.5, Weight: 0.15},
		},
	}

	var buf bytes.Buffer
	err = gen.GenerateReport(&buf, current, nil, baseline, nil)
	if err != nil {
		t.Fatalf("GenerateReport() error = %v", err)
	}

	html := buf.String()

	// Should contain trend comparison section
	if !strings.Contains(html, "Score Comparison") {
		t.Error("GenerateReport() with baseline missing trend section")
	}
}

func TestTierToClass(t *testing.T) {
	tests := []struct {
		tier  string
		class string
	}{
		{"Agent-Ready", "ready"},
		{"Agent-Assisted", "assisted"},
		{"Agent-Limited", "limited"},
		{"Agent-Hostile", "hostile"},
		{"Unknown", "hostile"},
	}

	for _, tt := range tests {
		t.Run(tt.tier, func(t *testing.T) {
			got := tierToClass(tt.tier)
			if got != tt.class {
				t.Errorf("tierToClass(%q) = %q, want %q", tt.tier, got, tt.class)
			}
		})
	}
}

func TestScoreToClass(t *testing.T) {
	tests := []struct {
		score float64
		class string
	}{
		{9.0, "ready"},
		{8.0, "ready"},
		{7.9, "assisted"},
		{6.0, "assisted"},
		{5.9, "limited"},
		{3.0, "limited"},
	}

	for _, tt := range tests {
		got := scoreToClass(tt.score)
		if got != tt.class {
			t.Errorf("scoreToClass(%.1f) = %q, want %q", tt.score, got, tt.class)
		}
	}
}

func TestFormatMetricValue(t *testing.T) {
	tests := []struct {
		name      string
		value     float64
		available bool
		want      string
	}{
		{"complexity_avg", 8.5, true, "8.5"},
		{"duplication_rate", 15.3, true, "15.3%"},
		{"test_to_code_ratio", 0.45, true, "0.45"},
		{"max_dir_depth", 5.0, true, "5"},
		{"changelog_present", 1.0, true, "yes"},
		{"changelog_present", 0.0, true, "no"},
		{"unknown_metric", 7.5, true, "7.5"},
		{"complexity_avg", 0.0, false, "n/a"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatMetricValue(tt.name, tt.value, tt.available)
			if got != tt.want {
				t.Errorf("formatMetricValue(%q, %.1f, %v) = %q, want %q",
					tt.name, tt.value, tt.available, got, tt.want)
			}
		})
	}
}

func TestCategoryDisplayName(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"C1", "C1: Code Health"},
		{"C2", "C2: Semantic Explicitness"},
		{"C3", "C3: Architecture"},
		{"C4", "C4: Documentation Quality"},
		{"C5", "C5: Temporal Dynamics"},
		{"C6", "C6: Testing"},
		{"Unknown", "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := categoryDisplayName(tt.name)
			if got != tt.want {
				t.Errorf("categoryDisplayName(%q) = %q, want %q", tt.name, got, tt.want)
			}
		})
	}
}

func TestHTMLGenerator_SelfContained(t *testing.T) {
	gen, err := NewHTMLGenerator()
	if err != nil {
		t.Fatalf("NewHTMLGenerator() error = %v", err)
	}

	scored := &types.ScoredResult{
		ProjectName: "test-project",
		Composite:   7.5,
		Tier:        "Agent-Assisted",
		Categories: []types.CategoryScore{
			{Name: "C1", Score: 7.5, Weight: 0.20},
			{Name: "C2", Score: 8.0, Weight: 0.15},
			{Name: "C3", Score: 7.0, Weight: 0.15},
		},
	}

	var buf bytes.Buffer
	err = gen.GenerateReport(&buf, scored, nil, nil, nil)
	if err != nil {
		t.Fatalf("GenerateReport() error = %v", err)
	}

	html := buf.String()

	// Should have inline CSS (no external stylesheet link)
	if strings.Contains(html, `<link rel="stylesheet"`) {
		t.Error("Report should not have external stylesheet link")
	}
	if !strings.Contains(html, "<style>") {
		t.Error("Report should have inline <style> tag")
	}

	// Should not have external script references
	if strings.Contains(html, `<script src=`) {
		t.Error("Report should not have external script references")
	}

	// CSS should be substantial (contains actual styles)
	if !strings.Contains(html, "--color-green") {
		t.Error("Report should have CSS custom properties")
	}
}

// buildPromptTestEvidence returns sample evidence items for testing.
func buildPromptTestEvidence() []types.EvidenceItem {
	return []types.EvidenceItem{
		{FilePath: "internal/foo.go", Line: 42, Value: 15.0, Description: "high complexity"},
		{FilePath: "internal/bar.go", Line: 10, Value: 12.0, Description: "long function"},
	}
}

// buildAllCategoriesScoredResult creates a ScoredResult with all 7 categories,
// each having at least one metric with the given score value.
func buildAllCategoriesScoredResult(score float64) *types.ScoredResult {
	evidence := buildPromptTestEvidence()
	return &types.ScoredResult{
		ProjectName: "prompt-test",
		Composite:   score,
		Tier:        "Agent-Assisted",
		Categories: []types.CategoryScore{
			{
				Name: "C1", Score: score, Weight: 0.25,
				SubScores: []types.SubScore{
					{MetricName: "complexity_avg", RawValue: 15.0, Score: score, Weight: 0.25, Available: true, Evidence: evidence},
					{MetricName: "func_length_avg", RawValue: 30.0, Score: score, Weight: 0.20, Available: true, Evidence: evidence},
				},
			},
			{
				Name: "C2", Score: score, Weight: 0.10,
				SubScores: []types.SubScore{
					{MetricName: "type_annotation_coverage", RawValue: 60.0, Score: score, Weight: 0.30, Available: true, Evidence: evidence},
				},
			},
			{
				Name: "C3", Score: score, Weight: 0.20,
				SubScores: []types.SubScore{
					{MetricName: "max_dir_depth", RawValue: 5.0, Score: score, Weight: 0.20, Available: true, Evidence: evidence},
				},
			},
			{
				Name: "C4", Score: score, Weight: 0.15,
				SubScores: []types.SubScore{
					{MetricName: "comment_density", RawValue: 8.0, Score: score, Weight: 0.20, Available: true, Evidence: evidence},
				},
			},
			{
				Name: "C5", Score: score, Weight: 0.10,
				SubScores: []types.SubScore{
					{MetricName: "churn_rate", RawValue: 200.0, Score: score, Weight: 0.20, Available: true, Evidence: evidence},
				},
			},
			{
				Name: "C6", Score: score, Weight: 0.15,
				SubScores: []types.SubScore{
					{MetricName: "coverage_percent", RawValue: 45.0, Score: score, Weight: 0.30, Available: true, Evidence: evidence},
				},
			},
			{
				Name: "C7", Score: score, Weight: 0.10,
				SubScores: []types.SubScore{
					{MetricName: "task_execution_consistency", RawValue: 5.0, Score: score, Weight: 0.20, Available: true, Evidence: evidence},
				},
			},
		},
	}
}

func TestHTMLGenerator_PromptModals(t *testing.T) {
	gen, err := NewHTMLGenerator()
	if err != nil {
		t.Fatalf("NewHTMLGenerator() error = %v", err)
	}

	scored := buildAllCategoriesScoredResult(5.0)
	trace := &TraceData{
		ScoringConfig: scoring.DefaultConfig(),
		Languages:     []string{"go"},
	}

	var buf bytes.Buffer
	err = gen.GenerateReport(&buf, scored, nil, nil, trace)
	if err != nil {
		t.Fatalf("GenerateReport() error = %v", err)
	}

	html := buf.String()

	checks := []struct {
		substring string
		desc      string
	}{
		{"Improve", "output should contain Improve button text"},
		{"prompt-copy-container", "output should contain prompt copy container class"},
		{"copyPromptText", "output should contain copyPromptText function"},
		{"## Context", "output should contain Context prompt section header"},
		{"## Build", "output should contain Build &amp; Test section header"},
		{"## Task", "output should contain Task prompt section header"},
		{"## Verification", "output should contain Verification prompt section header"},
		{`<template id="prompt-complexity_avg">`, "output should contain prompt template element for complexity_avg"},
	}

	for _, c := range checks {
		if !strings.Contains(html, c.substring) {
			t.Errorf("%s (missing %q)", c.desc, c.substring)
		}
	}
}

func TestHTMLGenerator_PromptModals_HighScore(t *testing.T) {
	gen, err := NewHTMLGenerator()
	if err != nil {
		t.Fatalf("NewHTMLGenerator() error = %v", err)
	}

	scored := buildAllCategoriesScoredResult(9.5)
	trace := &TraceData{
		ScoringConfig: scoring.DefaultConfig(),
		Languages:     []string{"go"},
	}

	var buf bytes.Buffer
	err = gen.GenerateReport(&buf, scored, nil, nil, trace)
	if err != nil {
		t.Fatalf("GenerateReport() error = %v", err)
	}

	html := buf.String()

	if strings.Contains(html, `<template id="prompt-`) {
		t.Error("high-scoring metrics (>= 9.0) should not have prompt templates")
	}
	// The prompt-copy-container class appears in CSS, but should not appear
	// in any <template> element content when all scores are high.
	if strings.Contains(html, `<template id="prompt-`) {
		t.Error("high-scoring metrics should not generate prompt template elements")
	}
}

func TestHTMLGenerator_PromptModals_AllCategories(t *testing.T) {
	gen, err := NewHTMLGenerator()
	if err != nil {
		t.Fatalf("NewHTMLGenerator() error = %v", err)
	}

	scored := buildAllCategoriesScoredResult(5.0)
	trace := &TraceData{
		ScoringConfig: scoring.DefaultConfig(),
		Languages:     []string{"go"},
	}

	var buf bytes.Buffer
	err = gen.GenerateReport(&buf, scored, nil, nil, trace)
	if err != nil {
		t.Fatalf("GenerateReport() error = %v", err)
	}

	html := buf.String()

	// Count prompt template elements -- should have at least 7 (one per category,
	// since each category has at least one metric with score 5.0)
	count := strings.Count(html, `<template id="prompt-`)
	if count < 7 {
		t.Errorf("expected at least 7 prompt template elements (one per category), got %d", count)
	}
}
