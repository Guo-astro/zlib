package cli

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
	"github.com/heartleo/zlib/internal/config"
	zlib "github.com/heartleo/zlib/internal/zlib"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Find working Z-Library domains",
	Long:  "Check known domains without logging in or changing the selected domain.",
	RunE: func(cmd *cobra.Command, args []string) error {
		proxyValue, _ := cmd.Flags().GetString("proxy")
		proxyURL, err := validateDoctorProxy(proxyValue)
		if err != nil {
			return err
		}
		jsonOut, _ := cmd.Flags().GetBool("json")
		if !jsonOut && isInteractive() {
			model := newDoctorPageModel(zlib.KnownDomains(), proxyURL, doctorCurrentDomain(), nil)
			defer model.stop()
			result, err := tea.NewProgram(model, tea.WithContext(cmd.Context())).Run()
			if final, ok := result.(doctorPageModel); ok {
				final.stop()
			}
			return err
		}
		results := checkAvailableDomains(cmd.Context(), zlib.KnownDomains(), proxyURL)
		if jsonOut {
			if err := printJSON(results); err != nil {
				return err
			}
			return noDomainsError(results)
		}
		if err := noDomainsError(results); err != nil {
			return err
		}

		current := doctorCurrentDomain()
		rows := make([][]string, 0, len(results))
		for _, result := range results {
			item := doctorDomainItem{check: result, current: result.Domain == current}
			rows = append(rows, []string{item.Title()})
		}

		t := table.New().
			Border(lipgloss.RoundedBorder()).
			BorderStyle(lipgloss.NewStyle().Foreground(currentTheme.Accent)).
			Headers("Available domains").
			Rows(rows...).
			StyleFunc(func(row, col int) lipgloss.Style {
				pad := lipgloss.NewStyle().PaddingLeft(1).PaddingRight(1)
				if row == table.HeaderRow {
					return pad.Bold(true).Foreground(currentTheme.Accent)
				}
				if results[row].Domain == current {
					return pad.Bold(true).Foreground(currentTheme.Accent)
				}
				return pad.Foreground(currentTheme.Title)
			})

		lipgloss.Fprintln(cmd.OutOrStdout(), t)
		return nil
	},
}

// noDomainsError makes doctor fail when nothing is usable, so a script can
// tell "no domain reachable" from success instead of parsing an empty list.
func noDomainsError(results []zlib.DomainCheck) error {
	if len(results) > 0 {
		return nil
	}
	return fmt.Errorf("%w: none of the known domains answered; check your network, or retry with --proxy URL (or set %s)", zlib.ErrNoDomain, zlib.EnvProxy)
}

func doctorCurrentDomain() string {
	session, _ := config.LoadSession()
	current := resolveClientDomain(os.Getenv(zlib.EnvDomain), session.Domain)
	if normalized, err := zlib.ParseAllowedDomain(current); err == nil {
		return normalized
	}
	return current
}

func checkAvailableDomains(ctx context.Context, candidates []string, proxy string) []zlib.DomainCheck {
	return availableDomains(zlib.ProbeDomainsWithOptions(ctx, candidates, zlib.DomainProbeOptions{ProxyURL: proxy, Path: "/eapi/info/domains"}))
}

func availableDomains(results []zlib.DomainCheck) []zlib.DomainCheck {
	available := make([]zlib.DomainCheck, 0, len(results))
	for _, result := range results {
		if result.Status != zlib.DomainStatusHealthy {
			continue
		}
		finalURL, err := url.Parse(result.FinalURL)
		if err != nil || finalURL.Host == "" {
			continue
		}
		if _, err := zlib.ParseAllowedDomain(finalURL.Scheme + "://" + finalURL.Host); err != nil {
			continue
		}
		available = append(available, result)
	}
	return available
}

func validateDoctorProxy(rawURL string) (string, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return "", nil
	}
	proxyURL, err := url.ParseRequestURI(rawURL)
	if err != nil || proxyURL.Host == "" {
		return "", fmt.Errorf("invalid proxy URL")
	}
	switch strings.ToLower(proxyURL.Scheme) {
	case "http", "https", "socks5", "socks5h":
	default:
		return "", fmt.Errorf("unsupported proxy scheme %q", proxyURL.Scheme)
	}
	if proxyURL.Path != "" && proxyURL.Path != "/" || proxyURL.RawQuery != "" || proxyURL.Fragment != "" {
		return "", fmt.Errorf("proxy URL must not contain a path, query, or fragment")
	}
	return rawURL, nil
}

func init() {
	doctorCmd.Flags().Bool("json", false, "Print available domains as JSON")
	doctorCmd.Flags().String("proxy", "", "Proxy URL for the checks (http, https, socks5, socks5h)")
	doctorCmd.Flags().Bool("eapi", false, "Check the EAPI health endpoint")
	_ = doctorCmd.Flags().MarkHidden("eapi")
}
