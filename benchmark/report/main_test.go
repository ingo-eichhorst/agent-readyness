package main

import (
	"encoding/json"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// ---- loadYAML ----

func TestLoadYAML_Valid(t *testing.T) {
	dir := t.TempDir()
	content := `repos:
  - name: foo
    language: go
    url: https://github.com/foo/bar
    commit: abc123
    size: small
    expected_quality: high
`
	if err := os.WriteFile(filepath.Join(dir, "benchmark.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := loadYAML(filepath.Join(dir, "benchmark.yaml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Repos) != 1 {
		t.Fatalf("got %d repos, want 1", len(cfg.Repos))
	}
	r := cfg.Repos[0]
	if r.Name != "foo" || r.Language != "go" || r.Commit != "abc123" {
		t.Errorf("unexpected repo fields: %+v", r)
	}
}

func TestLoadYAML_Missing(t *testing.T) {
	_, err := loadYAML("/nonexistent/path/benchmark.yaml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoadYAML_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "bad.yaml"), []byte(":\tinvalid: yaml: ["), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := loadYAML(filepath.Join(dir, "bad.yaml"))
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

// ---- loadGoldens ----

func writeGoldenJSON(t *testing.T, dir, lang, name string, gf goldenFile) {
	t.Helper()
	langDir := filepath.Join(dir, lang)
	if err := os.MkdirAll(langDir, 0755); err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(gf)
	if err := os.WriteFile(filepath.Join(langDir, name+".json"), data, 0644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadGoldens_Basic(t *testing.T) {
	dir := t.TempDir()
	writeGoldenJSON(t, dir, "go", "cobra", goldenFile{
		CompositeScore: 7.5,
		Tier:           "Agent-Assisted",
		Categories: []goldenCategory{
			{Name: "C1", Score: 8.0},
			{Name: "C7", Score: -1},
		},
	})
	meta := map[string]yamlRepo{
		"cobra": {Name: "cobra", Language: "go", URL: "https://github.com/spf13/cobra", Commit: "v1.9.1"},
	}
	repos, err := loadGoldens(dir, meta)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repos) != 1 {
		t.Fatalf("got %d repos, want 1", len(repos))
	}
	r := repos[0]
	if r.Name != "cobra" || r.Lang != "go" {
		t.Errorf("unexpected repo: %+v", r)
	}
	if math.Abs(r.Score-7.5) > 1e-9 {
		t.Errorf("score: got %v, want 7.5", r.Score)
	}
	if r.Tier != "Agent-Assisted" {
		t.Errorf("tier: got %q, want Agent-Assisted", r.Tier)
	}
	if r.Cats["C1"] != 8.0 {
		t.Errorf("C1: got %v, want 8.0", r.Cats["C1"])
	}
	if r.Cats["C7"] != -1 {
		t.Errorf("C7: got %v, want -1", r.Cats["C7"])
	}
}

func TestLoadGoldens_MissingLangDir(t *testing.T) {
	dir := t.TempDir()
	// Only create "go" subdir, python/typescript are absent — should not error
	writeGoldenJSON(t, dir, "go", "uuid", goldenFile{CompositeScore: 8.0, Tier: "Agent-Ready"})
	repos, err := loadGoldens(dir, map[string]yamlRepo{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repos) != 1 {
		t.Errorf("got %d repos, want 1", len(repos))
	}
}

func TestLoadGoldens_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	langDir := filepath.Join(dir, "go")
	if err := os.MkdirAll(langDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(langDir, "bad.json"), []byte("{invalid}"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := loadGoldens(dir, map[string]yamlRepo{})
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestLoadGoldens_SkipsNonJSON(t *testing.T) {
	dir := t.TempDir()
	langDir := filepath.Join(dir, "go")
	if err := os.MkdirAll(langDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(langDir, "README.md"), []byte("# ignore"), 0644); err != nil {
		t.Fatal(err)
	}
	repos, err := loadGoldens(dir, map[string]yamlRepo{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repos) != 0 {
		t.Errorf("expected 0 repos, got %d", len(repos))
	}
}

func TestLoadGoldens_SortOrder(t *testing.T) {
	dir := t.TempDir()
	writeGoldenJSON(t, dir, "typescript", "zod", goldenFile{CompositeScore: 8.0, Tier: "Agent-Ready"})
	writeGoldenJSON(t, dir, "go", "cobra", goldenFile{CompositeScore: 7.5, Tier: "Agent-Assisted"})
	writeGoldenJSON(t, dir, "python", "httpx", goldenFile{CompositeScore: 7.0, Tier: "Agent-Assisted"})
	// Two Go repos to exercise the secondary (name) sort within a language
	writeGoldenJSON(t, dir, "go", "uuid", goldenFile{CompositeScore: 8.6, Tier: "Agent-Ready"})

	repos, err := loadGoldens(dir, map[string]yamlRepo{})
	if err != nil {
		t.Fatal(err)
	}
	if len(repos) != 4 {
		t.Fatalf("got %d repos, want 4", len(repos))
	}
	// Language order: go, python, typescript
	if repos[0].Lang != "go" || repos[1].Lang != "go" || repos[2].Lang != "python" || repos[3].Lang != "typescript" {
		t.Errorf("wrong language order: %v", repos)
	}
	// Within go: cobra < uuid alphabetically
	if repos[0].Name != "cobra" || repos[1].Name != "uuid" {
		t.Errorf("wrong name order within go: %v %v", repos[0].Name, repos[1].Name)
	}
}

func TestLoadGoldens_FileReadError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("cannot test file permission errors as root")
	}
	dir := t.TempDir()
	langDir := filepath.Join(dir, "go")
	if err := os.MkdirAll(langDir, 0755); err != nil {
		t.Fatal(err)
	}
	// Write a valid JSON file then remove read permission.
	path := filepath.Join(langDir, "noperm.json")
	if err := os.WriteFile(path, []byte(`{"composite_score":7,"tier":"T","categories":[]}`), 0000); err != nil {
		t.Fatal(err)
	}
	_, err := loadGoldens(dir, map[string]yamlRepo{})
	if err == nil {
		t.Fatal("expected error reading unreadable file")
	}
}

// ---- computeStats ----

func TestComputeStats_Empty(t *testing.T) {
	s := computeStats(nil)
	if s.Total != 0 {
		t.Errorf("Total: got %d, want 0", s.Total)
	}
	if s.RangeGo != "—" || s.RangePython != "—" || s.RangeTS != "—" {
		t.Errorf("expected — ranges for empty input")
	}
}

func TestComputeStats_TierCounts(t *testing.T) {
	repos := []repoData{
		{Lang: "go", Score: 8.5, Tier: "Agent-Ready"},
		{Lang: "go", Score: 7.0, Tier: "Agent-Assisted"},
		{Lang: "python", Score: 5.0, Tier: "Agent-Limited"},
		{Lang: "typescript", Score: 3.0, Tier: "Agent-Blocked"},
	}
	s := computeStats(repos)
	if s.Total != 4 {
		t.Errorf("Total: got %d, want 4", s.Total)
	}
	if s.AgentReady != 1 {
		t.Errorf("AgentReady: got %d, want 1", s.AgentReady)
	}
	if s.AgentAssisted != 1 {
		t.Errorf("AgentAssisted: got %d, want 1", s.AgentAssisted)
	}
	if s.AgentLimited != 1 {
		t.Errorf("AgentLimited: got %d, want 1", s.AgentLimited)
	}
	if s.AgentBlocked != 1 {
		t.Errorf("AgentBlocked: got %d, want 1", s.AgentBlocked)
	}
}

func TestComputeStats_Averages(t *testing.T) {
	repos := []repoData{
		{Lang: "go", Score: 8.0, Tier: "Agent-Ready"},
		{Lang: "go", Score: 6.0, Tier: "Agent-Assisted"},
		{Lang: "python", Score: 7.0, Tier: "Agent-Assisted"},
		{Lang: "typescript", Score: 9.0, Tier: "Agent-Ready"},
	}
	s := computeStats(repos)
	if math.Abs(s.AvgGo-7.0) > 1e-9 {
		t.Errorf("AvgGo: got %v, want 7.0", s.AvgGo)
	}
	if math.Abs(s.AvgPython-7.0) > 1e-9 {
		t.Errorf("AvgPython: got %v, want 7.0", s.AvgPython)
	}
	if math.Abs(s.AvgTS-9.0) > 1e-9 {
		t.Errorf("AvgTS: got %v, want 9.0", s.AvgTS)
	}
}

func TestComputeStats_Range(t *testing.T) {
	repos := []repoData{
		{Lang: "go", Score: 6.0, Tier: "Agent-Assisted"},
		{Lang: "go", Score: 9.0, Tier: "Agent-Ready"},
	}
	s := computeStats(repos)
	if s.RangeGo != "6.00–9.00" {
		t.Errorf("RangeGo: got %q, want %q", s.RangeGo, "6.00–9.00")
	}
}

// ---- render ----

func TestRender_ValidOutput(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "report.html")
	data := templateData{
		GeneratedAt: "2026-01-01 00:00 UTC",
		Repos: []repoData{
			{
				Name: "cobra", Lang: "go", URL: "https://github.com/spf13/cobra",
				Commit: "v1.9.1", Score: 7.97, Tier: "Agent-Assisted",
				Cats: map[string]float64{"C1": 7.1, "C7": -1},
			},
		},
		Stats: stats{Total: 1, AgentAssisted: 1, AvgGo: 7.97, RangeGo: "7.97–7.97"},
	}
	if err := render(outPath, data); err != nil {
		t.Fatalf("render error: %v", err)
	}
	content, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}
	html := string(content)
	if !strings.Contains(html, "<!DOCTYPE html>") {
		t.Error("output missing <!DOCTYPE html>")
	}
	if !strings.Contains(html, "ARS Benchmark Report") {
		t.Error("output missing page title")
	}
	if !strings.Contains(html, "cobra") {
		t.Error("output missing repo name")
	}
}

func TestRender_WriteError(t *testing.T) {
	err := render("/nonexistent/dir/report.html", templateData{})
	if err == nil {
		t.Fatal("expected error writing to invalid path")
	}
}

func TestRender_TemplateParseError(t *testing.T) {
	orig := htmlTemplate
	t.Cleanup(func() { htmlTemplate = orig })
	htmlTemplate = `{{invalid template`

	dir := t.TempDir()
	err := render(filepath.Join(dir, "report.html"), templateData{})
	if err == nil {
		t.Fatal("expected parse error for invalid template")
	}
	if !strings.Contains(err.Error(), "parse template") {
		t.Errorf("unexpected error: %v", err)
	}
}

// ---- findRoot ----

// ---- run ----

func makeMinimalRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	// go.mod (needed by findRoot but run() receives root directly)
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module test\n"), 0644); err != nil {
		t.Fatal(err)
	}
	// benchmark.yaml
	yaml := "repos:\n  - name: foo\n    language: go\n    url: https://github.com/foo/foo\n    commit: abc\n"
	benchDir := filepath.Join(root, "benchmark")
	if err := os.MkdirAll(benchDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(benchDir, "benchmark.yaml"), []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}
	// One golden file
	writeGoldenJSON(t, filepath.Join(benchDir, "golden"), "go", "foo", goldenFile{
		CompositeScore: 7.5,
		Tier:           "Agent-Assisted",
		Categories:     []goldenCategory{{Name: "C1", Score: 8.0}},
	})
	return root
}

func TestRun_Success(t *testing.T) {
	root := makeMinimalRoot(t)
	if err := run(root); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "benchmark", "report.html")); err != nil {
		t.Errorf("report.html not created: %v", err)
	}
}

func TestRun_MissingYAML(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "benchmark"), 0755); err != nil {
		t.Fatal(err)
	}
	err := run(root)
	if err == nil {
		t.Fatal("expected error for missing benchmark.yaml")
	}
}

func TestRun_LoadGoldensError(t *testing.T) {
	root := t.TempDir()
	benchDir := filepath.Join(root, "benchmark")
	if err := os.MkdirAll(benchDir, 0755); err != nil {
		t.Fatal(err)
	}
	yaml := "repos:\n  - name: foo\n    language: go\n"
	if err := os.WriteFile(filepath.Join(benchDir, "benchmark.yaml"), []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}
	// Place a file named "go" where the golden/go directory is expected.
	goldenDir := filepath.Join(benchDir, "golden")
	if err := os.MkdirAll(goldenDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(goldenDir, "go"), []byte("not a dir"), 0644); err != nil {
		t.Fatal(err)
	}
	err := run(root)
	if err == nil {
		t.Fatal("expected error when loadGoldens fails")
	}
}

func TestRun_NoGoldens(t *testing.T) {
	root := t.TempDir()
	benchDir := filepath.Join(root, "benchmark")
	if err := os.MkdirAll(benchDir, 0755); err != nil {
		t.Fatal(err)
	}
	yaml := "repos:\n  - name: foo\n    language: go\n"
	if err := os.WriteFile(filepath.Join(benchDir, "benchmark.yaml"), []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}
	err := run(root)
	if err == nil {
		t.Fatal("expected error for empty golden dir")
	}
	if !strings.Contains(err.Error(), "no golden files") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRun_RenderError(t *testing.T) {
	orig := htmlTemplate
	t.Cleanup(func() { htmlTemplate = orig })
	htmlTemplate = `{{invalid`

	root := makeMinimalRoot(t)
	err := run(root)
	if err == nil {
		t.Fatal("expected render error")
	}
}

// ---- main (subprocess) ----

func TestMain_FindRootError(t *testing.T) {
	// Run main() in a subprocess cwd with no go.mod ancestor.
	if os.Getenv("TEST_MAIN_TRIGGER") == "find_root" {
		main()
		return
	}
	base := t.TempDir()
	deep := filepath.Join(base, "a", "b")
	if err := os.MkdirAll(deep, 0755); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestMain_FindRootError")
	cmd.Env = append(os.Environ(), "TEST_MAIN_TRIGGER=find_root")
	cmd.Dir = deep
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected non-zero exit from main")
	}
	if exitErr, ok := err.(*exec.ExitError); !ok || exitErr.ExitCode() != 1 {
		t.Errorf("expected exit code 1, got %v\noutput: %s", err, out)
	}
}

func TestMain_RunError(t *testing.T) {
	// Run main() from a dir where findRoot succeeds but run() fails (no benchmark.yaml).
	if os.Getenv("TEST_MAIN_TRIGGER") == "run_error" {
		main()
		return
	}
	// Create a temp dir with go.mod but no benchmark/benchmark.yaml.
	base := t.TempDir()
	if err := os.WriteFile(filepath.Join(base, "go.mod"), []byte("module test\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestMain_RunError")
	cmd.Env = append(os.Environ(), "TEST_MAIN_TRIGGER=run_error")
	cmd.Dir = base
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected non-zero exit from main")
	}
	if exitErr, ok := err.(*exec.ExitError); !ok || exitErr.ExitCode() != 1 {
		t.Errorf("expected exit code 1, got %v\noutput: %s", err, out)
	}
}

// ---- fatal (subprocess) ----

func TestFatal_ExitsWithCode1(t *testing.T) {
	// Re-invoke the test binary with a special env var to trigger fatal.
	if os.Getenv("TEST_FATAL_TRIGGER") == "1" {
		fatal("test error %s", "msg")
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestFatal_ExitsWithCode1")
	cmd.Env = append(os.Environ(), "TEST_FATAL_TRIGGER=1")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected non-zero exit")
	}
	if exitErr, ok := err.(*exec.ExitError); !ok || exitErr.ExitCode() != 1 {
		t.Errorf("expected exit code 1, got: %v", err)
	}
	if !strings.Contains(string(out), "report: test error msg") {
		t.Errorf("unexpected output: %s", out)
	}
}

// ---- findRoot ----

func TestFindRoot_FindsGoMod(t *testing.T) {
	// findRoot uses os.Getwd, so it will find the real repo root during tests.
	root, err := findRoot()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(root, "go.mod")); statErr != nil {
		t.Errorf("go.mod not found at %s", root)
	}
}

func TestFindRoot_NotFound(t *testing.T) {
	// Change cwd to a temp dir with no go.mod in any ancestor.
	// We create a deeply nested temp dir to avoid accidentally hitting a real go.mod.
	base := t.TempDir()
	deep := filepath.Join(base, "a", "b", "c")
	if err := os.MkdirAll(deep, 0755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(deep)
	_, err := findRoot()
	if err == nil {
		t.Fatal("expected error when go.mod not found")
	}
	if !strings.Contains(err.Error(), "go.mod not found") {
		t.Errorf("unexpected error: %v", err)
	}
}

// ---- loadGoldens ReadDir error (non-IsNotExist) ----

func TestLoadGoldens_ReadDirError(t *testing.T) {
	dir := t.TempDir()
	// Create a regular file named "python" where a directory is expected.
	// os.ReadDir on a file returns an error that is not os.IsNotExist.
	if err := os.WriteFile(filepath.Join(dir, "python"), []byte("not a dir"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := loadGoldens(dir, map[string]yamlRepo{})
	if err == nil {
		t.Fatal("expected error when ReadDir fails on a file")
	}
}
