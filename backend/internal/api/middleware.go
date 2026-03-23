package api

import (
	"net/http"
	"strings"
)

func (a *API) Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		var token string
		switch {
		case strings.HasPrefix(auth, "Bearer "):
			token = strings.TrimPrefix(auth, "Bearer ")
		case strings.HasPrefix(auth, "Api-Key "):
			token = strings.TrimPrefix(auth, "Api-Key ")
		}
		if token == "" || token != a.Cfg.APIKey {
			a.err(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		next.ServeHTTP(w, r)
	})
}
