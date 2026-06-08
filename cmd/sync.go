package cmd

import (
	"fmt"

	"github.com/nedlane/cctrack/internal/config"
	"github.com/nedlane/cctrack/internal/parser"
	"github.com/nedlane/cctrack/internal/store"
	"github.com/nedlane/cctrack/internal/tailnet"
	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Pull Claude Code logs from Tailscale peers and update the database",
	Long: "Discover SSH-reachable machines on your Tailscale network, mirror their\n" +
		"~/.claude/projects logs locally, and fold their token usage into the totals.",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}
		if !cfg.Tailnet.Enabled {
			fmt.Println("Tailnet sync is disabled. Enable it in config (tailnet.enabled = true) or the dashboard settings.")
			return nil
		}

		s, err := store.Open(cfg.DBPath)
		if err != nil {
			return fmt.Errorf("opening store: %w", err)
		}
		defer s.Close()

		p := parser.New(s)
		syncer := tailnet.FromConfig(cfg, p)

		report, err := syncer.Sync()
		if err != nil {
			return fmt.Errorf("syncing: %w", err)
		}
		if report.Skipped {
			fmt.Println("tailscale CLI unavailable — nothing to sync.")
			return nil
		}
		if len(report.Hosts) == 0 {
			fmt.Println("No SSH-reachable Tailscale peers found.")
			return nil
		}

		for _, h := range report.Hosts {
			if h.Err != nil {
				fmt.Printf("  %-16s  failed: %s\n", h.Host, h.ErrMsg)
				continue
			}
			fmt.Printf("  %-16s  %d files, %d sessions updated\n", h.Host, h.FilesParsed, h.SessionsAffected)
		}
		fmt.Printf("Synced %d host(s), %d sessions updated total\n", len(report.Hosts), report.TotalSessionsAffected)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(syncCmd)
}
