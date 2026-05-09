package fsrs_test

import (
	"math"
	"testing"
	"time"

	"github.com/jharaxus/rosetta/fsrs"
)

var (
	epoch = time.Date(2022, 11, 29, 12, 30, 0, 0, time.UTC)
)

func newNoFuzz() *fsrs.Scheduler {
	s := fsrs.NewScheduler()
	s.EnableFuzzing = false
	return s
}

// TestIntervalSequence is the primary regression test derived from the official
// py-fsrs test suite (test_review_card). The expected interval history is the
// authoritative reference value from the Python implementation.
func TestIntervalSequence(t *testing.T) {
	s := newNoFuzz()
	card := fsrs.NewCard()
	now := epoch

	ratings := []fsrs.Rating{
		fsrs.Good, fsrs.Good, fsrs.Good, fsrs.Good, fsrs.Good, fsrs.Good,
		fsrs.Again, fsrs.Again,
		fsrs.Good, fsrs.Good, fsrs.Good, fsrs.Good, fsrs.Good,
	}
	want := []int{0, 2, 11, 46, 163, 498, 0, 0, 2, 4, 7, 12, 21}

	for i, rating := range ratings {
		card = s.ReviewCard(card, rating, now)
		got := int(card.Due.Sub(card.LastReview).Hours() / 24)
		if got != want[i] {
			t.Errorf("review %d (rating=%v): interval = %d, want %d", i, rating, got, want[i])
		}
		now = card.Due
	}
}

// TestMemoState validates numerical precision against py-fsrs test_memo_state.
// Reference values: stability ≈ 53.62691, difficulty ≈ 6.3574867.
func TestMemoState(t *testing.T) {
	s := fsrs.NewScheduler()
	card := fsrs.NewCard()
	now := epoch

	type step struct {
		rating fsrs.Rating
		delay  int // days to add before this review
	}
	steps := []step{
		{fsrs.Again, 0},
		{fsrs.Good, 0},
		{fsrs.Good, 1},
		{fsrs.Good, 3},
		{fsrs.Good, 8},
		{fsrs.Good, 21},
	}

	for _, st := range steps {
		now = now.Add(time.Duration(st.delay) * 24 * time.Hour)
		card = s.ReviewCard(card, st.rating, now)
	}

	const tol = 1e-3
	if math.Abs(card.Stability-53.62691) > tol {
		t.Errorf("stability = %.5f, want ≈ 53.62691", card.Stability)
	}
	if math.Abs(card.Difficulty-6.3574867) > tol {
		t.Errorf("difficulty = %.7f, want ≈ 6.3574867", card.Difficulty)
	}
}

// TestRetrievability validates the forgetting-curve formula.
func TestRetrievability(t *testing.T) {
	s := fsrs.NewScheduler()

	// New card (never reviewed) → 0
	card := fsrs.NewCard()
	if r := s.Retrievability(card, epoch); r != 0 {
		t.Errorf("new card retrievability = %f, want 0", r)
	}

	// Card reviewed now → 1.0 (elapsed = 0)
	card.Stability = 10
	card.LastReview = epoch
	card.State = fsrs.Review
	if r := s.Retrievability(card, epoch); math.Abs(r-1.0) > 1e-9 {
		t.Errorf("elapsed=0 retrievability = %f, want 1.0", r)
	}

	// Card with S=10 after 10 days: R = (1 + FACTOR*10/10)^DECAY
	// Just check it's between 0 and 1 and close to the retention target.
	r10 := s.Retrievability(card, epoch.Add(10*24*time.Hour))
	if r10 <= 0 || r10 >= 1 {
		t.Errorf("retrievability after 10 days out of range: %f", r10)
	}

	// After stability days the retrievability should be approximately 0.9
	stab := 50.0
	card.Stability = stab
	rAtStab := s.Retrievability(card, epoch.Add(time.Duration(stab*24)*time.Hour))
	if math.Abs(rAtStab-0.9) > 0.01 {
		t.Errorf("retrievability at stability days = %f, want ≈ 0.9", rAtStab)
	}
}

// TestInitialValues checks that first-review stability and difficulty match
// the formula values for each rating.
func TestInitialValues(t *testing.T) {
	s := fsrs.NewScheduler()
	w := s.Weights

	ratings := []fsrs.Rating{fsrs.Again, fsrs.Hard, fsrs.Good, fsrs.Easy}
	for _, rating := range ratings {
		card := fsrs.NewCard()
		card = s.ReviewCard(card, rating, epoch)

		wantS := w[int(rating)-1]
		if math.Abs(card.Stability-wantS) > 1e-9 {
			t.Errorf("rating=%v: initial stability = %f, want %f", rating, card.Stability, wantS)
		}

		wantD := w[4] - math.Exp(w[5]*float64(rating-1)) + 1
		wantD = math.Max(1, math.Min(10, wantD))
		if math.Abs(card.Difficulty-wantD) > 1e-9 {
			t.Errorf("rating=%v: initial difficulty = %f, want %f", rating, card.Difficulty, wantD)
		}
	}
}

// TestRepeatedEasyCollapsesDifficulty mirrors py-fsrs test_repeated_correct_reviews.
// After many consecutive Easy reviews the difficulty must converge to 1.0.
// We start from a Good first-review (difficulty ≈ 2.12) to exercise mean-reversion.
func TestRepeatedEasyCollapsesDifficulty(t *testing.T) {
	s := fsrs.NewScheduler()
	card := fsrs.NewCard()
	now := epoch

	// First review: Good → non-trivial starting difficulty
	card = s.ReviewCard(card, fsrs.Good, now)
	now = card.Due

	for i := 0; i < 10; i++ {
		card = s.ReviewCard(card, fsrs.Easy, now)
		now = card.Due
	}

	if math.Abs(card.Difficulty-1.0) > 1e-9 {
		t.Errorf("difficulty after Good+10 Easy reviews = %f, want 1.0", card.Difficulty)
	}
}

// TestShortTermHardDoesNotDecreaseStability verifies that a Hard rating during a
// same-day (intra-step) review never reduces stability. Reference: go-fsrs arithmetic.go
// uses `r >= Hard` as the floor guard, meaning Hard, Good, and Easy all floor sinc to 1.0.
func TestShortTermHardDoesNotDecreaseStability(t *testing.T) {
	s := newNoFuzz()
	card := fsrs.NewCard()
	now := epoch

	// First review: Good — establishes non-zero stability
	card = s.ReviewCard(card, fsrs.Good, now)
	stabilityBeforeHard := card.Stability

	// Same-day Hard review — stability must not decrease
	card = s.ReviewCard(card, fsrs.Hard, now)
	if card.Stability < stabilityBeforeHard {
		t.Errorf("Hard same-day review decreased stability: %.6f → %.6f (want ≥ %.6f)",
			stabilityBeforeHard, card.Stability, stabilityBeforeHard)
	}
}

// TestLearningStepTransitions verifies intra-day step progression.
func TestLearningStepTransitions(t *testing.T) {
	s := newNoFuzz()
	now := epoch

	// Again at step 0 → stays Learning step 0, due in ≈ 1 min
	{
		card := fsrs.NewCard()
		card = s.ReviewCard(card, fsrs.Again, now)
		if card.State != fsrs.Learning {
			t.Errorf("Again: state = %v, want Learning", card.State)
		}
		if card.Step != 0 {
			t.Errorf("Again: step = %d, want 0", card.Step)
		}
		secs := card.Due.Sub(now).Seconds()
		if math.Abs(secs-60) > 5 {
			t.Errorf("Again: due in %.0fs, want ≈ 60s", secs)
		}
	}

	// Hard at step 0 (2 steps) → stays step 0, due ≈ average(1min, 10min) = 5.5min
	{
		card := fsrs.NewCard()
		card = s.ReviewCard(card, fsrs.Hard, now)
		if card.State != fsrs.Learning {
			t.Errorf("Hard: state = %v, want Learning", card.State)
		}
		if card.Step != 0 {
			t.Errorf("Hard: step = %d, want 0", card.Step)
		}
		secs := card.Due.Sub(now).Seconds()
		want := (60.0 + 600.0) / 2.0 // 330s
		if math.Abs(secs-want) > 5 {
			t.Errorf("Hard: due in %.0fs, want ≈ %.0fs", secs, want)
		}
	}

	// Good at step 0 → advances to step 1, due ≈ 10 min
	{
		card := fsrs.NewCard()
		card = s.ReviewCard(card, fsrs.Good, now)
		if card.State != fsrs.Learning {
			t.Errorf("Good (step0): state = %v, want Learning", card.State)
		}
		if card.Step != 1 {
			t.Errorf("Good (step0): step = %d, want 1", card.Step)
		}
		secs := card.Due.Sub(now).Seconds()
		if math.Abs(secs-600) > 5 {
			t.Errorf("Good (step0): due in %.0fs, want ≈ 600s", secs)
		}

		// Good at step 1 (last) → Review state
		card = s.ReviewCard(card, fsrs.Good, card.Due)
		if card.State != fsrs.Review {
			t.Errorf("Good (step1, last): state = %v, want Review", card.State)
		}
		ivl := int(card.Due.Sub(card.LastReview).Hours() / 24)
		if ivl < 1 {
			t.Errorf("Good (step1→Review): interval = %d days, want ≥ 1", ivl)
		}
	}

	// Easy at step 0 → immediately Review
	{
		card := fsrs.NewCard()
		card = s.ReviewCard(card, fsrs.Easy, now)
		if card.State != fsrs.Review {
			t.Errorf("Easy: state = %v, want Review", card.State)
		}
		ivl := int(card.Due.Sub(card.LastReview).Hours() / 24)
		if ivl < 1 {
			t.Errorf("Easy: interval = %d days, want ≥ 1", ivl)
		}
	}
}

// TestRelearningTransition exercises Review→Relearning→Review.
func TestRelearningTransition(t *testing.T) {
	s := newNoFuzz()
	card := fsrs.NewCard()
	now := epoch

	// Pass through Learning into Review
	card = s.ReviewCard(card, fsrs.Good, now)
	now = card.Due
	card = s.ReviewCard(card, fsrs.Good, now)
	now = card.Due
	if card.State != fsrs.Review {
		t.Fatalf("expected Review state, got %v", card.State)
	}

	// Again in Review → Relearning
	card = s.ReviewCard(card, fsrs.Again, now)
	if card.State != fsrs.Relearning {
		t.Errorf("Again in Review: state = %v, want Relearning", card.State)
	}
	if card.Step != 0 {
		t.Errorf("Relearning: step = %d, want 0", card.Step)
	}
	secs := card.Due.Sub(now).Seconds()
	if math.Abs(secs-600) > 5 {
		t.Errorf("Relearning due in %.0fs, want ≈ 600s (10min step)", secs)
	}

	// Good in Relearning at last step → Review
	now = card.Due
	card = s.ReviewCard(card, fsrs.Good, now)
	if card.State != fsrs.Review {
		t.Errorf("Good in Relearning (last step): state = %v, want Review", card.State)
	}
	ivl := int(card.Due.Sub(card.LastReview).Hours() / 24)
	if ivl < 1 {
		t.Errorf("After Relearning: interval = %d days, want ≥ 1", ivl)
	}
}

// TestStabilityLowerBound ensures stability never falls below StabilityMin.
func TestStabilityLowerBound(t *testing.T) {
	s := fsrs.NewScheduler()
	card := fsrs.NewCard()
	now := epoch

	for i := 0; i < 1000; i++ {
		now = card.Due.Add(24 * time.Hour)
		card = s.ReviewCard(card, fsrs.Again, now)
		if card.Stability < fsrs.StabilityMin {
			t.Fatalf("review %d: stability %f < StabilityMin %f", i, card.Stability, fsrs.StabilityMin)
		}
	}
}

// TestMaximumInterval ensures the scheduler never exceeds MaxInterval.
func TestMaximumInterval(t *testing.T) {
	s := newNoFuzz()
	s.MaxInterval = 100
	card := fsrs.NewCard()
	now := epoch

	for i := 0; i < 10; i++ {
		card = s.ReviewCard(card, fsrs.Easy, now)
		now = card.Due
		ivl := int(card.Due.Sub(card.LastReview).Hours() / 24)
		if ivl > s.MaxInterval {
			t.Errorf("review %d: interval %d > MaxInterval %d", i, ivl, s.MaxInterval)
		}
	}
}
