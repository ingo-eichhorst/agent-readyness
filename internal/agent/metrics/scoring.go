package metrics

import "strings"

// indicatorGroup defines a named group of string indicators.
// If ANY member matches, the group contributes its delta to the score.
type indicatorGroup struct {
	name    string
	members []string
}

// matchGroups checks each indicator group against the response and returns IndicatorMatches.
// Each group contributes +1 if any member matches.
func matchGroups(responseLower string, groups []indicatorGroup) []IndicatorMatch {
	indicators := make([]IndicatorMatch, 0, len(groups))
	for _, group := range groups {
		matched := false
		for _, member := range group.members {
			if strings.Contains(responseLower, member) {
				matched = true
				break
			}
		}
		delta := 0
		if matched {
			delta = 1
		}
		indicators = append(indicators, IndicatorMatch{
			Name: "group:" + group.name, Matched: matched, Delta: delta,
		})
	}
	return indicators
}

// matchNegativeIndicators checks each negative indicator individually and returns penalties.
// Each matched indicator contributes -1.
func matchNegativeIndicators(responseLower string, indicators []string) []IndicatorMatch {
	result := make([]IndicatorMatch, 0, len(indicators))
	for _, indicator := range indicators {
		matched := strings.Contains(responseLower, indicator)
		delta := 0
		if matched {
			delta = -1
		}
		result = append(result, IndicatorMatch{
			Name: "negative:" + indicator, Matched: matched, Delta: delta,
		})
	}
	return result
}

// computeScore calculates the final score from a ScoreTrace by summing base + all deltas,
// then clamping to [minScore, maxScore].
func computeScore(trace *ScoreTrace) int {
	score := trace.BaseScore
	for _, ind := range trace.Indicators {
		score += ind.Delta
	}
	if score < minScore {
		score = minScore
	}
	if score > maxScore {
		score = maxScore
	}
	trace.FinalScore = score
	return score
}
