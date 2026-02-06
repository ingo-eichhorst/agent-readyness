package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ingo/agent-readyness/internal/agent/metrics"
)

// DebugResponse represents a single captured C7 metric response for persistence and replay.
type DebugResponse struct {
	MetricID    string  `json:"metric_id"`
	SampleIndex int     `json:"sample_index"`
	FilePath    string  `json:"file_path"`
	Prompt      string  `json:"prompt"`
	Response    string  `json:"response"`
	Duration    float64 `json:"duration_seconds"`
	Error       string  `json:"error,omitempty"`
}

// SaveResponses persists metric results as individual JSON files in debugDir.
// Each sample is saved as {metric_id}_{sample_index}.json.
func SaveResponses(debugDir string, results []metrics.MetricResult) error {
	if err := os.MkdirAll(debugDir, 0755); err != nil {
		return fmt.Errorf("create debug dir: %w", err)
	}

	for _, mr := range results {
		for sampleIdx, sr := range mr.Samples {
			resp := DebugResponse{
				MetricID:    mr.MetricID,
				SampleIndex: sampleIdx,
				FilePath:    sr.Sample.FilePath,
				Prompt:      sr.Prompt,
				Response:    sr.Response,
				Duration:    sr.Duration.Seconds(),
				Error:       sr.Error,
			}

			filename := fmt.Sprintf("%s_%d.json", mr.MetricID, sampleIdx)
			path := filepath.Join(debugDir, filename)

			data, err := json.MarshalIndent(resp, "", "  ")
			if err != nil {
				return fmt.Errorf("marshal %s: %w", filename, err)
			}

			if err := os.WriteFile(path, data, 0644); err != nil {
				return fmt.Errorf("write %s: %w", filename, err)
			}
		}
	}

	return nil
}

// LoadResponses reads all JSON response files from debugDir and returns them keyed by
// "{metric_id}_{sample_index}".
func LoadResponses(debugDir string) (map[string]DebugResponse, error) {
	entries, err := os.ReadDir(debugDir)
	if err != nil {
		return nil, fmt.Errorf("read debug dir: %w", err)
	}

	responses := make(map[string]DebugResponse)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		path := filepath.Join(debugDir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", entry.Name(), err)
		}

		var resp DebugResponse
		if err := json.Unmarshal(data, &resp); err != nil {
			return nil, fmt.Errorf("unmarshal %s: %w", entry.Name(), err)
		}

		key := fmt.Sprintf("%s_%d", resp.MetricID, resp.SampleIndex)
		responses[key] = resp
	}

	return responses, nil
}

// ReplayExecutor replays previously captured responses instead of calling Claude CLI.
// It implements the metrics.Executor interface.
type ReplayExecutor struct {
	responses map[string]DebugResponse
	callIndex map[string]int // tracks per-metric call count for sample indexing
	mu        sync.Mutex
}

// NewReplayExecutor creates a ReplayExecutor from a map of captured responses.
func NewReplayExecutor(responses map[string]DebugResponse) *ReplayExecutor {
	return &ReplayExecutor{
		responses: responses,
		callIndex: make(map[string]int),
	}
}

// ExecutePrompt replays a captured response by identifying the metric from the prompt text.
// Implements metrics.Executor interface.
func (r *ReplayExecutor) ExecutePrompt(ctx context.Context, workDir, prompt, tools string, timeout time.Duration) (string, error) {
	r.mu.Lock()
	metricID := identifyMetricFromPrompt(prompt)
	idx := r.callIndex[metricID]
	r.callIndex[metricID]++
	r.mu.Unlock()

	key := fmt.Sprintf("%s_%d", metricID, idx)
	resp, ok := r.responses[key]
	if !ok {
		return "", fmt.Errorf("no replay data for %s", key)
	}

	if resp.Error != "" {
		return "", fmt.Errorf("replayed error: %s", resp.Error)
	}

	return resp.Response, nil
}

// identifyMetricFromPrompt detects which metric a prompt belongs to based on distinctive substrings.
func identifyMetricFromPrompt(prompt string) string {
	lower := strings.ToLower(prompt)

	switch {
	case strings.Contains(lower, "list all function names") || strings.Contains(lower, "list all exported function"):
		return "task_execution_consistency"
	case strings.Contains(lower, "explain what the code does"):
		return "code_behavior_comprehension"
	case strings.Contains(lower, "trace the dependencies") || strings.Contains(lower, "trace the complete dependency chain") || strings.Contains(lower, "trace its dependencies"):
		return "cross_file_navigation"
	case strings.Contains(lower, "interpret what the identifier") || strings.Contains(lower, "interpret what each identifier"):
		return "identifier_interpretability"
	case strings.Contains(lower, "review the documentation") || strings.Contains(lower, "identify any inaccuracies") || strings.Contains(lower, "documentation accuracy"):
		return "documentation_accuracy_detection"
	default:
		return "unknown"
	}
}

// Compile-time check that ReplayExecutor implements metrics.Executor.
var _ metrics.Executor = (*ReplayExecutor)(nil)
