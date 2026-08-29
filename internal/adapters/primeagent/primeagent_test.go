package primeagent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hmepas/resume/internal/resume"
)

func TestParseSessionUsesPrimeMetadata(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")
	data := `{"type":"session","version":3,"id":"01a-session","timestamp":"2026-08-29T12:00:00Z","cwd":"/repo"}
{"type":"message","id":"one","parentId":null,"timestamp":"2026-08-29T12:10:00Z","message":{"role":"user","content":[{"type":"text","text":"first prompt"}],"timestamp":1788004920000}}
{"type":"message","id":"two","parentId":"one","timestamp":"2026-08-29T12:11:00Z","message":{"role":"assistant","content":[{"type":"text","text":"answer"}],"timestamp":1788005040000}}
{"type":"message","id":"three","parentId":"two","timestamp":"2026-08-29T12:20:00Z","message":{"role":"toolResult","content":[{"type":"text","text":"output"}],"timestamp":1788006000000}}
{"type":"session_info","id":"four","parentId":"three","timestamp":"2026-08-29T12:21:00Z","name":"  Named session  "}
`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}

	session := parseSession(path)
	if session.Agent != "prime-agent" {
		t.Fatalf("Agent = %q, want prime-agent", session.Agent)
	}
	if session.ID != "01a-session" {
		t.Fatalf("ID = %q, want 01a-session", session.ID)
	}
	if session.Project != "/repo" {
		t.Fatalf("Project = %q, want /repo", session.Project)
	}
	if session.Title != "Named session" {
		t.Fatalf("Title = %q, want Named session", session.Title)
	}
	if !session.StartedAt.Equal(time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC)) {
		t.Fatalf("StartedAt = %s", session.StartedAt)
	}
	// The last activity is the session_info rename at 12:21; tool results and
	// renames count toward recency, not only chat messages.
	wantUpdated := time.Date(2026, 8, 29, 12, 21, 0, 0, time.UTC)
	if !session.UpdatedAt.Equal(wantUpdated) {
		t.Fatalf("UpdatedAt = %s, want %s", session.UpdatedAt, wantUpdated)
	}
	if session.SourcePath != path {
		t.Fatalf("SourcePath = %q, want %q", session.SourcePath, path)
	}
	if session.ResumeHint != "prime-agent --resume 01a-session" {
		t.Fatalf("ResumeHint = %q", session.ResumeHint)
	}
	if session.Confidence != "high" {
		t.Fatalf("Confidence = %q, want high", session.Confidence)
	}
}

func TestParseSessionClearedNameFallsBackToFirstUserMessage(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")
	data := `{"type":"session","version":3,"id":"01a-session","timestamp":"2026-08-29T12:00:00Z","cwd":"/repo"}
{"type":"session_info","id":"one","parentId":null,"timestamp":"2026-08-29T12:01:00Z","name":"old name"}
{"type":"message","id":"two","parentId":"one","timestamp":"2026-08-29T12:02:00Z","message":{"role":"user","content":"first prompt"}}
{"type":"session_info","id":"three","parentId":"two","timestamp":"2026-08-29T12:03:00Z","name":"   "}
`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}

	if got := parseSession(path).Title; got != "first prompt" {
		t.Fatalf("Title = %q, want first prompt", got)
	}
}

func TestParseSessionInnerTimestampFallsBackToOuter(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.jsonl")
	data := `{"type":"session","version":3,"id":"01a-session","timestamp":"2026-08-29T12:00:00Z","cwd":"/repo"}
{"type":"message","id":"one","parentId":null,"timestamp":"2026-08-29T12:10:00Z","message":{"role":"user","content":"first prompt","timestamp":null}}
{"type":"message","id":"two","parentId":"one","timestamp":"2026-08-29T12:12:00Z","message":{"role":"assistant","content":"answer","timestamp":"1788006000000"}}
`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}

	// A null inner timestamp and an unparseable string one both fall back to
	// the outer entry timestamp instead of contributing zero time.
	wantUpdated := time.Date(2026, 8, 29, 12, 12, 0, 0, time.UTC)
	if got := parseSession(path).UpdatedAt; !got.Equal(wantUpdated) {
		t.Fatalf("UpdatedAt = %s, want %s", got, wantUpdated)
	}
}

func TestParseSessionReadsMetadataAroundLargeEntries(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.jsonl")
	data := `{"type":"session","version":3,"id":"01a-session","timestamp":"2026-08-29T12:00:00Z","cwd":"/repo"}
{"type":"message","id":"one","parentId":null,"timestamp":"2026-08-29T12:01:00Z","message":{"role":"user","content":"first prompt","timestamp":1788004860000}}
{"type":"message","id":"two","parentId":"one","timestamp":"2026-08-29T12:02:00Z","message":{"role":"toolResult","content":"` + strings.Repeat("x", 256*1024) + `"}}
{"type":"message","id":"three","parentId":"two","timestamp":"2026-08-29T12:05:00Z","message":{"role":"assistant","content":"` + strings.Repeat("y", 128*1024) + `","timestamp":1788005100000}}
{"type":"session_info","id":"four","parentId":"three","timestamp":"2026-08-29T12:06:00Z","name":"name near the end"}
`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}

	// The file exceeds the bounded window, so only its head and tail are read;
	// the last-wins rename and the final timestamp live in the tail.
	session := parseSession(path)
	if session.Title != "name near the end" {
		t.Fatalf("Title = %q, want name near the end", session.Title)
	}
	wantUpdated := time.Date(2026, 8, 29, 12, 6, 0, 0, time.UTC)
	if !session.UpdatedAt.Equal(wantUpdated) {
		t.Fatalf("UpdatedAt = %s, want %s", session.UpdatedAt, wantUpdated)
	}
}

func TestParseSessionFallsBackToFullScanForBuriedTitle(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.jsonl")
	data := `{"type":"session","version":3,"id":"01a-session","timestamp":"2026-08-29T12:00:00Z","cwd":"/repo"}
{"type":"message","id":"one","parentId":null,"timestamp":"2026-08-29T12:02:00Z","message":{"role":"toolResult","content":"` + strings.Repeat("x", 256*1024) + `"}}
{"type":"message","id":"two","parentId":"one","timestamp":"2026-08-29T12:03:00Z","message":{"role":"user","content":"buried prompt"}}
{"type":"message","id":"three","parentId":"two","timestamp":"2026-08-29T12:04:00Z","message":{"role":"toolResult","content":"` + strings.Repeat("y", 192*1024) + `"}}
`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}

	// The bounded scan sees neither a name nor a title here, so the file is
	// re-scanned in full and the mid-file prompt is recovered.
	session := parseSession(path)
	if session.Title != "buried prompt" {
		t.Fatalf("Title = %q, want buried prompt", session.Title)
	}
	wantUpdated := time.Date(2026, 8, 29, 12, 4, 0, 0, time.UTC)
	if !session.UpdatedAt.Equal(wantUpdated) {
		t.Fatalf("UpdatedAt = %s, want %s", session.UpdatedAt, wantUpdated)
	}
}

func TestParseSessionRequiresValidHeader(t *testing.T) {
	validHeader := `{"type":"session","version":3,"id":"01a-session","timestamp":"2026-08-29T12:00:00Z","cwd":"/repo"}`
	tests := map[string]string{
		"message first":      `{"type":"message","id":"entry","timestamp":"2026-08-29T12:00:00Z","message":{"role":"user","content":"prompt"}}`,
		"missing id":         `{"type":"session","timestamp":"2026-08-29T12:00:00Z","cwd":"/repo"}`,
		"missing cwd":        `{"type":"session","id":"session","timestamp":"2026-08-29T12:00:00Z"}`,
		"garbage first line": "not json at all\n" + validHeader,
		"header on second line": `{"type":"message","id":"entry","timestamp":"2026-08-29T12:00:00Z","message":{"role":"user","content":"prompt"}}
` + validHeader,
		"oversized first line": strings.Repeat("x", maxHeaderLine+1) + "\n" + validHeader,
	}
	for name, data := range tests {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "session.jsonl")
			if err := os.WriteFile(path, []byte(data+"\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			if got := parseSession(path); got.SourcePath != "" {
				t.Fatalf("parseSession() accepted invalid header, got %#v", got)
			}
		})
	}
}

func TestParseSessionFallsBackToFileModTime(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.jsonl")
	data := `{"type":"session","version":3,"id":"01a-session","timestamp":"invalid","cwd":"/repo"}
`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, 8, 29, 12, 30, 0, 0, time.Local)
	if err := os.Chtimes(path, want, want); err != nil {
		t.Fatal(err)
	}

	if got := parseSession(path).UpdatedAt; !got.Equal(want) {
		t.Fatalf("UpdatedAt = %s, want %s", got, want)
	}
}

func TestSessionRootUsesPrimeSettingsPrecedence(t *testing.T) {
	dir := t.TempDir()
	agentDir := filepath.Join(dir, "agent")
	project := filepath.Join(dir, "project")
	globalSessions := filepath.Join(dir, "global-sessions")
	projectSessions := filepath.Join(dir, "project-sessions")
	envSessions := filepath.Join(dir, "env-sessions")
	writeSettings(t, filepath.Join(agentDir, "settings.json"), globalSessions)
	writeSettings(t, filepath.Join(project, ".prime", "agent", "settings.json"), projectSessions)
	t.Setenv("PRIME_AGENT_CODING_AGENT_DIR", agentDir)
	t.Setenv("PRIME_AGENT_SESSION_DIR", "")
	t.Setenv("PRIME_AGENT_CODING_AGENT_SESSION_DIR", "")
	ctx := resume.Context{Project: resume.Project{Path: project, Root: project}}

	got, err := sessionRoot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got != projectSessions {
		t.Fatalf("sessionRoot() = %q, want project setting %q", got, projectSessions)
	}

	projectSettings := filepath.Join(project, ".prime", "agent", "settings.json")
	if err := os.Remove(projectSettings); err != nil {
		t.Fatal(err)
	}
	got, err = sessionRoot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got != globalSessions {
		t.Fatalf("sessionRoot() = %q, want global setting %q", got, globalSessions)
	}

	writeSettings(t, projectSettings, "")
	got, err = sessionRoot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	wantDefault := filepath.Join(agentDir, "sessions")
	if got != wantDefault {
		t.Fatalf("sessionRoot() with empty project setting = %q, want %q", got, wantDefault)
	}

	t.Setenv("PRIME_AGENT_SESSION_DIR", envSessions)
	got, err = sessionRoot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got != envSessions {
		t.Fatalf("sessionRoot() = %q, want env override %q", got, envSessions)
	}
}

func TestSessionRootEmptyEnvFallsThrough(t *testing.T) {
	dir := t.TempDir()
	fallback := filepath.Join(dir, "fallback-sessions")
	t.Setenv("PRIME_AGENT_SESSION_DIR", "")
	t.Setenv("PRIME_AGENT_CODING_AGENT_SESSION_DIR", fallback)

	got, err := sessionRoot(resume.Context{})
	if err != nil {
		t.Fatal(err)
	}
	if got != fallback {
		t.Fatalf("sessionRoot() = %q, want fallback env %q", got, fallback)
	}

	primary := filepath.Join(dir, "primary-sessions")
	t.Setenv("PRIME_AGENT_SESSION_DIR", primary)
	got, err = sessionRoot(resume.Context{})
	if err != nil {
		t.Fatal(err)
	}
	if got != primary {
		t.Fatalf("sessionRoot() = %q, want primary env %q", got, primary)
	}
}

func TestSessionRootResolvesRelativeToSettingsFile(t *testing.T) {
	dir := t.TempDir()
	agentDir := filepath.Join(dir, "agent")
	writeSettings(t, filepath.Join(agentDir, "settings.json"), "sessions-v2")
	t.Setenv("PRIME_AGENT_CODING_AGENT_DIR", agentDir)
	t.Setenv("PRIME_AGENT_SESSION_DIR", "")
	t.Setenv("PRIME_AGENT_CODING_AGENT_SESSION_DIR", "")

	got, err := sessionRoot(resume.Context{})
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(agentDir, "sessions-v2")
	if got != want {
		t.Fatalf("sessionRoot() = %q, want %q resolved against the settings file", got, want)
	}
}

func TestSessionRootReadsProjectRootSettings(t *testing.T) {
	dir := t.TempDir()
	agentDir := filepath.Join(dir, "agent")
	root := filepath.Join(dir, "repo")
	rootSessions := filepath.Join(dir, "root-sessions")
	writeSettings(t, filepath.Join(root, ".prime", "agent", "settings.json"), rootSessions)
	t.Setenv("PRIME_AGENT_CODING_AGENT_DIR", agentDir)
	t.Setenv("PRIME_AGENT_SESSION_DIR", "")
	t.Setenv("PRIME_AGENT_CODING_AGENT_SESSION_DIR", "")
	ctx := resume.Context{Project: resume.Project{Path: filepath.Join(root, "sub"), Root: root}}

	got, err := sessionRoot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got != rootSessions {
		t.Fatalf("sessionRoot() from subdirectory = %q, want git-root setting %q", got, rootSessions)
	}
}

func TestSessionsFollowsSymlinkedRoot(t *testing.T) {
	dir := t.TempDir()
	real := filepath.Join(dir, "real-sessions")
	if err := os.MkdirAll(real, 0o700); err != nil {
		t.Fatal(err)
	}
	writeTestSession(t, filepath.Join(real, "one.jsonl"), "one", filepath.Join(dir, "repo"))
	link := filepath.Join(dir, "link-sessions")
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PRIME_AGENT_SESSION_DIR", link)

	sessions, err := Adapter{}.Sessions(resume.Context{All: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 1 || sessions[0].ID != "one" {
		t.Fatalf("sessions through symlinked root = %#v, want one", sessions)
	}
}

func TestSessionsFiltersByProject(t *testing.T) {
	dir := t.TempDir()
	repo := filepath.Join(dir, "repo")
	other := filepath.Join(dir, "other")
	if err := os.MkdirAll(repo, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(other, 0o700); err != nil {
		t.Fatal(err)
	}
	writeTestSession(t, filepath.Join(dir, "one.jsonl"), "one", repo)
	writeTestSession(t, filepath.Join(dir, "two.jsonl"), "two", other)
	t.Setenv("PRIME_AGENT_SESSION_DIR", dir)

	adapter := Adapter{}
	ctx := resume.Context{Project: resume.Project{Path: repo, Root: repo}}
	sessions, err := adapter.Sessions(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 1 || sessions[0].ID != "one" {
		t.Fatalf("filtered sessions = %#v, want only one", sessions)
	}

	ctx.All = true
	sessions, err = adapter.Sessions(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 2 {
		t.Fatalf("all sessions count = %d, want 2", len(sessions))
	}
}

func writeSettings(t *testing.T, path, sessionDir string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	data := `{"sessionDir":` + string(mustJSON(t, sessionDir)) + `}`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
}

func mustJSON(t *testing.T, value string) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func writeTestSession(t *testing.T, path, id, cwd string) {
	t.Helper()
	data := `{"type":"session","version":3,"id":"` + id + `","timestamp":"2026-08-29T12:00:00Z","cwd":"` + cwd + `"}
`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
}
