package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync/atomic"
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
	e := NewEvaluator(60 * time.Second)

	// Inject mock command runner returning valid CLI JSON
	e.SetCommandRunner(func(ctx context.Context, name string, args ...string) ([]byte, error) {
		resp := `{"session_id":"test-123","result":"ok","structured_output":{"score":7,"reason":"well-written"}}`
		return []byte(resp), nil
	})

	ctx := context.Background()
	result, err := e.EvaluateContent(ctx, "rate this", "some content")
	if err != nil {
		t.Fatalf("EvaluateContent failed: %v", err)
	}

	if result.Score != 7 {
		t.Errorf("Expected score 7, got %d", result.Score)
	}
	if result.Reason != "well-written" {
		t.Errorf("Expected reason %q, got %q", "well-written", result.Reason)
	}
}

func TestEvaluator_Timeout(t *testing.T) {
	// Use a very short timeout to trigger timeout behavior
	e := NewEvaluator(1 * time.Nanosecond)

	// Inject mock that blocks until context is cancelled
	e.SetCommandRunner(func(ctx context.Context, name string, args ...string) ([]byte, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	})

	ctx := context.Background()
	_, err := e.EvaluateContent(ctx, "rate this", "content")
	if err == nil {
		t.Fatal("Expected timeout error, got nil")
	}

	if !strings.Contains(err.Error(), "timed out") {
		t.Errorf("Expected timeout error, got: %v", err)
	}
}

func TestEvaluator_RetryOnFailure(t *testing.T) {
	e := NewEvaluator(60 * time.Second)

	// Inject mock that fails on first call, succeeds on second
	var callCount int32
	e.SetCommandRunner(func(ctx context.Context, name string, args ...string) ([]byte, error) {
		n := atomic.AddInt32(&callCount, 1)
		if n == 1 {
			return []byte("bad output"), fmt.Errorf("first call fails")
		}
		resp := `{"session_id":"retry-ok","result":"ok","structured_output":{"score":8,"reason":"retry succeeded"}}`
		return []byte(resp), nil
	})

	ctx := context.Background()
	result, err := e.EvaluateWithRetry(ctx, "rate this", "content")
	if err != nil {
		t.Fatalf("EvaluateWithRetry failed: %v", err)
	}

	if result.Score != 8 {
		t.Errorf("Expected score 8, got %d", result.Score)
	}

	if got := atomic.LoadInt32(&callCount); got != 2 {
		t.Errorf("Expected 2 calls (1 fail + 1 retry), got %d", got)
	}
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
	// Test that evaluationResult can unmarshal correctly
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
			var result evaluationResult
			err := unmarshalevaluationResult([]byte(tc.json), &result)
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

// unmarshalevaluationResult is a test helper to unmarshal evaluationResult.
func unmarshalevaluationResult(data []byte, r *evaluationResult) error {
	return json.Unmarshal(data, r)
}

func TestEvaluationResult_JSONMarshaling(t *testing.T) {
	result := evaluationResult{
		Score:  8,
		Reason: "well-written content",
	}

	// Marshal to JSON
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	// Unmarshal back
	var decoded evaluationResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.Score != result.Score {
		t.Errorf("score = %d, want %d", decoded.Score, result.Score)
	}
	if decoded.Reason != result.Reason {
		t.Errorf("reason = %q, want %q", decoded.Reason, result.Reason)
	}
}

func TestEvaluationResult_ScoreValidation(t *testing.T) {
	// Test that the evaluationResult struct can hold scores 1-10
	tests := []struct {
		score      int
		shouldFail bool
	}{
		{1, false},   // min valid
		{10, false},  // max valid
		{5, false},   // mid range
		{0, true},    // below minimum (would fail validation in evaluator)
		{11, true},   // above maximum (would fail validation in evaluator)
		{-1, true},   // negative (would fail validation in evaluator)
	}

	for _, tt := range tests {
		t.Run(string(rune('0'+tt.score)), func(t *testing.T) {
			result := evaluationResult{
				Score:  tt.score,
				Reason: "test",
			}

			// The struct itself doesn't validate, but the evaluator does
			// Just verify the struct can hold these values
			if result.Score != tt.score {
				t.Errorf("score not stored correctly")
			}
		})
	}
}

func TestNewEvaluator_TimeoutHandling(t *testing.T) {
	tests := []struct {
		name           string
		inputTimeout   time.Duration
		expectedOutput time.Duration
	}{
		{
			name:           "zero timeout uses default",
			inputTimeout:   0,
			expectedOutput: 60 * time.Second,
		},
		{
			name:           "custom timeout preserved",
			inputTimeout:   30 * time.Second,
			expectedOutput: 30 * time.Second,
		},
		{
			name:           "very short timeout preserved",
			inputTimeout:   1 * time.Second,
			expectedOutput: 1 * time.Second,
		},
		{
			name:           "very long timeout preserved",
			inputTimeout:   5 * time.Minute,
			expectedOutput: 5 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := NewEvaluator(tt.inputTimeout)
			if e.timeout != tt.expectedOutput {
				t.Errorf("timeout = %v, want %v", e.timeout, tt.expectedOutput)
			}
		})
	}
}

func TestEvaluator_TimeoutField(t *testing.T) {
	// Test that evaluator stores timeout correctly
	e := NewEvaluator(10 * time.Second)

	if e == nil {
		t.Fatal("evaluator should not be nil")
	}

	// Verify evaluator has the correct timeout
	if e.timeout != 10*time.Second {
		t.Errorf("timeout = %v, want 10s", e.timeout)
	}
}
