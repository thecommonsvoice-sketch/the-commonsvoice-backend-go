package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/thecommonsvoice-sketch/the-commonsvoice-backend-go/internal/services"
)

type contextKey string

const UserContextKey contextKey = "user"

func Authenticate(next http.Handler, jwtSecret string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		var tokenStr string

		// get token from cookies
		cookie, err := r.Cookie("access_token")

		if err == nil && cookie.Value != "" {
			tokenStr = cookie.Value
		}

		if tokenStr == "" {
			authHeader := r.Header.Get("Authorization")

			if strings.HasPrefix(authHeader, "Bearer ") {
				tokenStr = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}

		if tokenStr == "" {
			http.Error(w, `{"message":"Authorization token is missing"}`, http.StatusUnauthorized)
			return
		}

		claims, err := services.ValidateToken(tokenStr, jwtSecret)

		if err != nil {
			http.Error(w, `{"message":"Invalid or expired token"}`, http.StatusUnauthorized)

			return
		}

		ctx := context.WithValue(r.Context(), UserContextKey, claims)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func GetUser(r *http.Request) *services.AuthClaims {
	claims, _ := r.Context().Value(UserContextKey).(*services.AuthClaims)
	return claims
}
