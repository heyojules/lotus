package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/control-theory/lotus/internal/model"

	"github.com/charmbracelet/lipgloss"
)

// renderStatsContent renders the detailed statistics content using data from the StatsModal.
func (m *DashboardModel) renderStatsContent(contentWidth int, sm *StatsModal) string {
	var sections []string

	// Title section
	titleStyle := lipgloss.NewStyle().
		Foreground(ColorBlue).
		Bold(true).
		Align(lipgloss.Center).
		Width(contentWidth)

	sections = append(sections, titleStyle.Render("Log Analysis Statistics"))
	sections = append(sections, "")

	totalLogs := m.currentTotalLogs()
	totalBytes := sm.totalLogBytes

	// Calculate width for side-by-side sections (with spacing)
	halfWidth := (contentWidth - 3) / 2 // -3 for spacing between columns

	// Row 1: General Statistics | Severity Distribution (side by side)
	generalStats := m.renderStatsSection("General Statistics", []StatItem{
		{"Total Logs Processed", fmt.Sprintf("%d", totalLogs)},
		{"Logs in DuckDB", fmt.Sprintf("%d", totalLogs)},
		{"Filtered Logs Displayed", fmt.Sprintf("%d", len(m.logEntries))},
		{"Total Bytes Processed", m.formatBytes(totalBytes)},
		{"Uptime", m.formatUptime()},
		{"Current Processing Rate", m.formatCurrentRate()},
		{"Peak Logs per Second", fmt.Sprintf("%.1f", m.stats.PeakLogsPerSec)},
	}, halfWidth)

	// Severity Statistics with visual bar chart
	severityStats := m.calculateSeverityStatsFrom(sm.severityCounts)
	severitySection := m.renderSeveritySectionFrom(sm.severityCounts, severityStats, halfWidth)

	// Combine general and severity side by side
	row1 := m.combineSideBySide(generalStats, severitySection)
	sections = append(sections, row1)

	// Host Statistics Section
	hostStats := calculateHostStatsFrom(sm.hostStats, totalLogs)
	if len(hostStats) > 0 {
		sections = append(sections, m.renderStatsSection("Top Hosts", hostStats[:min(10, len(hostStats))], contentWidth))
	}

	// Row 2: Top Services | Pattern Analysis (side by side)
	serviceStats := calculateServiceStatsFrom(sm.serviceStats, totalLogs)
	var row2 string

	if len(serviceStats) > 0 {
		servicesSection := m.renderStatsSection("Top Services", serviceStats[:min(10, len(serviceStats))], halfWidth)

		// Pattern Statistics Section (if available)
		if m.drain3Manager != nil {
			patternCount, totalLogs := m.drain3Manager.GetStats()
			if patternCount > 0 {
				patternStats := []StatItem{
					{"Unique Patterns Detected", fmt.Sprintf("%d", patternCount)},
					{"Pattern Compression Ratio", fmt.Sprintf("%.1f:1", float64(totalLogs)/float64(patternCount))},
					{"Logs Analyzed for Patterns", fmt.Sprintf("%d", totalLogs)},
				}
				patternSection := m.renderStatsSection("Pattern Analysis", patternStats, halfWidth)
				row2 = m.combineSideBySide(servicesSection, patternSection)
			} else {
				row2 = servicesSection
			}
		} else {
			row2 = servicesSection
		}
		sections = append(sections, row2)
	}

	// Attribute Statistics Section (formatted with columns)
	attributeStats := calculateAttributeStatsFormattedFrom(sm.attributeStats, totalLogs)
	if len(attributeStats) > 0 {
		sections = append(sections, m.renderAttributeSection(attributeStats[:min(15, len(attributeStats))], contentWidth))
	}

	return strings.Join(sections, "\n")
}

// StatItem represents a statistics key-value pair
type StatItem struct {
	Key   string
	Value string
}

// renderStatsSection renders a section of statistics with consistent formatting (using dashboard styling)
func (m *DashboardModel) renderStatsSection(title string, items []StatItem, width int) string {
	// Use chartTitleStyle for consistent title formatting
	titleContent := chartTitleStyle.Render(title)

	var contentLines []string

	// Calculate the maximum key length for alignment
	maxKeyLen := 0
	for _, item := range items {
		if len(item.Key) > maxKeyLen {
			maxKeyLen = len(item.Key)
		}
	}

	// Add padding
	maxKeyLen += 3

	// Render each statistic item with aligned values
	for _, item := range items {
		keyStyle := lipgloss.NewStyle().
			Foreground(ColorWhite).
			Width(maxKeyLen).
			Align(lipgloss.Left)

		valueStyle := lipgloss.NewStyle().
			Foreground(ColorBlue).
			Bold(true)

		line := fmt.Sprintf("%s %s",
			keyStyle.Render(item.Key+":"),
			valueStyle.Render(item.Value))
		contentLines = append(contentLines, line)
	}

	content := strings.Join(contentLines, "\n")

	// Use sectionStyle for consistent section formatting with borders
	sectionContent := lipgloss.JoinVertical(lipgloss.Left, titleContent, content)

	return sectionStyle.
		Width(width).
		Render(sectionContent)
}

// Helper functions for statistics calculations

func (m *DashboardModel) calculateSeverityStatsFrom(counts map[string]int64) []StatItem {
	var stats []StatItem
	severityOrder := []string{"FATAL", "ERROR", "WARN", "INFO", "DEBUG", "TRACE"}
	colors := map[string]lipgloss.Color{
		"FATAL": ColorRed, "ERROR": ColorRed, "WARN": ColorOrange,
		"INFO": ColorBlue, "DEBUG": ColorGray, "TRACE": ColorGray,
	}

	if counts == nil {
		counts = map[string]int64{}
	}
	total := m.currentTotalLogs()
	if total == 0 {
		total = 1 // Avoid division by zero
	}

	for _, sev := range severityOrder {
		count := counts[sev]
		percentage := float64(count) * 100.0 / float64(total)
		valueStyle := lipgloss.NewStyle().Foreground(colors[sev])
		value := valueStyle.Render(fmt.Sprintf("%d (%.1f%%)", count, percentage))
		stats = append(stats, StatItem{sev, value})
	}

	return stats
}

func calculateHostStatsFrom(hostStats []model.DimensionCount, total int64) []StatItem {
	if len(hostStats) == 0 {
		return nil
	}
	return formatDimensionStats(hostStats, total)
}

func calculateServiceStatsFrom(serviceStats []model.DimensionCount, total int64) []StatItem {
	if len(serviceStats) == 0 {
		return nil
	}
	return formatDimensionStats(serviceStats, total)
}

func (m *DashboardModel) formatBytes(bytes int64) string {
	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	} else if bytes < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(bytes)/1024)
	} else if bytes < 1024*1024*1024 {
		return fmt.Sprintf("%.1f MB", float64(bytes)/(1024*1024))
	} else {
		return fmt.Sprintf("%.1f GB", float64(bytes)/(1024*1024*1024))
	}
}

func (m *DashboardModel) formatUptime() string {
	if m.stats.StartTime.IsZero() {
		return "0s"
	}
	duration := time.Since(m.stats.StartTime)

	if duration < time.Minute {
		return fmt.Sprintf("%.0fs", duration.Seconds())
	} else if duration < time.Hour {
		return fmt.Sprintf("%.1fm", duration.Minutes())
	} else {
		return fmt.Sprintf("%.1fh", duration.Hours())
	}
}

func (m *DashboardModel) formatCurrentRate() string {
	// If no historical data yet, show current second rate
	if len(m.stats.RecentCounts) == 0 {
		return fmt.Sprintf("%.1f logs/sec", float64(m.stats.LogsThisSecond))
	}

	// Calculate average over recent window (last 5 seconds for more responsive rate)
	totalLogs := 0
	validSeconds := 0
	cutoffTime := time.Now().Add(-5 * time.Second)

	// Count logs from recent complete seconds
	for i, timestamp := range m.stats.RecentTimes {
		if timestamp.After(cutoffTime) {
			totalLogs += m.stats.RecentCounts[i]
			validSeconds++
		}
	}

	// Always include current partial second (even if 0)
	totalLogs += m.stats.LogsThisSecond
	validSeconds++

	if validSeconds == 0 {
		return "0.0 logs/sec"
	}

	// Calculate rate over the window
	rate := float64(totalLogs) / float64(validSeconds)
	return fmt.Sprintf("%.1f logs/sec", rate)
}

// combineSideBySide combines two sections side by side (using lipgloss like the main dashboard)
func (m *DashboardModel) combineSideBySide(left, right string) string {
	// Use lipgloss.JoinHorizontal for consistent layout like the main dashboard
	return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
}

// renderSeveritySectionFrom renders severity statistics with a visual bar chart using provided counts.
func (m *DashboardModel) renderSeveritySectionFrom(counts map[string]int64, stats []StatItem, width int) string {
	// Use chartTitleStyle for consistent title formatting
	titleContent := chartTitleStyle.Render("Severity Distribution")

	var contentLines []string

	// Calculate the maximum key length for alignment
	maxKeyLen := 0
	for _, item := range stats {
		if len(item.Key) > maxKeyLen {
			maxKeyLen = len(item.Key)
		}
	}

	// Add padding
	maxKeyLen += 3

	// Render each statistic item with aligned values
	for _, item := range stats {
		keyStyle := lipgloss.NewStyle().
			Foreground(ColorWhite).
			Width(maxKeyLen).
			Align(lipgloss.Left)

		line := fmt.Sprintf("%s %s",
			keyStyle.Render(item.Key+":"),
			item.Value) // Value already has color styling
		contentLines = append(contentLines, line)
	}

	contentLines = append(contentLines, "") // Add spacing

	// Add horizontal stacked bar chart
	contentLines = append(contentLines, m.renderSeverityBarChartFrom(counts, width-4)) // Account for section padding

	content := strings.Join(contentLines, "\n")

	// Use sectionStyle for consistent section formatting with borders
	sectionContent := lipgloss.JoinVertical(lipgloss.Left, titleContent, content)

	return sectionStyle.
		Width(width).
		Render(sectionContent)
}

// renderSeverityBarChartFrom creates a horizontal stacked bar chart for severity distribution.
func (m *DashboardModel) renderSeverityBarChartFrom(counts map[string]int64, width int) string {
	// Use full available width for the bar
	barWidth := width - 2 // Account for border characters │ │

	severityOrder := []string{"FATAL", "ERROR", "WARN", "INFO", "DEBUG", "TRACE"}
	colors := map[string]lipgloss.Color{
		"FATAL": ColorRed, "ERROR": ColorRed, "WARN": ColorOrange,
		"INFO": ColorBlue, "DEBUG": ColorGray, "TRACE": ColorGray,
	}

	// Build the bar
	var bar string
	total := int(m.currentTotalLogs())
	if counts == nil {
		counts = map[string]int64{}
	}
	if total == 0 {
		// Empty bar if no logs
		bar = strings.Repeat("░", barWidth)
	} else {
		remainingWidth := barWidth
		for _, sev := range severityOrder {
			count := int(counts[sev])
			if count > 0 && remainingWidth > 0 {
				// Calculate width for this severity
				segmentWidth := int(float64(count) * float64(barWidth) / float64(total))
				if segmentWidth == 0 && count > 0 {
					segmentWidth = 1 // At least 1 char for non-zero counts
				}
				if segmentWidth > remainingWidth {
					segmentWidth = remainingWidth
				}

				// Add colored segment
				style := lipgloss.NewStyle().Foreground(colors[sev])
				bar += style.Render(strings.Repeat("█", segmentWidth))
				remainingWidth -= segmentWidth
			}
		}

		// Fill any remaining space with empty bar
		if remainingWidth > 0 {
			bar += strings.Repeat("░", remainingWidth)
		}
	}

	return fmt.Sprintf("│%s│", bar)
}

// AttributeStatFormatted represents a formatted attribute statistic
type AttributeStatFormatted struct {
	Key        string
	Value      string
	Count      int
	Percentage float64
}

// calculateAttributeStatsFormattedFrom returns formatted attribute statistics from the given data.
func calculateAttributeStatsFormattedFrom(attributeStats []model.AttributeStat, total int64) []AttributeStatFormatted {
	if len(attributeStats) == 0 {
		return nil
	}
	if total == 0 {
		total = 1
	}
	stats := make([]AttributeStatFormatted, 0, len(attributeStats))
	for _, attr := range attributeStats {
		if attr.Key == "host" || attr.Key == "service.name" || attr.Key == "service" {
			continue
		}
		stats = append(stats, AttributeStatFormatted{
			Key:        attr.Key,
			Value:      attr.Value,
			Count:      int(attr.Count),
			Percentage: float64(attr.Count) * 100.0 / float64(total),
		})
	}
	return stats
}

func (m *DashboardModel) currentTotalLogs() int64 {
	return int64(m.stats.TotalLogsEver)
}


func formatDimensionStats(rows []model.DimensionCount, total int64) []StatItem {
	if total <= 0 {
		total = 1
	}

	stats := make([]StatItem, 0, len(rows))
	for _, row := range rows {
		pct := float64(row.Count) * 100.0 / float64(total)
		stats = append(stats, StatItem{
			Key:   row.Value,
			Value: fmt.Sprintf("%d (%.1f%%)", row.Count, pct),
		})
	}
	return stats
}

// renderAttributeSection renders the attribute statistics section with columnar format (using dashboard styling)
func (m *DashboardModel) renderAttributeSection(stats []AttributeStatFormatted, width int) string {
	// Use chartTitleStyle for consistent title formatting
	titleContent := chartTitleStyle.Render("Top Attributes")

	var contentLines []string

	// Calculate column widths using full available width
	availableWidth := width - 4 // Account for section borders and padding

	// Reserve fixed space for count column and separators
	countColumnWidth := 15 // Fixed width for "Count (%)" column
	separatorWidth := 6    // " │ " separators (2 * 3 chars)

	// Use remaining space for key and value columns
	keyValueWidth := availableWidth - countColumnWidth - separatorWidth

	// Find actual max lengths in the data
	actualMaxKeyLen := 0
	actualMaxValueLen := 0
	for _, stat := range stats {
		if len(stat.Key) > actualMaxKeyLen {
			actualMaxKeyLen = len(stat.Key)
		}
		if len(stat.Value) > actualMaxValueLen {
			actualMaxValueLen = len(stat.Value)
		}
	}

	// Distribute available space between key and value columns
	// Give them proportional space based on their actual content, with reasonable minimums
	minKeyLen := max(8, min(actualMaxKeyLen, 15))      // At least 8, prefer actual up to 15
	minValueLen := max(12, min(actualMaxValueLen, 20)) // At least 12, prefer actual up to 20

	var maxKeyLen, maxValueLen int

	if minKeyLen+minValueLen <= keyValueWidth {
		// If we have enough space, distribute the extra proportionally
		extraSpace := keyValueWidth - minKeyLen - minValueLen
		if actualMaxKeyLen+actualMaxValueLen > 0 {
			keyRatio := float64(actualMaxKeyLen) / float64(actualMaxKeyLen+actualMaxValueLen)
			maxKeyLen = minKeyLen + int(float64(extraSpace)*keyRatio)
			maxValueLen = minValueLen + int(float64(extraSpace)*(1-keyRatio))
		} else {
			// Equal split if no actual data
			maxKeyLen = minKeyLen + extraSpace/2
			maxValueLen = minValueLen + extraSpace/2
		}
	} else {
		// If we don't have enough space, scale down proportionally
		ratio := float64(keyValueWidth) / float64(minKeyLen+minValueLen)
		maxKeyLen = max(8, int(float64(minKeyLen)*ratio))
		maxValueLen = max(8, int(float64(minValueLen)*ratio))
	}

	// Header
	headerStyle := lipgloss.NewStyle().Foreground(ColorWhite).Bold(true)
	header := fmt.Sprintf("%-*s │ %-*s │ %s",
		maxKeyLen, "Key",
		maxValueLen, "Value",
		"Count (%)")
	contentLines = append(contentLines, headerStyle.Render(header))

	// Divider line
	dividerStyle := lipgloss.NewStyle().Foreground(ColorGray)
	contentLines = append(contentLines, dividerStyle.Render(strings.Repeat("─", len(header))))

	// Render each attribute
	for _, stat := range stats {
		key := stat.Key
		if len(key) > maxKeyLen {
			key = key[:maxKeyLen-3] + "..."
		}

		value := stat.Value
		if len(value) > maxValueLen {
			value = value[:maxValueLen-3] + "..."
		}

		keyStyle := lipgloss.NewStyle().Foreground(ColorWhite)
		valueStyle := lipgloss.NewStyle().Foreground(ColorBlue)
		countStyle := lipgloss.NewStyle().Foreground(ColorGreen).Bold(true)

		line := fmt.Sprintf("%s │ %s │ %s",
			keyStyle.Render(fmt.Sprintf("%-*s", maxKeyLen, key)),
			valueStyle.Render(fmt.Sprintf("%-*s", maxValueLen, value)),
			countStyle.Render(fmt.Sprintf("%d (%.1f%%)", stat.Count, stat.Percentage)))

		contentLines = append(contentLines, line)
	}

	content := strings.Join(contentLines, "\n")

	// Use sectionStyle for consistent section formatting with borders
	sectionContent := lipgloss.JoinVertical(lipgloss.Left, titleContent, content)

	return sectionStyle.
		Width(width).
		Render(sectionContent)
}
