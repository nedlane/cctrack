package tailnet

import (
	"encoding/json"
	"reflect"
	"testing"
)

// statusFixture mirrors the shape of `tailscale status --json`, trimmed to the
// fields we read. It contains: an online SSH-capable mac and pi, an online
// Windows box with no sshHostKeys, and an offline linux box that does have
// sshHostKeys.
const statusFixture = `{
  "Peer": {
    "nodekey:aaa": {
      "HostName": "MusicBook2", "DNSName": "mac.tail2d62ca.ts.net.",
      "OS": "macOS", "Online": true,
      "sshHostKeys": ["ssh-ed25519 AAAA"]
    },
    "nodekey:bbb": {
      "HostName": "raspberrypi", "DNSName": "pi.tail2d62ca.ts.net.",
      "OS": "linux", "Online": true,
      "sshHostKeys": ["ssh-ed25519 BBBB"]
    },
    "nodekey:ccc": {
      "HostName": "homepc", "DNSName": "windows.tail2d62ca.ts.net.",
      "OS": "windows", "Online": true
    },
    "nodekey:ddd": {
      "HostName": "server", "DNSName": "server.tail2d62ca.ts.net.",
      "OS": "linux", "Online": false,
      "sshHostKeys": ["ssh-ed25519 DDDD"]
    }
  }
}`

func parseFixture(t *testing.T) tsStatus {
	t.Helper()
	var st tsStatus
	if err := json.Unmarshal([]byte(statusFixture), &st); err != nil {
		t.Fatalf("unmarshal fixture: %v", err)
	}
	return st
}

func TestFilterPeers_KeepsOnlineSSHCapable(t *testing.T) {
	got := filterPeers(parseFixture(t), Filter{})
	want := []Peer{{Host: "mac"}, {Host: "pi"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v (windows lacks sshHostKeys, server is offline)", got, want)
	}
}

func TestFilterPeers_Include(t *testing.T) {
	got := filterPeers(parseFixture(t), Filter{Include: []string{"pi"}})
	want := []Peer{{Host: "pi"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("include filter: got %v, want %v", got, want)
	}
}

func TestFilterPeers_Exclude(t *testing.T) {
	got := filterPeers(parseFixture(t), Filter{Exclude: []string{"pi"}})
	want := []Peer{{Host: "mac"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("exclude filter: got %v, want %v", got, want)
	}
}

func TestHostLabel_FallsBackToHostName(t *testing.T) {
	if got := hostLabel(tsPeer{HostName: "box", DNSName: ""}); got != "box" {
		t.Fatalf("empty DNSName should fall back to HostName, got %q", got)
	}
	if got := hostLabel(tsPeer{DNSName: "mac.tail2d62ca.ts.net."}); got != "mac" {
		t.Fatalf("expected short label 'mac', got %q", got)
	}
}

func TestDiscoverer_UsesInjectedStatus(t *testing.T) {
	d := &TailscaleDiscoverer{
		filter:    Filter{},
		runStatus: func() ([]byte, error) { return []byte(statusFixture), nil },
	}
	peers, err := d.Discover()
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if len(peers) != 2 {
		t.Fatalf("expected 2 peers, got %d (%v)", len(peers), peers)
	}
}
