//go:build darwin

package notify

import (
	"fmt"
	"os/exec"
)

// Send sends a macOS notification using osascript
func Send(title, message string) error {
	script := fmt.Sprintf(`display notification "%s" with title "%s" sound name "Glass"`, message, title)
	cmd := exec.Command("osascript", "-e", script)
	return cmd.Run()
}

// SendSuccess sends a success notification
func SendSuccess(taskTitle string, duration float64) error {
	title := "✅ vibe: Task Completed"
	message := fmt.Sprintf("%s (%.1fs)", taskTitle, duration)
	return Send(title, message)
}

// SendFailure sends a failure notification
func SendFailure(taskTitle string, errMsg string) error {
	title := "❌ vibe: Task Failed"
	message := fmt.Sprintf("%s: %s", taskTitle, truncate(errMsg, 50))
	return Send(title, message)
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
