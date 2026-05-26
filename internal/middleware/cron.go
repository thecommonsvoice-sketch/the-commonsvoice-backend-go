package middleware

import (
	"net/http"

	"github.com/thecommonsvoice-sketch/the-commonsvoice-backend-go/internal/response"
)

func CronAuth(next http.Handler, secret string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Header.Get("x-cron-key")
		if key == "" || key != secret {
			response.Error(w, http.StatusUnauthorized, "Unauthorized")
			return
		}
		next.ServeHTTP(w, r)
	})
}
