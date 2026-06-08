package tailnet

import (
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"sort"
	"strings"
)

// ErrTailscaleUnavailable is returned when the tailscale CLI cannot be run
// (not installed, daemon down). Callers should treat this as "no peers" rather
// than a hard failure — local tracking is independent of the tailnet.
var ErrTailscaleUnavailable = errors.New("tailscale CLI unavailable")

// Peer is an SSH-reachable Tailscale peer.
//
// Host is the short MagicDNS label (e.g. "mac", "pi") — the first segment of
// the peer's DNSName. It is used both as the `tailscale ssh` target and as the
// host label stored on sessions, matching how the user's tmux switcher refers
// to peers.
type Peer struct {
	Host string
}

// tsStatus mirrors the subset of `tailscale status --json` we consume.
type tsStatus struct {
	Peer map[string]tsPeer `json:"Peer"`
}

type tsPeer struct {
	HostName    string   `json:"HostName"`
	DNSName     string   `json:"DNSName"`
	OS          string   `json:"OS"`
	Online      bool     `json:"Online"`
	SSHHostKeys []string `json:"sshHostKeys"`
}

// Filter restricts which discovered peers are kept.
type Filter struct {
	Include []string // if non-empty, only these host labels are kept
	Exclude []string // these host labels are dropped
}

// Discoverer returns the set of peers cctrack should pull from.
type Discoverer interface {
	Discover() ([]Peer, error)
}

// TailscaleDiscoverer discovers peers by shelling out to the tailscale CLI.
type TailscaleDiscoverer struct {
	filter    Filter
	runStatus func() ([]byte, error) // overridable in tests
}

func NewTailscaleDiscoverer(f Filter) *TailscaleDiscoverer {
	return &TailscaleDiscoverer{filter: f, runStatus: runTailscaleStatus}
}

func runTailscaleStatus() ([]byte, error) {
	out, err := exec.Command("tailscale", "status", "--json").Output()
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return nil, ErrTailscaleUnavailable
		}
		return nil, fmt.Errorf("running tailscale status: %w", err)
	}
	return out, nil
}

func (d *TailscaleDiscoverer) Discover() ([]Peer, error) {
	raw, err := d.runStatus()
	if err != nil {
		return nil, err
	}
	var st tsStatus
	if err := json.Unmarshal(raw, &st); err != nil {
		return nil, fmt.Errorf("parsing tailscale status: %w", err)
	}
	return filterPeers(st, d.filter), nil
}

// filterPeers keeps online peers advertising Tailscale SSH host keys, then
// applies include/exclude overrides. Pure function — unit-tested against a
// captured status fixture.
func filterPeers(st tsStatus, f Filter) []Peer {
	include := toSet(f.Include)
	exclude := toSet(f.Exclude)

	var peers []Peer
	for _, p := range st.Peer {
		// `sshHostKeys` present and non-empty == Tailscale SSH is enabled on
		// this peer. This is requirement (b): "has SSH available".
		if !p.Online || len(p.SSHHostKeys) == 0 {
			continue
		}
		label := hostLabel(p)
		if label == "" {
			continue
		}
		if len(include) > 0 && !include[label] {
			continue
		}
		if exclude[label] {
			continue
		}
		peers = append(peers, Peer{Host: label})
	}
	sort.Slice(peers, func(i, j int) bool { return peers[i].Host < peers[j].Host })
	return peers
}

// hostLabel derives the short MagicDNS label from a peer: the first segment of
// DNSName ("mac.tail2d62ca.ts.net." -> "mac"), falling back to HostName.
func hostLabel(p tsPeer) string {
	name := strings.TrimSuffix(p.DNSName, ".")
	if name == "" {
		return p.HostName
	}
	if i := strings.IndexByte(name, '.'); i > 0 {
		return name[:i]
	}
	return name
}

func toSet(items []string) map[string]bool {
	m := make(map[string]bool, len(items))
	for _, it := range items {
		if it != "" {
			m[it] = true
		}
	}
	return m
}
