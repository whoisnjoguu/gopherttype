package vim

import (
	"math/rand/v2"
	"testing"
)

// TestGeneratedRoundsAreSolvable is the key gate for procedural content
func TestGeneratedRoundsAreSolvable(t *testing.T) {
	if !nvimAvailable() {
		t.Skip("nvim not installed")
	}
	rng := rand.New(rand.NewPCG(1, 2))
	for _, d := range []Difficulty{Easy, Medium, Hard, Mixed} {
		for _, c := range Round(d, RoundSize, rng) {
			e, err := newNvimEngine(c.Buffer, c.Cursor)
			if err != nil {
				t.Fatalf("start nvim: %v", err)
			}
			feedNvim(t, e, Tokenize(c.Hint()))
			ok := e.Matches(c.Goal)
			gotLines, gotCur, gotMode := e.Lines(), e.Cursor(), e.ModeLabel()
			e.Close()
			if !ok {
				t.Fatalf("diff=%s task=%q optimal=%q did not reach goal: got lines=%q cursor=%+v mode=%q",
					d.Label(), c.Task, c.Hint(), gotLines, gotCur, gotMode)
			}
		}
	}
}

func TestRoundCount(t *testing.T) {
	rng := rand.New(rand.NewPCG(7, 9))
	got := Round(Easy, RoundSize, rng)
	if len(got) != RoundSize {
		t.Fatalf("round size = %d, want %d", len(got), RoundSize)
	}
	for _, c := range got {
		if c.Difficulty != Easy {
			t.Errorf("expected easy challenge, got %v", c.Difficulty)
		}
	}
}

func TestRoundDeterministicWithNilRng(t *testing.T) {
	a := Round(Mixed, RoundSize, nil)
	b := Round(Mixed, RoundSize, nil)
	if len(a) != len(b) {
		t.Fatalf("length mismatch")
	}
	for i := range a {
		if a[i].Task != b[i].Task {
			t.Fatalf("nil rng not deterministic at %d: %q vs %q", i, a[i].Task, b[i].Task)
		}
	}
}
