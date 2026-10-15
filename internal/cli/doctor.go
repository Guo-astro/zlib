package cli

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
	zlib "github.com/heartleo/zlib"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Find available EAPI domains",
	Long:  "Probe EAPI domains without logging in or changing your selected domain.",
	RunE: func(cmd *cobra.Command, args []string) error {
		proxyValue, _ := cmd.Flags().GetString("proxy")
		proxyURL, err := validateDoctorProxy(proxyValue)
		if err != nil {
			return err
		}
		results := checkAvailableDomains(cmd.Context(), zlib.KnownDomains(), proxyURL)
		if jsonOut, _ := cmd.Flags().GetBool("json"); jsonOut {
			return printJSON(results)
		}
		if len(results) == 0 {
			lipgloss.Fprintln(cmd.OutOrStdout(), "No available EAPI domains found.")
			return nil
		}

		rows := make([][]string, 0, len(results))
		for _, result := range results {
			rows = append(rows, []string{result.Domain, "Available"})
		}

		t := table.New().
			Border(lipgloss.RoundedBorder()).
			BorderStyle(lipgloss.NewStyle().Foreground(currentTheme.Accent)).
			Headers("Domain", "Status").
			Rows(rows...).
			StyleFunc(func(row, col int) lipgloss.Style {
				pad := lipgloss.NewStyle().PaddingLeft(1).PaddingRight(1)
				if row == table.HeaderRow {
					return pad.Bold(true).Foreground(currentTheme.Accent)
				}
				switch col {
				case 0:
					return pad.Foreground(currentTheme.Muted)
				case 1:
					return pad.Foreground(currentTheme.Success)
				}
				return pad
			})

		lipgloss.Fprintln(cmd.OutOrStdout(), t)
		return nil
	},
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
	doctorCmd.Flags().Bool("json", false, "Output available domains as JSON")
	doctorCmd.Flags().String("proxy", "", "Proxy URL for checks (http, https, socks5, or socks5h)")
	doctorCmd.Flags().Bool("eapi", false, "Check the EAPI health endpoint")
	_ = doctorCmd.Flags().MarkHidden("eapi")
}
