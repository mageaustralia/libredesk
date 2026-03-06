// Package ws handles WebSocket connections and broadcasting messages to clients.
package ws

import (
	"encoding/json"
	"sync"

	"github.com/abhinavxd/libredesk/internal/ws/models"
	"github.com/fasthttp/websocket"
)

// PresenceInfo holds information about a user viewing a conversation.
type PresenceInfo struct {
	UserID    int    `json:"user_id"`
	FirstName string `json:"first_name"`
	AvatarURL string `json:"avatar_url"`
}

// Hub maintains the set of registered websockets clients.
type Hub struct {
	// Client ID to WS Client map, user can connect from multiple devices and each device will have a separate client.
	clients      map[int][]*Client
	clientsMutex sync.Mutex

	// Presence tracking: convUUID -> userID -> PresenceInfo
	presence map[string]map[int]*PresenceInfo
	// Reverse lookup: client -> convUUID they are viewing
	clientConv map[*Client]string

	userStore userStore
}

type userStore interface {
	UpdateLastActive(userID int) error
}

// NewHub creates a new websocket hub.
func NewHub(userStore userStore) *Hub {
	return &Hub{
		clients:      make(map[int][]*Client, 10000),
		clientsMutex: sync.Mutex{},
		presence:     make(map[string]map[int]*PresenceInfo),
		clientConv:   make(map[*Client]string),
		userStore:    userStore,
	}
}

// AddClient adds a new client to the hub.
func (h *Hub) AddClient(client *Client) {
	h.clientsMutex.Lock()
	defer h.clientsMutex.Unlock()
	h.clients[client.ID] = append(h.clients[client.ID], client)
}

// RemoveClient removes a client from the hub.
func (h *Hub) RemoveClient(client *Client) {
	h.clientsMutex.Lock()
	defer h.clientsMutex.Unlock()

	// Clear presence for this client.
	h.clearViewingLocked(client)

	if clients, ok := h.clients[client.ID]; ok {
		for i, c := range clients {
			if c == client {
				h.clients[client.ID] = append(clients[:i], clients[i+1:]...)
				break
			}
		}
	}
}

// SetViewing marks a client as viewing a conversation.
func (h *Hub) SetViewing(client *Client, convUUID string, info *PresenceInfo) {
	h.clientsMutex.Lock()
	defer h.clientsMutex.Unlock()

	// Clear previous viewing state for this client.
	prevUUID := h.clearViewingLocked(client)

	if convUUID == "" {
		// Client is no longer viewing any conversation — broadcast update for prev.
		if prevUUID != "" {
			h.broadcastPresenceLocked(prevUUID)
		}
		return
	}

	// Set new viewing state.
	if h.presence[convUUID] == nil {
		h.presence[convUUID] = make(map[int]*PresenceInfo)
	}
	h.presence[convUUID][client.ID] = info
	h.clientConv[client] = convUUID

	// Broadcast presence updates.
	if prevUUID != "" && prevUUID != convUUID {
		h.broadcastPresenceLocked(prevUUID)
	}
	h.broadcastPresenceLocked(convUUID)
}

// clearViewingLocked clears the viewing state for a client (must be called with lock held).
// Returns the previous convUUID.
func (h *Hub) clearViewingLocked(client *Client) string {
	prevUUID, ok := h.clientConv[client]
	if !ok {
		return ""
	}

	delete(h.clientConv, client)
	if viewers, ok := h.presence[prevUUID]; ok {
		delete(viewers, client.ID)
		if len(viewers) == 0 {
			delete(h.presence, prevUUID)
		}
	}
	return prevUUID
}

// broadcastPresenceLocked broadcasts the current viewers for a conversation (must be called with lock held).
func (h *Hub) broadcastPresenceLocked(convUUID string) {
	viewers := make([]PresenceInfo, 0)
	if m, ok := h.presence[convUUID]; ok {
		for _, info := range m {
			viewers = append(viewers, *info)
		}
	}

	msg := models.Message{
		Type: models.MessageTypePresenceUpdate,
		Data: map[string]interface{}{
			"conversation_uuid": convUUID,
			"viewers":           viewers,
		},
	}

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return
	}

	// Broadcast to all connected clients.
	for _, clients := range h.clients {
		for _, c := range clients {
			c.SendMessage(msgBytes, websocket.TextMessage)
		}
	}
}

// GetViewers returns the current viewers for a conversation.
func (h *Hub) GetViewers(convUUID string) []PresenceInfo {
	h.clientsMutex.Lock()
	defer h.clientsMutex.Unlock()
	viewers := make([]PresenceInfo, 0)
	if m, ok := h.presence[convUUID]; ok {
		for _, info := range m {
			viewers = append(viewers, *info)
		}
	}
	return viewers
}

// BroadcastMessage broadcasts a message to the specified users.
// If no users are specified, the message is broadcast to all users.
func (h *Hub) BroadcastMessage(msg models.BroadcastMessage) {
	h.clientsMutex.Lock()
	defer h.clientsMutex.Unlock()

	// Broadcast to all users if no users are specified.
	if len(msg.Users) == 0 {
		for _, clients := range h.clients {
			for _, client := range clients {
				client.SendMessage(msg.Data, websocket.TextMessage)
			}
		}
		return
	}

	// Broadcast to specified users.
	for _, userID := range msg.Users {
		for _, client := range h.clients[userID] {
			client.SendMessage(msg.Data, websocket.TextMessage)
		}
	}
}
