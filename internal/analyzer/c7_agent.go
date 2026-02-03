package analyzer

import (
	"context"
	"fmt"
	"time"

	"github.com/ingo/agent-readyness/internal/agent"
	"github.com/ingo/agent-readyness/internal/llm"
	"github.com/ingo/agent-readyness/pkg/types"
)

// C7Analyzer implements the pipeline.Analyzer interface for C7: Agent Evaluation.
type C7Analyzer struct {
	llmClient *llm.Client
	enabled   bool // only runs if explicitly enabled
}

// NewC7Analyzer creates a C7Analyzer. It's disabled by default.
func NewC7Analyzer() *C7Analyzer {
	return &C7Analyzer{enabled: false}
}

// Enable activates C7 analysis with the given LLM client.
func (a *C7Analyzer) Enable(client *llm.Client) {
	a.llmClient = client
	a.enabled = true
}

// Name returns the analyzer display name.
func (a *C7Analyzer) Name() string {
	return "C7: Agent Evaluation"
}

// Analyze runs C7 agent evaluation.
func (a *C7Analyzer) Analyze(targets []*types.AnalysisTarget) (*types.AnalysisResult, error) {
	if !a.enabled {
		return &types.AnalysisResult{
			Name:     "C7: Agent Evaluation",
			Category: "C7",
			Metrics:  map[string]interface{}{"c7": &types.C7Metrics{Available: false}},
		}, nil
	}

	if len(targets) == 0 {
		return nil, fmt.Errorf("no targets provided")
	}
	rootDir := targets[0].RootDir

	// Check for Claude CLI
	if err := agent.CheckClaudeCLI(); err != nil {
		return &types.AnalysisResult{
			Name:     "C7: Agent Evaluation",
			Category: "C7",
			Metrics:  map[string]interface{}{"c7": &types.C7Metrics{Available: false}},
		}, nil
	}

	// Create isolated workspace
	workDir, cleanup, err := agent.CreateWorkspace(rootDir)
	if err != nil {
		return nil, fmt.Errorf("create workspace: %w", err)
	}
	defer cleanup()

	// Execute tasks
	executor := agent.NewExecutor(workDir)
	scorer := agent.NewScorer(a.llmClient)
	tasks := agent.AllTasks()

	metrics := &types.C7Metrics{
		Available:   true,
		TaskResults: make([]types.C7TaskResult, 0, len(tasks)),
	}

	ctx := context.Background()
	startTime := time.Now()
	totalTokens := 0

	for _, task := range tasks {
		taskStart := time.Now()
		result := executor.ExecuteTask(ctx, task)
		taskDuration := time.Since(taskStart).Seconds()

		var scoreResult agent.ScoreResult
		if result.Status == agent.StatusCompleted && result.Response != "" {
			scoreResult, _ = scorer.Score(ctx, task, result.Response)
			totalTokens += estimateResponseTokens(result.Response)
		}

		taskResult := types.C7TaskResult{
			TaskID:    task.ID,
			TaskName:  task.Name,
			Score:     scoreResult.Score,
			Status:    string(result.Status),
			Duration:  taskDuration,
			Reasoning: scoreResult.Reasoning,
		}
		metrics.TaskResults = append(metrics.TaskResults, taskResult)

		// Set individual scores
		switch task.ID {
		case "intent_clarity":
			metrics.IntentClarity = scoreResult.Score
		case "modification_confidence":
			metrics.ModificationConfidence = scoreResult.Score
		case "cross_file_coherence":
			metrics.CrossFileCoherence = scoreResult.Score
		case "semantic_completeness":
			metrics.SemanticCompleteness = scoreResult.Score
		}
	}

	metrics.TotalDuration = time.Since(startTime).Seconds()
	metrics.TokensUsed = totalTokens

	// Calculate overall score (average of completed tasks)
	completedCount := 0
	totalScore := 0
	for _, tr := range metrics.TaskResults {
		if tr.Status == string(agent.StatusCompleted) && tr.Score > 0 {
			totalScore += tr.Score
			completedCount++
		}
	}
	if completedCount > 0 {
		metrics.OverallScore = float64(totalScore) / float64(completedCount)
	}

	// Estimate cost (Sonnet pricing for Claude CLI)
	// ~10k tokens per task (agent execution), ~500 tokens per scoring call
	metrics.CostUSD = float64(metrics.TokensUsed+len(tasks)*500) / 1_000_000 * 5.0 // ~$5/MTok blended

	return &types.AnalysisResult{
		Name:     "C7: Agent Evaluation",
		Category: "C7",
		Metrics:  map[string]interface{}{"c7": metrics},
	}, nil
}

func estimateResponseTokens(response string) int {
	return len(response) / 4 // ~4 chars per token
}
