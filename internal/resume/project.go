package resume

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func DetectProject(start string) (Project, error) {
	abs, err := filepath.Abs(start)
	if err != nil {
		return Project{}, err
	}
	abs, err = filepath.EvalSymlinks(abs)
	if err != nil {
		return Project{}, err
	}

	root := gitRoot(abs)
	if root == "" {
		root = abs
	}

	return Project{Path: abs, Root: cleanPath(root)}, nil
}

func gitRoot(dir string) string {
	cmd := exec.Command("git", "-C", dir, "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return cleanPath(strings.TrimSpace(string(out)))
}

func cleanPath(path string) string {
	if path == "" {
		return ""
	}
	if expanded, ok := strings.CutPrefix(path, "~/"); ok {
		if home, err := os.UserHomeDir(); err == nil {
			path = filepath.Join(home, expanded)
		}
	}
	abs, err := filepath.Abs(path)
	if err == nil {
		path = abs
	}
	eval, err := filepath.EvalSymlinks(path)
	if err == nil {
		path = eval
	}
	return filepath.Clean(path)
}

func PathMatches(project Project, candidate string) bool {
	candidate = cleanPath(candidate)
	return samePathOrChild(project.Root, candidate) || samePathOrChild(project.Path, candidate)
}

func samePathOrChild(parent, child string) bool {
	parent = cleanPath(parent)
	child = cleanPath(child)
	if parent == "" || child == "" {
		return false
	}
	if child == parent {
		return true
	}
	rel, err := filepath.Rel(parent, child)
	if err != nil {
		return false
	}
	return rel != "." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != ".."
}
