package cli

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// resolveAllowedDir validates the --allowed-dir root and returns its real path.
func resolveAllowedDir(raw string) (string, error) {
	if strings.TrimSpace(raw) == "" {
		return "", nil
	}
	expanded, err := expandTilde(strings.TrimSpace(raw))
	if err != nil {
		return "", err
	}
	abs, err := filepath.Abs(expanded)
	if err != nil {
		return "", err
	}
	real, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", fmt.Errorf("allowed directory %q: %w", raw, err)
	}
	info, err := os.Stat(real)
	if err != nil {
		return "", fmt.Errorf("allowed directory %q: %w", raw, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("allowed directory %q is not a directory", raw)
	}
	return real, nil
}

// confineToDir resolves target through any symlinks in its existing part and
// rejects it unless the result lies inside root (a real path from
// resolveAllowedDir). Like the reference filesystem MCP server, it checks the
// resolved path rather than the string the caller supplied, so a symlink
// inside root cannot point outside it. Components that do not exist yet cannot
// be symlinks, so they are appended unresolved.
func confineToDir(root, target string) (string, error) {
	abs, err := filepath.Abs(target)
	if err != nil {
		return "", err
	}
	existing, rest := abs, ""
	for {
		_, err := os.Lstat(existing)
		if err == nil {
			break
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return "", err
		}
		parent := filepath.Dir(existing)
		if parent == existing {
			return "", fmt.Errorf("%s: no existing parent directory", target)
		}
		rest = filepath.Join(filepath.Base(existing), rest)
		existing = parent
	}
	real, err := filepath.EvalSymlinks(existing)
	if err != nil {
		// Fail closed: Windows reports paths through a junction as missing here.
		return "", fmt.Errorf("%s: cannot resolve the path for the allowed directory check: %w", target, err)
	}
	resolved := filepath.Join(real, rest)
	rel, err := filepath.Rel(root, resolved)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "", fmt.Errorf("%s is outside the allowed directory %s", target, root)
	}
	// EvalSymlinks does not follow Windows junctions (mount points); they
	// report ModeIrregular and could still lead outside root, so refuse them.
	dir := root
	for _, part := range strings.Split(rel, string(filepath.Separator)) {
		if part == "." || part == "" {
			continue
		}
		dir = filepath.Join(dir, part)
		info, err := os.Lstat(dir)
		if errors.Is(err, fs.ErrNotExist) {
			break
		}
		if err != nil {
			return "", err
		}
		if info.Mode()&fs.ModeIrregular != 0 {
			return "", fmt.Errorf("%s passes through %s, a junction or mount point the allowed directory check cannot follow", target, dir)
		}
	}
	return resolved, nil
}
