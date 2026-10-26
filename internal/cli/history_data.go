package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/heartleo/zlib/internal/zlib"
)

func loadHistoryPage(ctx context.Context, c *zlib.Client, page int, formatFilter string) (zlib.DownloadHistoryResult, error) {
	result, err := c.DownloadHistoryContext(ctx, page)
	if err != nil {
		return zlib.DownloadHistoryResult{}, err
	}
	// Collect book IDs that need detail fetch for Extension/Size
	var missingIDs []string
	for _, item := range result.Items {
		if id := historyCommandID(item); id != "" && (item.Extension == "" || item.Size == "") {
			missingIDs = append(missingIDs, id)
		}
	}
	if len(missingIDs) > 0 {
		details, err := c.FetchBookDetailsContext(ctx, missingIDs)
		if err != nil {
			return zlib.DownloadHistoryResult{}, fmt.Errorf("failed to complete history metadata: %w", err)
		}
		for i, item := range result.Items {
			detail := details[historyCommandID(item)]
			if item.Extension == "" {
				result.Items[i].Extension = detail.Extension
			}
			if item.Size == "" {
				result.Items[i].Size = detail.Size
			}
		}
	}
	formatFilter = strings.TrimPrefix(strings.TrimSpace(formatFilter), ".")
	// Apply the same format filter as the table, and fill in the
	// alphanumeric ID that `zlib history --download` accepts.
	items := make([]zlib.DownloadHistoryItem, 0, len(result.Items))
	for _, item := range result.Items {
		if formatFilter != "" {
			if item.Extension == "" {
				return zlib.DownloadHistoryResult{}, fmt.Errorf("cannot filter history entry %q: file format is unknown", item.Name)
			}
			if !strings.EqualFold(item.Extension, formatFilter) {
				continue
			}
		}
		if id := historyCommandID(item); id != "" {
			item.ID = id
		}
		items = append(items, item)
	}
	result.Items = items
	return result, nil
}
