package claude

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/tkc/vibe-project/internal/domain"
)

// Executor はClaude Codeを実行する
type Executor struct {
	claudePath string
}

// NewExecutor は新しいExecutorを作成する
func NewExecutor(claudePath string) *Executor {
	return &Executor{
		claudePath: claudePath,
	}
}

// ExecuteOption は実行オプション
type ExecuteOption struct {
	DryRun    bool          // 実行せず確認のみ
	Timeout   time.Duration // タイムアウト
	SessionID string        // 継続するセッションID
}

// DefaultTimeout はデフォルトのタイムアウト時間
const DefaultTimeout = 30 * time.Minute

// Execute はタスクを実行する
func (e *Executor) Execute(ctx context.Context, task *domain.Task, opt *ExecuteOption) (*domain.Execution, error) {
	if opt == nil {
		opt = &ExecuteOption{}
	}
	if opt.Timeout == 0 {
		opt.Timeout = DefaultTimeout
	}

	execution := &domain.Execution{
		TaskID:    task.ID,
		StartedAt: time.Now(),
	}

	// ドライラン
	if opt.DryRun {
		execution.Success = true
		execution.Output = e.dryRunOutput(task, opt)
		execution.EndedAt = time.Now()
		execution.Duration = execution.EndedAt.Sub(execution.StartedAt)
		return execution, nil
	}

	// コマンドを構築
	args := e.buildArgs(task, opt)

	// タイムアウト付きコンテキスト
	ctx, cancel := context.WithTimeout(ctx, opt.Timeout)
	defer cancel()

	// 実行
	cmd := exec.CommandContext(ctx, e.claudePath, args...)
	cmd.Dir = task.WorkDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	execution.EndedAt = time.Now()
	execution.Duration = execution.EndedAt.Sub(execution.StartedAt)

	if err != nil {
		execution.Success = false
		execution.Error = formatError(err, stderr.String())
		execution.Output = stdout.String()
		return execution, nil // エラーは返さない（Failedとして処理）
	}

	execution.Success = true
	execution.Output = stdout.String()

	// セッションIDを抽出（あれば）
	execution.SessionID = e.extractSessionID(stdout.String())

	return execution, nil
}

func (e *Executor) buildArgs(task *domain.Task, opt *ExecuteOption) []string {
	args := []string{
		"--print", // 非対話モード
	}

	// セッション継続
	if opt.SessionID != "" {
		args = append(args, "--resume", opt.SessionID)
	} else if task.SessionID != "" {
		args = append(args, "--resume", task.SessionID)
	}

	// プロンプトを追加
	args = append(args, task.Prompt)

	return args
}

func (e *Executor) dryRunOutput(task *domain.Task, opt *ExecuteOption) string {
	args := e.buildArgs(task, opt)
	return fmt.Sprintf(`[DRY RUN] Would execute:
  Command: %s %s
  WorkDir: %s
  Prompt:  %s`, e.claudePath, strings.Join(args, " "), task.WorkDir, task.Prompt)
}

func (e *Executor) extractSessionID(output string) string {
	// Claude Codeの出力からセッションIDを抽出
	// 出力形式によって調整が必要
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "session") || strings.Contains(line, "Session") {
			// JSONとしてパースを試みる
			var data map[string]interface{}
			if err := json.Unmarshal([]byte(line), &data); err == nil {
				if sid, ok := data["session_id"].(string); ok {
					return sid
				}
			}
		}
	}
	return ""
}

func formatError(err error, stderr string) string {
	if stderr != "" {
		return fmt.Sprintf("%v: %s", err, stderr)
	}
	return err.Error()
}

// CheckInstalled はclaude コマンドがインストールされているか確認する
func (e *Executor) CheckInstalled() error {
	cmd := exec.Command(e.claudePath, "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("claude command not found at %s: %w", e.claudePath, err)
	}
	return nil
}
