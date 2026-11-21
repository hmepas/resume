package agy

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/hmepas/resume/internal/resume"
	_ "modernc.org/sqlite"
)

type row struct {
	id, title, preview string
	steps              int
	modified           string
	uris               string
	parent             string
	depth              int
	userInput          string
}

func createDB(t *testing.T, root string, wal bool, rows []row) string {
	t.Helper()
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	dbPath := filepath.Join(root, summariesDB)
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if wal {
		if _, err := db.Exec(`PRAGMA journal_mode=WAL`); err != nil {
			t.Fatal(err)
		}
	}
	stmts := []string{
		`CREATE TABLE conversation_summaries (
			conversation_id TEXT PRIMARY KEY,
			title TEXT,
			preview TEXT,
			step_count INTEGER,
			last_modified_time TIMESTAMP,
			workspace_uris TEXT,
			parent_conversation_id TEXT,
			nesting_depth INTEGER,
			last_user_input_time TIMESTAMP)`,
		`CREATE TABLE conversations_fts (conversation_id TEXT, body TEXT)`,
		`CREATE TABLE conversation_parents (conversation_id TEXT, parent_id TEXT)`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			t.Fatal(err)
		}
	}
	for _, r := range rows {
		if _, err := db.Exec(`INSERT INTO conversation_summaries VALUES (?,?,?,?,?,?,?,?,?)`,
			r.id, r.title, r.preview, r.steps, r.modified, r.uris, r.parent, r.depth, r.userInput); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(`INSERT INTO conversations_fts VALUES (?, 'text')`, r.id); err != nil {
			t.Fatal(err)
		}
		if r.parent != "" {
			if _, err := db.Exec(`INSERT INTO conversation_parents VALUES (?, ?)`, r.id, r.parent); err != nil {
				t.Fatal(err)
			}
		}
	}
	return dbPath
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func transcriptPath(root, id string) string {
	return filepath.Join(root, "brain", id, ".system_generated", "logs", "transcript.jsonl")
}

func byID(sessions []resume.Session) map[string]resume.Session {
	m := map[string]resume.Session{}
	for _, s := range sessions {
		m[s.ID] = s
	}
	return m
}

func TestLoadSessionsParsesAndFilters(t *testing.T) {
	tmp := t.TempDir()
	root := filepath.Join(tmp, "data")
	repo := filepath.Join(tmp, "my repo")
	other := filepath.Join(tmp, "other")
	for _, d := range []string{repo, other} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	repoURI := "file://" + strings.ReplaceAll(filepath.ToSlash(repo), " ", "%20")
	otherURI := "file://" + filepath.ToSlash(other)

	dbPath := createDB(t, root, true, []row{
		{id: "main", title: " Main title ", preview: "prev", steps: 5,
			modified: "2026-11-20T10:00:00Z", uris: `["` + otherURI + `","` + repoURI + `"]`,
			userInput: "2026-11-20T09:00:00Z"},
		{id: "otheronly", title: "Other", steps: 3, modified: "2026-11-19 10:00:00",
			uris: `["` + otherURI + `"]`},
		{id: "child", title: "Sub", steps: 3, modified: "2026-11-20T11:00:00Z",
			uris: `["` + repoURI + `"]`, parent: "main", depth: 1},
		{id: "deep", title: "Deep", steps: 3, modified: "2026-11-20T11:00:00Z",
			uris: `["` + repoURI + `"]`, depth: 2},
		{id: "empty", title: "", steps: 0, modified: "0001-01-01T00:00:00Z",
			uris: `["` + repoURI + `"]`},
		{id: "nows", title: "No workspace", steps: 2, modified: "2026-11-20T11:00:00Z", uris: `[]`},
		{id: "plainpath", preview: "From preview", steps: 1, modified: "0001-01-01 00:00:00+00:00",
			uris: `["relative/path","` + filepath.ToSlash(repo) + `"]`, userInput: "2026-11-18T08:00:00Z"},
	})

	ctx := resume.Context{Project: resume.Project{Path: repo, Root: repo}}
	sessions, err := loadSessions(root, ctx)
	if err != nil {
		t.Fatal(err)
	}
	got := byID(sessions)
	if len(got) != 2 {
		t.Fatalf("want main and plainpath, got %+v", sessions)
	}
	main := got["main"]
	if main.Project != repo {
		t.Errorf("project = %q, want %q", main.Project, repo)
	}
	if main.Title != "Main title" {
		t.Errorf("title = %q", main.Title)
	}
	if main.Agent != "agy" || main.Confidence != "high" {
		t.Errorf("unexpected fields %+v", main)
	}
	if !main.UpdatedAt.Equal(time.Date(2026, 11, 20, 10, 0, 0, 0, time.UTC)) {
		t.Errorf("updated = %v", main.UpdatedAt)
	}
	if !main.StartedAt.Equal(time.Date(2026, 11, 20, 9, 0, 0, 0, time.UTC)) {
		t.Errorf("started = %v", main.StartedAt)
	}
	if main.SourcePath != dbPath {
		t.Errorf("source = %q", main.SourcePath)
	}

	pp := got["plainpath"]
	if pp.Title != "From preview" {
		t.Errorf("plainpath title = %q", pp.Title)
	}
	// Zero-time sentinel falls back to last user input.
	if !pp.UpdatedAt.Equal(time.Date(2026, 11, 18, 8, 0, 0, 0, time.UTC)) {
		t.Errorf("plainpath updated = %v", pp.UpdatedAt)
	}

	all, err := loadSessions(root, resume.Context{All: true})
	if err != nil {
		t.Fatal(err)
	}
	gotAll := byID(all)
	if len(gotAll) != 3 {
		t.Fatalf("all: want 3 sessions, got %+v", all)
	}
	if gotAll["main"].Project != other {
		t.Errorf("all: main project = %q, want first workspace %q", gotAll["main"].Project, other)
	}
	if _, ok := gotAll["otheronly"]; !ok {
		t.Error("all: missing otheronly")
	}
	for _, skipped := range []string{"child", "deep", "empty", "nows"} {
		if _, ok := gotAll[skipped]; ok {
			t.Errorf("all: %s should be filtered", skipped)
		}
	}
}

func TestLoadSessionsSourcePathAndTranscript(t *testing.T) {
	tmp := t.TempDir()
	root := filepath.Join(tmp, "data")
	repo := filepath.Join(tmp, "repo")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	uris := `["file://` + filepath.ToSlash(repo) + `"]`
	dbPath := createDB(t, root, false, []row{
		{id: "withtranscript", steps: 2, modified: "2026-11-20T10:00:00Z", uris: uris, userInput: "2026-11-20T09:00:00Z"},
		{id: "withdb", title: "DB", steps: 2, modified: "2026-11-20T10:00:00Z", uris: uris},
		{id: "summaryonly", title: "S", steps: 2, modified: "2026-11-20T10:00:00Z", uris: uris},
	})
	writeFile(t, transcriptPath(root, "withtranscript"),
		`{"step_index":0,"type":"USER_INPUT","created_at":"2026-11-20T08:30:00Z","content":"<USER_REQUEST>\n  Fix the bug  \n</USER_REQUEST>"}`+"\n"+
			`{"step_index":1,"type":"PLANNER_RESPONSE","content":"ok"}`+"\n")
	writeFile(t, filepath.Join(root, "conversations", "withtranscript.db"), "x")
	writeFile(t, filepath.Join(root, "conversations", "withdb.db"), "x")

	sessions, err := loadSessions(root, resume.Context{Project: resume.Project{Path: repo, Root: repo}})
	if err != nil {
		t.Fatal(err)
	}
	got := byID(sessions)
	wt := got["withtranscript"]
	if wt.SourcePath != transcriptPath(root, "withtranscript") {
		t.Errorf("source = %q", wt.SourcePath)
	}
	if wt.Title != "Fix the bug" {
		t.Errorf("title = %q", wt.Title)
	}
	if !wt.StartedAt.Equal(time.Date(2026, 11, 20, 8, 30, 0, 0, time.UTC)) {
		t.Errorf("started = %v", wt.StartedAt)
	}
	if got["withdb"].SourcePath != filepath.Join(root, "conversations", "withdb.db") {
		t.Errorf("withdb source = %q", got["withdb"].SourcePath)
	}
	if got["summaryonly"].SourcePath != dbPath {
		t.Errorf("summaryonly source = %q", got["summaryonly"].SourcePath)
	}
}

func TestLoadSessionsMissing(t *testing.T) {
	tmp := t.TempDir()
	for _, root := range []string{filepath.Join(tmp, "nope"), tmp} {
		sessions, err := loadSessions(root, resume.Context{All: true})
		if err != nil || sessions != nil {
			t.Errorf("root %s: got %v, %v", root, sessions, err)
		}
	}
}

func TestToTime(t *testing.T) {
	want := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	for _, v := range []any{want, "2026-01-02T03:04:05Z", []byte("2026-01-02 03:04:05"), "2026-01-02 03:04:05+00:00"} {
		if got := toTime(v); !got.Equal(want) {
			t.Errorf("toTime(%v) = %v", v, got)
		}
	}
	for _, v := range []any{nil, "", "garbage", "0001-01-01T00:00:00Z", time.Time{}} {
		if got := toTime(v); !got.IsZero() {
			t.Errorf("toTime(%v) = %v, want zero", v, got)
		}
	}
}

func countRows(t *testing.T, dbPath, query string, args ...any) int {
	t.Helper()
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var n int
	if err := db.QueryRow(query, args...).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestDeleteSessionInRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), "data")
	uris := `["file:///tmp/repo"]`
	dbPath := createDB(t, root, true, []row{
		{id: "target", steps: 2, modified: "2026-11-20T10:00:00Z", uris: uris},
		{id: "kid", steps: 2, modified: "2026-11-20T10:00:00Z", uris: uris, parent: "target", depth: 1},
		{id: "sibling", steps: 2, modified: "2026-11-20T10:00:00Z", uris: uris},
	})
	for _, id := range []string{"target", "kid", "sibling"} {
		for _, ext := range []string{".db", ".db-wal", ".db-shm"} {
			writeFile(t, filepath.Join(root, "conversations", id+ext), "x")
		}
		writeFile(t, filepath.Join(root, "annotations", id+".pbtxt"), "x")
		writeFile(t, transcriptPath(root, id), "{}\n")
	}

	if err := deleteSessionInRoot(root, resume.Session{ID: "target"}); err != nil {
		t.Fatal(err)
	}

	if n := countRows(t, dbPath, `SELECT COUNT(*) FROM conversation_summaries`); n != 1 {
		t.Errorf("summaries rows = %d, want 1", n)
	}
	if n := countRows(t, dbPath, `SELECT COUNT(*) FROM conversation_summaries WHERE conversation_id = 'sibling'`); n != 1 {
		t.Error("sibling row removed")
	}
	if n := countRows(t, dbPath, `SELECT COUNT(*) FROM conversations_fts WHERE conversation_id != 'sibling'`); n != 0 {
		t.Errorf("fts rows left = %d", n)
	}
	if n := countRows(t, dbPath, `SELECT COUNT(*) FROM conversation_parents`); n != 0 {
		t.Errorf("parents rows left = %d", n)
	}
	for _, id := range []string{"target", "kid"} {
		for _, p := range []string{
			filepath.Join(root, "conversations", id+".db"),
			filepath.Join(root, "conversations", id+".db-wal"),
			filepath.Join(root, "conversations", id+".db-shm"),
			filepath.Join(root, "annotations", id+".pbtxt"),
			filepath.Join(root, "brain", id),
		} {
			if _, err := os.Stat(p); !os.IsNotExist(err) {
				t.Errorf("%s still exists", p)
			}
		}
	}
	for _, p := range []string{
		filepath.Join(root, "conversations", "sibling.db"),
		filepath.Join(root, "annotations", "sibling.pbtxt"),
		transcriptPath(root, "sibling"),
	} {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("sibling artifact %s removed: %v", p, err)
		}
	}

	// Deleting again is a no-op.
	if err := deleteSessionInRoot(root, resume.Session{ID: "target"}); err != nil {
		t.Errorf("second delete: %v", err)
	}
}

func TestDeleteSessionWithoutDB(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "conversations", "x.db"), "x")
	writeFile(t, transcriptPath(root, "x"), "{}\n")
	if err := deleteSessionInRoot(root, resume.Session{ID: "x"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "brain", "x")); !os.IsNotExist(err) {
		t.Error("brain dir not removed")
	}
}

func TestDeleteSessionRejectsUnsafeIDs(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "keep"), "x")
	for _, id := range []string{"", ".", "..", "../escape", "a/b", `a\b`} {
		if err := deleteSessionInRoot(root, resume.Session{ID: id}); err == nil {
			t.Errorf("id %q accepted", id)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "keep")); err != nil {
		t.Error("root contents touched")
	}
}

func execDB(t *testing.T, dbPath string, stmts ...string) {
	t.Helper()
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			t.Fatal(err)
		}
	}
}

func TestDeleteSessionRefusesLiveConversation(t *testing.T) {
	root := filepath.Join(t.TempDir(), "data")
	dbPath := createDB(t, root, false, []row{
		{id: "live", steps: 2, modified: "2026-11-20T10:00:00Z", uris: `["file:///tmp/repo"]`},
	})
	writeFile(t, transcriptPath(root, "live"), "{}\n")
	lockPath := filepath.Join(root, "presence", "live.lock")
	writeFile(t, lockPath, "")

	// A running agy holds an exclusive flock on the presence file.
	holder, err := os.Open(lockPath)
	if err != nil {
		t.Fatal(err)
	}
	defer holder.Close()
	beforeInfo, err := holder.Stat()
	if err != nil {
		t.Fatal(err)
	}
	if err := syscall.Flock(int(holder.Fd()), syscall.LOCK_EX); err != nil {
		t.Fatal(err)
	}

	if err := deleteSessionInRoot(root, resume.Session{ID: "live"}); err == nil {
		t.Fatal("deleted a conversation that is open in a running agy")
	}
	if n := countRows(t, dbPath, `SELECT COUNT(*) FROM conversation_summaries`); n != 1 {
		t.Errorf("summaries rows = %d, want 1", n)
	}
	if _, err := os.Stat(transcriptPath(root, "live")); err != nil {
		t.Errorf("transcript removed: %v", err)
	}

	if err := syscall.Flock(int(holder.Fd()), syscall.LOCK_UN); err != nil {
		t.Fatal(err)
	}
	if err := deleteSessionInRoot(root, resume.Session{ID: "live"}); err != nil {
		t.Fatal(err)
	}
	if n := countRows(t, dbPath, `SELECT COUNT(*) FROM conversation_summaries`); n != 0 {
		t.Errorf("summaries rows = %d, want 0", n)
	}
	// The lockfile stays in place on the same inode so lockers never split
	// across an unlinked file, and its lock is released after deletion.
	afterInfo, err := os.Stat(lockPath)
	if err != nil {
		t.Fatalf("presence lock removed or unreadable: %v", err)
	}
	if !os.SameFile(beforeInfo, afterInfo) {
		t.Error("presence lock inode changed during deletion")
	}
	if err := syscall.Flock(int(holder.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		t.Errorf("presence lock still held after delete: %v", err)
	}
}

func TestDeleteSessionLocksMissingPresenceThroughCommit(t *testing.T) {
	root := filepath.Join(t.TempDir(), "data")
	createDB(t, root, false, []row{
		{id: "target", steps: 2, modified: "2026-11-20T10:00:00Z", uris: `["file:///tmp/repo"]`},
	})
	writeFile(t, transcriptPath(root, "target"), "{}\n")

	lockPath := filepath.Join(root, "presence", "target.lock")
	if _, err := os.Stat(lockPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("presence lock unexpectedly existed before delete: %v", err)
	}

	origCommit := commitTx
	t.Cleanup(func() { commitTx = origCommit })
	checked := false
	commitTx = func(tx *sql.Tx) error {
		checked = true
		probe, err := os.OpenFile(lockPath, os.O_RDWR, 0)
		if err != nil {
			t.Errorf("presence lock missing during commit: %v", err)
			return origCommit(tx)
		}
		defer probe.Close()
		if err := syscall.Flock(int(probe.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); !errors.Is(err, syscall.EWOULDBLOCK) {
			t.Errorf("concurrent flock during commit = %v, want EWOULDBLOCK", err)
		}
		return origCommit(tx)
	}

	if err := deleteSessionInRoot(root, resume.Session{ID: "target"}); err != nil {
		t.Fatal(err)
	}
	if !checked {
		t.Fatal("commitTx was not called")
	}
	probe, err := os.OpenFile(lockPath, os.O_RDWR, 0)
	if err != nil {
		t.Fatalf("presence lock missing after delete: %v", err)
	}
	defer probe.Close()
	if err := syscall.Flock(int(probe.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		t.Errorf("presence lock still held after delete: %v", err)
	}
}

func TestDeleteSessionKeepsRowsWhenFilesFail(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions")
	}
	root := filepath.Join(t.TempDir(), "data")
	dbPath := createDB(t, root, false, []row{
		{id: "target", steps: 2, modified: "2026-11-20T10:00:00Z", uris: `["file:///tmp/repo"]`},
	})
	convDB := filepath.Join(root, "conversations", "target.db")
	writeFile(t, convDB, "conv-data")
	writeFile(t, transcriptPath(root, "target"), "{}\n")
	logs := filepath.Dir(transcriptPath(root, "target"))
	if err := os.Chmod(logs, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(logs, 0o700) })

	if err := deleteSessionInRoot(root, resume.Session{ID: "target"}); err == nil {
		t.Fatal("delete succeeded although the transcript could not be removed")
	}
	// The row and already-staged artifacts stay intact so the session can be retried.
	if n := countRows(t, dbPath, `SELECT COUNT(*) FROM conversation_summaries`); n != 1 {
		t.Fatalf("summaries rows = %d, want 1", n)
	}
	if _, err := os.Stat(convDB); err != nil {
		t.Fatalf("conversation DB was not restored: %v", err)
	}
	if _, err := os.Stat(transcriptPath(root, "target")); err != nil {
		t.Fatalf("transcript was not preserved: %v", err)
	}

	if err := os.Chmod(logs, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := deleteSessionInRoot(root, resume.Session{ID: "target"}); err != nil {
		t.Fatalf("retry: %v", err)
	}
	if n := countRows(t, dbPath, `SELECT COUNT(*) FROM conversation_summaries`); n != 0 {
		t.Errorf("summaries rows after retry = %d, want 0", n)
	}
	if _, err := os.Stat(convDB); !os.IsNotExist(err) {
		t.Error("conversation DB still exists after retry")
	}
}

func TestDeleteSessionRemovesNestedDescendants(t *testing.T) {
	root := filepath.Join(t.TempDir(), "data")
	uris := `["file:///tmp/repo"]`
	dbPath := createDB(t, root, false, []row{
		{id: "target", steps: 2, modified: "2026-11-20T10:00:00Z", uris: uris},
		{id: "kid", steps: 2, modified: "2026-11-20T10:00:00Z", uris: uris, parent: "target", depth: 1},
		{id: "grandkid", steps: 2, modified: "2026-11-20T10:00:00Z", uris: uris, parent: "kid", depth: 2},
		{id: "sibling", steps: 2, modified: "2026-11-20T10:00:00Z", uris: uris},
	})
	for _, id := range []string{"target", "kid", "grandkid", "sibling"} {
		writeFile(t, filepath.Join(root, "conversations", id+".db"), "x")
		writeFile(t, transcriptPath(root, id), "{}\n")
	}

	if err := deleteSessionInRoot(root, resume.Session{ID: "target"}); err != nil {
		t.Fatal(err)
	}
	if n := countRows(t, dbPath, `SELECT COUNT(*) FROM conversation_summaries WHERE conversation_id != 'sibling'`); n != 0 {
		t.Errorf("descendant rows left = %d", n)
	}
	if n := countRows(t, dbPath, `SELECT COUNT(*) FROM conversations_fts WHERE conversation_id != 'sibling'`); n != 0 {
		t.Errorf("descendant fts rows left = %d", n)
	}
	for _, p := range []string{
		filepath.Join(root, "conversations", "grandkid.db"),
		filepath.Join(root, "brain", "grandkid"),
	} {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Errorf("%s still exists", p)
		}
	}
	if n := countRows(t, dbPath, `SELECT COUNT(*) FROM conversation_summaries WHERE conversation_id = 'sibling'`); n != 1 {
		t.Error("sibling row removed")
	}
	if _, err := os.Stat(transcriptPath(root, "sibling")); err != nil {
		t.Errorf("sibling transcript removed: %v", err)
	}
}

func TestDeleteSessionWithoutOptionalTables(t *testing.T) {
	root := filepath.Join(t.TempDir(), "data")
	dbPath := createDB(t, root, false, []row{
		{id: "target", steps: 2, modified: "2026-11-20T10:00:00Z", uris: `["file:///tmp/repo"]`},
	})
	execDB(t, dbPath, `DROP TABLE conversations_fts`, `DROP TABLE conversation_parents`)

	if err := deleteSessionInRoot(root, resume.Session{ID: "target"}); err != nil {
		t.Fatal(err)
	}
	if n := countRows(t, dbPath, `SELECT COUNT(*) FROM conversation_summaries`); n != 0 {
		t.Errorf("summaries rows = %d, want 0", n)
	}
}

func TestDeleteSessionRowDeletesAreAtomic(t *testing.T) {
	root := filepath.Join(t.TempDir(), "data")
	dbPath := createDB(t, root, false, []row{
		{id: "target", steps: 2, modified: "2026-11-20T10:00:00Z", uris: `["file:///tmp/repo"]`},
	})
	convDB := filepath.Join(root, "conversations", "target.db")
	ann := filepath.Join(root, "annotations", "target.pbtxt")
	transcript := transcriptPath(root, "target")
	writeFile(t, convDB, "conv-data")
	writeFile(t, ann, "ann-data")
	writeFile(t, transcript, "{}\n")

	// Schema drift in a side table must fail the delete before touching files.
	execDB(t, dbPath, `DROP TABLE conversation_parents`, `CREATE TABLE conversation_parents (conversation_id TEXT)`)

	if err := deleteSessionInRoot(root, resume.Session{ID: "target"}); err == nil {
		t.Fatal("delete succeeded although a side table could not be cleaned")
	}
	if n := countRows(t, dbPath, `SELECT COUNT(*) FROM conversation_summaries`); n != 1 {
		t.Errorf("summaries rows = %d, want 1 (rolled back)", n)
	}
	if n := countRows(t, dbPath, `SELECT COUNT(*) FROM conversations_fts`); n != 1 {
		t.Errorf("fts rows = %d, want 1 (rolled back)", n)
	}
	for _, p := range []string{convDB, ann, transcript} {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("artifact %s was removed despite SQL failure: %v", p, err)
		}
	}
}

func TestDeleteSessionRejectsDirectoryAtFilePath(t *testing.T) {
	for _, rel := range []string{
		filepath.Join("conversations", "target.db", "keep.txt"),
		filepath.Join("annotations", "target.pbtxt", "keep.txt"),
	} {
		t.Run(rel, func(t *testing.T) {
			root := filepath.Join(t.TempDir(), "data")
			dbPath := createDB(t, root, false, []row{
				{id: "target", steps: 2, modified: "2026-11-20T10:00:00Z", uris: `["file:///tmp/repo"]`},
			})
			nested := filepath.Join(root, rel)
			transcript := transcriptPath(root, "target")
			writeFile(t, nested, "do-not-delete")
			writeFile(t, transcript, "{}\n")

			if err := deleteSessionInRoot(root, resume.Session{ID: "target"}); err == nil {
				t.Fatalf("delete succeeded although %s is a directory", filepath.Dir(nested))
			}
			if _, err := os.Stat(nested); err != nil {
				t.Errorf("nested file %s removed: %v", nested, err)
			}
			if _, err := os.Stat(transcript); err != nil {
				t.Errorf("transcript removed despite file-path directory error: %v", err)
			}
			if n := countRows(t, dbPath, `SELECT COUNT(*) FROM conversation_summaries`); n != 1 {
				t.Errorf("summaries rows = %d, want 1", n)
			}
		})
	}
}

func TestDeleteSessionFailsOnUnstatableDB(t *testing.T) {
	root := filepath.Join(t.TempDir(), "data")
	convDB := filepath.Join(root, "conversations", "target.db")
	transcript := transcriptPath(root, "target")
	writeFile(t, convDB, "conv-data")
	writeFile(t, transcript, "{}\n")

	// A symlink loop produces ELOOP (non-ErrNotExist) from os.Stat for all users.
	dbPath := filepath.Join(root, summariesDB)
	if err := os.Symlink(dbPath, dbPath); err != nil {
		t.Fatal(err)
	}

	if err := deleteSessionInRoot(root, resume.Session{ID: "target"}); err == nil {
		t.Fatal("delete succeeded although summaries DB could not be stat'd")
	}
	for _, p := range []string{convDB, transcript} {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("artifact %s removed despite DB stat error: %v", p, err)
		}
	}
}

func TestDeleteSessionRejectsNonSearchableBrainDir(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions")
	}
	root := filepath.Join(t.TempDir(), "data")
	dbPath := createDB(t, root, false, []row{
		{id: "target", steps: 2, modified: "2026-11-20T10:00:00Z", uris: `["file:///tmp/repo"]`},
	})
	convDB := filepath.Join(root, "conversations", "target.db")
	writeFile(t, convDB, "conv-data")
	writeFile(t, transcriptPath(root, "target"), "{}\n")
	logs := filepath.Dir(transcriptPath(root, "target"))
	// Mode 0600 is readable and writable, but lacks search/execute (X_OK),
	// which prevents unlinking regular-file entries inside it.
	if err := os.Chmod(logs, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(logs, 0o700) })

	if err := deleteSessionInRoot(root, resume.Session{ID: "target"}); err == nil {
		t.Fatal("delete succeeded although brain subdir lacked search permission")
	}
	if n := countRows(t, dbPath, `SELECT COUNT(*) FROM conversation_summaries`); n != 1 {
		t.Fatalf("summaries rows = %d, want 1 (uncommitted)", n)
	}
	if _, err := os.Stat(convDB); err != nil {
		t.Fatalf("conversation DB was not preserved: %v", err)
	}

	if err := os.Chmod(logs, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := deleteSessionInRoot(root, resume.Session{ID: "target"}); err != nil {
		t.Fatalf("retry: %v", err)
	}
	if n := countRows(t, dbPath, `SELECT COUNT(*) FROM conversation_summaries`); n != 0 {
		t.Errorf("summaries rows after retry = %d, want 0", n)
	}
}

func TestDeleteSessionAllowsReadOnlyPresenceLockfile(t *testing.T) {
	root := filepath.Join(t.TempDir(), "data")
	dbPath := createDB(t, root, false, []row{
		{id: "ro", steps: 2, modified: "2026-11-20T10:00:00Z", uris: `["file:///tmp/repo"]`},
	})
	lockPath := filepath.Join(root, "presence", "ro.lock")
	writeFile(t, lockPath, "")
	if err := os.Chmod(lockPath, 0o444); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(lockPath, 0o600) })

	if err := deleteSessionInRoot(root, resume.Session{ID: "ro"}); err != nil {
		t.Fatalf("delete with 0444 lockfile failed: %v", err)
	}
	if n := countRows(t, dbPath, `SELECT COUNT(*) FROM conversation_summaries`); n != 0 {
		t.Errorf("summaries rows = %d, want 0", n)
	}
}

func TestDeleteSessionExclusiveLockBlocksBeforeTouchingFiles(t *testing.T) {
	root := filepath.Join(t.TempDir(), "data")
	dbPath := createDB(t, root, false, []row{
		{id: "target", steps: 2, modified: "2026-11-20T10:00:00Z", uris: `["file:///tmp/repo"]`},
	})
	convDB := filepath.Join(root, "conversations", "target.db")
	transcript := transcriptPath(root, "target")
	writeFile(t, convDB, "conv-data")
	writeFile(t, transcript, "{}\n")

	// Hold a SHARED read lock in rollback-journal mode.
	reader, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	rtx, err := reader.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer rtx.Rollback()
	var n int
	if err := rtx.QueryRow(`SELECT COUNT(*) FROM conversation_summaries`).Scan(&n); err != nil {
		t.Fatal(err)
	}

	origTimeout := sqliteBusyTimeout
	sqliteBusyTimeout = 50
	t.Cleanup(func() { sqliteBusyTimeout = origTimeout })

	if err := deleteSessionInRoot(root, resume.Session{ID: "target"}); err == nil {
		t.Fatal("delete succeeded while reader held SHARED lock in rollback-journal mode")
	}
	for _, p := range []string{convDB, transcript} {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("artifact %s removed before EXCLUSIVE lock acquired: %v", p, err)
		}
	}

	if err := rtx.Rollback(); err != nil {
		t.Fatal(err)
	}
	if err := deleteSessionInRoot(root, resume.Session{ID: "target"}); err != nil {
		t.Fatalf("retry after reader released lock failed: %v", err)
	}
	if n := countRows(t, dbPath, `SELECT COUNT(*) FROM conversation_summaries`); n != 0 {
		t.Errorf("summaries rows = %d, want 0", n)
	}
}

func TestDeleteSessionRemovesEmptyUnreadableBrainDir(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions")
	}
	root := filepath.Join(t.TempDir(), "data")
	dbPath := createDB(t, root, false, []row{
		{id: "target", steps: 2, modified: "2026-11-20T10:00:00Z", uris: `["file:///tmp/repo"]`},
	})
	brainDir := filepath.Join(root, "brain", "target")
	if err := os.MkdirAll(brainDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(brainDir, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(brainDir, 0o700) })

	if err := deleteSessionInRoot(root, resume.Session{ID: "target"}); err != nil {
		t.Fatalf("delete of empty 000 brain dir failed: %v", err)
	}
	if _, err := os.Stat(brainDir); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("empty 000 brain dir still exists: %v", err)
	}
	if n := countRows(t, dbPath, `SELECT COUNT(*) FROM conversation_summaries`); n != 0 {
		t.Errorf("summaries rows = %d, want 0", n)
	}
}

func TestDeleteSessionPreservesParentWhenDescendantFails(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions")
	}
	root := filepath.Join(t.TempDir(), "data")
	uris := `["file:///tmp/repo"]`
	dbPath := createDB(t, root, false, []row{
		{id: "target", steps: 2, modified: "2026-11-20T10:00:00Z", uris: uris},
		{id: "kid", steps: 2, modified: "2026-11-20T10:00:00Z", uris: uris, parent: "target", depth: 1},
	})
	parentDB := filepath.Join(root, "conversations", "target.db")
	parentTr := transcriptPath(root, "target")
	writeFile(t, parentDB, "parent-db")
	writeFile(t, parentTr, "{}\n")
	writeFile(t, transcriptPath(root, "kid"), "{}\n")

	kidLogs := filepath.Dir(transcriptPath(root, "kid"))
	if err := os.Chmod(kidLogs, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(kidLogs, 0o700) })

	if err := deleteSessionInRoot(root, resume.Session{ID: "target"}); err == nil {
		t.Fatal("delete succeeded although descendant brain dir was unwritable")
	}
	for _, p := range []string{parentDB, parentTr} {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("parent artifact %s removed despite descendant failure: %v", p, err)
		}
	}
	if n := countRows(t, dbPath, `SELECT COUNT(*) FROM conversation_summaries`); n != 2 {
		t.Errorf("summaries rows = %d, want 2", n)
	}
}
