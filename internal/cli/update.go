package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"

	"github.com/heartleo/zlib/internal/zlib"
	"github.com/spf13/cobra"
	"golang.org/x/mod/semver"
)

const goInstallPackage = "github.com/heartleo/zlib/cmd/zlib@latest"

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update zlib to the latest release",
	Long:  "Update zlib through the channel that installed it: Homebrew, winget, go install, or a release download.",
	Args:  cobra.NoArgs,
	RunE:  runUpdate,
}

type updateMethod string

const (
	updateRelease updateMethod = "release"
	updateBrew    updateMethod = "brew"
	updateWinget  updateMethod = "winget"
	updateGo      updateMethod = "go"
)

func runUpdate(cmd *cobra.Command, _ []string) error {
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("find running executable: %w", err)
	}
	executable, err = filepath.EvalSymlinks(executable)
	if err != nil {
		return fmt.Errorf("resolve running executable: %w", err)
	}

	method := detectUpdateMethod(cmd.Context(), executable)
	if method != updateRelease {
		name, args := updateCommand(method)
		fmt.Fprintf(cmd.OutOrStdout(), "Updating zlib via %s...\n", name)
		process := exec.CommandContext(cmd.Context(), name, args...)
		process.Stdout = cmd.OutOrStdout()
		process.Stderr = cmd.ErrOrStderr()
		process.Stdin = cmd.InOrStdin()
		if err := process.Run(); err != nil {
			return fmt.Errorf("%s update failed: %w", name, err)
		}
		fmt.Fprintln(cmd.OutOrStdout(), "Update command finished. Run zlib version to verify it.")
		return nil
	}

	currentVersion := zlib.Version
	if !semver.IsValid(currentVersion) {
		return fmt.Errorf("cannot update a development build (%s); install a release or update the source checkout", zlib.Version)
	}
	result, err := newReleaseUpdater().update(cmd.Context(), executable, currentVersion, runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return err
	}
	if result.updated {
		fmt.Fprintf(cmd.OutOrStdout(), "Updated zlib %s → %s. Restart zlib to use the new version.\n", zlib.Version, result.version)
		fmt.Fprintf(cmd.OutOrStdout(), "Previous executable: %s\n", result.backup)
	} else if semver.Compare(currentVersion, result.version) > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "This build (%s) is newer than the latest release (%s).\n", zlib.Version, result.version)
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "zlib %s is already up to date (latest release: %s).\n", zlib.Version, result.version)
	}
	return nil
}

func updateCommand(method updateMethod) (string, []string) {
	switch method {
	case updateBrew:
		return "brew", []string{"upgrade", "heartleo/tap/zlib"}
	case updateWinget:
		return "winget", []string{"upgrade", "--id", "heartleo.zlib", "--exact"}
	case updateGo:
		return "go", []string{"install", goInstallPackage}
	default:
		return "", nil
	}
}

func detectUpdateMethod(ctx context.Context, executable string) updateMethod {
	path := strings.ToLower(strings.ReplaceAll(executable, `\`, "/"))
	if strings.Contains(path, "/cellar/zlib/") {
		return updateBrew
	}
	if strings.Contains(path, "/microsoft/winget/packages/heartleo.zlib_") {
		return updateWinget
	}
	if isModuleBuild(debug.ReadBuildInfo()) && goBinPath(ctx, executable) {
		return updateGo
	}
	return updateRelease
}

// isModuleBuild reports whether the binary is the zlib command built from a
// versioned module, as go install produces. info.Path is the main package;
// info.Main.Path is the module path and never names cmd/zlib.
func isModuleBuild(info *debug.BuildInfo, ok bool) bool {
	return ok && info.Path == "github.com/heartleo/zlib/cmd/zlib" && semver.IsValid(info.Main.Version)
}

func goBinPath(ctx context.Context, executable string) bool {
	output, err := exec.CommandContext(ctx, "go", "env", "-json", "GOBIN", "GOPATH").Output()
	if err != nil {
		return false
	}
	var env struct{ GOBIN, GOPATH string }
	if err := json.Unmarshal(output, &env); err != nil {
		return false
	}
	bin := env.GOBIN
	if bin == "" {
		paths := filepath.SplitList(env.GOPATH)
		if len(paths) == 0 {
			return false
		}
		bin = filepath.Join(paths[0], "bin")
	}
	return sameExecutable(filepath.Join(bin, filepath.Base(executable)), executable)
}

func sameExecutable(a, b string) bool {
	a, errA := filepath.Abs(a)
	b, errB := filepath.Abs(b)
	if errA != nil || errB != nil {
		return false
	}
	if runtime.GOOS == "windows" {
		return strings.EqualFold(a, b)
	}
	return a == b
}
