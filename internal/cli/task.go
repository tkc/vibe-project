package cli

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/tkc/vibe-project/internal/domain"
	"github.com/tkc/vibe-project/internal/github"
)

var (
	taskStatusFilter string
)

var taskCmd = &cobra.Command{
	Use:   "task",
	Short: "Manage tasks",
}

var taskListCmd = &cobra.Command{
	Use:   "list",
	Short: "List tasks in the project",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := cfg.Validate(); err != nil {
			return err
		}

		client := github.NewClient(cfg.GitHubToken, cfg.ProjectOwner)
		taskSvc := github.NewTaskService(client, cfg.ProjectNumber)

		ctx := context.Background()
		if err := taskSvc.Initialize(ctx); err != nil {
			return fmt.Errorf("failed to initialize: %w", err)
		}

		var filter *domain.TaskFilter
		if taskStatusFilter != "" {
			status := domain.Status(taskStatusFilter)
			filter = &domain.TaskFilter{Status: &status}
		}

		tasks, err := taskSvc.GetTasks(ctx, filter)
		if err != nil {
			return fmt.Errorf("failed to get tasks: %w", err)
		}

		if len(tasks) == 0 {
			fmt.Println("No tasks found")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "STATUS\tTITLE\tID")
		fmt.Fprintln(w, "------\t-----\t--")

		for _, t := range tasks {
			status := statusIcon(t.Status) + " " + string(t.Status)
			// IDを短縮表示
			shortID := t.ID
			if len(shortID) > 12 {
				shortID = shortID[:12] + "..."
			}
			fmt.Fprintf(w, "%s\t%s\t%s\n", status, truncate(t.Title, 50), shortID)
		}
		w.Flush()

		fmt.Println()
		fmt.Printf("Total: %d tasks\n", len(tasks))
		return nil
	},
}

var taskShowCmd = &cobra.Command{
	Use:   "show <task-id>",
	Short: "Show task details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := cfg.Validate(); err != nil {
			return err
		}

		taskID := args[0]

		client := github.NewClient(cfg.GitHubToken, cfg.ProjectOwner)
		taskSvc := github.NewTaskService(client, cfg.ProjectNumber)

		ctx := context.Background()
		if err := taskSvc.Initialize(ctx); err != nil {
			return fmt.Errorf("failed to initialize: %w", err)
		}

		task, err := taskSvc.GetTask(ctx, taskID)
		if err != nil {
			return fmt.Errorf("failed to get task: %w", err)
		}

		printTaskDetail(task)
		return nil
	},
}

func printTaskDetail(t *domain.Task) {
	fmt.Printf("Task: %s\n", t.Title)
	fmt.Printf("ID:     %s\n", t.ID)
	fmt.Printf("Status: %s %s\n", statusIcon(t.Status), t.Status)
	fmt.Println()

	if t.Prompt != "" {
		fmt.Println("Prompt:")
		fmt.Println("  " + t.Prompt)
		fmt.Println()
	}

	if t.WorkDir != "" {
		fmt.Printf("WorkDir: %s\n", t.WorkDir)
	}

	if t.Result != "" {
		fmt.Println()
		fmt.Println("Result:")
		fmt.Println("  " + t.Result)
	}

	if t.SessionID != "" {
		fmt.Printf("\nSessionID: %s\n", t.SessionID)
	}

	if t.ExecutedAt != nil {
		fmt.Printf("ExecutedAt: %s\n", t.ExecutedAt.Format("2006-01-02 15:04:05"))
	}

	if t.IssueURL != "" {
		fmt.Printf("\nIssue: %s\n", t.IssueURL)
	}

	if t.IsExecutable() {
		fmt.Println()
		fmt.Println("This task is executable. Run:")
		fmt.Printf("  vive run %s\n", t.ID)
	}
}

func statusIcon(s domain.Status) string {
	switch s {
	case domain.StatusTodo:
		return "○"
	case domain.StatusInProgress:
		return "◐"
	case domain.StatusDone:
		return "●"
	case domain.StatusFailed:
		return "✗"
	default:
		return "?"
	}
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

func init() {
	taskListCmd.Flags().StringVarP(&taskStatusFilter, "status", "s", "", "Filter by status (Todo, InProgress, Done, Failed)")

	taskCmd.AddCommand(taskListCmd)
	taskCmd.AddCommand(taskShowCmd)
}
