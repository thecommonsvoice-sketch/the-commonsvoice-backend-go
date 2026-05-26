package middleware

import "net/http"


func Authorize(roles ...string) func(
	http.Handler,
) http.Handler {
	return func(next http.Handler) http.Handler {

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request){

			user := GetUser(r)

			if user == nil {
				http.Error(w, "Unauthorized",
					http.StatusUnauthorized)
					return
			}

			allowed := false
			for _, role := range roles {
				if user.Role == role {
					allowed = true
					break
				}
			}

			if !allowed {
				http.Error(w, `{"message":"Access denied: insufficient permissions"}`, http.StatusForbidden)
				return
			}

			next.ServeHTTP(w,r)

		})
	}
}