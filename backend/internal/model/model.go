package model

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID          uuid.UUID `db:"id"`
	Subject     string    `db:"subject"`
	Email       string    `db:"email"`
	DisplayName string    `db:"display_name"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
}

type LoginRecord struct {
	ID         uuid.UUID `db:"id"`
	UserID     uuid.UUID `db:"user_id"`
	LoggedInAt time.Time `db:"logged_in_at"`
	IPAddress  string    `db:"ip_address"`
	UserAgent  string    `db:"user_agent"`
	SessionID  string    `db:"session_id"`
}

// SessionUser is stored in the SCS session. No tokens — only identity claims.
type SessionUser struct {
	ID          uuid.UUID `json:"id"`
	Subject     string    `json:"sub"`
	Email       string    `json:"email"`
	DisplayName string    `json:"display_name"`
}
