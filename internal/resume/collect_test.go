package resume

import (
	"context"
	"errors"
	"testing"
)

func TestDedupeUsesSessionIDWhenAvailable(t *testing.T) {
	sessions := []Session{
		{Agent: "opencode", ID: "ses_one", SourcePath: "/tmp/opencode.db"},
		{Agent: "opencode", ID: "ses_two", SourcePath: "/tmp/opencode.db"},
		{Agent: "opencode", ID: "ses_one", SourcePath: "/tmp/opencode.db"},
	}

	got := dedupe(sessions)
	if len(got) != 2 {
		t.Fatalf("len(dedupe) = %d, want 2", len(got))
	}
	if got[0].ID != "ses_one" || got[1].ID != "ses_two" {
		t.Fatalf("dedupe IDs = %q, %q; want ses_one, ses_two", got[0].ID, got[1].ID)
	}
}

type fakeAdapter struct {
	sessions []Session
}

func (fakeAdapter) ID() string                            { return "fake" }
func (fakeAdapter) Color() Color                          { return ColorRed }
func (a fakeAdapter) Sessions(Context) ([]Session, error) { return a.sessions, nil }
func (fakeAdapter) Command(session Session) (Command, error) {
	if session.ID == "" {
		return Command{}, errors.New("fake session has no resume id")
	}
	return Command{Name: "fake", Args: []string{"--resume", session.ID}, Dir: "/repo"}, nil
}

func TestCollectDerivesResumeHint(t *testing.T) {
	adapter := fakeAdapter{sessions: []Session{
		{Agent: "fake", ID: "abc", SourcePath: "/tmp/a"},
		{Agent: "fake", SourcePath: "/tmp/b"},
	}}

	sessions, diagnostics := Collect(context.Background(), []Adapter{adapter}, CollectOptions{All: true})
	if len(diagnostics) != 0 {
		t.Fatalf("diagnostics = %v, want none", diagnostics)
	}
	hints := map[string]string{}
	for _, session := range sessions {
		hints[session.SourcePath] = session.ResumeHint
	}
	if got, want := hints["/tmp/a"], "fake --resume abc"; got != want {
		t.Fatalf("ResumeHint = %q, want %q", got, want)
	}
	if got := hints["/tmp/b"]; got != "" {
		t.Fatalf("ResumeHint = %q, want empty", got)
	}
}
