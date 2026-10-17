package cmd

import (
	_ "embed"
	"os"
	"text/template"

	"github.com/spf13/cobra"
	"github.com/hmans/beans/internal/config"
)

//go:embed prompt.tmpl
var agentPromptTemplate string

// promptData holds all data needed to render the prompt template.
type promptData struct {
	GraphQLSchema string
	Types         []config.TypeConfig
	Statuses      []config.StatusConfig
	Priorities    []config.PriorityConfig
}

var primeCmd = &cobra.Command{
	Use:   "prime",
	Short: "Output instructions for AI coding agents",
	Long:  `Outputs a prompt that primes AI coding agents on how to use the beans CLI to manage project issues.`,
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		// If no explicit path given, check if a beans project exists by searching
		// upward for a .beans.yml config file
		if beansPath == "" && configPath == "" {
			cwd, err := os.Getwd()
			if err != nil {
				return nil // Silently exit on error
			}
			configFile, err := config.FindConfig(cwd)
			if err != nil || configFile == "" {
				// No config file found - silently exit
				return nil
			}
		}

		tmpl, err := template.New("prompt").Parse(agentPromptTemplate)
		if err != nil {
			return err
		}

		data := promptData{
			GraphQLSchema: GetGraphQLSchema(),
			Types:         config.DefaultTypes,
			Statuses:      primeConfig().Statuses(),
			Priorities:    config.DefaultPriorities,
		}

		return tmpl.Execute(os.Stdout, data)
	},
}

// primeConfig loads the project config for its custom statuses. The root
// command skips loading it for prime, which must not fail on a broken config,
// so this falls back to the defaults.
func primeConfig() *config.Config {
	var loaded *config.Config
	var err error
	if configPath != "" {
		loaded, err = config.Load(configPath)
	} else if cwd, cwdErr := os.Getwd(); cwdErr == nil {
		loaded, err = config.LoadFromDirectory(cwd)
	}
	if err != nil || loaded == nil {
		return config.Default()
	}
	return loaded
}

func init() {
	rootCmd.AddCommand(primeCmd)
}
