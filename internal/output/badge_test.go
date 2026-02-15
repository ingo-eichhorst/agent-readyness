package output

import (
	"strings"
	"testing"

	"github.com/ingo-eichhorst/agent-readyness/pkg/types"
)

func TestGenerateBadge(t *testing.T) {
	tests := []struct {
		name          string
		scored        *types.ScoredResult
		wantColor     string
		wantTierInURL string
		wantScoreURL  string
		wantRepoLink  bool
	}{
		{
			name: "Agent-Ready high score",
			scored: &types.ScoredResult{
				Tier:      "Agent-Ready",
				Composite: 8.2,
			},
			wantColor:     "green",
			wantTierInURL: "Agent--Ready",
			wantScoreURL:  "8.2%2F10",
			wantRepoLink:  true,
		},
		{
			name: "Agent-Assisted mid score",
			scored: &types.ScoredResult{
				Tier:      "Agent-Assisted",
				Composite: 6.5,
			},
			wantColor:     "yellow",
			wantTierInURL: "Agent--Assisted",
			wantScoreURL:  "6.5%2F10",
			wantRepoLink:  true,
		},
		{
			name: "Agent-Limited low score",
			scored: &types.ScoredResult{
				Tier:      "Agent-Limited",
				Composite: 4.5,
			},
			wantColor:     "orange",
			wantTierInURL: "Agent--Limited",
			wantScoreURL:  "4.5%2F10",
			wantRepoLink:  true,
		},
		{
			name: "Agent-Hostile very low score",
			scored: &types.ScoredResult{
				Tier:      "Agent-Hostile",
				Composite: 3.0,
			},
			wantColor:     "red",
			wantTierInURL: "Agent--Hostile",
			wantScoreURL:  "3.0%2F10",
			wantRepoLink:  true,
		},
		{
			name:   "nil scored returns empty",
			scored: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			badge := generateBadge(tt.scored)

			if tt.scored == nil {
				if badge.URL != "" || badge.Markdown != "" {
					t.Errorf("expected empty badgeInfo for nil scored, got URL=%q Markdown=%q", badge.URL, badge.Markdown)
				}
				return
			}

			// Check URL contains shields.io
			if !strings.Contains(badge.URL, "img.shields.io/badge/ARS-") {
				t.Errorf("URL should contain shields.io badge path, got %q", badge.URL)
			}

			// Check tier is properly encoded with double dashes
			if !strings.Contains(badge.URL, tt.wantTierInURL) {
				t.Errorf("URL should contain tier with double dashes %q, got %q", tt.wantTierInURL, badge.URL)
			}

			// Check score is properly encoded
			if !strings.Contains(badge.URL, tt.wantScoreURL) {
				t.Errorf("URL should contain encoded score %q, got %q", tt.wantScoreURL, badge.URL)
			}

			// Check color
			if !strings.HasSuffix(badge.URL, "-"+tt.wantColor) {
				t.Errorf("URL should end with color -%s, got %q", tt.wantColor, badge.URL)
			}

			// Check space encoding (%20)
			if !strings.Contains(badge.URL, "%20") {
				t.Errorf("URL should contain %%20 for space encoding, got %q", badge.URL)
			}

			// Check markdown wraps URL with link to repo
			if tt.wantRepoLink {
				if !strings.Contains(badge.Markdown, badge.URL) {
					t.Errorf("Markdown should contain badge URL, got %q", badge.Markdown)
				}
				if !strings.Contains(badge.Markdown, arsRepoURL) {
					t.Errorf("Markdown should contain repo URL %q, got %q", arsRepoURL, badge.Markdown)
				}
				if !strings.HasPrefix(badge.Markdown, "[![ARS](") {
					t.Errorf("Markdown should start with [![ARS](, got %q", badge.Markdown)
				}
			}
		})
	}
}

func TestEncodeBadgeText(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "Agent-Ready with score",
			input: "Agent-Ready 8.2/10",
			want:  "Agent--Ready%208.2%2F10",
		},
		{
			name:  "simple text with space",
			input: "hello world",
			want:  "hello%20world",
		},
		{
			name:  "text with dash",
			input: "foo-bar",
			want:  "foo--bar",
		},
		{
			name:  "text with slash",
			input: "8.0/10",
			want:  "8.0%2F10",
		},
		{
			name:  "multiple dashes",
			input: "a-b-c",
			want:  "a--b--c",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := encodeBadgeText(tt.input)
			if got != tt.want {
				t.Errorf("encodeBadgeText(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestTierToColor(t *testing.T) {
	tests := []struct {
		tier  string
		color string
	}{
		{"Agent-Ready", "green"},
		{"Agent-Assisted", "yellow"},
		{"Agent-Limited", "orange"},
		{"Agent-Hostile", "red"},
		{"Unknown-Tier", "red"},
		{"", "red"},
	}

	for _, tt := range tests {
		t.Run(tt.tier, func(t *testing.T) {
			got := tierToColor(tt.tier)
			if got != tt.color {
				t.Errorf("tierToColor(%q) = %q, want %q", tt.tier, got, tt.color)
			}
		})
	}
}
