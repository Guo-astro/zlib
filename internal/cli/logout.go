package cli

import (
	"charm.land/lipgloss/v2"
	"os"

	"github.com/heartleo/zlib/internal/config"
	"github.com/spf13/cobra"
)

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Log out from Z-Library",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := os.Remove(config.SessionPath()); err != nil && !os.IsNotExist(err) {
			return err
		}
		lipgloss.Fprintf(os.Stdout, "%s Logged out.\n", colorFaint(symbolSuccess))
		return nil
	},
}
