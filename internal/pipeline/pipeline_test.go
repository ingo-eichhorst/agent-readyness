package pipeline

import (
	"bytes"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ingo/agent-readyness/internal/parser"
	"github.com/ingo/agent-readyness/pkg/types"
)

func TestPipelineRun(t *testing.T) {
	root, err := filepath.Abs("../../testdata/valid-go-project")
	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	p := New(&buf, false)

	if err := p.Run(root); err != nil {
		t.Fatalf("Pipeline.Run() returned error: %v", err)
	}

	out := buf.String()

	// Should contain file discovery labels
	if !strings.Contains(out, "Source files:") {
		t.Error("output missing 'Source files:' label")
	}
	if !strings.Contains(out, "Test files:") {
		t.Error("output missing 'Test files:' label")
	}
	if !strings.Contains(out, "Go files discovered:") {
		t.Error("output missing 'Go files discovered:' label")
	}

	// Should contain metric category headers
	if !strings.Contains(out, "C1: Code Health") {
		t.Error("output missing 'C1: Code Health' section")
	}
	if !strings.Contains(out, "C3: Architecture") {
		t.Error("output missing 'C3: Architecture' section")
	}
	if !strings.Contains(out, "C6: Testing") {
		t.Error("output missing 'C6: Testing' section")
	}

	// Should contain key metric labels
	metricChecks := []string{
		"Complexity avg:",
		"Complexity max:",
		"Max directory depth:",
		"Test-to-code ratio:",
	}
	for _, check := range metricChecks {
		if !strings.Contains(out, check) {
			t.Errorf("output missing metric %q\nGot:\n%s", check, out)
		}
	}
}

func TestPipelineRunVerbose(t *testing.T) {
	root, err := filepath.Abs("../../testdata/valid-go-project")
	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	p := New(&buf, true)

	if err := p.Run(root); err != nil {
		t.Fatalf("Pipeline.Run() returned error: %v", err)
	}

	out := buf.String()

	// Verbose should list individual files
	if !strings.Contains(out, "Discovered files:") {
		t.Error("verbose output missing 'Discovered files:' header")
	}
	if !strings.Contains(out, "main.go") {
		t.Error("verbose output missing main.go")
	}
}

func TestStubParserReturnsEmpty(t *testing.T) {
	p := &StubParser{}
	pkgs, err := p.Parse("/nonexistent")
	if err != nil {
		t.Fatalf("StubParser.Parse() returned error: %v", err)
	}

	if len(pkgs) != 0 {
		t.Fatalf("expected 0 packages from StubParser, got %d", len(pkgs))
	}
}

func TestStubAnalyzerReturnsEmpty(t *testing.T) {
	a := &StubAnalyzer{}
	if a.Name() != "stub" {
		t.Errorf("expected name 'stub', got %q", a.Name())
	}

	result, err := a.Analyze(nil)
	if err != nil {
		t.Fatalf("StubAnalyzer.Analyze() returned error: %v", err)
	}

	if result.Name != "stub" {
		t.Errorf("expected result name 'stub', got %q", result.Name)
	}
}

func TestPipelineAnalyzerErrorContinues(t *testing.T) {
	root, err := filepath.Abs("../../testdata/valid-go-project")
	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	p := New(&buf, false)

	// Replace analyzers with one that errors and one stub
	p.analyzers = []Analyzer{
		&errorAnalyzer{},
		&StubAnalyzer{},
	}

	if err := p.Run(root); err != nil {
		t.Fatalf("Pipeline.Run() should not fail when analyzer errors: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Warning:") {
		t.Error("expected warning about analyzer error in output")
	}
}

// errorAnalyzer is a test helper that always returns an error.
type errorAnalyzer struct{}

func (e *errorAnalyzer) Name() string { return "error-test" }

func (e *errorAnalyzer) Analyze(_ []*parser.ParsedPackage) (*types.AnalysisResult, error) {
	return nil, errors.New("test error")
}
