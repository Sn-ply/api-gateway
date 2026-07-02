package handler

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/snaply/api-gateway/internal/ws"
	"go.uber.org/zap"
)

// InternalHandler serves api-gateway's service-to-service endpoints — reachable only
// from other containers on snaply-net, gated by a shared secret instead of a JWT
// (there's no end-user token to validate for a backend-to-backend call).
type InternalHandler struct {
	hub    *ws.Hub
	secret string
	log    *zap.Logger
}

func NewInternalHandler(hub *ws.Hub, secret string, log *zap.Logger) *InternalHandler {
	return &InternalHandler{hub: hub, secret: secret, log: log}
}

// SendWS delivers a payload to a user's WebSocket connection, if any. Called by
// notification-service and message-service to push real-time events. Returns 404
// (not an error) when the user isn't connected — callers treat that as normal.
func (h *InternalHandler) SendWS(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("X-Internal-Secret") != h.secret {
		respondError(w, http.StatusUnauthorized, "invalid internal secret")
		return
	}

	var req struct {
		UserID  string `json:"user_id"`
		Payload any    `json:"payload"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid user_id")
		return
	}

	delivered, err := h.hub.SendToUser(userID, req.Payload)
	if err != nil {
		h.log.Error("internal ws send error", zap.Error(err))
		respondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if !delivered {
		respondError(w, http.StatusNotFound, "user not connected")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func respondError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
