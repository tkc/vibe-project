package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tkc/vibe-project/internal/github"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Manage project status",
}

var statusListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available status options",
	RunE: func(cmd *cobra.Command, args []string) error {
		if cfg.GitHubToken == "" {
			return fmt.Errorf("not logged in. Run: vibe auth login")
		}
		if cfg.ProjectOwner == "" || cfg.ProjectNumber == 0 {
			return fmt.Errorf("project not configured. Run: vibe project select")
		}

		ctx := context.Background()
		client := github.NewClient(cfg.GitHubToken, cfg.ProjectOwner)
		taskService := github.NewTaskService(client, cfg.ProjectNumber)

		if err := taskService.Initialize(ctx); err != nil {
			return fmt.Errorf("failed to initialize: %w", err)
		}

		options := taskService.GetStatusOptions()
		if len(options) == 0 {
			fmt.Println("No status options found. Make sure the 'Status' field exists in your project.")
			return nil
		}

		fmt.Println("Available status options:")
		fmt.Println()
		for _, opt := range options {
			fmt.Printf("  • %s\n", opt.Name)
		}
		fmt.Printf("\nTotal: %d options\n", len(options))

		return nil
	},
}

var statusFieldsCmd = &cobra.Command{
	Use:   "fields",
	Short: "List all project fields",
	RunE: func(cmd *cobra.Command, args []string) error {
		if cfg.GitHubToken == "" {
			return fmt.Errorf("not logged in. Run: vibe auth login")
		}
		if cfg.ProjectOwner == "" || cfg.ProjectNumber == 0 {
			return fmt.Errorf("project not configured. Run: vibe project select")
		}

		ctx := context.Background()
		client := github.NewClient(cfg.GitHubToken, cfg.ProjectOwner)
		taskService := github.NewTaskService(client, cfg.ProjectNumber)

		if err := taskService.Initialize(ctx); err != nil {
			return fmt.Errorf("failed to initialize: %w", err)
		}

		fields := taskService.GetFields()
		if len(fields) == 0 {
			fmt.Println("No fields found.")
			return nil
		}

		fmt.Println("Project fields:")
		fmt.Println()
		for name, field := range fields {
			if len(field.Options) > 0 {
				fmt.Printf("  • %s (Single Select)\n", name)
				for _, opt := range field.Options {
					fmt.Printf("      - %s\n", opt.Name)
				}
			} else {
				fmt.Printf("  • %s\n", name)
			}
		}
		fmt.Printf("\nTotal: %d fields\n", len(fields))

		return nil
	},
}

func init() {
	statusCmd.AddCommand(statusListCmd)
	statusCmd.AddCommand(statusFieldsCmd)
}
