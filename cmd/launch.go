package cmd

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/hmans/beans/internal/bean"
	"github.com/hmans/beans/internal/config"
	"github.com/hmans/beans/internal/graph"
	launcherexec "github.com/hmans/beans/internal/launcher"
	"github.com/spf13/cobra"
)

var (
	launchList bool
	launchJSON bool
)

// maxLaunchDepth caps nested `beans launch` calls, e.g. a `prompt` override
// that calls `prompt` instead of `parent.prompt`.
const maxLaunchDepth = 16

var launchCmd = &cobra.Command{
	Use:   "launch <name> [id] [-- args...]",
	Short: "Run a launch for a bean",
	Long: `Runs a launch for a bean, attached to the terminal, and exits with its exit
code. A launch is a launcher configured in .beans.yml or a builtin:

  beans launch claude beans-abc1      configured launcher "claude"
  beans launch agent -- claude        builtin "agent" (when not configured)
  beans launch builtin.tmux-window    always the builtin
  beans launch parent.prompt          "prompt" from the config above the caller's

Arguments after -- are passed to the launch script. The bean ID (or a unique
prefix of it) is taken from the argument, else from $BEANS_ID (set inside
launches), else from the first line of stdin:

  echo beans-abc1 | beans launch claude

Use --list to show the configured launchers and the builtins.`,
	Args: cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if launchList {
			return listLaunches(cmd.OutOrStdout(), cfg.Launchers, launcherexec.Builtins(), launchJSON)
		}

		positional, scriptArgs := splitLaunchArgs(args, cmd.ArgsLenAtDash())
		if len(positional) == 0 {
			return fmt.Errorf("no launch given, see beans launch --list")
		}
		if len(positional) > 2 {
			return fmt.Errorf("too many arguments, pass arguments for the launch after --")
		}

		depth, err := launchDepth(os.Getenv("BEANS_LAUNCH_DEPTH"))
		if err != nil {
			return err
		}
		resolved, err := launcherexec.Resolve(cfg, positional[0], os.Getenv("BEANS_LAUNCH_LAYER"))
		if err != nil {
			return err
		}

		id, fromStdin, err := launchBeanID(positional[1:], os.Getenv("BEANS_ID"), os.Stdin)
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

		// From here on, failures are the launch's, not a usage problem.
		cmd.SilenceUsage = true
		launch := newCLILaunch(resolved, b)
		launch.Args = scriptArgs
		launch.Depth = depth
		return runLaunch(launch, fromStdin)
	},
}

// splitLaunchArgs splits args at the -- cobra saw (dash is
// cmd.ArgsLenAtDash(), -1 without one) into beans' own positional args and
// the launch script's args.
func splitLaunchArgs(args []string, dash int) (positional, scriptArgs []string) {
	if dash < 0 {
		return args, nil
	}
	return args[:dash], args[dash:]
}

// launchDepth parses BEANS_LAUNCH_DEPTH, the number of launches this call is
// nested in, and refuses to go deeper than maxLaunchDepth.
func launchDepth(env string) (int, error) {
	if env == "" {
		return 0, nil
	}
	depth, err := strconv.Atoi(env)
	if err != nil {
		return 0, fmt.Errorf("invalid BEANS_LAUNCH_DEPTH %q", env)
	}
	if depth >= maxLaunchDepth {
		return 0, fmt.Errorf("launches nested more than %d deep, does a launch call itself? (use parent.<name> to call the definition it overrides)", maxLaunchDepth)
	}
	return depth, nil
}

// launchBeanID returns the bean ID from args, else envID ($BEANS_ID), else the
// first non-empty line of stdin. fromStdin reports whether stdin was consumed.
func launchBeanID(args []string, envID string, stdin *os.File) (id string, fromStdin bool, err error) {
	if len(args) > 0 {
		return args[0], false, nil
	}
	if envID != "" {
		return envID, false, nil
	}
	if stat, err := stdin.Stat(); err == nil && stat.Mode()&os.ModeCharDevice != 0 {
		return "", false, fmt.Errorf("no bean ID given, pass it as argument, via $BEANS_ID or via stdin")
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

// newCLILaunch describes running r for b in the current project.
func newCLILaunch(r launcherexec.Resolved, b *bean.Bean) launcherexec.Launch {
	projectRoot := cfg.ConfigDir()
	if projectRoot == "" {
		projectRoot = filepath.Dir(core.Root())
	}
	return launcherexec.Launch{
		Name:        r.Name,
		Script:      r.Script,
		Builtin:     r.Builtin,
		ProjectRoot: projectRoot,
		ConfigDir:   r.Dir,
		ConfigPath:  cfg.LoadedPath(),
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
			return fmt.Errorf("running launch: %w", err)
		}
		return &exitCodeError{code: code, err: fmt.Errorf("launch exited with code %d", code)}
	}
	return nil
}

// launcherInfo is a launch as listed by --list --json.
type launcherInfo struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Exec        string `json:"exec,omitempty"`
	Multiple    bool   `json:"multiple"`
	Background  bool   `json:"background"`
	Builtin     bool   `json:"builtin,omitempty"`
}

// listLaunches writes the configured launchers and the builtins to w, as a
// table or as JSON.
func listLaunches(w io.Writer, launchers []config.Launcher, builtins []launcherexec.Builtin, asJSON bool) error {
	if asJSON {
		infos := make([]launcherInfo, 0, len(launchers)+len(builtins))
		for _, l := range launchers {
			infos = append(infos, launcherInfo{Name: l.Name, Description: l.Description, Exec: l.Exec, Multiple: l.Multiple, Background: l.Background})
		}
		for _, b := range builtins {
			infos = append(infos, launcherInfo{Name: "builtin." + b.Name, Description: b.Description, Builtin: true})
		}
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(infos)
	}

	if len(launchers) == 0 {
		fmt.Fprintln(w, "No launchers configured.")
	}
	rows := make([][]string, 0, len(launchers))
	for _, l := range launchers {
		var flags []string
		if l.Multiple {
			flags = append(flags, "multiple")
		}
		if l.Background {
			flags = append(flags, "background")
		}
		details := []string{l.Description}
		if len(flags) > 0 {
			details = append(details, "["+strings.Join(flags, ", ")+"]")
		}
		rows = append(rows, []string{l.Name, strings.TrimSpace(strings.Join(details, "  "))})
	}
	writeRows(w, rows)

	if len(builtins) > 0 {
		fmt.Fprintln(w)
		rows = rows[:0]
		for _, b := range builtins {
			rows = append(rows, []string{"builtin." + b.Name, b.Description})
		}
		writeRows(w, rows)
	}
	return nil
}

// writeRows writes name/details rows with the names padded to one width.
func writeRows(w io.Writer, rows [][]string) {
	width := 0
	for _, r := range rows {
		width = max(width, len(r[0]))
	}
	for _, r := range rows {
		fmt.Fprintln(w, strings.TrimRight(fmt.Sprintf("%-*s  %s", width, r[0], r[1]), " "))
	}
}

func init() {
	launchCmd.Flags().BoolVar(&launchList, "list", false, "List configured launchers and builtins")
	launchCmd.Flags().BoolVar(&launchJSON, "json", false, "Output the --list as JSON")
	rootCmd.AddCommand(launchCmd)
}
