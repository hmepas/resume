package tui

import (
	"testing"

	"github.com/hmepas/resume/internal/resume"
)

func TestCtrlDAsksDeleteInSearchMode(t *testing.T) {
	session := resume.Session{
		Agent:      "claude",
		ID:         "abc",
		SourcePath: "/tmp/session.jsonl",
	}
	p := &Picker{
		sessions: []resume.Session{session},
		filtered: []resume.Session{session},
		search:   true,
		query:    "cla",
	}

	if got := p.handle([]byte{4}); got != "render" {
		t.Fatalf("handle(ctrl-d) = %q, want render", got)
	}
	if p.confirm == nil {
		t.Fatal("handle(ctrl-d) did not open delete confirmation")
	}
	if p.query != "cla" {
		t.Fatalf("query = %q, want unchanged", p.query)
	}
}

func TestDTypesInSearchMode(t *testing.T) {
	session := resume.Session{Agent: "claude", SourcePath: "/tmp/session.jsonl"}
	p := &Picker{
		sessions: []resume.Session{session},
		filtered: []resume.Session{session},
		search:   true,
	}

	if got := p.handle([]byte("d")); got != "render" {
		t.Fatalf("handle(d) = %q, want render", got)
	}
	if p.confirm != nil {
		t.Fatal("handle(d) opened delete confirmation in search mode")
	}
	if p.query != "d" {
		t.Fatalf("query = %q, want d", p.query)
	}
}

func TestOpenCodeDatabaseSessionCanBeDeleted(t *testing.T) {
	session := resume.Session{Agent: "opencode", ID: "ses_123", SourcePath: "/tmp/opencode.db"}
	p := &Picker{
		sessions: []resume.Session{session},
		filtered: []resume.Session{session},
	}

	p.askDelete()
	if p.confirm != nil {
		if *p.confirm != session {
			t.Fatalf("confirm = %#v, want %#v", *p.confirm, session)
		}
	} else {
		t.Fatal("askDelete did not open confirmation for opencode database session")
	}
	if p.status != "delete opencode session? y/N" {
		t.Fatalf("status = %q", p.status)
	}
}

func TestOpenCodeDeleteUsesSessionID(t *testing.T) {
	called := ""
	original := runOpenCodeSessionDelete
	runOpenCodeSessionDelete = func(sessionID string) error {
		called = sessionID
		return nil
	}
	defer func() { runOpenCodeSessionDelete = original }()

	deleted := resume.Session{Agent: "opencode", ID: "ses_123", SourcePath: "/tmp/opencode.db"}
	kept := resume.Session{Agent: "opencode", ID: "ses_456", SourcePath: "/tmp/opencode.db"}
	p := &Picker{
		sessions: []resume.Session{deleted, kept},
		filtered: []resume.Session{deleted, kept},
		confirm:  &deleted,
	}

	p.deleteConfirmed()
	if called != "ses_123" {
		t.Fatalf("deleted session id = %q, want ses_123", called)
	}
	if len(p.sessions) != 1 || p.sessions[0] != kept {
		t.Fatalf("sessions after delete = %#v, want only %#v", p.sessions, kept)
	}
	if p.status != "deleted opencode session" {
		t.Fatalf("status = %q", p.status)
	}
}

func TestCtrlJAndCtrlKMoveSelection(t *testing.T) {
	sessions := []resume.Session{
		{Agent: "claude", SourcePath: "/tmp/one.jsonl"},
		{Agent: "codex", SourcePath: "/tmp/two.jsonl"},
	}
	p := &Picker{
		sessions: sessions,
		filtered: sessions,
	}

	if got := p.handle([]byte{10}); got != "render" {
		t.Fatalf("handle(ctrl-j) = %q, want render", got)
	}
	if p.selected != 1 {
		t.Fatalf("selected after ctrl-j = %d, want 1", p.selected)
	}
	if got := p.handle([]byte{11}); got != "render" {
		t.Fatalf("handle(ctrl-k) = %q, want render", got)
	}
	if p.selected != 0 {
		t.Fatalf("selected after ctrl-k = %d, want 0", p.selected)
	}
}

func TestAgentColorCodeIsStable(t *testing.T) {
	if agentColorCode("claude") != "31" {
		t.Fatalf("claude color = %q, want 31", agentColorCode("claude"))
	}
	if agentColorCode("codex") != "32" {
		t.Fatalf("codex color = %q, want 32", agentColorCode("codex"))
	}
	if agentColorCode("prime-agent") != "95" {
		t.Fatalf("prime-agent color = %q, want 95", agentColorCode("prime-agent"))
	}
	if agentColorCode("gemini") == agentColorCode("prime-agent") {
		t.Fatal("gemini and prime-agent share a color")
	}
	if agentColorCode("agy") != "96" {
		t.Fatalf("agy color = %q, want 96", agentColorCode("agy"))
	}
	for _, other := range []string{"pi", "gemini", "prime-agent"} {
		if agentColorCode("agy") == agentColorCode(other) {
			t.Fatalf("agy and %s share a color", other)
		}
	}
}

func TestEscapePreservesThenClearsSearchThenQuits(t *testing.T) {
	sessions := []resume.Session{{ID: "one", Title: "match one"}, {ID: "two", Title: "match two"}, {ID: "other", Title: "other"}}
	p := &Picker{sessions: sessions, filtered: sessions[:2], query: "match", search: true, selected: 1, height: 24}
	if got := p.handle([]byte{27}); got != "render" || p.search || p.query != "match" || len(p.filtered) != 2 || p.selected != 1 {
		t.Fatalf("first escape: action=%q picker=%+v", got, p)
	}
	p.handle([]byte("k"))
	if p.selected != 0 || p.query != "match" {
		t.Fatal("navigation did not preserve search")
	}
	if got := p.handle([]byte{27}); got != "render" || p.query != "" || len(p.filtered) != 3 {
		t.Fatalf("second escape: action=%q picker=%+v", got, p)
	}
	if got := p.handle([]byte{27}); got != "cancel" {
		t.Fatalf("third escape = %q", got)
	}
}

func TestAgySessionCanBeDeleted(t *testing.T) {
	session := resume.Session{Agent: "agy", ID: "conv_123", SourcePath: "/tmp/conv_123.pb"}
	p := &Picker{
		sessions: []resume.Session{session},
		filtered: []resume.Session{session},
	}

	p.askDelete()
	if p.confirm == nil {
		t.Fatal("askDelete did not open confirmation for agy session")
	}
	if *p.confirm != session {
		t.Fatalf("confirm = %#v, want %#v", *p.confirm, session)
	}
	if p.status != "delete agy session? y/N" {
		t.Fatalf("status = %q", p.status)
	}

	empty := resume.Session{Agent: "agy", SourcePath: "/tmp/conv.pb"}
	p = &Picker{
		sessions: []resume.Session{empty},
		filtered: []resume.Session{empty},
	}
	p.askDelete()
	if p.confirm != nil {
		t.Fatal("askDelete opened confirmation for agy session without id")
	}
	if p.status != "cannot delete: agy session id is empty" {
		t.Fatalf("status = %q", p.status)
	}
}

func TestAgyDeleteUsesSessionID(t *testing.T) {
	var called resume.Session
	calls := 0
	original := runAgySessionDelete
	runAgySessionDelete = func(session resume.Session) error {
		called = session
		calls++
		return nil
	}
	defer func() { runAgySessionDelete = original }()

	deleted := resume.Session{Agent: "agy", ID: "conv_123", SourcePath: "/tmp/conv_123.pb"}
	kept := resume.Session{Agent: "agy", ID: "conv_456", SourcePath: "/tmp/conv_456.pb"}
	p := &Picker{
		sessions: []resume.Session{deleted, kept},
		filtered: []resume.Session{deleted, kept},
		confirm:  &deleted,
	}

	p.deleteConfirmed()
	if calls != 1 || called != deleted {
		t.Fatalf("delete called %d times with %#v, want once with %#v", calls, called, deleted)
	}
	if len(p.sessions) != 1 || p.sessions[0] != kept {
		t.Fatalf("sessions after delete = %#v, want only %#v", p.sessions, kept)
	}
	if p.status != "deleted agy session" {
		t.Fatalf("status = %q", p.status)
	}
}
