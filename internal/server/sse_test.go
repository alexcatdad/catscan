package server_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/alexcatdad/catscan/internal/server"
)

// TestSSEHubRegisterClient tests client registration.
func TestSSEHubRegisterClient(t *testing.T) {
	hub := server.NewSSEHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go hub.Run(ctx)

	client := &server.SSEClient{
		ID:     "test-client",
		Chan:   make(chan server.SSEEvent, 10),
		Ctx:    ctx,
		Cancel: cancel,
	}

	hub.Register(client)

	// Give the hub time to register
	time.Sleep(10 * time.Millisecond)

	if hub.ClientCount() != 1 {
		t.Errorf("ClientCount = %d, want 1", hub.ClientCount())
	}
}

// TestSSEHubUnregisterClient tests client unregistration.
func TestSSEHubUnregisterClient(t *testing.T) {
	hub := server.NewSSEHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go hub.Run(ctx)

	client := &server.SSEClient{
		ID:     "test-client",
		Chan:   make(chan server.SSEEvent, 10),
		Ctx:    ctx,
		Cancel: cancel,
	}

	hub.Register(client)
	time.Sleep(10 * time.Millisecond)

	hub.Unregister("test-client")
	time.Sleep(10 * time.Millisecond)

	if hub.ClientCount() != 0 {
		t.Errorf("ClientCount = %d, want 0", hub.ClientCount())
	}
}

// TestSSEHubBroadcastReachesAllClients tests that broadcast reaches all clients.
func TestSSEHubBroadcastReachesAllClients(t *testing.T) {
	hub := server.NewSSEHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go hub.Run(ctx)

	// Create multiple clients
	clients := []*server.SSEClient{
		{
			ID:     "client-1",
			Chan:   make(chan server.SSEEvent, 10),
			Ctx:    ctx,
			Cancel: cancel,
		},
		{
			ID:     "client-2",
			Chan:   make(chan server.SSEEvent, 10),
			Ctx:    ctx,
			Cancel: cancel,
		},
		{
			ID:     "client-3",
			Chan:   make(chan server.SSEEvent, 10),
			Ctx:    ctx,
			Cancel: cancel,
		},
	}

	for _, client := range clients {
		hub.Register(client)
	}
	time.Sleep(10 * time.Millisecond)

	// Broadcast an event
	testData := map[string]string{"message": "test"}
	hub.Broadcast("test_event", testData)

	// Wait for event to propagate
	time.Sleep(10 * time.Millisecond)

	// Check that all clients received the event
	for i, client := range clients {
		select {
		case event := <-client.Chan:
			if event.Type != "test_event" {
				t.Errorf("client %d: event.Type = %s, want test_event", i, event.Type)
			}
		default:
			t.Errorf("client %d: did not receive event", i)
		}
	}
}

// TestSSEHubBroadcastDoesntBlock tests that broadcast doesn't block
// when a client's channel is full.
func TestSSEHubBroadcastDoesntBlock(t *testing.T) {
	hub := server.NewSSEHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go hub.Run(ctx)

	// Create a slow client with a full channel
	slowClientCtx, slowClientCancel := context.WithCancel(context.Background())
	slowClient := &server.SSEClient{
		ID:     "slow-client",
		Chan:   make(chan server.SSEEvent, 1), // Small buffer
		Ctx:    slowClientCtx,
		Cancel: slowClientCancel,
	}

	// Fill the channel
	slowClient.Chan <- server.SSEEvent{Type: "filler"}

	// Create a normal client
	normalClientCtx, normalClientCancel := context.WithCancel(context.Background())
	normalClient := &server.SSEClient{
		ID:     "normal-client",
		Chan:   make(chan server.SSEEvent, 10),
		Ctx:    normalClientCtx,
		Cancel: normalClientCancel,
	}

	hub.Register(slowClient)
	hub.Register(normalClient)
	time.Sleep(10 * time.Millisecond)

	// Broadcast multiple events rapidly
	for i := 0; i < 5; i++ {
		hub.Broadcast("test", map[string]int{"value": i})
	}

	// Wait a bit
	time.Sleep(50 * time.Millisecond)

	// The slow client should have been unregistered (channel was full)
	// The normal client should still be registered
	count := hub.ClientCount()
	// We expect at least the normal client
	if count < 1 {
		t.Errorf("ClientCount = %d, want at least 1", count)
	}
}

// TestSSEHubSendToClient tests sending to a specific client.
func TestSSEHubSendToClient(t *testing.T) {
	hub := server.NewSSEHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go hub.Run(ctx)

	client := &server.SSEClient{
		ID:     "test-client",
		Chan:   make(chan server.SSEEvent, 10),
		Ctx:    ctx,
		Cancel: cancel,
	}

	hub.Register(client)
	time.Sleep(10 * time.Millisecond)

	event := server.SSEEvent{
		Type: "direct_message",
		Data: map[string]string{"hello": "world"},
	}

	if !hub.SendToClient("test-client", event) {
		t.Error("SendToClient returned false, want true")
	}

	// Verify client received the event
	select {
	case received := <-client.Chan:
		if received.Type != "direct_message" {
			t.Errorf("event.Type = %s, want direct_message", received.Type)
		}
	default:
		t.Error("client did not receive direct message")
	}
}

// TestSSEHubSendToNonExistentClient tests sending to a non-existent client.
func TestSSEHubSendToNonExistentClient(t *testing.T) {
	hub := server.NewSSEHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go hub.Run(ctx)

	event := server.SSEEvent{
		Type: "test",
		Data: nil,
	}

	if hub.SendToClient("non-existent", event) {
		t.Error("SendToClient returned true for non-existent client, want false")
	}
}

// TestSSEHubConcurrentAccess tests that the hub handles concurrent access safely.
func TestSSEHubConcurrentAccess(t *testing.T) {
	hub := server.NewSSEHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go hub.Run(ctx)

	// Create multiple clients with separate contexts
	var clients []*server.SSEClient
	for i := 0; i < 5; i++ {
		clientCtx, clientCancel := context.WithCancel(context.Background())
		client := &server.SSEClient{
			ID:     fmt.Sprintf("client-%d", i),
			Chan:   make(chan server.SSEEvent, 100), // Larger buffer
			Ctx:    clientCtx,
			Cancel: clientCancel,
		}
		clients = append(clients, client)
		hub.Register(client)
	}
	time.Sleep(10 * time.Millisecond)

	// Perform concurrent operations
	var wg sync.WaitGroup

	// Broadcast from multiple goroutines
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			for j := 0; j < 5; j++ {
				hub.Broadcast(fmt.Sprintf("goroutine-%d", idx), map[string]int{"value": j})
			}
		}(i)
	}

	wg.Wait()

	// Verify hub is still functional
	time.Sleep(10 * time.Millisecond)
	count := hub.ClientCount()
	if count != 5 {
		t.Errorf("ClientCount = %d, want 5", count)
	}
}
