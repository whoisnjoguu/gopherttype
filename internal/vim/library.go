// TODO
// This file contains sample challenges for the vim typing game mode
// I've added just a couple to test the nvim client.
// sort of a PoC for testing you can actually run a vim client
// IMPORTANT: In a later version, challenges should not be hard-coded but rather curated or generated dynamically.

package vim

import "math/rand/v2"

// RoundSize is the number of challenges in a standard round.
const RoundSize = 8

// library is the full curated set of challenges
var library = buildLibrary()

func buildLibrary() []Challenge {
	challenges := []Challenge{
		// ---- Easy: single motions, deletes and edits. ----
		NewChallenge(
			"Delete the word 'quick' under the cursor (keep the surrounding spaces).",
			[]string{"the quick brown fox"}, Pos{0, 4}, Easy,
			Goal{Lines: []string{"the  brown fox"}},
			"diw",
		),
		NewChallenge(
			"Delete the extra 'l' under the cursor to fix 'helllo'.",
			[]string{"helllo world"}, Pos{0, 3}, Easy,
			Goal{Lines: []string{"hello world"}},
			"x",
		),
		NewChallenge(
			"Delete from the cursor to the end of the line.",
			[]string{"keep DELETE ME"}, Pos{0, 5}, Easy,
			Goal{Lines: []string{"keep "}},
			"D",
		),
		NewChallenge(
			"Move the cursor to the last character of the line.",
			[]string{"move to the end"}, Pos{0, 0}, Easy,
			Goal{Cursor: At(0, 14)},
			"$",
		),
		NewChallenge(
			"Move the cursor to the first character of the line.",
			[]string{"        indented"}, Pos{0, 8}, Easy,
			Goal{Cursor: At(0, 0)},
			"0",
		),
		NewChallenge(
			"Delete the entire current line.",
			[]string{"first line", "delete this line", "third line"}, Pos{1, 3}, Easy,
			Goal{Lines: []string{"first line", "third line"}},
			"dd",
		),
		NewChallenge(
			"Delete the character before the cursor to fix 'caat'.",
			[]string{"caat"}, Pos{0, 2}, Easy,
			Goal{Lines: []string{"cat"}},
			"X",
		),
		NewChallenge(
			"Delete the first word and the space after it.",
			[]string{"remove keep the rest"}, Pos{0, 0}, Easy,
			Goal{Lines: []string{"keep the rest"}},
			"dw",
		),

		// ---- Medium: operators with motions, search, registers. ----
		NewChallenge(
			"Change the word 'counter' to 'total'.",
			[]string{"let counter = 0"}, Pos{0, 4}, Medium,
			Goal{Lines: []string{"let total = 0"}},
			"cwtotal<Esc>",
		),
		NewChallenge(
			"Search forward for the word 'needle'.",
			[]string{"find the needle in", "this large haystack"}, Pos{0, 0}, Medium,
			Goal{Cursor: At(0, 9)},
			"/needle<CR>",
		),
		NewChallenge(
			"Replace the 'c' under the cursor with 'b' to make 'bat'.",
			[]string{"cat"}, Pos{0, 0}, Medium,
			Goal{Lines: []string{"bat"}},
			"rb",
		),
		NewChallenge(
			"Delete everything inside the parentheses.",
			[]string{"call(remove me)"}, Pos{0, 8}, Medium,
			Goal{Lines: []string{"call()"}},
			"di(",
		),
		NewChallenge(
			"Duplicate the current line (copy it and paste a copy below).",
			[]string{"copy me"}, Pos{0, 0}, Medium,
			Goal{Lines: []string{"copy me", "copy me"}},
			"yyp",
		),
		NewChallenge(
			"Join this line and the next into one.",
			[]string{"first half", "second half"}, Pos{0, 0}, Medium,
			Goal{Lines: []string{"first half second half"}},
			"J",
		),
		NewChallenge(
			"Append a ';' at the end of the line.",
			[]string{"int x = 5"}, Pos{0, 0}, Medium,
			Goal{Lines: []string{"int x = 5;"}},
			"A;<Esc>",
		),
		NewChallenge(
			"Jump to the last line of the file.",
			[]string{"line one", "line two", "line three", "line four"}, Pos{0, 0}, Medium,
			Goal{Cursor: At(3, 0)},
			"G",
		),
		NewChallenge(
			"Jump to the first line of the file.",
			[]string{"line one", "line two", "line three", "line four"}, Pos{3, 0}, Medium,
			Goal{Cursor: At(0, 0)},
			"gg",
		),

		// ---- Hard: ex commands, text objects, multi-step edits. ----
		NewChallenge(
			"Replace every 'foo' on this line with 'bar'.",
			[]string{"foo foo foo"}, Pos{0, 0}, Hard,
			Goal{Lines: []string{"bar bar bar"}},
			":s/foo/bar/g<CR>",
		),
		NewChallenge(
			"Replace every 'foo' in the whole file with 'bar'.",
			[]string{"foo one", "two foo", "foo three"}, Pos{0, 0}, Hard,
			Goal{Lines: []string{"bar one", "two bar", "bar three"}},
			":%s/foo/bar/g<CR>",
		),
		NewChallenge(
			"Delete from the cursor up to and including the first '-'.",
			[]string{"remove-then-keep"}, Pos{0, 0}, Hard,
			Goal{Lines: []string{"then-keep"}},
			"df-",
		),
		NewChallenge(
			"Uppercase the word 'this' under the cursor.",
			[]string{"shout this word"}, Pos{0, 6}, Hard,
			Goal{Lines: []string{"shout THIS word"}},
			"gUiw",
		),
		NewChallenge(
			"Insert '# ' at the very start of the line.",
			[]string{"value"}, Pos{0, 2}, Hard,
			Goal{Lines: []string{"# value"}},
			"I# <Esc>",
		),
		NewChallenge(
			"Delete the first two words so the line starts with 'two'.",
			[]string{"drop these two words"}, Pos{0, 0}, Hard,
			Goal{Lines: []string{"two words"}},
			"d2w",
		),
		NewChallenge(
			"Open a new line below and type 'done', then return to normal mode.",
			[]string{"first line"}, Pos{0, 0}, Hard,
			Goal{Lines: []string{"first line", "done"}},
			"odone<Esc>",
		),
		NewChallenge(
			"Change the text inside the double quotes to 'new'.",
			[]string{"title = \"old\""}, Pos{0, 9}, Hard,
			Goal{Lines: []string{"title = \"new\""}},
			"ci\"new<Esc>",
		),
		NewChallenge(
			"Delete the text inside the double quotes.",
			[]string{"name = \"remove me\""}, Pos{0, 10}, Hard,
			Goal{Lines: []string{"name = \"\""}},
			"di\"",
		),
	}
	sortByDifficulty(challenges)
	return challenges
}

// Library returns a copy of every challenge, ordered by difficulty then task.
func Library() []Challenge {
	out := make([]Challenge, len(library))
	copy(out, library)
	return out
}

// SelectChallenges returns up to count challenges for the difficulty, shuffled with rng
func SelectChallenges(difficulty Difficulty, count int, rng *rand.Rand) []Challenge {
	var pool []Challenge
	for _, c := range library {
		if difficulty == Mixed || c.Difficulty == difficulty {
			pool = append(pool, c)
		}
	}
	if rng != nil {
		rng.Shuffle(len(pool), func(i, j int) { pool[i], pool[j] = pool[j], pool[i] })
	}
	if count > 0 && count < len(pool) {
		pool = pool[:count]
	}
	return pool
}
