package tui

import (
	"regexp"
	"time"

	"github.com/tinytelemetry/lotus/internal/model"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// Section represents different dashboard sections
type Section int

const (
	SectionSidebar Section = iota // app sidebar
	SectionDecks                 // a deck is focused
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
	selectedApp    string   // "" = all apps (global view)
	appList        []string // from store.ListApps(), refreshed on tick
	sidebarCursor  int      // unified sidebar cursor (pages + apps)
	sidebarVisible bool     // toggled with 'a'
}

// ModalStackState holds the modal stack that replaces boolean flag explosion.
type ModalStackState struct {
	modalStack []Modal
}

// NavigationState holds panel and section navigation state.
type NavigationState struct {
	activeSection  Section
	activeDeckIdx int
	decks         []Deck
	deckSelIdx    []int
	views      []ViewState
	activeViewIdx  int
}

// ViewState represents one right-side view composed of independent decks.
type ViewState struct {
	ID             string
	Title          string
	Decks         []Deck
	DeckSelIdx    []int
	ActiveDeckIdx int
}

// DeckDeps provides dependencies for deck constructors, replacing *DashboardModel.
type DeckDeps struct {
	Store          model.LogQuerier
	Drain3Manager  *Drain3Manager
	PushCountsModal  tea.Cmd
	PushPatternsModal tea.Cmd
	FormatAttrModal  func(entry *AttributeEntry, maxWidth int) string
	PushContentModal func(content string) tea.Cmd
}

// ViewSpec defines how to build a view and its decks.
type ViewSpec struct {
	ID    string
	Title string
	Build func(deps DeckDeps) []Deck
}

// LogViewState holds log entries and scroll/selection state.
type LogViewState struct {
	logEntries               []model.LogRecord // Filtered view for display (refreshed from DuckDB)
	selectedLogIndex         int               // For log section navigation
	viewPaused               bool              // Pause view updates when navigating logs
	logAutoScroll            bool              // Auto-scroll to latest logs in log viewer
	instructionsScrollOffset int               // Scroll position for instructions/filter status screen
	showColumns              bool              // Toggle Host and Service columns in log view
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

	// Last DB error for status line display (auto-clears after 30s).
	lastError   string
	lastErrorAt time.Time

	// Async tick query guard to avoid overlapping DB fetches.
	tickInFlight bool

	// Inline handlers for filter/search input (NOT modals — part of dashboard layout)
	inlineHandlers []inlineHandlerEntry

	// Source connectivity tracking
	lastTickOK        bool      // whether the most recent tick had no errors
	lastTickAt        time.Time // when the last successful tick completed
	consecutiveErrors int       // count of consecutive ticks with errors

	// Per-panel-type tick/pause/error tracking.
	deckStates map[string]*DeckTypeState

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

// DeckTickMsg fires independently for each deck type.
type DeckTickMsg struct {
	DeckTypeID string
	At          time.Time
}

// DeckDataMsg carries fetched data back to a deck type.
type DeckDataMsg struct {
	DeckTypeID string
	Data        interface{}
	Err         error
}

// DeckPauseMsg toggles pause for a specific deck type.
type DeckPauseMsg struct {
	DeckTypeID string
}

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
			activeSection:  SectionDecks,
			activeDeckIdx: 0,
		},
		LogViewState: LogViewState{
			logEntries:    make([]model.LogRecord, 0, maxLogBuffer),
			logAutoScroll: true,
			showColumns:   true,
		},

		updateInterval:     updateInterval,
		reverseScrollWheel: reverseScrollWheel,
		useLogTime:         useLogTime,
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
		lastTickOK:  true,
		lastTickAt:  time.Now(),
		deckStates: make(map[string]*DeckTypeState),
		store:       store,
		dataSource:  dataSource,
	}

	m.SetViews(DefaultViewSpecs())

	// Register inline handlers for filter/search input (NOT modals).
	m.inlineHandlers = []inlineHandlerEntry{
		{isActive: func(m *DashboardModel) bool { return m.filterActive }, handler: filterInputHandler{}},
		{isActive: func(m *DashboardModel) bool { return m.searchActive }, handler: searchInputHandler{}},
	}

	return m
}

// SetDecks replaces decks and resets deck selection state.
func (m *DashboardModel) SetDecks(panels []Deck) {
	if len(panels) == 0 {
		m.decks = nil
		m.deckSelIdx = nil
		m.activeDeckIdx = 0
		m.persistActiveViewState()
		return
	}

	m.decks = append([]Deck(nil), panels...)
	m.deckSelIdx = make([]int, len(m.decks))
	if m.activeDeckIdx >= len(m.decks) {
		m.activeDeckIdx = 0
	}
	m.persistActiveViewState()
}

// SetViews configures right-side views and activates the first one.
func (m *DashboardModel) SetViews(specs []ViewSpec) {
	deps := DeckDeps{
		Store:             m.store,
		Drain3Manager:     m.drain3Manager,
		PushCountsModal:   m.pushCountsModalCmd(),
		PushPatternsModal: m.pushPatternsModalCmd(),
		FormatAttrModal:   m.formatAttributeValuesModal,
		PushContentModal:  m.pushContentModalCmd(),
	}

	pages := make([]ViewState, 0, len(specs))
	for _, spec := range specs {
		if spec.Build == nil {
			continue
		}
		panels := spec.Build(deps)
		state := ViewState{
			ID:             spec.ID,
			Title:          spec.Title,
			Decks:         append([]Deck(nil), panels...),
			DeckSelIdx:    make([]int, len(panels)),
			ActiveDeckIdx: 0,
		}
		pages = append(pages, state)
	}

	if len(pages) == 0 {
		m.views = nil
		m.decks = nil
		m.deckSelIdx = nil
		m.activeDeckIdx = 0
		m.activeViewIdx = 0
		m.sidebarCursor = 0
		return
	}

	m.views = pages
	m.activeViewIdx = -1
	m.sidebarCursor = 0
	m.activateView(0)
}

// DefaultViewSpecs declares built-in views and their decks.
func DefaultViewSpecs() []ViewSpec {
	return []ViewSpec{
		{
			ID:    "overview",
			Title: "Overview",
			Build: func(deps DeckDeps) []Deck {
				return []Deck{
					NewWordsDeck(),
					NewAttributesDeck(deps.Store, deps.FormatAttrModal, deps.PushContentModal),
					NewPatternsDeck(deps.Drain3Manager, deps.PushPatternsModal),
					NewCountsDeck(deps.PushCountsModal),
				}
			},
		},
		{
			ID:    "patterns",
			Title: "Patterns",
			Build: func(deps DeckDeps) []Deck {
				return []Deck{
					NewPatternsDeck(deps.Drain3Manager, deps.PushPatternsModal),
					NewCountsDeck(deps.PushCountsModal),
					NewWordsDeck(),
				}
			},
		},
		{
			ID:    "attributes",
			Title: "Attributes",
			Build: func(deps DeckDeps) []Deck {
				return []Deck{
					NewAttributesDeck(deps.Store, deps.FormatAttrModal, deps.PushContentModal),
					NewWordsDeck(),
					NewCountsDeck(deps.PushCountsModal),
				}
			},
		},
	}
}

// pushCountsModalCmd returns a tea.Cmd that pushes the counts modal.
func (m *DashboardModel) pushCountsModalCmd() tea.Cmd {
	return func() tea.Msg {
		modal := NewCountsModal(m)
		return ActionMsg{Action: ActionPushModal, Payload: modal}
	}
}

// pushPatternsModalCmd returns a tea.Cmd that pushes the patterns modal.
func (m *DashboardModel) pushPatternsModalCmd() tea.Cmd {
	return func() tea.Msg {
		modal := NewPatternsModal(m)
		return ActionMsg{Action: ActionPushModal, Payload: modal}
	}
}

// pushContentModalCmd returns a function that creates a tea.Cmd to push a detail modal with content.
func (m *DashboardModel) pushContentModalCmd() func(content string) tea.Cmd {
	return func(content string) tea.Cmd {
		return func() tea.Msg {
			modal := NewDetailModalWithContent(m, content)
			return ActionMsg{Action: ActionPushModal, Payload: modal}
		}
	}
}

func (m *DashboardModel) persistActiveViewState() {
	if len(m.views) == 0 || m.activeViewIdx < 0 || m.activeViewIdx >= len(m.views) {
		return
	}

	vw := &m.views[m.activeViewIdx]
	vw.Decks = append([]Deck(nil), m.decks...)
	vw.DeckSelIdx = append([]int(nil), m.deckSelIdx...)
	vw.ActiveDeckIdx = m.activeDeckIdx
}

func (m *DashboardModel) activateView(idx int) {
	if len(m.views) == 0 || idx < 0 || idx >= len(m.views) {
		return
	}

	if idx != m.activeViewIdx || len(m.decks) > 0 || len(m.deckSelIdx) > 0 {
		m.persistActiveViewState()
	}
	m.activeViewIdx = idx

	vw := &m.views[m.activeViewIdx]
	if len(vw.DeckSelIdx) != len(vw.Decks) {
		vw.DeckSelIdx = make([]int, len(vw.Decks))
	}

	m.decks = append([]Deck(nil), vw.Decks...)
	m.deckSelIdx = append([]int(nil), vw.DeckSelIdx...)

	if len(m.decks) == 0 {
		m.activeDeckIdx = 0
		return
	}

	if vw.ActiveDeckIdx < 0 || vw.ActiveDeckIdx >= len(m.decks) {
		vw.ActiveDeckIdx = 0
	}
	m.activeDeckIdx = vw.ActiveDeckIdx
}

func (m *DashboardModel) nextView() {
	if len(m.views) <= 1 {
		return
	}
	m.activateView((m.activeViewIdx + 1) % len(m.views))
}

func (m *DashboardModel) prevView() {
	if len(m.views) <= 1 {
		return
	}
	m.activateView((m.activeViewIdx - 1 + len(m.views)) % len(m.views))
}

func (m *DashboardModel) currentViewTitle() string {
	if len(m.views) == 0 || m.activeViewIdx < 0 || m.activeViewIdx >= len(m.views) {
		return ""
	}
	return m.views[m.activeViewIdx].Title
}

// queryOpts returns the current QueryOpts based on selected app.
func (m *DashboardModel) queryOpts() model.QueryOpts {
	return model.QueryOpts{App: m.selectedApp}
}

// viewContext builds a ViewContext snapshot for deck rendering.
func (m *DashboardModel) viewContext() ViewContext {
	return ViewContext{
		ContentWidth:  m.contentWidth(),
		ContentHeight: m.height,
		SearchTerm:    m.searchTerm,
		SelectedApp:   m.selectedApp,
		UseLogTime:    m.useLogTime,
	}
}

// DashboardView adapts DashboardModel to the Page interface.
type DashboardView struct {
	Model *DashboardModel
}

// NewDashboardView wraps a DashboardModel as a Page.
func NewDashboardView(m *DashboardModel) *DashboardView {
	return &DashboardView{Model: m}
}

func (p *DashboardView) ID() string { return "dashboard" }

func (p *DashboardView) Init() tea.Cmd {
	return p.Model.Init()
}

func (p *DashboardView) Update(msg tea.Msg) (tea.Cmd, *ViewNav) {
	_, cmd := p.Model.Update(msg)
	return cmd, nil // no navigation yet
}

func (p *DashboardView) View(width, height int) string {
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

	// Set up regular tick for dashboard updates (core tick)
	cmds = append(cmds, tea.Tick(m.updateInterval, func(t time.Time) tea.Msg {
		return TickMsg(t)
	}))

	// Start independent deck ticks
	cmds = append(cmds, m.registerDeckTicks()...)

	return tea.Batch(cmds...)
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

// autoPauseLiveUpdates returns true when the user is in log-reading context.
// This prevents incoming refreshes from shifting the selection while reading.
func (m *DashboardModel) autoPauseLiveUpdates() bool {
	return m.activeSection == SectionLogs || m.isLogViewerOpen()
}

// liveUpdatesPaused returns true when refreshes should be skipped.
func (m *DashboardModel) liveUpdatesPaused() bool {
	return m.viewPaused || m.autoPauseLiveUpdates()
}
