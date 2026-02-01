package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ingo/agent-readyness/internal/scoring"
)

func TestLoadProjectConfig_ValidYml(t *testing.T) {
	tmpDir := t.TempDir()

	content := `version: 1
scoring:
  weights:
    C1: 0.30
    C2: 0.20
  threshold: 7.0
languages:
  - go
  - python
`
	if err := os.WriteFile(filepath.Join(tmpDir, ".arsrc.yml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadProjectConfig(tmpDir, "")
	if err != nil {
		t.Fatalf("LoadProjectConfig() error: %v", err)
	}

	if cfg == nil {
		t.Fatal("expected non-nil config")
	}

	if cfg.Version != 1 {
		t.Errorf("Version = %d, want 1", cfg.Version)
	}

	if cfg.Scoring.Weights["C1"] != 0.30 {
		t.Errorf("C1 weight = %v, want 0.30", cfg.Scoring.Weights["C1"])
	}

	if cfg.Scoring.Threshold != 7.0 {
		t.Errorf("Threshold = %v, want 7.0", cfg.Scoring.Threshold)
	}

	if len(cfg.Languages) != 2 {
		t.Errorf("Languages count = %d, want 2", len(cfg.Languages))
	}
}

func TestLoadProjectConfig_MissingFile(t *testing.T) {
	tmpDir := t.TempDir()

	cfg, err := LoadProjectConfig(tmpDir, "")
	if err != nil {
		t.Fatalf("LoadProjectConfig() error: %v", err)
	}
	if cfg != nil {
		t.Errorf("expected nil config for missing file, got %+v", cfg)
	}
}

func TestLoadProjectConfig_InvalidWeight(t *testing.T) {
	tmpDir := t.TempDir()

	content := `version: 1
scoring:
  weights:
    C1: -0.5
`
	if err := os.WriteFile(filepath.Join(tmpDir, ".arsrc.yml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadProjectConfig(tmpDir, "")
	if err == nil {
		t.Fatal("expected error for negative weight")
	}
}

func TestLoadProjectConfig_InvalidVersion(t *testing.T) {
	tmpDir := t.TempDir()

	content := `version: 99
`
	if err := os.WriteFile(filepath.Join(tmpDir, ".arsrc.yml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadProjectConfig(tmpDir, "")
	if err == nil {
		t.Fatal("expected error for unsupported version")
	}
}

func TestLoadProjectConfig_ExplicitPath(t *testing.T) {
	tmpDir := t.TempDir()

	content := `version: 1
scoring:
  weights:
    C6: 0.40
`
	customPath := filepath.Join(tmpDir, "custom-config.yml")
	if err := os.WriteFile(customPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadProjectConfig(tmpDir, customPath)
	if err != nil {
		t.Fatalf("LoadProjectConfig() error: %v", err)
	}

	if cfg.Scoring.Weights["C6"] != 0.40 {
		t.Errorf("C6 weight = %v, want 0.40", cfg.Scoring.Weights["C6"])
	}
}

func TestProjectConfig_ApplyToScoringConfig(t *testing.T) {
	sc := scoring.DefaultConfig()

	pc := &ProjectConfig{
		Version: 1,
		Scoring: ScoringOverrides{
			Weights: map[string]float64{
				"C1": 0.50,
				"C3": 0.30,
			},
		},
	}

	pc.ApplyToScoringConfig(sc)

	if sc.Categories["C1"].Weight != 0.50 {
		t.Errorf("C1 weight = %v, want 0.50", sc.Categories["C1"].Weight)
	}
	if sc.Categories["C3"].Weight != 0.30 {
		t.Errorf("C3 weight = %v, want 0.30", sc.Categories["C3"].Weight)
	}
	// C6 should remain at default
	if sc.Categories["C6"].Weight != 0.15 {
		t.Errorf("C6 weight = %v, want 0.15 (default)", sc.Categories["C6"].Weight)
	}
}

func TestProjectConfig_YamlExtension(t *testing.T) {
	tmpDir := t.TempDir()

	content := `version: 1
scoring:
  threshold: 5.0
`
	if err := os.WriteFile(filepath.Join(tmpDir, ".arsrc.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadProjectConfig(tmpDir, "")
	if err != nil {
		t.Fatalf("LoadProjectConfig() error: %v", err)
	}

	if cfg == nil {
		t.Fatal("expected non-nil config for .arsrc.yaml")
	}

	if cfg.Scoring.Threshold != 5.0 {
		t.Errorf("Threshold = %v, want 5.0", cfg.Scoring.Threshold)
	}
}

func TestValidate_NegativeThreshold(t *testing.T) {
	cfg := &ProjectConfig{
		Version: 1,
		Scoring: ScoringOverrides{
			Threshold: -1.0,
		},
	}

	if err := cfg.Validate(); err == nil {
		t.Error("expected error for negative threshold")
	}
}
