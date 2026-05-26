package response

import (
	"net/http"
	"os"
	"time"
)

const (
	AccessTokenMaxAge  = 24 * time.Hour
	RefreshTokenMaxAge = 7 * 24 * time.Hour
)

// isProd returns true when NODE_ENV is "production".
func isProd() bool {
	return os.Getenv("NODE_ENV") == "production"
}

func SetAccessToken(w http.ResponseWriter, token string) {
	sameSite := http.SameSiteLaxMode
	secure := false
	if isProd() {
		sameSite = http.SameSiteNoneMode
		secure = true
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		MaxAge:   int(AccessTokenMaxAge.Seconds()),
		SameSite: sameSite,
		Secure:   secure,
	})
}

func SetRefreshToken(w http.ResponseWriter, token string) {
	sameSite := http.SameSiteLaxMode
	secure := false
	if isProd() {
		sameSite = http.SameSiteNoneMode
		secure = true
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		MaxAge:   int(RefreshTokenMaxAge.Seconds()),
		SameSite: sameSite,
		Secure:   secure,
	})
}

func ClearAuthCookies(w http.ResponseWriter) {
	sameSite := http.SameSiteLaxMode
	secure := false
	if isProd() {
		sameSite = http.SameSiteNoneMode
		secure = true
	}

	http.SetCookie(w, &http.Cookie{
		Name: "access_token", Value: "", Path: "/", MaxAge: -1,
		SameSite: sameSite, Secure: secure,
	})
	http.SetCookie(w, &http.Cookie{
		Name: "refresh_token", Value: "", Path: "/", MaxAge: -1,
		SameSite: sameSite, Secure: secure,
	})
}