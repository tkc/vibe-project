//go:build !darwin

package notify

// Send is a no-op on non-darwin platforms
func Send(title, message string) error {
	return nil
}

// SendSuccess is a no-op on non-darwin platforms
func SendSuccess(taskTitle string, duration float64) error {
	return nil
}

// SendFailure is a no-op on non-darwin platforms
func SendFailure(taskTitle string, errMsg string) error {
	return nil
}
