package cli

import (
	"context"
	"encoding/json"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	zlib "github.com/heartleo/zlib"
	"github.com/spf13/cobra"
)

func settingsCommand(name, page, key string) *cobra.Command {
	return &cobra.Command{Use: name, Short: "Configure " + name, Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		store, err := newSettingsStore()
		if err != nil {
			return err
		}
		ctx, cancel := context.WithCancel(cmd.Context())
		defer cancel()
		m, err := newSettingsModel(ctx, store, page)
		if err != nil {
			return err
		}
		jsonOut, _ := cmd.Flags().GetBool("json")
		if jsonOut || !isInteractive() {
			values := m.values
			if page == "network" || page == "domain" {
				values = values[:2]
			} else if page == "download" {
				values = values[2:]
			}
			var output []settingValue
			for _, v := range values {
				if page == "domain" && v.Key != zlib.EnvDomain {
					continue
				}
				if key != "" && key != v.Key {
					continue
				}
				v.Value = displaySetting(v.Key, v.Value)
				v.Saved = displaySetting(v.Key, v.Saved)
				output = append(output, v)
			}
			if jsonOut {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(output)
			}
			for _, v := range output {
				lipgloss.Fprintf(cmd.OutOrStdout(), "%s: %s\n  Source: %s\n", v.Key, v.Value, v.Source)
			}
			return nil
		}
		var initial tea.Cmd
		if key != "" {
			initial = m.edit(key)
		} else if page == "domain" {
			initial = m.probe(m.effectiveProxy(), "domain")
		}
		p := tea.NewProgram(settingsStart{settingsModel: m, initial: initial})
		_, err = p.Run()
		return err
	}}
}

type settingsStart struct {
	settingsModel
	initial tea.Cmd
}

func (m settingsStart) Init() tea.Cmd { return m.initial }

func init() {
	root := settingsCommand("config", "home", "")
	root.Short = "Configure network and download settings"
	root.PersistentFlags().Bool("json", false, "Show settings and sources as JSON without opening the TUI")
	network := settingsCommand("network", "network", "")
	network.AddCommand(settingsCommand("domain", "domain", ""), settingsCommand("proxy", "network", zlib.EnvProxy))
	download := settingsCommand("download", "download", "")
	download.AddCommand(settingsCommand("directory", "download", downloadDirEnvVar), settingsCommand("retries", "download", zlib.EnvRetries))
	root.AddCommand(network, download)
	rootCmd.AddCommand(root)
}
