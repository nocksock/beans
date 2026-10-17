package cmd

import (
	"context"
	_ "embed"
	"fmt"
	"strings"
	"text/template"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/hmans/beans/internal/bean"
	"github.com/hmans/beans/internal/graph"
	"github.com/hmans/beans/internal/output"
	"github.com/hmans/beans/internal/ui"
	"github.com/spf13/cobra"
)

//go:embed show_prompt.tmpl
var beanPromptTemplate string

var (
	showJSON     bool
	showRaw      bool
	showBodyOnly bool
	showPrompt   bool
)

var showCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show a bean's contents",
	Long:  `Displays the full contents of a bean, including front matter and body.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Query bean via GraphQL resolver
		resolver := &graph.Resolver{Core: core}
		b, err := resolver.Query().Bean(context.Background(), args[0])

		// Handle query errors (including ambiguous IDs)
		if err != nil {
			return handleBeanQueryError(err, showJSON, args[0])
		}

		if b == nil {
			if showJSON {
				return output.Error(output.ErrNotFound, "bean not found")
			}
			return fmt.Errorf("bean not found: %s", args[0])
		}

		// JSON output
		if showJSON {
			return output.SuccessSingle(b)
		}

		// Raw markdown output (frontmatter + body)
		if showRaw {
			content, err := b.Render()
			if err != nil {
				return fmt.Errorf("failed to render bean: %w", err)
			}
			fmt.Print(string(content))
			return nil
		}

		// Agent prompt (bean content, related beans, progress guide)
		if showPrompt {
			prompt, err := renderBeanPrompt(b, cfg.PromptExtra())
			if err != nil {
				return err
			}
			fmt.Print(prompt)
			return nil
		}

		// Body only (no header, no styling)
		if showBodyOnly {
			fmt.Print(b.Body)
			return nil
		}

		// Default: styled human-friendly output
		statusCfg := cfg.GetStatus(b.Status)
		statusColor := "gray"
		if statusCfg != nil {
			statusColor = statusCfg.Color
		}
		isArchive := cfg.IsArchiveStatus(b.Status)

		var header strings.Builder
		header.WriteString(ui.ID.Render(b.ID))
		header.WriteString(" ")
		header.WriteString(ui.RenderStatusWithColor(b.Status, statusColor, isArchive))
		if b.Priority != "" {
			priorityCfg := cfg.GetPriority(b.Priority)
			priorityColor := "gray"
			if priorityCfg != nil {
				priorityColor = priorityCfg.Color
			}
			header.WriteString(" ")
			header.WriteString(ui.RenderPriorityWithColor(b.Priority, priorityColor))
		}
		if len(b.Tags) > 0 {
			header.WriteString("  ")
			header.WriteString(ui.Muted.Render(strings.Join(b.Tags, ", ")))
		}
		header.WriteString("\n")
		header.WriteString(ui.Title.Render(b.DisplayTitle()))

		// Display relationships
		if b.Parent != "" || len(b.Blocking) > 0 || len(b.BlockedBy) > 0 {
			header.WriteString("\n")
			header.WriteString(ui.Muted.Render(strings.Repeat("─", 50)))
			header.WriteString("\n")
			header.WriteString(formatRelationships(b))
		}

		header.WriteString("\n")
		header.WriteString(ui.Muted.Render(strings.Repeat("─", 50)))

		headerBox := lipgloss.NewStyle().
			MarginBottom(1).
			Render(header.String())

		fmt.Println(headerBox)

		// Render the body with Glamour
		if b.Body != "" {
			renderer, err := glamour.NewTermRenderer(
				glamour.WithAutoStyle(),
				glamour.WithWordWrap(80),
			)
			if err != nil {
				return fmt.Errorf("failed to create renderer: %w", err)
			}

			rendered, err := renderer.Render(b.Body)
			if err != nil {
				return fmt.Errorf("failed to render markdown: %w", err)
			}

			fmt.Print(rendered)
		}

		return nil
	},
}

// formatRelationships formats parent and blocks for display.
func formatRelationships(b *bean.Bean) string {
	var parts []string

	// Display parent
	if b.Parent != "" {
		parts = append(parts, fmt.Sprintf("%s %s",
			ui.Muted.Render("parent:"),
			ui.ID.Render(b.Parent)))
	}

	// Display blocking
	for _, target := range b.Blocking {
		parts = append(parts, fmt.Sprintf("%s %s",
			ui.Muted.Render("blocking:"),
			ui.ID.Render(target)))
	}

	// Display blocked-by
	for _, blocker := range b.BlockedBy {
		parts = append(parts, fmt.Sprintf("%s %s",
			ui.Muted.Render("blocked by:"),
			ui.ID.Render(blocker)))
	}
	return strings.Join(parts, "\n")
}

// relatedBean is a single entry in the related beans section of a bean prompt.
type relatedBean struct {
	ID       string
	Relation string
	Status   string
	Title    string
}

// beanPromptData holds all data needed to render the bean prompt template.
type beanPromptData struct {
	Bean    *bean.Bean
	Related []relatedBean
	Extra   string
}

// renderBeanPrompt renders a prompt for an AI agent to start working on a bean.
func renderBeanPrompt(b *bean.Bean, extra string) (string, error) {
	related, err := collectRelatedBeans(b)
	if err != nil {
		return "", err
	}

	tmpl, err := template.New("bean-prompt").
		Funcs(template.FuncMap{"join": strings.Join, "trim": strings.TrimSpace}).
		Parse(beanPromptTemplate)
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	data := beanPromptData{Bean: b, Related: related, Extra: strings.TrimSpace(extra)}
	if err := tmpl.Execute(&sb, data); err != nil {
		return "", fmt.Errorf("failed to render prompt: %w", err)
	}
	return sb.String(), nil
}

// collectRelatedBeans resolves the parent, blocking, blocked-by and child beans via GraphQL.
func collectRelatedBeans(b *bean.Bean) ([]relatedBean, error) {
	ctx := context.Background()
	resolver := (&graph.Resolver{Core: core}).Bean()

	var related []relatedBean
	add := func(relation string, beans ...*bean.Bean) {
		for _, r := range beans {
			if r != nil {
				related = append(related, relatedBean{ID: r.ID, Relation: relation, Status: r.Status, Title: r.DisplayTitle()})
			}
		}
	}

	parent, err := resolver.Parent(ctx, b)
	if err != nil {
		return nil, err
	}
	add("parent", parent)

	blocking, err := resolver.Blocking(ctx, b, nil)
	if err != nil {
		return nil, err
	}
	add("blocking", blocking...)

	blockedBy, err := resolver.BlockedBy(ctx, b, nil)
	if err != nil {
		return nil, err
	}
	add("blocked by", blockedBy...)

	children, err := resolver.Children(ctx, b, nil)
	if err != nil {
		return nil, err
	}
	add("child", children...)

	return related, nil
}

func init() {
	showCmd.Flags().BoolVar(&showJSON, "json", false, "Output as JSON")
	showCmd.Flags().BoolVar(&showRaw, "raw", false, "Output raw markdown without styling")
	showCmd.Flags().BoolVar(&showBodyOnly, "body-only", false, "Output only the body content")
	showCmd.Flags().BoolVar(&showPrompt, "prompt", false, "Output a prompt for an AI agent to start working on the bean")
	showCmd.MarkFlagsMutuallyExclusive("json", "raw", "body-only", "prompt")
	rootCmd.AddCommand(showCmd)
}
