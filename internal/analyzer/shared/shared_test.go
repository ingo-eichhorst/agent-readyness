package shared

import (
	"testing"

	"golang.org/x/tools/go/packages"

	"github.com/ingo/agent-readyness/internal/parser"
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
