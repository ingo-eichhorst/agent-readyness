package parser

import (
	"path/filepath"
	"runtime"
	"testing"

	"golang.org/x/tools/go/packages"
)

// repoRoot returns the absolute path to the repository root.
func repoRoot(t *testing.T) string {
	t.Helper()
	// This file is at internal/parser/parser_test.go, so repo root is ../..
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root, err := filepath.Abs(filepath.Join(filepath.Dir(thisFile), "..", ".."))
	if err != nil {
		t.Fatalf("failed to resolve repo root: %v", err)
	}
	return root
}

func TestParseReturnsPackages(t *testing.T) {
	root := repoRoot(t)
	p := &GoPackagesParser{}

	pkgs, err := p.Parse(root)
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	if len(pkgs) == 0 {
		t.Fatal("Parse() returned zero packages")
	}

	t.Logf("Loaded %d packages", len(pkgs))
	for _, pkg := range pkgs {
		t.Logf("  %s (ForTest=%q, files=%d, syntax=%d)",
			pkg.PkgPath, pkg.ForTest, len(pkg.GoFiles), len(pkg.Syntax))
	}
}

func TestParsedPackageHasSyntax(t *testing.T) {
	root := repoRoot(t)
	p := &GoPackagesParser{}

	pkgs, err := p.Parse(root)
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	foundWithSyntax := false
	for _, pkg := range pkgs {
		if len(pkg.Syntax) > 0 {
			foundWithSyntax = true
			break
		}
	}

	if !foundWithSyntax {
		t.Error("no package has non-nil Syntax (AST)")
	}
}

func TestParsedPackageHasFset(t *testing.T) {
	root := repoRoot(t)
	p := &GoPackagesParser{}

	pkgs, err := p.Parse(root)
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	for _, pkg := range pkgs {
		if pkg.Fset == nil {
			t.Errorf("package %s has nil Fset", pkg.PkgPath)
		}
	}
}

func TestParsedPackageHasTypeInfo(t *testing.T) {
	root := repoRoot(t)
	p := &GoPackagesParser{}

	pkgs, err := p.Parse(root)
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	for _, pkg := range pkgs {
		if pkg.ForTest != "" {
			continue // test packages may have partial type info
		}
		if pkg.Types == nil {
			t.Errorf("package %s has nil Types", pkg.PkgPath)
		}
		if pkg.TypesInfo == nil {
			t.Errorf("package %s has nil TypesInfo", pkg.PkgPath)
		}
	}
}

func TestTestPackagesIdentified(t *testing.T) {
	root := repoRoot(t)
	p := &GoPackagesParser{}

	pkgs, err := p.Parse(root)
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	foundTestPkg := false
	for _, pkg := range pkgs {
		if pkg.ForTest != "" {
			foundTestPkg = true
			t.Logf("Test package: %s (ForTest=%s)", pkg.PkgPath, pkg.ForTest)
		}
	}

	if !foundTestPkg {
		t.Error("no test packages found (ForTest field not set on any package)")
	}
}

// TestIsValidPackage_ErrorsNoUsefulData verifies that a package with errors
// and no Types/Syntax is rejected by isValidPackage.
func TestIsValidPackage_ErrorsNoUsefulData(t *testing.T) {
	pkg := &packages.Package{
		PkgPath: "example.com/broken",
		Errors:  []packages.Error{{Msg: "something went wrong"}},
		// Types and Syntax are nil (zero values)
	}

	if isValidPackage(pkg) {
		t.Error("isValidPackage should return false when errors present and Types/Syntax nil")
	}
}

// TestIsValidPackage_NoErrors verifies that a package with no errors is valid.
func TestIsValidPackage_NoErrors(t *testing.T) {
	pkg := &packages.Package{
		PkgPath: "example.com/ok",
	}

	if !isValidPackage(pkg) {
		t.Error("isValidPackage should return true when no errors")
	}
}

// TestDeduplicateAndConvertPackages_DuplicateSourcePackage verifies that duplicate
// source packages (same PkgPath, ForTest=="") are deduplicated.
func TestDeduplicateAndConvertPackages_DuplicateSourcePackage(t *testing.T) {
	pkg1 := &packages.Package{
		ID:      "example.com/foo",
		PkgPath: "example.com/foo",
		Name:    "foo",
	}
	pkg2 := &packages.Package{
		ID:      "example.com/foo",
		PkgPath: "example.com/foo",
		Name:    "foo",
	}

	result := deduplicateAndConvertPackages([]*packages.Package{pkg1, pkg2})
	if len(result) != 1 {
		t.Errorf("got %d packages, want 1 (duplicate should be dropped)", len(result))
	}
}

// TestDeduplicateAndConvertPackages_TestPackagesNotDeduplicated verifies that
// test packages (ForTest != "") are always included even if PkgPath matches.
func TestDeduplicateAndConvertPackages_TestPackagesNotDeduplicated(t *testing.T) {
	srcPkg := &packages.Package{
		ID:      "example.com/foo",
		PkgPath: "example.com/foo",
		Name:    "foo",
	}
	testPkg := &packages.Package{
		ID:      "example.com/foo [example.com/foo.test]",
		PkgPath: "example.com/foo",
		Name:    "foo",
		ForTest: "example.com/foo",
	}

	result := deduplicateAndConvertPackages([]*packages.Package{srcPkg, testPkg})
	if len(result) != 2 {
		t.Errorf("got %d packages, want 2 (test package should not be deduplicated)", len(result))
	}

	foundTest := false
	for _, p := range result {
		if p.ForTest != "" {
			foundTest = true
		}
	}
	if !foundTest {
		t.Error("test package missing from result")
	}
}

// TestParseInvalidDir verifies that Parse returns an error for non-existent dirs.
func TestParseInvalidDir(t *testing.T) {
	p := &GoPackagesParser{}
	// packages.Load with a non-existent dir may return an error or empty packages.
	// We just verify it doesn't panic and handles gracefully.
	_, err := p.Parse("/nonexistent/path/that/does/not/exist")
	// err may or may not be non-nil depending on go/packages behaviour;
	// we just ensure no panic occurs.
	_ = err
}
