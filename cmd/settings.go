package cmd

import (
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/hmans/beans/internal/config"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var settingsCmd = &cobra.Command{
	Use:   "settings",
	Short: "Read and change settings in .beans.yml",
	Long: `Read and change settings in .beans.yml.

Settings are addressed by dotted keys, e.g. "beans.prefix" or
"tui.disable_launcher_warning".`,
}

var settingsGetCmd = &cobra.Command{
	Use:   "get [key]",
	Short: "Show the effective value of a setting",
	Long: `Shows the effective value of a setting, after defaults, the user config and
cascading configs are applied. Without a key, shows the whole config.

List items can be addressed by index or name, e.g. "launchers.claude.exec".`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := ""
		if len(args) == 1 {
			key = args[0]
		}

		value, err := cfg.Get(key)
		if err != nil {
			return err
		}
		out, err := formatSetting(value)
		if err != nil {
			return err
		}
		fmt.Print(out)
		return nil
	},
}

var settingsSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Change a setting in the nearest .beans.yml",
	Long: `Changes a setting in the nearest .beans.yml (or the file given with --config),
keeping the rest of the file and its comments intact.

The value is parsed as YAML: "6" is a number, "true" a boolean and "[a, b]" a list.
Config files that take part in cascading can't be changed yet.`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		path, err := settingsFile()
		if err != nil {
			return err
		}
		if err := config.SetSetting(path, args[0], args[1]); err != nil {
			return err
		}
		fmt.Printf("Set %s in %s\n", args[0], path)
		return nil
	},
}

// isSettingsCmd reports whether cmd is one of the settings commands.
func isSettingsCmd(cmd *cobra.Command) bool {
	for c := cmd; c != nil; c = c.Parent() {
		if c == settingsCmd {
			return true
		}
	}
	return false
}

// settingsFile returns the config file that settings are written to.
func settingsFile() (string, error) {
	if configPath != "" {
		return configPath, nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	path, err := config.FindConfig(cwd)
	if err != nil {
		return "", err
	}
	if path == "" {
		return "", fmt.Errorf("no %s found (run 'beans init' to create one)", config.ConfigFileName)
	}
	return path, nil
}

// formatSetting renders a setting's value: scalars as plain text, anything
// else as YAML.
func formatSetting(value any) (string, error) {
	switch reflect.ValueOf(value).Kind() {
	case reflect.Struct, reflect.Map, reflect.Slice:
		var out strings.Builder
		enc := yaml.NewEncoder(&out)
		enc.SetIndent(2)
		if err := enc.Encode(value); err != nil {
			return "", err
		}
		return out.String(), enc.Close()
	default:
		return fmt.Sprintf("%v\n", value), nil
	}
}

func init() {
	settingsCmd.AddCommand(settingsGetCmd, settingsSetCmd)
	rootCmd.AddCommand(settingsCmd)
}
