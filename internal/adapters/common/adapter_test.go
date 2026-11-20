package common

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/hmepas/resume/internal/resume"
)

func TestResumeCommand(t *testing.T) {
	args := make([]string, 1, 4)
	args[0] = "--resume"
	session := resume.Session{Agent: "claude", ID: "abc", Project: "/repo"}

	got, err := ResumeCommand(session, args...)
	if err != nil {
		t.Fatal(err)
	}
	want := resume.Command{Name: "claude", Args: []string{"--resume", "abc"}, Dir: "/repo"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ResumeCommand() = %+v, want %+v", got, want)
	}
	if extended := args[:2]; extended[1] != "" {
		t.Fatalf("ResumeCommand() wrote into caller's args backing array: %q", extended)
	}
}

func TestResumeCommandRequiresID(t *testing.T) {
	_, err := ResumeCommand(resume.Session{Agent: "codex", Project: "/repo"}, "resume")
	if err == nil || err.Error() != "codex session has no resume id" {
		t.Fatalf("ResumeCommand() error = %v, want %q", err, "codex session has no resume id")
	}
}

func TestFileDeleterRemovesSourceFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.jsonl")
	if err := os.WriteFile(path, []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := (FileDeleter{}).Delete(resume.Session{SourcePath: path}); err != nil {
		t.Fatal(err)
	}
	if Exists(path) {
		t.Fatal("Delete() left the session file in place")
	}
}
