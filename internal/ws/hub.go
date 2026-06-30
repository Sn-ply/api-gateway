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

func (h *Hub) SendToUser(userID uuid.UUID, message any) error {
	h.mu.RLock()
	conn, ok := h.connections[userID]
	h.mu.RUnlock()
	if !ok {
		return nil
	}
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}
	return conn.WriteMessage(websocket.TextMessage, data)
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

	// Read loop — keeps the connection alive and handles client-initiated close
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}
}
