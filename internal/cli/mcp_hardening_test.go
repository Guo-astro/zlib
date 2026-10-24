package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/heartleo/zlib"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestMCPErrorsRedactURLPathsAndCredentials(t *testing.T) {
	for _, err := range []error{
		&zlib.HTTPStatusError{StatusCode: 403, URL: "https://cdn.example/files/42?token=signed-fixture"},
		&zlib.BotProtectionError{StatusCode: 513, URL: "https://mirror.example/eapi/book/42?token=signed-fixture"},
		&url.Error{Op: "Get", URL: "https://cdn.example/files/42?token=signed-fixture", Err: errors.New("x509: certificate signed by unknown authority")},
		errors.New(`proxyconnect tcp: dial http://fixture-user:fixture-pass@proxy.example:8080: refused`),
	} {
		got := formatMCPErrorText(err)
		for _, secret := range []string{"signed-fixture", "/files/42", "fixture-pass", "fixture-user"} {
			if strings.Contains(got, secret) {
				t.Errorf("%T leaked %q: %s", err, secret, got)
			}
		}
		if !strings.Contains(got, ".example") {
			t.Errorf("%T lost the host needed for diagnosis: %s", err, got)
		}
	}
}

func TestMCPCheckDomainsRejectsAgentProxy(t *testing.T) {
	withDemoMCPFixtures(t)
	var probes []string
	previous := probeAvailableDomains
	probeAvailableDomains = func(ctx context.Context, domains []string, proxy string) []zlib.DomainCheck {
		probes = append(probes, proxy)
		return previous(ctx, domains, proxy)
	}
	t.Cleanup(func() { probeAvailableDomains = previous })
	session := connectMCPTestSession(t, mcpServerOptions{}, nil)

	tools, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, tool := range tools.Tools {
		if tool.Name == "check_domains" {
			raw, _ := json.Marshal(tool.InputSchema)
			if strings.Contains(string(raw), "proxy") {
				t.Fatalf("check_domains still advertises a proxy argument: %s", raw)
			}
		}
	}
	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "check_domains",
		Arguments: map[string]any{"proxy": "http://attacker.example:8080"},
	})
	if err != nil || !res.IsError {
		t.Fatalf("agent-supplied proxy accepted: err=%v res=%+v", err, res)
	}
	if len(probes) != 0 {
		t.Fatalf("probe ran with rejected arguments: %v", probes)
	}
	if res, err = session.CallTool(context.Background(), &mcp.CallToolParams{Name: "check_domains", Arguments: map[string]any{}}); err != nil || res.IsError {
		t.Fatalf("check_domains failed: err=%v res=%+v", err, res)
	}
	if len(probes) != 1 || probes[0] != "" {
		t.Fatalf("probe proxy = %q, want empty so ZLIB_PROXY applies", probes)
	}
}

func TestMCPAllowedDirConfinesWrites(t *testing.T) {
	withDemoMCPFixtures(t)
	t.Setenv(downloadDirEnvVar, "")
	root, err := resolveAllowedDir(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	session := connectMCPTestSession(t, mcpServerOptions{AllowedDir: root}, nil)
	call := func(name string, args map[string]any) *mcp.CallToolResult {
		t.Helper()
		res, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: args})
		if err != nil {
			t.Fatal(err)
		}
		return res
	}

	escaped := filepath.Join(outside, "escaped")
	if res := call("download_book", map[string]any{"id": "115066162:e9f13b", "dir": escaped}); !res.IsError || !strings.Contains(toolText(t, res), "outside the allowed directory") {
		t.Fatalf("download outside the allowed directory was not rejected: %+v", res)
	}
	if _, err := os.Stat(escaped); !os.IsNotExist(err) {
		t.Fatalf("rejected download created %s", escaped)
	}

	res := call("download_book", map[string]any{"id": "115066162:e9f13b"})
	if res.IsError {
		t.Fatalf("default download failed: %s", toolText(t, res))
	}
	var saved downloadResult
	decodeToolJSON(t, res, &saved)
	if rel, err := filepath.Rel(root, saved.Path); err != nil || strings.HasPrefix(rel, "..") {
		t.Fatalf("default download saved outside the allowed directory: %s", saved.Path)
	}

	// A ZLIB_DOWNLOAD_DIR outside the root is skipped, not an error the agent
	// cannot fix; one inside the root is honoured.
	t.Setenv(downloadDirEnvVar, filepath.Join(outside, "env"))
	if res := call("download_book", map[string]any{"id": "27462851:4c0a7d"}); res.IsError {
		t.Fatalf("download with ZLIB_DOWNLOAD_DIR outside the root failed: %s", toolText(t, res))
	}
	if _, err := os.Stat(filepath.Join(outside, "env")); !os.IsNotExist(err) {
		t.Fatal("ZLIB_DOWNLOAD_DIR outside the root was created")
	}
	inside := filepath.Join(root, "env")
	t.Setenv(downloadDirEnvVar, inside)
	res = call("download_book", map[string]any{"id": "38104421:b81e22"})
	if res.IsError {
		t.Fatalf("download with ZLIB_DOWNLOAD_DIR inside the root failed: %s", toolText(t, res))
	}
	decodeToolJSON(t, res, &saved)
	if filepath.Dir(saved.Path) != inside {
		t.Fatalf("ZLIB_DOWNLOAD_DIR inside the root ignored: %s", saved.Path)
	}
	t.Setenv(downloadDirEnvVar, "")

	book := filepath.Join(outside, "book.epub")
	if err := os.WriteFile(book, []byte("fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	if res := call("send_to_kindle", map[string]any{"file_path": book}); !res.IsError || !strings.Contains(toolText(t, res), "outside the allowed directory") {
		t.Fatalf("Kindle attachment outside the allowed directory was not rejected: %+v", res)
	}

	link := filepath.Join(root, "link")
	makeDirLink(t, outside, link)
	if res := call("download_book", map[string]any{"id": "115066162:e9f13b", "dir": filepath.Join(link, "sub")}); !res.IsError {
		t.Fatal("download through a link that leaves the allowed directory was accepted")
	}
	if _, err := os.Stat(filepath.Join(outside, "sub")); !os.IsNotExist(err) {
		t.Fatal("rejected link download created a directory outside the root")
	}
	if res := call("send_to_kindle", map[string]any{"file_path": filepath.Join(link, "book.epub")}); !res.IsError || !strings.Contains(toolText(t, res), "allowed directory") {
		t.Fatalf("Kindle attachment reached through a link was not rejected: %+v", res)
	}
}

// makeDirLink links link to target with a symlink, or on Windows without the
// symlink privilege, a junction (which needs no privilege).
func makeDirLink(t *testing.T, target, link string) {
	t.Helper()
	symErr := os.Symlink(target, link)
	if symErr == nil {
		return
	}
	if runtime.GOOS == "windows" {
		out, err := exec.Command("cmd", "/c", "mklink", "/J", link, target).CombinedOutput()
		if err == nil {
			return
		}
		t.Skipf("cannot create a directory link: symlink: %v; junction: %v %s", symErr, err, out)
	}
	t.Skipf("symlinks unavailable: %v", symErr)
}

func TestResolveAllowedDirRequiresDirectory(t *testing.T) {
	file := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	for _, raw := range []string{file, filepath.Join(t.TempDir(), "missing")} {
		if _, err := resolveAllowedDir(raw); err == nil {
			t.Errorf("resolveAllowedDir(%q) accepted a non-directory", raw)
		}
	}
}

var idHashRef = regexp.MustCompile(`"id":"[0-9]+:[0-9a-f]+"`)

func TestMCPBookOutputIsCompact(t *testing.T) {
	withDemoMCPFixtures(t)
	session := connectMCPTestSession(t, mcpServerOptions{}, nil)
	for _, tc := range []struct {
		name string
		args map[string]any
	}{
		{"search_books", map[string]any{"query": "Dune", "count": 3}},
		{"get_saved_books", map[string]any{}},
		{"get_recommended_books", map[string]any{}},
		{"get_download_history", map[string]any{}},
		{"get_book", map[string]any{"id": "115066162:e9f13b"}},
	} {
		res, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: tc.name, Arguments: tc.args})
		if err != nil || res.IsError {
			t.Fatalf("%s failed: err=%v res=%+v", tc.name, err, res)
		}
		text := toolText(t, res)
		for _, field := range []string{`"download_url"`, `"cover"`, `"url"`, `"hash"`} {
			if strings.Contains(text, field) {
				t.Errorf("%s output still carries %s: %s", tc.name, field, text)
			}
		}
		if !idHashRef.MatchString(text) {
			t.Errorf("%s output lost the id:hash reference: %s", tc.name, text)
		}
		if tc.name == "get_book" && !strings.Contains(text, `"description"`) {
			t.Errorf("get_book dropped the description: %s", text)
		}
	}
}

func TestMCPArgumentErrorsAreActionable(t *testing.T) {
	session := connectMCPTestSession(t, mcpServerOptions{}, nil)
	for _, tc := range []struct {
		args map[string]any
		want string
	}{
		{map[string]any{"query": "Dune", "count": 99}, `invalid argument "count": must be between 1 and 50`},
		{map[string]any{"query": "Dune", "page": 0}, `invalid argument "page": must be 1 or greater`},
		{map[string]any{"query": "Dune", "from_year": 99}, `invalid argument "from_year": must be 0 (unset) or a year from 1000 to 9999`},
		{map[string]any{"query": "Dune", "order": "bogus"}, `invalid argument "order": must be one of: popular, bestmatch`},
		{map[string]any{"query": ""}, `invalid argument "query": must not be empty`},
		{map[string]any{"query": "Dune", "extra": 1}, "invalid arguments:"},
	} {
		res, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "search_books", Arguments: tc.args})
		if err != nil || !res.IsError {
			t.Fatalf("invalid arguments accepted: %v", tc.args)
		}
		if got := toolText(t, res); !strings.HasPrefix(got, tc.want) {
			t.Errorf("args %v: error = %q, want prefix %q", tc.args, got, tc.want)
		}
	}
}

func TestFriendlyArgumentErrorLeavesToolErrorsAlone(t *testing.T) {
	if _, ok := friendlyArgumentError("not logged in. Ask the user to run `zlib login`"); ok {
		t.Fatal("rewrote a handler error that is not a schema error")
	}
}

type lockedBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func TestMCPLogsToolCallsWithoutArguments(t *testing.T) {
	withDemoMCPFixtures(t)
	var out lockedBuffer
	logger, err := newMCPLogger(&out, "debug")
	if err != nil {
		t.Fatal(err)
	}
	session := connectMCPTestSession(t, mcpServerOptions{logger: logger}, nil)
	ctx := context.Background()
	if _, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "search_books", Arguments: map[string]any{"query": "private-fixture-query"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "search_books", Arguments: map[string]any{"query": "x", "count": 99}}); err != nil {
		t.Fatal(err)
	}
	logged := out.String()
	if !strings.Contains(logged, "tool=search_books") || !strings.Contains(logged, "tool returned an error") {
		t.Fatalf("tool calls not logged: %s", logged)
	}
	if strings.Contains(logged, "private-fixture-query") {
		t.Fatalf("log leaked tool arguments: %s", logged)
	}
}

func TestNewMCPLoggerLevels(t *testing.T) {
	var out bytes.Buffer
	logger, err := newMCPLogger(&out, "off")
	if err != nil {
		t.Fatal(err)
	}
	logger.Error("hidden")
	if out.Len() != 0 {
		t.Fatalf("off level wrote %q", out.String())
	}
	if _, err := newMCPLogger(&out, "verbose"); err == nil {
		t.Fatal("invalid level accepted")
	}
}
