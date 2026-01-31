package pipeline

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

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

func TestStubParserPassthrough(t *testing.T) {
	files := []types.DiscoveredFile{
		{Path: "/a/b/main.go", RelPath: "main.go", Class: types.ClassSource},
		{Path: "/a/b/main_test.go", RelPath: "main_test.go", Class: types.ClassTest},
		{Path: "/a/b/vendor/x.go", RelPath: "vendor/x.go", Class: types.ClassExcluded, ExcludeReason: "vendor"},
	}

	parser := &StubParser{}
	parsed, err := parser.Parse(files)
	if err != nil {
		t.Fatalf("StubParser.Parse() returned error: %v", err)
	}

	if len(parsed) != len(files) {
		t.Fatalf("expected %d parsed files, got %d", len(files), len(parsed))
	}

	for i, p := range parsed {
		if p.Path != files[i].Path {
			t.Errorf("file %d: expected Path %q, got %q", i, files[i].Path, p.Path)
		}
		if p.RelPath != files[i].RelPath {
			t.Errorf("file %d: expected RelPath %q, got %q", i, files[i].RelPath, p.RelPath)
		}
		if p.Class != files[i].Class {
			t.Errorf("file %d: expected Class %v, got %v", i, files[i].Class, p.Class)
		}
	}
}
