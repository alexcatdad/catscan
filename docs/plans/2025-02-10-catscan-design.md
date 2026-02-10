# CatScan Design Document

**Date:** 2025-02-10
**Status:** Draft

## Problem Statement

A developer with 20+ repositories across GitHub needs a unified view of their entire project portfolio. Currently, tracking what's cloned locally, what's missing, which repos have proper public-facing metadata, and where CI is failing requires jumping between GitHub web, terminal, and manually maintained index files. This doesn't scale.

## What CatScan Is

A local-only background dashboard that continuously monitors all GitHub repos for a given user. It provides a single pane of glass showing clone status, git state, GitHub activity, and public visibility hygiene.

CatScan is **read-only** — the only write action it takes is cloning missing repos. Everything else (pulling, editing descriptions, fixing READMEs) is handled by the developer in their own workflow.

## Architecture

CatScan runs as a Go daemon managed by launchd, always on in the background.

### Components

**Go server** — binds to 127.0.0.1 on a configurable port. Runs two polling loops on separate goroutines: local git state every 60 seconds, GitHub data every 300 seconds. Serves the static frontend, exposes a REST API, and pushes live updates via Server-Sent Events.

**Svelte frontend** — a static SPA built at compile time, served by the Go binary. Connects to the SSE endpoint for live updates. All filtering and sorting happens client-side.

**paw-proxy** — reverse proxies `projects.dashboard` to the Go server's loopback port. Provides local HTTPS. Already part of the developer's toolchain.

**launchd** — keeps the Go binary running. Auto-starts on login, restarts on crash.

### Data Flow

1. Go poller scans local filesystem for git repos and runs git commands to extract branch, dirty state, last commit
2. Go poller calls `gh` CLI for GitHub metadata: repos, PRs, Actions, releases, branch protection, visibility, file presence checks
3. Both datasets merge into a unified repo list
4. Changes are persisted to a JSON cache file
5. SSE pushes updates to any connected browser
6. Native macOS notifications fire for configured events (independent of browser)

### Diagram

```
                    launchd (auto-start, restart on crash)
                        |
                    Go Server (127.0.0.1:PORT)
                    /       |        \
            Local Poller  GitHub Poller  HTTP Server
            (every 1m)   (every 5m)     /     |     \
              |               |        REST   SSE   Static Files
            git CLI         gh CLI      |      |        |
              |               |         └──────┴────────┘
              └───────┬───────┘                 |
                 Merge + Cache           Svelte Frontend
              ~/.config/catscan/         (projects.dashboard)
                      |
              terminal-notifier
              (native macOS notifications)
```

## Data Model

Each repository is represented as a single unified object populated from two sources: local git and GitHub API.

### Identity

| Field | Description |
|-------|-------------|
| Name | Repository name |
| Full name | Owner/repo format |
| Visibility | Public or Private |

### Clone State

| Field | Description |
|-------|-------------|
| Cloned | Whether the repo exists locally |
| Local path | Filesystem path if cloned |

### Local Git (cloned repos only)

| Field | Description |
|-------|-------------|
| Branch | Current checked-out branch |
| Dirty | Uncommitted changes present |
| Local last commit | Date of most recent local commit |

### GitHub Metadata

| Field | Description |
|-------|-------------|
| Description | Repo description |
| Homepage | Homepage URL if set |
| Language | Primary language |
| Topics | Tags/topics list |
| Has GitHub Pages | Pages enabled |
| Has README | README file present |
| Has license | License file present |
| Has CLAUDE.md | AI context file present |
| Has AGENTS.md | AI agents file present |
| Has .project.json | Portfolio metadata file present |
| Branch protected | Main/master branch protection enabled |

### Activity

| Field | Description |
|-------|-------------|
| GitHub last push | Most recent push date |
| Open PRs | Count of open pull requests |
| Actions status | Passing, failing, or none |
| Latest release | Tag and date of most recent release |
| New release | Whether a release occurred since last seen |

### Lifecycle Classification

Derived automatically from activity signals.

| Status | Criteria |
|--------|----------|
| Ongoing | Commits within threshold OR open PRs OR active CI |
| Maintenance | No recent commits but CI is passing — stable and maintained |
| Stale | No commits beyond stale threshold, no CI activity |
| Abandoned | No commits beyond abandoned threshold, no CI |

Thresholds are configurable (default: stale at 30 days, abandoned at 90 days).

## Storage

Two JSON files in `~/.config/catscan/`:

**cache.json** — full repo data, rebuilt on every poll cycle. On startup, the dashboard loads from cache immediately while the first poll runs in the background. This ensures the UI is never empty on boot.

**state.json** — persistent user state such as last-seen release versions for notification tracking. Survives cache rebuilds.

JSON is the right choice for this scale. The dataset is approximately 50 repos — a few kilobytes. The access pattern is "load everything, serve it, update on poll." No relational queries, no concurrent writes.

## API

### REST Endpoints

| Method | Path | Purpose |
|--------|------|---------|
| GET | /api/repos | Full repo list with optional filter/sort query params |
| GET | /api/repos/:name | Single repo detail |
| POST | /api/repos/:name/clone | Trigger clone to scan path |
| GET | /api/config | Current configuration |
| PUT | /api/config | Update configuration |
| GET | /api/health | Server status and last poll times |
| GET | /api/events | SSE stream |

### SSE Events

| Event | Description |
|-------|-------------|
| repos_updated | Full repo list after local poll |
| github_updated | Full repo list after GitHub poll |
| clone_progress | Clone started, completed, or failed |
| actions_changed | A repo's CI status changed |
| new_release | A repo published a new release |
| pr_opened | New pull request opened on a repo |
| error | gh auth failure, scan path missing, etc. |

## Notifications

CatScan sends native macOS notifications via terminal-notifier, independent of whether the browser is open. Clicking a notification opens the dashboard in the default browser, deep-linked to the relevant repo.

Notification types are individually configurable. Defaults:

| Event | Default |
|-------|---------|
| CI status changed | On |
| New release | On |
| PR opened | Off |
| Clone completed/failed | On |
| Error (gh auth, etc.) | On |

If terminal-notifier is not installed, falls back to osascript (notifications work but are not clickable).

## Frontend

### Layout

Single-page dashboard with four zones:

**Header bar** — logo, filter dropdowns (lifecycle, visibility, language, clone status), sort controls, config gear icon.

**Summary cards** — total repos, cloned count, public count, stale count. Clickable as quick filters.

**Repo table** — the primary view. Each row shows: checkbox (for multi-select), repo name (links to GitHub), visibility badge, branch and dirty indicator (if cloned), CI status dot, completeness warning icons, relative time since last update, lifecycle badge. Rows expand inline to show full detail: description, topics, homepage, latest release, open PRs, complete checklist.

**Action bar** — appears when uncloned repos are selected. "Clone selected (N)" button triggers batch clone.

### Design

Dark mode by default. Responsive layout — table columns collapse gracefully for narrow/half-screen windows. Consistent with the developer's existing aesthetic across projects.

## Configuration

All configuration is editable via the web UI settings panel. The JSON file is the source of truth; the UI reads and writes it.

**Config file:** `~/.config/catscan/config.json`

| Field | Description | Default |
|-------|-------------|---------|
| scanPath | Local directory to scan for cloned repos | ~/REPOS/alexcatdad |
| githubOwner | GitHub username | alexcatdad |
| port | Loopback port for the Go server | 7700 |
| localIntervalSeconds | Local git polling frequency | 60 |
| githubIntervalSeconds | GitHub polling frequency | 300 |
| staleDays | Days until a repo is considered stale | 30 |
| abandonedDays | Days until a repo is considered abandoned | 90 |
| notifications | Per-event-type toggle map | See Notifications section |

## Installation

Clone the repo and run make install. This builds the Go binary and Svelte frontend, installs the binary, and loads a launchd plist for background operation.

To uninstall: make uninstall removes the plist and binary.

Add `projects.dashboard` to paw-proxy configuration to access the dashboard via local HTTPS.

### First Run Behavior

If no config file exists, CatScan creates one with sensible defaults. If `gh` is not installed or not authenticated, the dashboard loads with a clear error banner explaining what's wrong. The app never crashes due to missing `gh`.

## CI Pipeline

### Linting

**Go** — golangci-lint with strict configuration: errcheck (no unchecked errors), govet, staticcheck, gosimple, unused (no dead code), gocritic.

**Svelte** — Biome with strict rules: recommended ruleset as baseline, a11y rules enforced (not warned), suspicious + complexity + correctness all set to error, no explicit any types.

### Testing

**Go** — unit tests with race detection enabled.

**E2E** — Playwright test suite against a running CatScan instance. The binary starts with a test flag that points to fixture repos and mock gh responses, so tests don't need real GitHub auth.

E2E coverage: repo list rendering, filtering and sorting, clone action, SSE live updates, config panel read/write, gh failure error banner.

### Accessibility

Static analysis via Biome's enforced a11y lint rules catches issues at dev time. Runtime testing via axe-core integrated into the Playwright suite — every e2e test runs an axe scan, CI fails on any violation.

### CI Jobs

All run on push and pull request:

1. Lint Go
2. Lint frontend
3. Go unit tests
4. Build (Svelte + Go)
5. E2E with Playwright and axe-core

## Project Structure

```
catscan/
├── cmd/catscan/           Entry point, config load, start server and pollers
├── internal/
│   ├── config/            Load and save config.json
│   ├── scanner/
│   │   ├── local          Git status, branch, dirty, clone detection
│   │   └── github         gh CLI calls for all GitHub data
│   ├── model/             Repo struct, lifecycle classification
│   ├── cache/             Read and write cache.json and state.json
│   ├── poller/            Two polling loops, merge results, broadcast changes
│   └── server/            HTTP server, routes, SSE, static file serving
├── frontend/              Svelte app with its own package.json
├── dist/                  Built Svelte output, served by Go
├── docs/plans/            Design documents
├── Makefile
├── go.mod
└── .github/workflows/     CI pipeline
```

## Feature List

| # | Feature |
|---|---------|
| F1 | List all GitHub repos for authenticated user |
| F2 | Detect which repos are cloned locally |
| F3 | One-click clone for missing repos |
| F4 | Clone status indicator |
| F5 | Last update date |
| F6 | Local git state (branch, dirty) for cloned repos |
| F7 | Open PRs per repo |
| F8 | GitHub Actions status per repo |
| F9 | Release notifications |
| F10 | Latest release version and date |
| F11 | Public/Private visibility |
| F12 | Branch protection status |
| F13 | Repo metadata completeness (description, README, license, topics, Pages, homepage) |
| F14 | .project.json presence check |
| F15 | CLAUDE.md and AGENTS.md presence check |
| F16 | Completeness checklist with at-a-glance scoring |
| F17 | Primary language per repo |
| F18 | Lifecycle status (ongoing, stale, maintenance, abandoned) |
| F19 | Web dashboard UI |
| F20 | Graceful gh failure handling |
| F21 | Localhost-only binding (127.0.0.1) |
| F22 | E2E tests in CI |
| F23 | Strict linting (golangci-lint + Biome) |
| F24 | Filtering and sorting |
| F25 | Multi-select and batch clone |
| F26 | Dark mode |
| F27 | Responsive layout |
| F28 | Native macOS notifications from Go daemon |
| F29 | Clickable notifications with deep-link to dashboard |
| F30 | Accessibility testing (axe-core + Biome a11y rules) |

## Principles

- Read-only dashboard — clone is the only write action
- gh CLI is the primary data source
- Graceful degradation when gh is unavailable
- .project.json is the portfolio metadata format
- No authentication — local-only, loopback-bound
- Minimal dependencies — Go stdlib where possible
