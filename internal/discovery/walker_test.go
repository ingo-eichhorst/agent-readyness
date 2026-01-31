package discovery

import (
	"path/filepath"
	"testing"

	"github.com/ingo/agent-readyness/pkg/types"
)

func TestDiscoverValidProject(t *testing.T) {
	root, err := filepath.Abs("../../testdata/valid-go-project")
	if err != nil {
		t.Fatal(err)
	}

	w := NewWalker()
	result, err := w.Discover(root)
	if err != nil {
		t.Fatalf("Discover(%q) returned error: %v", root, err)
	}

	// Build a lookup map by relative path
	fileMap := make(map[string]types.DiscoveredFile)
	for _, f := range result.Files {
		fileMap[f.RelPath] = f
	}

	// main.go should be ClassSource
	assertFile(t, fileMap, "main.go", types.ClassSource, "")

	// main_test.go should be ClassTest
	assertFile(t, fileMap, "main_test.go", types.ClassTest, "")

	// doc_generated.go should be ClassGenerated
	assertFile(t, fileMap, "doc_generated.go", types.ClassGenerated, "")

	// util_linux.go should be ClassSource (platform-specific but still source)
	assertFile(t, fileMap, "util_linux.go", types.ClassSource, "")

	// vendor/dep/dep.go should be ClassExcluded with reason "vendor"
	// Actually vendor dir is skipped entirely, so vendor files should NOT appear
	// The plan says vendor dirs are skipped via SkipDir, so they won't be in results.
	// But the plan also says "vendor/dep/dep.go is ClassExcluded with ExcludeReason vendor"
	// Let's check: the plan wants it in results as excluded.
	assertFile(t, fileMap, filepath.Join("vendor", "dep", "dep.go"), types.ClassExcluded, "vendor")

	// ignored_by_gitignore.go should be ClassExcluded with reason "gitignore"
	assertFile(t, fileMap, "ignored_by_gitignore.go", types.ClassExcluded, "gitignore")

	// .git directory contents should NOT be in results at all
	for relPath := range fileMap {
		if filepath.Base(relPath) == ".git" || len(relPath) > 4 && relPath[:5] == ".git/" {
			t.Errorf("found .git file in results: %s", relPath)
		}
	}

	// Verify counts
	// Expected: main.go (source), util_linux.go (source), main_test.go (test),
	//           doc_generated.go (generated), vendor/dep/dep.go (excluded/vendor),
	//           ignored_by_gitignore.go (excluded/gitignore)
	if result.SourceCount != 2 {
		t.Errorf("SourceCount = %d, want 2", result.SourceCount)
	}
	if result.TestCount != 1 {
		t.Errorf("TestCount = %d, want 1", result.TestCount)
	}
	if result.GeneratedCount != 1 {
		t.Errorf("GeneratedCount = %d, want 1", result.GeneratedCount)
	}
	if result.VendorCount != 1 {
		t.Errorf("VendorCount = %d, want 1", result.VendorCount)
	}
	if result.GitignoreCount != 1 {
		t.Errorf("GitignoreCount = %d, want 1", result.GitignoreCount)
	}
	if result.TotalFiles != 6 {
		t.Errorf("TotalFiles = %d, want 6", result.TotalFiles)
	}
}

func TestDiscoverEmptyDir(t *testing.T) {
	tmpDir := t.TempDir()

	w := NewWalker()
	result, err := w.Discover(tmpDir)
	if err != nil {
		t.Fatalf("Discover(%q) returned error: %v", tmpDir, err)
	}

	if len(result.Files) != 0 {
		t.Errorf("expected empty file list, got %d files", len(result.Files))
	}
	if result.TotalFiles != 0 {
		t.Errorf("TotalFiles = %d, want 0", result.TotalFiles)
	}
}

func TestDiscoverNonExistentDir(t *testing.T) {
	w := NewWalker()
	_, err := w.Discover("/nonexistent/path/that/does/not/exist")
	if err == nil {
		t.Error("expected error for non-existent directory, got nil")
	}
}

func assertFile(t *testing.T, fileMap map[string]types.DiscoveredFile, relPath string, wantClass types.FileClass, wantReason string) {
	t.Helper()
	f, ok := fileMap[relPath]
	if !ok {
		t.Errorf("file %q not found in results", relPath)
		return
	}
	if f.Class != wantClass {
		t.Errorf("file %q: Class = %v, want %v", relPath, f.Class, wantClass)
	}
	if wantReason != "" && f.ExcludeReason != wantReason {
		t.Errorf("file %q: ExcludeReason = %q, want %q", relPath, f.ExcludeReason, wantReason)
	}
}
