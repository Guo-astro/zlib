package cli

import (
	"fmt"
	"os"
	"strings"

	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
	zlib "github.com/heartleo/zlib"
	"github.com/mattn/go-runewidth"
	"github.com/spf13/cobra"
)

const defaultSearchPageSize = 50

var searchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search books by title, author, or ISBN",
	RunE: func(cmd *cobra.Command, args []string) error {
		jsonOut, _ := cmd.Flags().GetBool("json")

		if len(args) == 0 {
			if jsonOut {
				return fmt.Errorf("--json requires a search query (interactive mode has no JSON output)")
			}
			fullTitle, _ := cmd.Flags().GetBool("full-title")
			selectedID, err := interactiveSearch(nil, searchOptsFromFlags(cmd), fullTitle)
			if err != nil {
				return err
			}
			if selectedID == "" {
				return nil
			}
			dir, err := destDirFlag(cmd)
			if err != nil {
				return err
			}
			sendToKindle, _ := cmd.Flags().GetBool("send-to-kindle")

			return runDownload(newDownloadModel(selectedID, dir, newClient()), sendToKindle, cmd)
		}

		query := strings.Join(args, " ")
		page, _ := cmd.Flags().GetInt("page")
		count, _ := cmd.Flags().GetInt("count")
		fullTitle, _ := cmd.Flags().GetBool("full-title")
		opts := searchOptsFromFlags(cmd)

		result, err := newClient().Search(query, page, count, opts)
		if err != nil {
			return err
		}

		if jsonOut {
			// Keep references accepted by `zlib download` in JSON output.
			for i, b := range result.Books {
				if id := bookCommandID(b); id != "" {
					result.Books[i].ID = id
				}
			}
			return printJSON(result)
		}

		if len(result.Books) == 0 {
			lipgloss.Fprintf(os.Stdout, "%s No results found for %q\n", colorRed(symbolError), query)
			return nil
		}

		lipgloss.Fprintf(os.Stdout, "%s Found %s · Page %d / %d\n\n",
			colorGreen(symbolSuccess),
			colorBold(fmt.Sprintf("%d results", len(result.Books))),
			result.Page, result.TotalPages,
		)

		rows := make([][]string, 0, len(result.Books))
		for i, b := range result.Books {
			id := bookCommandID(b)
			if id == "" {
				id = "-"
			}
			authors := strings.Join(b.Authors, ", ")
			title := b.Name
			if !fullTitle {
				title = runewidth.Truncate(title, 40, "")
			}
			ext := strings.ToUpper(b.Extension)
			if ext == "" {
				ext = "-"
			}
			size := b.Size
			if size == "" {
				size = "-"
			}
			year := b.Year
			if year == "" || year == "0" {
				year = "-"
			}
			rating := b.Rating
			if rating == "" {
				rating = "-"
			}
			rows = append(rows, []string{
				fmt.Sprintf("%d", i+1), id,
				title,
				runewidth.Truncate(authors, 20, ""),
				year, ext, size, rating,
			})
		}

		t := table.New().
			Border(lipgloss.RoundedBorder()).
			BorderStyle(lipgloss.NewStyle().Foreground(currentTheme.Accent)).
			Headers("#", "ID", "Title", "Authors", "Year", "Format", "Size", "Rating").
			Rows(rows...).
			StyleFunc(func(row, col int) lipgloss.Style {
				pad := lipgloss.NewStyle().PaddingLeft(1).PaddingRight(1)
				if row == table.HeaderRow {
					return pad.Bold(true).Foreground(currentTheme.Accent)
				}
				switch col {
				case 1: // ID
					return pad.Foreground(currentTheme.Link)
				case 2: // Title
					return pad.Foreground(currentTheme.Title)
				case 5: // Format
					return pad.Foreground(extLipglossColor(rows[row][5]))
				case 7: // Rating
					return pad.Foreground(currentTheme.Success)
				case 0, 3, 4, 6: // #, Authors, Year, Size
					return pad.Foreground(currentTheme.Muted)
				}
				return pad
			})

		lipgloss.Fprintln(os.Stdout, t)
		return nil
	},
}

// bookIDFromURL extracts the alphanumeric book ID from a URL like
// https://z-library.sk/book/2RAqApzDRL/title.html
func bookIDFromURL(u string) string {
	parts := strings.Split(u, "/")
	for i, p := range parts {
		if p == "book" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return u
}

func bookCommandID(book zlib.Book) string {
	if book.ID != "" && book.Hash != "" {
		return book.ID + ":" + book.Hash
	}
	id := bookIDFromURL(book.URL)
	if id == book.URL {
		return ""
	}
	return id
}

func historyCommandID(item zlib.DownloadHistoryItem) string {
	if item.ID != "" && item.Hash != "" {
		return item.ID + ":" + item.Hash
	}
	id := bookIDFromURL(item.URL)
	if id == item.URL {
		return ""
	}
	return id
}

func searchOptsFromFlags(cmd *cobra.Command) *zlib.SearchOptions {
	exts, _ := cmd.Flags().GetStringArray("ext")
	formats, _ := cmd.Flags().GetStringArray("format")
	exts = append(exts, formats...)
	if len(exts) == 0 {
		return nil
	}
	seen := make(map[zlib.Extension]bool)
	var extensions []zlib.Extension
	for _, e := range exts {
		ext := zlib.Extension(strings.ToUpper(e))
		if seen[ext] {
			continue
		}
		seen[ext] = true
		extensions = append(extensions, ext)
	}
	return &zlib.SearchOptions{Extensions: extensions}
}

func init() {
	searchCmd.Flags().IntP("page", "p", 1, "Page number")
	searchCmd.Flags().IntP("count", "n", defaultSearchPageSize, "Results per page")
	addDestDirFlag(searchCmd)
	searchCmd.Flags().Bool("send-to-kindle", false, "Send the downloaded file to Kindle.")
	searchCmd.Flags().StringArray("ext", nil, "Filter by file extension (repeatable): epub, pdf, mobi, …")
	searchCmd.Flags().StringArray("format", nil, "Alias for --ext: filter by file format (repeatable)")
	searchCmd.Flags().Bool("full-title", false, "Do not truncate book titles in the results table")
	searchCmd.Flags().Bool("json", false, "Output results as JSON (requires a query)")
}
