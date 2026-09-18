package vim

import (
	"sort"
	"strings"
)

// Difficulty groups challenges by how much Vim knowledge they demand
type Difficulty int

const (
	Easy Difficulty = iota
	Medium
	Hard
	Mixed
)

// Label is the human-readable difficulty name shown on the home screen
func (d Difficulty) Label() string {
	switch d {
	case Easy:
		return "easy"
	case Medium:
		return "medium"
	case Hard:
		return "hard"
	default:
		return "mixed"
	}
}

// Goal is the target state a challenge must reach
type Goal struct {
	Lines  []string
	Cursor *Pos
}

// At builds a goal cursor pointer inline.
func At(row, col int) *Pos { return &Pos{Row: row, Col: col} }

// Challenge is one task
type Challenge struct {
	Task       string
	Buffer     []string
	Cursor     Pos
	Difficulty Difficulty
	Goal       Goal
	optimal    []string
}

// NewChallenge builds a Challenge
func NewChallenge(task string, buffer []string, cursor Pos, difficulty Difficulty, goal Goal, optimal string) Challenge {
	tokens := Tokenize(optimal)
	if len(tokens) == 0 {
		panic("challenge requires a non-empty optimal solution")
	}
	if goal.Lines == nil {
		goal.Lines = append([]string(nil), buffer...)
	}
	return Challenge{
		Task:       task,
		Buffer:     buffer,
		Cursor:     cursor,
		Difficulty: difficulty,
		Goal:       goal,
		optimal:    tokens,
	}
}

// Par is the keystroke count of the authored optimal solution.
func (c Challenge) Par() int { return len(c.optimal) }

// Hint renders the authored optimal solution, e.g. "diw".
func (c Challenge) Hint() string { return JoinTokens(c.optimal) }

// NewEngine returns the live engine for this challenge
func (c Challenge) NewEngine() (Engine, error) {
	return NewEngine(c.Buffer, c.Cursor)
}

// Tokenize splits a solution string into individual keystroke tokens
func Tokenize(s string) []string {
	tokens := make([]string, 0, len(s))
	runes := []rune(s)
	for i := 0; i < len(runes); i++ {
		if runes[i] == '<' {
			if end := indexRune(runes, '>', i+1); end > i {
				tokens = append(tokens, string(runes[i:end+1]))
				i = end
				continue
			}
		}
		tokens = append(tokens, string(runes[i]))
	}
	return tokens
}

// JoinTokens renders tokens back into their readable keystroke string.
func JoinTokens(tokens []string) string {
	return strings.Join(tokens, "")
}

func indexRune(runes []rune, r rune, from int) int {
	for i := from; i < len(runes); i++ {
		if runes[i] == r {
			return i
		}
	}
	return -1
}

// sortByDifficulty orders challenges by difficulty then task
func sortByDifficulty(challenges []Challenge) {
	sort.SliceStable(challenges, func(i, j int) bool {
		if challenges[i].Difficulty != challenges[j].Difficulty {
			return challenges[i].Difficulty < challenges[j].Difficulty
		}
		return challenges[i].Task < challenges[j].Task
	})
}
