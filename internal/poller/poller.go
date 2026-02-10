// Package poller manages background polling for local and GitHub data.
//
// Two independent goroutines poll local git state and GitHub metadata
// on configurable intervals. Results are merged, cached, and broadcast
// via SSE to connected clients.
package poller
