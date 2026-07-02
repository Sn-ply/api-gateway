package ws

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Hub struct {
	mu          sync.RWMutex
	connections map[uuid.UUID]*websocket.Conn
	log         *zap.Logger
}

func NewHub(log *zap.Logger) *Hub {
	return &Hub{
		connections: make(map[uuid.UUID]*websocket.Conn),
		log:         log,
	}
}

func (h *Hub) Register(userID uuid.UUID, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if old, ok := h.connections[userID]; ok {
		old.Close()
	}
	h.connections[userID] = conn
	h.log.Info("ws client registered", zap.String("user_id", userID.String()))
}

func (h *Hub) Unregister(userID uuid.UUID) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if conn, ok := h.connections[userID]; ok {
		conn.Close()
		delete(h.connections, userID)
	}
	h.log.Info("ws client unregistered", zap.String("user_id", userID.String()))
}

// SendToUser writes message to userID's connection, if any. The returned bool reports
// whether the user was connected — callers (the internal delivery endpoint, in
// particular) use this to distinguish "delivered" from "no such connection".
func (h *Hub) SendToUser(userID uuid.UUID, message any) (bool, error) {
	h.mu.RLock()
	conn, ok := h.connections[userID]
	h.mu.RUnlock()
	if !ok {
		return false, nil
	}
	data, err := json.Marshal(message)
	if err != nil {
		return false, err
	}
	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		return false, err
	}
	return true, nil
}

// ServeWS upgrades the connection and blocks reading until the client disconnects.
// userID must already be resolved (from JWT validated upstream).
func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request, userID uuid.UUID) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.log.Error("ws upgrade error", zap.Error(err))
		return
	}

	h.Register(userID, conn)
	defer h.Unregister(userID)

	// Read loop — keeps the connection alive, handles client-initiated close, and
	// forwards typing indicators directly to the target user (no DB write, no Kafka —
	// this fires on every keystroke and has to be as close to instant as possible).
	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			break
		}
		h.handleClientMessage(userID, data)
	}
}

type clientMessage struct {
	Type           string `json:"type"`
	ConversationID string `json:"conversation_id"`
	RecipientID    string `json:"recipient_id"`
}

func (h *Hub) handleClientMessage(senderID uuid.UUID, data []byte) {
	var msg clientMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return
	}

	var typing bool
	switch msg.Type {
	case "typing.start":
		typing = true
	case "typing.stop":
		typing = false
	default:
		return
	}

	recipientID, err := uuid.Parse(msg.RecipientID)
	if err != nil {
		return
	}

	if _, err := h.SendToUser(recipientID, map[string]any{
		"type": "typing.indicator",
		"data": map[string]any{
			"conversation_id": msg.ConversationID,
			"user_id":         senderID.String(),
			"typing":          typing,
		},
	}); err != nil {
		h.log.Warn("failed to forward typing indicator", zap.Error(err))
	}
}
