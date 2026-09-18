package vimresults

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/mkhamat/gopherttype/internal/app/mascot"
	"github.com/mkhamat/gopherttype/internal/app/ui"
)

const fieldSep = "  ·  "

func (m *Model) renderContent() string {
	width, _ := ui.TerminalSize(m.width, m.height)
	padding := min(4, (width-24)/2)
	inner := min(80, width-2*padding)

	lines := []string{
		centerBlock(m.tierStyle().Render(m.tier.Label()), inner),
		"",
		m.renderHero(inner),
	}
	if detail := m.renderDetail(inner); detail != "" {
		lines = append(lines, detail)
	}
	lines = append(lines, "", m.renderHelp(inner))
	return strings.Join(lines, "\n")
}

func (m *Model) tierStyle() lipgloss.Style {
	switch m.tier {
	case mascot.TierCelebrate, mascot.TierProud:
		return m.styles.Accent
	case mascot.TierWorried:
		return m.styles.Warning
	default:
		return m.styles.Text
	}
}

func (m *Model) renderHero(inner int) string {
	score := m.styles.Highlight().Render(fmt.Sprintf(" %d pts ", m.report.Score))
	solved := m.tierStyle().Render(fmt.Sprintf("%d/%d", m.report.Solved, m.report.Total)) +
		m.styles.Muted.Render(" solved")
	efficiency := m.tierStyle().Render(fmt.Sprintf("%.0f%%", m.report.Efficiency())) +
		m.styles.Muted.Render(" efficiency")
	return m.centerClauses([]string{score, solved, efficiency}, inner)
}

func (m *Model) renderDetail(inner int) string {
	var clauses []string
	if m.report.Solved > 0 {
		clauses = append(clauses, m.styles.Muted.Render(
			fmt.Sprintf("%d keys · par %d", m.report.TotalUsed, m.report.TotalPar)))
	}
	if m.report.TotalMistakes > 0 {
		clauses = append(clauses, m.styles.Incorrect.Render(
			fmt.Sprintf("%d wrong keys", m.report.TotalMistakes)))
	}
	clauses = append(clauses, m.styles.Muted.Render(
		fmt.Sprintf("%.0fs", m.report.Duration.Seconds())))
	return m.centerClauses(clauses, inner)
}

func (m *Model) renderHelp(inner int) string {
	action := func(k, label string) string {
		return m.styles.Accent.Render(k) + m.styles.Muted.Render(" "+label)
	}
	return m.centerClauses([]string{action("enter", "retry"), action("tab", "home"), action("q", "quit")}, inner)
}

func (m *Model) centerClauses(clauses []string, inner int) string {
	if len(clauses) == 0 {
		return ""
	}
	sep := m.styles.Muted.Render(fieldSep)
	var lines []string
	current := clauses[0]
	for _, clause := range clauses[1:] {
		if joined := current + sep + clause; lipgloss.Width(joined) <= inner {
			current = joined
		} else {
			lines = append(lines, current)
			current = clause
		}
	}
	return centerBlock(strings.Join(append(lines, current), "\n"), inner)
}

func centerBlock(block string, inner int) string {
	lines := strings.Split(block, "\n")
	for i, line := range lines {
		lines[i] = lipgloss.PlaceHorizontal(inner, lipgloss.Center, line)
	}
	return strings.Join(lines, "\n")
}
