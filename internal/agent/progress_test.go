package agent

import (
	"os"
	"testing"
)

func TestNewC7Progress(t *testing.T) {
	ids := []string{"m1", "m2", "m3"}
	names := []string{"Metric 1", "Metric 2", "Metric 3"}

	p := NewC7Progress(os.Stderr, ids, names)
	if p == nil {
		t.Fatal("NewC7Progress returned nil")
	}

	if len(p.metrics) != 3 {
		t.Errorf("got %d metrics, want 3", len(p.metrics))
	}

	// Check that names were assigned correctly
	p.mu.Lock()
	for i, id := range ids {
		if p.metrics[id].Name != names[i] {
			t.Errorf("metric %s name = %q, want %q", id, p.metrics[id].Name, names[i])
		}
	}
	p.mu.Unlock()
}

func TestNewC7Progress_NilNames(t *testing.T) {
	ids := []string{"m1", "m2"}

	p := NewC7Progress(os.Stderr, ids, nil)
	if p == nil {
		t.Fatal("NewC7Progress returned nil")
	}

	// Names should default to IDs when not provided
	p.mu.Lock()
	for _, id := range ids {
		if p.metrics[id].Name != id {
			t.Errorf("metric %s name = %q, want %q (ID as default)", id, p.metrics[id].Name, id)
		}
	}
	p.mu.Unlock()
}

func TestC7Progress_SetMetricRunning(t *testing.T) {
	ids := []string{"m1"}
	p := NewC7Progress(os.Stderr, ids, nil)

	p.SetMetricRunning("m1", 5)

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.metrics["m1"].Status != StatusRunning {
		t.Errorf("status = %v, want Running", p.metrics["m1"].Status)
	}
	if p.metrics["m1"].TotalSamples != 5 {
		t.Errorf("TotalSamples = %d, want 5", p.metrics["m1"].TotalSamples)
	}
	if p.metrics["m1"].CurrentSample != 0 {
		t.Errorf("CurrentSample = %d, want 0", p.metrics["m1"].CurrentSample)
	}
}

func TestC7Progress_SetMetricSample(t *testing.T) {
	ids := []string{"m1"}
	p := NewC7Progress(os.Stderr, ids, nil)

	p.SetMetricRunning("m1", 5)
	p.SetMetricSample("m1", 3)

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.metrics["m1"].CurrentSample != 3 {
		t.Errorf("CurrentSample = %d, want 3", p.metrics["m1"].CurrentSample)
	}
}

func TestC7Progress_SetMetricComplete(t *testing.T) {
	ids := []string{"m1"}
	p := NewC7Progress(os.Stderr, ids, nil)

	p.SetMetricComplete("m1", 8)

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.metrics["m1"].Status != StatusComplete {
		t.Errorf("status = %v, want Complete", p.metrics["m1"].Status)
	}
	if p.metrics["m1"].Score != 8 {
		t.Errorf("Score = %d, want 8", p.metrics["m1"].Score)
	}
}

func TestC7Progress_SetMetricFailed(t *testing.T) {
	ids := []string{"m1"}
	p := NewC7Progress(os.Stderr, ids, nil)

	p.SetMetricFailed("m1", "timeout exceeded")

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.metrics["m1"].Status != StatusFailed {
		t.Errorf("status = %v, want Failed", p.metrics["m1"].Status)
	}
	if p.metrics["m1"].Error != "timeout exceeded" {
		t.Errorf("Error = %q, want %q", p.metrics["m1"].Error, "timeout exceeded")
	}
}

func TestC7Progress_AddTokens(t *testing.T) {
	p := NewC7Progress(os.Stderr, nil, nil)

	p.AddTokens(100)
	p.AddTokens(200)

	if got := p.TotalTokens(); got != 300 {
		t.Errorf("TotalTokens() = %d, want 300", got)
	}
}

func TestC7Progress_TotalTokens_ThreadSafe(t *testing.T) {
	p := NewC7Progress(os.Stderr, nil, nil)

	// Concurrently add tokens
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			p.AddTokens(100)
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	if got := p.TotalTokens(); got != 1000 {
		t.Errorf("TotalTokens() = %d, want 1000", got)
	}
}

func TestC7Progress_UnknownMetricID(t *testing.T) {
	ids := []string{"m1"}
	p := NewC7Progress(os.Stderr, ids, nil)

	// These should not panic, just be no-ops
	p.SetMetricRunning("unknown", 5)
	p.SetMetricSample("unknown", 3)
	p.SetMetricComplete("unknown", 8)
	p.SetMetricFailed("unknown", "error")

	// Original metric should be unchanged
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.metrics["m1"].Status != StatusPending {
		t.Errorf("m1 status = %v, want Pending", p.metrics["m1"].Status)
	}
}

func TestShortMetricID(t *testing.T) {
	tests := []struct {
		id   string
		want string
	}{
		{"task_execution_consistency", "M1"},
		{"code_behavior_comprehension", "M2"},
		{"cross_file_navigation", "M3"},
		{"identifier_interpretability", "M4"},
		{"documentation_accuracy_detection", "M5"},
		{"unknown", "un"},
		{"x", "x"},
		{"", ""},
	}

	for _, tc := range tests {
		got := shortMetricID(tc.id)
		if got != tc.want {
			t.Errorf("shortMetricID(%q) = %q, want %q", tc.id, got, tc.want)
		}
	}
}

func TestFormatTokens(t *testing.T) {
	tests := []struct {
		n    int
		want string
	}{
		{0, "0"},
		{999, "999"},
		{1000, "1,000"},
		{12345, "12,345"},
		{123456, "123,456"},
		{999999, "999,999"},
		{1000000, "1,000,000"},
		{1234567, "1,234,567"},
	}

	for _, tc := range tests {
		got := formatTokens(tc.n)
		if got != tc.want {
			t.Errorf("formatTokens(%d) = %q, want %q", tc.n, got, tc.want)
		}
	}
}

func TestMetricStatusConstants(t *testing.T) {
	// Verify status constants are defined correctly
	statuses := []MetricStatus{
		StatusPending,
		StatusRunning,
		StatusComplete,
		StatusFailed,
	}

	expectedValues := []string{
		"pending",
		"running",
		"complete",
		"failed",
	}

	for i, status := range statuses {
		if string(status) != expectedValues[i] {
			t.Errorf("status %d expected %q, got %q", i, expectedValues[i], status)
		}
	}
}

func TestMetricProgress_InitialState(t *testing.T) {
	p := &MetricProgress{
		ID:   "test",
		Name: "Test Metric",
	}

	// Default status should be empty string (zero value)
	// When used with NewC7Progress, it's explicitly set to StatusPending
	if p.Status != "" {
		t.Errorf("initial Status = %q, want empty", p.Status)
	}
	if p.CurrentSample != 0 {
		t.Errorf("initial CurrentSample = %d, want 0", p.CurrentSample)
	}
	if p.TotalSamples != 0 {
		t.Errorf("initial TotalSamples = %d, want 0", p.TotalSamples)
	}
	if p.Score != 0 {
		t.Errorf("initial Score = %d, want 0", p.Score)
	}
}

func TestC7Progress_MetricOrder(t *testing.T) {
	ids := []string{"m3", "m1", "m2"}
	p := NewC7Progress(os.Stderr, ids, nil)

	// Order should be preserved
	for i, id := range p.metricOrder {
		if id != ids[i] {
			t.Errorf("metricOrder[%d] = %q, want %q", i, id, ids[i])
		}
	}
}
