package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/tkc/vibe-project/internal/config"
)

var (
	cfg     *config.Config
	verbose bool
)

// rootCmd はルートコマンド
var rootCmd = &cobra.Command{
	Use:   "vibe",
	Short: "GitHub Project + Claude Code automation CLI",
	Long: `vibe is a CLI tool that integrates GitHub Projects with Claude Code.

It fetches tasks from GitHub Projects, executes them using Claude Code,
and updates the results back to the project.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		var err error
		cfg, err = config.LoadWithPrecedence()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		return nil
	},
}

// Execute はCLIを実行する
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")

	rootCmd.AddCommand(authCmd)
	rootCmd.AddCommand(projectCmd)
	rootCmd.AddCommand(taskCmd)
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(watchCmd)
	rootCmd.AddCommand(statusCmd)
}
