package model

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID            uuid.UUID `db:"id"`
	Subject       string    `db:"subject"`
	Email         string    `db:"email"`
	DisplayName   string    `db:"display_name"`
	AssimilNumber int       `db:"assimil_number"`
	CreatedAt     time.Time `db:"created_at"`
	UpdatedAt     time.Time `db:"updated_at"`
}

type LoginRecord struct {
	ID         uuid.UUID `db:"id"`
	UserID     uuid.UUID `db:"user_id"`
	LoggedInAt time.Time `db:"logged_in_at"`
	IPAddress  string    `db:"ip_address"`
	UserAgent  string    `db:"user_agent"`
	SessionID  string    `db:"session_id"`
}

type Word struct {
	ID            uuid.UUID `db:"id"`
	French        string    `db:"french"`
	German        string    `db:"german"`
	AssimilNumber int       `db:"assimil_number"`
	Category      string    `db:"category"`
	IsRegular     *bool     `db:"is_regular"` // nil for non-verbs
	CreatedAt     time.Time `db:"created_at"`
}

type Card struct {
	UserID     uuid.UUID
	WordID     uuid.UUID
	Stability  float64
	Difficulty float64
	State      int        // 1=Learning 2=Review 3=Relearning
	Step       int
	Due        time.Time
	LastReview *time.Time // nil = never reviewed
	Reps       int
	Lapses     int
}

// CardWithWord is the joined result returned by GetNextDueCard.
type CardWithWord struct {
	Card
	French        string
	German        string
	AssimilNumber int
	Category      string
	IsRegular     *bool
}

type UserSettings struct {
	UserID          uuid.UUID `db:"user_id"`
	NumberDigitSize int       `db:"number_digit_size"`
	UpdatedAt       time.Time `db:"updated_at"`
}

type DigitStat struct {
	UserID    uuid.UUID `db:"user_id"`
	Digit     int       `db:"digit"`
	Successes int       `db:"successes"`
}

// SessionUser is stored in the SCS session. No tokens — only identity claims,
// plus the raw id_token needed for Keycloak end-session (front-channel logout).
type SessionUser struct {
	ID          uuid.UUID `json:"id"`
	Subject     string    `json:"sub"`
	Email       string    `json:"email"`
	DisplayName string    `json:"display_name"`
	IDToken     string    `json:"id_token,omitempty"`
}
