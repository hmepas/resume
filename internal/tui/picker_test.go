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

func TestOpenCodeDatabaseSessionCannotBeDeleted(t *testing.T) {
	session := resume.Session{Agent: "opencode", ID: "ses_123", SourcePath: "/tmp/opencode.db"}
	p := &Picker{
		sessions: []resume.Session{session},
		filtered: []resume.Session{session},
	}

	p.askDelete()
	if p.confirm != nil {
		t.Fatal("askDelete opened confirmation for opencode database session")
	}
	if p.status != "cannot delete: OpenCode sessions are stored in opencode.db" {
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
}
