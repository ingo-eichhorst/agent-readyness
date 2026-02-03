package llm

// CostEstimate holds estimated API cost for LLM analysis.
type CostEstimate struct {
	InputTokens  int     // Estimated input tokens
	OutputTokens int     // Estimated output tokens
	MinCost      float64 // Minimum expected cost (USD)
	MaxCost      float64 // Maximum expected cost (USD)
	FilesCount   int     // Number of files to analyze
}

// EstimateCost calculates expected cost before running analysis.
//
// Claude 3.5 Haiku pricing (as of 2024):
//   - Input: $0.25 per million tokens
//   - Output: $1.25 per million tokens
//   - Cache writes: 1.25x input price
//   - Cache reads: 0.1x input price (after first call)
//
// With prompt caching, the rubric (system prompt) is cached after the first call,
// significantly reducing costs for multi-file analysis.
func EstimateCost(readmeWordCount int, sampleFileCount int) CostEstimate {
	// System prompt (rubric) tokens - cached after first call
	// Each prompt is ~300-400 words, ~500 tokens; we have 4 prompts
	rubricTokens := 2000

	// Per-file input: file content + question
	// Average source file ~100 lines = ~300 tokens + overhead
	perFileInputTokens := 500

	// Per-file output: JSON with score and reason
	// {"score": N, "reason": "..."} ~50-100 tokens
	perFileOutputTokens := 100

	// README analysis (larger content)
	readmeTokens := readmeWordCount * 2 // ~2 tokens per word for markdown

	totalInput := rubricTokens + readmeTokens + (sampleFileCount * perFileInputTokens)
	totalOutput := (1 + sampleFileCount) * perFileOutputTokens // +1 for README

	// Cost calculation
	// First call: cache write (1.25x), subsequent: cache read (0.1x)
	// For multi-file analysis, most calls use cache reads
	// Simplified: assume 10% cache writes, 90% cache reads after first prompt
	inputCostPerMTok := 0.25
	outputCostPerMTok := 1.25

	// Effective input cost with caching benefit
	// First file per prompt type pays write, rest pay read
	// With 4 prompt types and N files, ~4 writes + (N-1)*4 reads
	cacheWriteCost := float64(4*rubricTokens/4) / 1_000_000 * inputCostPerMTok * 1.25
	cacheReadCost := float64(totalInput-rubricTokens) / 1_000_000 * inputCostPerMTok * 0.1
	outputCost := float64(totalOutput) / 1_000_000 * outputCostPerMTok

	minCost := cacheWriteCost + cacheReadCost + outputCost
	maxCost := minCost * 1.5 // Buffer for variance in content size

	return CostEstimate{
		InputTokens:  totalInput,
		OutputTokens: totalOutput,
		MinCost:      minCost,
		MaxCost:      maxCost,
		FilesCount:   sampleFileCount,
	}
}

// FormatCost returns a human-readable cost range string.
func (c CostEstimate) FormatCost() string {
	if c.MaxCost < 0.01 {
		return "< $0.01"
	}
	return "$" + formatFloat(c.MinCost) + " - $" + formatFloat(c.MaxCost)
}

func formatFloat(f float64) string {
	if f < 0.01 {
		return "0.00"
	}
	// Format to 3 decimal places
	whole := int(f)
	frac := int((f - float64(whole)) * 1000)
	if whole > 0 {
		return itoa(whole) + "." + padLeft(itoa(frac/10), 2, '0')
	}
	return "0." + padLeft(itoa(frac/10), 2, '0')
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

func padLeft(s string, n int, pad byte) string {
	if len(s) >= n {
		return s
	}
	result := make([]byte, n)
	for i := 0; i < n-len(s); i++ {
		result[i] = pad
	}
	copy(result[n-len(s):], s)
	return string(result)
}
