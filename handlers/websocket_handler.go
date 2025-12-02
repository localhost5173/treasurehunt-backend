package handlers

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// WebSocketMessage represents a message sent through WebSocket
type WebSocketMessage struct {
	Type      string      `json:"type"`
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
}

// Client represents a WebSocket client connection
type Client struct {
	ID       string
	UserID   primitive.ObjectID
	Conn     *websocket.Conn
	Send     chan []byte
	isClosed bool
	mu       sync.Mutex
}

// Close safely closes the client's send channel
func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.isClosed {
		c.isClosed = true
		close(c.Send)
	}
}

// Hub maintains the set of active clients and broadcasts messages to them
type Hub struct {
	// Registered clients mapped by UserID
	clients map[primitive.ObjectID]*Client

	// Inbound messages from clients
	broadcast chan []byte

	// Register requests from clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Mutex for thread-safe operations
	mu sync.RWMutex
}

// NewHub creates a new Hub instance
func NewHub() *Hub {
	return &Hub{
		broadcast:  make(chan []byte, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[primitive.ObjectID]*Client),
	}
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			// If user already has a connection, close the old one
			if existingClient, exists := h.clients[client.UserID]; exists {
				existingClient.Close()
				existingClient.Conn.Close()
			}
			h.clients[client.UserID] = client
			h.mu.Unlock()
			log.Printf("Client registered: UserID=%s, Total clients=%d", client.UserID.Hex(), len(h.clients))

		case client := <-h.unregister:
			h.mu.Lock()
			// Only unregister if this is the current client for this user
			// (prevents new connection from being removed when old one closes)
			if currentClient, ok := h.clients[client.UserID]; ok && currentClient.ID == client.ID {
				delete(h.clients, client.UserID)
				client.Close()
				log.Printf("Client unregistered: UserID=%s, Total clients=%d", client.UserID.Hex(), len(h.clients))
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			// Broadcast to all clients (not used for targeted notifications)
			h.mu.RLock()
			for _, client := range h.clients {
				select {
				case client.Send <- message:
				default:
					// Client's send channel is full, close and unregister
					client.Close()
					delete(h.clients, client.UserID)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// SendToUser sends a notification message to a specific user
func (h *Hub) SendToUser(userID primitive.ObjectID, message interface{}) error {
	return h.SendToUserWithType(userID, "notification", message)
}

// SendToUserWithType sends a message with a custom type to a specific user
func (h *Hub) SendToUserWithType(userID primitive.ObjectID, messageType string, message interface{}) error {
	h.mu.RLock()
	client, exists := h.clients[userID]
	h.mu.RUnlock()

	if !exists {
		log.Printf("User not connected: %s", userID.Hex())
		return nil // Not an error, user is just offline
	}

	wsMessage := WebSocketMessage{
		Type:      messageType,
		Data:      message,
		Timestamp: time.Now(),
	}

	messageBytes, err := json.Marshal(wsMessage)
	if err != nil {
		log.Printf("Error marshaling message: %v", err)
		return err
	}

	select {
	case client.Send <- messageBytes:
		log.Printf("Message sent to user: %s (type: %s)", userID.Hex(), messageType)
	default:
		log.Printf("Failed to send message, client buffer full: %s", userID.Hex())
	}

	return nil
}

// BroadcastToAll sends a message to all connected clients
func (h *Hub) BroadcastToAll(message interface{}) error {
	wsMessage := WebSocketMessage{
		Type:      "broadcast",
		Data:      message,
		Timestamp: time.Now(),
	}

	messageBytes, err := json.Marshal(wsMessage)
	if err != nil {
		return err
	}

	h.broadcast <- messageBytes
	return nil
}

// GetConnectedUserIDs returns a list of currently connected user IDs
func (h *Hub) GetConnectedUserIDs() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	userIDs := make([]string, 0, len(h.clients))
	for userID := range h.clients {
		userIDs = append(userIDs, userID.Hex())
	}
	return userIDs
}

// IsUserConnected checks if a user is currently connected
func (h *Hub) IsUserConnected(userID primitive.ObjectID) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	_, exists := h.clients[userID]
	return exists
}

// ReadPump reads messages from the WebSocket connection
func (c *Client) ReadPump(hub *Hub) {
	defer func() {
		hub.unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Handle incoming messages if needed
		log.Printf("Received message from user %s: %s", c.UserID.Hex(), string(message))
	}
}

// WritePump writes messages to the WebSocket connection
func (c *Client) WritePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				// Hub closed the channel
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("Error writing message: %v", err)
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// WebSocketHandler handles WebSocket upgrade and client management
type WebSocketHandler struct {
	Hub *Hub
}

// NewWebSocketHandler creates a new WebSocket handler
func NewWebSocketHandler(hub *Hub) *WebSocketHandler {
	return &WebSocketHandler{
		Hub: hub,
	}
}

// HandleWebSocket handles WebSocket connections (must be used after auth middleware)
func (h *WebSocketHandler) HandleWebSocket(c *websocket.Conn) {
	// Get userID from Fiber context (set by auth middleware)
	// Note: In Fiber v2 with WebSocket, we need to get this from the HTTP request
	userIDStr := c.Locals("userID").(string)
	userID, err := primitive.ObjectIDFromHex(userIDStr)
	if err != nil {
		log.Printf("Invalid user ID: %v", err)
		c.Close()
		return
	}

	// Generate a unique connection ID to distinguish between old and new connections
	connectionID := primitive.NewObjectID().Hex()

	client := &Client{
		ID:     connectionID,
		UserID: userID,
		Conn:   c,
		Send:   make(chan []byte, 256),
	}

	h.Hub.register <- client

	// Start goroutines for reading and writing
	go client.WritePump()
	client.ReadPump(h.Hub) // This blocks until connection is closed
}

// UpgradeConnection handles the WebSocket upgrade
func (h *WebSocketHandler) UpgradeConnection() fiber.Handler {
	return websocket.New(func(c *websocket.Conn) {
		h.HandleWebSocket(c)
	})
}
