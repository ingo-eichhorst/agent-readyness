package agent

import (
	"testing"
)

func TestGetRubric_AllTasksHaveRubrics(t *testing.T) {
	tasks := AllTasks()

	for _, task := range tasks {
		rubric := getRubric(task.ID)
		if rubric == "" {
			t.Errorf("task %q has empty rubric", task.ID)
		}
		// Verify rubric contains expected structure
		if len(rubric) < 100 {
			t.Errorf("task %q rubric is too short (%d chars)", task.ID, len(rubric))
		}
	}
}

func TestGetRubric_Fallback(t *testing.T) {
	// Unknown task ID should fallback to intent_clarity rubric
	rubric := getRubric("unknown_task")
	expected := getRubric("intent_clarity")
	if rubric != expected {
		t.Errorf("unknown task should fallback to intent_clarity rubric")
	}
}

func TestScoreResult_Fields(t *testing.T) {
	// Verify ScoreResult has expected fields
	result := ScoreResult{
		Score:     80,
		Reasoning: "Good response with accurate explanation",
	}

	if result.Score != 80 {
		t.Errorf("expected Score 80, got %d", result.Score)
	}
	if result.Reasoning == "" {
		t.Error("Reasoning should not be empty")
	}
}

func TestNewScorer(t *testing.T) {
	// Test NewScorer with nil client (should not panic)
	scorer := NewScorer(nil)
	if scorer == nil {
		t.Error("NewScorer returned nil")
	}
}

func TestGetRubric_ContainsJSONInstruction(t *testing.T) {
	// All rubrics should instruct to respond with JSON
	tasks := AllTasks()
	for _, task := range tasks {
		rubric := getRubric(task.ID)
		if !contains(rubric, "JSON") {
			t.Errorf("task %q rubric should mention JSON response format", task.ID)
		}
		if !contains(rubric, "score") {
			t.Errorf("task %q rubric should mention score", task.ID)
		}
	}
}

// contains checks if s contains substr (simple implementation)
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
