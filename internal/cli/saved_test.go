package cli

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestSavedAndFavoritesAreSeparateCommands(t *testing.T) {
	saved, _, err := rootCmd.Find([]string{"saved"})
	if err != nil {
		t.Fatal(err)
	}
	favorites, _, err := rootCmd.Find([]string{"favorites"})
	if err != nil {
		t.Fatal(err)
	}
	if saved == favorites || saved.Name() != "saved" || favorites.Name() != "favorites" {
		t.Fatalf("commands = %v, %v", saved, favorites)
	}
	for _, command := range []*cobra.Command{saved, favorites} {
		for _, flag := range []string{"page", "json", "dir", "send-to-kindle"} {
			if command.Flags().Lookup(flag) == nil {
				t.Errorf("%s is missing --%s", command.Name(), flag)
			}
		}
	}
}
