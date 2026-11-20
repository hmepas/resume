package resume

import "testing"

func TestCommandStrings(t *testing.T) {
	tests := []struct {
		name    string
		command Command
		str     string
		shell   string
	}{
		{
			name:    "plain args",
			command: Command{Name: "codex", Args: []string{"resume", "abc-123"}},
			str:     "codex resume abc-123",
			shell:   "codex resume abc-123",
		},
		{
			name:    "arg with space and dir",
			command: Command{Name: "claude", Args: []string{"--resume", "a b"}, Dir: "/tmp/my project"},
			str:     "claude --resume 'a b'",
			shell:   "cd '/tmp/my project' && claude --resume 'a b'",
		},
		{
			name:    "arg with single quote",
			command: Command{Name: "pi", Args: []string{"it's"}, Dir: "/repo"},
			str:     `pi 'it'\''s'`,
			shell:   `cd /repo && pi 'it'\''s'`,
		},
		{
			name:    "empty arg",
			command: Command{Name: "gemini", Args: []string{""}},
			str:     "gemini ''",
			shell:   "gemini ''",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := append([]string(nil), tt.command.Args...)
			if got := tt.command.String(); got != tt.str {
				t.Fatalf("String() = %q, want %q", got, tt.str)
			}
			if got := tt.command.ShellString(); got != tt.shell {
				t.Fatalf("ShellString() = %q, want %q", got, tt.shell)
			}
			for i := range args {
				if tt.command.Args[i] != args[i] {
					t.Fatalf("Args mutated: %q, want %q", tt.command.Args, args)
				}
			}
		})
	}
}
