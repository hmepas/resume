package tui

import (
	"errors"
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
		adapters: resume.NewRegistry([]resume.Adapter{&fakeDeleter{fakeAdapter: fakeAdapter{id: "claude"}}}),
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

type fakeAdapter struct {
	id    string
	color resume.Color
}

func (a fakeAdapter) ID() string                                      { return a.id }
func (a fakeAdapter) Color() resume.Color                             { return a.color }
func (fakeAdapter) Sessions(resume.Context) ([]resume.Session, error) { return nil, nil }
func (fakeAdapter) Command(resume.Session) (resume.Command, error) {
	return resume.Command{}, nil
}

type fakeDeleter struct {
	fakeAdapter
	err     error
	deleted []resume.Session
}

func (*fakeDeleter) DeleteScope() string { return "fake scope" }

func (d *fakeDeleter) Delete(session resume.Session) error {
	d.deleted = append(d.deleted, session)
	return d.err
}

func TestAskDeleteOpensConfirmationForDeleter(t *testing.T) {
	session := resume.Session{Agent: "fake", ID: "abc"}
	p := &Picker{
		sessions: []resume.Session{session},
		filtered: []resume.Session{session},
		adapters: resume.NewRegistry([]resume.Adapter{&fakeDeleter{fakeAdapter: fakeAdapter{id: "fake"}}}),
	}

	p.askDelete()
	if p.confirm == nil || *p.confirm != session {
		t.Fatalf("confirm = %v, want %#v", p.confirm, session)
	}
	if p.status != "delete fake fake scope? y/N" {
		t.Fatalf("status = %q", p.status)
	}
}

func TestAskDeleteRefusesNonDeleter(t *testing.T) {
	session := resume.Session{Agent: "plain", ID: "abc"}
	p := &Picker{
		sessions: []resume.Session{session},
		filtered: []resume.Session{session},
		adapters: resume.NewRegistry([]resume.Adapter{fakeAdapter{id: "plain"}}),
	}

	p.askDelete()
	if p.confirm != nil {
		t.Fatal("askDelete opened confirmation for non-deleter agent")
	}
	if p.status != "cannot delete: plain sessions cannot be deleted" {
		t.Fatalf("status = %q", p.status)
	}
}

func TestDeleteConfirmedDeletesThroughAdapter(t *testing.T) {
	deleter := &fakeDeleter{fakeAdapter: fakeAdapter{id: "fake"}}
	deleted := resume.Session{Agent: "fake", ID: "one", SourcePath: "/tmp/one"}
	kept := resume.Session{Agent: "fake", ID: "two", SourcePath: "/tmp/two"}
	p := &Picker{
		sessions: []resume.Session{deleted, kept},
		filtered: []resume.Session{deleted, kept},
		confirm:  &deleted,
		adapters: resume.NewRegistry([]resume.Adapter{deleter}),
	}

	p.deleteConfirmed()
	if len(deleter.deleted) != 1 || deleter.deleted[0] != deleted {
		t.Fatalf("Delete calls = %#v, want once with %#v", deleter.deleted, deleted)
	}
	if len(p.sessions) != 1 || p.sessions[0] != kept {
		t.Fatalf("sessions after delete = %#v, want only %#v", p.sessions, kept)
	}
	if p.status != "deleted fake session" {
		t.Fatalf("status = %q", p.status)
	}
}

func TestDeleteConfirmedKeepsSessionOnError(t *testing.T) {
	deleter := &fakeDeleter{fakeAdapter: fakeAdapter{id: "fake"}, err: errors.New("boom")}
	session := resume.Session{Agent: "fake", ID: "one", SourcePath: "/tmp/one"}
	p := &Picker{
		sessions: []resume.Session{session},
		filtered: []resume.Session{session},
		confirm:  &session,
		adapters: resume.NewRegistry([]resume.Adapter{deleter}),
	}

	p.deleteConfirmed()
	if len(p.sessions) != 1 || p.sessions[0] != session {
		t.Fatalf("sessions after failed delete = %#v, want %#v", p.sessions, session)
	}
	if p.status != "delete failed: boom" {
		t.Fatalf("status = %q", p.status)
	}
}

func TestAgentColorCodeComesFromAdapter(t *testing.T) {
	p := &Picker{adapters: resume.NewRegistry([]resume.Adapter{fakeAdapter{id: "fake", color: resume.ColorCyan}})}

	if got := p.agentColorCode("fake"); got != "36" {
		t.Fatalf("fake color = %q, want 36", got)
	}
	if got := p.agentColorCode("unknown"); got != "" {
		t.Fatalf("unknown color = %q, want empty", got)
	}
}

func TestAgentWidthFitsLongestID(t *testing.T) {
	adapters := resume.NewRegistry([]resume.Adapter{
		fakeAdapter{id: "pi"},
		fakeAdapter{id: "longest-id"},
		fakeAdapter{id: "medium"},
	})

	if got := agentWidth(adapters); got != len("longest-id") {
		t.Fatalf("agentWidth = %d, want %d", got, len("longest-id"))
	}
}
