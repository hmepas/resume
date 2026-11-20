package adapters

import (
	"slices"
	"testing"

	"github.com/hmepas/resume/internal/resume"
)

// TestBuiltinContract pins every built-in adapter's resume command, picker
// color and delete scope. Adding an adapter requires adding a row here.
func TestBuiltinContract(t *testing.T) {
	want := map[string]struct {
		command     string
		color       resume.Color
		deleteScope string // empty: the adapter must not implement resume.Deleter
	}{
		"codex":       {"codex resume id-1", resume.ColorGreen, "session and its guardian reviews"},
		"claude":      {"claude --resume id-1", resume.ColorRed, "session file"},
		"gemini":      {"gemini", resume.ColorMagenta, "session file"},
		"cursor":      {"cursor /repo", resume.ColorBlue, "session file"},
		"opencode":    {"opencode -s id-1", resume.ColorYellow, "session"},
		"pi":          {"pi --session id-1", resume.ColorCyan, "session file"},
		"prime-agent": {"prime-agent --resume id-1", resume.ColorBrightMagenta, "session file"},
		"agy":         {"agy --conversation id-1", resume.ColorBrightCyan, "session"},
	}

	ids := map[string]bool{}
	colors := map[resume.Color]string{}
	for _, adapter := range Builtin() {
		id := adapter.ID()
		if id == "" {
			t.Errorf("%T: empty ID", adapter)
			continue
		}
		if ids[id] {
			t.Errorf("duplicate adapter ID %q", id)
		}
		ids[id] = true

		color := adapter.Color()
		if color == "" {
			t.Errorf("%s: empty Color", id)
		} else if other, ok := colors[color]; ok {
			t.Errorf("%s: Color %q already used by %s", id, color, other)
		}
		colors[color] = id

		w, ok := want[id]
		if !ok {
			t.Errorf("%s: no row in the contract table; add one to TestBuiltinContract", id)
			continue
		}

		session := resume.Session{Agent: id, ID: "id-1", Project: "/repo"}
		cmd, err := adapter.Command(session)
		if err != nil {
			t.Errorf("%s: Command: %v", id, err)
		} else {
			if got := cmd.String(); got != w.command {
				t.Errorf("%s: Command = %q, want %q", id, got, w.command)
			}
			if cmd.Dir != "/repo" {
				t.Errorf("%s: Command.Dir = %q, want /repo", id, cmd.Dir)
			}
			if slices.Contains(cmd.Args, "id-1") {
				session.ID = ""
				if _, err := adapter.Command(session); err == nil {
					t.Errorf("%s: Command with empty ID returned nil error", id)
				}
			}
		}
		if color != w.color {
			t.Errorf("%s: Color = %q, want %q", id, color, w.color)
		}

		deleter, isDeleter := adapter.(resume.Deleter)
		switch {
		case w.deleteScope == "" && isDeleter:
			t.Errorf("%s: implements resume.Deleter, want no delete support", id)
		case w.deleteScope != "" && !isDeleter:
			t.Errorf("%s: does not implement resume.Deleter", id)
		case isDeleter && deleter.DeleteScope() != w.deleteScope:
			t.Errorf("%s: DeleteScope = %q, want %q", id, deleter.DeleteScope(), w.deleteScope)
		}
	}
	for id := range want {
		if !ids[id] {
			t.Errorf("contract table row %q has no adapter in Builtin()", id)
		}
	}
}
