package tailnet

import (
	"errors"
	"log"
	"path/filepath"
	"time"

	"github.com/nedlane/cctrack/internal/config"
	"github.com/nedlane/cctrack/internal/parser"
)

// HostResult reports the outcome of syncing one peer.
type HostResult struct {
	Host             string `json:"host"`
	FilesParsed      int    `json:"files_parsed"`
	SessionsAffected int    `json:"sessions_affected"`
	Err              error  `json:"-"`
	ErrMsg           string `json:"error,omitempty"`
}

// SyncReport is the aggregate result of one sync cycle.
type SyncReport struct {
	Hosts                 []HostResult `json:"hosts"`
	TotalSessionsAffected int          `json:"total_sessions_affected"`
	Skipped               bool         `json:"skipped"` // tailscale unavailable
}

// Syncer orchestrates discover -> pull -> parse for every SSH-reachable peer.
// Dependencies are injected so the loop can be tested without a network.
type Syncer struct {
	discoverer Discoverer
	puller     Puller
	parser     *parser.Parser
	mirrorRoot string
	remoteDir  string
}

func NewSyncer(d Discoverer, pl Puller, ps *parser.Parser, mirrorRoot, remoteDir string) *Syncer {
	return &Syncer{
		discoverer: d,
		puller:     pl,
		parser:     ps,
		mirrorRoot: mirrorRoot,
		remoteDir:  remoteDir,
	}
}

// FromConfig builds a Syncer wired to the real tailscale CLI and SSH puller.
func FromConfig(cfg *config.Config, ps *parser.Parser) *Syncer {
	tc := cfg.Tailnet
	timeout := time.Duration(tc.SSHTimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 20 * time.Second
	}
	return NewSyncer(
		NewTailscaleDiscoverer(Filter{Include: tc.IncludeHosts, Exclude: tc.ExcludeHosts}),
		NewSSHPuller(timeout),
		ps,
		config.MirrorRoot(),
		tc.RemoteClaudeDir,
	)
}

// Sync runs one full cycle. A failure for one peer is recorded on its
// HostResult and never aborts the others. If tailscale itself is unavailable,
// the report is marked Skipped with no error (local tracking is independent).
func (s *Syncer) Sync() (*SyncReport, error) {
	peers, err := s.discoverer.Discover()
	if err != nil {
		if errors.Is(err, ErrTailscaleUnavailable) {
			return &SyncReport{Skipped: true}, nil
		}
		return nil, err
	}

	report := &SyncReport{}
	for _, peer := range peers {
		res := s.syncPeer(peer)
		report.Hosts = append(report.Hosts, res)
		report.TotalSessionsAffected += res.SessionsAffected
	}
	return report, nil
}

func (s *Syncer) syncPeer(peer Peer) HostResult {
	res := HostResult{Host: peer.Host}
	mirrorDir := filepath.Join(s.mirrorRoot, peer.Host, "projects")

	if err := s.puller.Pull(peer.Host, s.remoteDir, mirrorDir); err != nil {
		res.Err = err
		res.ErrMsg = err.Error()
		log.Printf("tailnet: pull from %s failed: %v", peer.Host, err)
		return res
	}

	files, sessions, err := s.parser.ParseAllForHost(mirrorDir, peer.Host)
	if err != nil {
		res.Err = err
		res.ErrMsg = err.Error()
		log.Printf("tailnet: parse of %s mirror failed: %v", peer.Host, err)
		return res
	}
	res.FilesParsed = files
	res.SessionsAffected = sessions
	return res
}
