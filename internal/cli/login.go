package cli

import (
	"charm.land/lipgloss/v2"
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/heartleo/zlib"
	"github.com/heartleo/zlib/internal/config"
	"github.com/spf13/cobra"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Log in with a password or cookie file",
	Long:  `Log in with email and password, or import a Netscape cookie file. The session is saved locally.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		email, _ := cmd.Flags().GetString("email")
		password, _ := cmd.Flags().GetString("password")
		cookieFile, _ := cmd.Flags().GetString("cookies")
		domain, _ := cmd.Flags().GetString("domain")
		domain, err := resolveLoginDomain(domain)
		if err != nil {
			return err
		}
		mode := string(zlib.ClientModeEAPI)
		if cookieFile != "" {
			if email != "" || password != "" {
				return fmt.Errorf("--cookies cannot be combined with --email or --password")
			}
			resolved, err := resolveEAPIDomain(cmd.Context(), domain)
			if err != nil {
				return err
			}
			if resolved != domain {
				lipgloss.Fprintf(os.Stdout, "%s %s redirects to %s; logging in there.\n", colorCyan(symbolInfo), domain, resolved)
				domain = resolved
			}
			count, err := importCookieSession(cookieFile, domain, time.Now(), mode)
			if err != nil {
				return fmt.Errorf("failed to import cookies: %w", err)
			}
			lipgloss.Fprintf(os.Stdout, "%s Imported %d cookies.\n", colorGreen(symbolSuccess), count)
			reportPersistedDomain(cmd.OutOrStdout(), cmd.ErrOrStderr(), domain)
			return nil
		}
		// Decide whether to prompt from the flags, before the remembered email
		// is filled in. Letting the prefill satisfy the check would make
		// `zlib login --password X` log into whatever account was remembered,
		// without ever showing the user which one.
		promptNeeded := email == "" || password == ""
		if email == "" {
			email = rememberedLoginEmail()
		}

		if promptNeeded {
			fields, accepted, err := runInlineForm(newInlineForm("Log in", "Log in", []formField{
				{label: "Email", value: email, hint: "Your Z-Library email address", validate: requiredField("email")},
				{label: "Password", value: password, secret: true, hint: "Password is not saved", validate: requiredField("password")},
			}))
			if err != nil {
				return err
			}
			if !accepted {
				return nil
			}
			email, password = fields[0].value, fields[1].value
		}

		resolved, err := resolveEAPIDomain(cmd.Context(), domain)
		if err != nil {
			return err
		}
		if resolved != domain {
			lipgloss.Fprintf(os.Stdout, "%s %s redirects to %s; logging in there.\n", colorCyan(symbolInfo), domain, resolved)
			domain = resolved
		}
		c := zlib.NewClient()
		c.SetDomain(domain)
		if err := c.LoginEAPI(email, password); err != nil {
			return fmt.Errorf("login failed: %w", err)
		}

		if err := config.SaveSession(config.Session{
			Cookies: c.Cookies(),
			Domain:  c.Domain(),
			Mode:    mode,
		}); err != nil {
			return fmt.Errorf("failed to save session: %w", err)
		}
		if err := rememberLoginEmail(email); err != nil {
			lipgloss.Fprintf(cmd.ErrOrStderr(), "Could not remember login email: %v\n", err)
		}

		lipgloss.Fprintf(os.Stdout, "%s Logged in successfully.\n", colorGreen(symbolSuccess))
		reportPersistedDomain(cmd.OutOrStdout(), cmd.ErrOrStderr(), c.Domain())
		return nil
	},
}

// reportPersistedDomain records the domain a login just used as ZLIB_DOMAIN in
// the config-dir .env and says so. A failure here is reported but not fatal:
// the login itself succeeded and the session already carries the domain.
func reportPersistedDomain(stdout, stderr io.Writer, domain string) {
	path, changed, err := persistDotEnvDomain(domain)
	if err != nil {
		lipgloss.Fprintf(stderr, "Could not record %s: %v\n", zlib.EnvDomain, err)
		return
	}
	if changed {
		lipgloss.Fprintf(stdout, "%s Wrote %s=%s to %s\n", colorCyan(symbolInfo), zlib.EnvDomain, domain, path)
	}
}

// eapiProbePath is the unauthenticated endpoint resolveEAPIDomain uses to
// decide whether an origin speaks the EAPI at all.
const eapiProbePath = "/eapi/info/domains"

// resolveEAPIDomain checks that domain answers the EAPI and follows a pure
// redirector to the origin that actually serves it.
//
// Adopting the final origin is not cosmetic. net/http turns a 302 POST into a
// GET and drops the body, so posting the login form at a redirector (zliba.ru
// 302s to zlib.bz) arrives at the target as a bare GET and comes back 404 --
// the failure reported in issue #18. Resolving up front is also what keeps the
// saved session domain on a host the allowlist accepts later.
func resolveEAPIDomain(ctx context.Context, domain string) (string, error) {
	check := zlib.ProbeDomainWithOptions(ctx, domain, zlib.DomainProbeOptions{Path: eapiProbePath})
	if check.Status != zlib.DomainStatusHealthy {
		return "", fmt.Errorf("%s is not usable for EAPI (%s): set %s to a domain reported healthy by `zlib doctor`, or pass --domain", domain, check.Status, zlib.EnvDomain)
	}

	requested, err := url.Parse(domain)
	if err != nil {
		return domain, nil
	}
	finalURL, err := url.Parse(check.FinalURL)
	if err != nil || finalURL.Host == "" || strings.EqualFold(finalURL.Host, requested.Host) {
		return domain, nil
	}
	// The probe left the requested host, so the adopted origin has to clear the
	// allowlist on its own. resolveLoginDomain only vetted what the user typed.
	resolved, err := zlib.ParseAllowedDomain(finalURL.Scheme + "://" + finalURL.Host)
	if err != nil {
		return "", fmt.Errorf("%s redirects to %s, which is not an allowed Z-Library domain: %w", domain, finalURL.Host, err)
	}
	return resolved, nil
}

func resolveLoginDomain(flagDomain string) (string, error) {
	if domain := strings.TrimSpace(flagDomain); domain != "" {
		allowed, err := zlib.ParseAllowedDomain(domain)
		if err != nil {
			return "", fmt.Errorf("EAPI requires an allowed Z-Library domain: %w", err)
		}
		return allowed, err
	}

	if domain := strings.TrimSpace(os.Getenv(zlib.EnvDomain)); domain != "" {
		allowed, err := zlib.ParseAllowedDomain(domain)
		if err != nil {
			return "", fmt.Errorf("EAPI requires an allowed Z-Library domain in %s: %w", zlib.EnvDomain, err)
		}
		return allowed, err
	}

	example := zlib.DefaultEAPIDomain
	return "", fmt.Errorf("domain is not configured: pass --domain %s or set %s=%s (for example in ~/.config/zlib/.env)", example, zlib.EnvDomain, example)
}

// rememberedLoginEmail returns the most recently successful login email. It
// is a convenience for the TUI only; passwords are never persisted.
func rememberedLoginEmail() string {
	cfg, err := config.LoadConfig()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(cfg.LastLoginEmail)
}

func rememberLoginEmail(email string) error {
	email = strings.TrimSpace(email)
	if email == "" {
		return nil
	}

	cfg, err := config.LoadConfig()
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	cfg.LastLoginEmail = email
	return config.SaveConfig(cfg)
}

func init() {
	loginCmd.Flags().String("email", "", "Account email")
	loginCmd.Flags().String("password", "", "Account password")
	loginCmd.Flags().String("cookies", "", "Import a Netscape cookie file")
	loginCmd.Flags().String("domain", "", "Login domain (falls back to $ZLIB_DOMAIN)")
	loginCmd.Flags().Bool("eapi", false, "Use EAPI (now the only mode)")
	loginCmd.Flags().MarkHidden("eapi")
}
