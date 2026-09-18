package app

import (
	"math/rand/v2"

	tea "charm.land/bubbletea/v2"

	"github.com/mkhamat/gopherttype/internal/app/home"
	"github.com/mkhamat/gopherttype/internal/app/play"
	"github.com/mkhamat/gopherttype/internal/app/results"
	"github.com/mkhamat/gopherttype/internal/app/vimplay"
	"github.com/mkhamat/gopherttype/internal/app/vimresults"
	"github.com/mkhamat/gopherttype/internal/engine"
	"github.com/mkhamat/gopherttype/internal/vim"
	"github.com/mkhamat/gopherttype/internal/words"
)

type screen interface {
	Init() tea.Cmd
	Update(tea.Msg) tea.Cmd
	View() string
}

type Model struct {
	active        screen
	roundConfig   engine.Config
	vimDifficulty vim.Difficulty
	generator     *words.Generator
	rng           *rand.Rand
	size          tea.WindowSizeMsg
	background    *tea.BackgroundColorMsg
}

func New() *Model {
	return &Model{
		active:    home.New(),
		generator: words.New(rand.Uint64(), rand.Uint64()),
		rng:       rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64())),
	}
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(m.active.Init(), tea.RequestBackgroundColor)
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.size = msg
	case tea.BackgroundColorMsg:
		m.background = &msg
	case home.StartMsg:
		m.roundConfig = msg.Config
		return m, m.switchScreen(play.New(m.roundConfig, m.generator.Generate))
	case home.StartVimMsg:
		m.vimDifficulty = msg.Difficulty
		return m, m.switchScreen(vimplay.New(m.vimDifficulty, m.selectVim()))
	case play.FinishedMsg:
		return m, m.switchScreen(results.New(msg.Metrics))
	case vimplay.FinishedMsg:
		return m, m.switchScreen(vimresults.New(msg.Report))
	case play.HomeMsg, results.HomeMsg, vimplay.HomeMsg, vimresults.HomeMsg:
		return m, m.switchScreen(home.New())
	case results.RetryMsg:
		return m, m.switchScreen(play.New(m.roundConfig, m.generator.Generate))
	case vimresults.RetryMsg:
		return m, m.switchScreen(vimplay.New(m.vimDifficulty, m.selectVim()))
	}
	return m, m.active.Update(msg)
}

// selectVim builds a fresh, procedurally generated round for the difficulty.
func (m *Model) selectVim() []vim.Challenge {
	return vim.Round(m.vimDifficulty, vim.RoundSize, m.rng)
}

func (m *Model) View() tea.View {
	view := tea.NewView(m.active.View())
	view.AltScreen = true
	return view
}

func (m *Model) switchScreen(next screen) tea.Cmd {
	m.active = next
	initCmd := m.active.Init()
	sizeCmd := m.active.Update(m.size)
	var backgroundCmd tea.Cmd
	if m.background != nil {
		backgroundCmd = m.active.Update(*m.background)
	}
	return tea.Batch(initCmd, sizeCmd, backgroundCmd)
}
