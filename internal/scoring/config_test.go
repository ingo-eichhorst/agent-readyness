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

	// C1 should have 6 metrics
	if got := len(cfg.C1.Metrics); got != 6 {
		t.Errorf("C1 metrics count = %d, want 6", got)
	}
	if cfg.C1.Weight != 0.25 {
		t.Errorf("C1 weight = %v, want 0.25", cfg.C1.Weight)
	}
	if cfg.C1.Name != "Code Health" {
		t.Errorf("C1 name = %q, want %q", cfg.C1.Name, "Code Health")
	}

	// C3 should have 5 metrics
	if got := len(cfg.C3.Metrics); got != 5 {
		t.Errorf("C3 metrics count = %d, want 5", got)
	}
	if cfg.C3.Weight != 0.20 {
		t.Errorf("C3 weight = %v, want 0.20", cfg.C3.Weight)
	}
	if cfg.C3.Name != "Architecture" {
		t.Errorf("C3 name = %q, want %q", cfg.C3.Name, "Architecture")
	}

	// C6 should have 5 metrics
	if got := len(cfg.C6.Metrics); got != 5 {
		t.Errorf("C6 metrics count = %d, want 5", got)
	}
	if cfg.C6.Weight != 0.15 {
		t.Errorf("C6 weight = %v, want 0.15", cfg.C6.Weight)
	}
	if cfg.C6.Name != "Testing" {
		t.Errorf("C6 name = %q, want %q", cfg.C6.Name, "Testing")
	}

	// Tiers should have 4 entries
	if got := len(cfg.Tiers); got != 4 {
		t.Errorf("tiers count = %d, want 4", got)
	}
}

func TestDefaultConfig_MetricWeightsSum(t *testing.T) {
	cfg := DefaultConfig()

	categories := []struct {
		name    string
		metrics []MetricThresholds
	}{
		{"C1", cfg.C1.Metrics},
		{"C3", cfg.C3.Metrics},
		{"C6", cfg.C6.Metrics},
	}

	for _, cat := range categories {
		sum := 0.0
		for _, m := range cat.metrics {
			sum += m.Weight
		}
		if diff := sum - 1.0; diff > 0.001 || diff < -0.001 {
			t.Errorf("%s metric weights sum to %v, want 1.0", cat.name, sum)
		}
	}
}

func TestDefaultConfig_BreakpointsSorted(t *testing.T) {
	cfg := DefaultConfig()

	allMetrics := append(cfg.C1.Metrics, cfg.C3.Metrics...)
	allMetrics = append(allMetrics, cfg.C6.Metrics...)

	for _, m := range allMetrics {
		if len(m.Breakpoints) == 0 {
			t.Errorf("metric %q has no breakpoints", m.Name)
			continue
		}
		for i := 1; i < len(m.Breakpoints); i++ {
			if m.Breakpoints[i].Value <= m.Breakpoints[i-1].Value {
				t.Errorf("metric %q breakpoints not sorted by Value ascending at index %d: %v <= %v",
					m.Name, i, m.Breakpoints[i].Value, m.Breakpoints[i-1].Value)
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
	for _, m := range cfg.C1.Metrics {
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

	c3Names := map[string]bool{
		"max_dir_depth":        false,
		"module_fanout_avg":    false,
		"circular_deps":       false,
		"import_complexity_avg": false,
		"dead_exports":        false,
	}
	for _, m := range cfg.C3.Metrics {
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
	for _, m := range cfg.C6.Metrics {
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
	if cfg.C1.Weight != 0.25 {
		t.Errorf("C1 weight = %v, want 0.25", cfg.C1.Weight)
	}
}

func TestLoadConfig_YAMLOverride(t *testing.T) {
	yamlContent := `
c1:
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
	if cfg.C1.Weight != 0.40 {
		t.Errorf("C1 weight = %v, want 0.40", cfg.C1.Weight)
	}

	// C1 metrics should be overridden (only 1 metric now)
	if len(cfg.C1.Metrics) != 1 {
		t.Errorf("C1 metrics count = %d, want 1", len(cfg.C1.Metrics))
	}

	// C3 should retain defaults since not in YAML
	if cfg.C3.Weight != 0.20 {
		t.Errorf("C3 weight = %v, want 0.20 (default)", cfg.C3.Weight)
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
