package metadata

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ingo-eichhorst/agent-readyness/pkg/types"
)

func TestCollectNonGitRepo(t *testing.T) {
	dir := t.TempDir()
	scanResult := &types.ScanResult{RootDir: dir}

	meta := Collect(dir, scanResult)

	if meta.Available {
		t.Error("expected Available=false for non-git directory")
	}
	if meta.TotalCommits != 0 {
		t.Errorf("expected 0 commits, got %d", meta.TotalCommits)
	}
}

func TestCollectLOCAndLanguages(t *testing.T) {
	dir := t.TempDir()

	// Create test files
	goFile := filepath.Join(dir, "main.go")
	os.WriteFile(goFile, []byte("package main\n\nfunc main() {\n}\n"), 0644)
	pyFile := filepath.Join(dir, "app.py")
	os.WriteFile(pyFile, []byte("print('hello')\n"), 0644)

	scanResult := &types.ScanResult{
		RootDir: dir,
		Files: []types.DiscoveredFile{
			{Path: goFile, RelPath: "main.go", Class: types.ClassSource, Language: types.LangGo},
			{Path: pyFile, RelPath: "app.py", Class: types.ClassSource, Language: types.LangPython},
		},
	}

	meta := Collect(dir, scanResult)

	if meta.LinesOfCode != 5 { // 4 go lines + 1 python line
		t.Errorf("expected 5 LOC, got %d", meta.LinesOfCode)
	}
	if len(meta.LanguageBreakdown) != 2 {
		t.Errorf("expected 2 languages, got %d", len(meta.LanguageBreakdown))
	}
	if meta.LanguageBreakdown[types.LangGo] < 70 {
		t.Errorf("expected Go > 70%%, got %.1f%%", meta.LanguageBreakdown[types.LangGo])
	}
}

func TestCollectExcludesNonSource(t *testing.T) {
	dir := t.TempDir()

	goFile := filepath.Join(dir, "main.go")
	os.WriteFile(goFile, []byte("package main\n"), 0644)
	testFile := filepath.Join(dir, "main_test.go")
	os.WriteFile(testFile, []byte("package main\n\nfunc TestX(t *testing.T) {}\n"), 0644)

	scanResult := &types.ScanResult{
		RootDir: dir,
		Files: []types.DiscoveredFile{
			{Path: goFile, RelPath: "main.go", Class: types.ClassSource, Language: types.LangGo},
			{Path: testFile, RelPath: "main_test.go", Class: types.ClassTest, Language: types.LangGo},
		},
	}

	meta := Collect(dir, scanResult)

	if meta.LinesOfCode != 1 {
		t.Errorf("expected 1 LOC (source only), got %d", meta.LinesOfCode)
	}
}
