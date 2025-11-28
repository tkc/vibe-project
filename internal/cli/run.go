package cli

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/tkc/vibe-project/internal/claude"
	"github.com/tkc/vibe-project/internal/domain"
	"github.com/tkc/vibe-project/internal/github"
	"github.com/tkc/vibe-project/internal/notify"
)

var (
	runDryRun  bool
	runTimeout time.Duration
)

var runCmd = &cobra.Command{
	Use:   "run [task-id]",
	Short: "Execute a Ready task using Claude Code",
	Long: `Execute a task using Claude Code.

If no task ID is specified, the first Ready task will be executed.
The task's Prompt field will be passed to Claude Code,
and the result will be commented on the associated Issue.

Examples:
  vibe run              # Run the first Ready task
  vibe run <task-id>    # Run a specific task
  vibe run --dry-run    # Preview without executing`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := cfg.Validate(); err != nil {
			return err
		}

		// Claude Codeの確認
		executor := claude.NewExecutor(cfg.ClaudePath)
		if !runDryRun {
			if err := executor.CheckInstalled(); err != nil {
				return fmt.Errorf("claude is not installed: %w", err)
			}
		}

		// GitHub接続
		client := github.NewClient(cfg.GitHubToken, cfg.ProjectOwner)
		taskSvc := github.NewTaskService(client, cfg.ProjectNumber)

		ctx := context.Background()
		if err := taskSvc.Initialize(ctx); err != nil {
			return fmt.Errorf("failed to initialize: %w", err)
		}

		// タスク取得
		var task *domain.Task
		var err error

		if len(args) > 0 {
			// 指定されたタスクIDを取得
			task, err = taskSvc.GetTask(ctx, args[0])
			if err != nil {
				return fmt.Errorf("failed to get task: %w", err)
			}
		} else {
			// Readyの最初のタスクを取得
			task, err = taskSvc.GetFirstReadyTask(ctx)
			if err != nil {
				return fmt.Errorf("failed to get ready task: %w", err)
			}
			if task == nil {
				fmt.Println("No Ready tasks found")
				return nil
			}
		}

		// WorkDirが空の場合はカレントディレクトリを使用
		if task.WorkDir == "" {
			wd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get current working directory: %w", err)
			}
			task.WorkDir = wd
		}

		// Issueのコメントからプロンプトを読み込む
		fmt.Println("📥 Loading prompt from issue comments...")
		if err := taskSvc.LoadTaskPrompt(ctx, task); err != nil {
			return fmt.Errorf("failed to load prompt: %w", err)
		}

		// 実行可能か確認
		if !task.IsExecutable() {
			return fmt.Errorf("task is not executable (Status: %s, Prompt: %v)",
				task.Status, task.Prompt != "")
		}

		fmt.Printf("📋 Task: %s\n", task.Title)
		fmt.Printf("   ID: %s\n", task.ID)
		fmt.Printf("   WorkDir: %s\n", task.WorkDir)
		fmt.Printf("   Prompt: %s\n", truncate(task.Prompt, 80))
		fmt.Println()

		// ドライラン
		if runDryRun {
			fmt.Println("[DRY RUN] Would execute:")
			fmt.Printf("  claude --print \"%s\"\n", truncate(task.Prompt, 50))
			return nil
		}

		// InProgressに設定
		fmt.Println("⏳ Setting status to InProgress...")
		if err := taskSvc.SetTaskInProgress(ctx, task.ID); err != nil {
			fmt.Printf("   ⚠️  Failed to update status: %v\n", err)
		}

		// 実行オプション
		opt := &claude.ExecuteOption{
			Timeout: runTimeout,
		}

		// Claude Code実行
		fmt.Println("🚀 Executing Claude Code...")
		exec, err := executor.Execute(ctx, task, opt)
		if err != nil {
			return fmt.Errorf("execution error: %w", err)
		}

		// 結果を表示
		fmt.Println()
		if exec.Success {
			fmt.Printf("✅ Completed (%.1fs)\n", exec.Duration.Seconds())
			// macOS notification
			_ = notify.SendSuccess(task.Title, exec.Duration.Seconds())
		} else {
			fmt.Printf("❌ Failed (%.1fs)\n", exec.Duration.Seconds())
			fmt.Printf("   Error: %s\n", truncate(exec.Error, 100))
			// macOS notification
			_ = notify.SendFailure(task.Title, exec.Error)
		}

		// Projectのフィールドを更新
		fmt.Println()
		fmt.Println("📝 Updating project fields...")
		if err := taskSvc.UpdateTask(ctx, task, exec); err != nil {
			fmt.Printf("   ⚠️  Failed to update task: %v\n", err)
		}

		// Issueにコメント
		if task.IssueURL != "" {
			fmt.Println("💬 Adding comment to Issue...")
			comment := buildIssueComment(task, exec)
			if err := taskSvc.AddIssueComment(ctx, task, comment); err != nil {
				fmt.Printf("   ⚠️  Failed to add comment: %v\n", err)
			} else {
				fmt.Println("   ✅ Comment added")
			}
		}

		fmt.Println()
		fmt.Println("🎉 Done!")
		return nil
	},
}

// buildIssueComment は実行結果から3行程度のコメントを生成する
func buildIssueComment(task *domain.Task, exec *domain.Execution) string {
	status := "✅ Completed"
	if !exec.Success {
		status = "❌ Failed"
	}

	// サマリーを3行以内に収める
	summary := exec.Summary()
	lines := splitLines(summary, 3)

	comment := fmt.Sprintf(`vibe project comment

## Claude Code Execution Result

**Status:** %s
**Duration:** %.1fs

### Summary
%s

---
*Executed by vibe-project*`, status, exec.Duration.Seconds(), lines)

	return comment
}

// splitLines は文字列を指定行数に分割する
func splitLines(s string, maxLines int) string {
	if len(s) == 0 {
		return "(no output)"
	}

	// 改行で分割
	result := ""
	lineCount := 0
	for i, c := range s {
		if c == '\n' {
			lineCount++
			if lineCount >= maxLines {
				return result + "..."
			}
		}
		result += string(c)
		// 1行が長すぎる場合も切る
		if i > 500 {
			return result + "..."
		}
	}
	return result
}

func init() {
	runCmd.Flags().BoolVar(&runDryRun, "dry-run", false, "Preview execution without running")
	runCmd.Flags().DurationVar(&runTimeout, "timeout", 30*time.Minute, "Timeout for the task")
}
