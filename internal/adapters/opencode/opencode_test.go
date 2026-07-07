package opencode

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseSessionFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ses_123.json")
	data := `{
  "id": "ses_123",
  "projectID": "abc",
  "directory": "/repo",
  "title": "New session - 2026-07-07T12:11:53.404Z",
  "time": {
    "created": 1783426313404,
    "updated": 1783426316402
  }
}`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}

	session := parseSessionFile(path, nil)
	if session.Agent != "opencode" {
		t.Fatalf("Agent = %q, want opencode", session.Agent)
	}
	if session.ID != "ses_123" {
		t.Fatalf("ID = %q, want ses_123", session.ID)
	}
	if session.Project != "/repo" {
		t.Fatalf("Project = %q, want /repo", session.Project)
	}
	if session.ResumeHint != "opencode -s ses_123" {
		t.Fatalf("ResumeHint = %q, want opencode -s ses_123", session.ResumeHint)
	}
}

func TestParseSessionFileUsesProjectWorktree(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ses_123.json")
	data := `{
  "id": "ses_123",
  "projectID": "abc",
  "title": "New session",
  "time": {
    "created": 1783426313404,
    "updated": 1783426316402
  }
}`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}

	session := parseSessionFile(path, map[string]string{"abc": "/repo"})
	if session.Project != "/repo" {
		t.Fatalf("Project = %q, want /repo", session.Project)
	}
}
