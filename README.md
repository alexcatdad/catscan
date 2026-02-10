# CatScan

> A local-only GitHub repository dashboard for tracking project health and lifecycle.

CatScan monitors your GitHub repositories from a local dashboard, tracking lifecycle status (ongoing, stale, abandoned), CI health, completeness checklists, and more. It runs as a background service on macOS with zero cloud dependencies.

![CatScan Dashboard](assets/screenshot.png)

## Features

- **Repository Lifecycle Tracking** — Automatically classifies repos as ongoing, maintenance, stale, or abandoned based on activity
- **Completeness Checklist** — Tracks presence of README, LICENSE, topics, CI/CD, and more
- **GitHub Integration** — Shows PRs, releases, CI status, and branch protection via `gh` CLI
- **Local Clone Management** — One-click cloning to your local filesystem
- **Real-time Updates** — SSE-based live updates without page refresh
- **macOS Notifications** — Native notifications for CI changes, new releases, and PRs
- **Dark Mode** — Full dark mode support with system preference detection

## Prerequisites

- **macOS** 12.0 or later (for launchd integration and notifications)
- **Go** 1.25 or later
- **Bun** 1.3 or later (for building the frontend)
- **git** — for local clone management
- **gh** — GitHub CLI for GitHub API access

## Installation

1. **Clone the repository**
   ```bash
   git clone https://github.com/alexcatdad/catscan.git
   cd catscan
   ```

2. **Install CatScan**
   ```bash
   make install
   ```

   This builds the binary, installs it to `~/.local/bin/catscan`, and sets up a launchd agent to start CatScan on login.

3. **Configure CatScan**
   - Open http://localhost:7700 in your browser
   - Click the settings icon (gear) in the top right
   - Set your scan path (where you want repos cloned) and GitHub owner
   - Adjust polling intervals and thresholds as desired

4. **Configure `gh` authentication** (if not already done)
   ```bash
   gh auth login
   ```

## Usage

### Accessing the Dashboard

Open your browser to http://localhost:7700 (or your configured port).

The dashboard shows:
- **Summary cards** — Quick counts of total, cloned, public, private repos by lifecycle
- **Filter & sort** — Filter by lifecycle, visibility, clone status; sort by name or last update
- **Repo table** — List of all repos with key metrics and one-click detail expansion
- **Detail panel** — Expanded view with description, topics, completeness, releases, and PRs

### Managing Services

```bash
# Start CatScan manually (if not using launchd)
catscan

# Stop the launchd service
launchctl unload ~/Library/LaunchAgents/com.alexcatdad.catscan.plist

# Restart the launchd service
launchctl unload ~/Library/LaunchAgents/com.alexcatdad.catscan.plist
launchctl load ~/Library/LaunchAgents/com.alexcatdad.catscan.plist

# View logs
tail -f ~/.config/catscan/logs/catscan.log
```

### Configuration

All settings are managed through the web UI:

- **Scan Path** — Local directory containing your repositories
- **GitHub Owner** — GitHub username or organization to scan
- **Port** — HTTP port for the dashboard (default: 7700)
- **Poll Intervals** — How often to check local state (default: 30s) and GitHub API (default: 5min)
- **Lifecycle Thresholds** — Days before marking a repo stale (default: 90) or abandoned (default: 365)
- **Notifications** — Toggle notifications for CI changes, new releases, and PRs

Config is stored at `~/.config/catscan/config.json`.

## Development

### Running in Development Mode

```bash
# Run both Go backend and Svelte dev server
make dev

# Or run individually
make dev-backend  # Go server
make dev-frontend # Svelte dev server on http://localhost:5173
```

### Running Tests

```bash
# Run all tests (Go unit tests + Svelte type checking)
make test

# Run E2E tests (requires Playwright)
make e2e
```

### Linting

```bash
# Run all linters
make lint

# This runs:
# - golangci-lint on Go code
# - Biome check on frontend code
```

### Building

```bash
# Build frontend and Go binary
make build

# Output: ./bin/catscan and ./dist/
```

## Uninstallation

```bash
make uninstall
```

This removes the launchd agent and binary. Config and cache files in `~/.config/catscan/` are preserved for potential reinstallation.

## Project Structure

```
catscan/
├── cmd/catscan/         # Go binary entry point
├── internal/
│   ├── cache/          # In-memory repo cache
│   ├── config/         # Configuration loading/saving
│   ├── model/          # Data models (Repo, Completeness, etc.)
│   ├── poller/         # Background polling for local and GitHub data
│   ├── scanner/        # Local and GitHub repo scanning
│   ├── server/         # HTTP server and API endpoints
│   └── sse/            # Server-Sent Events for live updates
├── frontend/
│   ├── src/
│   │   ├── components/ # Svelte components
│   │   ├── lib/        # Utilities, API client, types, store
│   │   └── routes/     # SvelteKit routes
│   ├── e2e/            # Playwright E2E tests
│   └── static/         # Static assets
├── test/               # Test fixtures
└── Makefile           # Build and install targets
```

## Contributing

Contributions are welcome! Please:
1. Fork the repository
2. Create a feature branch (`git checkout -b feat/amazing-feature`)
3. Commit your changes (`git commit -m 'feat: add amazing feature'`)
4. Push to the branch (`git push origin feat/amazing-feature`)
5. Open a pull request

Ensure `make lint` and `make test` pass before submitting.

## License

MIT

## Acknowledgments

- [Svelte 5](https://svelte.dev/) — Reactive UI framework
- [Tailwind CSS](https://tailwindcss.com/) — Styling
- [Go](https://go.dev/) — Backend
- [gh](https://cli.github.com/) — GitHub CLI
- [Playwright](https://playwright.dev/) — E2E testing
