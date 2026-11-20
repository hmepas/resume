package opencode

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/hmepas/resume/internal/adapters/common"
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

func writeDB(t *testing.T, stmts ...string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "open code.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatal(err)
		}
	}
	return path
}

func TestDBSessionsV2Schema(t *testing.T) {
	path := writeDB(t,
		`create table session_v2 (id text primary key, project_id text not null, directory text not null, title text, time_created integer not null, time_updated integer not null)`,
		`insert into session_v2 values ('ses_a', 'p', '/repo', 'Hello', 1783426313404, 1783426316402)`,
		`insert into session_v2 values ('ses_b', 'p', '/repo', null, 1783426313404, 1783426319000)`,
	)

	sessions, err := dbSessions(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 2 {
		t.Fatalf("len = %d, want 2", len(sessions))
	}
	if sessions[0].ID != "ses_b" || sessions[0].Title != "" {
		t.Fatalf("sessions[0] = %+v, want ses_b with empty title", sessions[0])
	}
	if sessions[1].ID != "ses_a" || sessions[1].Title != "Hello" || sessions[1].Project != "/repo" {
		t.Fatalf("sessions[1] = %+v", sessions[1])
	}
}

func TestDBSessionsLegacySchema(t *testing.T) {
	path := writeDB(t,
		`create table session (id text primary key, directory text not null, title text not null, time_created integer not null, time_updated integer not null)`,
		`insert into session values ('ses_a', '/repo', 'Hello', 1783426313404, 1783426316402)`,
	)

	sessions, err := dbSessions(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 1 || sessions[0].ID != "ses_a" {
		t.Fatalf("sessions = %+v", sessions)
	}
}

func TestDBSessionsUnknownSchema(t *testing.T) {
	path := writeDB(t, `create table other (id text)`)

	sessions, err := dbSessions(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 0 {
		t.Fatalf("sessions = %+v, want none", sessions)
	}
}

func TestSQLiteURIIsReadOnly(t *testing.T) {
	path := writeDB(t, `create table t (id text)`)

	db, err := sql.Open("sqlite", common.SQLiteURI(path, "mode=ro"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`insert into t values ('x')`); err == nil {
		t.Fatal("write succeeded on read-only connection")
	}
}
