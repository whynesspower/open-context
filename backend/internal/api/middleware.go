package api

import (
	"net/http"
	"strings"
)

func (a *API) Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		const prefix = "Api-Key "
		if !strings.HasPrefix(auth, prefix) || strings.TrimPrefix(auth, prefix) != a.Cfg.APIKey {
			a.err(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		next.ServeHTTP(w, r)
	})
}
