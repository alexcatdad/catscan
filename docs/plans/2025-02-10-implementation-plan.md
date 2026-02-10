# CatScan Implementation Plan

**Date:** 2025-02-10
**Design Reference:** `docs/plans/2025-02-10-catscan-design.md`
**Status:** Not started

This plan is structured in phases with explicit dependencies. Each phase must be fully complete before moving to the next unless noted otherwise. Each task has a checkbox for tracking progress.

---

## Version Pinning & Tooling Requirements

All dependency versions must be pinned explicitly. Never use `latest`, `^`, or `~` ranges. Lock files must be committed.

### Go

- Go version: **1.23** (or latest stable at time of implementation — check go.dev/dl, pin in go.mod)
- golangci-lint: pin version in CI workflow (do not use `latest` tag)
- Module path: `github.com/alexcatdad/catscan`
- Go module must specify minimum Go version in `go.mod`

### Frontend

- Bun: latest stable (used only as package manager and build tool, not runtime)
- Svelte: **5.x** (runes syntax: `$state`, `$derived`, `$effect`, `$props()`)
- SvelteKit: **not used** — this is a static Svelte SPA, not a SvelteKit app
- Vite: latest stable, pinned in package.json
- Tailwind CSS: **v4** (CSS-first configuration, `@theme`, `@import`)
- Biome: latest stable, pinned in package.json
- Playwright: latest stable, pinned in package.json
- axe-core/playwright: latest stable, pinned in package.json
- TypeScript: strict mode enabled in tsconfig.json

### Important

- Every dependency in `go.mod` and `package.json` must have an exact pinned version
- `bun.lock` and `go.sum` must be committed to the repository
- When looking up versions, verify the version actually exists before pinning it — do not guess version numbers

---

## Linting & Code Quality Rules

### Go — golangci-lint

Create `.golangci.yml` in the project root with the following linters enabled at minimum:

- **errcheck** — all errors must be checked, no ignored return values
- **govet** — all standard vet checks enabled
- **staticcheck** — full suite of static analysis checks
- **gosimple** — flag code that can be simplified
- **unused** — no dead code (unused functions, variables, types, constants)
- **gocritic** — opinionated style and correctness checks
- **gofmt** — all code must be formatted with gofmt
- **goimports** — imports must be organized (stdlib, then third-party, then local)
- **misspell** — catch common spelling mistakes in comments and strings
- **ineffassign** — no ineffectual assignments
- **unconvert** — no unnecessary type conversions

Additional rules:
- All exported types, functions, and methods must have doc comments
- Error wrapping must use `fmt.Errorf("context: %w", err)` format
- No use of `panic` outside of initialization code
- No use of `init()` functions
- Prefer stdlib over third-party packages wherever possible

### Frontend — Biome

Create `biome.json` in the `frontend/` directory with the following configuration:

- Extend from the **recommended** ruleset
- **Linter rules** — all set to error (not warn):
  - `a11y` — all accessibility rules enforced, not just warned. This includes: use alt text on images, valid ARIA roles, no ARIA on unsupported elements, use semantic HTML elements, form labels required, heading hierarchy enforced
  - `suspicious` — all rules enforced (catch common mistakes, suspicious constructs)
  - `complexity` — all rules enforced (flag unnecessarily complex code)
  - `correctness` — all rules enforced (catch definite bugs)
  - `style` — all rules enforced (consistent code style)
  - `noExplicitAny` — no use of the `any` type anywhere in TypeScript
- **Formatter** — enabled, using Biome's built-in formatter (not Prettier)
  - Indent style: tabs
  - Line width: 100
  - Quote style: double quotes
- **Organize imports** — enabled

### TypeScript

- `strict: true` in tsconfig.json — this enables all strict type checking flags
- `noUncheckedIndexedAccess: true` — array/object index access returns `T | undefined`
- `noImplicitReturns: true` — all code paths must return a value
- `noFallthroughCasesInSwitch: true` — switch cases must break or return
- `exactOptionalPropertyTypes: true` — distinguish between `undefined` and missing

---

## Phase 0 — Project Scaffold

**Dependencies:** None
**Goal:** Repository structure, tooling, and CI pipeline established. Everything builds and lints with zero errors on an empty project.

- [ ] Create the Go module. Initialize `go.mod` with the module path `github.com/alexcatdad/catscan` and the pinned Go version. Create `cmd/catscan/main.go` with a minimal main function that prints a startup message and exits. Verify it compiles with `go build`.

- [ ] Create the `internal/` package structure. Create empty Go files with package declarations (no logic yet) in each of these directories: `internal/config`, `internal/scanner`, `internal/model`, `internal/cache`, `internal/poller`, `internal/server`. Each file should have only the package declaration and a doc comment explaining the package's purpose as described in the design document.

- [ ] Initialize the Svelte frontend. Create the `frontend/` directory. Initialize with `bun init`. Install Svelte 5, Vite, the Svelte Vite plugin, Tailwind CSS v4, and TypeScript as dev dependencies. All versions pinned. Create a minimal `index.html`, `src/main.ts`, and `src/App.svelte` that renders a "CatScan" heading. Verify it builds with `bun run build` and outputs to `../dist/`.

- [ ] Configure Tailwind CSS v4. Set up the CSS-first configuration approach (no JavaScript config file). Create the main CSS entry point that imports Tailwind. Define a dark-mode-first theme with CSS custom properties for colors, spacing, and typography. The design aesthetic should be clean and minimal.

- [ ] Configure TypeScript. Create `tsconfig.json` in `frontend/` with all the strict rules listed in the Linting & Code Quality Rules section above.

- [ ] Configure Biome. Create `biome.json` in `frontend/` with all the rules listed in the Linting & Code Quality Rules section above. Run `biome check` on the existing files and fix any violations.

- [ ] Configure golangci-lint. Create `.golangci.yml` in the project root with all the linters and rules listed in the Linting & Code Quality Rules section above. Run `golangci-lint run` on the existing Go files and fix any violations.

- [ ] Create the Makefile. Implement the following targets as described in the design document: `build` (builds both Go and Svelte), `dev` (runs Go with the frontend dev server), `test` (runs Go tests and svelte-check), `lint` (runs golangci-lint and biome check), `clean` (removes build artifacts). The `build` target must build the Svelte frontend first, then build the Go binary. The Go binary should be output to `./bin/catscan`.

- [ ] Create the CI pipeline. Create `.github/workflows/ci.yml` that triggers on push and pull request. Define these jobs: lint-go (runs golangci-lint), lint-frontend (runs biome check and svelte-check), test-go (runs go test with race detection), build (builds Svelte then Go). Pin all action versions to specific SHA or tag — never use `@latest` or `@v4` style tags without the full version. Use Bun for frontend steps (not npm or yarn).

- [ ] Create `.gitignore`. Ignore: `bin/`, `dist/`, `node_modules/`, `.DS_Store`, `*.exe`, `tmp/`. Do not ignore lock files.

- [ ] Verify the full pipeline locally. Run `make lint`, `make build`, and `make test` in sequence. All must pass with zero warnings and zero errors. Commit and push. Verify CI passes on GitHub.

---

## Phase 1 — Configuration & Data Model

**Dependencies:** Phase 0 complete
**Goal:** Config loading/saving works, data model is defined, cache layer reads and writes JSON files.

- [ ] Implement the Repo model. In `internal/model/repo.go`, define the Repo struct with all fields described in the Data Model section of the design document. Include JSON tags for serialization. Define the lifecycle status as a string type with constants for ongoing, stale, maintenance, and abandoned. Implement a method on Repo that computes lifecycle status from the activity fields and configurable thresholds.

- [ ] Implement the config package. In `internal/config/config.go`, define the Config struct matching the Configuration section of the design document. Include all fields: scanPath, githubOwner, port, polling intervals, lifecycle thresholds, and notification toggles. Implement functions to load config from `~/.config/catscan/config.json`, save config back to the file, and return a default config if the file doesn't exist. The config directory must be created automatically if it doesn't exist. Use `os.UserHomeDir()` to resolve the home directory — never hardcode paths.

- [ ] Implement the cache package. In `internal/cache/cache.go`, implement functions to read and write `cache.json` and `state.json` in the config directory. `cache.json` stores the full list of Repo objects. `state.json` stores persistent user state: a map of repo names to last-seen release tags. Both files must be written atomically (write to a temp file, then rename) to prevent corruption if the process is interrupted during a write. Handle the case where files don't exist yet gracefully — return empty data, don't error.

- [ ] Write unit tests for config. Test: loading from a valid file, loading when file doesn't exist (should return defaults), saving and re-loading round-trip, expanding tilde in scanPath.

- [ ] Write unit tests for cache. Test: writing and reading cache.json round-trip, writing and reading state.json round-trip, handling missing files, atomic write doesn't corrupt existing data.

- [ ] Write unit tests for the lifecycle classification method on Repo. Test each status: ongoing (recent commit), ongoing (open PRs, old commit), maintenance (old commit, CI passing), stale (no activity within threshold), abandoned (no activity beyond abandoned threshold). Test with custom thresholds.

- [ ] Run `make lint` and `make test`. All must pass. Commit.

---

## Phase 2 — Scanners

**Dependencies:** Phase 1 complete
**Goal:** Both local git scanning and GitHub data fetching work independently and return populated Repo data.

### Local Scanner

- [ ] Implement local repo discovery. In `internal/scanner/local.go`, implement a function that takes a scan path and returns a list of directories that contain a `.git` folder. Expand tilde in the scan path. Only scan one level deep (direct children of the scan path). Skip hidden directories (those starting with a dot). Return the list sorted alphabetically.

- [ ] Implement local git state extraction. In the same file, implement a function that takes a repo path and returns: current branch name (from `git rev-parse --abbrev-ref HEAD`), whether the working tree is dirty (from `git status --porcelain`), and the last commit date (from `git log -1 --format=%aI`). Use `os/exec` to run git commands. Always use the absolute path `/usr/bin/git` since the binary may run in a context without shell PATH. Pass arguments as a string slice, never as a single shell string — this prevents injection. Handle errors gracefully: if any git command fails for a repo, log the error and skip that repo rather than crashing.

- [ ] Implement clone detection. Implement a function that takes a list of GitHub repo names and a scan path, and returns a map of repo name to local path for repos that exist locally. A repo is considered cloned if a directory with the same name exists in the scan path and contains a `.git` folder.

- [ ] Implement the clone action. Implement a function that clones a GitHub repo to the scan path. Use `/usr/bin/git clone` with the HTTPS URL format `https://github.com/{owner}/{name}.git`. The function should return progress/status information suitable for streaming to the frontend via SSE. Handle errors: repo already exists, network failure, invalid repo name.

### GitHub Scanner

- [ ] Implement GitHub repo listing. In `internal/scanner/github.go`, implement a function that lists all repos for the configured GitHub owner using the `gh` CLI. Use `gh repo list {owner} --json name,description,visibility,homepageUrl,primaryLanguage,repositoryTopics,hasPages,defaultBranchRef,latestRelease --limit 200`. Parse the JSON output into structured data. If `gh` is not found at common paths (`/opt/homebrew/bin/gh`, `/usr/local/bin/gh`, `/usr/bin/gh`), return a descriptive error — never crash. If `gh` is not authenticated, detect this from the error output and return a specific error type so the server can communicate it to the frontend.

- [ ] Implement PR fetching. Implement a function that fetches open PR count per repo. Use `gh pr list --repo {owner}/{name} --state open --json number --limit 100`. Return the count. Handle errors per-repo — if one repo fails, log and continue with others. Rate limit awareness: if the GitHub API returns rate limit errors, back off and retry after the reset window.

- [ ] Implement Actions status fetching. Implement a function that fetches the latest workflow run status per repo. Use `gh run list --repo {owner}/{name} --limit 1 --json status,conclusion`. Map the conclusion to passing, failing, or none. If a repo has no workflows, return none.

- [ ] Implement release fetching. Implement a function that fetches the latest release per repo. The data should already be available from the repo listing call's `latestRelease` field. Extract the tag name and published date. Compare against `state.json` to determine if this is a new release since last seen. Update state.json when a release is acknowledged.

- [ ] Implement branch protection check. Implement a function that checks whether the default branch is protected. Use `gh api repos/{owner}/{name}/branches/{defaultBranch}/protection`. A 200 response means protected, a 404 means not protected. Handle 403 (insufficient permissions) gracefully.

- [ ] Implement file presence checks. Implement a function that checks for the presence of specific files in a repo: README (any README* file), LICENSE (any LICENSE* file), CLAUDE.md, AGENTS.md, .project.json. Use `gh api repos/{owner}/{name}/contents/{path}`. A 200 means present, 404 means absent. Batch these checks per repo to minimize API calls — consider using the repository tree endpoint instead if more efficient.

- [ ] Implement the merge function. Create a function that takes the local scan results and GitHub results and merges them into a unified list of Repo objects. Local fields populate clone state and git state. GitHub fields populate everything else. Repos that exist on GitHub but not locally get `cloned: false`. Repos that exist locally but not on GitHub should still appear (they might be private repos the `gh` call missed, or local-only repos). Compute lifecycle status during merge.

### Tests

- [ ] Write unit tests for local scanner. Create test fixture directories with `.git` folders. Test: discovery finds repos, skips non-git directories, skips hidden directories. Test git state extraction with a real temporary git repo (init, commit, make dirty).

- [ ] Write unit tests for GitHub scanner. Mock the `gh` CLI output by creating a test helper that intercepts exec calls or uses a fake gh script. Test: parsing repo list JSON, handling gh not found, handling auth failure, parsing PR counts, parsing Actions status, parsing releases, file presence checks.

- [ ] Write unit tests for the merge function. Test: GitHub-only repo appears as not cloned, local-only repo appears with minimal data, fully matched repo has all fields populated, lifecycle is computed correctly.

- [ ] Run `make lint` and `make test`. All must pass. Commit.

---

## Phase 3 — Poller & SSE

**Dependencies:** Phase 2 complete
**Goal:** Background polling runs on two intervals, merges data, caches to disk, and broadcasts changes via SSE to connected clients.

- [ ] Implement the SSE broadcaster. In `internal/server/sse.go`, implement an SSE hub that manages connected clients. It must support: registering new client connections, removing disconnected clients, broadcasting an event (with event type and JSON data) to all connected clients simultaneously. Use Go channels for thread-safe communication. The hub must run in its own goroutine. When a client disconnects (detected via context cancellation or write error), remove it cleanly without blocking the broadcast to other clients.

- [ ] Implement the local poller. In `internal/poller/poller.go`, implement a goroutine that runs the local scanner on a configurable interval (default 60 seconds). After each scan, merge results with the most recent GitHub data from cache, update the cache file, and broadcast a `repos_updated` event via the SSE hub. The poller must be cancellable via context for clean shutdown. On the very first run, it should execute immediately (don't wait for the first interval to elapse).

- [ ] Implement the GitHub poller. In the same file, implement a second goroutine that runs the GitHub scanner on a configurable interval (default 300 seconds). After each scan, merge results with the most recent local data from cache, update the cache file, and broadcast a `github_updated` event. Same cancellation and immediate-first-run behavior as the local poller.

- [ ] Implement change detection for granular events. Before broadcasting, compare the new repo list against the previous one. Detect and emit specific events: `actions_changed` when a repo's CI status transitions (pass to fail, fail to pass), `new_release` when a repo has a release not seen in state.json, `pr_opened` when a repo's open PR count increases. These granular events are emitted in addition to the full `repos_updated` or `github_updated` event.

- [ ] Implement notification dispatch. When granular change events are detected (actions_changed, new_release, pr_opened), check the notification config. If the event type is enabled, dispatch a macOS notification. Look for `terminal-notifier` at `/opt/homebrew/bin/terminal-notifier` and `/usr/local/bin/terminal-notifier`. If found, use it with the `-open` flag pointing to `https://projects.dashboard/repo/{name}` for deep linking. If not found, fall back to `osascript` with `display notification`. Notification text should be concise and informative, for example: "meowtern — CI failed on main" or "paw v2.3.0 released".

- [ ] Implement startup behavior. On server start: load cache.json and serve it immediately to any connecting clients (the dashboard should never be empty on boot). Then kick off both pollers. The first poll cycle refreshes the cache with live data.

- [ ] Write unit tests for the SSE hub. Test: client registration, client removal on disconnect, broadcast reaches all clients, broadcast doesn't block if a client is slow (use buffered channels or drop strategy).

- [ ] Write unit tests for change detection. Test: no change emits no granular events, CI status change is detected, new release is detected, PR count increase is detected, PR count decrease does not emit pr_opened.

- [ ] Run `make lint` and `make test`. All must pass. Commit.

---

## Phase 4 — HTTP Server & REST API

**Dependencies:** Phase 3 complete
**Goal:** Go server serves the static frontend, exposes REST endpoints, and handles the SSE connection. Binds to localhost only.

- [ ] Implement the HTTP server. In `internal/server/server.go`, create the HTTP server that binds to `127.0.0.1:{port}` — never `0.0.0.0`. Use Go's `net/http` standard library. The server must serve static files from the `dist/` directory for the frontend SPA. For any path that doesn't match an API route or a static file, serve `index.html` (SPA fallback routing).

- [ ] Implement the repos list endpoint. `GET /api/repos` returns the full cached repo list as JSON. Support optional query parameters for filtering: `lifecycle` (comma-separated values), `visibility` (public or private), `cloned` (true or false), `language` (string match). Support `sort` (name, lastUpdate, lifecycle) and `order` (asc, desc) query parameters. Even though the frontend will do its own client-side filtering, the API should support it for potential future use.

- [ ] Implement the single repo endpoint. `GET /api/repos/:name` returns a single repo by name. Return 404 with a JSON error body if the repo doesn't exist in the cache.

- [ ] Implement the clone endpoint. `POST /api/repos/:name/clone` triggers a clone of the named repo to the configured scan path. The clone must run asynchronously — return 202 Accepted immediately, then broadcast `clone_progress` SSE events as the clone proceeds (started, completed, failed). Validate that the repo exists on GitHub and is not already cloned before starting. Return 409 Conflict if already cloned. Return 404 if repo not found.

- [ ] Implement the config endpoints. `GET /api/config` returns the current config as JSON. `PUT /api/config` accepts a JSON body, validates it, saves to disk, and applies changes. If polling intervals changed, restart the pollers with new intervals. If the scan path changed, trigger an immediate re-scan. Return 400 with a descriptive error for invalid config values (negative intervals, empty scan path, etc).

- [ ] Implement the health endpoint. `GET /api/health` returns: server uptime, last local poll time, last GitHub poll time, total repo count, whether gh CLI is available, whether gh is authenticated. This endpoint is useful for debugging and monitoring.

- [ ] Implement the SSE endpoint. `GET /api/events` establishes a Server-Sent Events connection. Set appropriate headers: `Content-Type: text/event-stream`, `Cache-Control: no-cache`, `Connection: keep-alive`. Register the client with the SSE hub. On connect, immediately send the current repo list as a `repos_updated` event so the client doesn't have to wait for the next poll. Keep the connection alive with a comment heartbeat every 30 seconds to prevent proxy/load-balancer timeouts.

- [ ] Implement CORS and security headers. Since paw-proxy terminates TLS and proxies to localhost, add `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`, and `Referrer-Policy: no-referrer` headers to all responses. CORS is not needed since the frontend is served from the same origin.

- [ ] Implement graceful shutdown. On SIGINT or SIGTERM: stop the pollers (cancel their contexts), close all SSE connections, wait for in-flight requests to complete (with a timeout), then exit. This ensures clean behavior when launchd stops the service.

- [ ] Write unit tests for API endpoints. Use `httptest` to test each endpoint. Test: repos list returns correct JSON shape, filtering works, sorting works, single repo returns correct data, single repo 404 for unknown name, clone returns 202, clone returns 409 for already-cloned, config round-trip, health endpoint shape, SSE connection receives events.

- [ ] Run `make lint` and `make test`. All must pass. Commit.

---

## Phase 5 — Frontend Core

**Dependencies:** Phase 4 complete (API must be working and testable)
**Goal:** Svelte SPA connects to the API, renders the repo list, and updates live via SSE.

- [ ] Set up the Svelte app structure. Organize the frontend source into: `src/lib/` for shared types, API client, and stores; `src/components/` for UI components; `src/App.svelte` as the root. Define TypeScript types matching the Repo model from the Go API — keep these in a single types file. Do not duplicate type definitions across files.

- [ ] Implement the API client. Create a typed API client in `src/lib/api.ts` that wraps fetch calls to the Go backend. Functions for: `getRepos()`, `getRepo(name)`, `cloneRepo(name)`, `getConfig()`, `updateConfig(config)`, `getHealth()`. All functions should return typed responses and throw on non-2xx status codes. Base URL should be relative (same origin).

- [ ] Implement the SSE connection. Create an SSE client in `src/lib/sse.ts` that connects to `/api/events`. Parse incoming events by type and update the Svelte store accordingly. Implement automatic reconnection with exponential backoff if the connection drops. Use Svelte 5 runes for reactive state — the repo list should be a `$state` that updates when SSE events arrive.

- [ ] Implement the main store. Create a reactive store using Svelte 5 runes that holds: the repo list, current filter state, current sort state, loading state, error state (for gh failures), and SSE connection status. The store should expose derived values: filtered and sorted repo list, summary counts (total, cloned, public, stale, etc.), completeness statistics.

- [ ] Implement the header bar component. Create a header with: the CatScan logo/name, filter dropdowns (lifecycle, visibility, language, clone status), sort controls (field selector + asc/desc toggle), and a config gear icon. Filters should update the store state, which reactively updates the displayed list. Multiple filters combine with AND logic.

- [ ] Implement the summary cards component. Create a row of clickable stat cards: total repos, cloned count, public count, and a count per lifecycle status. Clicking a card applies it as a filter (clicking "stale" filters the table to stale repos only). Active filter card should be visually highlighted. Clicking an active card clears that filter.

- [ ] Implement the repo table component. Create the main table view. Each row displays: checkbox (for multi-select, only enabled on uncloned repos), repo name (clickable, opens GitHub in new tab), visibility badge (public/private), branch name and dirty indicator (if cloned, omitted if not), CI status dot (green/red/gray), completeness warning icons (small icons for missing description, README, license — only shown when missing), relative time since last update, lifecycle badge (color-coded). The table must be keyboard accessible — arrow keys to navigate rows, Enter to expand, Space to toggle checkbox.

- [ ] Implement the row expansion panel. Clicking a repo row expands an inline detail panel below it showing: full description, topics as tag pills, homepage URL (linked), latest release tag and date, open PR count, full completeness checklist (description, README, license, topics, Pages, homepage, .project.json, CLAUDE.md, AGENTS.md — each with a check or X icon), branch protection status. Only one row should be expanded at a time (expanding another collapses the previous).

- [ ] Implement the action bar component. When one or more uncloned repos are checked, display a sticky bottom bar with: count of selected repos, a "Clone selected" button. Clicking clone triggers the API call and updates the UI with progress via SSE clone_progress events. Show per-repo status in the table (cloning spinner, success check, failure X). Disable the button while cloning is in progress.

- [ ] Implement the error banner. When the Go backend reports a gh auth failure or gh not found error (via the SSE error event or the health endpoint), display a prominent but non-blocking banner at the top of the page explaining the issue and how to fix it. The banner should be dismissable but reappear if the error persists on the next poll.

- [ ] Implement dark mode. Style the entire app in dark mode by default. Use Tailwind CSS v4 custom properties for all colors so the theme is centralized. Follow a visual style that is clean, minimal, and information-dense — similar to terminal dashboards. Ensure sufficient contrast ratios for accessibility (WCAG AA minimum, AAA preferred for text).

- [ ] Implement responsive layout. The table must work at narrow widths (half-screen window). At narrow breakpoints: hide lower-priority columns (language, branch, homepage) and make them visible only in the expanded row detail. Summary cards should wrap to two rows if needed. The action bar should remain fixed at the bottom. Test at 600px, 900px, and 1200px widths minimum.

- [ ] Run `bun run build`, `biome check`, and `svelte-check`. All must pass with zero errors and zero warnings. Commit.

---

## Phase 6 — Settings Panel

**Dependencies:** Phase 5 complete
**Goal:** Users can view and edit all CatScan settings from the web UI.

- [ ] Implement the settings panel. Accessible via the gear icon in the header. Should open as a slide-over panel or modal — not a separate page. Display all config fields from the design document in a form: scan path (text input), GitHub owner (text input), port (number input), local poll interval in seconds (number input), GitHub poll interval in seconds (number input), stale days threshold (number input), abandoned days threshold (number input), and notification toggles (checkbox for each event type).

- [ ] Implement form validation. Validate inputs before saving: scan path must not be empty, port must be between 1024 and 65535, poll intervals must be positive integers with a minimum of 10 seconds for local and 60 seconds for GitHub (to prevent API abuse), threshold days must be positive integers, stale threshold must be less than abandoned threshold. Display inline validation errors next to the relevant fields.

- [ ] Implement save and apply. When the user saves, call `PUT /api/config`. On success, close the panel and show a brief success toast. On validation error from the API, display the error. The Go backend applies changes immediately (restarting pollers if intervals changed, re-scanning if scan path changed) — no restart required.

- [ ] Implement the notification permission prompt. If browser notification permission hasn't been granted, show an informational note in the notifications section explaining that native macOS notifications work regardless, but browser notifications require permission. This is informational only — the primary notification path is the Go daemon, not the browser.

- [ ] Run linting and type checking. All must pass. Commit.

---

## Phase 7 — E2E Tests

**Dependencies:** Phase 6 complete
**Goal:** Playwright e2e test suite covers all major user flows. Accessibility is verified on every test. CI runs the full suite.

- [ ] Set up the Playwright test infrastructure. Install Playwright and axe-core/playwright in the frontend dev dependencies (versions pinned). Create `playwright.config.ts` in the project root. Configure it to: start the CatScan binary before tests (using the `webServer` config), use a test-specific port to avoid conflicts, run tests in Chromium only (sufficient for a local dashboard), set a reasonable timeout (30 seconds per test).

- [ ] Create the test fixture system. The Go binary needs a `--test` flag or `CATSCAN_TEST=1` environment variable that activates test mode. In test mode: the scan path points to a fixture directory within the repo containing fake git repos (directories with `.git` folders, some with dirty state, some clean); GitHub data comes from a mock — either a fake `gh` script placed on PATH that returns canned JSON responses, or a fixture JSON file loaded directly. The fixture data should include a mix of: public and private repos, cloned and uncloned repos, repos with passing and failing CI, repos with and without releases, repos with varying completeness levels.

- [ ] Create the axe-core accessibility helper. Create a shared test utility that runs an axe accessibility scan on the current page state. Every e2e test must call this helper at least once during its flow. The helper should assert that `results.violations` is empty. If violations are found, the error message should include the violation details (rule ID, affected elements, impact level) for easy debugging.

- [ ] Write e2e test: dashboard loads and displays repos. Navigate to the dashboard. Verify the header, summary cards, and repo table render. Verify repos from the fixture data appear with correct names. Verify summary card counts match the fixture data. Run accessibility scan.

- [ ] Write e2e test: filtering works. Apply lifecycle filter to "stale" — verify only stale repos are shown. Apply visibility filter to "public" — verify only public repos are shown. Combine filters — verify AND logic. Clear filters — verify all repos return. Run accessibility scan on each filter state.

- [ ] Write e2e test: sorting works. Sort by name ascending — verify alphabetical order. Sort by last update descending — verify most recent first. Switch sort direction — verify order reverses. Run accessibility scan.

- [ ] Write e2e test: row expansion shows detail. Click a repo row. Verify the detail panel expands showing description, topics, completeness checklist. Click another row — verify the first collapses and the second expands. Run accessibility scan on the expanded state.

- [ ] Write e2e test: clone action. Select two uncloned repos via checkboxes. Verify the action bar appears with "Clone selected (2)". Click clone. Verify SSE events update the UI (cloning state, then completed). Verify the repos now show as cloned. Run accessibility scan.

- [ ] Write e2e test: SSE live updates. Connect to the dashboard. Trigger a data change in the fixture (modify a fixture file or mock response). Wait for the next poll. Verify the UI updates without manual refresh. Run accessibility scan.

- [ ] Write e2e test: config panel. Open the settings panel. Verify all config fields are populated with current values. Change a value. Save. Reopen the panel — verify the new value persisted. Test validation by entering an invalid value — verify inline error appears and save is blocked. Run accessibility scan on the settings panel.

- [ ] Write e2e test: gh failure error banner. Start CatScan with a broken gh mock (returns auth error). Navigate to the dashboard. Verify the error banner appears with a clear message about gh authentication. Verify the rest of the dashboard still loads (from cache or with empty state). Run accessibility scan.

- [ ] Add e2e job to CI. Update `.github/workflows/ci.yml` to add an e2e job that: builds the full CatScan binary with test fixtures, runs the Playwright suite, uploads test results and screenshots as artifacts on failure. This job depends on the build job passing first.

- [ ] Run the full CI pipeline locally. `make lint`, `make test`, `make e2e` — all must pass. Push and verify CI passes on GitHub.

---

## Phase 8 — Polish & Packaging

**Dependencies:** Phase 7 complete (all tests passing)
**Goal:** Production-ready build, launchd integration, documentation.

- [ ] Create the launchd plist. Create a `com.alexcatdad.catscan.plist` file in the repo. It should: run the CatScan binary, set `KeepAlive` to true (restart on crash), set `RunAtLoad` to true (start on login), redirect stdout and stderr to log files in `~/.config/catscan/logs/`, set the working directory to the user's home directory. Do not hardcode the binary path — the Makefile install target should template it.

- [ ] Implement `make install`. This target should: run `make build`, copy the binary to a sensible location (e.g., `~/.local/bin/catscan` or `/usr/local/bin/catscan`), copy the launchd plist to `~/Library/LaunchAgents/` with the correct binary path, load the plist with `launchctl load`. Print a clear success message with next steps (configuring paw-proxy).

- [ ] Implement `make uninstall`. This target should: unload the launchd plist with `launchctl unload`, remove the plist from `~/Library/LaunchAgents/`, remove the binary. Do not remove config or cache files — the user may want to reinstall later.

- [ ] Write the README. Cover: what CatScan is (one paragraph), screenshot of the dashboard, prerequisites (Go, Bun, git, gh — with version requirements), installation steps (clone, make install, configure paw-proxy), configuration (reference the settings panel), how to access the dashboard (projects.dashboard), how to stop/start/restart (launchctl commands), how to uninstall, project structure overview, how to contribute (make dev, make lint, make test, make e2e), license (MIT).

- [ ] Create the .project.json file. Following the established format used across all alexcatdad repos: `what`, `why`, `tags`, `featured`, `order`. Set appropriate values for CatScan.

- [ ] Final verification. Fresh clone of the repo on a clean state. Run `make install` from scratch. Verify the dashboard loads at localhost on the configured port. Verify data populates. Verify notifications work. Run `make lint`, `make test`, `make e2e` — all pass. Push and verify CI is green.

---

## Dependency Summary

```
Phase 0 (Scaffold)
  └── Phase 1 (Config & Model)
       └── Phase 2 (Scanners)
            └── Phase 3 (Poller & SSE)
                 └── Phase 4 (HTTP Server & API)
                      └── Phase 5 (Frontend Core)
                           └── Phase 6 (Settings Panel)
                                └── Phase 7 (E2E Tests)
                                     └── Phase 8 (Polish & Packaging)
```

All phases are strictly sequential. Each phase must fully pass linting and tests before proceeding to the next.
