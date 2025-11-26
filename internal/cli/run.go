package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/tkc/vibe-project/internal/claude"
	"github.com/tkc/vibe-project/internal/domain"
	"github.com/tkc/vibe-project/internal/github"
)

var (
	runDryRun  bool
	runAll     bool
	runTimeout time.Duration
	runResume  string
)

var runCmd = &cobra.Command{
	Use:   "run [task-id]",
	Short: "Execute task(s) using Claude Code",
	Long: `Execute one or more tasks using Claude Code.

The task's Prompt field will be passed to Claude Code,
and the result will be updated back to the GitHub Project.

Examples:
  vive run <task-id>           # Run a specific task
  vive run --all               # Run all Todo tasks
  vive run <task-id> --dry-run # Preview without executing`,
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
		var tasks []*domain.Task
		if runAll {
			status := domain.StatusTodo
			filter := &domain.TaskFilter{Status: &status}
			var err error
			tasks, err = taskSvc.GetTasks(ctx, filter)
			if err != nil {
				return fmt.Errorf("failed to get tasks: %w", err)
			}
		} else if len(args) > 0 {
			task, err := taskSvc.GetTask(ctx, args[0])
			if err != nil {
				return fmt.Errorf("failed to get task: %w", err)
			}
			tasks = []*domain.Task{task}
		} else {
			return fmt.Errorf("specify task ID or use --all flag")
		}

		if len(tasks) == 0 {
			fmt.Println("No tasks to execute")
			return nil
		}

		// 実行対象を確認
		executableTasks := make([]*domain.Task, 0)
		for _, t := range tasks {
			if t.IsExecutable() {
				executableTasks = append(executableTasks, t)
			} else {
				fmt.Printf("⏭  Skipping %s (not executable)\n", t.Title)
			}
		}

		if len(executableTasks) == 0 {
			fmt.Println("No executable tasks found")
			return nil
		}

		fmt.Printf("Found %d executable task(s)\n\n", len(executableTasks))

		// 実行オプション
		opt := &claude.ExecuteOption{
			DryRun:    runDryRun,
			Timeout:   runTimeout,
			SessionID: runResume,
		}

		// 1つずつ順番に実行
		var succeeded, failed int
		for i, task := range executableTasks {
			fmt.Printf("[%d/%d] %s\n", i+1, len(executableTasks), task.Title)
			fmt.Printf("       WorkDir: %s\n", task.WorkDir)

			if runDryRun {
				fmt.Println("       [DRY RUN]")
				exec, _ := executor.Execute(ctx, task, opt)
				fmt.Println(exec.Output)
				fmt.Println()
				continue
			}

			// InProgressに設定
			if err := taskSvc.SetTaskInProgress(ctx, task.ID); err != nil {
				fmt.Printf("       ⚠️  Failed to update status: %v\n", err)
			}

			// 実行
			fmt.Println("       Executing...")
			exec, err := executor.Execute(ctx, task, opt)
			if err != nil {
				fmt.Printf("       ❌ Error: %v\n", err)
				failed++
				continue
			}

			// 結果を更新
			if err := taskSvc.UpdateTask(ctx, task, exec); err != nil {
				fmt.Printf("       ⚠️  Failed to update task: %v\n", err)
			}

			if exec.Success {
				fmt.Printf("       ✅ Done (%.1fs)\n", exec.Duration.Seconds())
				succeeded++
			} else {
				fmt.Printf("       ❌ Failed: %s\n", truncate(exec.Error, 100))
				failed++
			}
			fmt.Println()
		}

		// サマリー
		if !runDryRun {
			fmt.Println("=" + "=================================")
			fmt.Printf("Results: %d succeeded, %d failed\n", succeeded, failed)
		}

		return nil
	},
}

func init() {
	runCmd.Flags().BoolVar(&runDryRun, "dry-run", false, "Preview execution without running")
	runCmd.Flags().BoolVar(&runAll, "all", false, "Run all Todo tasks")
	runCmd.Flags().DurationVar(&runTimeout, "timeout", 30*time.Minute, "Timeout for each task")
	runCmd.Flags().StringVar(&runResume, "resume", "", "Resume from a specific session ID")
}
