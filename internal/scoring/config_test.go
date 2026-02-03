package scoring

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig_Structure(t *testing.T) {
	cfg := DefaultConfig()

	if cfg == nil {
		t.Fatal("DefaultConfig() returned nil")
	}

	if cfg.Categories == nil {
		t.Fatal("Categories map is nil")
	}

	// C1 should have 6 metrics
	c1 := cfg.Categories["C1"]
	if got := len(c1.Metrics); got != 6 {
		t.Errorf("C1 metrics count = %d, want 6", got)
	}
	if c1.Weight != 0.25 {
		t.Errorf("C1 weight = %v, want 0.25", c1.Weight)
	}
	if c1.Name != "Code Health" {
		t.Errorf("C1 name = %q, want %q", c1.Name, "Code Health")
	}

	// C2 should have 5 metrics
	c2 := cfg.Categories["C2"]
	if got := len(c2.Metrics); got != 5 {
		t.Errorf("C2 metrics count = %d, want 5", got)
	}
	if c2.Weight != 0.10 {
		t.Errorf("C2 weight = %v, want 0.10", c2.Weight)
	}
	if c2.Name != "Semantic Explicitness" {
		t.Errorf("C2 name = %q, want %q", c2.Name, "Semantic Explicitness")
	}

	// C3 should have 5 metrics
	c3 := cfg.Categories["C3"]
	if got := len(c3.Metrics); got != 5 {
		t.Errorf("C3 metrics count = %d, want 5", got)
	}
	if c3.Weight != 0.20 {
		t.Errorf("C3 weight = %v, want 0.20", c3.Weight)
	}
	if c3.Name != "Architecture" {
		t.Errorf("C3 name = %q, want %q", c3.Name, "Architecture")
	}

	// C6 should have 5 metrics
	c6 := cfg.Categories["C6"]
	if got := len(c6.Metrics); got != 5 {
		t.Errorf("C6 metrics count = %d, want 5", got)
	}
	if c6.Weight != 0.15 {
		t.Errorf("C6 weight = %v, want 0.15", c6.Weight)
	}
	if c6.Name != "Testing" {
		t.Errorf("C6 name = %q, want %q", c6.Name, "Testing")
	}

	// C4 should have 7 metrics
	c4 := cfg.Categories["C4"]
	if got := len(c4.Metrics); got != 7 {
		t.Errorf("C4 metrics count = %d, want 7", got)
	}
	if c4.Weight != 0.15 {
		t.Errorf("C4 weight = %v, want 0.15", c4.Weight)
	}
	if c4.Name != "Documentation Quality" {
		t.Errorf("C4 name = %q, want %q", c4.Name, "Documentation Quality")
	}

	// C5 should have 5 metrics
	c5 := cfg.Categories["C5"]
	if got := len(c5.Metrics); got != 5 {
		t.Errorf("C5 metrics count = %d, want 5", got)
	}
	if c5.Weight != 0.10 {
		t.Errorf("C5 weight = %v, want 0.10", c5.Weight)
	}
	if c5.Name != "Temporal Dynamics" {
		t.Errorf("C5 name = %q, want %q", c5.Name, "Temporal Dynamics")
	}

	// Should have 6 categories
	if got := len(cfg.Categories); got != 6 {
		t.Errorf("categories count = %d, want 6", got)
	}

	// Tiers should have 4 entries
	if got := len(cfg.Tiers); got != 4 {
		t.Errorf("tiers count = %d, want 4", got)
	}
}

func TestDefaultConfig_MetricWeightsSum(t *testing.T) {
	cfg := DefaultConfig()

	for name, cat := range cfg.Categories {
		sum := 0.0
		for _, m := range cat.Metrics {
			sum += m.Weight
		}
		if diff := sum - 1.0; diff > 0.001 || diff < -0.001 {
			t.Errorf("%s metric weights sum to %v, want 1.0", name, sum)
		}
	}
}

func TestDefaultConfig_BreakpointsSorted(t *testing.T) {
	cfg := DefaultConfig()

	for catName, cat := range cfg.Categories {
		for _, m := range cat.Metrics {
			if len(m.Breakpoints) == 0 {
				t.Errorf("%s metric %q has no breakpoints", catName, m.Name)
				continue
			}
			for i := 1; i < len(m.Breakpoints); i++ {
				if m.Breakpoints[i].Value <= m.Breakpoints[i-1].Value {
					t.Errorf("%s metric %q breakpoints not sorted by Value ascending at index %d: %v <= %v",
						catName, m.Name, i, m.Breakpoints[i].Value, m.Breakpoints[i-1].Value)
				}
			}
		}
	}
}

func TestDefaultConfig_TiersSorted(t *testing.T) {
	cfg := DefaultConfig()

	for i := 1; i < len(cfg.Tiers); i++ {
		if cfg.Tiers[i].MinScore >= cfg.Tiers[i-1].MinScore {
			t.Errorf("tiers not sorted descending at index %d: %v >= %v",
				i, cfg.Tiers[i].MinScore, cfg.Tiers[i-1].MinScore)
		}
	}
}

func TestDefaultConfig_MetricNames(t *testing.T) {
	cfg := DefaultConfig()

	c1Names := map[string]bool{
		"complexity_avg":        false,
		"func_length_avg":      false,
		"file_size_avg":        false,
		"afferent_coupling_avg": false,
		"efferent_coupling_avg": false,
		"duplication_rate":     false,
	}
	c1 := cfg.Categories["C1"]
	for _, m := range c1.Metrics {
		if _, ok := c1Names[m.Name]; !ok {
			t.Errorf("unexpected C1 metric %q", m.Name)
		}
		c1Names[m.Name] = true
	}
	for name, found := range c1Names {
		if !found {
			t.Errorf("missing C1 metric %q", name)
		}
	}

	c2Names := map[string]bool{
		"type_annotation_coverage": false,
		"naming_consistency":       false,
		"magic_number_ratio":       false,
		"type_strictness":          false,
		"null_safety":              false,
	}
	c2 := cfg.Categories["C2"]
	for _, m := range c2.Metrics {
		if _, ok := c2Names[m.Name]; !ok {
			t.Errorf("unexpected C2 metric %q", m.Name)
		}
		c2Names[m.Name] = true
	}
	for name, found := range c2Names {
		if !found {
			t.Errorf("missing C2 metric %q", name)
		}
	}

	c3Names := map[string]bool{
		"max_dir_depth":        false,
		"module_fanout_avg":    false,
		"circular_deps":       false,
		"import_complexity_avg": false,
		"dead_exports":        false,
	}
	c3 := cfg.Categories["C3"]
	for _, m := range c3.Metrics {
		if _, ok := c3Names[m.Name]; !ok {
			t.Errorf("unexpected C3 metric %q", m.Name)
		}
		c3Names[m.Name] = true
	}
	for name, found := range c3Names {
		if !found {
			t.Errorf("missing C3 metric %q", name)
		}
	}

	c6Names := map[string]bool{
		"test_to_code_ratio":   false,
		"coverage_percent":     false,
		"test_isolation":       false,
		"assertion_density_avg": false,
		"test_file_ratio":     false,
	}
	c6 := cfg.Categories["C6"]
	for _, m := range c6.Metrics {
		if _, ok := c6Names[m.Name]; !ok {
			t.Errorf("unexpected C6 metric %q", m.Name)
		}
		c6Names[m.Name] = true
	}
	for name, found := range c6Names {
		if !found {
			t.Errorf("missing C6 metric %q", name)
		}
	}

	c4Names := map[string]bool{
		"readme_word_count":    false,
		"comment_density":      false,
		"api_doc_coverage":     false,
		"changelog_present":    false,
		"examples_present":     false,
		"contributing_present": false,
		"diagrams_present":     false,
	}
	c4 := cfg.Categories["C4"]
	for _, m := range c4.Metrics {
		if _, ok := c4Names[m.Name]; !ok {
			t.Errorf("unexpected C4 metric %q", m.Name)
		}
		c4Names[m.Name] = true
	}
	for name, found := range c4Names {
		if !found {
			t.Errorf("missing C4 metric %q", name)
		}
	}

	c5Names := map[string]bool{
		"churn_rate":            false,
		"temporal_coupling_pct": false,
		"author_fragmentation":  false,
		"commit_stability":      false,
		"hotspot_concentration": false,
	}
	c5 := cfg.Categories["C5"]
	for _, m := range c5.Metrics {
		if _, ok := c5Names[m.Name]; !ok {
			t.Errorf("unexpected C5 metric %q", m.Name)
		}
		c5Names[m.Name] = true
	}
	for name, found := range c5Names {
		if !found {
			t.Errorf("missing C5 metric %q", name)
		}
	}
}

func TestDefaultConfig_CategoryAccessor(t *testing.T) {
	cfg := DefaultConfig()

	c1 := cfg.Category("C1")
	if c1.Name != "Code Health" {
		t.Errorf("Category(C1).Name = %q, want Code Health", c1.Name)
	}

	missing := cfg.Category("C99")
	if missing.Name != "" {
		t.Errorf("Category(C99).Name = %q, want empty", missing.Name)
	}
}

func TestLoadConfig_EmptyPath(t *testing.T) {
	cfg, err := LoadConfig("")
	if err != nil {
		t.Fatalf("LoadConfig('') returned error: %v", err)
	}
	if cfg == nil {
		t.Fatal("LoadConfig('') returned nil")
	}
	// Should be equivalent to DefaultConfig
	c1 := cfg.Categories["C1"]
	if c1.Weight != 0.25 {
		t.Errorf("C1 weight = %v, want 0.25", c1.Weight)
	}
}

func TestLoadConfig_YAMLOverride(t *testing.T) {
	yamlContent := `
categories:
  C1:
    weight: 0.40
    name: "Code Health"
    metrics:
      - name: complexity_avg
        weight: 1.0
        breakpoints:
          - value: 1
            score: 10
          - value: 50
            score: 1
`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(yamlContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig() returned error: %v", err)
	}

	// C1 weight should be overridden
	c1 := cfg.Categories["C1"]
	if c1.Weight != 0.40 {
		t.Errorf("C1 weight = %v, want 0.40", c1.Weight)
	}

	// C1 metrics should be overridden (only 1 metric now)
	if len(c1.Metrics) != 1 {
		t.Errorf("C1 metrics count = %d, want 1", len(c1.Metrics))
	}

	// C3 should retain defaults since not in YAML
	c3 := cfg.Categories["C3"]
	if c3.Weight != 0.20 {
		t.Errorf("C3 weight = %v, want 0.20 (default)", c3.Weight)
	}
}

func TestLoadConfig_FileNotFound(t *testing.T) {
	_, err := LoadConfig("/nonexistent/config.yaml")
	if err == nil {
		t.Fatal("LoadConfig() should return error for missing file")
	}
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")
	if err := os.WriteFile(path, []byte("{{{{not yaml"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadConfig(path)
	if err == nil {
		t.Fatal("LoadConfig() should return error for invalid YAML")
	}
}
