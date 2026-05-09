// Package fsrs implements the FSRS v6 spaced-repetition scheduling algorithm.
// Reference: https://github.com/open-spaced-repetition/py-fsrs
package fsrs

import (
	"math"
	"time"
)

// StabilityMin is the lower bound for card stability (days).
const StabilityMin = 0.001

// State represents the current learning state of a card.
type State int

const (
	Learning   State = 1
	Review     State = 2
	Relearning State = 3
)

func (s State) String() string {
	switch s {
	case Learning:
		return "Learning"
	case Review:
		return "Review"
	case Relearning:
		return "Relearning"
	default:
		return "Unknown"
	}
}

// Rating represents the outcome of a review.
type Rating int

const (
	Again Rating = 1
	Hard  Rating = 2
	Good  Rating = 3
	Easy  Rating = 4
)

func (r Rating) String() string {
	switch r {
	case Again:
		return "Again"
	case Hard:
		return "Hard"
	case Good:
		return "Good"
	case Easy:
		return "Easy"
	default:
		return "Unknown"
	}
}

// Card holds the scheduling state for a single flashcard.
// Stability=0 and LastReview.IsZero() indicate a brand-new, never-reviewed card.
type Card struct {
	Stability  float64
	Difficulty float64
	State      State
	Step       int       // index into learning/relearning steps (0-based)
	Due        time.Time
	LastReview time.Time // zero value = never reviewed
	Reps       int
	Lapses     int
}

// NewCard returns a card in the Learning state, due immediately.
func NewCard() Card {
	return Card{
		State: Learning,
		Due:   time.Now().UTC(),
	}
}

// DefaultWeights are the FSRS v6 default model parameters.
var DefaultWeights = [21]float64{
	0.212,  // w[0]  initial stability – Again
	1.2931, // w[1]  initial stability – Hard
	2.3065, // w[2]  initial stability – Good
	8.2956, // w[3]  initial stability – Easy
	6.4133, // w[4]  difficulty base
	0.8334, // w[5]  difficulty exponent
	3.0194, // w[6]  difficulty change rate
	0.001,  // w[7]  mean-reversion weight
	1.8722, // w[8]  recall stability – exponential base
	0.1666, // w[9]  recall stability – saturation exponent
	0.796,  // w[10] recall stability – retrievability exponent
	1.4835, // w[11] forget stability – scaling
	0.0614, // w[12] forget stability – difficulty exponent
	0.2629, // w[13] forget stability – stability exponent
	1.6483, // w[14] forget stability – retrievability exponent
	0.6014, // w[15] Hard penalty
	1.8729, // w[16] Easy bonus
	0.5425, // w[17] short-term – grade impact
	0.0912, // w[18] short-term – grade offset
	0.0658, // w[19] short-term – saturation exponent
	0.1542, // w[20] decay magnitude
}

// Scheduler drives the FSRS v6 algorithm.
type Scheduler struct {
	Weights          [21]float64
	DesiredRetention float64
	MaxInterval      int
	LearningSteps    []time.Duration
	RearningSteps    []time.Duration // relearning steps
	EnableFuzzing    bool

	decay  float64 // -Weights[20]
	factor float64 // pow(0.9, 1/decay) - 1
}

// NewScheduler returns a Scheduler with FSRS v6 defaults.
func NewScheduler() *Scheduler {
	s := &Scheduler{
		Weights:          DefaultWeights,
		DesiredRetention: 0.9,
		MaxInterval:      36500,
		LearningSteps:    []time.Duration{1 * time.Minute, 10 * time.Minute},
		RearningSteps:    []time.Duration{10 * time.Minute},
		EnableFuzzing:    false,
	}
	s.updateDerivedConstants()
	return s
}

func (s *Scheduler) updateDerivedConstants() {
	s.decay = -s.Weights[20]
	s.factor = math.Pow(0.9, 1.0/s.decay) - 1
}

// Retrievability returns the predicted recall probability for card at time now.
// Returns 0 for a card that has never been reviewed.
func (s *Scheduler) Retrievability(card Card, now time.Time) float64 {
	if card.LastReview.IsZero() || card.Stability == 0 {
		return 0
	}
	elapsed := math.Max(0, now.Sub(card.LastReview).Hours()/24)
	return math.Pow(1+s.factor*elapsed/card.Stability, s.decay)
}

// ReviewCard processes a review of card with the given rating at time now,
// returning the updated card. The original card is not modified.
func (s *Scheduler) ReviewCard(card Card, rating Rating, now time.Time) Card {
	// Compute elapsed days since last review (nil-safe).
	var elapsedDays float64
	if !card.LastReview.IsZero() {
		elapsedDays = now.Sub(card.LastReview).Hours() / 24
	}
	sameDay := !card.LastReview.IsZero() && elapsedDays < 1

	switch card.State {
	case Learning:
		s.updateLearning(&card, rating, sameDay, now)
	case Review:
		s.updateReview(&card, rating, sameDay, now)
	case Relearning:
		s.updateRelearning(&card, rating, sameDay, now)
	}

	card.LastReview = now
	card.Reps++
	return card
}

// ─── Learning ────────────────────────────────────────────────────────────────

func (s *Scheduler) updateLearning(card *Card, rating Rating, sameDay bool, now time.Time) {
	isFirstReview := card.Stability == 0

	if isFirstReview {
		card.Stability = s.initialStability(rating)
		card.Difficulty = s.initialDifficulty(rating)
	} else if sameDay {
		card.Stability = s.shortTermStability(card.Stability, rating)
		card.Difficulty = s.nextDifficulty(card.Difficulty, rating)
	} else {
		r := s.Retrievability(*card, now)
		card.Stability = s.nextStability(card.Difficulty, card.Stability, r, rating)
		card.Difficulty = s.nextDifficulty(card.Difficulty, rating)
	}

	steps := s.LearningSteps
	var next time.Duration

	if len(steps) == 0 || (card.Step >= len(steps) && rating != Again) {
		card.State = Review
		card.Step = 0
		next = time.Duration(s.nextInterval(card.Stability)) * 24 * time.Hour
	} else {
		switch rating {
		case Again:
			card.Step = 0
			next = steps[0]
		case Hard:
			// step stays the same
			if card.Step == 0 && len(steps) == 1 {
				next = time.Duration(float64(steps[0]) * 1.5)
			} else if card.Step == 0 && len(steps) >= 2 {
				next = (steps[0] + steps[1]) / 2
			} else {
				next = steps[card.Step]
			}
		case Good:
			if card.Step+1 == len(steps) {
				card.State = Review
				card.Step = 0
				next = time.Duration(s.nextInterval(card.Stability)) * 24 * time.Hour
			} else {
				card.Step++
				next = steps[card.Step]
			}
		case Easy:
			card.State = Review
			card.Step = 0
			next = time.Duration(s.nextInterval(card.Stability)) * 24 * time.Hour
		}
	}

	card.Due = now.Add(next)
}

// ─── Review ──────────────────────────────────────────────────────────────────

func (s *Scheduler) updateReview(card *Card, rating Rating, sameDay bool, now time.Time) {
	if sameDay {
		card.Stability = s.shortTermStability(card.Stability, rating)
	} else {
		r := s.Retrievability(*card, now)
		card.Stability = s.nextStability(card.Difficulty, card.Stability, r, rating)
	}
	card.Difficulty = s.nextDifficulty(card.Difficulty, rating)

	switch rating {
	case Again:
		card.Lapses++
		if len(s.RearningSteps) == 0 {
			next := time.Duration(s.nextInterval(card.Stability)) * 24 * time.Hour
			card.Due = now.Add(next)
		} else {
			card.State = Relearning
			card.Step = 0
			card.Due = now.Add(s.RearningSteps[0])
		}
	default:
		next := time.Duration(s.nextInterval(card.Stability)) * 24 * time.Hour
		card.Due = now.Add(next)
	}
}

// ─── Relearning ──────────────────────────────────────────────────────────────

func (s *Scheduler) updateRelearning(card *Card, rating Rating, sameDay bool, now time.Time) {
	if sameDay {
		card.Stability = s.shortTermStability(card.Stability, rating)
		card.Difficulty = s.nextDifficulty(card.Difficulty, rating)
	} else {
		r := s.Retrievability(*card, now)
		card.Stability = s.nextStability(card.Difficulty, card.Stability, r, rating)
		card.Difficulty = s.nextDifficulty(card.Difficulty, rating)
	}

	steps := s.RearningSteps
	var next time.Duration

	if len(steps) == 0 || (card.Step >= len(steps) && rating != Again) {
		card.State = Review
		card.Step = 0
		next = time.Duration(s.nextInterval(card.Stability)) * 24 * time.Hour
	} else {
		switch rating {
		case Again:
			card.Step = 0
			next = steps[0]
		case Hard:
			if card.Step == 0 && len(steps) == 1 {
				next = time.Duration(float64(steps[0]) * 1.5)
			} else if card.Step == 0 && len(steps) >= 2 {
				next = (steps[0] + steps[1]) / 2
			} else {
				next = steps[card.Step]
			}
		case Good:
			if card.Step+1 == len(steps) {
				card.State = Review
				card.Step = 0
				next = time.Duration(s.nextInterval(card.Stability)) * 24 * time.Hour
			} else {
				card.Step++
				next = steps[card.Step]
			}
		case Easy:
			card.State = Review
			card.Step = 0
			next = time.Duration(s.nextInterval(card.Stability)) * 24 * time.Hour
		}
	}

	card.Due = now.Add(next)
}

// ─── Math helpers ─────────────────────────────────────────────────────────────

func (s *Scheduler) initialStability(rating Rating) float64 {
	return math.Max(0.1, math.Min(float64(s.MaxInterval), s.Weights[rating-1]))
}

func (s *Scheduler) initialDifficulty(rating Rating) float64 {
	d := s.Weights[4] - math.Exp(s.Weights[5]*float64(rating-1)) + 1
	return clampDifficulty(d)
}

// nextInterval returns the number of days until the next review.
func (s *Scheduler) nextInterval(stability float64) int {
	ivl := math.Round((stability / s.factor) * (math.Pow(s.DesiredRetention, 1.0/s.decay) - 1))
	return int(math.Max(1, math.Min(float64(s.MaxInterval), ivl)))
}

// shortTermStability applies same-day review adjustment (w[17..19]).
func (s *Scheduler) shortTermStability(stability float64, rating Rating) float64 {
	w := s.Weights
	mult := math.Exp(w[17]*(float64(rating)-3+w[18])) * math.Pow(stability, -w[19])
	if rating >= Hard {
		if mult < 1.0 {
			mult = 1.0
		}
	}
	return math.Max(stability*mult, StabilityMin)
}

// nextDifficulty applies mean-reversion difficulty update.
func (s *Scheduler) nextDifficulty(difficulty float64, rating Rating) float64 {
	w := s.Weights
	d0Easy := w[4] - math.Exp(w[5]*3) + 1 // initial difficulty for Easy (unclamped)
	delta := -w[6] * float64(rating-3)
	damped := (10.0 - difficulty) * delta / 9.0
	d := w[7]*d0Easy + (1-w[7])*(difficulty+damped)
	return clampDifficulty(d)
}

// nextStability dispatches to recall or forget stability based on rating.
func (s *Scheduler) nextStability(difficulty, stability, retrievability float64, rating Rating) float64 {
	if rating == Again {
		return s.nextForgetStability(difficulty, stability, retrievability)
	}
	return s.nextRecallStability(difficulty, stability, retrievability, rating)
}

// nextRecallStability computes post-review stability for Hard/Good/Easy.
func (s *Scheduler) nextRecallStability(difficulty, stability, retrievability float64, rating Rating) float64 {
	w := s.Weights
	hardPenalty := 1.0
	if rating == Hard {
		hardPenalty = w[15]
	}
	easyBonus := 1.0
	if rating == Easy {
		easyBonus = w[16]
	}
	s_ := stability * (1 +
		math.Exp(w[8])*
			(11-difficulty)*
			math.Pow(stability, -w[9])*
			(math.Exp((1-retrievability)*w[10])-1)*
			hardPenalty*easyBonus)
	return math.Max(s_, StabilityMin)
}

// nextForgetStability computes post-lapse stability for Again.
func (s *Scheduler) nextForgetStability(difficulty, stability, retrievability float64) float64 {
	w := s.Weights
	longTerm := w[11] *
		math.Pow(difficulty, -w[12]) *
		(math.Pow(stability+1, w[13]) - 1) *
		math.Exp((1-retrievability)*w[14])
	shortTerm := stability / math.Exp(w[17]*w[18])
	return math.Max(math.Min(longTerm, shortTerm), StabilityMin)
}

func clampDifficulty(d float64) float64 {
	return math.Max(1.0, math.Min(10.0, d))
}
