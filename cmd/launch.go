package cmd

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/hmans/beans/internal/bean"
	"github.com/hmans/beans/internal/config"
	"github.com/hmans/beans/internal/graph"
	launcherexec "github.com/hmans/beans/internal/launcher"
	"github.com/spf13/cobra"
)

var (
	launchLauncher string
	launchList     bool
	launchJSON     bool
)

var launchCmd = &cobra.Command{
	Use:   "launch [id]",
	Short: "Run a configured launcher for a bean",
	Long: `Runs one of the launchers configured in .beans.yml for a bean, attached to
the terminal, and exits with the launcher's exit code.

The bean ID (or a unique prefix of it) is taken from the argument, or from the
first line of stdin when no argument is given:

  beans launch beans-abc1 --launcher claude
  echo beans-abc1 | beans launch --launcher claude

Use --list to show the configured launchers.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if launchList {
			return listLaunchers(cmd.OutOrStdout(), cfg.Launchers, launchJSON)
		}
		if launchLauncher == "" {
			return fmt.Errorf("no launcher given, use --launcher <name> (see --list)")
		}

		l, err := findLauncher(cfg.Launchers, launchLauncher)
		if err != nil {
			return err
		}

		id, fromStdin, err := launchBeanID(args, os.Stdin)
		if err != nil {
			return err
		}

		resolver := &graph.Resolver{Core: core}
		b, err := resolver.Query().Bean(context.Background(), id)
		if err != nil {
			return handleBeanQueryError(err, false, id)
		}
		if b == nil {
			return fmt.Errorf("bean not found: %s", id)
		}

		// From here on, failures are the launcher's, not a usage problem.
		cmd.SilenceUsage = true
		return runLaunch(newCLILaunch(l, b), fromStdin)
	},
}

// launchBeanID returns the bean ID from args, or else from the first non-empty
// line of stdin. fromStdin reports whether stdin was consumed for it.
func launchBeanID(args []string, stdin *os.File) (id string, fromStdin bool, err error) {
	if len(args) > 0 {
		return args[0], false, nil
	}
	if stat, err := stdin.Stat(); err == nil && stat.Mode()&os.ModeCharDevice != 0 {
		return "", false, fmt.Errorf("no bean ID given, pass it as argument or via stdin")
	}
	id, err = firstLine(stdin)
	if err != nil {
		return "", true, err
	}
	if id == "" {
		return "", true, fmt.Errorf("no bean ID given, stdin was empty")
	}
	return id, true, nil
}

// firstLine returns the first non-empty line of r, trimmed.
func firstLine(r io.Reader) (string, error) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		if line := strings.TrimSpace(scanner.Text()); line != "" {
			return line, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("reading stdin: %w", err)
	}
	return "", nil
}

// findLauncher returns the launcher with the given name.
func findLauncher(launchers []config.Launcher, name string) (config.Launcher, error) {
	var names []string
	for _, l := range launchers {
		if l.Name == name {
			return l, nil
		}
		names = append(names, l.Name)
	}
	if len(names) == 0 {
		return config.Launcher{}, fmt.Errorf("unknown launcher %q, no launchers are configured", name)
	}
	return config.Launcher{}, fmt.Errorf("unknown launcher %q, available: %s", name, strings.Join(names, ", "))
}

// newCLILaunch describes running l for b in the current project.
func newCLILaunch(l config.Launcher, b *bean.Bean) launcherexec.Launch {
	projectRoot := cfg.ConfigDir()
	if projectRoot == "" {
		projectRoot = filepath.Dir(core.Root())
	}
	return launcherexec.Launch{
		Script:      l.Exec,
		ProjectRoot: projectRoot,
		ConfigDir:   l.Dir,
		BeansDir:    core.Root(),
		BeanID:      b.ID,
		BeanTitle:   b.DisplayTitle(),
	}
}

// runLaunch runs launch attached to the terminal and waits for it. When stdin
// was used up for the bean ID, the launcher gets the controlling terminal
// instead (if there is one), so interactive launchers keep working.
func runLaunch(launch launcherexec.Launch, stdinConsumed bool) error {
	c := launcherexec.CreateShellCommand(launch)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	if stdinConsumed {
		c.Stdin = nil
		if tty, err := os.Open("/dev/tty"); err == nil {
			defer tty.Close()
			c.Stdin = tty
		}
	}

	if err := c.Run(); err != nil {
		code := launcherexec.ExitCode(err)
		if code < 0 {
			return fmt.Errorf("running launcher: %w", err)
		}
		return &exitCodeError{code: code, err: fmt.Errorf("launcher exited with code %d", code)}
	}
	return nil
}

// launcherInfo is a launcher as listed by --list --json.
type launcherInfo struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Exec        string `json:"exec"`
	Multiple    bool   `json:"multiple"`
	Background  bool   `json:"background"`
}

// listLaunchers writes the configured launchers to w, as a table or as JSON.
func listLaunchers(w io.Writer, launchers []config.Launcher, asJSON bool) error {
	if asJSON {
		infos := make([]launcherInfo, 0, len(launchers))
		for _, l := range launchers {
			infos = append(infos, launcherInfo{l.Name, l.Description, l.Exec, l.Multiple, l.Background})
		}
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(infos)
	}

	if len(launchers) == 0 {
		fmt.Fprintln(w, "No launchers configured.")
		return nil
	}
	width := 0
	for _, l := range launchers {
		width = max(width, len(l.Name))
	}
	for _, l := range launchers {
		var flags []string
		if l.Multiple {
			flags = append(flags, "multiple")
		}
		if l.Background {
			flags = append(flags, "background")
		}
		details := []string{fmt.Sprintf("%-*s", width, l.Name)}
		if l.Description != "" {
			details = append(details, l.Description)
		}
		if len(flags) > 0 {
			details = append(details, "["+strings.Join(flags, ", ")+"]")
		}
		fmt.Fprintln(w, strings.TrimRight(strings.Join(details, "  "), " "))
	}
	return nil
}

func init() {
	launchCmd.Flags().StringVarP(&launchLauncher, "launcher", "l", "", "Name of the launcher to run")
	launchCmd.Flags().BoolVar(&launchList, "list", false, "List configured launchers")
	launchCmd.Flags().BoolVar(&launchJSON, "json", false, "Output the --list as JSON")
	rootCmd.AddCommand(launchCmd)
}
