package vimplay

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/mkhamat/gopherttype/internal/app/ui"
	"github.com/mkhamat/gopherttype/internal/vim"
)

const (
	vimContentWidth = 60
	bufferGutter    = "│ "
)

func (m *Model) innerWidth() int {
	width, _ := ui.TerminalSize(m.width, m.height)
	padding := min(4, max(0, (width-24)/2))
	return min(vimContentWidth, max(1, width-2*padding))
}

func (m *Model) renderContent() string {
	if m.nvimErr != nil {
		return m.renderInstallScreen()
	}
	inner := m.innerWidth()
	challenge := m.session.Current()

	lines := []string{m.renderHeader(challenge)}
	lines = append(lines, "")
	lines = append(lines, wrap(m.styles.Text.Render(challenge.Task), inner)...)
	lines = append(lines, "")
	lines = append(lines, m.renderBuffer()...)
	lines = append(lines, "")
	lines = append(lines, m.renderStatus())
	if m.showHint {
		lines = append(lines, m.styles.Muted.Render("hint  ")+m.styles.Accent.Render(challenge.Hint()))
	}
	lines = append(lines, "")
	lines = append(lines, m.renderHelp())

	return strings.Join(center(lines), "\n")
}

func (m *Model) renderHeader(challenge vim.Challenge) string {
	sep := m.styles.Muted.Render("  ·  ")
	progress := m.styles.Accent.Render(fmt.Sprintf("Challenge %d/%d", m.session.Index()+1, m.session.Total()))
	difficulty := m.styles.Muted.Render(m.difficulty.Label())
	par := m.styles.Muted.Render(fmt.Sprintf("par %d", challenge.Par()))
	return progress + sep + difficulty + sep + par
}

// renderBuffer draws the live engine buffer with a muted gutter
func (m *Model) renderBuffer() []string {
	engine := m.session.Engine()
	gutter := m.styles.Muted.Render(bufferGutter)
	buffer := engine.Lines()
	cursor := engine.Cursor()
	span, active := engine.VisualSpan()

	lines := make([]string, 0, len(buffer))
	for li, line := range buffer {
		cursorCol := -1
		if li == cursor.Row {
			cursorCol = cursor.Col
		}
		lo, hi, sel := selectedRange(li, line, span, active)
		lines = append(lines, gutter+m.renderLine(line, cursorCol, lo, hi, sel))
	}
	return lines
}

// selectedRange returns the inclusive column range of the visual selection on row
func selectedRange(row int, line string, span vim.Span, active bool) (lo, hi int, ok bool) {
	if !active || row < span.Start.Row || row > span.End.Row {
		return 0, 0, false
	}
	last := len([]rune(line)) - 1
	if span.Linewise {
		return 0, last, true
	}
	lo = 0
	if row == span.Start.Row {
		lo = span.Start.Col
	}
	hi = last
	if row == span.End.Row {
		hi = span.End.Col
	}
	return lo, hi, true
}

// renderLine styles one buffer line
func (m *Model) renderLine(line string, cursorCol, lo, hi int, sel bool) string {
	runes := []rune(line)
	var b strings.Builder
	for i, r := range runes {
		switch {
		case i == cursorCol:
			b.WriteString(m.styles.Cursor.Render(string(r)))
		case sel && i >= lo && i <= hi:
			b.WriteString(m.styles.Selection.Render(string(r)))
		default:
			b.WriteString(m.styles.Text.Render(string(r)))
		}
	}
	// trailing cell: show the cursor past end-of-line
	switch {
	case cursorCol >= len(runes):
		b.WriteString(m.styles.Cursor.Render(" "))
	case sel && len(runes) == 0:
		b.WriteString(m.styles.Selection.Render(" "))
	}
	return b.String()
}

// renderStatus shows the current engine mode plus either the command-line being
// typed (in : / ? modes) or any pending normal-mode keystrokes.
func (m *Model) renderStatus() string {
	engine := m.session.Engine()
	label := engine.ModeLabel()
	if label == "" {
		label = "NORMAL"
	}
	badge := m.styles.Accent.Render("-- " + label + " --")

	if kind, text := engine.CommandLine(); kind != 0 {
		return badge + "  " + m.styles.Text.Render(string(kind)+text)
	}
	if pending := engine.PendingKeys(); pending != "" {
		return badge + "  " + m.styles.Muted.Render("pending ") + m.styles.Accent.Render(pending)
	}
	return badge
}

func (m *Model) renderHelp() string {
	action := func(k, label string) string {
		return m.styles.Accent.Render(k) + m.styles.Muted.Render(" "+label)
	}
	sep := m.styles.Muted.Render("  ·  ")
	return strings.Join([]string{
		action("ctrl+n", "skip"),
		action("ctrl+g", "hint"),
		action("ctrl+b", "home"),
	}, sep)
}

// renderInstallScreen replaces the game when a real embedded Neovim could not be started
func (m *Model) renderInstallScreen() string {
	inner := m.innerWidth()
	lines := []string{m.styles.Accent.Render("Neovim required")}
	lines = append(lines, "")
	lines = append(lines, wrap(m.styles.Text.Render(
		"This game plays through a real embedded Neovim so every keybinding behaves exactly like Vim. Install \"nvim\" to continue.",
	), inner)...)
	lines = append(lines, "")

	switch {
	case m.installing:
		lines = append(lines, m.styles.Muted.Render("installing via "+m.installPlan.Manager+" …"))
	case m.installMsg != "":
		lines = append(lines, wrap(m.styles.Muted.Render(m.installMsg), inner)...)
	case m.installPlan.Manager != "":
		lines = append(lines, m.styles.Muted.Render("detected: "+m.installPlan.Manager))
		lines = append(lines, wrap(m.styles.Accent.Render(strings.Join(m.installPlan.Command, " ")), inner)...)
		if !m.installPlan.AutoRun {
			lines = append(lines, wrap(m.styles.Muted.Render(
				"this needs elevated privileges — run it yourself in a terminal, then press r to retry",
			), inner)...)
		}
	default:
		lines = append(lines, wrap(m.styles.Muted.Render(
			"no supported package manager detected; install Neovim manually, then press r to retry",
		), inner)...)
	}
	lines = append(lines, "")

	action := func(k, label string) string {
		return m.styles.Accent.Render(k) + m.styles.Muted.Render(" "+label)
	}
	sep := m.styles.Muted.Render("  ·  ")
	var help []string
	if m.installPlan.AutoRun && !m.installing {
		help = append(help, action("i", "install"))
	}
	help = append(help, action("r", "retry"), action("ctrl+b", "home"))
	lines = append(lines, strings.Join(help, sep))

	return strings.Join(center(lines), "\n")
}

func wrap(s string, width int) []string {
	wrapped := lipgloss.NewStyle().Width(width).Render(s)
	return strings.Split(wrapped, "\n")
}

func center(lines []string) []string {
	width := 0
	for _, line := range lines {
		if w := lipgloss.Width(line); w > width {
			width = w
		}
	}
	for i, line := range lines {
		if lipgloss.Width(line) > 0 {
			lines[i] = lipgloss.PlaceHorizontal(width, lipgloss.Center, line)
		}
	}
	return lines
}
