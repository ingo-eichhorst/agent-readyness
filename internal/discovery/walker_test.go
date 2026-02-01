package discovery

import (
	"os"
	"path/filepath"
	"runtime"
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

func TestWalkerSymlink(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a regular .go file
	goContent := []byte("package main\n")
	if err := os.WriteFile(filepath.Join(tmpDir, "real.go"), goContent, 0o644); err != nil {
		t.Fatal(err)
	}

	// Create a target directory with a .go file
	targetDir := filepath.Join(tmpDir, "target")
	if err := os.Mkdir(targetDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(targetDir, "target.go"), goContent, 0o644); err != nil {
		t.Fatal(err)
	}

	// Create a symlink to the .go file
	if err := os.Symlink(filepath.Join(tmpDir, "real.go"), filepath.Join(tmpDir, "link.go")); err != nil {
		t.Skipf("symlink creation not supported: %v", err)
	}

	// Create a symlink to the directory
	if err := os.Symlink(targetDir, filepath.Join(tmpDir, "linkdir")); err != nil {
		t.Skipf("directory symlink creation not supported: %v", err)
	}

	w := NewWalker()
	result, err := w.Discover(tmpDir)
	if err != nil {
		t.Fatalf("Discover returned error: %v", err)
	}

	// The regular file should be found
	found := false
	for _, f := range result.Files {
		if f.RelPath == "real.go" {
			found = true
			break
		}
	}
	if !found {
		t.Error("real.go not found in results")
	}

	// target/target.go should also be found (real directory)
	found = false
	for _, f := range result.Files {
		if f.RelPath == filepath.Join("target", "target.go") {
			found = true
			break
		}
	}
	if !found {
		t.Error("target/target.go not found in results")
	}

	// Symlinks should have been detected
	if result.SymlinkCount < 1 {
		t.Errorf("SymlinkCount = %d, want >= 1", result.SymlinkCount)
	}
}

func TestWalkerPermissionDenied(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission test not reliable on Windows")
	}

	tmpDir := t.TempDir()

	// Create a regular .go file
	goContent := []byte("package main\n")
	if err := os.WriteFile(filepath.Join(tmpDir, "accessible.go"), goContent, 0o644); err != nil {
		t.Fatal(err)
	}

	// Create an unreadable subdirectory
	subdir := filepath.Join(tmpDir, "noperm")
	if err := os.Mkdir(subdir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subdir, "hidden.go"), goContent, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(subdir, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		os.Chmod(subdir, 0o755)
	})

	w := NewWalker()
	result, err := w.Discover(tmpDir)
	if err != nil {
		t.Fatalf("Discover returned error: %v (should have continued)", err)
	}

	// The accessible file should be found
	found := false
	for _, f := range result.Files {
		if f.RelPath == "accessible.go" {
			found = true
			break
		}
	}
	if !found {
		t.Error("accessible.go not found in results")
	}

	// Should have skipped at least one entry
	if result.SkippedCount < 1 {
		t.Errorf("SkippedCount = %d, want >= 1", result.SkippedCount)
	}
}

func TestWalkerUnicodePaths(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a Unicode-named subdirectory
	unicodeDir := filepath.Join(tmpDir, "pkg_unicod\u00e9")
	if err := os.Mkdir(unicodeDir, 0o755); err != nil {
		t.Fatal(err)
	}

	goContent := []byte("package main\n")
	if err := os.WriteFile(filepath.Join(unicodeDir, "main.go"), goContent, 0o644); err != nil {
		t.Fatal(err)
	}

	w := NewWalker()
	result, err := w.Discover(tmpDir)
	if err != nil {
		t.Fatalf("Discover returned error: %v", err)
	}

	// The file in the Unicode directory should be found and classified
	found := false
	for _, f := range result.Files {
		if f.RelPath == filepath.Join("pkg_unicod\u00e9", "main.go") {
			found = true
			if f.Class != types.ClassSource {
				t.Errorf("Unicode path file: Class = %v, want ClassSource", f.Class)
			}
			break
		}
	}
	if !found {
		t.Errorf("file in Unicode directory not found in results; files: %v", result.Files)
	}
}

func TestWalkerContinuesOnBadGeneratedCheck(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission test not reliable on Windows")
	}

	tmpDir := t.TempDir()

	// Create a .go file with no read permissions (IsGeneratedFile will fail)
	goFile := filepath.Join(tmpDir, "unreadable.go")
	if err := os.WriteFile(goFile, []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(goFile, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		os.Chmod(goFile, 0o644)
	})

	// Create a readable .go file too
	if err := os.WriteFile(filepath.Join(tmpDir, "readable.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	w := NewWalker()
	result, err := w.Discover(tmpDir)
	if err != nil {
		t.Fatalf("Discover returned error: %v (should have continued)", err)
	}

	// Should have skipped the unreadable file
	if result.SkippedCount < 1 {
		t.Errorf("SkippedCount = %d, want >= 1", result.SkippedCount)
	}

	// The readable file should still be found
	found := false
	for _, f := range result.Files {
		if f.RelPath == "readable.go" {
			found = true
			break
		}
	}
	if !found {
		t.Error("readable.go not found in results")
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
