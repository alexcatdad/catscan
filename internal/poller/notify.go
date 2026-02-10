// Package poller manages background polling for local and GitHub data.
//
// The notify subpackage handles macOS notifications.
package poller

import (
	"fmt"
	"os/exec"
	"strings"
	"sync"
)

// Notifier sends macOS notifications.
type Notifier struct {
	useTerminalNotifier bool
	terminalNotifierPath string
	once                 sync.Once
}

// NewNotifier creates a new Notifier.
func NewNotifier() *Notifier {
	n := &Notifier{}
	n.init()
	return n
}

// init checks for terminal-notifier availability.
func (n *Notifier) init() {
	n.once.Do(func() {
		// Check common paths for terminal-notifier
		paths := []string{
			"/opt/homebrew/bin/terminal-notifier",
			"/usr/local/bin/terminal-notifier",
		}

		for _, path := range paths {
			if _, err := exec.LookPath(path); err == nil {
				n.useTerminalNotifier = true
				n.terminalNotifierPath = path
				return
			}
		}
	})
}

// Notify sends a macOS notification.
func (n *Notifier) Notify(title, message, url string) error {
	if n.useTerminalNotifier {
		return n.notifyTerminalNotifier(title, message, url)
	}
	return n.notifyOSAScript(title, message)
}

// notifyTerminalNotifier sends a notification using terminal-notifier.
func (n *Notifier) notifyTerminalNotifier(title, message, url string) error {
	args := []string{
		"-title", title,
		"-message", message,
	}

	if url != "" {
		args = append(args, "-open", url)
	}

	cmd := exec.Command(n.terminalNotifierPath, args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("terminal-notifier: %w", err)
	}

	return nil
}

// notifyOSAScript sends a notification using osascript.
func (n *Notifier) notifyOSAScript(title, message string) error {
	// Escape quotes in title and message
	title = strings.ReplaceAll(title, `"`, `\"`)
	message = strings.ReplaceAll(message, `"`, `\"`)

	script := fmt.Sprintf(`display notification "%s" with title "%s"`, message, title)
	cmd := exec.Command("osascript", "-e", script)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("osascript: %w", err)
	}

	return nil
}

// SendNotification sends a notification for a repo event.
func SendNotification(eventType, repoName, message string) {
	notifier := NewNotifier()

	title := fmt.Sprintf("CatScan — %s", repoName)
	url := fmt.Sprintf("https://projects.dashboard/repo/%s", repoName)

	if err := notifier.Notify(title, message, url); err != nil {
		// Log but don't fail — notification failures are non-critical
		fmt.Printf("notification error: %v\n", err)
	}
}
