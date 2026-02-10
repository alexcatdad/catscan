// Package poller manages background polling for local and GitHub data.
//
// Two independent goroutines poll local git state and GitHub metadata
// on configurable intervals. Results are merged, cached, and broadcast
// via SSE to connected clients.
package poller

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/alexcatdad/catscan/internal/cache"
	"github.com/alexcatdad/catscan/internal/config"
	"github.com/alexcatdad/catscan/internal/model"
	"github.com/alexcatdad/catscan/internal/scanner"
	"github.com/alexcatdad/catscan/internal/sse"
)

// Poller manages background polling for repository data.
type Poller struct {
	cfg             *config.Config
	hub             *sse.Hub
	state           cache.RepoState
	stateMu         sync.RWMutex
	lastLocalPoll   time.Time
	lastGitHubPoll  time.Time
	lastLocalPollMu sync.RWMutex
	lastGitHubPollMu sync.RWMutex

	// Previous data for change detection
	previousRepos   []model.Repo
	previousReposMu sync.RWMutex
}

// NewPoller creates a new Poller.
func NewPoller(cfg *config.Config, hub *sse.Hub) *Poller {
	return &Poller{
		cfg:   cfg,
		hub:   hub,
		state: make(cache.RepoState),
	}
}

// Start starts both local and GitHub pollers.
// It should be run in a separate goroutine.
func (p *Poller) Start(ctx context.Context) {
	// Load initial state from disk
	if state, err := cache.ReadState(); err == nil {
		p.state = state
	}

	// Load initial cache and serve immediately
	if repos, err := cache.ReadRepos(); err == nil && len(repos) > 0 {
		p.hub.Broadcast("repos_updated", repos)
		p.setPreviousRepos(repos)
	}

	// Start local poller
	go p.runLocalPoller(ctx)

	// Start GitHub poller
	go p.runGitHubPoller(ctx)

	// Start heartbeat goroutine to keep SSE connections alive
	go p.runHeartbeat(ctx)
}

// runLocalPoller runs the local scanner on a configurable interval.
func (p *Poller) runLocalPoller(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(p.cfg.LocalIntervalSeconds) * time.Second)
	defer ticker.Stop()

	// First run immediately
	p.localPoll(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.localPoll(ctx)
		}
	}
}

// runGitHubPoller runs the GitHub scanner on a configurable interval.
func (p *Poller) runGitHubPoller(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(p.cfg.GitHubIntervalSeconds) * time.Second)
	defer ticker.Stop()

	// First run immediately
	p.githubPoll(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.githubPoll(ctx)
		}
	}
}

// localPoll performs a single local poll cycle.
func (p *Poller) localPoll(ctx context.Context) {
	// Discover local repos
	localRepoNames, err := scanner.DiscoverLocalRepos(p.cfg.ScanPath)
	if err != nil {
		log.Printf("local poll error: %v", err)
		return
	}

	// Build local repo map
	localRepos := make(map[string]scanner.LocalRepo)
	for _, name := range localRepoNames {
		clonedMap := scanner.FindClonedRepos([]string{name}, p.cfg.ScanPath)
		if path, ok := clonedMap[name]; ok {
			branch, dirty, lastCommit, err := scanner.GetGitState(path)
			if err != nil {
				log.Printf("error getting git state for %s: %v", name, err)
				continue
			}
			localRepos[name] = scanner.LocalRepo{
				Name:       name,
				Path:       path,
				Branch:     branch,
				Dirty:      dirty,
				LastCommit: lastCommit,
			}
		}
	}

	// Get previous GitHub data from cache
	var githubRepos []scanner.GitHubRepo
	if cachedRepos, err := cache.ReadRepos(); err == nil {
		// Extract GitHub repo data from cached repos
		for _, repo := range cachedRepos {
			ghRepo := scanner.GitHubRepo{
				Name:         repo.Name,
				Description:  repo.Description,
				Visibility:   string(repo.Visibility),
				HomepageURL:  repo.HomepageURL,
				Topics:       repo.Topics,
				HasPages:     repo.HasPages,
			}
			if repo.Language != "" {
				ghRepo.PrimaryLanguage = &scanner.PrimaryLanguage{Name: repo.Language}
			}
			if repo.LatestRelease != nil {
				ghRepo.LatestRelease = &scanner.LatestRelease{
					TagName:     repo.LatestRelease.TagName,
					PublishedAt: repo.LatestRelease.PublishedAt.Format(time.RFC3339),
				}
			}
			githubRepos = append(githubRepos, ghRepo)
		}
	}

	// Merge data
	thresholds := model.LifecycleThresholds{
		StaleDays:     p.cfg.StaleDays,
		AbandonedDays: p.cfg.AbandonedDays,
	}

	repos := scanner.Merge(localRepos, githubRepos, p.cfg.ScanPath, p.state, thresholds)

	// Detect changes and emit granular events
	p.detectAndEmitChanges(repos, "local")

	// Update cache
	if err := cache.WriteRepos(repos); err != nil {
		log.Printf("error writing cache: %v", err)
	}

	// Broadcast update
	p.hub.Broadcast("repos_updated", repos)

	// Update previous repos and poll time
	p.setPreviousRepos(repos)
	p.setLastLocalPoll(time.Now())
}

// githubPoll performs a single GitHub poll cycle.
func (p *Poller) githubPoll(ctx context.Context) {
	// List GitHub repos
	githubRepos, err := scanner.ListGitHubRepos(p.cfg.GitHubOwner)
	if err != nil {
		if scanner.IsGHNotFound(err) {
			log.Printf("gh CLI not found")
			p.hub.Broadcast("error", map[string]string{
				"type":  "gh_not_found",
				"error": "gh CLI not found. Please install gh CLI.",
			})
		} else if scanner.IsGHAuthError(err) {
			log.Printf("gh CLI not authenticated")
			p.hub.Broadcast("error", map[string]string{
				"type":  "gh_auth_error",
				"error": "gh CLI not authenticated. Please run 'gh auth login'.",
			})
		} else {
			log.Printf("github poll error: %v", err)
		}
		return
	}

	// Get local data from cache
	var localRepos map[string]scanner.LocalRepo
	if cachedRepos, err := cache.ReadRepos(); err == nil {
		localRepos = make(map[string]scanner.LocalRepo)
		for _, repo := range cachedRepos {
			if repo.Cloned {
				localRepos[repo.Name] = scanner.LocalRepo{
					Name:       repo.Name,
					Path:       repo.LocalPath,
					Branch:     repo.Branch,
					Dirty:      repo.Dirty,
					LastCommit: repo.LocalLastCommit,
				}
			}
		}
	}

	// Fetch additional GitHub data for each repo
	for i := range githubRepos {
		repo := &githubRepos[i]

		// Get PR count
		prCount, err := scanner.GetPROpenCount(p.cfg.GitHubOwner, repo.Name)
		if err != nil {
			log.Printf("error getting PRs for %s: %v", repo.Name, err)
		}
		_ = prCount // Will be used when we extend the merge

		// Get Actions status
		actionsStatus, err := scanner.GetActionsStatus(p.cfg.GitHubOwner, repo.Name)
		if err != nil {
			log.Printf("error getting Actions status for %s: %v", repo.Name, err)
		}
		_ = actionsStatus // Will be used when we extend the merge

		// Get file presence
		filePresence, err := scanner.GetFilePresence(p.cfg.GitHubOwner, repo.Name)
		if err != nil {
			log.Printf("error getting file presence for %s: %v", repo.Name, err)
		}
		_ = filePresence // Will be used when we extend the merge
	}

	// Merge data
	thresholds := model.LifecycleThresholds{
		StaleDays:     p.cfg.StaleDays,
		AbandonedDays: p.cfg.AbandonedDays,
	}

	repos := scanner.Merge(localRepos, githubRepos, p.cfg.ScanPath, p.state, thresholds)

	// Detect changes and emit granular events
	p.detectAndEmitChanges(repos, "github")

	// Update state with new release tags
	p.updateReleaseState(repos)

	// Update cache
	if err := cache.WriteRepos(repos); err != nil {
		log.Printf("error writing cache: %v", err)
	}

	// Broadcast update
	p.hub.Broadcast("github_updated", repos)

	// Update previous repos and poll time
	p.setPreviousRepos(repos)
	p.setLastGitHubPoll(time.Now())
}

// detectAndEmitChanges compares new repos with previous and emits granular events.
func (p *Poller) detectAndEmitChanges(newRepos []model.Repo, source string) {
	previousRepos := p.getPreviousRepos()

	// Build previous repo map
	prevMap := make(map[string]model.Repo)
	for _, repo := range previousRepos {
		prevMap[repo.Name] = repo
	}

	// Check for changes
	for _, newRepo := range newRepos {
		prevRepo, ok := prevMap[newRepo.Name]
		if !ok {
			continue
		}

		// Check for Actions status change
		if prevRepo.ActionsStatus != newRepo.ActionsStatus {
			if p.cfg.Notifications.ActionsChanged {
				p.sendNotification("actions_changed", newRepo.Name, formatActionsStatusChange(newRepo.ActionsStatus))
			}
			p.hub.Broadcast("actions_changed", map[string]interface{}{
				"repo":        newRepo.Name,
				"oldStatus":   prevRepo.ActionsStatus,
				"newStatus":   newRepo.ActionsStatus,
			})
		}

		// Check for new release
		if newRepo.NewRelease {
			if p.cfg.Notifications.NewRelease {
				releaseName := "unknown"
				if newRepo.LatestRelease != nil {
					releaseName = newRepo.LatestRelease.TagName
				}
				p.sendNotification("new_release", newRepo.Name, releaseName)
			}
			p.hub.Broadcast("new_release", map[string]interface{}{
				"repo":     newRepo.Name,
				"tagName":  newRepo.LatestRelease.TagName,
				"released": newRepo.LatestRelease.PublishedAt,
			})
		}

		// Check for opened PRs
		if newRepo.OpenPRs > prevRepo.OpenPRs {
			if p.cfg.Notifications.PROpened {
				p.sendNotification("pr_opened", newRepo.Name, fmt.Sprintf("%d open", newRepo.OpenPRs))
			}
			p.hub.Broadcast("pr_opened", map[string]interface{}{
				"repo":     newRepo.Name,
				"oldCount": prevRepo.OpenPRs,
				"newCount": newRepo.OpenPRs,
			})
		}
	}
}

// updateReleaseState updates the state with new release tags.
func (p *Poller) updateReleaseState(repos []model.Repo) {
	p.stateMu.Lock()
	defer p.stateMu.Unlock()

	if p.state == nil {
		p.state = make(cache.RepoState)
	}

	for _, repo := range repos {
		if repo.LatestRelease != nil {
			if p.state[repo.Name] == nil {
				p.state[repo.Name] = &cache.RepoStateEntry{}
			}
			p.state[repo.Name].LastSeenReleaseTag = repo.LatestRelease.TagName
		}
	}

	// Save state
	if err := cache.WriteState(p.state); err != nil {
		log.Printf("error writing state: %v", err)
	}
}

// sendNotification sends a macOS notification.
func (p *Poller) sendNotification(eventType, repo, message string) {
	SendNotification(eventType, repo, message)
}

// runHeartbeat sends a comment every 30 seconds to keep SSE connections alive.
func (p *Poller) runHeartbeat(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Send a comment as heartbeat
			p.hub.Broadcast("heartbeat", map[string]string{"time": time.Now().Format(time.RFC3339)})
		}
	}
}

// GetLastLocalPoll returns the time of the last local poll.
func (p *Poller) GetLastLocalPoll() time.Time {
	p.lastLocalPollMu.RLock()
	defer p.lastLocalPollMu.RUnlock()
	return p.lastLocalPoll
}

// GetLastGitHubPoll returns the time of the last GitHub poll.
func (p *Poller) GetLastGitHubPoll() time.Time {
	p.lastGitHubPollMu.RLock()
	defer p.lastGitHubPollMu.RUnlock()
	return p.lastGitHubPoll
}

// setLastLocalPoll sets the time of the last local poll.
func (p *Poller) setLastLocalPoll(t time.Time) {
	p.lastLocalPollMu.Lock()
	defer p.lastLocalPollMu.Unlock()
	p.lastLocalPoll = t
}

// setLastGitHubPoll sets the time of the last GitHub poll.
func (p *Poller) setLastGitHubPoll(t time.Time) {
	p.lastGitHubPollMu.Lock()
	defer p.lastGitHubPollMu.Unlock()
	p.lastGitHubPoll = t
}

// setPreviousRepos sets the previous repo list for change detection.
func (p *Poller) setPreviousRepos(repos []model.Repo) {
	p.previousReposMu.Lock()
	defer p.previousReposMu.Unlock()
	p.previousRepos = repos
}

// getPreviousRepos gets the previous repo list for change detection.
func (p *Poller) getPreviousRepos() []model.Repo {
	p.previousReposMu.RLock()
	defer p.previousReposMu.RUnlock()
	return p.previousRepos
}

// formatActionsStatusChange formats an Actions status change for notification.
func formatActionsStatusChange(status model.ActionsStatus) string {
	switch status {
	case model.ActionsStatusPassing:
		return "CI passing"
	case model.ActionsStatusFailing:
		return "CI failing"
	default:
		return "CI status unknown"
	}
}
