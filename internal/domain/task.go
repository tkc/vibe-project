package domain

import "time"

// Status はタスクの状態を表す
type Status string

const (
	StatusTodo       Status = "Todo"
	StatusInProgress Status = "InProgress"
	StatusDone       Status = "Done"
	StatusFailed     Status = "Failed"
)

// Task はGitHub Projectのタスクを表す
type Task struct {
	ID         string     // GitHub ProjectのItem ID
	Title      string     // タスクタイトル
	Status     Status     // 現在のステータス
	Prompt     string     // Claude Codeに渡すプロンプト
	WorkDir    string     // 作業ディレクトリ
	Result     string     // 実行結果サマリー
	SessionID  string     // Claude CodeのセッションID
	ExecutedAt *time.Time // 最終実行日時
	IssueURL   string     // 関連Issue/PR URL
}

// IsExecutable はタスクが実行可能かどうかを返す
func (t *Task) IsExecutable() bool {
	return t.Status == StatusTodo && t.Prompt != "" && t.WorkDir != ""
}

// TaskFilter はタスクのフィルタ条件
type TaskFilter struct {
	Status  *Status
	Limit   int
	OrderBy string
}
