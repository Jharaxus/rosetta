package words

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"

	"github.com/jharaxus/rosetta/internal/auth"
	"github.com/jharaxus/rosetta/internal/db"
)

type Handler struct {
	queries *db.Queries
}

func NewHandler(queries *db.Queries) *Handler {
	return &Handler{queries: queries}
}

func (h *Handler) GetFlashCard(c *gin.Context) {
	su, ok := auth.GetSessionUser(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "context_missing_user"})
		return
	}

	user, err := h.queries.GetUserByID(c.Request.Context(), su.ID)
	if err != nil {
		slog.Error("get user for flashcard", "id", su.ID, "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
		return
	}

	word, err := h.queries.GetRandomWordForUser(c.Request.Context(), user.AssimilNumber)
	if errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": "no_cards_available"})
		return
	}
	if err != nil {
		slog.Error("get random word", "assimil_number", user.AssimilNumber, "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":             word.ID,
		"french":         word.French,
		"german":         word.German,
		"assimil_number": word.AssimilNumber,
		"category":       word.Category,
		"is_regular":     word.IsRegular,
	})
}
