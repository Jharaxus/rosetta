package auth

import (
	"net/http"
	"time"

	"github.com/gorilla/securecookie"

	"github.com/jharaxus/rosetta/internal/config"
)

const (
	cookiePreAuth = "rosetta_preauth"
	cookieIDHint  = "rosetta_id_hint"
)

// preAuthData is signed+encrypted in the pre-auth cookie.
type preAuthData struct {
	State    string `json:"state"`
	Verifier string `json:"verifier"`
}

var sc *securecookie.SecureCookie

func InitCookies(cfg *config.Config) {
	sc = securecookie.New(cfg.CookieHashKey, cfg.CookieEncryptKey)
	sc.SetSerializer(securecookie.JSONEncoder{})
	sc.MaxAge(300) // reject decode of cookies older than 5 min
}

func setPreAuthCookie(w http.ResponseWriter, r *http.Request, state, verifier string, secure bool) error {
	encoded, err := sc.Encode(cookiePreAuth, preAuthData{State: state, Verifier: verifier})
	if err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     cookiePreAuth,
		Value:    encoded,
		Path:     "/api/auth/callback",
		MaxAge:   300,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
	return nil
}

func readPreAuthCookie(r *http.Request) (*preAuthData, error) {
	cookie, err := r.Cookie(cookiePreAuth)
	if err != nil {
		return nil, err
	}
	var data preAuthData
	if err := sc.Decode(cookiePreAuth, cookie.Value, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

func deletePreAuthCookie(w http.ResponseWriter, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookiePreAuth,
		Value:    "",
		Path:     "/api/auth/callback",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

// SetIDHintCookie stores the raw id_token in a short-lived httpOnly cookie
// scoped exclusively to the logout endpoint.
func setIDHintCookie(w http.ResponseWriter, rawIDToken string, expiry time.Time, secure bool) {
	maxAge := int(time.Until(expiry).Seconds())
	if maxAge <= 0 {
		maxAge = 1
	}
	http.SetCookie(w, &http.Cookie{
		Name:     cookieIDHint,
		Value:    rawIDToken,
		Path:     "/api/auth/logout",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func readIDHintCookie(r *http.Request) string {
	cookie, err := r.Cookie(cookieIDHint)
	if err != nil {
		return ""
	}
	return cookie.Value
}

func deleteIDHintCookie(w http.ResponseWriter, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieIDHint,
		Value:    "",
		Path:     "/api/auth/logout",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}
