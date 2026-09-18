package home

import (
	"image/color"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/mkhamat/gopherttype/internal/app/mascot"
	"github.com/mkhamat/gopherttype/internal/app/ui"
	"github.com/mkhamat/gopherttype/internal/engine"
	"github.com/mkhamat/gopherttype/internal/vim"
)

var (
	wordPresets     = [...]int{10, 25, 50, 100}
	durationPresets = [...]time.Duration{15 * time.Second, 30 * time.Second, 60 * time.Second, 120 * time.Second}
	vimPresets      = [...]vim.Difficulty{vim.Easy, vim.Medium, vim.Hard, vim.Mixed}
)

// mode is the home selection axis
type mode int

const (
	modeTime mode = iota
	modeWords
	modeVim
)

var modeOrder = [...]mode{modeTime, modeWords, modeVim}

type settings struct {
	mode          mode
	wordIndex     int
	durationIndex int
	vimIndex      int
}

func (s settings) config() engine.Config {
	if s.mode == modeWords {
		return engine.Config{Mode: engine.ModeWords, WordCount: wordPresets[s.wordIndex]}
	}
	return engine.Config{Mode: engine.ModeTime, Duration: durationPresets[s.durationIndex]}
}

func (s settings) difficulty() vim.Difficulty {
	return vimPresets[s.vimIndex]
}

// stepMode moves the selected mode by delta, wrapping through the mode list.
func (m *Model) stepMode(delta int) {
	index := wrapIndex(int(m.settings.mode)+delta, len(modeOrder))
	m.settings.mode = modeOrder[index]
}

func (m *Model) stepLength(delta int) {
	switch m.settings.mode {
	case modeTime:
		m.settings.durationIndex = wrapIndex(m.settings.durationIndex+delta, len(durationPresets))
	case modeWords:
		m.settings.wordIndex = wrapIndex(m.settings.wordIndex+delta, len(wordPresets))
	case modeVim:
		m.settings.vimIndex = wrapIndex(m.settings.vimIndex+delta, len(vimPresets))
	}
}

func wrapIndex(index, length int) int {
	return ((index % length) + length) % length
}

// StartMsg begins a typing round.
type StartMsg struct{ Config engine.Config }

// StartVimMsg begins a Vim challenge round.
type StartVimMsg struct{ Difficulty vim.Difficulty }

type Model struct {
	settings      settings
	styles        *ui.Styles
	width, height int

	mascot     *mascot.Model
	background color.Color
	content    string
	selection  ui.Point
	layout     ui.MascotLayout
	view       string
}

func New() *Model {
	m := &Model{
		settings: settings{mode: modeTime},
		styles:   ui.StylesFor(true),
		mascot:   mascot.New(),
	}
	m.refreshContent()
	m.rebuild()
	return m
}

func (m *Model) Init() tea.Cmd {
	return m.configure(time.Now())
}

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
		case "q":
			return tea.Quit
		case "enter":
			if m.settings.mode == modeVim {
				difficulty := m.settings.difficulty()
				return func() tea.Msg { return StartVimMsg{Difficulty: difficulty} }
			}
			config := m.settings.config()
			return func() tea.Msg { return StartMsg{Config: config} }
		case "up", "k":
			m.stepMode(-1)
		case "down", "j":
			m.stepMode(1)
		case "left", "h":
			m.stepLength(-1)
		case "right", "l":
			m.stepLength(1)
		}
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
		m.content, m.selection, m.layout = "", ui.Point{}, ui.MascotLayout{}
		return
	}
	m.content, m.selection = m.renderContent()
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
	point, track := m.selectedPoint()
	return mascot.Scene{
		Slot:       m.layout.Mascot,
		Content:    m.layout.Content,
		Target:     point,
		Track:      track,
		Background: background,
	}
}
