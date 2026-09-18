package vim

import (
	"math/rand/v2"
	"testing"
	"time"
)

// feedNvim drives an engine to completion for the given tokens
func feedNvim(t *testing.T, e Engine, tokens []string) {
	t.Helper()
	for _, tok := range tokens {
		e.Feed(tok)
		e.Settle()
	}
	// Give any final read a moment to land.
	deadline := time.Now().Add(nvimSettleCap)
	for e.Pending() && time.Now().Before(deadline) {
		e.Settle()
	}
}

// typeSettled feeds one token through a Session and, if it reports Pending,
// gives it a single settle+regrade pass, mirroring feedNvim's per-token
// cadence. Intermediate keys of a multi-key command are expected to stay
// Pending until the rest of the sequence arrives, so this does not retry.
func typeSettled(t *testing.T, s *Session, token string, at time.Time) Outcome {
	t.Helper()
	out := s.Type(token, at)
	if out.Pending {
		s.WaitSettle()
		out = s.Regrade(at)
	}
	return out
}

// TestNvimVisualSpan checks that the embedded Neovim reports a visual selection
// span so the UI can highlight it.
func TestNvimVisualSpan(t *testing.T) {
	if !nvimAvailable() {
		t.Skip("nvim not installed")
	}
	e, err := newNvimEngine([]string{"hello world"}, Pos{0, 0})
	if err != nil {
		t.Fatalf("start nvim: %v", err)
	}
	defer e.Close()
	feedNvim(t, e, []string{"v", "e"})
	span, active := e.VisualSpan()
	if !active {
		t.Fatalf("expected an active visual span, mode=%q", e.ModeLabel())
	}
	if span.Start != (Pos{0, 0}) || span.End != (Pos{0, 4}) {
		t.Fatalf("span = %+v, want start (0,0) end (0,4)", span)
	}
}

// TestNvimSessionRound drives a whole generated round through the exact Session
// API the UI uses against real Neovim
func TestNvimSessionRound(t *testing.T) {
	if !nvimAvailable() {
		t.Skip("nvim not installed")
	}

	rng := rand.New(rand.NewPCG(11, 22))
	round := Round(Mixed, RoundSize, rng)
	now := time.Now()
	s, err := NewSession(round, now)
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}

	for _, c := range round {
		toks := Tokenize(c.Hint())
		for _, tok := range toks {
			now = now.Add(50 * time.Millisecond)
			s.Type(tok, now)
		}
		// After the final key of the solution, the command may still be
		// mid-flight in nvim; settle and regrade like the UI does.
		for attempt := 0; attempt < 20 && s.Pending(); attempt++ {
			s.WaitSettle()
			out := s.Regrade(now)
			if out.Solved {
				break
			}
		}
	}

	if s.Status() != Finished {
		t.Fatalf("round did not finish: status=%v index=%d", s.Status(), s.Index())
	}
	report := s.Report()
	if report.Solved != report.Total {
		t.Fatalf("solved %d/%d via nvim session", report.Solved, report.Total)
	}
}
