package cursor

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/hmepas/resume/internal/adapters/common"
	"github.com/hmepas/resume/internal/resume"
)

type Adapter struct{}

func (Adapter) ID() string { return "cursor" }

func (Adapter) Sessions(ctx resume.Context) ([]resume.Session, error) {
	root, err := storageRoot()
	if err != nil || root == "" || !common.Exists(root) {
		return nil, err
	}

	var sessions []resume.Session
	_ = common.WalkFiles(root, func(path string) {
		if filepath.Base(path) != "workspace.json" {
			return
		}
		project := readWorkspaceFolder(path)
		if project == "" {
			return
		}
		if !ctx.All && !resume.PathMatches(ctx.Project, project) {
			return
		}
		sessions = append(sessions, resume.Session{
			Agent:      "cursor",
			ID:         filepath.Base(filepath.Dir(path)),
			Project:    project,
			UpdatedAt:  common.FileModTime(filepath.Dir(path)),
			Title:      "Cursor workspace",
			SourcePath: path,
			ResumeHint: "cursor " + project,
			Confidence: "low",
		})
	})
	return sessions, nil
}

func storageRoot() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", "Cursor", "User", "workspaceStorage"), nil
	case "linux":
		if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
			return filepath.Join(xdg, "Cursor", "User", "workspaceStorage"), nil
		}
		return filepath.Join(home, ".config", "Cursor", "User", "workspaceStorage"), nil
	default:
		return "", nil
	}
}

func readWorkspaceFolder(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var obj struct {
		Folder string `json:"folder"`
	}
	if err := json.Unmarshal(data, &obj); err != nil {
		return ""
	}
	return strings.TrimPrefix(obj.Folder, "file://")
}
