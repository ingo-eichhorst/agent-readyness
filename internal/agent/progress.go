package agent

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/mattn/go-isatty"
)

// metricStatus represents the current state of a single metric evaluation.
type metricStatus string

const (
	statusPending  metricStatus = "pending"
	statusRunning  metricStatus = "running"
	statusComplete metricStatus = "complete"
	statusFailed   metricStatus = "failed"
)

// Progress display constants.
const (
	progressTickInterval = 200 * time.Millisecond // Refresh rate for progress display
	percentMultiplier    = 100                     // Multiplier for percentage calculation
	costRatePerMTok      = 5.0                     // Sonnet 4.5 blended rate $/MTok
	tokensPerMillion     = 1_000_000               // Token-to-millions conversion
	tokensPerThousand    = 1000                    // Token-to-thousands conversion
	progressLineWidth    = 130                     // Terminal line width for padding
)

// metricProgress tracks the progress of a single metric.
type metricProgress struct {
	ID            string
	Name          string
	Status        metricStatus
	CurrentSample int
	TotalSamples  int
	Score         int // Final score when complete (1-10)
	Error         string
}

// c7Progress displays real-time progress for C7 agent evaluation.
// Thread-safe for concurrent metric updates.
type c7Progress struct {
	mu          sync.Mutex
	metrics     map[string]*metricProgress
	metricOrder []string // Preserve display order
	totalTokens int
	startTime   time.Time
	isTTY       bool
	writer      *os.File
	ticker      *time.Ticker
	done        chan struct{}
	active      bool
}

// NewC7Progress creates a new progress display.
// If writer is not a TTY, display operations are no-ops.
func NewC7Progress(w *os.File, metricIDs []string, metricNames []string) *c7Progress {
	metrics := make(map[string]*metricProgress, len(metricIDs))
	for i, id := range metricIDs {
		name := id
		if i < len(metricNames) {
			name = metricNames[i]
		}
		metrics[id] = &metricProgress{
			ID:     id,
			Name:   name,
			Status: statusPending,
		}
	}

	return &c7Progress{
		metrics:     metrics,
		metricOrder: metricIDs,
		isTTY:       isatty.IsTerminal(w.Fd()) || isatty.IsCygwinTerminal(w.Fd()),
		writer:      w,
		done:        make(chan struct{}),
	}
}

// Start begins the progress display refresh loop.
func (p *c7Progress) Start() {
	if !p.isTTY {
		return
	}

	p.mu.Lock()
	p.active = true
	p.startTime = time.Now()
	p.mu.Unlock()

	p.ticker = time.NewTicker(progressTickInterval)
	go func() {
		for {
			select {
			case <-p.done:
				return
			case <-p.ticker.C:
				p.render()
			}
		}
	}()
}

// SetMetricRunning marks a metric as running and sets total samples.
func (p *c7Progress) SetMetricRunning(id string, totalSamples int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if m, ok := p.metrics[id]; ok {
		m.Status = statusRunning
		m.TotalSamples = totalSamples
		m.CurrentSample = 0
	}
}

// SetMetricSample updates the current sample number for a running metric.
func (p *c7Progress) SetMetricSample(id string, current int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if m, ok := p.metrics[id]; ok {
		m.CurrentSample = current
	}
}

// SetMetricComplete marks a metric as complete with its final score.
func (p *c7Progress) SetMetricComplete(id string, score int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if m, ok := p.metrics[id]; ok {
		m.Status = statusComplete
		m.Score = score
	}
}

// SetMetricFailed marks a metric as failed with an error message.
func (p *c7Progress) SetMetricFailed(id string, err string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if m, ok := p.metrics[id]; ok {
		m.Status = statusFailed
		m.Error = err
	}
}

// AddTokens adds to the running token count.
func (p *c7Progress) AddTokens(tokens int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.totalTokens += tokens
}

// TotalTokens returns the current token count.
func (p *c7Progress) TotalTokens() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.totalTokens
}

// render draws the current progress state.
// Format includes "progress" text for CLI visibility requirement (C7-IMPL-06).
// Example: "C7 progress [15s]: M1: 60% (3/5) | M2: Done(8) | M3: Pending | Tokens: 12,345 | Est. $0.15"
func (p *c7Progress) render() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.active {
		return
	}

	var parts []string
	for _, id := range p.metricOrder {
		m := p.metrics[id]
		// Use short ID for display (e.g., "M1" from "task_execution_consistency")
		shortID := shortMetricID(id)
		switch m.Status {
		case statusPending:
			parts = append(parts, fmt.Sprintf("%s: Pending", shortID))
		case statusRunning:
			// Show percentage for progress visibility (C7-IMPL-06)
			pct := 0
			if m.TotalSamples > 0 {
				pct = (m.CurrentSample * percentMultiplier) / m.TotalSamples
			}
			parts = append(parts, fmt.Sprintf("%s: %d%% (%d/%d)", shortID, pct, m.CurrentSample, m.TotalSamples))
		case statusComplete:
			parts = append(parts, fmt.Sprintf("%s: Done(%d)", shortID, m.Score))
		case statusFailed:
			parts = append(parts, fmt.Sprintf("%s: Failed", shortID))
		}
	}

	// Token count with comma formatting
	tokenStr := formatTokens(p.totalTokens)

	// Cost estimation: Sonnet 4.5 blended rate ~$5/MTok
	costUSD := float64(p.totalTokens) / tokensPerMillion * costRatePerMTok

	// Build and write line - includes "progress" text for C7-IMPL-06 compliance
	elapsed := time.Since(p.startTime).Round(time.Second)
	line := fmt.Sprintf("\rC7 progress [%s]: ", elapsed)
	for i, part := range parts {
		if i > 0 {
			line += " | "
		}
		line += part
	}
	line += fmt.Sprintf(" | Tokens: %s | Est. $%.2f", tokenStr, costUSD)

	// Pad with spaces to clear previous longer lines, then write
	fmt.Fprintf(p.writer, "%-*s", progressLineWidth, line)
}

// Stop halts the progress display and prints a final summary.
func (p *c7Progress) Stop() {
	if !p.isTTY {
		return
	}

	p.mu.Lock()
	if !p.active {
		p.mu.Unlock()
		return
	}
	p.active = false
	p.mu.Unlock()

	if p.ticker != nil {
		p.ticker.Stop()
	}
	close(p.done)

	// Clear line and print final summary
	fmt.Fprintf(p.writer, "\r\033[K") // Clear line
	p.printSummary()
}

// printSummary outputs a final summary of all metric results.
func (p *c7Progress) printSummary() {
	p.mu.Lock()
	defer p.mu.Unlock()

	elapsed := time.Since(p.startTime).Round(time.Second)
	tokenStr := formatTokens(p.totalTokens)
	costUSD := float64(p.totalTokens) / tokensPerMillion * costRatePerMTok

	fmt.Fprintf(p.writer, "C7 Evaluation complete in %s | Tokens: %s | Cost: $%.2f\n", elapsed, tokenStr, costUSD)
}

// shortMetricID returns a short display ID (M1-M5) for a metric ID.
func shortMetricID(id string) string {
	switch id {
	case "task_execution_consistency":
		return "M1"
	case "code_behavior_comprehension":
		return "M2"
	case "cross_file_navigation":
		return "M3"
	case "identifier_interpretability":
		return "M4"
	case "documentation_accuracy_detection":
		return "M5"
	default:
		if len(id) >= 2 {
			return id[:2]
		}
		return id
	}
}

// formatTokens formats a token count with comma separators.
func formatTokens(n int) string {
	if n < tokensPerThousand {
		return fmt.Sprintf("%d", n)
	}
	if n < tokensPerMillion {
		return fmt.Sprintf("%d,%03d", n/tokensPerThousand, n%tokensPerThousand)
	}
	return fmt.Sprintf("%d,%03d,%03d", n/tokensPerMillion, (n/tokensPerThousand)%tokensPerThousand, n%tokensPerThousand)
}
