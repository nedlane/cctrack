package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	LogDir             string        `json:"log_dir"`
	DBPath             string        `json:"db_path"`
	Port               int           `json:"port"`
	MonthlyBudgetUSD   float64       `json:"monthly_budget_usd"`
	OpenBrowserOnServe bool          `json:"open_browser_on_serve"`
	Tailnet            TailnetConfig `json:"tailnet"`
}

// TailnetConfig controls pulling Claude Code logs from other machines on the
// user's Tailscale network. Disabled by default so single-machine setups are
// unaffected.
type TailnetConfig struct {
	Enabled             bool     `json:"enabled"`
	SyncIntervalMinutes int      `json:"sync_interval_minutes"`
	RemoteClaudeDir     string   `json:"remote_claude_dir"` // relative to remote home
	IncludeHosts        []string `json:"include_hosts"`     // empty = all SSH-capable peers
	ExcludeHosts        []string `json:"exclude_hosts"`
	SSHTimeoutSeconds   int      `json:"ssh_timeout_seconds"`
}

func DefaultConfig() *Config {
	home, _ := os.UserHomeDir()
	return &Config{
		LogDir:             filepath.Join(home, ".claude", "projects"),
		DBPath:             filepath.Join(home, ".cctrack", "cctrack.db"),
		Port:               7432,
		MonthlyBudgetUSD:   0,
		OpenBrowserOnServe: true,
		Tailnet: TailnetConfig{
			Enabled:             false,
			SyncIntervalMinutes: 5,
			RemoteClaudeDir:     ".claude/projects",
			IncludeHosts:        []string{},
			ExcludeHosts:        []string{},
			SSHTimeoutSeconds:   20,
		},
	}
}

func ConfigDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".cctrack")
}

// MirrorRoot is where pulled remote logs are mirrored, one subdir per host:
// ~/.cctrack/hosts/<host>/projects
func MirrorRoot() string {
	return filepath.Join(ConfigDir(), "hosts")
}

func ConfigPath() string {
	return filepath.Join(ConfigDir(), "config.json")
}

func Load() (*Config, error) {
	cfg := DefaultConfig()
	data, err := os.ReadFile(ConfigPath())
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) Save() error {
	if err := os.MkdirAll(ConfigDir(), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(ConfigPath(), data, 0644)
}
