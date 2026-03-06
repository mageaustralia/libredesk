package ws

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/abhinavxd/libredesk/internal/ws/models"
	"github.com/fasthttp/websocket"
)

// SafeBool is a thread-safe boolean.
type SafeBool struct {
	flag bool
	mu   sync.Mutex
}

// Set sets the value of the SafeBool.
func (b *SafeBool) Set(value bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.flag = value
}

// Get returns the value of the SafeBool.
func (b *SafeBool) Get() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.flag
}

// Client is a single connected WS user.
type Client struct {
	// Client ID (user ID).
	ID int

	// User display info for presence.
	FirstName string
	AvatarURL string

	// Hub.
	Hub *Hub

	// WebSocket connection.
	Conn *websocket.Conn

	// To prevent pushes to the channel.
	Closed SafeBool

	// Buffered channel of outbound ws messages.
	Send chan models.WSMessage
}

// Serve handles heartbeats and sending messages to the client.
func (c *Client) Serve() {
	var heartBeatTicker = time.NewTicker(2 * time.Second)
	defer heartBeatTicker.Stop()

Loop:
	for {
		select {
		case <-heartBeatTicker.C:
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				fmt.Println("error writing message", err)
				return
			}
		case msg, ok := <-c.Send:
			if !ok {
				break Loop
			}
			c.Conn.WriteMessage(msg.MessageType, msg.Data)
		}
	}
	c.Conn.Close()
}

// Listen is a block method that listens for incoming messages from the client.
func (c *Client) Listen() {
	for {
		msgType, msg, err := c.Conn.ReadMessage()
		if err != nil {
			break
		}

		if msgType == websocket.TextMessage {
			c.processIncomingMessage(msg)
		} else {
			c.Hub.RemoveClient(c)
			c.close()
			return
		}
	}
	c.Hub.RemoveClient(c)
	c.close()
}

// incomingMessage represents a JSON message from the client.
type incomingMessage struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

// viewConversationData represents the data for a view_conversation message.
type viewConversationData struct {
	ConversationUUID string `json:"conversation_uuid"`
}

// processIncomingMessage processes incoming messages from the client.
func (c *Client) processIncomingMessage(data []byte) {
	// Handle ping messages, and update last active time for user.
	if string(data) == "ping" {
		c.Hub.userStore.UpdateLastActive(c.ID)
		c.SendMessage([]byte("pong"), websocket.TextMessage)
		return
	}

	// Try to parse as JSON message.
	var msg incomingMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		c.SendError("invalid message format")
		return
	}

	switch msg.Type {
	case models.MessageTypeViewConversation:
		var viewData viewConversationData
		if err := json.Unmarshal(msg.Data, &viewData); err != nil {
			c.SendError("invalid view_conversation data")
			return
		}
		c.Hub.SetViewing(c, viewData.ConversationUUID, &PresenceInfo{
			UserID:    c.ID,
			FirstName: c.FirstName,
			AvatarURL: c.AvatarURL,
		})
	default:
		c.SendError("unknown incoming message type")
	}
}

// close closes the client connection.
func (c *Client) close() {
	c.Closed.Set(true)
	close(c.Send)
}

// SendError sends an error message to client.
func (c *Client) SendError(msg string) {
	out := models.Message{
		Type: models.MessageTypeError,
		Data: msg,
	}
	b, _ := json.Marshal(out)

	select {
	case c.Send <- models.WSMessage{Data: b, MessageType: websocket.TextMessage}:
	default:
		log.Println("Client send channel is full. Could not send error message.")
		c.Hub.RemoveClient(c)
		c.close()
	}
}

// SendMessage sends a message to client.
func (c *Client) SendMessage(b []byte, typ byte) {
	if c.Closed.Get() {
		log.Println("Attempted to send message to closed client")
		return
	}
	select {
	case c.Send <- models.WSMessage{Data: b, MessageType: websocket.TextMessage}:
	default:
	}
}
