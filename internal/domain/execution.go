package domain

import "time"

// Execution はClaude Code実行結果を表す
type Execution struct {
	TaskID    string
	Success   bool
	Output    string
	Error     string
	SessionID string
	StartedAt time.Time
	EndedAt   time.Time
	Duration  time.Duration
}

// Summary は実行結果の概要を返す（Projectに保存する用）
func (e *Execution) Summary() string {
	if !e.Success {
		if len(e.Error) > 200 {
			return "Error: " + e.Error[:200] + "..."
		}
		return "Error: " + e.Error
	}

	if len(e.Output) > 500 {
		return e.Output[:500] + "..."
	}
	return e.Output
}

// NewStatus は実行結果に基づいて新しいStatusを返す
func (e *Execution) NewStatus() Status {
	if e.Success {
		return StatusInReview
	}
	// 失敗してもInReviewにして確認を促す
	return StatusInReview
}
