package shared

import (
	"testing"

	"golang.org/x/tools/go/packages"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"
	tree_sitter_python "github.com/tree-sitter/tree-sitter-python/bindings/go"

	"github.com/ingo-eichhorst/agent-readyness/internal/parser"
)

// makeImports creates a map for testing import graphs
func makeImports(paths ...string) map[string]*packages.Package {
	m := make(map[string]*packages.Package)
	for _, p := range paths {
		m[p] = &packages.Package{}
	}
	return m
}

func TestBuildImportGraph(t *testing.T) {
	tests := []struct {
		name       string
		pkgs       []*parser.ParsedPackage
		modulePath string
		wantFwd    map[string]int // pkg -> count of forward edges
		wantRev    map[string]int // pkg -> count of reverse edges
	}{
		{
			name: "simple graph",
			pkgs: []*parser.ParsedPackage{
				{PkgPath: "mymod/pkg/a", Imports: makeImports("mymod/pkg/b", "fmt")},
				{PkgPath: "mymod/pkg/b", Imports: makeImports("fmt")},
			},
			modulePath: "mymod",
			wantFwd:    map[string]int{"mymod/pkg/a": 1}, // only intra-module
			wantRev:    map[string]int{"mymod/pkg/b": 1},
		},
		{
			name: "skip test packages",
			pkgs: []*parser.ParsedPackage{
				{PkgPath: "mymod/pkg/a", ForTest: "", Imports: makeImports("mymod/pkg/b")},
				{PkgPath: "mymod/pkg/a", ForTest: "mymod/pkg/a", Imports: makeImports("mymod/pkg/c")},
			},
			modulePath: "mymod",
			wantFwd:    map[string]int{"mymod/pkg/a": 1}, // test package ignored
			wantRev:    map[string]int{"mymod/pkg/b": 1},
		},
		{
			name: "external imports filtered",
			pkgs: []*parser.ParsedPackage{
				{PkgPath: "mymod/pkg/a", Imports: makeImports("github.com/external/lib")},
			},
			modulePath: "mymod",
			wantFwd:    map[string]int{},
			wantRev:    map[string]int{},
		},
		{
			name:       "empty packages",
			pkgs:       []*parser.ParsedPackage{},
			modulePath: "mymod",
			wantFwd:    map[string]int{},
			wantRev:    map[string]int{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			graph := BuildImportGraph(tt.pkgs, tt.modulePath)

			// Verify forward edges
			for pkg, wantCount := range tt.wantFwd {
				gotCount := len(graph.Forward[pkg])
				if gotCount != wantCount {
					t.Errorf("Forward[%s] = %d edges, want %d", pkg, gotCount, wantCount)
				}
			}

			// Verify reverse edges
			for pkg, wantCount := range tt.wantRev {
				gotCount := len(graph.Reverse[pkg])
				if gotCount != wantCount {
					t.Errorf("Reverse[%s] = %d edges, want %d", pkg, gotCount, wantCount)
				}
			}
		})
	}
}

func TestCountLines(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		want  int
	}{
		{"empty", []byte(""), 0},
		{"single line", []byte("hello"), 1},
		{"two lines", []byte("hello\nworld"), 2},
		{"trailing newline", []byte("hello\n"), 2},
		{"multiple newlines", []byte("a\nb\nc\n"), 4},
		{"just newline", []byte("\n"), 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CountLines(tt.input)
			if got != tt.want {
				t.Errorf("CountLines(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsTestFileByPath(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		// Positive cases
		{"test_foo.py", true},
		{"tests/test_bar.py", true},
		{"foo_test.py", true},
		{"src/module/test_something.py", true},
		{"conftest.py", true},
		{"tests/conftest.py", true},

		// Case insensitive
		{"Test_Foo.PY", true},
		{"FOO_TEST.PY", true},

		// Negative cases
		{"foo.py", false},
		{"testing.py", false},
		{"mytest.py", false},
		{"test.txt", false},
		{"src/module/utils.py", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := IsTestFileByPath(tt.path)
			if got != tt.want {
				t.Errorf("IsTestFileByPath(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestTsIsTestFile(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		// Test file suffixes
		{"foo.test.ts", true},
		{"foo.spec.ts", true},
		{"foo.test.tsx", true},
		{"foo.spec.tsx", true},
		{"foo.test.js", true},
		{"foo.spec.js", true},

		// __tests__ directory
		{"src/__tests__/foo.ts", true},
		{"__tests__/utils.ts", true},
		{"src/module/__tests__/helper.js", true},

		// Case insensitive
		{"Foo.TEST.TS", true},
		{"src/__TESTS__/bar.ts", true},

		// Negative cases
		{"foo.ts", false},
		{"testing.ts", false},
		{"test.ts", false},
		{"spec.ts", false},
		{"foo.tests.ts", false},
		{"src/module/utils.ts", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := TsIsTestFile(tt.path)
			if got != tt.want {
				t.Errorf("TsIsTestFile(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestTsStripQuotes(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		// Double quotes
		{`"hello"`, "hello"},
		{`"hello world"`, "hello world"},

		// Single quotes
		{`'hello'`, "hello"},
		{`'hello world'`, "hello world"},

		// Backticks
		{"`hello`", "hello"},
		{"`hello world`", "hello world"},

		// No quotes
		{"hello", "hello"},
		{"", ""},

		// Mismatched quotes (should not strip)
		{`"hello'`, `"hello'`},
		{`'hello"`, `'hello"`},

		// Only one character
		{`"`, `"`},
		{`'`, `'`},

		// Empty quoted string
		{`""`, ""},
		{`''`, ""},
		{"``", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := TsStripQuotes(tt.input)
			if got != tt.want {
				t.Errorf("TsStripQuotes(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// newPythonTree creates a parsed tree for testing WalkTree and NodeText.
func newPythonTree(t *testing.T, src string) (*tree_sitter.Tree, []byte) {
	t.Helper()
	p := tree_sitter.NewParser()
	lang := tree_sitter.NewLanguage(tree_sitter_python.Language())
	if err := p.SetLanguage(lang); err != nil {
		t.Fatalf("set language: %v", err)
	}
	content := []byte(src)
	tree := p.Parse(content, nil)
	if tree == nil {
		t.Fatal("parse returned nil tree")
	}
	t.Cleanup(func() { tree.Close() })
	return tree, content
}

func TestWalkTree(t *testing.T) {
	tree, _ := newPythonTree(t, "x = 1\n")
	root := tree.RootNode()

	var visited int
	WalkTree(root, func(n *tree_sitter.Node) {
		visited++
	})

	if visited == 0 {
		t.Error("WalkTree visited 0 nodes, expected > 0")
	}
}

func TestWalkTree_Nil(t *testing.T) {
	// Should not panic on nil node.
	WalkTree(nil, func(n *tree_sitter.Node) {
		t.Error("fn called for nil node")
	})
}

func TestNodeText(t *testing.T) {
	src := "x = 1\n"
	tree, content := newPythonTree(t, src)
	root := tree.RootNode()

	// The root node should span the entire source.
	got := NodeText(root, content)
	if got != src {
		t.Errorf("NodeText(root) = %q, want %q", got, src)
	}
}

func TestAnalyzeDirectoryDepth_Empty(t *testing.T) {
	max, avg := AnalyzeDirectoryDepth([]*parser.ParsedTreeSitterFile{}, "")
	if max != 0 || avg != 0 {
		t.Errorf("expected 0,0 for empty files, got %d,%f", max, avg)
	}
}

func TestAnalyzeDirectoryDepth_WithRelPath(t *testing.T) {
	files := []*parser.ParsedTreeSitterFile{
		{RelPath: "a/b/c.py"},
		{RelPath: "a/d.py"},
		{RelPath: "e.py"},
	}
	maxDepth, avgDepth := AnalyzeDirectoryDepth(files, "")

	// depths: 2, 1, 0 → max=2, avg=1.0
	if maxDepth != 2 {
		t.Errorf("maxDepth = %d, want 2", maxDepth)
	}
	wantAvg := 1.0
	if avgDepth != wantAvg {
		t.Errorf("avgDepth = %f, want %f", avgDepth, wantAvg)
	}
}

func TestAnalyzeDirectoryDepth_WithAbsPath(t *testing.T) {
	// RelPath empty: fall back to computing from Path and rootDir.
	rootDir := "/project"
	files := []*parser.ParsedTreeSitterFile{
		{Path: "/project/src/foo.py"},
		{Path: "/project/src/bar/baz.py"},
	}
	maxDepth, avgDepth := AnalyzeDirectoryDepth(files, rootDir)

	// depths: 1, 2 → max=2, avg=1.5
	if maxDepth != 2 {
		t.Errorf("maxDepth = %d, want 2", maxDepth)
	}
	wantAvg := 1.5
	if avgDepth != wantAvg {
		t.Errorf("avgDepth = %f, want %f", avgDepth, wantAvg)
	}
}
