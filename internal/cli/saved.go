package cli

import (
	"charm.land/lipgloss/v2"

	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"
)

var savedCmd = newSavedBooksCommand("saved")
var favoritesCmd = newSavedBooksCommand("favorites")

func newSavedBooksCommand(name string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   name,
		Short: "Browse saved books",
		Args:  cobra.NoArgs,
		RunE:  runSavedBooks,
	}
	cmd.Flags().IntP("page", "p", 1, "Page number")
	cmd.Flags().Bool("json", false, "Output saved books as JSON")
	addDestDirFlag(cmd)
	cmd.Flags().Bool("send-to-kindle", false, "Send the selected download to Kindle")
	return cmd
}

func runSavedBooks(cmd *cobra.Command, args []string) error {
	c := newClient()
	jsonOut, _ := cmd.Flags().GetBool("json")
	if !jsonOut && !cmd.Flags().Changed("page") && isInteractive() {
		p := tea.NewProgram(newSavedBooksSelectModel(c.SavedBooks))
		result, err := p.Run()
		if err != nil {
			return err
		}
		selected := result.(searchSelectModel)
		if selected.err != nil {
			return selected.err
		}
		if selected.selectedID == "" {
			return nil
		}
		dir, err := destDirFlag(cmd)
		if err != nil {
			return err
		}
		sendToKindle, _ := cmd.Flags().GetBool("send-to-kindle")
		return runDownload(newDownloadModel(selected.selectedID, dir, c), sendToKindle, cmd)
	}

	page, _ := cmd.Flags().GetInt("page")
	books, err := c.SavedBooks(page)
	if err != nil {
		return err
	}
	if jsonOut {
		return printJSON(books)
	}
	if len(books.Books) == 0 {
		lipgloss.Fprintf(cmd.OutOrStdout(), "No saved books found on page %d.\n", books.Page)
		return nil
	}
	lipgloss.Fprintf(cmd.OutOrStdout(), "Saved books, page %d of %d\n\n", books.Page, books.TotalPages)
	printBookTable(cmd.OutOrStdout(), books.Books)
	return nil
}

func init() {
	rootCmd.AddCommand(savedCmd)
	rootCmd.AddCommand(favoritesCmd)
}
