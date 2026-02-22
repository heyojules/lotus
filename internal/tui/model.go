package tui

import (
	"regexp"
	"time"

	"github.com/control-theory/lotus/internal/model"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// Section represents different dashboard sections
type Section int

const (
	SectionSidebar Section = iota // app sidebar
	SectionCharts                 // a chart panel is focused
	SectionFilter                 // filter bar
	SectionLogs                   // log scroll
)

// PatternCount represents a pattern and its count for a specific severity
type PatternCount struct {
	Pattern string
	Count   int
}

// AttributeEntry holds per-key attribute stats with value breakdown.
type AttributeEntry struct {
	Key              string
	UniqueValueCount int
	TotalCount       int64
	Values           map[string]int64
}

// StatsTracker tracks processing statistics derived from DuckDB count deltas.
type StatsTracker struct {
	StartTime      time.Time
	TotalBytes     int64
	PeakLogsPerSec float64
	LastSecond     time.Time
	LogsThisSecond int         // Delta logs since last tick (used by formatCurrentRate)
	TotalLogsEver  int         // Total logs from DuckDB (refreshed on tick)
	RecentCounts   []int       // Count of logs per tick (sliding window)
	RecentTimes    []time.Time // Timestamp for each tick

	// Tick-based delta tracking
	lastTickCount int       // Total log count at previous tick
	lastTickTime  time.Time // Timestamp of previous tick
}

// VersionInfo holds version update status without importing the version package.
type VersionInfo struct {
	HasUpdate     bool
	LatestVersion string
}

// FilterState holds text filter, search, and severity filter state.
type FilterState struct {
	filterInput  textinput.Model
	filterActive bool
	filterRegex  *regexp.Regexp

	searchInput  textinput.Model
	searchActive bool
	searchTerm   string // For 's' command - highlights just the term

	severityFilter       map[string]bool // Which severity levels are enabled (true = show, false = hide)
	severityFilterActive bool            // Whether severity filtering is active (any severity disabled)
}

// SidebarState holds app sidebar state.
type SidebarState struct {
	selectedApp    string           // "" = all apps (global view)
	appList        []string         // from store.ListApps(), refreshed on tick
	appListIdx     int              // sidebar selection index (0 = "All")
	sidebarVisible bool             // toggled with 'a'
	appCounts      map[string]int64 // per-app log counts, refreshed on tick (not render)
}

// ModalStackState holds the modal stack that replaces boolean flag explosion.
type ModalStackState struct {
	modalStack []Modal
}

// NavigationState holds panel and section navigation state.
type NavigationState struct {
	activeSection  Section
	activePanelIdx int
	panels         []ChartPanel
	panelSelIdx    []int
}

// LogViewState holds log entries and scroll/selection state.
type LogViewState struct {
	logEntries               []model.LogRecord // Filtered view for display (refreshed from DuckDB)
	selectedLogIndex         int        // For log section navigation
	viewPaused               bool       // Pause view updates when navigating logs
	logAutoScroll            bool       // Auto-scroll to latest logs in log viewer
	instructionsScrollOffset int        // Scroll position for instructions/filter status screen
	showColumns              bool       // Toggle Host and Service columns in log view
}

// DashboardModel represents the main TUI model.
// Sub-state is organized into embedded structs for readability;
// Go's field promotion means existing m.fieldName access is unchanged.
type DashboardModel struct {
	// Composed sub-states (embedded for field promotion)
	FilterState
	SidebarState
	ModalStackState
	NavigationState
	LogViewState

	// Window dimensions
	width  int
	height int

	// Data — refreshed every tick from DuckDB
	countsHistory []SeverityCounts // Line counts per interval by severity

	// Drain3 pattern extraction (per-severity instances for counts modal)
	drain3BySeverity map[string]*Drain3Manager // Separate drain3 instance for each severity

	// Configuration
	updateInterval     time.Duration
	reverseScrollWheel bool
	useLogTime         bool // Use OrigTimestamp instead of Timestamp for heatmap/display

	// Update interval management
	availableIntervals []time.Duration
	currentIntervalIdx int

	// Drain3 pattern extraction
	drain3Manager       *Drain3Manager
	drain3LastProcessed int // Track last processed log count for incremental drain3 feeding

	// Statistics tracking
	stats StatsTracker

	// Inline handlers for filter/search input (NOT modals — part of dashboard layout)
	inlineHandlers []inlineHandlerEntry

	// DuckDB read primitives used by the TUI.
	store      model.LogQuerier
	dataSource string // "Socket" or "DuckDB" — shown in status bar

	// Version update info (decoupled from version package)
	versionInfo *VersionInfo
}

// TickMsg represents periodic updates
type TickMsg time.Time

// UpdateIntervalMsg represents a request to change update interval
type UpdateIntervalMsg time.Duration

// ManualResetMsg represents a manual reset request triggered by user
type ManualResetMsg struct{}

// initializeDrain3BySeverity creates separate drain3 instances for each severity level
func initializeDrain3BySeverity() map[string]*Drain3Manager {
	severities := []string{"FATAL", "ERROR", "WARN", "INFO", "DEBUG", "TRACE", "UNKNOWN"}
	drain3Map := make(map[string]*Drain3Manager)

	for _, severity := range severities {
		drain3Map[severity] = NewDrain3Manager()
	}

	return drain3Map
}

// NewDashboardModel creates a new dashboard model.
func NewDashboardModel(maxLogBuffer int, updateInterval time.Duration, reverseScrollWheel bool, useLogTime bool, store model.LogQuerier, dataSource string) *DashboardModel {
	filterInput := textinput.New()
	filterInput.Placeholder = "Filter logs by message or attributes (regex supported)..."
	filterInput.CharLimit = 200

	searchInput := textinput.New()
	searchInput.Placeholder = "Search and highlight text..."
	searchInput.CharLimit = 200

	// Available update intervals
	availableIntervals := []time.Duration{
		500 * time.Millisecond,
		1 * time.Second,
		2 * time.Second,
		5 * time.Second,
		10 * time.Second,
		30 * time.Second,
		1 * time.Minute,
	}

	// Find current interval index
	currentIdx := 2 // Default to 2 seconds if not found
	for i, interval := range availableIntervals {
		if interval == updateInterval {
			currentIdx = i
			break
		}
	}

	m := &DashboardModel{
		FilterState: FilterState{
			filterInput: filterInput,
			searchInput: searchInput,
			severityFilter: map[string]bool{
				"TRACE": true, "DEBUG": true, "INFO": true, "WARN": true,
				"ERROR": true, "FATAL": true, "CRITICAL": true, "UNKNOWN": true,
			},
		},
		SidebarState: SidebarState{
			sidebarVisible: true,
		},
		NavigationState: NavigationState{
			activeSection:  SectionCharts,
			activePanelIdx: 0,
		},
		LogViewState: LogViewState{
			logEntries:    make([]model.LogRecord, 0, maxLogBuffer),
			logAutoScroll: true,
			showColumns:   true,
		},

		updateInterval:     updateInterval,
		reverseScrollWheel: reverseScrollWheel,
		useLogTime:         useLogTime,
		countsHistory:      make([]SeverityCounts, 0),
		drain3BySeverity:   initializeDrain3BySeverity(),
		availableIntervals: availableIntervals,
		currentIntervalIdx: currentIdx,
		drain3Manager:      NewDrain3Manager(),
		stats: StatsTracker{
			StartTime:    time.Now(),
			LastSecond:   time.Now(),
			RecentCounts: make([]int, 0, 10),
			RecentTimes:  make([]time.Time, 0, 10),
			lastTickTime: time.Now(),
		},
		store:      store,
		dataSource: dataSource,
	}

	defaultPanels := []ChartPanel{
		NewWordsChartPanel(),
		NewAttributesChartPanel(m),
		NewPatternsChartPanel(m),
		NewCountsChartPanel(m),
	}

	m.SetPanels(defaultPanels)

	// Register inline handlers for filter/search input (NOT modals).
	m.inlineHandlers = []inlineHandlerEntry{
		{isActive: func(m *DashboardModel) bool { return m.filterActive }, handler: filterInputHandler{}},
		{isActive: func(m *DashboardModel) bool { return m.searchActive }, handler: searchInputHandler{}},
	}

	return m
}

// SetPanels replaces chart panels and resets panel selection state.
func (m *DashboardModel) SetPanels(panels []ChartPanel) {
	if len(panels) == 0 {
		m.panels = nil
		m.panelSelIdx = nil
		m.activePanelIdx = 0
		return
	}

	m.panels = append([]ChartPanel(nil), panels...)
	m.panelSelIdx = make([]int, len(m.panels))
	if m.activePanelIdx >= len(m.panels) {
		m.activePanelIdx = 0
	}
}

// queryOpts returns the current QueryOpts based on selected app.
func (m *DashboardModel) queryOpts() model.QueryOpts {
	return model.QueryOpts{App: m.selectedApp}
}

// viewContext builds a ViewContext snapshot for chart panel rendering.
func (m *DashboardModel) viewContext() ViewContext {
	return ViewContext{
		ContentWidth:  m.contentWidth(),
		ContentHeight: m.height,
		SearchTerm:    m.searchTerm,
		SelectedApp:   m.selectedApp,
		UseLogTime:    m.useLogTime,
	}
}


// DashboardPage adapts DashboardModel to the Page interface.
type DashboardPage struct {
	Model *DashboardModel
}

// NewDashboardPage wraps a DashboardModel as a Page.
func NewDashboardPage(m *DashboardModel) *DashboardPage {
	return &DashboardPage{Model: m}
}

func (p *DashboardPage) ID() string { return "dashboard" }

func (p *DashboardPage) Init() tea.Cmd {
	return p.Model.Init()
}

func (p *DashboardPage) Update(msg tea.Msg) (tea.Cmd, *PageNav) {
	_, cmd := p.Model.Update(msg)
	return cmd, nil // no navigation yet
}

func (p *DashboardPage) View(width, height int) string {
	p.Model.width = width
	p.Model.height = height
	return p.Model.View()
}

// SetVersionInfo sets version update info for display in the status line.
func (m *DashboardModel) SetVersionInfo(info *VersionInfo) {
	m.versionInfo = info
}

// hasK8sAttributes returns true if recent logs have k8s namespace/pod attributes
func (m *DashboardModel) hasK8sAttributes() bool {
	checkCount := min(10, len(m.logEntries))
	for i := len(m.logEntries) - checkCount; i < len(m.logEntries); i++ {
		if i < 0 {
			continue
		}
		entry := m.logEntries[i]
		if entry.Attributes["k8s.namespace"] != "" || entry.Attributes["k8s.pod"] != "" {
			return true
		}
	}
	return false
}


// Init initializes the model
func (m *DashboardModel) Init() tea.Cmd {
	var cmds []tea.Cmd

	// Enable mouse support
	cmds = append(cmds, func() tea.Msg { return tea.EnableMouseCellMotion() })

	// Set up regular tick for dashboard updates
	cmds = append(cmds, tea.Tick(m.updateInterval, func(t time.Time) tea.Msg {
		return TickMsg(t)
	}))

	return tea.Batch(cmds...)
}

// GetCountsHistory returns the current counts history for debugging
func (m *DashboardModel) GetCountsHistory() []SeverityCounts {
	return m.countsHistory
}

// PushModal pushes a modal onto the stack. Deduplicates by ID.
func (m *DashboardModel) PushModal(modal Modal) {
	for _, existing := range m.modalStack {
		if existing.ID() == modal.ID() {
			return
		}
	}
	m.modalStack = append(m.modalStack, modal)
}

// PopModal removes the topmost modal from the stack.
func (m *DashboardModel) PopModal() {
	if len(m.modalStack) > 0 {
		m.modalStack = m.modalStack[:len(m.modalStack)-1]
	}
}

// TopModal returns the topmost modal, or nil if the stack is empty.
func (m *DashboardModel) TopModal() Modal {
	if len(m.modalStack) == 0 {
		return nil
	}
	return m.modalStack[len(m.modalStack)-1]
}

// HasModal returns true if any modal is on the stack.
func (m *DashboardModel) HasModal() bool {
	return len(m.modalStack) > 0
}

// isLogViewerOpen returns true if the log viewer modal is currently on the stack.
func (m *DashboardModel) isLogViewerOpen() bool {
	for _, modal := range m.modalStack {
		if modal.ID() == "logviewer" {
			return true
		}
	}
	return false
}
