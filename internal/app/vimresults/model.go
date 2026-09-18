package vimresults

import (
	"image/color"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/mkhamat/gopherttype/internal/app/mascot"
	"github.com/mkhamat/gopherttype/internal/app/ui"
	"github.com/mkhamat/gopherttype/internal/vim"
)

// RetryMsg asks the app to start a fresh round with the same settings.
type RetryMsg struct{}

// HomeMsg asks the app to return to the home screen.
type HomeMsg struct{}

// paceToWPM scales solved-per-minute into the mascot's WPM band
const paceToWPM = 20.0

type Model struct {
	report        vim.Report
	styles        *ui.Styles
	width, height int

	mascot     *mascot.Model
	tier       mascot.Tier
	background color.Color
	content    string
	layout     ui.MascotLayout
	view       string
}

func New(report vim.Report) *Model {
	m := &Model{report: report, styles: ui.StylesFor(true), mascot: mascot.New()}
	m.setResult(time.Now())
	m.refreshContent()
	m.rebuild()
	return m
}

// setResult classifies the round once and locks the mascot expression
func (m *Model) setResult(at time.Time) {
	wpm, accuracy := m.analogs()
	m.tier = mascot.ClassifyTier(wpm, accuracy, m.report.Duration)
	m.mascot.SetResult(mascot.ClassifyResult(wpm, accuracy, m.report.Duration), at)
}

func (m *Model) analogs() (wpm, accuracy float64) {
	return m.report.Pace() * paceToWPM, m.report.Efficiency()
}

func (m *Model) Init() tea.Cmd { return m.configure(time.Now()) }

func (m *Model) View() string { return m.view }

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case mascot.FrameMsg:
		changed, cmd := m.mascot.Update(msg, time.Now())
		if changed {
			m.rebuild()
		}
		return cmd
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tea.BackgroundColorMsg:
		m.background = msg.Color
		m.styles = ui.StylesFor(msg.IsDark())
	case tea.KeyPressMsg:
		switch msg.String() {
		case "enter":
			return func() tea.Msg { return RetryMsg{} }
		case "tab":
			return func() tea.Msg { return HomeMsg{} }
		case "q":
			return tea.Quit
		}
		return nil
	}
	return m.configure(time.Now())
}

func (m *Model) configure(at time.Time) tea.Cmd {
	m.refreshContent()
	_, cmd := m.mascot.Configure(m.scene(), at)
	m.rebuild()
	return cmd
}

func (m *Model) refreshContent() {
	width, height := ui.TerminalSize(m.width, m.height)
	if width < ui.MinimumWidth || height < ui.MinimumHeight {
		m.content, m.layout = "", ui.MascotLayout{}
		return
	}
	m.content = m.renderContent()
	m.layout = ui.LayoutWithMascot(width, height, m.content)
}

func (m *Model) rebuild() {
	width, height := ui.TerminalSize(m.width, m.height)
	if width < ui.MinimumWidth || height < ui.MinimumHeight {
		m.view = ui.ResizeView(width, height, "q / Ctrl+C quit")
		return
	}
	m.view = ui.ComposeWithMascot(m.content, m.mascot.View(), m.layout, width, height)
}

func (m *Model) scene() mascot.Scene {
	background := m.background
	if background == nil {
		background = m.styles.Background
	}
	return mascot.Scene{
		Slot:       m.layout.Mascot,
		Content:    m.layout.Content,
		Track:      false,
		Background: background,
	}
}
