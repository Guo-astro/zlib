package cli

import (
	"testing"

	zlib "github.com/heartleo/zlib"
)

// favorites, recommended and most-popular --json used to print the bare
// numeric ID, which `zlib download` rejects; every JSON book list must carry
// the id:hash reference.
func TestDownloadableBooksUseIDHash(t *testing.T) {
	books := downloadableBooks([]zlib.Book{
		{ID: "123", Hash: "abc"},
		{ID: "456"},
	})
	if books[0].ID != "123:abc" {
		t.Fatalf("ID = %q, want 123:abc", books[0].ID)
	}
	if books[1].ID != "456" {
		t.Fatalf("book without a hash changed to %q", books[1].ID)
	}
}

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
