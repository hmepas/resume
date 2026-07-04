package resume

import "testing"

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
