package vim

import (
	"math"
	"time"
)

// Status tracks the lifecycle of a round.
type Status int

const (
	Ready Status = iota
	Playing
	Finished
	Errored
)

// Scoring weights.
// A solved challenge earns a solve base plus an efficiency
// bonus (how close to par) and a speed bonus (how close to an "expert" time,
// estimated as a fixed budget per par keystroke). Everything is centralized so
// the results screen and docs stay in sync.
const (
	solveBase              = 100.0
	efficiencyBonusMax     = 100.0
	speedBonusMax          = 100.0
	expertSecondsPerStroke = 0.9
)

// Outcome reports what one keystroke did
type Outcome struct {
	Accepted bool
	Solved   bool
	Pending  bool
}

// ChallengeResult is the graded outcome of a single challenge
type ChallengeResult struct {
	Task     string
	Solved   bool
	Par      int
	Used     int
	Mistakes int
	Elapsed  time.Duration
	Solution string
}

// Efficiency is par/used, clamped to 1, or 0 when unsolved
func (r ChallengeResult) Efficiency() float64 {
	if !r.Solved || r.Used <= 0 {
		return 0
	}
	return math.Min(1, float64(r.Par)/float64(r.Used))
}

// speedRatio compares the solve time against an expert budget of
// expertSecondsPerStroke per par keystroke
func (r ChallengeResult) speedRatio() float64 {
	if !r.Solved {
		return 0
	}
	budget := float64(r.Par) * expertSecondsPerStroke
	elapsed := r.Elapsed.Seconds()
	if budget <= 0 {
		return 0
	}
	if elapsed <= budget {
		return 1
	}
	return math.Max(0, budget/elapsed)
}

// Score combines the solve base with the efficiency and speed bonuses.
func (r ChallengeResult) Score() int {
	if !r.Solved {
		return 0
	}
	return int(math.Round(solveBase + efficiencyBonusMax*r.Efficiency() + speedBonusMax*r.speedRatio()))
}

// Session drives a round of challenges
type Session struct {
	challenges []Challenge
	current    int
	engine     Engine
	used       int
	mistakes   int
	status     Status
	startedAt  time.Time
	shownAt    time.Time
	results    []ChallengeResult
	err        error
}

// NewSession starts a round over the given challenges.
func NewSession(challenges []Challenge, at time.Time) (*Session, error) {
	if len(challenges) == 0 {
		panic("a session needs at least one challenge")
	}
	engine, err := challenges[0].NewEngine()
	if err != nil {
		return nil, err
	}
	return &Session{
		challenges: challenges,
		engine:     engine,
		status:     Ready,
		startedAt:  at,
		shownAt:    at,
		results:    make([]ChallengeResult, 0, len(challenges)),
	}, nil
}

// Err reports why the session stopped when Status is Errored.
func (s *Session) Err() error { return s.err }

// Status reports the round lifecycle.
func (s *Session) Status() Status { return s.status }

// Total is the number of challenges in the round.
func (s *Session) Total() int { return len(s.challenges) }

// Index is the zero-based position of the current challenge.
func (s *Session) Index() int { return s.current }

// Current returns the challenge in progress. It must not be called once the
// session is finished.
func (s *Session) Current() Challenge {
	return s.challenges[s.current]
}

// Engine exposes the live Vim engine for the current challenge
func (s *Session) Engine() Engine { return s.engine }

// Used is the number of keystrokes fed on the current challenge.
func (s *Session) Used() int { return s.used }

// Mistakes is the number of rejected (beeped) keystrokes on the current challenge.
func (s *Session) Mistakes() int { return s.mistakes }

// Pending reports whether the engine is still catching up to a multi-key command
func (s *Session) Pending() bool {
	if s.status == Finished || s.engine == nil {
		return false
	}
	return s.engine.Pending()
}

// Type feeds one canonical keystroke token to the live engine and grades the result
func (s *Session) Type(token string, at time.Time) Outcome {
	if s.status == Finished || s.status == Errored {
		return Outcome{}
	}
	if s.status == Ready {
		s.status = Playing
		if s.startedAt.IsZero() {
			s.startedAt = at
		}
	}

	accepted := s.engine.Feed(token)
	s.used++
	if !accepted {
		s.mistakes++
	}
	return s.grade(accepted, at)
}

// WaitSettle blocks until the engine finishes applying the last keystroke
func (s *Session) WaitSettle() {
	if s.status == Finished || s.engine == nil {
		return
	}
	s.engine.Settle()
}

// Regrade re-checks whether the current challenge is solved after a settle,
// recording and advancing when it is.
func (s *Session) Regrade(at time.Time) Outcome {
	if s.status == Finished || s.engine == nil {
		return Outcome{}
	}
	return s.grade(true, at)
}

// grade checks whether the current challenge is solved
func (s *Session) grade(accepted bool, at time.Time) Outcome {
	if s.engine.Matches(s.challenges[s.current].Goal) {
		s.record(true, at)
		s.advance(at)
		return Outcome{Accepted: accepted, Solved: true}
	}
	if s.engine.Pending() {
		return Outcome{Accepted: accepted, Pending: true}
	}
	return Outcome{Accepted: accepted}
}

func (s *Session) Skip(at time.Time) {
	if s.status == Finished || s.status == Errored {
		return
	}
	if s.status == Ready {
		s.status = Playing
	}
	s.record(false, at)
	s.advance(at)
}

func (s *Session) record(solved bool, at time.Time) {
	challenge := s.challenges[s.current]
	solution := ""
	if solved {
		solution = challenge.Hint()
	}
	s.results = append(s.results, ChallengeResult{
		Task:     challenge.Task,
		Solved:   solved,
		Par:      challenge.Par(),
		Used:     s.used,
		Mistakes: s.mistakes,
		Elapsed:  at.Sub(s.shownAt),
		Solution: solution,
	})
}

func (s *Session) advance(at time.Time) {
	s.current++
	s.used = 0
	s.mistakes = 0
	s.shownAt = at
	if s.engine != nil {
		s.engine.Close()
	}
	if s.current >= len(s.challenges) {
		s.status = Finished
		s.engine = nil
		return
	}
	engine, err := s.challenges[s.current].NewEngine()
	if err != nil {
		s.status = Errored
		s.err = err
		s.engine = nil
		return
	}
	s.engine = engine
}

// Report aggregates the finished round
type Report struct {
	Total         int
	Solved        int
	TotalPar      int
	TotalUsed     int
	TotalMistakes int
	Score         int
	Duration      time.Duration
	Results       []ChallengeResult
}

// Report builds the aggregate scoreboard
func (s *Session) Report() Report {
	report := Report{
		Total:    len(s.challenges),
		Duration: s.duration(),
		Results:  append([]ChallengeResult(nil), s.results...),
	}
	for _, r := range s.results {
		if r.Solved {
			report.Solved++
			report.TotalPar += r.Par
			report.TotalUsed += r.Used
		}
		report.TotalMistakes += r.Mistakes
		report.Score += r.Score()
	}
	return report
}

func (s *Session) duration() time.Duration {
	if len(s.results) == 0 {
		return 0
	}
	var total time.Duration
	for _, r := range s.results {
		total += r.Elapsed
	}
	return total
}

// Efficiency is the average per-challenge efficiency across every challenge in the round
func (r Report) Efficiency() float64 {
	if r.Total == 0 {
		return 0
	}
	var sum float64
	for _, res := range r.Results {
		sum += res.Efficiency()
	}
	return roundToTwo(sum / float64(r.Total) * 100)
}

// SolveRate is the fraction of solved challenges as a 0..100 percentage.
func (r Report) SolveRate() float64 {
	if r.Total == 0 {
		return 0
	}
	return roundToTwo(float64(r.Solved) / float64(r.Total) * 100)
}

// Pace is solved challenges per minute over the round.
func (r Report) Pace() float64 {
	minutes := r.Duration.Minutes()
	if minutes <= 0 {
		return 0
	}
	return roundToTwo(float64(r.Solved) / minutes)
}

func roundToTwo(value float64) float64 {
	return math.Round(value*100) / 100
}
