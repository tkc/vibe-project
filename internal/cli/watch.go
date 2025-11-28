package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/tkc/vibe-project/internal/claude"
	"github.com/tkc/vibe-project/internal/domain"
	"github.com/tkc/vibe-project/internal/github"
	"github.com/tkc/vibe-project/internal/notify"
)

var (
	watchInterval time.Duration
)

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Watch for new tasks and execute them automatically",
	Long: `Watch the GitHub Project for new Ready tasks and execute them automatically.

This command polls the project at regular intervals, picks up Ready tasks,
executes them one by one using Claude Code, and updates the results.

Press Ctrl+C to stop watching.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := cfg.Validate(); err != nil {
			return err
		}

		// Claude Codeの確認
		executor := claude.NewExecutor(cfg.ClaudePath)
		if err := executor.CheckInstalled(); err != nil {
			return fmt.Errorf("claude is not installed: %w", err)
		}

		// GitHub接続
		client := github.NewClient(cfg.GitHubToken, cfg.ProjectOwner)
		taskSvc := github.NewTaskService(client, cfg.ProjectNumber)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		if err := taskSvc.Initialize(ctx); err != nil {
			return fmt.Errorf("failed to initialize: %w", err)
		}

		fmt.Printf("👀 Watching project #%d for new tasks...\n", cfg.ProjectNumber)
		fmt.Printf("   Interval: %s\n", watchInterval)
		fmt.Println("   Press Ctrl+C to stop")
		fmt.Println()

		// シグナルハンドリング
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

		ticker := time.NewTicker(watchInterval)
		defer ticker.Stop()

		// 初回実行
		processNewTasks(ctx, taskSvc, executor)

		for {
			select {
			case <-ticker.C:
				processNewTasks(ctx, taskSvc, executor)
			case <-sigCh:
				fmt.Println("\n👋 Stopping watch...")
				return nil
			case <-ctx.Done():
				return nil
			}
		}
	},
}

func processNewTasks(ctx context.Context, taskSvc *github.TaskService, executor *claude.Executor) {
	status := domain.StatusReady
	filter := &domain.TaskFilter{Status: &status}

	tasks, err := taskSvc.GetTasks(ctx, filter)
	if err != nil {
		fmt.Printf("⚠️  Failed to get tasks: %v\n", err)
		return
	}

	executableTasks := make([]*domain.Task, 0)
	for _, t := range tasks {
		if t.IsExecutable() {
			executableTasks = append(executableTasks, t)
		}
	}

	if len(executableTasks) == 0 {
		timestamp := time.Now().Format("15:04:05")
		fmt.Printf("[%s] No new tasks\n", timestamp)
		return
	}

	fmt.Printf("📋 Found %d new task(s)\n", len(executableTasks))

	opt := &claude.ExecuteOption{
		Timeout: 30 * time.Minute,
	}

	for _, task := range executableTasks {
		fmt.Printf("▶  Executing: %s\n", task.Title)

		// InProgressに設定
		if err := taskSvc.SetTaskInProgress(ctx, task.ID); err != nil {
			fmt.Printf("   ⚠️  Failed to update status: %v\n", err)
		}

		// 実行
		exec, err := executor.Execute(ctx, task, opt)
		if err != nil {
			fmt.Printf("   ❌ Error: %v\n", err)
			continue
		}

		// 結果を更新
		if err := taskSvc.UpdateTask(ctx, task, exec); err != nil {
			fmt.Printf("   ⚠️  Failed to update task: %v\n", err)
		}

		if exec.Success {
			fmt.Printf("   ✅ Done (%.1fs)\n", exec.Duration.Seconds())
			_ = notify.SendSuccess(task.Title, exec.Duration.Seconds())
		} else {
			fmt.Printf("   ❌ Failed: %s\n", truncate(exec.Error, 100))
			_ = notify.SendFailure(task.Title, exec.Error)
		}
	}
	fmt.Println()
}

func init() {
	watchCmd.Flags().DurationVarP(&watchInterval, "interval", "i", 5*time.Minute, "Polling interval")
}
