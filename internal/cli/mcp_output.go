package cli

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"regexp"
	"strings"
	"time"

	zlib "github.com/heartleo/zlib/internal/zlib"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// mcpBook is the compact book an agent needs to choose and download a result.
// Covers, page URLs and EAPI download paths are left out to save tokens.
type mcpBook struct {
	ID        string   `json:"id" jsonschema:"Book reference in id:hash format; pass it to get_book or download_book."`
	Name      string   `json:"name"`
	Authors   []string `json:"authors,omitempty"`
	Publisher string   `json:"publisher,omitempty"`
	Year      string   `json:"year,omitempty"`
	Language  string   `json:"language,omitempty"`
	Extension string   `json:"extension,omitempty"`
	Size      string   `json:"size,omitempty"`
	Rating    string   `json:"rating,omitempty"`
}

type mcpBookDetails struct {
	mcpBook
	ISBN        string `json:"isbn,omitempty"`
	ISBN10      string `json:"isbn_10,omitempty"`
	ISBN13      string `json:"isbn_13,omitempty"`
	Edition     string `json:"edition,omitempty"`
	Pages       string `json:"pages,omitempty"`
	Categories  string `json:"categories,omitempty"`
	Description string `json:"description,omitempty"`
}

type mcpBookPage struct {
	Books      []mcpBook `json:"books" jsonschema:"Books on this page."`
	Page       int       `json:"page"`
	TotalPages int       `json:"total_pages"`
}

type mcpHistoryItem struct {
	ID        string   `json:"id" jsonschema:"Book reference in id:hash format."`
	Name      string   `json:"name"`
	Authors   []string `json:"authors,omitempty"`
	Extension string   `json:"extension,omitempty"`
	Size      string   `json:"size,omitempty"`
	Date      string   `json:"date,omitempty"`
}

type mcpHistoryPage struct {
	Items      []mcpHistoryItem `json:"items" jsonschema:"Download history entries on this page."`
	Page       int              `json:"page"`
	TotalPages int              `json:"total_pages"`
}

func compactBook(b zlib.Book) mcpBook {
	id := bookCommandID(b)
	if id == "" {
		id = b.ID
	}
	year := b.Year
	if year == "0" {
		year = ""
	}
	return mcpBook{
		ID:        id,
		Name:      b.Name,
		Authors:   b.Authors,
		Publisher: b.Publisher,
		Year:      year,
		Language:  b.Language,
		Extension: b.Extension,
		Size:      b.Size,
		Rating:    b.Rating,
	}
}

// compactBooks returns an empty slice rather than nil, so JSON carries [].
func compactBooks(books []zlib.Book) []mcpBook {
	out := make([]mcpBook, 0, len(books))
	for _, b := range books {
		out = append(out, compactBook(b))
	}
	return out
}

func compactBookPage(result zlib.SearchResult) mcpBookPage {
	return mcpBookPage{Books: compactBooks(result.Books), Page: result.Page, TotalPages: result.TotalPages}
}

func compactBookDetails(b zlib.Book) mcpBookDetails {
	return mcpBookDetails{
		mcpBook:     compactBook(b),
		ISBN:        b.ISBN,
		ISBN10:      b.ISBN10,
		ISBN13:      b.ISBN13,
		Edition:     b.Edition,
		Pages:       b.Pages,
		Categories:  b.Categories,
		Description: b.Description,
	}
}

func compactHistoryPage(result zlib.DownloadHistoryResult) mcpHistoryPage {
	items := make([]mcpHistoryItem, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, mcpHistoryItem{
			ID:        item.ID,
			Name:      item.Name,
			Authors:   item.Authors,
			Extension: item.Extension,
			Size:      item.Size,
			Date:      item.Date,
		})
	}
	return mcpHistoryPage{Items: items, Page: result.Page, TotalPages: result.TotalPages}
}

var urlInText = regexp.MustCompile(`[A-Za-z][A-Za-z0-9+.-]*://[^\s"'<>]+`)

// redactURLs reduces every URL in s to its origin. Download errors can carry
// signed CDN links and proxy credentials; the host is enough to diagnose them.
func redactURLs(s string) string {
	return urlInText.ReplaceAllStringFunc(s, func(raw string) string {
		// Sentence punctuation after a URL ("dial http://host:8080: refused")
		// is not part of it and would make the port unparseable.
		trimmed := strings.TrimRight(raw, ".,;:)]}")
		u, err := url.Parse(trimmed)
		if err != nil || u.Host == "" {
			return "[URL hidden]"
		}
		return u.Scheme + "://" + u.Host + raw[len(trimmed):]
	})
}

// argumentConstraints words each constrained input for an agent. The SDK's
// schema errors ("maximum: 99/1 is greater than 50.000000") are accurate but
// hard to act on.
func argumentConstraints() map[string]string {
	orders := make([]string, 0, len(searchSorts))
	for _, choice := range searchSorts {
		if choice.value != "" {
			orders = append(orders, string(choice.value))
		}
	}
	years := "must be 0 (unset) or a year from 1000 to 9999"
	return map[string]string{
		"query":     "must not be empty",
		"id":        "must not be empty (expected id:hash)",
		"file_path": "must not be empty",
		"page":      "must be 1 or greater",
		"count":     "must be between 1 and 50",
		"from_year": years,
		"to_year":   years,
		"order":     "must be one of: " + strings.Join(orders, ", "),
	}
}

var argumentErrorProperty = regexp.MustCompile(`validating /properties/([A-Za-z_]+): `)

const sdkArgumentErrorPrefix = `validating "arguments": `

func friendlyArgumentError(text string) (string, bool) {
	if !strings.HasPrefix(text, sdkArgumentErrorPrefix) {
		return "", false
	}
	if m := argumentErrorProperty.FindStringSubmatch(text); m != nil {
		if constraint, ok := argumentConstraints()[m[1]]; ok {
			return fmt.Sprintf("invalid argument %q: %s", m[1], constraint), true
		}
	}
	return "invalid arguments: " + strings.TrimPrefix(text, sdkArgumentErrorPrefix), true
}

// mcpCallMiddleware rewrites schema validation errors into actionable text and
// logs each tool call (name, duration, outcome; never arguments) to the logger.
func mcpCallMiddleware(logger *slog.Logger) mcp.Middleware {
	return func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
			start := time.Now()
			res, err := next(ctx, method, req)
			if method != "tools/call" {
				return res, err
			}
			name := ""
			if params, ok := req.GetParams().(*mcp.CallToolParamsRaw); ok && params != nil {
				name = params.Name
			}
			attrs := []any{"tool", name, "duration", time.Since(start).Round(time.Millisecond)}
			result, _ := res.(*mcp.CallToolResult)
			switch {
			case err != nil:
				logger.Warn("tool call failed", append(attrs, "error", redactURLs(safeText(err.Error())))...)
			case result != nil && result.IsError:
				if len(result.Content) > 0 {
					if text, ok := result.Content[0].(*mcp.TextContent); ok {
						if msg, ok := friendlyArgumentError(text.Text); ok {
							text.Text = msg
						}
						attrs = append(attrs, "error", text.Text)
					}
				}
				logger.Info("tool returned an error", attrs...)
			default:
				logger.Debug("tool call", attrs...)
			}
			return res, err
		}
	}
}
