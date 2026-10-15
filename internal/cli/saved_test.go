package cli

import (
	"testing"
)

func TestFavoritesIsOnlyCollectionCommand(t *testing.T) {
	if _, _, err := rootCmd.Find([]string{"saved"}); err == nil {
		t.Fatal("saved command should not be registered")
	}
	favorites, _, err := rootCmd.Find([]string{"favorites"})
	if err != nil {
		t.Fatal(err)
	}
	if favorites.Name() != "favorites" {
		t.Fatalf("command = %v", favorites)
	}
	command := favorites
	for _, flag := range []string{"page", "json", "dir", "send-to-kindle"} {
		if command.Flags().Lookup(flag) == nil {
			t.Errorf("%s is missing --%s", command.Name(), flag)
		}
	}
}
