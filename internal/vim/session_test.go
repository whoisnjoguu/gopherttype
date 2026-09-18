package vim

import (
	"math/rand/v2"
	"testing"
	"time"
)

func TestTokenizeBracketEscapes(t *testing.T) {
	got := Tokenize("ciwbaz<Esc>")
	want := []string{"c", "i", "w", "b", "a", "z", "<Esc>"}
	if len(got) != len(want) {
		t.Fatalf("token count = %d, want %d (%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("token %d = %q, want %q", i, got[i], want[i])
		}
	}
	if joined := JoinTokens(got); joined != "ciwbaz<Esc>" {
		t.Errorf("JoinTokens round-trip = %q", joined)
	}
}

func TestTokenizeSearchAndEx(t *testing.T) {
	got := JoinTokens(Tokenize(":%s/foo/bar/g<CR>"))
	if got != ":%s/foo/bar/g<CR>" {
		t.Fatalf("round-trip = %q", got)
	}
	if n := len(Tokenize("/foo<CR>")); n != 5 {
		t.Errorf("'/foo<CR>' token count = %d, want 5", n)
	}
}

func TestChallengeParAndHint(t *testing.T) {
	c := NewChallenge("t", []string{"x word"}, Pos{0, 2}, Easy,
		Goal{Lines: []string{"x "}}, "dw")
	if c.Par() != 2 {
		t.Errorf("Par = %d, want 2", c.Par())
	}
	if c.Hint() != "dw" {
		t.Errorf("Hint = %q, want dw", c.Hint())
	}
}

func TestNewChallengePanicsWithoutOptimal(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic for missing optimal solution")
		}
	}()
	NewChallenge("t", []string{"x"}, Pos{0, 0}, Easy, Goal{}, "")
}

func TestSessionSolvePath(t *testing.T) {
	c := NewChallenge("delete word", []string{"the quick fox"}, Pos{0, 4}, Easy,
		Goal{Lines: []string{"the  fox"}}, "diw")
	start := time.Unix(0, 0)
	s, err := NewSession([]Challenge{c}, start)
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}

	if got := s.Type("d", start); got.Solved {
		t.Fatalf("after 'd' = %+v, want not solved", got)
	}
	if got := s.Type("i", start); got.Solved {
		t.Fatalf("after 'i' = %+v", got)
	}
	solved := s.Type("w", start.Add(2*time.Second))
	if !solved.Solved {
		t.Fatalf("after 'w' = %+v, want solved", solved)
	}
	if s.Status() != Finished {
		t.Fatalf("status = %v, want Finished", s.Status())
	}
	report := s.Report()
	if report.Solved != 1 || report.Total != 1 {
		t.Fatalf("solved/total = %d/%d, want 1/1", report.Solved, report.Total)
	}
	if report.Results[0].Used != 3 || report.Results[0].Par != 3 {
		t.Fatalf("used/par = %d/%d, want 3/3", report.Results[0].Used, report.Results[0].Par)
	}
	if eff := report.Efficiency(); eff != 100 {
		t.Errorf("efficiency = %v, want 100", eff)
	}
}

func TestSessionCountsBeepsAsMistakes(t *testing.T) {
	c := NewChallenge("delete char", []string{"caat"}, Pos{0, 0}, Easy,
		Goal{Lines: []string{"cat"}}, "x")
	start := time.Unix(0, 0)
	s, err := NewSession([]Challenge{c}, start)
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}

	// 'h' at column 0 is a no-op: it should beep and count as a mistake.
	if got := s.Type("h", start); got.Accepted {
		t.Fatalf("'h' at col 0 should beep, got %+v", got)
	}
	if s.Mistakes() != 1 {
		t.Fatalf("mistakes = %d, want 1", s.Mistakes())
	}
	// A beep must not stall the real solution.
	s.Type("l", start) // move onto the extra 'a'
	if out := s.Type("x", start); !out.Solved {
		t.Fatalf("valid path after beep failed: %+v", out)
	}
}

func TestSessionSkipCountsAsUnsolved(t *testing.T) {
	challenges := []Challenge{
		NewChallenge("a", []string{"xy"}, Pos{0, 0}, Easy, Goal{Lines: []string{"y"}}, "x"),
		NewChallenge("b", []string{"xy"}, Pos{0, 0}, Easy, Goal{Lines: []string{"y"}}, "x"),
	}
	start := time.Unix(0, 0)
	s, err := NewSession(challenges, start)
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	s.Skip(start.Add(time.Second))
	s.Type("x", start.Add(2*time.Second))
	report := s.Report()
	if report.Solved != 1 {
		t.Fatalf("solved = %d, want 1", report.Solved)
	}
	if report.Total != 2 {
		t.Fatalf("total = %d, want 2", report.Total)
	}
	if report.Results[0].Solved {
		t.Error("skipped challenge should be unsolved")
	}
	if report.Results[0].Efficiency() != 0 {
		t.Error("skipped efficiency should be 0")
	}
	if eff := report.Efficiency(); eff != 50 {
		t.Errorf("efficiency = %v, want 50", eff)
	}
}

func TestSpeedBonusRewardsFastSolves(t *testing.T) {
	fast := ChallengeResult{Solved: true, Par: 3, Used: 3, Elapsed: time.Second}
	slow := ChallengeResult{Solved: true, Par: 3, Used: 3, Elapsed: time.Minute}
	if fast.Score() <= slow.Score() {
		t.Fatalf("fast score %d should beat slow %d", fast.Score(), slow.Score())
	}
	if slow.Score() < int(solveBase) {
		t.Fatalf("any solve should score at least the solve base")
	}
}

// TestLibraryIntegrity is the key validation gate: for every challenge, feeding
// its authored optimal solution into a fresh emulator must reach the goal. This
// validates both the emulator and the authored solutions at once.
func TestLibraryIntegrity(t *testing.T) {
	for _, c := range Library() {
		if len(c.Buffer) == 0 {
			t.Errorf("%q: empty buffer", c.Task)
			continue
		}
		if c.Cursor.Row < 0 || c.Cursor.Row >= len(c.Buffer) {
			t.Errorf("%q: cursor row %d out of range", c.Task, c.Cursor.Row)
		}
		if line := []rune(c.Buffer[c.Cursor.Row]); c.Cursor.Col < 0 || c.Cursor.Col > len(line) {
			t.Errorf("%q: cursor col %d out of range", c.Task, c.Cursor.Col)
		}
		ed := c.NewEditor()
		for _, tok := range Tokenize(c.Hint()) {
			ed.Feed(tok)
		}
		if !ed.Matches(c.Goal) {
			t.Errorf("%q: optimal %q did not reach goal. got lines=%q cursor=%+v mode=%v",
				c.Task, c.Hint(), ed.Lines(), ed.Cursor(), ed.ModeLabel())
		}
	}
}

func TestSelectChallengesByDifficulty(t *testing.T) {
	for _, d := range []Difficulty{Easy, Medium, Hard} {
		got := SelectChallenges(d, RoundSize, nil)
		if len(got) == 0 {
			t.Fatalf("%s: no challenges", d.Label())
		}
		for _, c := range got {
			if c.Difficulty != d {
				t.Errorf("%s: got challenge of difficulty %v", d.Label(), c.Difficulty)
			}
		}
	}
	mixed := SelectChallenges(Mixed, 5, nil)
	if len(mixed) != 5 {
		t.Fatalf("mixed count = %d, want 5", len(mixed))
	}
}

func TestSelectChallengesShuffleDeterministic(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))
	a := SelectChallenges(Mixed, RoundSize, rng)
	rng2 := rand.New(rand.NewPCG(1, 2))
	b := SelectChallenges(Mixed, RoundSize, rng2)
	if len(a) != len(b) {
		t.Fatalf("length mismatch")
	}
	for i := range a {
		if a[i].Task != b[i].Task {
			t.Fatalf("same seed produced different order at %d", i)
		}
	}
}
