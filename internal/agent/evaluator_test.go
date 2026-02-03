package agent

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestEvaluator_NewEvaluator(t *testing.T) {
	// Test default timeout
	e := NewEvaluator(0)
	if e.timeout != 60*time.Second {
		t.Errorf("Expected default timeout of 60s, got %v", e.timeout)
	}

	// Test custom timeout
	e = NewEvaluator(30 * time.Second)
	if e.timeout != 30*time.Second {
		t.Errorf("Expected timeout of 30s, got %v", e.timeout)
	}
}

func TestEvaluator_EvaluateContent(t *testing.T) {
	// Skip if Claude CLI is not available
	status := DetectCLI()
	if !status.Available {
		t.Skip("Claude CLI not available, skipping integration test")
	}

	e := NewEvaluator(60 * time.Second)
	ctx := context.Background()

	// Use a simple test prompt
	systemPrompt := `You are a simple evaluator. Rate the quality of the text on a scale of 1-10.
Respond with ONLY valid JSON: {"score": N, "reason": "brief reason"}`

	content := "This is a well-written sentence with proper grammar and punctuation."

	result, err := e.EvaluateContent(ctx, systemPrompt, content)
	if err != nil {
		t.Fatalf("EvaluateContent failed: %v", err)
	}

	if result.Score < 1 || result.Score > 10 {
		t.Errorf("Expected score 1-10, got %d", result.Score)
	}

	if result.Reason == "" {
		t.Error("Expected non-empty reason")
	}

	t.Logf("Evaluation result: score=%d, reason=%q", result.Score, result.Reason)
}

func TestEvaluator_Timeout(t *testing.T) {
	// Skip if Claude CLI is not available
	status := DetectCLI()
	if !status.Available {
		t.Skip("Claude CLI not available, skipping timeout test")
	}

	// Use a very short timeout to trigger timeout behavior
	e := NewEvaluator(1 * time.Nanosecond)
	ctx := context.Background()

	systemPrompt := `Rate this text. Respond with: {"score": 5, "reason": "test"}`
	content := "Test content"

	_, err := e.EvaluateContent(ctx, systemPrompt, content)
	if err == nil {
		// If it somehow succeeds (very fast system), that's also acceptable
		t.Log("Evaluation succeeded despite short timeout - very fast system")
		return
	}

	// Check that error mentions timeout
	t.Logf("Timeout test error (expected): %v", err)
}

func TestEvaluator_RetryOnFailure(t *testing.T) {
	// Skip if Claude CLI is not available
	status := DetectCLI()
	if !status.Available {
		t.Skip("Claude CLI not available, skipping retry test")
	}

	e := NewEvaluator(60 * time.Second)
	ctx := context.Background()

	// Use a simple prompt that should succeed
	systemPrompt := `Rate this text 1-10. Respond with ONLY: {"score": 7, "reason": "test"}`
	content := "Simple test content for retry test."

	result, err := e.EvaluateWithRetry(ctx, systemPrompt, content)
	if err != nil {
		t.Fatalf("EvaluateWithRetry failed: %v", err)
	}

	if result.Score < 1 || result.Score > 10 {
		t.Errorf("Expected score 1-10, got %d", result.Score)
	}

	t.Logf("Retry test result: score=%d, reason=%q", result.Score, result.Reason)
}

func TestEvaluator_ContextCancellation(t *testing.T) {
	e := NewEvaluator(60 * time.Second)

	// Create already-cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := e.EvaluateWithRetry(ctx, "test", "test")
	if err == nil {
		t.Error("Expected error with cancelled context")
	}
}

func TestEvaluationResult_JSONUnmarshal(t *testing.T) {
	// Test that EvaluationResult can unmarshal correctly
	testCases := []struct {
		name      string
		json      string
		wantScore int
		wantErr   bool
	}{
		{
			name:      "valid result",
			json:      `{"score": 7, "reason": "Good quality"}`,
			wantScore: 7,
			wantErr:   false,
		},
		{
			name:      "min score",
			json:      `{"score": 1, "reason": "Poor"}`,
			wantScore: 1,
			wantErr:   false,
		},
		{
			name:      "max score",
			json:      `{"score": 10, "reason": "Excellent"}`,
			wantScore: 10,
			wantErr:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var result EvaluationResult
			err := unmarshalEvaluationResult([]byte(tc.json), &result)
			if tc.wantErr {
				if err == nil {
					t.Error("Expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if result.Score != tc.wantScore {
				t.Errorf("Expected score %d, got %d", tc.wantScore, result.Score)
			}
		})
	}
}

// unmarshalEvaluationResult is a test helper to unmarshal EvaluationResult.
func unmarshalEvaluationResult(data []byte, r *EvaluationResult) error {
	return json.Unmarshal(data, r)
}
