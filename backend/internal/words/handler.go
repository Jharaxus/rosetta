package words

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/jharaxus/rosetta/fsrs"
	"github.com/jharaxus/rosetta/internal/auth"
	"github.com/jharaxus/rosetta/internal/db"
	"github.com/jharaxus/rosetta/internal/model"
)

// scheduler is stateless after construction and safe for concurrent use.
var scheduler = fsrs.NewScheduler()

type Handler struct {
	queries *db.Queries
}

func NewHandler(queries *db.Queries) *Handler {
	return &Handler{queries: queries}
}

// GetFlashCard returns the oldest due card for the authenticated user.
func (h *Handler) GetFlashCard(c *gin.Context) {
	su, ok := auth.GetSessionUser(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "context_missing_user"})
		return
	}

	cw, err := h.queries.GetNextDueCard(c.Request.Context(), su.ID)
	if errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": "no_cards_due"})
		return
	}
	if err != nil {
		slog.Error("get next due card", "user_id", su.ID, "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":             cw.Card.WordID,
		"french":         cw.French,
		"german":         cw.German,
		"assimil_number": cw.AssimilNumber,
		"category":       cw.Category,
		"is_regular":     cw.IsRegular,
	})
}

type reviewRequest struct {
	Rating int `json:"rating" binding:"required,min=1,max=4"`
}

// PostReview applies a user rating to the card and reschedules it via FSRS.
func (h *Handler) PostReview(c *gin.Context) {
	su, ok := auth.GetSessionUser(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "context_missing_user"})
		return
	}

	wordID, err := uuid.Parse(c.Param("word_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_word_id"})
		return
	}

	var req reviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "validation_error", "detail": err.Error()})
		return
	}

	card, err := h.queries.GetCard(c.Request.Context(), su.ID, wordID)
	if errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": "card_not_found"})
		return
	}
	if err != nil {
		slog.Error("get card for review", "user_id", su.ID, "word_id", wordID, "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
		return
	}

	updated := scheduler.ReviewCard(cardToFSRS(card), fsrs.Rating(req.Rating), time.Now().UTC())

	if err := h.queries.UpdateCard(c.Request.Context(), fsrsCardToModel(updated, su.ID, wordID)); err != nil {
		slog.Error("update card after review", "user_id", su.ID, "word_id", wordID, "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
		return
	}

	c.Status(http.StatusNoContent)
}

// GetWritingFlashCard returns the oldest due writing card for the authenticated user.
func (h *Handler) GetWritingFlashCard(c *gin.Context) {
	su, ok := auth.GetSessionUser(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "context_missing_user"})
		return
	}

	cw, err := h.queries.GetNextDueWritingCard(c.Request.Context(), su.ID)
	if errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": "no_cards_due"})
		return
	}
	if err != nil {
		slog.Error("get next due writing card", "user_id", su.ID, "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":             cw.Card.WordID,
		"french":         cw.French,
		"german":         cw.German,
		"assimil_number": cw.AssimilNumber,
		"category":       cw.Category,
		"is_regular":     cw.IsRegular,
	})
}

// PostWritingReview applies a frontend-computed rating to the writing card and reschedules it via FSRS.
func (h *Handler) PostWritingReview(c *gin.Context) {
	su, ok := auth.GetSessionUser(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "context_missing_user"})
		return
	}

	wordID, err := uuid.Parse(c.Param("word_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_word_id"})
		return
	}

	var req reviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "validation_error", "detail": err.Error()})
		return
	}

	card, err := h.queries.GetWritingCard(c.Request.Context(), su.ID, wordID)
	if errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": "card_not_found"})
		return
	}
	if err != nil {
		slog.Error("get writing card for review", "user_id", su.ID, "word_id", wordID, "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
		return
	}

	updated := scheduler.ReviewCard(cardToFSRS(card), fsrs.Rating(req.Rating), time.Now().UTC())

	if err := h.queries.UpdateWritingCard(c.Request.Context(), fsrsCardToModel(updated, su.ID, wordID)); err != nil {
		slog.Error("update writing card after review", "user_id", su.ID, "word_id", wordID, "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
		return
	}

	c.Status(http.StatusNoContent)
}

func cardToFSRS(c model.Card) fsrs.Card {
	lastReview := time.Time{}
	if c.LastReview != nil {
		lastReview = *c.LastReview
	}
	return fsrs.Card{
		Stability:  c.Stability,
		Difficulty: c.Difficulty,
		State:      fsrs.State(c.State),
		Step:       c.Step,
		Due:        c.Due,
		LastReview: lastReview,
		Reps:       c.Reps,
		Lapses:     c.Lapses,
	}
}

func fsrsCardToModel(fc fsrs.Card, userID, wordID uuid.UUID) model.Card {
	lr := fc.LastReview
	return model.Card{
		UserID:     userID,
		WordID:     wordID,
		Stability:  fc.Stability,
		Difficulty: fc.Difficulty,
		State:      int(fc.State),
		Step:       fc.Step,
		Due:        fc.Due,
		LastReview: &lr,
		Reps:       fc.Reps,
		Lapses:     fc.Lapses,
	}
}
