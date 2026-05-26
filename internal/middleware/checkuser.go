package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/thecommonsvoice-sketch/the-commonsvoice-backend-go/internal/services"
)

func CheckUser(next http.Handler, jwtSecret string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var tokenStr string

		// 1. Try cookie first
		cookie, err := r.Cookie("access_token")
		if err == nil && cookie.Value != "" {
			tokenStr = cookie.Value
		}

		// 2. Fallback to Authorization: Bearer header (Chrome may block cross-domain cookies)
		if tokenStr == "" {
			authHeader := r.Header.Get("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				tokenStr = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}

		// No token found — proceed as guest
		if tokenStr == "" {
			next.ServeHTTP(w, r)
			return
		}

		claims, err := services.ValidateToken(tokenStr, jwtSecret)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}

		ctx := context.WithValue(r.Context(), UserContextKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
