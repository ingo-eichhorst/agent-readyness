package pipeline

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
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

	// Should contain summary labels
	if !strings.Contains(out, "Source files:") {
		t.Error("output missing 'Source files:' label")
	}
	if !strings.Contains(out, "Test files:") {
		t.Error("output missing 'Test files:' label")
	}
	if !strings.Contains(out, "Go files discovered:") {
		t.Error("output missing 'Go files discovered:' label")
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
