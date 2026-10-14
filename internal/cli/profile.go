package cli

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	zlib "github.com/heartleo/zlib"
	"github.com/spf13/cobra"
)

var profileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Show profile",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := newClient()
		profile, err := c.GetProfile()
		if err != nil {
			return err
		}

		if jsonOut, _ := cmd.Flags().GetBool("json"); jsonOut {
			return printJSON(profile)
		}
		lipgloss.Fprintln(cmd.OutOrStdout(), renderProfileCard(profile))
		return nil
	},
}

func renderProfileCard(profile zlib.UserProfile) string {
	limits := profile.DownloadLimit
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(currentTheme.Accent)
	labelStyle := lipgloss.NewStyle().Foreground(currentTheme.Muted).Width(17)
	var rows []string
	rows = append(rows, titleStyle.Render("Profile"))
	appendField := func(label, value string) {
		if value != "" {
			rows = append(rows, fmt.Sprintf("  %s%s", labelStyle.Render(label), value))
		}
	}
	appendField("Name", profile.Name)
	appendField("Email", profile.Email)
	appendField("Kindle email", profile.KindleEmail)
	if profile.Premium != nil {
		if *profile.Premium {
			appendField("Plan", "Premium")
		} else {
			appendField("Plan", "Standard")
		}
	}
	if profile.Confirmed != nil {
		if *profile.Confirmed {
			appendField("Email confirmed", "Yes")
		} else {
			appendField("Email confirmed", "No")
		}
	}
	if profile.DonationsActive != nil {
		if *profile.DonationsActive {
			appendField("Donation", "Active")
		} else {
			appendField("Donation", "Inactive")
		}
	}
	appendField("Donation expires", profile.DonationsExpire)
	appendField("Personal domains", strings.Join(profile.PersonalDomains, ", "))
	rows = append(rows, "", titleStyle.Render("Daily Downloads"), "")
	if !profile.QuotaKnown {
		rows = append(rows, "  Quota unavailable")
	} else {
		// Progress bar
		barWidth := 30
		var filled int
		if limits.DailyAllowed > 0 {
			filled = limits.DailyAmount * barWidth / limits.DailyAllowed
			if filled > barWidth {
				filled = barWidth
			}
		}
		empty := barWidth - filled

		barColor := remainingColor(limits.DailyRemaining)
		filledStyle := lipgloss.NewStyle().Foreground(barColor)
		emptyStyle := lipgloss.NewStyle().Foreground(currentTheme.Surface)
		bar := filledStyle.Render(strings.Repeat("█", filled)) +
			emptyStyle.Render(strings.Repeat("░", empty))

		ratio := lipgloss.NewStyle().Bold(true).Render(
			fmt.Sprintf("%d / %d", limits.DailyAmount, limits.DailyAllowed))
		remainingStyle := lipgloss.NewStyle().Foreground(barColor).Bold(true)

		rows = append(rows, fmt.Sprintf("  %s  %s", bar, ratio))
		rows = append(rows, "")
		appendField("Remaining", remainingStyle.Render(fmt.Sprintf("%d", limits.DailyRemaining)))
		if limits.DailyReset != "" {
			appendField("Resets at", limits.DailyReset)
		}
	}

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(currentTheme.Accent).
		Padding(1, 2).
		Render(strings.Join(rows, "\n"))
}

func init() {
	profileCmd.Flags().Bool("json", false, "Output account profile and download quota as JSON")
}
