package parser

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ingo-eichhorst/agent-readyness/pkg/types"
)

func TestNewTreeSitterParser(t *testing.T) {
	p, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("NewTreeSitterParser() error: %v", err)
	}
	defer p.Close()
}

func TestParsePythonFile(t *testing.T) {
	p, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("NewTreeSitterParser() error: %v", err)
	}
	defer p.Close()

	pyFile, err := filepath.Abs("../../testdata/valid-python-project/app.py")
	if err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(pyFile)
	if err != nil {
		t.Fatal(err)
	}

	tree, err := p.ParseFile(types.LangPython, ".py", content)
	if err != nil {
		t.Fatalf("ParseFile(Python) error: %v", err)
	}
	defer tree.Close()

	root := tree.RootNode()
	if root == nil {
		t.Fatal("root node is nil")
	}
	if root.ChildCount() == 0 {
		t.Error("root node has no children")
	}

	// Python module root should be "module"
	if root.Kind() != "module" {
		t.Errorf("root node kind = %q, want %q", root.Kind(), "module")
	}
}

func TestParseTypeScriptFile(t *testing.T) {
	p, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("NewTreeSitterParser() error: %v", err)
	}
	defer p.Close()

	tsFile, err := filepath.Abs("../../testdata/valid-ts-project/src/index.ts")
	if err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(tsFile)
	if err != nil {
		t.Fatal(err)
	}

	tree, err := p.ParseFile(types.LangTypeScript, ".ts", content)
	if err != nil {
		t.Fatalf("ParseFile(TypeScript) error: %v", err)
	}
	defer tree.Close()

	root := tree.RootNode()
	if root == nil {
		t.Fatal("root node is nil")
	}
	if root.ChildCount() == 0 {
		t.Error("root node has no children")
	}

	// TypeScript module root should be "program"
	if root.Kind() != "program" {
		t.Errorf("root node kind = %q, want %q", root.Kind(), "program")
	}
}

func TestParserReuse(t *testing.T) {
	p, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("NewTreeSitterParser() error: %v", err)
	}
	defer p.Close()

	// Parse two different Python files sequentially
	content1 := []byte("def foo():\n    return 42\n")
	tree1, err := p.ParseFile(types.LangPython, ".py", content1)
	if err != nil {
		t.Fatalf("ParseFile #1 error: %v", err)
	}
	defer tree1.Close()

	content2 := []byte("class Bar:\n    pass\n")
	tree2, err := p.ParseFile(types.LangPython, ".py", content2)
	if err != nil {
		t.Fatalf("ParseFile #2 error: %v", err)
	}
	defer tree2.Close()

	// Both should have valid root nodes
	if tree1.RootNode() == nil || tree2.RootNode() == nil {
		t.Error("one or both trees have nil root nodes")
	}
}

func TestCloseDoesNotPanic(t *testing.T) {
	p, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("NewTreeSitterParser() error: %v", err)
	}

	// Close should not panic
	p.Close()

	// CloseAll with nil should not panic
	CloseAll(nil)
	CloseAll([]*ParsedTreeSitterFile{})
}

func TestParseTargetFiles(t *testing.T) {
	p, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("NewTreeSitterParser() error: %v", err)
	}
	defer p.Close()

	pyRoot, err := filepath.Abs("../../testdata/valid-python-project")
	if err != nil {
		t.Fatal(err)
	}

	target := &types.AnalysisTarget{
		Language: types.LangPython,
		RootDir:  pyRoot,
		Files: []types.SourceFile{
			{
				Path:     filepath.Join(pyRoot, "app.py"),
				RelPath:  "app.py",
				Language: types.LangPython,
				Class:    types.ClassSource,
			},
			{
				Path:     filepath.Join(pyRoot, "test_app.py"),
				RelPath:  "test_app.py",
				Language: types.LangPython,
				Class:    types.ClassTest,
			},
		},
	}

	files, err := p.ParseTargetFiles(target)
	if err != nil {
		t.Fatalf("ParseTargetFiles error: %v", err)
	}
	defer CloseAll(files)

	if len(files) != 2 {
		t.Fatalf("got %d parsed files, want 2", len(files))
	}

	for _, f := range files {
		if f.Tree == nil {
			t.Errorf("file %s has nil tree", f.RelPath)
		}
		if f.Tree.RootNode() == nil {
			t.Errorf("file %s has nil root node", f.RelPath)
		}
		if len(f.Content) == 0 {
			t.Errorf("file %s has empty content", f.RelPath)
		}
		if f.Language != types.LangPython {
			t.Errorf("file %s language = %q, want %q", f.RelPath, f.Language, types.LangPython)
		}
	}
}

func TestParseUnsupportedLanguage(t *testing.T) {
	p, err := NewTreeSitterParser()
	if err != nil {
		t.Fatalf("NewTreeSitterParser() error: %v", err)
	}
	defer p.Close()

	_, err = p.ParseFile(types.LangGo, ".go", []byte("package main"))
	if err == nil {
		t.Error("expected error for unsupported language Go, got nil")
	}
}
