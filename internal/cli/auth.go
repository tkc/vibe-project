package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage authentication",
}

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login to GitHub",
	Long: `Login to GitHub using a personal access token.

For Classic tokens (https://github.com/settings/tokens):
  Required scopes:
    - project (Full control of projects)
    - read:org (for organization projects)
    - repo (Required for commenting on issues)

For Fine-grained tokens (https://github.com/settings/tokens?type=beta):
  Account permissions:
    - Projects: Read and write

Create a token at: https://github.com/settings/tokens`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("GitHub Personal Access Token を入力してください")
		fmt.Println("(必要なスコープ: project, read:org, repo)")
		fmt.Println()
		fmt.Print("Token: ")

		reader := bufio.NewReader(os.Stdin)
		token, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read token: %w", err)
		}
		token = strings.TrimSpace(token)

		if token == "" {
			return fmt.Errorf("token cannot be empty")
		}

		cfg.GitHubToken = token
		if err := cfg.Save(); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		fmt.Println("✓ Token saved successfully")
		fmt.Println()
		fmt.Println("Next step: vibe project select")
		return nil
	},
}

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show authentication status",
	RunE: func(cmd *cobra.Command, args []string) error {
		if cfg.GitHubToken == "" {
			fmt.Println("✗ Not logged in")
			fmt.Println()
			fmt.Println("Run: vibe auth login")
			return nil
		}

		// トークンの一部を表示
		token := cfg.GitHubToken
		masked := token[:4] + strings.Repeat("*", len(token)-8) + token[len(token)-4:]
		fmt.Printf("✓ Logged in (token: %s)\n", masked)

		if cfg.ProjectOwner != "" {
			fmt.Printf("  Project: %s #%d\n", cfg.ProjectOwner, cfg.ProjectNumber)
		}
		return nil
	},
}

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Logout from GitHub",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg.GitHubToken = ""
		if err := cfg.Save(); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}
		fmt.Println("✓ Logged out successfully")
		return nil
	},
}

func init() {
	authCmd.AddCommand(authLoginCmd)
	authCmd.AddCommand(authStatusCmd)
	authCmd.AddCommand(authLogoutCmd)
}
