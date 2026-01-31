package output

import (
	"bytes"
	"strings"
	"testing"

	"github.com/ingo/agent-readyness/pkg/types"
)

func newTestResult() *types.ScanResult {
	return &types.ScanResult{
		RootDir:        "/test/project",
		TotalFiles:     5,
		SourceCount:    2,
		TestCount:      1,
		GeneratedCount: 1,
		VendorCount:    1,
		GitignoreCount: 0,
		Files: []types.DiscoveredFile{
			{Path: "/test/project/main.go", RelPath: "main.go", Class: types.ClassSource},
			{Path: "/test/project/util.go", RelPath: "util.go", Class: types.ClassSource},
			{Path: "/test/project/main_test.go", RelPath: "main_test.go", Class: types.ClassTest},
			{Path: "/test/project/gen.go", RelPath: "gen.go", Class: types.ClassGenerated},
			{Path: "/test/project/vendor/dep.go", RelPath: "vendor/dep.go", Class: types.ClassExcluded, ExcludeReason: "vendor"},
		},
	}
}

func TestRenderSummary(t *testing.T) {
	var buf bytes.Buffer
	result := newTestResult()

	RenderSummary(&buf, result, false)
	out := buf.String()

	checks := []string{
		"ARS Scan: /test/project",
		"Go files discovered: 5",
		"Source files:",
		"Test files:",
		"Generated (excluded):",
		"Vendor (excluded):",
	}

	for _, check := range checks {
		if !strings.Contains(out, check) {
			t.Errorf("output missing %q\nGot:\n%s", check, out)
		}
	}

	// Gitignored should NOT appear (count is 0)
	if strings.Contains(out, "Gitignored") {
		t.Error("output should not contain 'Gitignored' when count is 0")
	}

	// Verbose content should NOT appear
	if strings.Contains(out, "Discovered files:") {
		t.Error("non-verbose output should not contain 'Discovered files:'")
	}
}

func TestRenderSummaryVerbose(t *testing.T) {
	var buf bytes.Buffer
	result := newTestResult()

	RenderSummary(&buf, result, true)
	out := buf.String()

	// Should have file listing
	if !strings.Contains(out, "Discovered files:") {
		t.Error("verbose output missing 'Discovered files:' header")
	}

	// Individual files should appear
	fileChecks := []string{
		"[source] main.go",
		"[test] main_test.go",
		"[generated] gen.go",
		"[excluded] vendor/dep.go (vendor)",
	}

	for _, check := range fileChecks {
		if !strings.Contains(out, check) {
			t.Errorf("verbose output missing %q\nGot:\n%s", check, out)
		}
	}
}
