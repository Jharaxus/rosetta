package auth

import (
	"encoding/json"
	"net/http"

	"github.com/alexedwards/scs/v2"
	"github.com/gin-gonic/gin"

	"github.com/jharaxus/rosetta/internal/model"
)

// RequireAuth is a Gin middleware that validates the SCS session and sets the
// authenticated user in the Gin context under contextKeyUser.
func RequireAuth(sessions *scs.SessionManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := sessions.GetString(c.Request.Context(), sessionKeyUser)
		if raw == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
			return
		}
		var u model.SessionUser
		if err := json.Unmarshal([]byte(raw), &u); err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
			return
		}
		c.Set(contextKeyUser, &u)
		c.Next()
	}
}

// GetSessionUser retrieves the authenticated user placed in the Gin context by RequireAuth.
func GetSessionUser(c *gin.Context) (*model.SessionUser, bool) {
	v, ok := c.Get(contextKeyUser)
	if !ok {
		return nil, false
	}
	u, ok := v.(*model.SessionUser)
	return u, ok
}
