package home

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/mkhamat/gopherttype/internal/app/mascot"
	"github.com/mkhamat/gopherttype/internal/app/ui"
	"github.com/mkhamat/gopherttype/internal/engine"
	"github.com/mkhamat/gopherttype/internal/vim"
)

func key(code rune) tea.KeyPressMsg {
	k := tea.Key{Code: code}
	if code >= ' ' && code <= '~' {
		k.Text = string(code)
	}
	return tea.KeyPressMsg(k)
}

func TestHomeVisibility(t *testing.T) {
	m := New()
	m.Init()
	if want := (ui.Rect{X: 24, Y: 1, Width: 32, Height: 12}); m.layout.Mascot != want {
		t.Errorf("default slot = %+v, want %+v", m.layout.Mascot, want)
	}
	if m.mascot.View() == "" || strings.Count(m.View(), "\n") != 23 {
		t.Error("default home must show the portrait in 24 rows")
	}
	m.Update(tea.WindowSizeMsg{Width: 40, Height: 20})
	if m.mascot.View() != "" || m.layout.Mascot != (ui.Rect{}) || m.View() != ui.FitView(m.content, 40, 20) {
		t.Error("short home must hide the portrait and reclaim its rows")
	}
	m.Update(tea.WindowSizeMsg{Width: 27, Height: 11})
	if m.View() != ui.ResizeView(27, 11, "q / Ctrl+C quit") {
		t.Error("below minimum must keep ResizeView")
	}
}

func TestHomeFrameBypassesSettings(t *testing.T) {
	m := New()
	cmd := m.Init()
	if cmd == nil {
		t.Fatal("Init did not arm the clock")
	}
	if m.Init() != nil {
		t.Error("repeated Init must not arm another chain")
	}
	frame, ok := cmd().(mascot.FrameMsg)
	if !ok {
		t.Fatal("armed command must return a mascot frame")
	}
	settings, content := m.settings, m.content
	if m.Update(frame) == nil {
		t.Error("valid frame must return its successor")
	}
	if m.settings != settings || m.content != content {
		t.Error("mascot frame must not change the menu")
	}
	if m.Update(frame) != nil {
		t.Error("stale frame must not reschedule")
	}
}

func TestHomeKeysChangeSelection(t *testing.T) {
	for _, keys := range [][4]rune{{tea.KeyUp, tea.KeyDown, tea.KeyLeft, tea.KeyRight}, {'k', 'j', 'h', 'l'}} {
		m := New()
		if m.settings.mode != modeTime {
			t.Fatal("default must be time mode")
		}
		down, left, right := keys[1], keys[2], keys[3]
		for _, tc := range []struct {
			key  rune
			want settings
		}{
			{down, settings{mode: modeWords}},
			{down, settings{mode: modeVim}},
			{down, settings{mode: modeTime}}, // wraps through the three modes
			{left, settings{mode: modeTime, durationIndex: 3}},
			{right, settings{mode: modeTime}},
		} {
			m.Update(key(tc.key))
			if m.settings != tc.want {
				t.Errorf("key %q: settings=%+v, want %+v", key(tc.key).String(), m.settings, tc.want)
			}
		}
	}
}

func TestHomeStartUsesIndependentLengths(t *testing.T) {
	m := New()
	for _, tc := range []struct {
		keys []rune
		want engine.Config
	}{
		{nil, engine.Config{Mode: engine.ModeTime, Duration: 15 * time.Second}},
		{[]rune{tea.KeyRight, tea.KeyRight}, engine.Config{Mode: engine.ModeTime, Duration: 60 * time.Second}},
		{[]rune{tea.KeyDown, tea.KeyRight}, engine.Config{Mode: engine.ModeWords, WordCount: 25}},
		{[]rune{tea.KeyDown, tea.KeyDown}, engine.Config{Mode: engine.ModeTime, Duration: 60 * time.Second}},
	} {
		for _, code := range tc.keys {
			m.Update(key(code))
		}
		cmd := m.Update(key(tea.KeyEnter))
		if cmd == nil {
			t.Fatal("Enter must return StartMsg")
		}
		msg, ok := cmd().(StartMsg)
		if !ok || msg.Config != tc.want {
			t.Errorf("start = %+v, want %+v", msg, tc.want)
		}
	}
}

func TestHomeStartsVimRound(t *testing.T) {
	m := New()
	m.Update(key(tea.KeyDown)) // time -> words
	m.Update(key(tea.KeyDown)) // words -> vim

	cmd := m.Update(key(tea.KeyEnter))
	if cmd == nil {
		t.Fatal("Enter in vim mode must return a command")
	}
	msg, ok := cmd().(StartVimMsg)
	if !ok {
		t.Fatalf("vim mode Enter = %T, want StartVimMsg", cmd())
	}
	if msg.Difficulty != vim.Easy {
		t.Errorf("default vim difficulty = %v, want easy", msg.Difficulty)
	}

	m.Update(key(tea.KeyRight)) // easy -> medium
	cmd = m.Update(key(tea.KeyEnter))
	if msg, ok := cmd().(StartVimMsg); !ok || msg.Difficulty != vim.Medium {
		t.Errorf("stepped vim difficulty = %+v, want medium", msg)
	}
}
