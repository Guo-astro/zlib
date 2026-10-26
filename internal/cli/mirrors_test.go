package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"

	zlib "github.com/heartleo/zlib"
	"github.com/heartleo/zlib/internal/config"
)

// isolateMirrors gives a test its own config dir and mirror set, and clears
// ZLIB_DOMAIN so a developer .env cannot leak in.
func isolateMirrors(t *testing.T, mirrors ...string) {
	t.Helper()
	isolateConfigHome(t)
	t.Setenv(zlib.EnvDomain, "")
	previous := zlib.UserMirrors()
	if err := zlib.SetUserMirrors(mirrors); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = zlib.SetUserMirrors(previous) })
	if len(mirrors) > 0 {
		if err := config.SaveConfig(config.Config{UserMirrors: mirrors}); err != nil {
			t.Fatal(err)
		}
	}
}

// stubVerifyMirror replaces the network check and records the origins asked.
func stubVerifyMirror(t *testing.T, err error) *[]string {
	t.Helper()
	var calls []string
	previous := verifyMirror
	verifyMirror = func(_ context.Context, raw string, _ zlib.DomainProbeOptions) (zlib.DomainCheck, error) {
		calls = append(calls, raw)
		return zlib.DomainCheck{Domain: raw, Status: zlib.DomainStatusHealthy, FinalURL: raw + "/eapi/info/domains"}, err
	}
	t.Cleanup(func() { verifyMirror = previous })
	return &calls
}

func savedMirrors(t *testing.T) []string {
	t.Helper()
	cfg, err := config.LoadConfig()
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		t.Fatal(err)
	}
	return cfg.UserMirrors
}

func TestMirrorsAddChecksThenTrustsWithYes(t *testing.T) {
	isolateMirrors(t)
	calls := stubVerifyMirror(t, nil)
	var out, status bytes.Buffer

	if err := addMirrorCommand(t.Context(), "HTTPS://Mirror.Example/", "", true, true, nil, &out, &status); err != nil {
		t.Fatal(err)
	}

	if want := []string{"https://mirror.example"}; !slices.Equal(*calls, want) {
		t.Errorf("verified %v, want %v", *calls, want)
	}
	var got mirrorResult
	if err := json.Unmarshal(out.Bytes(), &got); err != nil || got.Mirror != "https://mirror.example" || got.Added == nil || !*got.Added {
		t.Errorf("json = %s (%v)", out.String(), err)
	}
	if want := []string{"https://mirror.example"}; !slices.Equal(savedMirrors(t), want) || !slices.Equal(zlib.UserMirrors(), want) {
		t.Errorf("saved %v, active %v, want %v", savedMirrors(t), zlib.UserMirrors(), want)
	}
	if _, err := zlib.ParseAllowedDomain("https://mirror.example"); err != nil {
		t.Errorf("added mirror is not trusted: %v", err)
	}
}

func TestMirrorsAddWithoutYesOffTerminalIsUsageError(t *testing.T) {
	isolateMirrors(t)
	stubVerifyMirror(t, nil)
	var out, status bytes.Buffer

	err := addMirrorCommand(t.Context(), "mirror.example", "", false, false, strings.NewReader("y\n"), &out, &status)

	if exitCode(err) != exitUsage || !strings.Contains(err.Error(), "--yes") || !strings.Contains(err.Error(), "remix_userkey") {
		t.Fatalf("err = %v (exit %d), want usage error naming --yes and the credential risk", err, exitCode(err))
	}
	if len(savedMirrors(t)) != 0 || len(zlib.UserMirrors()) != 0 {
		t.Error("mirror saved without confirmation")
	}
}

func TestMirrorsAddRejectsFailedCheck(t *testing.T) {
	isolateMirrors(t)
	stubVerifyMirror(t, fmt.Errorf("%w: https://mirror.example answered, but not with the Z-Library EAPI", zlib.ErrMirrorUnverified))
	var out, status bytes.Buffer

	err := addMirrorCommand(t.Context(), "mirror.example", "", true, false, nil, &out, &status)

	if !errors.Is(err, zlib.ErrMirrorUnverified) || exitCode(err) != exitNetwork {
		t.Fatalf("err = %v (exit %d), want ErrMirrorUnverified with exit %d", err, exitCode(err), exitNetwork)
	}
	if len(savedMirrors(t)) != 0 {
		t.Error("unverified mirror saved")
	}
}

func TestMirrorsAddRejectsBadInputWithoutChecking(t *testing.T) {
	isolateMirrors(t)
	calls := stubVerifyMirror(t, nil)
	for _, raw := range []string{"z-lib.gd", "http://mirror.example", "mirror.example:8443", "127.0.0.1", "mirror"} {
		err := addMirrorCommand(t.Context(), raw, "", true, false, nil, &bytes.Buffer{}, &bytes.Buffer{})
		if exitCode(err) != exitUsage {
			t.Errorf("add %q: err = %v (exit %d), want usage error", raw, err, exitCode(err))
		}
	}
	if len(*calls) != 0 {
		t.Errorf("checked %v before validating input", *calls)
	}
}

func TestMirrorsAddExistingIsNoOp(t *testing.T) {
	isolateMirrors(t, "https://mirror.example")
	calls := stubVerifyMirror(t, nil)
	var out bytes.Buffer

	if err := addMirrorCommand(t.Context(), "mirror.example", "", false, true, nil, &out, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if len(*calls) != 0 || !strings.Contains(out.String(), `"added":false`) {
		t.Errorf("calls %v, output %s", *calls, out.String())
	}
}

func TestMirrorsRemoveReportsCurrentDomain(t *testing.T) {
	isolateMirrors(t, "https://mirror.example", "https://other.example")
	t.Setenv(zlib.EnvDomain, "https://mirror.example")
	var out, status bytes.Buffer

	if err := removeMirrorCommand("mirror.example", true, &out, &status); err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(out.String(), `"removed":true`) || !strings.Contains(out.String(), `"current":true`) {
		t.Errorf("json = %s", out.String())
	}
	if !strings.Contains(status.String(), "current domain") {
		t.Errorf("status = %q, want a current-domain warning", status.String())
	}
	if want := []string{"https://other.example"}; !slices.Equal(savedMirrors(t), want) || !slices.Equal(zlib.UserMirrors(), want) {
		t.Errorf("saved %v, active %v, want %v", savedMirrors(t), zlib.UserMirrors(), want)
	}
	if _, err := zlib.ParseAllowedDomain("https://mirror.example"); err == nil {
		t.Error("removed mirror is still trusted")
	}
}

func TestMirrorsRemoveUnknown(t *testing.T) {
	isolateMirrors(t)
	for _, raw := range []string{"mirror.example", "z-lib.gd"} {
		if err := removeMirrorCommand(raw, false, &bytes.Buffer{}, &bytes.Buffer{}); exitCode(err) != exitError {
			t.Errorf("remove %q: err = %v (exit %d), want exit %d", raw, err, exitCode(err), exitError)
		}
	}
}

func TestMirrorsListJSON(t *testing.T) {
	isolateMirrors(t, "https://mirror.example")
	var out bytes.Buffer
	if err := printMirrors(&out, true); err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(out.String()); got != `["https://mirror.example"]` {
		t.Errorf("list = %s", got)
	}
}

func TestConfirmMirrorTrust(t *testing.T) {
	for answer, want := range map[string]bool{"y\n": true, "YES\n": true, "\n": false, "n\n": false, "": false} {
		var out bytes.Buffer
		got, err := confirmMirrorTrust(strings.NewReader(answer), &out, "https://mirror.example")
		if err != nil || got != want {
			t.Errorf("answer %q = %v, %v; want %v", answer, got, err, want)
		}
		if !strings.Contains(out.String(), "remix_userkey") {
			t.Errorf("prompt %q does not warn about the login", out.String())
		}
	}
}

func TestLoadUserMirrorsKeepsValidEntries(t *testing.T) {
	isolateMirrors(t)
	if err := loadUserMirrors(); err != nil || len(zlib.UserMirrors()) != 0 {
		t.Fatalf("missing config: %v, mirrors %v", err, zlib.UserMirrors())
	}
	if err := config.SaveConfig(config.Config{UserMirrors: []string{"https://mirror.example", "http://bad.example"}}); err != nil {
		t.Fatal(err)
	}
	if err := loadUserMirrors(); err == nil {
		t.Error("invalid entry not reported")
	}
	if want := []string{"https://mirror.example"}; !slices.Equal(zlib.UserMirrors(), want) {
		t.Errorf("mirrors = %v, want %v", zlib.UserMirrors(), want)
	}
}

func TestConfigSaversKeepUserMirrors(t *testing.T) {
	isolateMirrors(t, "https://mirror.example")
	if err := rememberLoginEmail("reader@example.com"); err != nil {
		t.Fatal(err)
	}
	if want := []string{"https://mirror.example"}; !slices.Equal(savedMirrors(t), want) {
		t.Errorf("after saving the login email, mirrors = %v, want %v", savedMirrors(t), want)
	}
}
