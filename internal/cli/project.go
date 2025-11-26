package cli

import (
	"context"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/tkc/vibe-project/internal/github"
)

var projectCmd = &cobra.Command{
	Use:   "project",
	Short: "Manage GitHub Projects",
}

var projectListCmd = &cobra.Command{
	Use:   "list [owner]",
	Short: "List projects for a user or organization",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if cfg.GitHubToken == "" {
			return fmt.Errorf("not logged in. Run: vive auth login")
		}

		var owner string
		if len(args) > 0 {
			owner = args[0]
		} else if cfg.ProjectOwner != "" {
			owner = cfg.ProjectOwner
		} else {
			return fmt.Errorf("owner is required. Usage: vive project list <owner>")
		}
		client := github.NewClient(cfg.GitHubToken, owner)

		ctx := context.Background()
		projects, err := client.GetProjects(ctx)
		if err != nil {
			return fmt.Errorf("failed to get projects: %w", err)
		}

		if len(projects) == 0 {
			fmt.Printf("No projects found for %s\n", owner)
			return nil
		}

		fmt.Printf("Projects for %s:\n\n", owner)
		for _, p := range projects {
			fmt.Printf("  #%-4d %s\n", p.Number, p.Title)
			fmt.Printf("        %s\n\n", p.URL)
		}

		fmt.Println("To select a project, run:")
		fmt.Printf("  vive project select %s <number>\n", owner)
		return nil
	},
}

var projectSelectCmd = &cobra.Command{
	Use:   "select <owner> <number>",
	Short: "Select a project to work with",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if cfg.GitHubToken == "" {
			return fmt.Errorf("not logged in. Run: vive auth login")
		}

		owner := args[0]
		number, err := strconv.Atoi(args[1])
		if err != nil {
			return fmt.Errorf("invalid project number: %s", args[1])
		}

		// プロジェクトが存在するか確認
		client := github.NewClient(cfg.GitHubToken, owner)
		ctx := context.Background()

		project, err := client.GetProjectByNumber(ctx, number)
		if err != nil {
			return fmt.Errorf("failed to find project: %w", err)
		}

		cfg.ProjectOwner = owner
		cfg.ProjectNumber = number
		if err := cfg.Save(); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		fmt.Printf("✓ Selected project: %s (#%d)\n", project.Title, project.Number)
		fmt.Printf("  URL: %s\n", project.URL)
		fmt.Println()
		fmt.Println("Next step: vive task list")
		return nil
	},
}

var projectShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current project",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := cfg.Validate(); err != nil {
			return err
		}

		client := github.NewClient(cfg.GitHubToken, cfg.ProjectOwner)
		ctx := context.Background()

		project, err := client.GetProjectByNumber(ctx, cfg.ProjectNumber)
		if err != nil {
			return fmt.Errorf("failed to get project: %w", err)
		}

		fmt.Printf("Current Project:\n")
		fmt.Printf("  Title:  %s\n", project.Title)
		fmt.Printf("  Number: #%d\n", project.Number)
		fmt.Printf("  Owner:  %s\n", cfg.ProjectOwner)
		fmt.Printf("  URL:    %s\n", project.URL)
		return nil
	},
}

func init() {
	projectCmd.AddCommand(projectListCmd)
	projectCmd.AddCommand(projectSelectCmd)
	projectCmd.AddCommand(projectShowCmd)
}
