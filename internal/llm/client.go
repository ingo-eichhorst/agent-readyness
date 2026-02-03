// Package llm provides LLM client abstraction for content quality evaluation.
package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// Client wraps the Anthropic SDK for content evaluation.
type Client struct {
	client *anthropic.Client
	model  anthropic.Model
}

// NewClient creates an LLM client with the given API key.
// Returns error if API key is empty.
func NewClient(apiKey string) (*Client, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("ANTHROPIC_API_KEY not set")
	}
	c := anthropic.NewClient(option.WithAPIKey(apiKey))
	return &Client{
		client: &c,
		model:  anthropic.ModelClaudeHaiku4_5, // Cost-effective for evaluation
	}, nil
}

// Evaluation holds LLM evaluation result.
type Evaluation struct {
	Score     int    // 1-10
	Reasoning string // Brief explanation
}

// EvaluateContent runs LLM judge on content with given system prompt.
// The prompt should instruct the model to return JSON: {"score": N, "reason": "..."}
// Implements retry with exponential backoff on rate limits.
func (c *Client) EvaluateContent(ctx context.Context, systemPrompt, content string) (Evaluation, error) {
	var lastErr error
	maxRetries := 3
	backoff := time.Second

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return Evaluation{}, ctx.Err()
			case <-time.After(backoff):
				backoff *= 2 // exponential backoff
			}
		}

		eval, err := c.doEvaluate(ctx, systemPrompt, content)
		if err == nil {
			return eval, nil
		}

		lastErr = err
		// Check if error is retryable (rate limit)
		if !isRetryableError(err) {
			return Evaluation{}, err
		}
	}

	return Evaluation{}, fmt.Errorf("max retries exceeded: %w", lastErr)
}

// doEvaluate performs a single API call with prompt caching.
func (c *Client) doEvaluate(ctx context.Context, systemPrompt, content string) (Evaluation, error) {
	// Use prompt caching for system prompt (rubric) to reduce costs
	message, err := c.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     c.model,
		MaxTokens: 256, // Evaluation responses are small
		System: []anthropic.TextBlockParam{
			{
				Text:         systemPrompt,
				CacheControl: anthropic.NewCacheControlEphemeralParam(),
			},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(
				anthropic.NewTextBlock(content),
			),
		},
	})
	if err != nil {
		return Evaluation{}, fmt.Errorf("API call failed: %w", err)
	}

	// Extract text response
	if len(message.Content) == 0 {
		return Evaluation{}, fmt.Errorf("empty response from API")
	}

	var responseText string
	for _, block := range message.Content {
		if block.Type == "text" {
			responseText = block.Text
			break
		}
	}

	if responseText == "" {
		return Evaluation{}, fmt.Errorf("no text content in response")
	}

	// Parse JSON response
	var result struct {
		Score  int    `json:"score"`
		Reason string `json:"reason"`
	}

	if err := json.Unmarshal([]byte(responseText), &result); err != nil {
		return Evaluation{}, fmt.Errorf("invalid JSON response: %w (got: %s)", err, truncate(responseText, 100))
	}

	if result.Score < 1 || result.Score > 10 {
		return Evaluation{}, fmt.Errorf("score out of range (1-10): %d", result.Score)
	}

	return Evaluation{
		Score:     result.Score,
		Reasoning: result.Reason,
	}, nil
}

// isRetryableError checks if an error indicates a rate limit or transient failure.
func isRetryableError(err error) bool {
	// Anthropic SDK wraps API errors; check for 429 or 5xx status
	if err == nil {
		return false
	}
	errStr := err.Error()
	// Simple heuristic: rate limit or overloaded messages
	return contains(errStr, "429") ||
		contains(errStr, "rate") ||
		contains(errStr, "overloaded") ||
		contains(errStr, "503")
}

func contains(s, substr string) bool {
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
