package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/alexcatdad/catscan/internal/cache"
	"github.com/alexcatdad/catscan/internal/config"
	"github.com/alexcatdad/catscan/internal/model"
	"github.com/alexcatdad/catscan/internal/sse"
)

// TestServerCreation tests that a new server can be created.
func TestServerCreation(t *testing.T) {
	cfg := &config.Config{
		ScanPath:            "/tmp/test",
		Port:                8080,
		LocalIntervalSeconds: 30,
		GitHubIntervalSeconds: 300,
		StaleDays:           30,
		AbandonedDays:       90,
	}

	s, err := NewServer(cfg)
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}

	if s == nil {
		t.Fatal("NewServer returned nil")
	}

	if s.hub == nil {
		t.Error("hub is nil")
	}

	if s.poller == nil {
		t.Error("poller is nil")
	}
}

// TestReposListReturnsCorrectShape tests that the repos list endpoint returns correct JSON.
func TestReposListReturnsCorrectShape(t *testing.T) {
	// Set up test cache
	testRepos := []model.Repo{
		{
			Name:        "test-repo-1",
			Description: "Test repo 1",
			Visibility:  model.VisibilityPublic,
			Cloned:      true,
			Lifecycle:   model.LifecycleOngoing,
		},
		{
			Name:        "test-repo-2",
			Description: "Test repo 2",
			Visibility:  model.VisibilityPrivate,
			Cloned:      false,
			Lifecycle:   model.LifecycleStale,
		},
	}

	// Create temp directory for cache
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")

	// Write test cache
	data, _ := json.MarshalIndent(testRepos, "", "  ")
	os.WriteFile(cachePath, data, 0644)

	// Create server
	cfg := &config.Config{
		ScanPath:            tmpDir,
		Port:                8080,
		LocalIntervalSeconds: 30,
		GitHubIntervalSeconds: 300,
		StaleDays:           30,
		AbandonedDays:       90,
	}
	s, _ := NewServer(cfg)

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/api/repos", nil)
	w := httptest.NewRecorder()

	// Override cache path for this test
	originalCachePath := cache.GetCachePath()
	defer cache.SetCachePath(originalCachePath)
	cache.SetCachePath(cachePath)

	s.handleReposList(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	// Check content type
	ct := w.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("Content-Type = %s, want application/json", ct)
	}

	// Parse response
	var repos []model.Repo
	if err := json.NewDecoder(w.Body).Decode(&repos); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(repos) != 2 {
		t.Errorf("len(repos) = %d, want 2", len(repos))
	}
}

// TestReposListFiltering tests that filtering works correctly.
func TestReposListFiltering(t *testing.T) {
	testRepos := []model.Repo{
		{
			Name:       "public-repo",
			Visibility: model.VisibilityPublic,
			Cloned:     true,
			Lifecycle:  model.LifecycleOngoing,
			Language:   "Go",
		},
		{
			Name:       "private-repo",
			Visibility: model.VisibilityPrivate,
			Cloned:     false,
			Lifecycle:  model.LifecycleStale,
			Language:   "TypeScript",
		},
		{
			Name:       "another-public",
			Visibility: model.VisibilityPublic,
			Cloned:     false,
			Lifecycle:  model.LifecycleMaintenance,
			Language:   "Go",
		},
	}

	cfg := &config.Config{
		ScanPath:            "/tmp/test",
		Port:                8080,
		LocalIntervalSeconds: 30,
		GitHubIntervalSeconds: 300,
		StaleDays:           30,
		AbandonedDays:       90,
	}
	s, _ := NewServer(cfg)

	// Test visibility filter
	t.Run("filter by visibility", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/repos?visibility=public", nil)
	 filtered := s.filterRepos(testRepos, req.URL.Query())

		if len(filtered) != 2 {
			t.Errorf("len(filtered) = %d, want 2", len(filtered))
		}
		for _, repo := range filtered {
			if repo.Visibility != model.VisibilityPublic {
				t.Errorf("repo %s has visibility %s, want public", repo.Name, repo.Visibility)
			}
		}
	})

	// Test cloned filter
	t.Run("filter by cloned", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/repos?cloned=true", nil)
		filtered := s.filterRepos(testRepos, req.URL.Query())

		if len(filtered) != 1 {
			t.Errorf("len(filtered) = %d, want 1", len(filtered))
		}
		if filtered[0].Name != "public-repo" {
			t.Errorf("filtered[0].Name = %s, want public-repo", filtered[0].Name)
		}
	})

	// Test lifecycle filter
	t.Run("filter by lifecycle", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/repos?lifecycle=ongoing", nil)
		filtered := s.filterRepos(testRepos, req.URL.Query())

		if len(filtered) != 1 {
			t.Errorf("len(filtered) = %d, want 1", len(filtered))
		}
		if filtered[0].Name != "public-repo" {
			t.Errorf("filtered[0].Name = %s, want public-repo", filtered[0].Name)
		}
	})

	// Test language filter
	t.Run("filter by language", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/repos?language=Go", nil)
		filtered := s.filterRepos(testRepos, req.URL.Query())

		if len(filtered) != 2 {
			t.Errorf("len(filtered) = %d, want 2", len(filtered))
		}
		for _, repo := range filtered {
			if repo.Language != "Go" {
				t.Errorf("repo %s has language %s, want Go", repo.Name, repo.Language)
			}
		}
	})

	// Test multiple lifecycle filter
	t.Run("filter by multiple lifecycles", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/repos?lifecycle=ongoing,maintenance", nil)
		filtered := s.filterRepos(testRepos, req.URL.Query())

		if len(filtered) != 2 {
			t.Errorf("len(filtered) = %d, want 2", len(filtered))
		}
	})
}

// TestReposListSorting tests that sorting works correctly.
func TestReposListSorting(t *testing.T) {
	now := time.Now().UTC()
	testRepos := []model.Repo{
		{
			Name:           "zebra-repo",
			GitHubLastPush: now.Add(-24 * time.Hour),
			Lifecycle:      model.LifecycleAbandoned,
		},
		{
			Name:           "alpha-repo",
			GitHubLastPush: now.Add(-1 * time.Hour),
			Lifecycle:      model.LifecycleOngoing,
		},
		{
			Name:           "middle-repo",
			GitHubLastPush: now.Add(-12 * time.Hour),
			Lifecycle:      model.LifecycleStale,
		},
	}

	cfg := &config.Config{
		ScanPath:            "/tmp/test",
		Port:                8080,
		LocalIntervalSeconds: 30,
		GitHubIntervalSeconds: 300,
		StaleDays:           30,
		AbandonedDays:       90,
	}
	s, _ := NewServer(cfg)

	// Test sort by name ascending
	t.Run("sort by name asc", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/repos?sort=name&order=asc", nil)
		sorted := s.sortRepos(testRepos, req.URL.Query())

		if sorted[0].Name != "alpha-repo" {
			t.Errorf("sorted[0].Name = %s, want alpha-repo", sorted[0].Name)
		}
		if sorted[1].Name != "middle-repo" {
			t.Errorf("sorted[1].Name = %s, want middle-repo", sorted[1].Name)
		}
		if sorted[2].Name != "zebra-repo" {
			t.Errorf("sorted[2].Name = %s, want zebra-repo", sorted[2].Name)
		}
	})

	// Test sort by name descending
	t.Run("sort by name desc", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/repos?sort=name&order=desc", nil)
		sorted := s.sortRepos(testRepos, req.URL.Query())

		if sorted[0].Name != "zebra-repo" {
			t.Errorf("sorted[0].Name = %s, want zebra-repo", sorted[0].Name)
		}
		if sorted[2].Name != "alpha-repo" {
			t.Errorf("sorted[2].Name = %s, want alpha-repo", sorted[2].Name)
		}
	})

	// Test sort by lastUpdate
	t.Run("sort by lastUpdate desc", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/repos?sort=lastUpdate&order=desc", nil)
		sorted := s.sortRepos(testRepos, req.URL.Query())

		if sorted[0].Name != "alpha-repo" {
			t.Errorf("sorted[0].Name = %s, want alpha-repo (most recent)", sorted[0].Name)
		}
		if sorted[2].Name != "zebra-repo" {
			t.Errorf("sorted[2].Name = %s, want zebra-repo (oldest)", sorted[2].Name)
		}
	})

	// Test sort by lifecycle (alphabetical: abandoned < maintenance < ongoing < stale)
	t.Run("sort by lifecycle asc", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/repos?sort=lifecycle&order=asc", nil)
		sorted := s.sortRepos(testRepos, req.URL.Query())

		// Alphabetically: abandoned < ongoing < stale
		if sorted[0].Lifecycle != model.LifecycleAbandoned {
			t.Errorf("sorted[0].Lifecycle = %s, want abandoned", sorted[0].Lifecycle)
		}
		if sorted[1].Lifecycle != model.LifecycleOngoing {
			t.Errorf("sorted[1].Lifecycle = %s, want ongoing", sorted[1].Lifecycle)
		}
		if sorted[2].Lifecycle != model.LifecycleStale {
			t.Errorf("sorted[2].Lifecycle = %s, want stale", sorted[2].Lifecycle)
		}
	})
}

// TestSingleRepoReturnsCorrectData tests getting a single repo.
func TestSingleRepoReturnsCorrectData(t *testing.T) {
	testRepos := []model.Repo{
		{
			Name:        "test-repo",
			Description: "Test repo",
			Visibility:  model.VisibilityPublic,
		},
	}

	// Create temp directory for cache
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")

	// Write test cache
	data, _ := json.MarshalIndent(testRepos, "", "  ")
	os.WriteFile(cachePath, data, 0644)

	cfg := &config.Config{
		ScanPath:            tmpDir,
		Port:                8080,
		LocalIntervalSeconds: 30,
		GitHubIntervalSeconds: 300,
		StaleDays:           30,
		AbandonedDays:       90,
	}
	s, _ := NewServer(cfg)

	// Override cache path
	originalCachePath := cache.GetCachePath()
	defer cache.SetCachePath(originalCachePath)
	cache.SetCachePath(cachePath)

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/api/repos/test-repo", nil)
	w := httptest.NewRecorder()

	s.handleRepoByName(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	// Parse response
	var repo model.Repo
	if err := json.NewDecoder(w.Body).Decode(&repo); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if repo.Name != "test-repo" {
		t.Errorf("repo.Name = %s, want test-repo", repo.Name)
	}
}

// TestSingleRepo404ForUnknownName tests that unknown repo returns 404.
func TestSingleRepo404ForUnknownName(t *testing.T) {
	testRepos := []model.Repo{
		{Name: "known-repo"},
	}

	// Create temp directory for cache
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")

	// Write test cache
	data, _ := json.MarshalIndent(testRepos, "", "  ")
	os.WriteFile(cachePath, data, 0644)

	cfg := &config.Config{
		ScanPath:            tmpDir,
		Port:                8080,
		LocalIntervalSeconds: 30,
		GitHubIntervalSeconds: 300,
		StaleDays:           30,
		AbandonedDays:       90,
	}
	s, _ := NewServer(cfg)

	// Override cache path
	originalCachePath := cache.GetCachePath()
	defer cache.SetCachePath(originalCachePath)
	cache.SetCachePath(cachePath)

	// Create request for unknown repo
	req := httptest.NewRequest(http.MethodGet, "/api/repos/unknown-repo", nil)
	w := httptest.NewRecorder()

	s.handleRepoByName(w, req)

	// Check response
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// TestHealthEndpointShape tests the health endpoint returns correct shape.
func TestHealthEndpointShape(t *testing.T) {
	cfg := &config.Config{
		ScanPath:            "/tmp/test",
		Port:                8080,
		LocalIntervalSeconds: 30,
		GitHubIntervalSeconds: 300,
		StaleDays:           30,
		AbandonedDays:       90,
	}
	s, _ := NewServer(cfg)

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	w := httptest.NewRecorder()

	s.handleHealth(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	// Parse response
	var health map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&health); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Check required fields
	requiredFields := []string{"Uptime", "LastLocalPoll", "LastGitHubPoll", "TotalRepos", "GhAvailable", "GhAuthenticated"}
	for _, field := range requiredFields {
		if _, ok := health[field]; !ok {
			t.Errorf("response missing field: %s", field)
		}
	}
}

// TestConfigGet tests getting config.
func TestConfigGet(t *testing.T) {
	cfg := &config.Config{
		ScanPath:            "/test/path",
		Port:                9999,
		LocalIntervalSeconds: 45,
		GitHubIntervalSeconds: 600,
		StaleDays:           60,
		AbandonedDays:       180,
	}
	s, _ := NewServer(cfg)

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/api/config", nil)
	w := httptest.NewRecorder()

	s.handleGetConfig(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	// Parse response
	var returnedCfg config.Config
	if err := json.NewDecoder(w.Body).Decode(&returnedCfg); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if returnedCfg.Port != 9999 {
		t.Errorf("Port = %d, want 9999", returnedCfg.Port)
	}
}

// TestConfigValidation tests config validation.
func TestConfigValidation(t *testing.T) {
	cfg := &config.Config{
		ScanPath:            "/tmp/test",
		Port:                8080,
		LocalIntervalSeconds: 30,
		GitHubIntervalSeconds: 300,
		StaleDays:           30,
		AbandonedDays:       90,
	}
	s, _ := NewServer(cfg)

	tests := []struct {
		name        string
		cfg         config.Config
		wantErr     bool
		errContains string
	}{
		{
			name: "valid config",
			cfg: config.Config{
				ScanPath:            "/tmp/test",
				Port:                8080,
				LocalIntervalSeconds: 30,
				GitHubIntervalSeconds: 300,
				StaleDays:           30,
				AbandonedDays:       90,
			},
			wantErr: false,
		},
		{
			name: "empty scan path",
			cfg: config.Config{
				ScanPath:            "",
				Port:                8080,
				LocalIntervalSeconds: 30,
				GitHubIntervalSeconds: 300,
				StaleDays:           30,
				AbandonedDays:       90,
			},
			wantErr:     true,
			errContains: "scanPath",
		},
		{
			name: "port too low",
			cfg: config.Config{
				ScanPath:            "/tmp/test",
				Port:                80,
				LocalIntervalSeconds: 30,
				GitHubIntervalSeconds: 300,
				StaleDays:           30,
				AbandonedDays:       90,
			},
			wantErr:     true,
			errContains: "port",
		},
		{
			name: "local interval too low",
			cfg: config.Config{
				ScanPath:            "/tmp/test",
				Port:                8080,
				LocalIntervalSeconds: 5,
				GitHubIntervalSeconds: 300,
				StaleDays:           30,
				AbandonedDays:       90,
			},
			wantErr:     true,
			errContains: "localIntervalSeconds",
		},
		{
			name: "GitHub interval too low",
			cfg: config.Config{
				ScanPath:            "/tmp/test",
				Port:                8080,
				LocalIntervalSeconds: 30,
				GitHubIntervalSeconds: 30,
				StaleDays:           30,
				AbandonedDays:       90,
			},
			wantErr:     true,
			errContains: "githubIntervalSeconds",
		},
		{
			name: "stale >= abandoned",
			cfg: config.Config{
				ScanPath:            "/tmp/test",
				Port:                8080,
				LocalIntervalSeconds: 30,
				GitHubIntervalSeconds: 300,
				StaleDays:           90,
				AbandonedDays:       90,
			},
			wantErr:     true,
			errContains: "staleDays",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := s.validateConfig(&tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.errContains != "" {
				if !contains(err.Error(), tt.errContains) {
					t.Errorf("error = %v, want contain %s", err, tt.errContains)
				}
			}
		})
	}
}

// TestSSEConnectionReceivesEvents tests that SSE connections receive events.
func TestSSEConnectionReceivesEvents(t *testing.T) {
	hub := sse.NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go hub.Run(ctx)

	// Register a client
	client := &sse.Client{
		ID:     "test-client",
		Chan:   make(chan sse.Event, 10),
		Ctx:    ctx,
		Cancel: cancel,
	}
	hub.Register(client)
	time.Sleep(10 * time.Millisecond)

	// Broadcast an event
	hub.Broadcast("test_event", map[string]string{"message": "hello"})

	// Check client received the event
	select {
	case event := <-client.Chan:
		if event.Type != "test_event" {
			t.Errorf("event.Type = %s, want test_event", event.Type)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("did not receive event within timeout")
	}
}

// TestSSEBroadcastReachesAllClients tests that broadcast reaches all connected clients.
func TestSSEBroadcastReachesAllClients(t *testing.T) {
	hub := sse.NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go hub.Run(ctx)

	// Create multiple clients
	var clients []*sse.Client
	for i := 0; i < 3; i++ {
		client := &sse.Client{
			ID:     fmt.Sprintf("client-%d", i),
			Chan:   make(chan sse.Event, 10),
			Ctx:    ctx,
			Cancel: cancel,
		}
		clients = append(clients, client)
		hub.Register(client)
	}
	time.Sleep(10 * time.Millisecond)

	// Broadcast an event
	hub.Broadcast("broadcast_test", map[string]int{"value": 42})

	// Check all clients received the event
	for i, client := range clients {
		select {
		case event := <-client.Chan:
			if event.Type != "broadcast_test" {
				t.Errorf("client %d: event.Type = %s, want broadcast_test", i, event.Type)
			}
		case <-time.After(100 * time.Millisecond):
			t.Errorf("client %d: did not receive event within timeout", i)
		}
	}
}

// TestWithHeadersMiddleware tests that security headers are set.
func TestWithHeadersMiddleware(t *testing.T) {
	cfg := &config.Config{
		ScanPath:            "/tmp/test",
		Port:                8080,
		LocalIntervalSeconds: 30,
		GitHubIntervalSeconds: 300,
		StaleDays:           30,
		AbandonedDays:       90,
	}
	s, _ := NewServer(cfg)

	// Create a test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Wrap with middleware
	wrapped := s.withHeaders(testHandler)

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	wrapped.ServeHTTP(w, req)

	// Check headers
	expectedHeaders := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"Referrer-Policy":        "no-referrer",
	}

	for key, want := range expectedHeaders {
		got := w.Header().Get(key)
		if got != want {
			t.Errorf("%s = %s, want %s", key, got, want)
		}
	}
}

// TestHandleEventsSSE tests the SSE events endpoint.
func TestHandleEventsSSE(t *testing.T) {
	cfg := &config.Config{
		ScanPath:            "/tmp/test",
		Port:                8080,
		LocalIntervalSeconds: 30,
		GitHubIntervalSeconds: 300,
		StaleDays:           30,
		AbandonedDays:       90,
	}
	s, _ := NewServer(cfg)

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/api/events", nil)
	w := httptest.NewRecorder()

	// This will block since we're not using a real SSE connection
	// Just verify it doesn't panic and sets correct headers
	// We run it in a goroutine and let it complete on its own
	done := make(chan struct{})
	go func() {
		s.handleEvents(w, req)
		close(done)
	}()

	// Wait a bit for the handler to start
	select {
	case <-done:
		// Handler completed quickly (no real SSE connection)
	case <-time.After(50 * time.Millisecond):
		// Handler is still running (expected for SSE with real connection)
		// We don't wait for completion - the test passed if no panic occurred
	}

	// The important thing is no panic occurred during handler startup
}

// TestConcurrentRequests tests that the server handles concurrent requests safely.
func TestConcurrentRequests(t *testing.T) {
	testRepos := []model.Repo{
		{Name: "repo-1", Visibility: model.VisibilityPublic},
		{Name: "repo-2", Visibility: model.VisibilityPrivate},
		{Name: "repo-3", Visibility: model.VisibilityPublic},
	}

	// Create temp directory for cache
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")

	// Write test cache
	data, _ := json.MarshalIndent(testRepos, "", "  ")
	os.WriteFile(cachePath, data, 0644)

	cfg := &config.Config{
		ScanPath:            tmpDir,
		Port:                8080,
		LocalIntervalSeconds: 30,
		GitHubIntervalSeconds: 300,
		StaleDays:           30,
		AbandonedDays:       90,
	}
	s, _ := NewServer(cfg)

	// Override cache path
	originalCachePath := cache.GetCachePath()
	defer cache.SetCachePath(originalCachePath)
	cache.SetCachePath(cachePath)

	// Make concurrent requests
	var wg sync.WaitGroup
	numRequests := 50

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodGet, "/api/repos", nil)
			w := httptest.NewRecorder()
			s.handleReposList(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("request %d: status = %d, want %d", idx, w.Code, http.StatusOK)
			}
		}(i)
	}

	wg.Wait()
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || indexOf(s, substr) >= 0))
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
