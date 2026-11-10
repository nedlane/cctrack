package parser

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nedlane/cctrack/internal/store"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func openTestStore(t *testing.T) *store.Store {
	t.Helper()
	s, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

const sessID = "2ae8a045-b14b-41e5-aeea-6a9e21e13b5f"

func event(reqID, ts, model string, input, output int64) string {
	return `{"type":"assistant","sessionId":"` + sessID + `","requestId":"` + reqID + `","timestamp":"` + ts + `","message":{"role":"assistant","model":"` + model + `","usage":{"input_tokens":` + itoa(input) + `,"output_tokens":` + itoa(output) + `}}}` + "\n"
}

func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

// A subagent transcript running on a different model must not override the
// session's model, which should reflect the main transcript.
func TestSubagentModelDoesNotOverrideSessionModel(t *testing.T) {
	s := openTestStore(t)
	p := New(s)

	logDir := t.TempDir()
	projDir := filepath.Join(logDir, "-home-u-projects-myproj")
	mainFile := filepath.Join(projDir, sessID+".jsonl")
	subFile := filepath.Join(projDir, sessID, "subagents", "agent-abc123.jsonl")

	writeFile(t, mainFile, event("req_main1", "2026-07-04T10:00:00.000Z", "claude-fable-5", 100, 50))
	writeFile(t, subFile, event("req_sub1", "2026-07-04T10:05:00.000Z", "claude-opus-4-8", 200, 80))

	// Main transcript first, subagent after — the order that clobbers the model.
	if _, err := p.ParseFile(mainFile); err != nil {
		t.Fatal(err)
	}
	if _, err := p.ParseFile(subFile); err != nil {
		t.Fatal(err)
	}

	sess, err := s.GetSession(sessID)
	if err != nil {
		t.Fatal(err)
	}
	if sess.Model != "claude-fable-5" {
		t.Errorf("session model = %q, want claude-fable-5 (main transcript model)", sess.Model)
	}
	// Subagent usage still counts toward the session totals.
	if sess.TotalInput != 300 || sess.TotalOutput != 130 {
		t.Errorf("session totals = %d in / %d out, want 300 / 130", sess.TotalInput, sess.TotalOutput)
	}
}

// Within the main transcript, the chronologically latest model wins,
// regardless of map iteration order.
func TestSessionModelIsLatestByTimestamp(t *testing.T) {
	s := openTestStore(t)
	p := New(s)

	logDir := t.TempDir()
	mainFile := filepath.Join(logDir, "-home-u-projects-myproj", sessID+".jsonl")

	var content string
	// Many earlier sonnet events followed by one later fable event; with
	// map-order model selection this fails often, with timestamp selection never.
	for i := int64(0); i < 20; i++ {
		content += event("req_s"+itoa(i+1), "2026-07-04T09:00:00.000Z", "claude-sonnet-4-6", 10, 5)
	}
	content += event("req_last", "2026-07-04T11:00:00.000Z", "claude-fable-5", 10, 5)
	writeFile(t, mainFile, content)

	if _, err := p.ParseFile(mainFile); err != nil {
		t.Fatal(err)
	}

	sess, err := s.GetSession(sessID)
	if err != nil {
		t.Fatal(err)
	}
	if sess.Model != "claude-fable-5" {
		t.Errorf("session model = %q, want claude-fable-5 (latest event)", sess.Model)
	}
}

// Synthetic error events must never become the session model.
func TestSyntheticModelIgnored(t *testing.T) {
	s := openTestStore(t)
	p := New(s)

	logDir := t.TempDir()
	mainFile := filepath.Join(logDir, "-home-u-projects-myproj", sessID+".jsonl")
	writeFile(t, mainFile,
		event("req_1", "2026-07-04T10:00:00.000Z", "claude-fable-5", 100, 50)+
			event("req_2", "2026-07-04T10:10:00.000Z", "<synthetic>", 1, 1))

	if _, err := p.ParseFile(mainFile); err != nil {
		t.Fatal(err)
	}

	sess, err := s.GetSession(sessID)
	if err != nil {
		t.Fatal(err)
	}
	if sess.Model != "claude-fable-5" {
		t.Errorf("session model = %q, want claude-fable-5", sess.Model)
	}
}

// Mixed-model files should cost each request at its own model's rates,
// not the whole file at the last-seen model's rates.
func TestMixedModelCostUsesPerRequestRates(t *testing.T) {
	s := openTestStore(t)
	p := New(s)

	logDir := t.TempDir()
	mainFile := filepath.Join(logDir, "-home-u-projects-myproj", sessID+".jsonl")
	// 1M output tokens on fable ($50/M) + 1M output tokens on haiku ($5/M).
	writeFile(t, mainFile,
		event("req_f", "2026-07-04T10:00:00.000Z", "claude-fable-5", 0, 1_000_000)+
			event("req_h", "2026-07-04T10:01:00.000Z", "claude-haiku-4-5-20251001", 0, 1_000_000))

	if _, err := p.ParseFile(mainFile); err != nil {
		t.Fatal(err)
	}

	sess, err := s.GetSession(sessID)
	if err != nil {
		t.Fatal(err)
	}
	want := 55.0
	if sess.TotalCost < want-0.001 || sess.TotalCost > want+0.001 {
		t.Errorf("session cost = %f, want %f (per-request rates)", sess.TotalCost, want)
	}
}
