package pixelclient

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// View represents the current UI state
type View int

const (
	ViewSelection View = iota
	ViewConnecting
	ViewDisplay
	ViewCreateDevice
)

// InputField tracks which field is being edited in create device view
type InputField int

const (
	FieldName InputField = iota
	FieldServer
)

// Model is the Bubbletea model for the application
type Model struct {
	// Configuration
	config *Config

	// Current view state
	view View

	// Device selection state
	selectedIdx   int
	showCreateNew bool

	// Create device state
	inputField   InputField
	inputName    string
	inputServer  string
	createError  string

	// Display state
	device      *Device
	client      *Client
	renderer    *Renderer
	currentGrid string
	status      string
	connected   bool

	// Animation state
	animFrames   []AnimationFrame
	animIndex    int
	animating    bool
	animGen      int // generation counter to invalidate old ticks

	// Terminal dimensions
	width  int
	height int

	// Context for cancellation
	ctx    context.Context
	cancel context.CancelFunc

	// Error state
	err error
}

// NewModel creates a new Model with the given config
func NewModel(config *Config, initialDevice *Device) Model {
	ctx, cancel := context.WithCancel(context.Background())

	m := Model{
		config:   config,
		view:     ViewSelection,
		renderer: NewRenderer(80, 24),
		ctx:      ctx,
		cancel:   cancel,
	}

	if initialDevice != nil {
		m.device = initialDevice
		m.view = ViewConnecting
	} else if len(config.Devices) == 0 {
		m.view = ViewCreateDevice
	}

	return m
}

// Messages for async operations
type frameMsg struct {
	data []byte
}

type connectedMsg struct {
	client *Client
}

type connectionErrorMsg struct {
	err error
}

type tickMsg struct{}

type animTickMsg struct {
	gen int
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	if m.view == ViewConnecting {
		return m.connectCmd()
	}
	return nil
}

// connectCmd starts connecting to the device
func (m Model) connectCmd() tea.Cmd {
	device := m.device
	ctx := m.ctx
	return func() tea.Msg {
		client := NewClient(device)
		if err := client.Connect(ctx); err != nil {
			return connectionErrorMsg{err: err}
		}
		return connectedMsg{client: client}
	}
}

// animTickCmd schedules the next animation frame after the given delay
func animTickCmd(delay time.Duration, gen int) tea.Cmd {
	return tea.Tick(delay, func(time.Time) tea.Msg {
		return animTickMsg{gen: gen}
	})
}

// waitForFrameCmd waits for the next frame from the client
func (m Model) waitForFrameCmd() tea.Cmd {
	client := m.client
	ctx := m.ctx
	return func() tea.Msg {
		if client == nil {
			return connectionErrorMsg{err: fmt.Errorf("no client")}
		}
		select {
		case data := <-client.Frames():
			return frameMsg{data: data}
		case err := <-client.Errors():
			return connectionErrorMsg{err: err}
		case <-client.Done():
			return connectionErrorMsg{err: fmt.Errorf("connection closed")}
		case <-ctx.Done():
			return nil
		}
	}
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.renderer.UpdateSize(msg.Width, msg.Height)
		return m, nil

	case connectedMsg:
		m.client = msg.client
		m.connected = true
		m.status = fmt.Sprintf("Connected to %s @ %s", m.device.Name, m.device.Server)
		m.view = ViewDisplay
		m.currentGrid = m.renderer.RenderPlaceholder()
		return m, m.waitForFrameCmd()

	case connectionErrorMsg:
		m.connected = false
		m.status = fmt.Sprintf("Connection error: %v - reconnecting...", msg.err)
		// Attempt reconnection
		return m, m.reconnectCmd()

	case frameMsg:
		frames, err := m.renderer.DecodeAnimation(msg.data)
		if err != nil {
			m.status = fmt.Sprintf("Decode error: %v", err)
			return m, m.waitForFrameCmd()
		}
		if len(frames) == 0 {
			m.status = "No frames decoded"
			return m, m.waitForFrameCmd()
		}

		// Increment generation to invalidate any pending animation ticks
		m.animGen++
		m.animFrames = frames
		m.animIndex = 0
		m.currentGrid = frames[0].Grid

		// If animated, start the animation loop
		if len(frames) > 1 {
			m.animating = true
			return m, tea.Batch(m.waitForFrameCmd(), animTickCmd(frames[0].Delay, m.animGen))
		}
		m.animating = false
		return m, m.waitForFrameCmd()

	case animTickMsg:
		// Ignore stale ticks from previous animations
		if msg.gen != m.animGen || !m.animating || len(m.animFrames) <= 1 {
			return m, nil
		}
		// Advance to next frame
		m.animIndex = (m.animIndex + 1) % len(m.animFrames)
		m.currentGrid = m.animFrames[m.animIndex].Grid
		// Schedule next tick
		return m, animTickCmd(m.animFrames[m.animIndex].Delay, m.animGen)
	}

	return m, nil
}

// reconnectCmd attempts to reconnect
func (m Model) reconnectCmd() tea.Cmd {
	oldClient := m.client
	device := m.device
	ctx := m.ctx
	return func() tea.Msg {
		if oldClient != nil {
			oldClient.Close()
		}
		client := NewClient(device)
		if err := client.Reconnect(ctx); err != nil {
			return connectionErrorMsg{err: err}
		}
		return connectedMsg{client: client}
	}
}

// handleKey processes keyboard input
func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.view {
	case ViewSelection:
		return m.handleSelectionKey(msg)
	case ViewDisplay:
		return m.handleDisplayKey(msg)
	case ViewCreateDevice:
		return m.handleCreateDeviceKey(msg)
	case ViewConnecting:
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			m.cancel()
			return m, tea.Quit
		}
	}
	return m, nil
}

// handleSelectionKey handles keys in device selection view
func (m Model) handleSelectionKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	totalItems := len(m.config.Devices) + 1 // +1 for "Create new"

	switch msg.String() {
	case "q", "ctrl+c":
		m.cancel()
		return m, tea.Quit

	case "up", "k":
		if m.selectedIdx > 0 {
			m.selectedIdx--
		}

	case "down", "j":
		if m.selectedIdx < totalItems-1 {
			m.selectedIdx++
		}

	case "enter":
		if m.selectedIdx < len(m.config.Devices) {
			// Selected a device
			m.device = &m.config.Devices[m.selectedIdx]
			m.view = ViewConnecting
			m.status = fmt.Sprintf("Connecting to %s...", m.device.Server)
			return m, m.connectCmd()
		} else {
			// Selected "Create new"
			m.view = ViewCreateDevice
			m.inputField = FieldName
			m.inputName = ""
			m.inputServer = ""
			m.createError = ""
		}
	}

	return m, nil
}

// handleDisplayKey handles keys in display view
func (m Model) handleDisplayKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		if m.client != nil {
			m.client.Close()
		}
		m.cancel()
		return m, tea.Quit

	case "d":
		// Switch to device menu
		if m.client != nil {
			m.client.Close()
		}
		m.view = ViewSelection
		m.connected = false
		m.currentGrid = ""

	case "r":
		// Force reconnect
		m.status = "Reconnecting..."
		return m, m.reconnectCmd()
	}

	return m, nil
}

// handleCreateDeviceKey handles keys in create device view
func (m Model) handleCreateDeviceKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		m.cancel()
		return m, tea.Quit

	case "esc":
		// Go back to selection (if there are devices) or quit
		if len(m.config.Devices) > 0 {
			m.view = ViewSelection
		} else {
			return m, tea.Quit
		}

	case "tab", "down":
		if m.inputField == FieldName {
			m.inputField = FieldServer
		} else {
			m.inputField = FieldName
		}

	case "shift+tab", "up":
		if m.inputField == FieldServer {
			m.inputField = FieldName
		} else {
			m.inputField = FieldServer
		}

	case "enter":
		// Validate and create device
		if m.inputName == "" {
			m.createError = "Name is required"
			return m, nil
		}
		if m.inputServer == "" {
			m.createError = "Server is required"
			return m, nil
		}

		device, err := m.config.AddDevice(m.inputName, m.inputServer, "")
		if err != nil {
			m.createError = err.Error()
			return m, nil
		}

		if err := m.config.Save(); err != nil {
			m.createError = fmt.Sprintf("Failed to save config: %v", err)
			return m, nil
		}

		m.device = device
		m.view = ViewConnecting
		m.status = fmt.Sprintf("Connecting to %s...", m.device.Server)
		return m, m.connectCmd()

	case "backspace":
		if m.inputField == FieldName && len(m.inputName) > 0 {
			m.inputName = m.inputName[:len(m.inputName)-1]
		} else if m.inputField == FieldServer && len(m.inputServer) > 0 {
			m.inputServer = m.inputServer[:len(m.inputServer)-1]
		}
		m.createError = ""

	default:
		// Add character to current field
		if len(msg.String()) == 1 {
			if m.inputField == FieldName {
				m.inputName += msg.String()
			} else {
				m.inputServer += msg.String()
			}
			m.createError = ""
		}
	}

	return m, nil
}

// View renders the UI
func (m Model) View() string {
	switch m.view {
	case ViewSelection:
		return m.viewSelection()
	case ViewConnecting:
		return m.viewConnecting()
	case ViewDisplay:
		return m.viewDisplay()
	case ViewCreateDevice:
		return m.viewCreateDevice()
	}
	return ""
}

// Styles
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205"))

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("212"))

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))

	inputActiveStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("212")).
				Bold(true)

	inputInactiveStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("241"))
)

// viewSelection renders the device selection view
func (m Model) viewSelection() string {
	var sb strings.Builder

	sb.WriteString(titleStyle.Render("Select a device:"))
	sb.WriteString("\n\n")

	for i, device := range m.config.Devices {
		cursor := "  "
		style := dimStyle
		if i == m.selectedIdx {
			cursor = "> "
			style = selectedStyle
		}
		line := fmt.Sprintf("%s (%s)", device.Name, device.Server)
		sb.WriteString(cursor + style.Render(line) + "\n")
	}

	// "Create new" option
	createIdx := len(m.config.Devices)
	cursor := "  "
	style := dimStyle
	if m.selectedIdx == createIdx {
		cursor = "> "
		style = selectedStyle
	}
	sb.WriteString(cursor + style.Render("[+] Create new device") + "\n")

	sb.WriteString("\n")
	sb.WriteString(dimStyle.Render("↑/↓ navigate • enter select • q quit"))

	return sb.String()
}

// viewConnecting renders the connecting view
func (m Model) viewConnecting() string {
	var sb strings.Builder

	sb.WriteString(titleStyle.Render("Connecting..."))
	sb.WriteString("\n\n")
	sb.WriteString(fmt.Sprintf("Device: %s\n", m.device.Name))
	sb.WriteString(fmt.Sprintf("Server: %s\n", m.device.Server))
	sb.WriteString("\n")
	sb.WriteString(dimStyle.Render("q to cancel"))

	return sb.String()
}

// viewDisplay renders the LED display view
func (m Model) viewDisplay() string {
	var sb strings.Builder

	// Calculate vertical padding
	gridHeight := 32 // Fixed height for the LED matrix
	statusHeight := 1
	availableHeight := m.height - statusHeight
	topPadding := 0
	if availableHeight > gridHeight {
		topPadding = (availableHeight - gridHeight) / 2
	}

	// Add top padding
	for i := 0; i < topPadding; i++ {
		sb.WriteString("\n")
	}

	// Add the LED grid (or placeholder)
	if m.currentGrid != "" {
		sb.WriteString(m.currentGrid)
	} else {
		sb.WriteString(m.renderer.RenderPlaceholder())
	}

	// Pad to fill remaining space before status bar
	gridLines := gridHeight
	totalLines := topPadding + gridLines
	for i := totalLines; i < m.height-1; i++ {
		sb.WriteString("\n")
	}

	// Status bar
	sb.WriteString("\n")
	statusText := fmt.Sprintf("%s [%s] @ %s", m.device.Name, m.device.ID, m.device.Server)
	if !m.connected {
		statusText = "Disconnected - " + statusText
	}
	hotkeys := "[d]evice [r]econnect [q]uit"

	// Pad status bar
	padding := m.width - len(statusText) - len(hotkeys) - 3
	if padding < 1 {
		padding = 1
	}
	sb.WriteString(statusStyle.Render(statusText + strings.Repeat(" ", padding) + hotkeys))

	return sb.String()
}

// viewCreateDevice renders the create device view
func (m Model) viewCreateDevice() string {
	var sb strings.Builder

	sb.WriteString(titleStyle.Render("Create new device"))
	sb.WriteString("\n\n")

	// Name field
	nameLabel := "Name:   "
	nameStyle := inputInactiveStyle
	if m.inputField == FieldName {
		nameStyle = inputActiveStyle
		nameLabel = "> " + nameLabel
	} else {
		nameLabel = "  " + nameLabel
	}
	sb.WriteString(nameStyle.Render(nameLabel))
	sb.WriteString(m.inputName)
	if m.inputField == FieldName {
		sb.WriteString("_")
	}
	sb.WriteString("\n")

	// Server field
	serverLabel := "Server: "
	serverStyle := inputInactiveStyle
	if m.inputField == FieldServer {
		serverStyle = inputActiveStyle
		serverLabel = "> " + serverLabel
	} else {
		serverLabel = "  " + serverLabel
	}
	sb.WriteString(serverStyle.Render(serverLabel))
	sb.WriteString(m.inputServer)
	if m.inputField == FieldServer {
		sb.WriteString("_")
	}
	sb.WriteString("\n")

	// Error message
	if m.createError != "" {
		sb.WriteString("\n")
		sb.WriteString(errorStyle.Render("Error: " + m.createError))
	}

	sb.WriteString("\n\n")
	sb.WriteString(dimStyle.Render("tab/↑/↓ switch fields • enter create • esc back"))

	return sb.String()
}
