package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/hmans/beans/internal/config"
	"github.com/hmans/beans/internal/graph"
	"github.com/hmans/beans/internal/graph/model"
	"github.com/hmans/beans/internal/output"
	"github.com/hmans/beans/internal/ui"
	"github.com/spf13/cobra"
)

var (
	createStatus    string
	createType      string
	createPriority  string
	createSize      string
	createBody      string
	createBodyFile  string
	createTag       []string
	createParent    string
	createBlocking  []string
	createBlockedBy []string
	createJSON      bool
)

var createCmd = &cobra.Command{
	Use:     "create [title]",
	Aliases: []string{"c", "new"},
	Short:   "Create a new bean",
	Long:    `Creates a new bean (issue) with a generated ID and optional title.

The title may be omitted when a body is given; untitled beans are displayed
using the first line of their body.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		title := strings.TrimSpace(strings.Join(args, " "))

		// Validate inputs
		if createStatus != "" && !cfg.IsValidStatus(createStatus) {
			return cmdError(createJSON, output.ErrInvalidStatus, "invalid status: %s (must be %s)", createStatus, cfg.StatusList())
		}
		if createType != "" && !cfg.IsValidType(createType) {
			return cmdError(createJSON, output.ErrValidation, "invalid type: %s (must be %s)", createType, cfg.TypeList())
		}
		if createPriority != "" && !cfg.IsValidPriority(createPriority) {
			return cmdError(createJSON, output.ErrValidation, "invalid priority: %s (must be %s)", createPriority, cfg.PriorityList())
		}
		if !cfg.IsValidSize(createSize) {
			return cmdError(createJSON, output.ErrValidation, "invalid size: %s (must be %s)", createSize, cfg.SizeList())
		}

		body, err := resolveContent(createBody, createBodyFile)
		if err != nil {
			return cmdError(createJSON, output.ErrFileError, "%s", err)
		}
		if title == "" && strings.TrimSpace(body) == "" {
			return cmdError(createJSON, output.ErrValidation, "a title or a body is required")
		}

		// Build GraphQL input
		input := model.CreateBeanInput{}
		if title != "" {
			input.Title = &title
		}
		if createStatus != "" {
			input.Status = &createStatus
		} else {
			defaultStatus := cfg.GetDefaultStatus()
			input.Status = &defaultStatus
		}
		if createType != "" {
			input.Type = &createType
		} else {
			defaultType := cfg.GetDefaultType()
			input.Type = &defaultType
		}
		if createPriority != "" {
			input.Priority = &createPriority
		}
		if createSize != "" {
			input.Size = &createSize
		}
		if body != "" {
			input.Body = &body
		}
		if len(createTag) > 0 {
			input.Tags = createTag
		}

		// Add parent
		if createParent != "" {
			input.Parent = &createParent
		}

		// Add blocking
		if len(createBlocking) > 0 {
			input.Blocking = createBlocking
		}

		// Add blocked-by
		if len(createBlockedBy) > 0 {
			input.BlockedBy = createBlockedBy
		}

		// Create via GraphQL mutation
		resolver := &graph.Resolver{Core: core}
		b, err := resolver.Mutation().CreateBean(context.Background(), input)
		if err != nil {
			return cmdError(createJSON, output.ErrFileError, "failed to create bean: %v", err)
		}

		if createJSON {
			return output.Success(b, "Bean created")
		}

		fmt.Println(ui.Success.Render("Created ") + ui.ID.Render(b.ID) + " " + ui.Muted.Render(b.Path))
		return nil
	},
}

func init() {
	// Build help text with allowed values from hardcoded config
	statusNames := make([]string, len(config.DefaultStatuses))
	for i, s := range config.DefaultStatuses {
		statusNames[i] = s.Name
	}
	typeNames := make([]string, len(config.DefaultTypes))
	for i, t := range config.DefaultTypes {
		typeNames[i] = t.Name
	}
	priorityNames := make([]string, len(config.DefaultPriorities))
	for i, p := range config.DefaultPriorities {
		priorityNames[i] = p.Name
	}

	createCmd.Flags().StringVarP(&createStatus, "status", "s", "", "Initial status ("+strings.Join(statusNames, ", ")+", or a custom status from .beans.yml)")
	createCmd.Flags().StringVarP(&createType, "type", "t", "", "Bean type ("+strings.Join(typeNames, ", ")+")")
	createCmd.Flags().StringVarP(&createPriority, "priority", "p", "", "Priority level ("+strings.Join(priorityNames, ", ")+")")
	createCmd.Flags().StringVar(&createSize, "size", "", "Size ("+strings.Join(sizeNames(), ", ")+"; default "+config.DefaultSize+")")
	createCmd.Flags().StringVarP(&createBody, "body", "d", "", "Body content (use '-' to read from stdin)")
	createCmd.Flags().StringVar(&createBodyFile, "body-file", "", "Read body from file")
	createCmd.Flags().StringArrayVar(&createTag, "tag", nil, "Add tag (can be repeated)")
	createCmd.Flags().StringVar(&createParent, "parent", "", "Parent bean ID")
	createCmd.Flags().StringArrayVar(&createBlocking, "blocking", nil, "ID of bean this blocks (can be repeated)")
	createCmd.Flags().StringArrayVar(&createBlockedBy, "blocked-by", nil, "ID of bean that blocks this one (can be repeated)")
	createCmd.Flags().BoolVar(&createJSON, "json", false, "Output as JSON")
	createCmd.MarkFlagsMutuallyExclusive("body", "body-file")
	rootCmd.AddCommand(createCmd)
}

// sizeNames returns the hardcoded size names for flag help texts, which are
// built before the project config is loaded.
func sizeNames() []string {
	names := make([]string, len(config.DefaultSizes))
	for i, s := range config.DefaultSizes {
		names[i] = s.Name
	}
	return names
}
