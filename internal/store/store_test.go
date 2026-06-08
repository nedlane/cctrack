package store

import (
	"path/filepath"
	"testing"
)

// TestMigrateIdempotent ensures opening the same database twice does not fail
// on the additive `ALTER TABLE ... ADD COLUMN host` migration.
func TestMigrateIdempotent(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")

	s1, err := Open(dbPath)
	if err != nil {
		t.Fatalf("first open: %v", err)
	}
	if err := s1.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	s2, err := Open(dbPath)
	if err != nil {
		t.Fatalf("second open (migration must be idempotent): %v", err)
	}
	defer s2.Close()
}

// TestHostColumnRoundTrip verifies the host column is written and grouped.
func TestHostColumnRoundTrip(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer s.Close()

	if err := s.UpsertSession(SessionDelta{
		ID: "s1", Host: "pi", Model: "claude-sonnet-4-5",
		Timestamp: "2026-06-08T10:00:00Z", DeltaInput: 100, DeltaCost: 1.5,
	}); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	sess, err := s.GetSession("s1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if sess.Host != "pi" {
		t.Fatalf("expected host 'pi', got %q", sess.Host)
	}

	hosts, err := s.GetHostBreakdown()
	if err != nil {
		t.Fatalf("breakdown: %v", err)
	}
	if len(hosts) != 1 || hosts[0].Host != "pi" || hosts[0].SessionCount != 1 {
		t.Fatalf("unexpected breakdown: %+v", hosts)
	}
}
