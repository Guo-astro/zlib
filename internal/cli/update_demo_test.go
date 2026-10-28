package cli

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"
	"time"
)

// Opt-in terminal walkthrough of `zlib update` against a throttled local
// release; it never contacts GitHub and replaces only a temporary file.
//
//	go test -c ./internal/cli -o update-demo
//	ZLIB_UPDATE_DEMO=1 ./update-demo -test.run TestUpdateProgressDemo -test.v
//
// ZLIB_UPDATE_DEMO_SECONDS sets the download time (default 4).
func TestUpdateProgressDemo(t *testing.T) {
	if os.Getenv("ZLIB_UPDATE_DEMO") != "1" {
		t.Skip("manual terminal walkthrough; set ZLIB_UPDATE_DEMO=1")
	}
	seconds, err := strconv.ParseFloat(os.Getenv("ZLIB_UPDATE_DEMO_SECONDS"), 64)
	if err != nil || seconds <= 0 {
		seconds = 4
	}
	archive, binary, err := releaseArtifact("v0.0.13", runtime.GOOS, runtime.GOARCH)
	if err != nil {
		t.Skip(err)
	}
	// Random content does not compress, so the archive keeps a realistic size.
	content := make([]byte, 9<<20)
	_, _ = rand.Read(content)
	payload := testReleaseArchive(t, runtime.GOOS, binary, content)
	sum := sha256.Sum256(payload)
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/latest":
			time.Sleep(300 * time.Millisecond)
			_ = json.NewEncoder(w).Encode(map[string]any{"tag_name": "v0.0.13", "assets": []map[string]string{
				{"name": archive, "browser_download_url": server.URL + "/archive"},
				{"name": "checksums.txt", "browser_download_url": server.URL + "/checksums"},
			}})
		case "/checksums":
			fmt.Fprintf(w, "%s  %s\n", hex.EncodeToString(sum[:]), archive)
		case "/archive":
			w.Header().Set("Content-Length", strconv.Itoa(len(payload)))
			const chunks = 80
			step := len(payload)/chunks + 1
			for start := 0; start < len(payload); start += step {
				if _, err := w.Write(payload[start:min(len(payload), start+step)]); err != nil {
					return
				}
				w.(http.Flusher).Flush()
				time.Sleep(time.Duration(seconds * float64(time.Second) / chunks))
			}
		}
	}))
	defer server.Close()
	executable := filepath.Join(t.TempDir(), binary)
	if err := os.WriteFile(executable, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	updater := releaseUpdater{client: server.Client(), apiURL: server.URL + "/latest"}
	if err := runReleaseUpdate(t.Context(), os.Stdin, os.Stdout, isInteractive(), updater, executable, "v0.0.12"); err != nil {
		t.Fatal(err)
	}
}
