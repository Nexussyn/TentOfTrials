# Fix for Issue #1: [$45 BOUNTY] [Go] Add WebSocket hub lifecycle tests

// market/ws/hub.go
package ws

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512

	// Size of the client send buffer
	sendBufferSize = 256
)

// Client represents a connected WebSocket client
type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan []byte
	id   string
}

// Hub maintains the set of active clients and broadcasts messages to them
type Hub struct {
	// Registered clients
	clients map[*Client]bool

	// Inbound messages from clients to broadcast
	broadcast chan []byte

	// Register requests from clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Mutex for clients map
	mu sync.RWMutex

	// Done channel to signal shutdown
	done chan struct{}

	// Track slow clients to remove (collected during broadcast, removed after)
	slowClients chan *Client
}

// NewHub creates a new Hub instance
func NewHub() *Hub {
	return &Hub{
		broadcast:   make(chan []byte, 256),
		register:    make(chan *Client),
		unregister:  make(chan *Client),
		clients:     make(map[*Client]bool),
		done:        make(chan struct{}),
		slowClients: make(chan *Client, 100),
	}
}

// Run starts the hub's main event loop
func (h *Hub) Run() {
	// Start slow client cleanup goroutine
	go h.cleanupSlowClients()

	for {
		select {
		case <-h.done:
			h.mu.Lock()
			for client := range h.clients {
				close(client.send)
				delete(h.clients, client)
			}
			h.mu.Unlock()
			return

		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			h.broadcastMessage(message)
		}
	}
}

// broadcastMessage sends a message to all connected clients
// Slow clients are queued for removal rather than removed under read lock
func (h *Hub) broadcastMessage(message []byte) {
	h.mu.RLock()
	for client := range h.clients {
		select {
		case client.send <- message:
			// Message sent successfully
		default:
			// Client send buffer is full - queue for removal
			// Don't modify h.clients while holding RLock
			select {
			case h.slowClients <- client:
			default:
				// Slow client channel full, will be cleaned up next iteration
			}
		}
	}
	h.mu.RUnlock()
}

// cleanupSlowClients removes slow clients in a separate goroutine
// This avoids modifying h.clients while holding a read lock
func (h *Hub) cleanupSlowClients() {
	for {
		select {
		case <-h.done:
			return
		case client := <-h.slowClients:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()
		}
	}
}

// Stop gracefully shuts down the hub
func (h *Hub) Stop() {
	close(h.done)
}

// Broadcast sends a message to all connected clients
func (h *Hub) Broadcast(message []byte) {
	select {
	case h.broadcast <- message:
	case <-h.done:
	}
}

// Register adds a client to the hub
func (h *Hub) Register(client *Client) {
	select {
	case h.register <- client:
	case <-h.done:
	}
}

// Unregister removes a client from the hub
func (h *Hub) Unregister(client *Client) {
	select {
	case h.unregister <- client:
	case <-h.done:
	}
}

// ClientCount returns the number of connected clients
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// NewClient creates a new client for the hub
func NewClient(hub *Hub, conn *websocket.Conn, id string) *Client {
	return &Client{
		hub:  hub,
		conn: conn,
		send: make(chan []byte, sendBufferSize),
		id:   id,
	}
}

// NewTestClient creates a client for testing without a real connection
func NewTestClient(hub *Hub, id string) *Client {
	return &Client{
		hub:  hub,
		conn: nil,
		send: make(chan []byte, sendBufferSize),
		id:   id,
	}
}

// NewTestClientWithBufferSize creates a client with custom buffer size for testing
func NewTestClientWithBufferSize(hub *Hub, id string, bufferSize int) *Client {
	return &Client{
		hub:  hub,
		conn: nil,
		send: make(chan []byte, bufferSize),
		id:   id,
	}
}

// ID returns the client's identifier
func (c *Client) ID() string {
	return c.id
}

// Send returns the client's send channel (for testing)
func (c *Client) Send() chan []byte {
	return c.send
}