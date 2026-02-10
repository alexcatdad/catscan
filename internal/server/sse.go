// Package server provides the HTTP server for CatScan.
//
// The sse subpackage handles Server-Sent Events (SSE) for real-time updates.
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

// SSEEvent represents a server-sent event.
type SSEEvent struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

// SSEClient represents a connected SSE client.
type SSEClient struct {
	ID     string
	Chan   chan SSEEvent
	Ctx    context.Context
	Cancel context.CancelFunc
}

// SSEHub manages connected SSE clients and broadcasts events.
type SSEHub struct {
	clients map[string]*SSEClient
	mu      sync.RWMutex
	register chan *SSEClient
	unregister chan string
	broadcast  chan SSEEvent
}

// NewSSEHub creates a new SSE hub.
func NewSSEHub() *SSEHub {
	return &SSEHub{
		clients:    make(map[string]*SSEClient),
		register:   make(chan *SSEClient),
		unregister: make(chan string),
		broadcast:  make(chan SSEEvent, 100), // Buffered to prevent blocking
	}
}

// Run starts the SSE hub's event loop.
// It should be run in a separate goroutine.
func (h *SSEHub) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			// Shutdown: close all client channels
			h.mu.Lock()
			for _, client := range h.clients {
				close(client.Chan)
			}
			h.clients = make(map[string]*SSEClient)
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
func (h *SSEHub) Register(client *SSEClient) {
	h.register <- client
}

// Unregister unregisters an SSE client by ID.
func (h *SSEHub) Unregister(id string) {
	h.unregister <- id
}

// Broadcast broadcasts an event to all connected clients.
func (h *SSEHub) Broadcast(eventType string, data interface{}) {
	h.broadcast <- SSEEvent{
		Type: eventType,
		Data: data,
	}
}

// broadcastEvent sends an event to all connected clients.
// It does not block if a client's channel is full.
func (h *SSEHub) broadcastEvent(event SSEEvent) {
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
func (h *SSEHub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// SendToClient sends an event to a specific client.
// Returns false if the client is not found or the channel is full.
func (h *SSEHub) SendToClient(id string, event SSEEvent) bool {
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

// formatSSE formats an SSE event for HTTP response.
func formatSSE(event SSEEvent) string {
	data, err := json.Marshal(event.Data)
	if err != nil {
		data = []byte(`{"error":"failed to marshal data"}`)
	}

	return fmt.Sprintf("event: %s\ndata: %s\n\n", event.Type, string(data))
}

// SSEHandler wraps an SSE client to provide an http.Handler.
// It handles the SSE connection lifecycle.
type SSEHandler struct {
	hub    *SSEHub
	client *SSEClient
}

// NewSSEHandler creates a new SSE handler for the given hub.
func NewSSEHandler(hub *SSEHub, clientID string) *SSEHandler {
	ctx, cancel := context.WithCancel(context.Background())

	return &SSEHandler{
		hub: hub,
		client: &SSEClient{
			ID:     clientID,
			Chan:   make(chan SSEEvent, 10), // Buffered for client
			Ctx:    ctx,
			Cancel: cancel,
		},
	}
}

// ServeHTTP implements http.Handler for SSE connections.
func (h *SSEHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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
	h.sendEvent(w, SSEEvent{
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
func (h *SSEHandler) sendEvent(w http.ResponseWriter, event SSEEvent, flusher http.Flusher) bool {
	// Check if client is still connected
	select {
	case <-h.client.Ctx.Done():
		return false
	default:
	}

	fmt.Fprint(w, formatSSE(event))
	flusher.Flush()
	return true
}

// GetClient returns the SSE client for this handler.
func (h *SSEHandler) GetClient() *SSEClient {
	return h.client
}
