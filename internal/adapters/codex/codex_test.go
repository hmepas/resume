package codex

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/hmepas/resume/internal/resume"
)

func TestGuardianFilteringAndDeletion(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	write := func(id, owner, parent, source, dir string) string {
		t.Helper()
		p := filepath.Join(home, ".codex", dir, id+".jsonl")
		if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
			t.Fatal(err)
		}
		obj := map[string]any{"type": "session_meta", "payload": map[string]any{"id": id, "session_id": owner, "parent_thread_id": parent, "thread_source": source, "cwd": "/project"}}
		b, _ := json.Marshal(obj)
		if err := os.WriteFile(p, append(b, '\n'), 0600); err != nil {
			t.Fatal(err)
		}
		return p
	}
	root := write("root", "root", "", "user", "sessions")
	direct := write("review", "root", "root", "guardian_review", "sessions")
	child := write("child", "root", "root", "subagent", "sessions")
	nested := write("nested", "", "child", "guardian_review", "archived_sessions")
	unrelated := write("other", "elsewhere", "elsewhere", "guardian_review", "sessions")
	orphan := write("orphan", "missing", "missing", "guardian_review", "sessions")
	sessions, err := (Adapter{}).Sessions(resume.Context{All: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 2 {
		t.Fatalf("visible sessions: %#v", sessions)
	}
	if err := DeleteSession(resume.Session{ID: "root", SourcePath: root}); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{root, direct, nested} {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Errorf("not deleted: %s (%v)", p, err)
		}
	}
	for _, p := range []string{child, unrelated, orphan} {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("not retained: %s (%v)", p, err)
		}
	}
}

func TestLegacyGuardianHidden(t *testing.T) {
	p := filepath.Join(t.TempDir(), "legacy.jsonl")
	err := os.WriteFile(p, []byte(`{"type":"session_meta","payload":{"id":"review","cwd":"/project","source":{"subagent":{"other":"guardian"}}}}`), 0600)
	if err != nil {
		t.Fatal(err)
	}
	if s := parseSession(p, nil); s.SourcePath != "" {
		t.Fatalf("guardian visible: %#v", s)
	}
}
