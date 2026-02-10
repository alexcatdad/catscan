// Package sse provides Server-Sent Events (SSE) for real-time updates.
//
// The SSE hub manages connected clients and broadcasts events
// to all connected clients for real-time updates.
package sse

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

// Event represents a server-sent event.
type Event struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

// Client represents a connected SSE client.
type Client struct {
	ID     string
	Chan   chan Event
	Ctx    context.Context
	Cancel context.CancelFunc
}

// Hub manages connected SSE clients and broadcasts events.
type Hub struct {
	clients    map[string]*Client
	mu         sync.RWMutex
	register   chan *Client
	unregister chan string
	broadcast  chan Event
}

// NewHub creates a new SSE hub.
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[string]*Client),
		register:   make(chan *Client),
		unregister: make(chan string),
		broadcast:  make(chan Event, 100), // Buffered to prevent blocking
	}
}

// Run starts the SSE hub's event loop.
// It should be run in a separate goroutine.
func (h *Hub) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			// Shutdown: close all client channels
			h.mu.Lock()
			for _, client := range h.clients {
				close(client.Chan)
			}
			h.clients = make(map[string]*Client)
			h.mu.Unlock()
			return

		case client := <-h.register:
			h.mu.Lock()
			h.clients[client.ID] = client
			h.mu.Unlock()

		case id := <-h.unregister:
			h.mu.Lock()
			if client, ok := h.clients[id]; ok {
				delete(h.clients, id)
				close(client.Chan)
			}
			h.mu.Unlock()

		case event := <-h.broadcast:
			h.broadcastEvent(event)
		}
	}
}

// Register registers a new SSE client.
func (h *Hub) Register(client *Client) {
	h.register <- client
}

// Unregister unregisters an SSE client by ID.
func (h *Hub) Unregister(id string) {
	h.unregister <- id
}

// Broadcast broadcasts an event to all connected clients.
func (h *Hub) Broadcast(eventType string, data interface{}) {
	h.broadcast <- Event{
		Type: eventType,
		Data: data,
	}
}

// broadcastEvent sends an event to all connected clients.
// It does not block if a client's channel is full.
func (h *Hub) broadcastEvent(event Event) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for id, client := range h.clients {
		select {
		case client.Chan <- event:
			// Event sent successfully
		default:
			// Client channel is full, likely slow or disconnected
			// Unregister this client to prevent blocking
			go h.Unregister(id)
		}
	}
}

// ClientCount returns the number of connected clients.
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// SendToClient sends an event to a specific client.
// Returns false if the client is not found or the channel is full.
func (h *Hub) SendToClient(id string, event Event) bool {
	h.mu.RLock()
	client, ok := h.clients[id]
	h.mu.RUnlock()

	if !ok {
		return false
	}

	select {
	case client.Chan <- event:
		return true
	default:
		return false
	}
}

// formatEvent formats an SSE event for HTTP response.
func formatEvent(event Event) string {
	data, err := json.Marshal(event.Data)
	if err != nil {
		data = []byte(`{"error":"failed to marshal data"}`)
	}

	return fmt.Sprintf("event: %s\ndata: %s\n\n", event.Type, string(data))
}

// Handler wraps an SSE client to provide an http.Handler.
// It handles the SSE connection lifecycle.
type Handler struct {
	hub    *Hub
	client *Client
}

// NewHandler creates a new SSE handler for the given hub.
func NewHandler(hub *Hub, clientID string) *Handler {
	ctx, cancel := context.WithCancel(context.Background())

	return &Handler{
		hub: hub,
		client: &Client{
			ID:     clientID,
			Chan:   make(chan Event, 10), // Buffered for client
			Ctx:    ctx,
			Cancel: cancel,
		},
	}
}

// ServeHTTP implements http.Handler for SSE connections.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Flush headers to ensure connection is established
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		return
	}
	flusher.Flush()

	// Register client with hub
	h.hub.Register(h.client)
	defer h.hub.Unregister(h.client.ID)

	// Send initial connection message
	h.sendEvent(w, Event{
		Type: "connected",
		Data: map[string]string{"clientId": h.client.ID},
	}, flusher)

	// Listen for client disconnect
	go func() {
		<-r.Context().Done()
		<-h.client.Ctx.Done()
	}()

	// Listen for events from hub and send to client
	for {
		select {
		case <-h.client.Ctx.Done():
			return
		case <-r.Context().Done():
			return
		case event := <-h.client.Chan:
			if !h.sendEvent(w, event, flusher) {
				return
			}
		}
	}
}

// sendEvent sends an SSE event to the response writer.
// Returns false if the client disconnected.
func (h *Handler) sendEvent(w http.ResponseWriter, event Event, flusher http.Flusher) bool {
	// Check if client is still connected
	select {
	case <-h.client.Ctx.Done():
		return false
	default:
	}

	fmt.Fprint(w, formatEvent(event))
	flusher.Flush()
	return true
}

// GetClient returns the SSE client for this handler.
func (h *Handler) GetClient() *Client {
	return h.client
}
