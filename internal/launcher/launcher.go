package launcher

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/hmepas/resume/internal/resume"
)

type Command struct {
	Name string
	Args []string
	Dir  string
}

func ForSession(session resume.Session) (Command, error) {
	dir := session.Project
	switch session.Agent {
	case "codex":
		return resumeCommand(session, "resume")
	case "claude", "prime-agent":
		return resumeCommand(session, "--resume")
	case "gemini":
		return Command{Name: "gemini", Dir: dir}, nil
	case "cursor":
		return Command{Name: "cursor", Args: []string{dir}, Dir: dir}, nil
	case "opencode":
		return resumeCommand(session, "-s")
	case "pi":
		return resumeCommand(session, "--session")
	default:
		return Command{}, fmt.Errorf("no launcher for agent %q", session.Agent)
	}
}

func resumeCommand(session resume.Session, args ...string) (Command, error) {
	if session.ID == "" || session.ID == session.ResumeHint {
		return Command{}, fmt.Errorf("%s session has no resume id", session.Agent)
	}
	return Command{Name: session.Agent, Args: append(args, session.ID), Dir: session.Project}, nil
}

func Run(command Command) error {
	if _, err := exec.LookPath(command.Name); err != nil {
		return fmt.Errorf("%s not found in PATH", command.Name)
	}

	cmd := exec.Command(command.Name, command.Args...)
	cmd.Dir = command.Dir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func ShellString(command Command) string {
	parts := append([]string{command.Name}, command.Args...)
	for i, part := range parts {
		parts[i] = quote(part)
	}
	if command.Dir != "" {
		return "cd " + quote(command.Dir) + " && " + strings.Join(parts, " ")
	}
	return strings.Join(parts, " ")
}

func quote(s string) string {
	if s == "" {
		return "''"
	}
	if strings.IndexFunc(s, func(r rune) bool {
		return !(r == '_' || r == '-' || r == '.' || r == '/' || r == ':' || r == '+' ||
			r >= '0' && r <= '9' || r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z')
	}) < 0 {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}
