package cli

import (
	"slices"
	"testing"

	zlib "github.com/heartleo/zlib"
	"github.com/spf13/cobra"
)

func TestSearchFlagsMapToEAPIOptions(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().StringArray("ext", nil, "")
	cmd.Flags().StringArray("format", nil, "")
	cmd.Flags().StringArray("language", nil, "")
	cmd.Flags().Int("from-year", 0, "")
	cmd.Flags().Int("to-year", 0, "")
	cmd.Flags().Bool("exact", false, "")
	cmd.Flags().String("order", "", "")
	for flag, value := range map[string]string{"ext": "epub", "format": "PDF", "language": "Chinese", "from-year": "2001", "to-year": "2020", "exact": "true", "order": "bestmatch"} {
		if err := cmd.Flags().Set(flag, value); err != nil {
			t.Fatal(err)
		}
	}
	opts, err := searchOptsFromFlags(cmd)
	if err != nil {
		t.Fatal(err)
	}
	if opts.FromYear != 2001 || opts.ToYear != 2020 || !opts.Exact || opts.Order != zlib.OrderBestMatch ||
		!slices.Equal(opts.Languages, []zlib.Language{zlib.LangChinese}) ||
		!slices.Equal(opts.Extensions, []zlib.Extension{zlib.ExtEPUB, zlib.ExtPDF}) {
		t.Fatalf("options = %+v", opts)
	}
	if err := cmd.Flags().Set("order", "date_created"); err != nil {
		t.Fatal(err)
	}
	if _, err := searchOptsFromFlags(cmd); err == nil {
		t.Fatal("obsolete search order accepted")
	}
}
