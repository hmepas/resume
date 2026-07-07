package launcher

import (
	"reflect"
	"testing"

	"github.com/hmepas/resume/internal/resume"
)

func TestForSessionPiUsesSessionID(t *testing.T) {
	command, err := ForSession(resume.Session{
		Agent:   "pi",
		ID:      "019e977a-107c-7244-8dd3-70e814c3d709",
		Project: "/Users/hmepas/projects/grappa",
	})
	if err != nil {
		t.Fatal(err)
	}

	if command.Name != "pi" {
		t.Fatalf("Name = %q, want pi", command.Name)
	}
	if !reflect.DeepEqual(command.Args, []string{"--session", "019e977a-107c-7244-8dd3-70e814c3d709"}) {
		t.Fatalf("Args = %#v", command.Args)
	}
	if command.Dir != "/Users/hmepas/projects/grappa" {
		t.Fatalf("Dir = %q", command.Dir)
	}
}

func TestForSessionPiRequiresSessionID(t *testing.T) {
	_, err := ForSession(resume.Session{Agent: "pi", Project: "/tmp"})
	if err == nil {
		t.Fatal("ForSession(pi without id) returned nil error")
	}
}

func TestForSessionOpenCodeUsesSessionID(t *testing.T) {
	command, err := ForSession(resume.Session{
		Agent:   "opencode",
		ID:      "ses_0c3821344fferjRqBxk9sCT8IO",
		Project: "/Users/hmepas/projects/grappa",
	})
	if err != nil {
		t.Fatal(err)
	}

	if command.Name != "opencode" {
		t.Fatalf("Name = %q, want opencode", command.Name)
	}
	if !reflect.DeepEqual(command.Args, []string{"-s", "ses_0c3821344fferjRqBxk9sCT8IO"}) {
		t.Fatalf("Args = %#v", command.Args)
	}
	if command.Dir != "/Users/hmepas/projects/grappa" {
		t.Fatalf("Dir = %q", command.Dir)
	}
}

func TestForSessionOpenCodeRequiresSessionID(t *testing.T) {
	_, err := ForSession(resume.Session{Agent: "opencode", Project: "/tmp"})
	if err == nil {
		t.Fatal("ForSession(opencode without id) returned nil error")
	}
}
