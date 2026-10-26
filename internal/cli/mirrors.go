package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/heartleo/zlib/internal/config"
	zlib "github.com/heartleo/zlib/internal/zlib"
	"github.com/spf13/cobra"
)

// User mirrors are domains the user trusts with their login beyond the
// built-in list (issue #27): Z-Library asks readers to host their own mirrors,
// and donors get personal domains. They live in config.json and are installed
// into the library's domain policy at startup.

// loadUserMirrors installs the saved mirrors. A missing config is not an error.
func loadUserMirrors() error {
	cfg, err := config.LoadConfig()
	if errors.Is(err, os.ErrNotExist) {
		return zlib.SetUserMirrors(nil)
	}
	if err != nil {
		return fmt.Errorf("could not read your mirrors from %s: %w", config.ConfigPath(), err)
	}
	return zlib.SetUserMirrors(cfg.UserMirrors)
}

// updateUserMirrors rewrites the saved mirror list through change, keeping the
// rest of config.json, and applies the result to the running process.
func updateUserMirrors(change func([]string) []string) error {
	cfg, err := config.LoadConfig()
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("could not read %s: %w", config.ConfigPath(), err)
	}
	cfg.UserMirrors = change(slices.Clone(cfg.UserMirrors))
	if err := config.SaveConfig(cfg); err != nil {
		return fmt.Errorf("could not save your mirrors: %w", err)
	}
	return zlib.SetUserMirrors(cfg.UserMirrors)
}

func addUserMirror(origin string) error {
	return updateUserMirrors(func(mirrors []string) []string {
		if slices.Contains(mirrors, origin) {
			return mirrors
		}
		return append(mirrors, origin)
	})
}

func removeUserMirror(origin string) error {
	return updateUserMirrors(func(mirrors []string) []string {
		return slices.DeleteFunc(mirrors, func(m string) bool {
			normalized, err := zlib.ParseMirrorOrigin(m)
			return m == origin || (err == nil && normalized == origin)
		})
	})
}

// isCurrentDomain reports whether origin is the domain commands use now.
func isCurrentDomain(origin string) bool {
	session, _ := config.LoadSession()
	current := resolveClientDomain(os.Getenv(zlib.EnvDomain), session.Domain)
	normalized, err := zlib.ParseMirrorOrigin(current)
	return err == nil && normalized == origin
}

// mirrorTrustWarning is shown before a mirror is trusted, in every interface.
func mirrorTrustWarning(origin string) string {
	return fmt.Sprintf("zlib will send your Z-Library login (remix_userid and remix_userkey) to %s. Only trust a mirror run by you or someone you trust.", origin)
}

// verifyMirror is VerifyUserMirror; tests replace it to avoid the network.
var verifyMirror = zlib.VerifyUserMirror

// confirmMirrorTrust asks on a terminal. Anything but y or yes declines.
func confirmMirrorTrust(in io.Reader, out io.Writer, origin string) (bool, error) {
	fmt.Fprintf(out, "%s\nTrust %s? [y/N] ", mirrorTrustWarning(origin), origin)
	line, err := bufio.NewReader(in).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return false, err
	}
	answer := strings.ToLower(strings.TrimSpace(line))
	return answer == "y" || answer == "yes", nil
}

type mirrorResult struct {
	Mirror  string `json:"mirror"`
	Added   *bool  `json:"added,omitempty"`
	Removed *bool  `json:"removed,omitempty"`
	Current bool   `json:"current,omitempty"`
}

var mirrorsCmd = &cobra.Command{
	Use:   "mirrors",
	Short: "Manage mirrors you trust with your login",
	Long: "Add or remove Z-Library mirrors you trust with your login, in addition to the built-in domains.\n" +
		"On a terminal, without a subcommand, this opens the domain settings; otherwise it lists your mirrors.",
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		jsonOut, _ := cmd.Flags().GetBool("json")
		if !jsonOut && isInteractive() {
			store, err := newSettingsStore()
			if err != nil {
				return err
			}
			ctx, cancel := context.WithCancel(cmd.Context())
			defer cancel()
			m, err := newSettingsModel(ctx, store, "domain")
			if err != nil {
				return err
			}
			return runSettingsTUI(m, "")
		}
		return printMirrors(cmd.OutOrStdout(), jsonOut)
	},
}

var mirrorsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List your mirrors",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		jsonOut, _ := cmd.Flags().GetBool("json")
		return printMirrors(cmd.OutOrStdout(), jsonOut)
	},
}

var mirrorsAddCmd = &cobra.Command{
	Use:   "add <domain>",
	Short: "Check a mirror and trust it with your login",
	Long: "Check that a domain answers as Z-Library, then trust it with your login.\n" +
		"It is trusted as that exact host only. On a terminal you are asked to confirm;\n" +
		"elsewhere pass --yes after reviewing the domain.",
	Example: "  zlib mirrors add libb.la\n  zlib mirrors add https://mirror.example --yes",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		yes, _ := cmd.Flags().GetBool("yes")
		jsonOut, _ := cmd.Flags().GetBool("json")
		proxyFlag, _ := cmd.Flags().GetString("proxy")
		return addMirrorCommand(cmd.Context(), args[0], proxyFlag, yes, jsonOut, cmd.InOrStdin(), cmd.OutOrStdout(), cmd.ErrOrStderr())
	},
}

var mirrorsRemoveCmd = &cobra.Command{
	Use:     "remove <domain>",
	Aliases: []string{"rm"},
	Short:   "Stop trusting a mirror",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		jsonOut, _ := cmd.Flags().GetBool("json")
		return removeMirrorCommand(args[0], jsonOut, cmd.OutOrStdout(), cmd.ErrOrStderr())
	},
}

func init() {
	mirrorsCmd.PersistentFlags().Bool("json", false, "Print the result as JSON")
	mirrorsAddCmd.Flags().BoolP("yes", "y", false, "Trust the mirror without asking")
	mirrorsAddCmd.Flags().String("proxy", "", "Proxy URL for the check (http, https, socks5 or socks5h; defaults to "+zlib.EnvProxy+")")
	mirrorsCmd.AddCommand(mirrorsListCmd, mirrorsAddCmd, mirrorsRemoveCmd)
	rootCmd.AddCommand(mirrorsCmd)
}

func printMirrors(out io.Writer, jsonOut bool) error {
	mirrors := zlib.UserMirrors()
	if jsonOut {
		return printJSONTo(out, mirrors)
	}
	if len(mirrors) == 0 {
		lipgloss.Fprintln(out, "No mirrors added. Add one with `zlib mirrors add <domain>`.")
		return nil
	}
	for _, mirror := range mirrors {
		line := mirror
		if isCurrentDomain(mirror) {
			line += "  (current)"
		}
		lipgloss.Fprintln(out, line)
	}
	return nil
}

func addMirrorCommand(ctx context.Context, raw, proxyFlag string, yes, jsonOut bool, in io.Reader, out, status io.Writer) error {
	origin, err := zlib.ParseMirrorOrigin(raw)
	if err != nil {
		return &usageError{err}
	}
	if zlib.IsBuiltInDomain(origin) {
		return &usageError{fmt.Errorf("%s is already a built-in Z-Library domain; there is no need to add it", origin)}
	}
	proxy, err := validateDoctorProxy(proxyFlag)
	if err != nil {
		return &usageError{err}
	}
	added := false
	if slices.Contains(zlib.UserMirrors(), origin) {
		if jsonOut {
			return printJSONTo(out, mirrorResult{Mirror: origin, Added: &added})
		}
		lipgloss.Fprintf(out, "%s is already one of your mirrors.\n", origin)
		return nil
	}

	fmt.Fprintf(status, "Checking %s...\n", origin)
	if _, err := verifyMirror(ctx, origin, zlib.DomainProbeOptions{ProxyURL: proxy}); err != nil {
		return err
	}
	if !yes {
		if !isInteractive() {
			return &usageError{fmt.Errorf("%s Review %s, then rerun with --yes to trust it", mirrorTrustWarning(origin), origin)}
		}
		ok, err := confirmMirrorTrust(in, status, origin)
		if err != nil {
			return err
		}
		if !ok {
			return errCancelled
		}
	}
	if err := addUserMirror(origin); err != nil {
		return err
	}
	added = true
	if jsonOut {
		return printJSONTo(out, mirrorResult{Mirror: origin, Added: &added})
	}
	lipgloss.Fprintf(out, "%s Trusted %s\n", colorGreen(symbolSuccess), origin)
	lipgloss.Fprintf(out, "%s Use it with `zlib config network domain`, or `zlib login --domain %s`.\n", colorFaint(symbolInfo), origin)
	return nil
}

func removeMirrorCommand(raw string, jsonOut bool, out, status io.Writer) error {
	origin, err := zlib.ParseMirrorOrigin(raw)
	if err != nil {
		return &usageError{err}
	}
	if !slices.Contains(zlib.UserMirrors(), origin) {
		if zlib.IsBuiltInDomain(origin) {
			return fmt.Errorf("%s is a built-in Z-Library domain, not one of your mirrors", origin)
		}
		return fmt.Errorf("%s is not one of your mirrors; `zlib mirrors list` shows them", origin)
	}
	current := isCurrentDomain(origin)
	if err := removeUserMirror(origin); err != nil {
		return err
	}
	if current {
		fmt.Fprintf(status, "%s was your current domain. Choose another with `zlib config network domain` or %s.\n", origin, zlib.EnvDomain)
	}
	removed := true
	if jsonOut {
		return printJSONTo(out, mirrorResult{Mirror: origin, Removed: &removed, Current: current})
	}
	lipgloss.Fprintf(out, "%s Removed %s\n", colorGreen(symbolSuccess), origin)
	return nil
}
