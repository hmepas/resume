package claude

import (
	"testing"

	"github.com/hmepas/resume/internal/resume"
)

func TestLooseDirMatch(t *testing.T) {
	project := resume.Project{
		Root: "/Users/me/projects/dot.files/dot-base",
		Path: "/Users/me/projects/dot.files/dot-base/sub_dir",
	}
	cases := []struct {
		dir  string
		want bool
	}{
		{"-Users-me-projects-dot-files-dot-base", true},
		{"-Users-me-projects-dot-files-dot-base-sub-dir", true},
		{"-Users-me-projects-dot-files-dot-base-deeper-child", true},
		{"-Users-me-projects-dot-files", false},
		{"-Users-me-projects-dot-files-dot-based", false},
		{"-Users-me-other", false},
	}
	for _, c := range cases {
		if got := looseDirMatch(c.dir, project); got != c.want {
			t.Errorf("looseDirMatch(%q) = %v, want %v", c.dir, got, c.want)
		}
	}
}
