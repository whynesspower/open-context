package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/opencontext/backend/internal/config"
	"github.com/opencontext/backend/internal/graphiti"
	"github.com/opencontext/backend/internal/store"
)

type API struct {
	Cfg  config.Config
	DB   *store.DB
	G    *graphiti.Client
	Now  func() time.Time
}

func (a *API) json(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func (a *API) err(w http.ResponseWriter, status int, msg string) {
	a.json(w, status, map[string]any{"message": msg})
}

func (a *API) readJSON(r *http.Request, v any) error {
	dec := json.NewDecoder(r.Body)
	return dec.Decode(v)
}

func ts(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339Nano)
}

