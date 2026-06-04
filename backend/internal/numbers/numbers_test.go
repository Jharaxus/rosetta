package numbers

import (
	"math"
	"math/rand/v2"
	"testing"

	"github.com/jharaxus/rosetta/internal/model"
)

// seededRNG returns a deterministic *rand.Rand for reproducible tests.
func seededRNG(seed uint64) *rand.Rand {
	return rand.New(rand.NewPCG(seed, 0))
}

// allZeroStats returns 10 DigitStat rows with zero successes.
func allZeroStats() []model.DigitStat {
	stats := make([]model.DigitStat, 10)
	for i := range stats {
		stats[i] = model.DigitStat{Digit: i}
	}
	return stats
}

// allSuccessOnOneDigit returns stats where `winner` holds all successes.
// This sets weight[winner] = 0, so weightedSample can never pick it.
func allSuccessOnOneDigit(winner int) []model.DigitStat {
	stats := allZeroStats()
	stats[winner].Successes = 1000
	return stats
}

// ── digitCounts ───────────────────────────────────────────────────────────────

func TestDigitCounts(t *testing.T) {
	cases := []struct {
		n    int
		want [10]int
	}{
		{0, [10]int{1, 0, 0, 0, 0, 0, 0, 0, 0, 0}},
		{1, [10]int{0, 1, 0, 0, 0, 0, 0, 0, 0, 0}},
		{9, [10]int{0, 0, 0, 0, 0, 0, 0, 0, 0, 1}},
		{10, [10]int{1, 1, 0, 0, 0, 0, 0, 0, 0, 0}},
		{100, [10]int{2, 1, 0, 0, 0, 0, 0, 0, 0, 0}},
		{113, [10]int{0, 2, 0, 1, 0, 0, 0, 0, 0, 0}},
		{5555, [10]int{0, 0, 0, 0, 0, 4, 0, 0, 0, 0}},
		{9876543210, [10]int{1, 1, 1, 1, 1, 1, 1, 1, 1, 1}},
	}

	for _, tc := range cases {
		got := digitCounts(tc.n)
		if got != tc.want {
			t.Errorf("digitCounts(%d) = %v, want %v", tc.n, got, tc.want)
		}
	}
}

// ── weightedSample ────────────────────────────────────────────────────────────

// TestWeightedSample_ConcentratedWeight verifies that when only one weight is
// nonzero, every draw returns that index — regardless of the RNG value.
// This holds because the CDF only advances at nonzero weights.
func TestWeightedSample_ConcentratedWeight(t *testing.T) {
	for _, k := range []int{0, 3, 9} {
		weights := make([]float64, 10)
		weights[k] = 1.0

		for _, seed := range []uint64{0, 1, 42, 999, 12345} {
			rng := seededRNG(seed)
			for range 20 {
				got := weightedSample(weights, rng)
				if got != k {
					t.Errorf("concentrated weight at %d, seed %d: got %d", k, seed, got)
				}
			}
		}
	}
}

// TestWeightedSample_ResultInRange verifies that equal weights always produce
// an index in [0, 9].
func TestWeightedSample_ResultInRange(t *testing.T) {
	weights := make([]float64, 10)
	for i := range weights {
		weights[i] = 1.0
	}
	rng := seededRNG(42)
	for range 50 {
		got := weightedSample(weights, rng)
		if got < 0 || got > 9 {
			t.Errorf("weightedSample returned %d, want [0, 9]", got)
		}
	}
}

// TestWeightedSample_ZeroWeightNeverChosen verifies that a zero-weight index
// is never returned.
func TestWeightedSample_ZeroWeightNeverChosen(t *testing.T) {
	// Weight on index 2 only; index 7 has weight 0.
	weights := make([]float64, 10)
	weights[2] = 1.0

	for _, seed := range []uint64{0, 1, 42, 999} {
		rng := seededRNG(seed)
		for range 30 {
			got := weightedSample(weights, rng)
			if got != 2 {
				t.Errorf("seed %d: expected 2, got %d", seed, got)
			}
		}
	}
}

// ── sampleNumber ─────────────────────────────────────────────────────────────

// TestSampleNumber_SingleDigit_InRange verifies that 1-digit results stay in [0, 9].
func TestSampleNumber_SingleDigit_InRange(t *testing.T) {
	stats := allZeroStats()
	rng := seededRNG(42)
	for range 100 {
		got := sampleNumber(stats, 1, rng)
		if got < 0 || got > 9 {
			t.Errorf("sampleNumber(stats, 1): got %d, want [0, 9]", got)
		}
	}
}

// TestSampleNumber_MultiDigit_NoLeadingZero verifies the leading-zero invariant
// across all supported digit counts.
func TestSampleNumber_MultiDigit_NoLeadingZero(t *testing.T) {
	stats := allZeroStats()
	for _, numDigits := range []int{2, 3, 5, 10} {
		min := int(math.Pow10(numDigits - 1))
		rng := seededRNG(42)
		for range 100 {
			got := sampleNumber(stats, numDigits, rng)
			if got < min {
				t.Errorf("numDigits=%d: got %d which has fewer than %d digits (leading zero)", numDigits, got, numDigits)
			}
		}
	}
}

// TestSampleNumber_MultiDigit_CorrectRange verifies that N-digit numbers fall
// in [10^(N-1), 10^N - 1].
func TestSampleNumber_MultiDigit_CorrectRange(t *testing.T) {
	stats := allZeroStats()
	for _, numDigits := range []int{2, 3, 5, 7} {
		min := int(math.Pow10(numDigits - 1))
		max := int(math.Pow10(numDigits)) - 1
		rng := seededRNG(99)
		for range 100 {
			got := sampleNumber(stats, numDigits, rng)
			if got < min || got > max {
				t.Errorf("numDigits=%d: got %d, want [%d, %d]", numDigits, got, min, max)
			}
		}
	}
}

// TestSampleNumber_ExcludedDigit_NeverChosen verifies that a digit with
// weight = 0 (all successes) is never sampled.
// This is deterministic: weight[winner] = 0 → CDF never advances at winner.
func TestSampleNumber_ExcludedDigit_NeverChosen(t *testing.T) {
	for _, winner := range []int{0, 5, 9} {
		stats := allSuccessOnOneDigit(winner)
		for _, seed := range []uint64{0, 1, 42, 999} {
			rng := seededRNG(seed)
			for range 50 {
				got := sampleNumber(stats, 1, rng)
				if got == winner {
					t.Errorf("winner=%d seed=%d: digit with weight 0 was sampled", winner, seed)
				}
			}
		}
	}
}

// TestSampleNumber_AllZeroStats_NoPanic verifies the uniform-fallback branch
// (total == 0) doesn't panic and returns a valid result.
func TestSampleNumber_AllZeroStats_NoPanic(t *testing.T) {
	stats := allZeroStats()
	for _, numDigits := range []int{1, 4, 10} {
		rng := seededRNG(7)
		got := sampleNumber(stats, numDigits, rng)
		if numDigits == 1 {
			if got < 0 || got > 9 {
				t.Errorf("numDigits=1: got %d", got)
			}
		} else {
			min := int(math.Pow10(numDigits - 1))
			max := int(math.Pow10(numDigits)) - 1
			if got < min || got > max {
				t.Errorf("numDigits=%d: got %d, want [%d, %d]", numDigits, got, min, max)
			}
		}
	}
}
