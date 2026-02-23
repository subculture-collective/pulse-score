package handler

import (
	"encoding/json"
	"net/http"

	"github.com/onnwee/pulse-score/internal/database"
)

// HealthHandler provides health check endpoints.
type HealthHandler struct {
	pool *database.Pool
}

// NewHealthHandler creates a new HealthHandler.
func NewHealthHandler(pool *database.Pool) *HealthHandler {
	return &HealthHandler{pool: pool}
}

// Liveness always returns 200 — the server is alive.
func (h *HealthHandler) Liveness(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// Readiness returns 200 if the database is reachable, 503 otherwise.
func (h *HealthHandler) Readiness(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if h.pool == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "degraded",
			"db":     "not configured",
		})
		return
	}

	if err := h.pool.P.Ping(r.Context()); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "degraded",
			"db":     "unreachable",
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
		"db":     "connected",
	})
}
