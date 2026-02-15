package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateProject_NonExistentDir(t *testing.T) {
	err := validateProject("/nonexistent/path/to/dir")
	if err == nil {
		t.Fatal("expected error for non-existent directory")
	}
	if got := err.Error(); got != "directory not found: /nonexistent/path/to/dir" {
		t.Errorf("unexpected error message: %s", got)
	}
}

func TestValidateProject_NotADirectory(t *testing.T) {
	f, err := os.CreateTemp("", "ars-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	f.Close()

	err = validateProject(f.Name())
	if err == nil {
		t.Fatal("expected error for a file path")
	}
	if got := err.Error(); got != "not a directory: "+f.Name() {
		t.Errorf("unexpected error: %s", got)
	}
}

func TestValidateProject_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	err := validateProject(dir)
	if err == nil {
		t.Fatal("expected error for empty directory")
	}
	if got := err.Error(); got == "" {
		t.Error("error message should not be empty")
	}
}

func TestValidateProject_WithGoMod(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := validateProject(dir); err != nil {
		t.Errorf("expected no error for dir with go.mod, got: %v", err)
	}
}

func TestValidateProject_WithPyprojectToml(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte("[tool]"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := validateProject(dir); err != nil {
		t.Errorf("expected no error for dir with pyproject.toml, got: %v", err)
	}
}

func TestValidateProject_WithSetupPy(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "setup.py"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}
	if err := validateProject(dir); err != nil {
		t.Errorf("expected no error for dir with setup.py, got: %v", err)
	}
}

func TestValidateProject_WithRequirementsTxt(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "requirements.txt"), []byte("flask"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := validateProject(dir); err != nil {
		t.Errorf("expected no error for dir with requirements.txt, got: %v", err)
	}
}

func TestValidateProject_WithTsconfigJson(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "tsconfig.json"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := validateProject(dir); err != nil {
		t.Errorf("expected no error for dir with tsconfig.json, got: %v", err)
	}
}

func TestValidateProject_WithPackageJson(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := validateProject(dir); err != nil {
		t.Errorf("expected no error for dir with package.json, got: %v", err)
	}
}

func TestValidateProject_WithGoSourceFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := validateProject(dir); err != nil {
		t.Errorf("expected no error for dir with .go file, got: %v", err)
	}
}

func TestValidateProject_WithPySourceFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "app.py"), []byte("print('hi')"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := validateProject(dir); err != nil {
		t.Errorf("expected no error for dir with .py file, got: %v", err)
	}
}

func TestValidateProject_WithTsSourceFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "app.ts"), []byte("const x = 1"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := validateProject(dir); err != nil {
		t.Errorf("expected no error for dir with .ts file, got: %v", err)
	}
}

func TestValidateProject_WithTsxSourceFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "App.tsx"), []byte("<div/>"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := validateProject(dir); err != nil {
		t.Errorf("expected no error for dir with .tsx file, got: %v", err)
	}
}

func TestValidateProject_UnrecognizedFilesOnly(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "readme.md"), []byte("# hi"), 0644); err != nil {
		t.Fatal(err)
	}
	err := validateProject(dir)
	if err == nil {
		t.Fatal("expected error for dir with only unrecognized files")
	}
}

func TestScanCmdFlags(t *testing.T) {
	flags := []struct {
		name     string
		defValue string
	}{
		{"config", ""},
		{"threshold", "0"},
		{"json", "false"},
		{"no-llm", "false"},
		{"debug", "false"},
		{"output-html", ""},
		{"baseline", ""},
		{"badge", "false"},
		{"debug-dir", ""},
	}

	for _, tt := range flags {
		f := scanCmd.Flags().Lookup(tt.name)
		if f == nil {
			t.Errorf("flag %q not registered on scan command", tt.name)
			continue
		}
		if f.DefValue != tt.defValue {
			t.Errorf("flag %q: expected default %q, got %q", tt.name, tt.defValue, f.DefValue)
		}
	}
}

func TestScanCmdRequiresExactlyOneArg(t *testing.T) {
	// Reset state to avoid interference
	cmd := scanCmd
	err := cmd.Args(cmd, []string{})
	if err == nil {
		t.Error("scan should require exactly 1 argument, got no error for 0 args")
	}

	err = cmd.Args(cmd, []string{"a", "b"})
	if err == nil {
		t.Error("scan should require exactly 1 argument, got no error for 2 args")
	}

	err = cmd.Args(cmd, []string{"a"})
	if err != nil {
		t.Errorf("scan should accept exactly 1 argument, got error: %v", err)
	}
}

func TestScanCmdMetadata(t *testing.T) {
	if scanCmd.Use != "scan <directory>" {
		t.Errorf("expected Use='scan <directory>', got %q", scanCmd.Use)
	}
	if scanCmd.Short == "" {
		t.Error("scan command should have a short description")
	}
	if !scanCmd.SilenceUsage {
		t.Error("scan command should have SilenceUsage=true")
	}
}

func TestDebugDirImpliesDebug(t *testing.T) {
	f := scanCmd.Flags().Lookup("debug-dir")
	if f == nil {
		t.Fatal("debug-dir flag not found")
	}
	if f.Usage == "" {
		t.Error("debug-dir flag should have usage text")
	}
}

// resetScanFlags resets package-level flags to defaults before each integration test.
func resetScanFlags() {
	configPath = ""
	threshold = 0
	jsonOutput = false
	noLLM = false
	debug = false
	outputHTML = ""
	baselinePath = ""
	badgeOutput = false
	debugDir = ""
	verbose = false
}

// makeMinimalGoProject creates a temp dir with a minimal Go module for scanning.
func makeMinimalGoProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/test\n\ngo 1.21\n"), 0644)
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0644)
	return dir
}

func TestScanRunE_InvalidDir(t *testing.T) {
	resetScanFlags()
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"scan", "/nonexistent/path/xyz"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for non-existent directory")
	}
	if !strings.Contains(err.Error(), "directory not found") {
		t.Errorf("expected 'directory not found' error, got: %v", err)
	}
}

func TestScanRunE_NoArgs(t *testing.T) {
	resetScanFlags()
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"scan"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing argument")
	}
}

func TestScanRunE_ValidProject_NoLLM(t *testing.T) {
	resetScanFlags()
	dir := makeMinimalGoProject(t)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"scan", "--no-llm", dir})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("scan with --no-llm should succeed, got: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "LLM features disabled") {
		t.Errorf("expected 'LLM features disabled' in output, got: %s", output)
	}
}

func TestScanRunE_JSONOutput(t *testing.T) {
	resetScanFlags()
	dir := makeMinimalGoProject(t)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"scan", "--no-llm", "--json", dir})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("scan with --json should succeed, got: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "{") {
		t.Errorf("expected JSON output containing '{', got: %s", output)
	}
}

func TestScanRunE_WithDebugFlag(t *testing.T) {
	resetScanFlags()
	dir := makeMinimalGoProject(t)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"scan", "--no-llm", "--debug", dir})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("scan with --debug should succeed, got: %v", err)
	}
}

func TestScanRunE_WithDebugDir(t *testing.T) {
	resetScanFlags()
	dir := makeMinimalGoProject(t)
	dbgDir := t.TempDir()

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"scan", "--no-llm", "--debug-dir", dbgDir, dir})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("scan with --debug-dir should succeed, got: %v", err)
	}
}

func TestScanRunE_WithThreshold_Pass(t *testing.T) {
	resetScanFlags()
	dir := makeMinimalGoProject(t)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"scan", "--no-llm", "--threshold", "0", dir})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("scan with --threshold 0 should succeed, got: %v", err)
	}
}

func TestScanRunE_WithBadge(t *testing.T) {
	resetScanFlags()
	dir := makeMinimalGoProject(t)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"scan", "--no-llm", "--badge", dir})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("scan with --badge should succeed, got: %v", err)
	}
}

func TestScanRunE_WithHTMLOutput(t *testing.T) {
	resetScanFlags()
	dir := makeMinimalGoProject(t)
	htmlFile := filepath.Join(t.TempDir(), "report.html")

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"scan", "--no-llm", "--output-html", htmlFile, dir})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("scan with --output-html should succeed, got: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "HTML report generated") {
		t.Errorf("expected HTML report message, got: %s", output)
	}
}

func TestScanRunE_VerboseNoCliAvailable(t *testing.T) {
	resetScanFlags()
	dir := makeMinimalGoProject(t)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"scan", "--no-llm", "-v", dir})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("scan with -v should succeed, got: %v", err)
	}
}
