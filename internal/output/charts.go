package output

import (
	"github.com/ingo/agent-readyness/pkg/types"
	charts "github.com/vicanso/go-charts/v2"
)

// Chart layout constants.
const (
	radarChartWidth   = 450
	radarChartHeight  = 400
	radarChartPad     = 20
	trendChartWidth   = 500
	trendChartHeight  = 300
	trendChartPadTop  = 40
	trendChartPadSide = 20
	trendChartPadLeft = 40
	maxCategoryScore  = 10.0 // Maximum score per category (radar chart axis)
	minRadarCategories = 3   // Minimum categories for radar chart rendering
)

// generateRadarChart creates an SVG radar chart for category scores.
// Returns the SVG string and any error.
// Requires at least 3 categories for radar chart (go-charts library requirement).
func generateRadarChart(categories []types.CategoryScore) (string, error) {
	if len(categories) < minRadarCategories {
		return "", nil
	}

	// Extract names and scores
	var names []string
	var maxValues []float64
	scores := make([]float64, len(categories))

	for i, cat := range categories {
		names = append(names, cat.Name)
		maxValues = append(maxValues, maxCategoryScore)
		scores[i] = cat.Score
	}

	values := [][]float64{scores}

	p, err := charts.RadarRender(
		values,
		charts.SVGTypeOption(),
		charts.TitleOptionFunc(charts.TitleOption{
			Text: "Agent Readiness Score",
			Left: "center",
		}),
		charts.RadarIndicatorOptionFunc(names, maxValues),
		charts.ThemeOptionFunc("light"),
		charts.WidthOptionFunc(radarChartWidth),
		charts.HeightOptionFunc(radarChartHeight),
		charts.PaddingOptionFunc(charts.Box{Top: radarChartPad, Right: radarChartPad, Bottom: radarChartPad, Left: radarChartPad}),
	)
	if err != nil {
		return "", err
	}

	buf, err := p.Bytes()
	if err != nil {
		return "", err
	}
	return string(buf), nil
}

// generateTrendChart creates an SVG line chart comparing current vs baseline scores.
// Returns empty string if baseline is nil.
func generateTrendChart(current, baseline *types.ScoredResult) (string, error) {
	if baseline == nil || len(current.Categories) == 0 {
		return "", nil
	}

	// Build baseline score map for lookup
	baselineScores := make(map[string]float64)
	for _, cat := range baseline.Categories {
		baselineScores[cat.Name] = cat.Score
	}

	// Extract category names and both score series
	var names []string
	var baselineSeries, currentSeries []float64

	for _, cat := range current.Categories {
		names = append(names, cat.Name)
		currentSeries = append(currentSeries, cat.Score)
		if bs, ok := baselineScores[cat.Name]; ok {
			baselineSeries = append(baselineSeries, bs)
		} else {
			baselineSeries = append(baselineSeries, 0)
		}
	}

	// Create line chart with two series
	values := [][]float64{baselineSeries, currentSeries}

	p, err := charts.LineRender(
		values,
		charts.SVGTypeOption(),
		charts.TitleTextOptionFunc("Score Comparison"),
		charts.XAxisDataOptionFunc(names),
		charts.LegendLabelsOptionFunc([]string{"Previous", "Current"}),
		charts.ThemeOptionFunc("light"),
		charts.WidthOptionFunc(trendChartWidth),
		charts.HeightOptionFunc(trendChartHeight),
		charts.PaddingOptionFunc(charts.Box{Top: trendChartPadTop, Right: trendChartPadSide, Bottom: trendChartPadSide, Left: trendChartPadLeft}),
	)
	if err != nil {
		return "", err
	}

	buf, err := p.Bytes()
	if err != nil {
		return "", err
	}
	return string(buf), nil
}
