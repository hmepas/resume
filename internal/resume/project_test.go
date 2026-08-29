package resume

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExpandHome(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	if got := ExpandHome("~"); got != home {
		t.Fatalf("ExpandHome(~) = %q, want %q", got, home)
	}
	if got, want := ExpandHome("~/x"), filepath.Join(home, "x"); got != want {
		t.Fatalf("ExpandHome(~/x) = %q, want %q", got, want)
	}
	if got := ExpandHome("/abs/path"); got != "/abs/path" {
		t.Fatalf("ExpandHome(/abs/path) = %q, want unchanged", got)
	}
	if got := ExpandHome("~other/x"); got != "~other/x" {
		t.Fatalf("ExpandHome(~other/x) = %q, want unchanged", got)
	}
}

func TestPathMatchesProjectSubdirectory(t *testing.T) {
	project := Project{
		Path: "/repo",
		Root: "/repo",
	}

	if !PathMatches(project, "/repo/apps/api") {
		t.Fatal("PathMatches() rejected project subdirectory")
	}
	if PathMatches(project, "/repo-other") {
		t.Fatal("PathMatches() accepted sibling path")
	}
}
