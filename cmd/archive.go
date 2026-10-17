package cmd

import (
	"context"
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/hmans/beans/internal/bean"
	"github.com/hmans/beans/internal/config"
	"github.com/hmans/beans/internal/graph"
	"github.com/hmans/beans/internal/graph/model"
	"github.com/hmans/beans/internal/output"
	"github.com/hmans/beans/internal/ui"
	"github.com/spf13/cobra"
)

var (
	archiveForce bool
	archiveJSON  bool
)

var archiveCmd = &cobra.Command{
	Use:   "archive",
	Short: "Move completed and scrapped beans to the archived status",
	Long: `Sets the status of all "completed" and "scrapped" beans to "archived". Archived beans are kept
(including their relationships) but hidden from "beans list" and the TUI by default.
Asks for confirmation unless --force is provided.

Use "beans list --include archived" to see archived beans, and "beans delete" to remove beans for good.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		resolver := &graph.Resolver{Core: core}

		allBeans, err := resolver.Query().Beans(ctx, nil)
		if err != nil {
			return err
		}

		// Find done beans that are not archived yet
		var archiveBeans []*bean.Bean
		for _, b := range allBeans {
			if cfg.IsArchivable(b.Status) {
				archiveBeans = append(archiveBeans, b)
			}
		}

		if len(archiveBeans) == 0 {
			if archiveJSON {
				return output.SuccessMessage("No beans to archive")
			}
			fmt.Println("No completed or scrapped beans to archive.")
			return nil
		}

		// Sort beans for consistent display
		bean.SortByStatusPriorityAndType(archiveBeans, cfg.StatusNames(), cfg.PriorityNames(), cfg.TypeNames())

		// JSON implies force (no prompts for machines)
		if !archiveForce && !archiveJSON {
			fmt.Printf("Beans to archive (%d):\n\n", len(archiveBeans))
			printBeanList(archiveBeans)
			fmt.Println()

			var confirm bool
			err := huh.NewConfirm().
				Title(fmt.Sprintf("Archive %d bean(s)?", len(archiveBeans))).
				Affirmative("Yes").
				Negative("No").
				Value(&confirm).
				Run()

			if err != nil {
				return err
			}

			if !confirm {
				fmt.Println("Cancelled")
				return nil
			}
		}

		status := config.StatusArchived
		for _, b := range archiveBeans {
			if _, err := resolver.Mutation().UpdateBean(ctx, b.ID, model.UpdateBeanInput{Status: &status}); err != nil {
				if archiveJSON {
					return output.Error(output.ErrFileError, fmt.Sprintf("failed to archive bean %s: %s", b.ID, err))
				}
				return fmt.Errorf("failed to archive bean %s: %w", b.ID, err)
			}
		}

		if archiveJSON {
			return output.SuccessMessage(fmt.Sprintf("Archived %d bean(s)", len(archiveBeans)))
		}
		fmt.Printf("Archived %d bean(s)\n", len(archiveBeans))
		return nil
	},
}

// printBeanList prints a formatted list of beans
func printBeanList(beans []*bean.Bean) {
	// Calculate max ID width
	maxIDWidth := 0
	for _, b := range beans {
		if len(b.ID) > maxIDWidth {
			maxIDWidth = len(b.ID)
		}
	}
	maxIDWidth += 2 // padding

	// Check if any beans have tags
	hasTags := false
	for _, b := range beans {
		if len(b.Tags) > 0 {
			hasTags = true
			break
		}
	}

	// Print each bean
	for _, b := range beans {
		colors := cfg.GetBeanColors(b.Status, b.Type, b.Priority)
		row := ui.RenderBeanRow(b.ID, b.Status, b.Type, b.DisplayTitle(), ui.BeanRowConfig{
			StatusColor:   colors.StatusColor,
			TypeColor:     colors.TypeColor,
			PriorityColor: colors.PriorityColor,
			Priority:      b.Priority,
			IsArchive:     colors.IsArchive,
			MaxTitleWidth: 60,
			ShowCursor:    false,
			IsSelected:    false,
			Tags:          b.Tags,
			ShowTags:      hasTags,
			IDColWidth:    maxIDWidth,
		})
		fmt.Println(row)
	}
}

func init() {
	archiveCmd.Flags().BoolVarP(&archiveForce, "force", "f", false, "Skip confirmation")
	archiveCmd.Flags().BoolVar(&archiveJSON, "json", false, "Output as JSON (implies --force)")
	rootCmd.AddCommand(archiveCmd)
}
