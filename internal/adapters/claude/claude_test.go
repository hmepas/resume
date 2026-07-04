package claude

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseSessionPrefersActiveName(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "abc.jsonl")
	data := `{"type":"last-prompt","sessionId":"abc"}
{"type":"user","sessionId":"abc","timestamp":"2026-07-04T12:00:00Z","cwd":"/repo","message":{"role":"user","content":"first prompt"}}
{"type":"ai-title","aiTitle":"AI title","sessionId":"abc"}
`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}

	session := parseSession(path, map[string]string{"abc": "runtime-name"})
	if session.Title != "runtime-name: first prompt" {
		t.Fatalf("Title = %q, want runtime-name: first prompt", session.Title)
	}
}

func TestParseSessionUsesCommandName(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "abc.jsonl")
	data := `{"type":"last-prompt","sessionId":"abc"}
{"type":"user","sessionId":"abc","timestamp":"2026-07-04T12:00:00Z","cwd":"/repo","message":{"role":"user","content":"<command-name>/plugin</command-name><command-args>install x</command-args>"}}
`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}

	session := parseSession(path, nil)
	if session.Title != "/plugin" {
		t.Fatalf("Title = %q, want /plugin", session.Title)
	}
}

func TestClaudeChildSessionPath(t *testing.T) {
	if !isClaudeChildSession("/x/session/subagents/agent.jsonl") {
		t.Fatal("subagent path was not detected as child session")
	}
	if isClaudeChildSession("/x/session.jsonl") {
		t.Fatal("top-level session was detected as child session")
	}
}
