package tailnet

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/nedlane/cctrack/internal/parser"
	"github.com/nedlane/cctrack/internal/store"
)

// fakeDiscoverer returns a fixed peer list.
type fakeDiscoverer struct {
	peers []Peer
	err   error
}

func (f fakeDiscoverer) Discover() ([]Peer, error) { return f.peers, f.err }

// fakePuller writes a canned JSONL session into each host's mirror dir, or
// returns an error for hosts listed in failOn.
type fakePuller struct {
	failOn map[string]bool
}

func (f fakePuller) Pull(host, remoteDir, mirrorDir string) error {
	if f.failOn[host] {
		return errors.New("simulated pull failure")
	}
	// Lay out a realistic mirror: <mirrorDir>/-home-user-proj/<session>.jsonl
	projDir := filepath.Join(mirrorDir, "-home-user-proj-"+host)
	if err := os.MkdirAll(projDir, 0755); err != nil {
		return err
	}
	line := `{"type":"assistant","sessionId":"sess-` + host + `","requestId":"req-` + host +
		`","timestamp":"2026-06-08T10:00:00Z","message":{"role":"assistant","model":"claude-sonnet-4-5",` +
		`"usage":{"input_tokens":100,"output_tokens":50}}}` + "\n"
	return os.WriteFile(filepath.Join(projDir, "sess-"+host+".jsonl"), []byte(line), 0644)
}

func newTestSyncer(t *testing.T, d Discoverer, pl Puller) (*Syncer, *store.Store, string) {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	s, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	mirrorRoot := t.TempDir()
	p := parser.New(s)
	return NewSyncer(d, pl, p, mirrorRoot, ".claude/projects"), s, mirrorRoot
}

func TestSync_StampsHostAndAggregates(t *testing.T) {
	d := fakeDiscoverer{peers: []Peer{{Host: "mac"}, {Host: "pi"}}}
	syncer, s, _ := newTestSyncer(t, d, fakePuller{})

	report, err := syncer.Sync()
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
	if len(report.Hosts) != 2 || report.TotalSessionsAffected != 2 {
		t.Fatalf("unexpected report: %+v", report)
	}

	hosts, err := s.GetHostBreakdown()
	if err != nil {
		t.Fatalf("GetHostBreakdown: %v", err)
	}
	got := map[string]int{}
	for _, h := range hosts {
		got[h.Host] = h.SessionCount
	}
	if got["mac"] != 1 || got["pi"] != 1 {
		t.Fatalf("expected one session per host, got %v", got)
	}
}

func TestSync_PeerFailureIsIsolated(t *testing.T) {
	d := fakeDiscoverer{peers: []Peer{{Host: "mac"}, {Host: "pi"}}}
	syncer, s, _ := newTestSyncer(t, d, fakePuller{failOn: map[string]bool{"mac": true}})

	report, err := syncer.Sync()
	if err != nil {
		t.Fatalf("Sync should not fail when one peer fails: %v", err)
	}

	var macRes, piRes *HostResult
	for i := range report.Hosts {
		switch report.Hosts[i].Host {
		case "mac":
			macRes = &report.Hosts[i]
		case "pi":
			piRes = &report.Hosts[i]
		}
	}
	if macRes == nil || macRes.Err == nil {
		t.Fatalf("expected mac to report an error, got %+v", macRes)
	}
	if piRes == nil || piRes.Err != nil || piRes.SessionsAffected != 1 {
		t.Fatalf("expected pi to succeed despite mac failing, got %+v", piRes)
	}

	hosts, _ := s.GetHostBreakdown()
	if len(hosts) != 1 || hosts[0].Host != "pi" {
		t.Fatalf("only pi's data should be in the DB, got %v", hosts)
	}
}

func TestSync_IncrementalNoDoubleCount(t *testing.T) {
	d := fakeDiscoverer{peers: []Peer{{Host: "pi"}}}
	syncer, s, _ := newTestSyncer(t, d, fakePuller{})

	if _, err := syncer.Sync(); err != nil {
		t.Fatalf("first sync: %v", err)
	}
	// Second sync over the unchanged mirror must not re-count tokens (file
	// offsets make re-parsing a no-op).
	if _, err := syncer.Sync(); err != nil {
		t.Fatalf("second sync: %v", err)
	}

	sess, err := s.GetSession("sess-pi")
	if err != nil {
		t.Fatalf("GetSession: %v", err)
	}
	if sess.TotalInput != 100 || sess.TotalOutput != 50 {
		t.Fatalf("double-counted tokens: input=%d output=%d (want 100/50)", sess.TotalInput, sess.TotalOutput)
	}
	if sess.Host != "pi" {
		t.Fatalf("expected host 'pi', got %q", sess.Host)
	}
}

func TestSync_TailscaleUnavailableSkips(t *testing.T) {
	d := fakeDiscoverer{err: ErrTailscaleUnavailable}
	syncer, _, _ := newTestSyncer(t, d, fakePuller{})

	report, err := syncer.Sync()
	if err != nil {
		t.Fatalf("unavailable tailscale should not error: %v", err)
	}
	if !report.Skipped {
		t.Fatalf("expected Skipped report, got %+v", report)
	}
}
