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
	path = ExpandHome(path)
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

// ExpandHome rewrites a leading "~" or "~/" to the user's home directory; the
// path is returned unchanged when the home directory is unknown.
func ExpandHome(path string) string {
	if path != "~" && !strings.HasPrefix(path, "~/") {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	return filepath.Join(home, strings.TrimPrefix(path[1:], "/"))
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
