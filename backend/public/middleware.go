package public

import (
	"fmt"
	"net/http"
)

// AllowGETOnly is middleware that enforces GET-only on public routes.
func AllowGETOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", "GET, OPTIONS")
			w.WriteHeader(http.StatusMethodNotAllowed)
			_, _ = fmt.Fprintln(w, `{"error":"method not allowed"}`)
			return
		}
		next.ServeHTTP(w, r)
	})
}
