package parser

import (
	"path/filepath"
	"runtime"
	"testing"
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
