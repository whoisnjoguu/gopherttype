// TODO: Implement more sophisticated procedural generation for challenges.
// like in internal/vim/library.go, this file provides procedurally generated challenges for the game.

package vim

import (
	"math/rand/v2"
	"strings"

	"github.com/mkhamat/gopherttype/internal/words"
)

// wordPool is the shared source of random words for generated challenges.
var wordPool = words.List()

// generator builds one concrete challenge from random words
type generator struct {
	difficulty Difficulty
	build      func(g *challengeGen) Challenge
}

// challengeGen provides word helpers to a generator.
type challengeGen struct {
	rng  *rand.Rand
	pool []string
}

func (g *challengeGen) pick() string { return g.pool[g.rng.IntN(len(g.pool))] }

func (g *challengeGen) distinct(from ...string) string {
	for {
		w := g.pick()
		unique := true
		for _, f := range from {
			if w == f {
				unique = false
				break
			}
		}
		if unique {
			return w
		}
	}
}

// Round builds a fresh, procedurally generated round
func Round(difficulty Difficulty, count int, rng *rand.Rand) []Challenge {
	if rng == nil {
		rng = rand.New(rand.NewPCG(1, 2))
	}
	if count <= 0 {
		count = RoundSize
	}
	gens := generatorsFor(difficulty)
	cg := &challengeGen{rng: rng, pool: wordPool}

	// Shuffle generator order for variety, then cycle if more are needed.
	order := make([]generator, len(gens))
	copy(order, gens)
	rng.Shuffle(len(order), func(i, j int) { order[i], order[j] = order[j], order[i] })

	out := make([]Challenge, 0, count)
	for i := 0; len(out) < count; i++ {
		out = append(out, order[i%len(order)].build(cg))
	}
	return out
}

func generatorsFor(difficulty Difficulty) []generator {
	if difficulty == Mixed {
		all := make([]generator, 0, len(generators))
		all = append(all, generators...)
		return all
	}
	var out []generator
	for _, g := range generators {
		if g.difficulty == difficulty {
			out = append(out, g)
		}
	}
	if len(out) == 0 {
		return generators
	}
	return out
}

// join renders a buffer line from words.
func join(w ...string) string { return strings.Join(w, " ") }

// generators is the full set of challenge templates.
var generators = []generator{
	// ---- Easy ----
	{Easy, func(g *challengeGen) Challenge {
		a, b, c := g.pick(), g.distinct(), g.pick()
		return NewChallenge(
			"Delete the middle word (keep the surrounding spaces).",
			[]string{join(a, b, c)}, Pos{0, len(a) + 1}, Easy,
			Goal{Lines: []string{a + "  " + c}}, "diw",
		)
	}},
	{Easy, func(g *challengeGen) Challenge {
		a, b, c := g.pick(), g.pick(), g.pick()
		return NewChallenge(
			"Delete from the cursor to the end of the line.",
			[]string{join(a, b, c)}, Pos{0, len(a) + 1}, Easy,
			Goal{Lines: []string{a + " "}}, "D",
		)
	}},
	{Easy, func(g *challengeGen) Challenge {
		a, b, c := g.pick(), g.pick(), g.pick()
		return NewChallenge(
			"Delete the whole middle line.",
			[]string{a, b, c}, Pos{1, 0}, Easy,
			Goal{Lines: []string{a, c}}, "dd",
		)
	}},
	{Easy, func(g *challengeGen) Challenge {
		a, b, c := g.pick(), g.pick(), g.pick()
		return NewChallenge(
			"Delete the first word and the space after it.",
			[]string{join(a, b, c)}, Pos{0, 0}, Easy,
			Goal{Lines: []string{join(b, c)}}, "dw",
		)
	}},
	{Easy, func(g *challengeGen) Challenge {
		a, b, c := g.pick(), g.pick(), g.pick()
		line := join(a, b, c)
		return NewChallenge(
			"Move the cursor to the last character of the line.",
			[]string{line}, Pos{0, 0}, Easy,
			Goal{Cursor: At(0, len([]rune(line))-1)}, "$",
		)
	}},

	// ---- Medium ----
	{Medium, func(g *challengeGen) Challenge {
		a, b, c := g.pick(), g.pick(), g.pick()
		nw := g.distinct(a, b, c)
		return NewChallenge(
			"Change the middle word to \""+nw+"\".",
			[]string{join(a, b, c)}, Pos{0, len(a) + 1}, Medium,
			Goal{Lines: []string{join(a, nw, c)}}, "cw"+nw+"<Esc>",
		)
	}},
	{Medium, func(g *challengeGen) Challenge {
		needle := g.distinct()
		a, c := g.distinct(needle), g.distinct(needle)
		line := join(a, needle, c)
		return NewChallenge(
			"Search forward for \""+needle+"\".",
			[]string{line, join(g.pick(), g.pick())}, Pos{0, 0}, Medium,
			Goal{Cursor: At(0, len(a)+1)}, "/"+needle+"<CR>",
		)
	}},
	{Medium, func(g *challengeGen) Challenge {
		a, b := g.pick(), g.pick()
		return NewChallenge(
			"Delete everything inside the parentheses.",
			[]string{"call(" + join(a, b) + ")"}, Pos{0, 6}, Medium,
			Goal{Lines: []string{"call()"}}, "di(",
		)
	}},
	{Medium, func(g *challengeGen) Challenge {
		line := join(g.pick(), g.pick())
		return NewChallenge(
			"Duplicate the current line.",
			[]string{line}, Pos{0, 0}, Medium,
			Goal{Lines: []string{line, line}}, "yyp",
		)
	}},
	{Medium, func(g *challengeGen) Challenge {
		a, b := g.pick(), g.pick()
		return NewChallenge(
			"Join this line and the next into one.",
			[]string{a, b}, Pos{0, 0}, Medium,
			Goal{Lines: []string{a + " " + b}}, "J",
		)
	}},
	{Medium, func(g *challengeGen) Challenge {
		line := join(g.pick(), g.pick(), g.pick())
		return NewChallenge(
			"Append a ';' at the end of the line.",
			[]string{line}, Pos{0, 0}, Medium,
			Goal{Lines: []string{line + ";"}}, "A;<Esc>",
		)
	}},

	// ---- Hard ----
	{Hard, func(g *challengeGen) Challenge {
		x := g.distinct()
		y := g.distinct(x)
		return NewChallenge(
			"Replace every \""+x+"\" on this line with \""+y+"\".",
			[]string{join(x, x, x)}, Pos{0, 0}, Hard,
			Goal{Lines: []string{join(y, y, y)}}, ":s/"+x+"/"+y+"/g<CR>",
		)
	}},
	{Hard, func(g *challengeGen) Challenge {
		x := g.distinct()
		y := g.distinct(x)
		a, b := g.distinct(x), g.distinct(x)
		return NewChallenge(
			"Replace every \""+x+"\" in the whole file with \""+y+"\".",
			[]string{join(x, a), join(b, x)}, Pos{0, 0}, Hard,
			Goal{Lines: []string{join(y, a), join(b, y)}}, ":%s/"+x+"/"+y+"/g<CR>",
		)
	}},
	{Hard, func(g *challengeGen) Challenge {
		a, b, c, d := g.pick(), g.pick(), g.pick(), g.pick()
		return NewChallenge(
			"Delete the first two words.",
			[]string{join(a, b, c, d)}, Pos{0, 0}, Hard,
			Goal{Lines: []string{join(c, d)}}, "d2w",
		)
	}},
	{Hard, func(g *challengeGen) Challenge {
		a, b, c := g.pick(), g.pick(), g.pick()
		return NewChallenge(
			"Uppercase the middle word.",
			[]string{join(a, b, c)}, Pos{0, len(a) + 1}, Hard,
			Goal{Lines: []string{join(a, strings.ToUpper(b), c)}}, "gUiw",
		)
	}},
	{Hard, func(g *challengeGen) Challenge {
		line := join(g.pick(), g.pick())
		return NewChallenge(
			"Insert '# ' at the very start of the line.",
			[]string{line}, Pos{0, 3}, Hard,
			Goal{Lines: []string{"# " + line}}, "I# <Esc>",
		)
	}},
	{Hard, func(g *challengeGen) Challenge {
		line := join(g.pick(), g.pick())
		nw := g.pick()
		return NewChallenge(
			"Open a new line below and type \""+nw+"\".",
			[]string{line}, Pos{0, 0}, Hard,
			Goal{Lines: []string{line, nw}}, "o"+nw+"<Esc>",
		)
	}},
	{Hard, func(g *challengeGen) Challenge {
		old := g.pick()
		nw := g.distinct(old)
		return NewChallenge(
			"Change the text inside the double quotes to \""+nw+"\".",
			[]string{"key = \"" + old + "\""}, Pos{0, 7}, Hard,
			Goal{Lines: []string{"key = \"" + nw + "\""}}, "ci\""+nw+"<Esc>",
		)
	}},
	{Hard, func(g *challengeGen) Challenge {
		a := g.pick()
		b := g.pick()
		return NewChallenge(
			"Delete from the cursor up to and including the '-'.",
			[]string{a + "-" + b}, Pos{0, 0}, Hard,
			Goal{Lines: []string{b}}, "df-",
		)
	}},
}
