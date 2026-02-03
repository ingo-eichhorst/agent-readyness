package output

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/ingo/agent-readyness/pkg/types"
)

// arsRepoURL is the URL to the ARS project repository.
const arsRepoURL = "https://github.com/ingo-eichhorst/agent-readyness"

// BadgeInfo contains the generated badge URL and markdown.
type BadgeInfo struct {
	URL      string // Raw shields.io badge URL
	Markdown string // Complete markdown with link to repo
}

// GenerateBadge creates a shields.io badge URL and markdown from a scored result.
// Returns empty BadgeInfo if scored is nil.
func GenerateBadge(scored *types.ScoredResult) BadgeInfo {
	if scored == nil {
		return BadgeInfo{}
	}

	// Format message: "{Tier} {Composite:.1f}/10"
	message := fmt.Sprintf("%s %.1f/10", scored.Tier, scored.Composite)

	// Encode for shields.io URL
	encodedMessage := encodeBadgeText(message)

	// Map tier to color
	color := tierToColor(scored.Tier)

	// Build URL: https://img.shields.io/badge/ARS-{encoded_message}-{color}
	badgeURL := fmt.Sprintf("https://img.shields.io/badge/ARS-%s-%s", encodedMessage, color)

	// Build markdown: [![ARS]({url})]({repo_url})
	markdown := fmt.Sprintf("[![ARS](%s)](%s)", badgeURL, arsRepoURL)

	return BadgeInfo{
		URL:      badgeURL,
		Markdown: markdown,
	}
}

// encodeBadgeText encodes text for use in a shields.io badge URL.
// Dashes must be escaped as double-dashes before URL encoding.
func encodeBadgeText(s string) string {
	// First, replace single dashes with double dashes (shields.io separator escape)
	escaped := strings.ReplaceAll(s, "-", "--")

	// Then URL path encode (spaces become %20, slashes become %2F, etc.)
	return url.PathEscape(escaped)
}

// tierToColor maps a tier classification to a shields.io color name.
func tierToColor(tier string) string {
	switch tier {
	case "Agent-Ready":
		return "green"
	case "Agent-Assisted":
		return "yellow"
	case "Agent-Limited":
		return "orange"
	case "Agent-Hostile":
		return "red"
	default:
		return "red" // Default to red for unknown tiers
	}
}
