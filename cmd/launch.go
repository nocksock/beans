package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/hmans/beans/internal/config"
	"github.com/hmans/beans/internal/graph"
	"github.com/hmans/beans/internal/launcher"
	"github.com/spf13/cobra"
)

var (
	launchList bool
	launchJSON bool
)

var launchCmd = &cobra.Command{
	Use:   "launch [flags] [<launcher-name> <bean-id>]",
	Short: "Launch external tools for a bean",
	Long: `Execute a configured launcher for a specific bean.

Launchers are defined in .beans.yml and can open beans in external tools
or execute custom scripts.

Examples:
  beans launch -l                     # List all configured launchers
  beans launch opencode abc123        # Launch OpenCode for bean abc123
  beans launch worktree-tmux def456   # Create worktree and open in tmux
  beans launch -l --json              # List launchers in JSON format`,
	Args: cobra.MaximumNArgs(2),
	RunE: runLaunch,
}

func init() {
	launchCmd.Flags().BoolVarP(&launchList, "list", "l", false, "List all configured launchers")
	launchCmd.Flags().BoolVar(&launchJSON, "json", false, "Output as JSON (with --list)")
	rootCmd.AddCommand(launchCmd)
}

func runLaunch(cmd *cobra.Command, args []string) error {
	// Handle --list flag
	if launchList {
		return listLaunchers()
	}

	// Require both launcher name and bean ID
	if len(args) != 2 {
		return fmt.Errorf("requires launcher name and bean ID\n\nUsage:\n  beans launch <launcher-name> <bean-id>\n  beans launch -l")
	}

	launcherName := args[0]
	beanID := args[1]

	return executeLauncher(launcherName, beanID)
}

func listLaunchers() error {
	if len(cfg.Launchers) == 0 {
		if launchJSON {
			fmt.Println("[]")
			return nil
		}
		fmt.Println("No launchers configured.")
		fmt.Println("\nAdd launchers to .beans.yml:")
		fmt.Println("launchers:")
		fmt.Println("  - name: opencode")
		fmt.Println("    exec: opencode -p \"Work on task $BEANS_ID\"")
		fmt.Println("    description: \"Open task in OpenCode\"")
		return nil
	}

	if launchJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(cfg.Launchers)
	}

	// Table output with availability status
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "NAME\tAVAILABLE\tDESCRIPTION")

	for _, l := range cfg.Launchers {
		desc := l.Description
		if desc == "" {
			desc = "-"
		}

		// Check availability
		available := "✓"
		if !launcher.IsLauncherAvailable(l.Exec) {
			available = "✗"
		}

		fmt.Fprintf(w, "%s\t%s\t%s\n", l.Name, available, desc)
	}
	w.Flush()

	return nil
}

func executeLauncher(launcherName, beanID string) error {
	// Find launcher by exact name match
	var foundLauncher *config.Launcher
	for i := range cfg.Launchers {
		if cfg.Launchers[i].Name == launcherName {
			foundLauncher = &cfg.Launchers[i]
			break
		}
	}

	if foundLauncher == nil {
		return fmt.Errorf("launcher not found: %s\n\nRun 'beans launch -l' to see available launchers", launcherName)
	}

	// Verify bean exists (supports partial ID matching)
	resolver := &graph.Resolver{Core: core}
	bean, err := resolver.Query().Bean(context.Background(), beanID)
	if err != nil || bean == nil {
		return fmt.Errorf("bean not found: %s", beanID)
	}

	// Use resolved full ID (in case partial was provided)
	fullBeanID := bean.ID

	// Execute launcher
	beanPath := filepath.Join(core.Root(), "beans-"+fullBeanID+".md")
	beansRoot := cfg.ConfigDir()

	result := launcher.CreateExecCommand(foundLauncher.Exec, fullBeanID, beansRoot, beanPath)
	defer result.Cleanup()

	// Connect to stdin/stdout/stderr for interactive execution
	result.Cmd.Stdin = os.Stdin
	result.Cmd.Stdout = os.Stdout
	result.Cmd.Stderr = os.Stderr

	// Execute and wait
	if err := result.Cmd.Run(); err != nil {
		return fmt.Errorf("launcher failed: %w", err)
	}

	return nil
}
