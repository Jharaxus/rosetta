package numbers

import (
	"math/rand/v2"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"

	"github.com/jharaxus/rosetta/internal/auth"
	"github.com/jharaxus/rosetta/internal/db"
	"github.com/jharaxus/rosetta/internal/model"
)

type Handler struct {
	queries *db.Queries
}

func NewHandler(queries *db.Queries) *Handler {
	return &Handler{queries: queries}
}

// GetPracticeNumber samples a random number of the requested digit count using
// per-digit success weights and returns it as { "number": N }.
func (h *Handler) GetPracticeNumber(c *gin.Context) {
	user, ok := auth.GetSessionUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	digitsStr := c.DefaultQuery("digits", "1")
	numDigits, err := strconv.Atoi(digitsStr)
	if err != nil || numDigits < 1 || numDigits > 10 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "digits must be between 1 and 10"})
		return
	}

	stats, err := h.queries.GetDigitStats(c.Request.Context(), user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	rng := rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), 0))
	number := sampleNumber(stats, numDigits, rng)
	c.JSON(http.StatusOK, gin.H{"number": number})
}

// PostDigitSuccess increments the success counter for each digit in the submitted number.
func (h *Handler) PostDigitSuccess(c *gin.Context) {
	user, ok := auth.GetSessionUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var body struct {
		Number int `json:"number" binding:"min=0"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "number is required"})
		return
	}

	if err := h.queries.IncrementDigitSuccesses(c.Request.Context(), user.ID, body.Number); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	c.Status(http.StatusNoContent)
}

// GetNextFailure returns a random number from the user's failure list.
// Returns 404 with code "no_failures" when the list is empty.
func (h *Handler) GetNextFailure(c *gin.Context) {
	user, ok := auth.GetSessionUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	number, err := h.queries.GetRandomNumberFailure(c.Request.Context(), user.ID)
	if err == pgx.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"code": "no_failures"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"number": number})
}

// PostFailure adds a number to the user's failure list.
func (h *Handler) PostFailure(c *gin.Context) {
	user, ok := auth.GetSessionUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var body struct {
		Number int `json:"number" binding:"min=0"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "number is required"})
		return
	}

	if err := h.queries.AddNumberFailure(c.Request.Context(), user.ID, body.Number); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	c.Status(http.StatusNoContent)
}

// DeleteFailure removes a number from the user's failure list.
func (h *Handler) DeleteFailure(c *gin.Context) {
	user, ok := auth.GetSessionUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	numberStr := c.Param("number")
	number, err := strconv.Atoi(numberStr)
	if err != nil || number < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid number"})
		return
	}

	if err := h.queries.RemoveNumberFailure(c.Request.Context(), user.ID, number); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	c.Status(http.StatusNoContent)
}

// GetSettings returns the user's number practice settings.
func (h *Handler) GetSettings(c *gin.Context) {
	user, ok := auth.GetSessionUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	settings, err := h.queries.GetOrCreateUserSettings(c.Request.Context(), user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"number_digit_size": settings.NumberDigitSize})
}

// UpdateSettings saves the user's preferred digit count.
func (h *Handler) UpdateSettings(c *gin.Context) {
	user, ok := auth.GetSessionUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var body struct {
		NumberDigitSize int `json:"number_digit_size" binding:"required,min=1,max=10"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "number_digit_size must be between 1 and 10"})
		return
	}

	if err := h.queries.UpdateNumberDigitSize(c.Request.Context(), user.ID, body.NumberDigitSize); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	c.Status(http.StatusNoContent)
}

// digitCounts returns how many times each digit 0–9 appears in n.
// Treats n == 0 as a single occurrence of digit 0.
func digitCounts(n int) [10]int {
	var counts [10]int
	if n == 0 {
		counts[0]++
		return counts
	}
	for n > 0 {
		counts[n%10]++
		n /= 10
	}
	return counts
}

// sampleNumber independently samples numDigits digits using success-weighted
// probabilities and assembles them into an integer. Digit 0 is excluded from
// the leading position in multi-digit numbers to avoid a leading zero.
func sampleNumber(stats []model.DigitStat, numDigits int, rng *rand.Rand) int {
	successes := make([]int, 10)
	total := 0
	for _, s := range stats {
		if s.Digit >= 0 && s.Digit < 10 {
			successes[s.Digit] = s.Successes
			total += s.Successes
		}
	}

	weights := make([]float64, 10)
	for i := range weights {
		if total == 0 {
			weights[i] = 1.0
		} else {
			weights[i] = 1.0 - float64(successes[i])/float64(total)
		}
	}

	result := 0
	for pos := 0; pos < numDigits; pos++ {
		w := weights
		if pos == 0 && numDigits > 1 {
			// Copy weights and zero out digit 0 to prevent a leading zero.
			w = make([]float64, 10)
			copy(w, weights)
			w[0] = 0
		}
		result = result*10 + weightedSample(w, rng)
	}
	return result
}

// weightedSample picks an index from weights using a CDF draw.
// weights need not be normalised.
func weightedSample(weights []float64, rng *rand.Rand) int {
	sum := 0.0
	for _, w := range weights {
		sum += w
	}
	r := rng.Float64() * sum
	cumulative := 0.0
	for i, w := range weights {
		cumulative += w
		if r < cumulative {
			return i
		}
	}
	// Fallback for floating-point rounding: return the last nonzero-weight index.
	for i := len(weights) - 1; i >= 0; i-- {
		if weights[i] > 0 {
			return i
		}
	}
	return 0
}
