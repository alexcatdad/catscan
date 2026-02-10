// Package server provides the HTTP server for CatScan.
//
// The server package handles HTTP server, routes, and static file serving.
// SSE functionality is provided by the sse package.
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/alexcatdad/catscan/internal/cache"
	"github.com/alexcatdad/catscan/internal/config"
	"github.com/alexcatdad/catscan/internal/model"
	"github.com/alexcatdad/catscan/internal/poller"
	"github.com/alexcatdad/catscan/internal/scanner"
	"github.com/alexcatdad/catscan/internal/sse"
)

// Server represents the CatScan HTTP server.
type Server struct {
	cfg              *config.Config
	hub              *sse.Hub
	poller           *poller.Poller
	server           *http.Server
	listener         net.Listener
	distDir          string
	startTime        time.Time
	shutdownCtx      context.Context
	shutdownCancel   context.CancelFunc
	wg               sync.WaitGroup
	mu               sync.RWMutex
}

// NewServer creates a new Server.
func NewServer(cfg *config.Config) (*Server, error) {
	hub := sse.NewHub()
	p := poller.NewPoller(cfg, hub)

	s := &Server{
		cfg:       cfg,
		hub:       hub,
		poller:    p,
		startTime: time.Now(),
		distDir:   "dist",
	}

	// Create shutdown context
	s.shutdownCtx, s.shutdownCancel = context.WithCancel(context.Background())

	return s, nil
}

// Start starts the HTTP server.
// This blocks until the server is stopped.
func (s *Server) Start() error {
	// Create listener
	addr := fmt.Sprintf("127.0.0.1:%d", s.cfg.Port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}
	s.listener = listener

	// Create HTTP server
	mux := http.NewServeMux()
	s.server = &http.Server{
		Handler:     s.withHeaders(mux),
		ReadTimeout: 15 * time.Second,
		// WriteTimeout must be 0 for SSE — a non-zero value kills
		// long-lived connections after the timeout elapses.
		WriteTimeout: 0,
		IdleTimeout:  60 * time.Second,
	}

	// Set up routes
	s.setupRoutes(mux)

	// Start SSE hub
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.hub.Run(s.shutdownCtx)
	}()

	// Start pollers
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.poller.Start(s.shutdownCtx)
	}()

	log.Printf("CatScan starting on http://%s", addr)

	// Start server in a goroutine
	serverErr := make(chan error, 1)
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		serverErr <- s.server.Serve(listener)
	}()

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigChan:
		log.Printf("Received signal %v, shutting down...", sig)
	case err := <-serverErr:
		log.Printf("Server error: %v", err)
		return err
	}

	// Graceful shutdown
	s.Shutdown()

	return <-serverErr
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown() {
	log.Println("Shutting down...")

	// Cancel pollers and SSE hub
	s.shutdownCancel()

	// Shutdown HTTP server with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := s.server.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	// Close listener
	if s.listener != nil {
		s.listener.Close()
	}

	// Wait for all goroutines to finish
	s.wg.Wait()

	log.Println("Shutdown complete")
}

// withHeaders wraps the handler with security headers.
func (s *Server) withHeaders(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Security headers
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")

		h.ServeHTTP(w, r)
	})
}

// setupRoutes sets up all HTTP routes.
func (s *Server) setupRoutes(mux *http.ServeMux) {
	// API routes
	mux.HandleFunc("/api/repos", s.handleReposList)
	mux.HandleFunc("/api/repos/", s.handleRepoByName)
	mux.HandleFunc("/api/config", s.handleConfig)
	mux.HandleFunc("/api/health", s.handleHealth)
	mux.HandleFunc("/api/events", s.handleEvents)

	// Static file serving for the Svelte frontend (dist/ directory)
	mux.Handle("/", http.FileServer(http.Dir(s.distDir)))
}

// handleReposList handles GET /api/repos with filtering and sorting.
func (s *Server) handleReposList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "method not allowed"})
		return
	}

	// Get repos from cache
	repos, err := cache.ReadRepos()
	if err != nil {
		http.Error(w, "Failed to read cache", http.StatusInternalServerError)
		return
	}

	// Apply filters
	repos = s.filterRepos(repos, r.URL.Query())

	// Apply sorting
	repos = s.sortRepos(repos, r.URL.Query())

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(repos)
}

// handleRepoByName handles GET /api/repos/:name.
func (s *Server) handleRepoByName(w http.ResponseWriter, r *http.Request) {
	// Check if it's the clone endpoint
	if strings.HasSuffix(r.URL.Path, "/clone") {
		s.handleClone(w, r)
		return
	}

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "method not allowed"})
		return
	}

	// Extract repo name from /api/repos/{name}
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/repos/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		http.Error(w, "Repo name required", http.StatusBadRequest)
		return
	}
	repoName := parts[0]

	// Get repos from cache
	repos, err := cache.ReadRepos()
	if err != nil {
		http.Error(w, "Failed to read cache", http.StatusInternalServerError)
		return
	}

	// Find the requested repo
	for _, repo := range repos {
		if repo.Name == repoName {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(repo)
			return
		}
	}

	// Not found
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)
	json.NewEncoder(w).Encode(map[string]string{"error": "repository not found"})
}

// handleClone handles POST /api/repos/:name/clone.
func (s *Server) handleClone(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "method not allowed"})
		return
	}

	// Extract repo name from path
	parts := strings.Split(strings.TrimPrefix(strings.TrimSuffix(r.URL.Path, "/clone"), "/api/repos/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		http.Error(w, "Repo name required", http.StatusBadRequest)
		return
	}
	repoName := parts[0]

	// Check if repo is already cloned locally
	cloned := scanner.FindClonedRepos([]string{repoName}, s.cfg.ScanPath)
	if _, ok := cloned[repoName]; ok {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]string{"error": "repository already cloned"})
		return
	}

	// Start clone asynchronously
	statusChan := scanner.CloneRepo(s.cfg.GitHubOwner, repoName, s.cfg.ScanPath)

	// Broadcast clone progress events in a goroutine
	go func() {
		for status := range statusChan {
			s.hub.Broadcast("clone_progress", map[string]interface{}{
				"repo":  status.Repo,
				"state": status.State,
				"error": status.Error,
			})
		}
	}()

	// Return 202 Accepted
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"status": "clone started"})
}

// handleConfig handles GET/PUT /api/config.
func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleGetConfig(w, r)
	case http.MethodPut:
		s.handlePutConfig(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "method not allowed"})
	}
}

// handleGetConfig handles GET /api/config.
func (s *Server) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.cfg)
}

// handlePutConfig handles PUT /api/config.
func (s *Server) handlePutConfig(w http.ResponseWriter, r *http.Request) {
	var newCfg config.Config
	if err := json.NewDecoder(r.Body).Decode(&newCfg); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid JSON"})
		return
	}

	// Validate config
	if err := s.validateConfig(&newCfg); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	// Save config
	if err := config.Save(newCfg); err != nil {
		http.Error(w, "Failed to save config", http.StatusInternalServerError)
		return
	}

	// Update server config
	s.mu.Lock()
	s.cfg = &newCfg
	s.mu.Unlock()

	// Notify connected clients that config changed
	s.hub.Broadcast("config_updated", newCfg)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(newCfg)
}

// validateConfig validates the config values.
func (s *Server) validateConfig(cfg *config.Config) error {
	if cfg.ScanPath == "" {
		return fmt.Errorf("scanPath cannot be empty")
	}
	if cfg.Port < 1024 || cfg.Port > 65535 {
		return fmt.Errorf("port must be between 1024 and 65535")
	}
	if cfg.LocalIntervalSeconds < 10 {
		return fmt.Errorf("localIntervalSeconds must be at least 10")
	}
	if cfg.GitHubIntervalSeconds < 60 {
		return fmt.Errorf("githubIntervalSeconds must be at least 60")
	}
	if cfg.StaleDays < 1 {
		return fmt.Errorf("staleDays must be at least 1")
	}
	if cfg.AbandonedDays < 1 {
		return fmt.Errorf("abandonedDays must be at least 1")
	}
	if cfg.StaleDays >= cfg.AbandonedDays {
		return fmt.Errorf("staleDays must be less than abandonedDays")
	}
	return nil
}

// handleHealth handles GET /api/health.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "method not allowed"})
		return
	}

	// Get repo count
	repos, _ := cache.ReadRepos()

	// Check gh CLI availability and authentication
	ghAvailable := false
	ghAuthenticated := false
	if ghPath, err := exec.LookPath("gh"); err == nil {
		ghAvailable = true
		cmd := exec.Command(ghPath, "auth", "status")
		if cmd.Run() == nil {
			ghAuthenticated = true
		}
	}

	// Get poll times
	lastLocal := s.poller.GetLastLocalPoll()
	lastGitHub := s.poller.GetLastGitHubPoll()

	health := map[string]interface{}{
		"Uptime":          time.Since(s.startTime).String(),
		"LastLocalPoll":   lastLocal.Format(time.RFC3339),
		"LastGitHubPoll":  lastGitHub.Format(time.RFC3339),
		"TotalRepos":      len(repos),
		"GhAvailable":     ghAvailable,
		"GhAuthenticated": ghAuthenticated,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

// handleEvents handles GET /api/events for SSE connections.
func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// Generate unique client ID
	clientID := generateClientID()

	// Create SSE handler
	handler := sse.NewHandler(s.hub, clientID)

	// Send current repo list immediately
	repos, err := cache.ReadRepos()
	if err == nil && len(repos) > 0 {
		// Send directly to the client
		handler.GetClient().Chan <- sse.Event{
			Type: "repos_updated",
			Data: repos,
		}
	}

	// Serve SSE connection
	handler.ServeHTTP(w, r)
}

// filterRepos applies query parameter filters to the repo list.
func (s *Server) filterRepos(repos []model.Repo, query url.Values) []model.Repo {
	var result []model.Repo

	// Filter by lifecycle
	if lifecycle := query.Get("lifecycle"); lifecycle != "" {
		lifecycles := strings.Split(lifecycle, ",")
		for _, repo := range repos {
			for _, lc := range lifecycles {
				if string(repo.Lifecycle) == strings.TrimSpace(lc) {
					result = append(result, repo)
					break
				}
			}
		}
		repos = result
		result = nil
	}

	// Filter by visibility
	if visibility := query.Get("visibility"); visibility != "" {
		for _, repo := range repos {
			if string(repo.Visibility) == visibility {
				result = append(result, repo)
			}
		}
		repos = result
		result = nil
	}

	// Filter by cloned status
	if cloned := query.Get("cloned"); cloned != "" {
		clonedBool := cloned == "true"
		for _, repo := range repos {
			if repo.Cloned == clonedBool {
				result = append(result, repo)
			}
		}
		repos = result
		result = nil
	}

	// Filter by language
	if language := query.Get("language"); language != "" {
		for _, repo := range repos {
			if repo.Language == language {
				result = append(result, repo)
			}
		}
		repos = result
	}

	if result == nil {
		return repos
	}
	return result
}

// sortRepos applies sorting to the repo list.
func (s *Server) sortRepos(repos []model.Repo, query url.Values) []model.Repo {
	// Get sort field and order
	sortField := query.Get("sort")
	if sortField == "" {
		sortField = "name"
	}
	order := query.Get("order")
	if order == "" {
		order = "asc"
	}

	// Sort using stdlib sort.Slice
	sorted := make([]model.Repo, len(repos))
	copy(sorted, repos)

	desc := order == "desc"
	switch sortField {
	case "name":
		sort.Slice(sorted, func(i, j int) bool {
			if desc {
				return sorted[i].Name > sorted[j].Name
			}
			return sorted[i].Name < sorted[j].Name
		})
	case "lastUpdate":
		sort.Slice(sorted, func(i, j int) bool {
			if desc {
				return sorted[i].GitHubLastPush.After(sorted[j].GitHubLastPush)
			}
			return sorted[i].GitHubLastPush.Before(sorted[j].GitHubLastPush)
		})
	case "lifecycle":
		sort.Slice(sorted, func(i, j int) bool {
			if desc {
				return sorted[i].Lifecycle > sorted[j].Lifecycle
			}
			return sorted[i].Lifecycle < sorted[j].Lifecycle
		})
	}
	repos = sorted

	return repos
}

// generateClientID generates a unique client ID for SSE connections.
func generateClientID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
