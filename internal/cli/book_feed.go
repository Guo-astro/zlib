package cli

import (
	"fmt"
	"io"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
	zlib "github.com/heartleo/zlib"
	"github.com/mattn/go-runewidth"
	"github.com/spf13/cobra"
)

var recommendedCmd = newBookFeedCommand("recommended", "Browse recommended books", func(c *zlib.Client) ([]zlib.Book, error) {
	return c.RecommendedBooks()
})

var mostPopularCmd = newBookFeedCommand("most-popular", "Browse the most popular books", func(c *zlib.Client) ([]zlib.Book, error) {
	return c.MostPopularBooks()
})

func newBookFeedCommand(name, summary string, load func(*zlib.Client) ([]zlib.Book, error)) *cobra.Command {
	cmd := &cobra.Command{
		Use:   name,
		Short: summary,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			fetch := func() ([]zlib.Book, error) { return load(c) }
			jsonOut, _ := cmd.Flags().GetBool("json")
			if !jsonOut && isInteractive() {
				p := tea.NewProgram(newBookFeedSelectModel(summary, fetch))
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

			books, err := fetch()
			if err != nil {
				return err
			}
			if jsonOut {
				return printJSON(books)
			}
			if len(books) == 0 {
				lipgloss.Fprintln(cmd.OutOrStdout(), "No books found.")
				return nil
			}
			lipgloss.Fprintf(cmd.OutOrStdout(), "%s · %d books\n\n", strings.TrimPrefix(summary, "Browse "), len(books))
			printBookTable(cmd.OutOrStdout(), books)
			lipgloss.Fprintln(cmd.OutOrStdout(), "Use --json for full titles and metadata.")
			return nil
		},
	}
	cmd.Flags().Bool("json", false, "Output books as JSON")
	addDestDirFlag(cmd)
	cmd.Flags().Bool("send-to-kindle", false, "Send the selected download to Kindle")
	return cmd
}

func printBookTable(out io.Writer, books []zlib.Book) {
	rows := make([][]string, 0, len(books))
	for i, book := range books {
		rows = append(rows, []string{
			fmt.Sprintf("%d", i+1), bookCommandID(book),
			runewidth.Truncate(bookCell(book.Name), 50, "…"),
			runewidth.Truncate(bookCell(strings.Join(book.Authors, ", ")), 24, "…"),
		})
	}
	t := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(currentTheme.Accent)).
		StyleFunc(func(row, col int) lipgloss.Style { return lipgloss.NewStyle().Padding(0, 1) }).
		Headers("#", "ID", "Title", "Author").
		Rows(rows...)
	lipgloss.Fprintln(out, t)
}
