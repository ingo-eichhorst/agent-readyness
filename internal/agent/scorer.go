package agent

import (
	"context"
	"fmt"
)

// scoreScaleToPercent converts a 1-10 score to a 0-100 scale.
const scoreScaleToPercent = 10

// scorer evaluates agent task responses using LLM-as-a-judge via CLI.
type scorer struct {
	evaluator *Evaluator
}

// newScorer creates a scorer with the given CLI evaluator.
func newScorer(evaluator *Evaluator) *scorer {
	return &scorer{evaluator: evaluator}
}

// scoreResult holds the scoring output for a task response.
type scoreResult struct {
	Score     int    // 0-100
	Reasoning string // explanation from LLM judge
}

// Score evaluates an agent's response against a task's rubric.
func (s *scorer) Score(ctx context.Context, t task, response string) (scoreResult, error) {
	// Build the scoring prompt based on task ID
	rubric := getRubric(t.ID)
	content := fmt.Sprintf("Task: %s\n\nAgent Response:\n%s", t.Prompt, response)

	eval, err := s.evaluator.EvaluateWithRetry(ctx, rubric, content)
	if err != nil {
		return scoreResult{}, err
	}

	// Scale from 1-10 to 0-100
	return scoreResult{
		Score:     eval.Score * scoreScaleToPercent,
		Reasoning: eval.Reason, // evaluationResult uses Reason, not Reasoning
	}, nil
}

// getRubric returns the LLM scoring prompt for a task.
func getRubric(taskID string) string {
	rubrics := map[string]string{
		"intent_clarity": `You are evaluating an AI coding agent's response to a code understanding task.

The agent was asked to find and explain a main entry point function.

Score the response from 1-10 based on these criteria:
- Correct identification (40%): Did the agent find the right function and file?
- Accuracy of explanation (40%): Is the explanation correct, clear, and specific?
- Use of codebase context (20%): Did the agent reference actual code details?

Consider:
- Score 8-10: Correct function found, accurate explanation, references specific code
- Score 5-7: Function found but explanation has minor issues or lacks specifics
- Score 3-4: Wrong function or significant explanation errors
- Score 1-2: Failed to find function or completely wrong explanation

Respond with JSON only: {"score": N, "reason": "brief explanation"}`,

		"modification_confidence": `You are evaluating an AI coding agent's response to a code modification task.

The agent was asked to propose input validation for a function.

Score the response from 1-10 based on these criteria:
- Correctness of change (50%): Is the proposed validation appropriate and correct?
- Appropriate scope (30%): Is the change well-scoped (not too broad or too narrow)?
- Follows patterns (20%): Does it match existing codebase patterns?

Consider:
- Score 8-10: Correct validation, well-scoped, matches existing patterns
- Score 5-7: Reasonable validation but minor issues with scope or patterns
- Score 3-4: Validation has significant issues or wrong approach
- Score 1-2: Proposed change would break code or is completely wrong

Respond with JSON only: {"score": N, "reason": "brief explanation"}`,

		"cross_file_coherence": `You are evaluating an AI coding agent's response to a code tracing task.

The agent was asked to trace data flow across multiple files.

Score the response from 1-10 based on these criteria:
- Completeness of trace (50%): Did the agent follow the full data path?
- Accuracy (30%): Are the files, functions, and flow described correctly?
- Efficiency (20%): Did the agent avoid unnecessary detours or confusion?

Consider:
- Score 8-10: Complete trace, all files/functions correct, clear presentation
- Score 5-7: Most of the trace correct but missing steps or minor errors
- Score 3-4: Major gaps in trace or significant errors
- Score 1-2: Failed to trace or completely wrong flow

Respond with JSON only: {"score": N, "reason": "brief explanation"}`,

		"semantic_completeness": `You are evaluating an AI coding agent's response to a pattern-matching task.

The agent was asked to propose error handling that matches existing patterns.

Score the response from 1-10 based on these criteria:
- Functional correctness (40%): Would the proposed error handling work?
- Pattern matching (40%): Does it actually match patterns found in the codebase?
- Edge case handling (20%): Does it consider edge cases appropriately?

Consider:
- Score 8-10: Correct error handling, clearly matches existing patterns, good edge cases
- Score 5-7: Reasonable error handling but pattern matching could be better
- Score 3-4: Error handling has issues or doesn't match patterns
- Score 1-2: Proposed handling would fail or ignores existing patterns

Respond with JSON only: {"score": N, "reason": "brief explanation"}`,
	}

	if rubric, ok := rubrics[taskID]; ok {
		return rubric
	}
	return rubrics["intent_clarity"] // fallback
}
