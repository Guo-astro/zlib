package cli

import (
	"errors"
	"os"

	"charm.land/lipgloss/v2"
	"github.com/heartleo/zlib/internal/config"
	"github.com/spf13/cobra"
)

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Delete the saved session",
	RunE: func(cmd *cobra.Command, args []string) error {
		removed, err := clearSavedSession()
		if err != nil {
			return err
		}
		if !removed {
			lipgloss.Fprintf(os.Stdout, "Not logged in.\n")
			return nil
		}
		lipgloss.Fprintf(os.Stdout, "%s Logged out.\n", colorFaint(symbolSuccess))
		return nil
	},
}

func clearSavedSession() (bool, error) {
	err := os.Remove(config.SessionPath())
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return err == nil, err
}
