package middlewares

import "net/http"

// TLS отклоняет HTTP-запросы без TLS.
func TLS() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.TLS == nil {
				http.Error(w, "TLS required", http.StatusBadRequest)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
