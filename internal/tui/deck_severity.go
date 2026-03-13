package tui

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/tinytelemetry/tiny-telemetry/internal/model"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// SeverityDeck displays a severity timeline chart with stacked colored bars,
// labeled axes, and a legend.
type SeverityDeck struct {
	pushModalCmd tea.Cmd
	data         []model.MinuteCounts
}

// NewSeverityDeck creates a new severity timeline deck.
func NewSeverityDeck(pushModalCmd tea.Cmd) *SeverityDeck {
	return &SeverityDeck{
		pushModalCmd: pushModalCmd,
		data:         make([]model.MinuteCounts, 0),
	}
}

func (p *SeverityDeck) ID() string         { return "severity" }
func (p *SeverityDeck) Title() string      { return "Severity Timeline" }
func (p *SeverityDeck) QuarterSized() bool { return true }

func (p *SeverityDeck) Refresh(_ model.LogQuerier, _ model.QueryOpts) {}

func (p *SeverityDeck) TypeID() string                { return "severity" }
func (p *SeverityDeck) DefaultInterval() time.Duration { return 2 * time.Second }

func (p *SeverityDeck) FetchCmd(store model.LogQuerier, opts model.QueryOpts) tea.Cmd {
	return func() tea.Msg {
		rows, err := store.SeverityCountsByMinute(opts)
		return DeckDataMsg{DeckTypeID: "severity", Data: rows, Err: err}
	}
}

func (p *SeverityDeck) ApplyData(data any, err error) {
	if err != nil {
		return
	}
	if rows, ok := data.([]model.MinuteCounts); ok {
		p.data = append([]model.MinuteCounts(nil), rows...)
	}
}

func (p *SeverityDeck) ContentLines(ctx ViewContext) int {
	if len(p.data) == 0 {
		return 1
	}
	deckHeight := 8
	if ctx.ContentWidth < 80 {
		deckHeight = 6
	}
	return deckHeight
}

func (p *SeverityDeck) ItemCount() int {
	return len(p.data)
}

func (p *SeverityDeck) Render(ctx ViewContext, width, height int, active bool, _ int) string {
	style := sectionStyle.Width(width).Height(height - 2)
	if active {
		style = activeSectionStyle.Width(width).Height(height - 2)
	}

	title := deckTitleStyle.Render(deckTitleWithBadges("Severity Timeline", ctx))

	overhead := 3
	contentLines := height - overhead
	if contentLines < 1 {
		contentLines = 1
	}

	var content string
	if len(p.data) > 0 {
		content = p.renderChart(width, contentLines)
	} else if ctx.DeckLoading {
		content = renderLoadingPlaceholder(width-2, contentLines, ctx.SpinnerFrame)
	} else {
		content = helpStyle.Render("No data available")
	}

	return style.Render(lipgloss.JoinVertical(lipgloss.Left, title, content))
}

func (p *SeverityDeck) OnSelect(_ ViewContext, _ int) tea.Cmd {
	if p.pushModalCmd == nil {
		return nil
	}
	return p.pushModalCmd
}

// chartSeverity defines a severity with its ANSI color code for chart rendering.
type chartSeverity struct {
	name  string
	color string
}

var chartSeverities = []chartSeverity{
	{"TRACE", "240"},
	{"DEBUG", "244"},
	{"INFO", "39"},
	{"WARN", "208"},
	{"ERROR", "196"},
	{"FATAL", "201"},
}

func (p *SeverityDeck) renderChart(deckWidth, availHeight int) string {
	if len(p.data) == 0 {
		return helpStyle.Render("No data available")
	}

	// Responsive bar width and legend visibility
	barWidth := 1
	showLegend := false
	switch {
	case deckWidth >= 160:
		barWidth = 3
		showLegend = true
	case deckWidth >= 100:
		barWidth = 2
		showLegend = true
	case deckWidth >= 60:
		barWidth = 1
		showLegend = true
	default:
		barWidth = 1
		showLegend = false
	}

	legendWidth := 0
	legendGap := 0
	if showLegend {
		legendWidth = 16
		legendGap = 2
	}

	borderChar := 1
	yAxisWidthEstimate := 4
	if deckWidth >= 100 {
		yAxisWidthEstimate = 6
	}

	chartAreaWidth := deckWidth - yAxisWidthEstimate - borderChar - legendGap - legendWidth - 2
	if chartAreaWidth < 10 {
		chartAreaWidth = 10
	}

	chartHeight := availHeight - 2
	if chartHeight < 4 {
		chartHeight = 4
	}

	stride := barWidth + 1
	maxBars := chartAreaWidth / stride
	if maxBars < 1 {
		maxBars = 1
	}

	// Build a full 24h timeline of slots, always covering now-24h → now.
	// Each slot = one bucket. We size buckets so that exactly maxBars fit in 24h.
	now := time.Now().Truncate(time.Minute)
	timelineStart := now.Add(-24 * time.Hour)
	bucketDuration := (24 * time.Hour) / time.Duration(maxBars)
	if bucketDuration < time.Minute {
		bucketDuration = time.Minute
	}
	numBars := maxBars

	// Index source data by Unix timestamp for O(1) lookup (avoids timezone mismatch)
	dataIndex := make(map[int64]*model.MinuteCounts, len(p.data))
	for i := range p.data {
		dataIndex[p.data[i].Minute.Truncate(time.Minute).Unix()] = &p.data[i]
	}

	// Aggregate source data into timeline buckets
	type bucket struct {
		model.MinuteCounts
		midTime time.Time // bucket midpoint for labels
	}
	buckets := make([]bucket, numBars)
	for i := 0; i < numBars; i++ {
		bStart := timelineStart.Add(time.Duration(i) * bucketDuration)
		bEnd := bStart.Add(bucketDuration)
		buckets[i].midTime = bStart.Add(bucketDuration / 2)

		// Sum all source minutes that fall into this bucket
		for t := bStart.Truncate(time.Minute); t.Before(bEnd); t = t.Add(time.Minute) {
			if mc, ok := dataIndex[t.Unix()]; ok {
				buckets[i].Trace += mc.Trace
				buckets[i].Debug += mc.Debug
				buckets[i].Info += mc.Info
				buckets[i].Warn += mc.Warn
				buckets[i].Error += mc.Error
				buckets[i].Fatal += mc.Fatal
				buckets[i].Total += mc.Total
			}
		}
	}

	// Compute Y-axis
	rawMax := int64(0)
	for _, b := range buckets {
		if b.Total > rawMax {
			rawMax = b.Total
		}
	}
	maxTicks := 3
	if deckWidth >= 100 {
		maxTicks = 4
	}
	yCfg := computeYAxis(rawMax, maxTicks)
	yAxisWidth := yCfg.LabelWidth

	barStyles := make(map[string]lipgloss.Style, len(chartSeverities))
	for _, sev := range chartSeverities {
		barStyles[sev.name] = lipgloss.NewStyle().Foreground(lipgloss.Color(sev.color))
	}

	// Render chart rows
	rows := make([]string, chartHeight)
	for row := 0; row < chartHeight; row++ {
		rowTopVal := yCfg.Max - (yCfg.Max*int64(row))/int64(chartHeight)
		rowBotVal := yCfg.Max - (yCfg.Max*int64(row+1))/int64(chartHeight)

		yLabel := renderYLabel(yCfg, row, chartHeight)

		var barArea strings.Builder
		for i, b := range buckets {
			segments := stackedSegments(b.MinuteCounts)
			cellStr := renderBarCell(segments, b.Total, yCfg.Max, rowBotVal, rowTopVal, barWidth, barStyles)
			barArea.WriteString(cellStr)
			if i < numBars-1 {
				barArea.WriteString(" ")
			}
		}

		rows[row] = yLabel + "│" + barArea.String()
	}

	// X-axis line
	xAxisLine := strings.Repeat(" ", yAxisWidth) + "└"
	for i := 0; i < numBars; i++ {
		xAxisLine += strings.Repeat("─", barWidth)
		if i < numBars-1 {
			xAxisLine += "┴"
		}
	}

	// X-axis time labels — evenly spaced across the timeline
	xLabels := buildAdaptiveTimeLabels(timelineStart, now, numBars, yAxisWidth+1, stride, chartAreaWidth)

	// Legend
	var legendLines []string
	if showLegend {
		// Sum totals across all buckets for legend
		var totals model.MinuteCounts
		for _, b := range buckets {
			totals.Trace += b.Trace
			totals.Debug += b.Debug
			totals.Info += b.Info
			totals.Warn += b.Warn
			totals.Error += b.Error
			totals.Fatal += b.Fatal
			totals.Total += b.Total
		}
		legendLines = buildLegendLines(totals, chartHeight+2)
	}

	// Combine chart rows with legend
	var outputLines []string
	for i, row := range rows {
		line := row
		if showLegend && i < len(legendLines) {
			line += strings.Repeat(" ", legendGap) + legendLines[i]
		}
		outputLines = append(outputLines, line)
	}

	xAxisWithLegend := xAxisLine
	if showLegend && len(rows) < len(legendLines) {
		xAxisWithLegend += strings.Repeat(" ", legendGap) + legendLines[len(rows)]
	}
	outputLines = append(outputLines, xAxisWithLegend)

	xLabelsWithLegend := xLabels
	if showLegend && len(rows)+1 < len(legendLines) {
		xLabelsWithLegend += strings.Repeat(" ", legendGap) + legendLines[len(rows)+1]
	}
	outputLines = append(outputLines, xLabelsWithLegend)

	return strings.Join(outputLines, "\n")
}

// stackedSegment represents one severity slice of a stacked bar.
type stackedSegment struct {
	name  string
	count int64
}

func stackedSegments(mc model.MinuteCounts) []stackedSegment {
	// Bottom to top ordering
	return []stackedSegment{
		{"TRACE", mc.Trace},
		{"DEBUG", mc.Debug},
		{"INFO", mc.Info},
		{"WARN", mc.Warn},
		{"ERROR", mc.Error},
		{"FATAL", mc.Fatal},
	}
}

func renderBarCell(segments []stackedSegment, totalHeight, yMax, rowBot, rowTop int64, barWidth int, styles map[string]lipgloss.Style) string {
	if totalHeight == 0 {
		return strings.Repeat(" ", barWidth)
	}

	// Find which severity is at this vertical position
	// Walk segments bottom-up, tracking cumulative height
	cumulative := int64(0)
	for _, seg := range segments {
		if seg.count == 0 {
			continue
		}
		segBot := (cumulative * yMax) / totalHeight
		segTop := ((cumulative + seg.count) * yMax) / totalHeight
		cumulative += seg.count

		// Check if this segment overlaps with this row
		if segTop > rowBot && segBot < rowTop {
			block := strings.Repeat("█", barWidth)
			if style, ok := styles[seg.name]; ok {
				return style.Render(block)
			}
			return block
		}
	}

	// The bar doesn't reach this row height
	return strings.Repeat(" ", barWidth)
}

// buildAdaptiveTimeLabels places evenly-spaced time labels anchored to the
// actual timeline range. Left edge = start time, right edge = end time, with
// adaptive interior labels based on available chart width.
func buildAdaptiveTimeLabels(timelineStart, timelineEnd time.Time, numBars, offset, stride, chartAreaWidth int) string {
	totalWidth := offset + numBars*stride
	buf := make([]byte, totalWidth)
	for i := range buf {
		buf[i] = ' '
	}

	// Determine number of interior labels based on chart width
	numInterior := 0
	switch {
	case chartAreaWidth >= 120:
		numInterior = 3
	case chartAreaWidth >= 60:
		numInterior = 2
	case chartAreaWidth >= 30:
		numInterior = 1
	}

	// Build label time points: start, evenly-spaced interior, end
	totalLabels := numInterior + 2
	type labelInfo struct {
		t      time.Time
		barIdx int
	}
	labels := make([]labelInfo, totalLabels)
	totalSpan := timelineEnd.Sub(timelineStart)

	for i := 0; i < totalLabels; i++ {
		frac := float64(i) / float64(totalLabels-1)
		lt := timelineStart.Add(time.Duration(frac * float64(totalSpan)))
		barIdx := int(float64(numBars-1) * frac)
		if barIdx < 0 {
			barIdx = 0
		}
		if barIdx >= numBars {
			barIdx = numBars - 1
		}
		labels[i] = labelInfo{t: lt, barIdx: barIdx}
	}

	// Pick label format based on time span
	//   <= 24h  → "15:04"       (e.g. "09:30")
	//   <= 7d   → "Mon 15:04"   (e.g. "Tue 09:30")
	//   > 7d    → "Jan 02"      (e.g. "Mar 01")
	labelFmt := "15:04"
	switch {
	case totalSpan > 7*24*time.Hour:
		labelFmt = "Jan 02"
	case totalSpan > 24*time.Hour:
		labelFmt = "Mon 15:04"
	}

	// Place labels, skipping any that would overlap
	lastEnd := -1
	for _, l := range labels {
		label := l.t.Format(labelFmt)
		minSpacing := len(label) + 2

		pos := offset + l.barIdx*stride
		// Center the label on the bar position
		pos -= len(label) / 2
		if pos < offset {
			pos = offset
		}
		if pos+len(label) > totalWidth {
			pos = totalWidth - len(label)
		}
		if pos < 0 {
			continue
		}
		// Overlap prevention
		if lastEnd >= 0 && pos < lastEnd+minSpacing-len(label)+2 {
			continue
		}
		copy(buf[pos:pos+len(label)], label)
		lastEnd = pos + len(label)
	}

	return string(buf)
}

func buildLegendLines(latest model.MinuteCounts, totalLines int) []string {
	type legendEntry struct {
		name  string
		count int64
		color string
	}
	entries := []legendEntry{
		{"FATAL", latest.Fatal, "201"},
		{"ERROR", latest.Error, "196"},
		{"WARN", latest.Warn, "208"},
		{"INFO", latest.Info, "39"},
		{"DEBUG", latest.Debug, "244"},
		{"TRACE", latest.Trace, "240"},
	}

	var lines []string
	for _, e := range entries {
		label := fmt.Sprintf("%-6s:", e.name)
		value := fmt.Sprintf("%5d", e.count)
		colorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(e.color))
		lines = append(lines, colorStyle.Render(label+value))
	}
	// Separator
	sepStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
	lines = append(lines, sepStyle.Render("─────────────"))
	// Total
	totalStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
	lines = append(lines, totalStyle.Render(fmt.Sprintf("TOTAL: %5d", latest.Total)))

	// Pad to fill vertical space
	for len(lines) < totalLines {
		lines = append(lines, "")
	}

	return lines
}

// yAxisConfig holds computed Y-axis parameters.
type yAxisConfig struct {
	Max        int64   // nice-rounded maximum
	Ticks      []int64 // tick values from 0 to Max (ascending, deduplicated)
	LabelWidth int     // character width for labels (auto-computed from Max)
}

// computeYAxis computes professional Y-axis ticks for any data range.
// maxTicks is the desired number of tick intervals (actual count may be fewer
// to avoid duplicate labels at small ranges).
func computeYAxis(rawMax int64, maxTicks int) yAxisConfig {
	if rawMax <= 0 {
		rawMax = 1
	}
	yMax := niceMax(rawMax)

	step := niceTickStep(yMax, maxTicks)

	ticks := make([]int64, 0, maxTicks+1)
	for v := int64(0); v <= yMax; v += step {
		if len(ticks) == 0 || v != ticks[len(ticks)-1] {
			ticks = append(ticks, v)
		}
	}
	if len(ticks) == 0 || ticks[len(ticks)-1] != yMax {
		ticks = append(ticks, yMax)
	}

	labelWidth := len(fmt.Sprintf("%d", yMax)) + 1
	if labelWidth < 3 {
		labelWidth = 3
	}

	return yAxisConfig{Max: yMax, Ticks: ticks, LabelWidth: labelWidth}
}

// niceTickStep returns a "nice" step size that divides the range into
// at most maxTicks intervals. Returns values like 1, 2, 5, 10, 20, 50...
func niceTickStep(yMax int64, maxTicks int) int64 {
	if maxTicks <= 0 {
		maxTicks = 1
	}
	if yMax <= int64(maxTicks) {
		return 1
	}

	rawStep := float64(yMax) / float64(maxTicks)
	magnitude := math.Pow(10, math.Floor(math.Log10(rawStep)))
	residual := rawStep / magnitude

	var niceStep float64
	switch {
	case residual <= 1.0:
		niceStep = 1.0
	case residual <= 2.0:
		niceStep = 2.0
	case residual <= 5.0:
		niceStep = 5.0
	default:
		niceStep = 10.0
	}

	step := int64(niceStep * magnitude)
	if step <= 0 {
		step = 1
	}
	return step
}

// renderYLabel returns the Y-axis label string for a given chart row.
// It maps tick values to exact row positions — no duplicates possible.
func renderYLabel(cfg yAxisConfig, row, chartHeight int) string {
	blank := strings.Repeat(" ", cfg.LabelWidth)

	for _, tick := range cfg.Ticks {
		var tickRow int
		if cfg.Max == 0 {
			tickRow = chartHeight - 1
		} else {
			tickRow = int(math.Round(float64(chartHeight-1) * float64(cfg.Max-tick) / float64(cfg.Max)))
		}

		if tickRow == row {
			return fmt.Sprintf("%*d ", cfg.LabelWidth-1, tick)
		}
	}

	return blank
}

// niceMax rounds up yMax to a "nice" axis value.
func niceMax(v int64) int64 {
	if v <= 0 {
		return 1
	}
	niceSteps := []int64{1, 2, 5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000, 10000, 25000, 50000, 100000}
	for _, step := range niceSteps {
		rounded := ((v + step - 1) / step) * step
		if rounded >= v {
			return rounded
		}
	}
	// Fallback for very large values
	magnitude := int64(math.Pow(10, math.Floor(math.Log10(float64(v)))))
	return ((v + magnitude - 1) / magnitude) * magnitude
}
