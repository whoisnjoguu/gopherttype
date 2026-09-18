package vimplay

import (
	"image/color"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/mkhamat/gopherttype/internal/app/mascot"
	"github.com/mkhamat/gopherttype/internal/app/ui"
	"github.com/mkhamat/gopherttype/internal/vim"
)

// FinishedMsg is emitted once every challenge in the round is resolved
type FinishedMsg struct{ Report vim.Report }

// HomeMsg asks the app to return to the home screen
type HomeMsg struct{}

// settleMsg is delivered after a keystroke left the engine mid-command
type settleMsg struct{}

// settleCmd waits for the engine to finish applying the last keystroke to re-grade
func settleCmd(session *vim.Session) tea.Cmd {
	return func() tea.Msg {
		session.WaitSettle()
		return settleMsg{}
	}
}

// installDoneMsg reports the outcome of a background Neovim install attempt.
type installDoneMsg struct{ err error }

// installCmd runs an install plan off the UI loop and reports when it's done.
func installCmd(plan vim.InstallPlan) tea.Cmd {
	return func() tea.Msg {
		return installDoneMsg{err: plan.Run()}
	}
}

type Model struct {
	session       *vim.Session
	difficulty    vim.Difficulty
	challenges    []vim.Challenge
	styles        *ui.Styles
	width, height int

	mascot     *mascot.Model
	background color.Color
	showHint   bool
	content    string
	layout     ui.MascotLayout
	view       string

	nvimErr     error
	installPlan vim.InstallPlan
	installing  bool
	installMsg  string
}

// New starts a round over challenges
func New(difficulty vim.Difficulty, challenges []vim.Challenge) *Model {
	m := &Model{
		difficulty: difficulty,
		challenges: challenges,
		styles:     ui.StylesFor(true),
		mascot:     mascot.New(),
	}
	m.startSession()
	m.refreshContent()
	m.rebuild()
	return m
}

// startSession opens a live engine for the round
func (m *Model) startSession() {
	session, err := vim.NewSession(m.challenges, time.Now())
	if err != nil {
		m.session = nil
		m.nvimErr = err
		m.installPlan, _ = vim.DetectInstallPlan()
		return
	}
	m.session = session
	m.nvimErr = nil
	m.installMsg = ""
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
		return m.configure(time.Now())
	case tea.BackgroundColorMsg:
		m.styles = ui.StylesFor(msg.IsDark())
		m.background = msg.Color
		return m.configure(time.Now())
	case tea.KeyPressMsg:
		if m.nvimErr != nil {
			return m.handleInstallKey(msg, time.Now())
		}
		return m.handleKey(msg, time.Now())
	case settleMsg:
		return m.settle(time.Now())
	case installDoneMsg:
		return m.handleInstallDone(msg)
	}
	return nil
}

// handleInstallKey drives the "Neovim required" screen
func (m *Model) handleInstallKey(key tea.KeyPressMsg, at time.Time) tea.Cmd {
	switch key.String() {
	case "ctrl+b":
		return func() tea.Msg { return HomeMsg{} }
	case "i":
		if m.installing || !m.installPlan.AutoRun {
			return nil
		}
		m.installing = true
		m.installMsg = ""
		m.refreshContent()
		m.rebuild()
		return installCmd(m.installPlan)
	case "r":
		m.startSession()
		m.refreshContent()
		m.rebuild()
	}
	return nil
}

// handleInstallDone applies the result of a background install attempt and
// retries starting the session on success
func (m *Model) handleInstallDone(msg installDoneMsg) tea.Cmd {
	m.installing = false
	if msg.err != nil {
		m.installMsg = "install failed: " + msg.err.Error()
		m.refreshContent()
		m.rebuild()
		return nil
	}
	m.startSession()
	m.refreshContent()
	m.rebuild()
	return nil
}

// handleKey maps a raw key to either a game control or one or more vim keystroke tokens
func (m *Model) handleKey(key tea.KeyPressMsg, at time.Time) tea.Cmd {
	switch key.String() {
	case "ctrl+b":
		return func() tea.Msg { return HomeMsg{} }
	case "ctrl+g":
		m.showHint = !m.showHint
		return m.configure(at)
	case "ctrl+n":
		m.mascot.Activity(at)
		m.session.Skip(at)
		return m.finishOrConfigure(at)
	case "backspace":
		return m.feed("<BS>", at)
	case "esc":
		return m.feed("<Esc>", at)
	case "enter":
		return m.feed("<CR>", at)
	case "space":
		return m.feed(" ", at)
	default:
		if key.Text == "" {
			return nil
		}
		var cmd tea.Cmd
		for _, r := range key.Text {
			cmd = m.feed(string(r), at)
			if m.session.Status() == vim.Finished {
				break
			}
		}
		return cmd
	}
}

// feed submits one canonical token, updates the mascot reaction, and finishes or reconfigures the scene
func (m *Model) feed(token string, at time.Time) tea.Cmd {
	outcome := m.session.Type(token, at)
	m.mascot.ObserveAttempt(mascot.Attempt{At: at, Correct: outcome.Accepted, Pace: true})
	if outcome.Solved {
		m.showHint = false
	}
	cmd := m.finishOrConfigure(at)
	if outcome.Pending {
		return tea.Batch(cmd, settleCmd(m.session))
	}
	return cmd
}

// settle applies the engine's now-finished state after a pending keystroke
func (m *Model) settle(at time.Time) tea.Cmd {
	outcome := m.session.Regrade(at)
	if outcome.Solved {
		m.showHint = false
	}
	return m.finishOrConfigure(at)
}

func (m *Model) finishOrConfigure(at time.Time) tea.Cmd {
	switch m.session.Status() {
	case vim.Finished:
		report := m.session.Report()
		return func() tea.Msg { return FinishedMsg{Report: report} }
	case vim.Errored:
		m.nvimErr = m.session.Err()
		m.installPlan, _ = vim.DetectInstallPlan()
		m.refreshContent()
		m.rebuild()
		return nil
	}
	return m.configure(at)
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
		m.view = ui.ResizeView(width, height, "Ctrl+B home · Ctrl+C quit")
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
