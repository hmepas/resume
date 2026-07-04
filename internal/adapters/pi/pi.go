package pi

import (
	"path/filepath"
	"strings"

	"github.com/hmepas/resume/internal/adapters/common"
	"github.com/hmepas/resume/internal/resume"
)

type Adapter struct{}

func (Adapter) ID() string { return "pi" }

func (Adapter) Sessions(ctx resume.Context) ([]resume.Session, error) {
	roots := [][]string{
		{".pi"},
		{".config", "pi"},
		{".local", "share", "pi"},
	}

	var sessions []resume.Session
	for _, parts := range roots {
		root, err := common.HomePath(parts...)
		if err != nil || !common.Exists(root) {
			continue
		}
		_ = common.WalkFiles(root, func(path string) {
			base := filepath.Base(path)
			if !strings.HasSuffix(base, ".json") && !strings.HasSuffix(base, ".jsonl") {
				return
			}
			session := parseLoose(path)
			if session.SourcePath == "" {
				return
			}
			if !ctx.All && !resume.PathMatches(ctx.Project, session.Project) {
				return
			}
			sessions = append(sessions, session)
		})
	}
	return sessions, nil
}

func parseLoose(path string) resume.Session {
	var session resume.Session
	_ = common.JSONLLines(path, func(obj map[string]any) {
		if session.Project == "" {
			for _, key := range []string{"cwd", "project", "projectPath", "workspace", "workspacePath"} {
				if value, ok := obj[key].(string); ok && value != "" {
					session.Project = value
					break
				}
			}
		}
		if session.Title == "" {
			for _, key := range []string{"title", "summary", "prompt"} {
				if value, ok := obj[key].(string); ok && value != "" {
					session.Title = value
					break
				}
			}
		}
		if session.Title == "" {
			if text := common.FirstUserText(obj); common.UsefulTitle(text) {
				session.Title = text
			}
		}
		if ts := common.ParseTime(common.String(obj, "timestamp")); !ts.IsZero() && ts.After(session.UpdatedAt) {
			session.UpdatedAt = ts
		}
	})
	if session.Project == "" {
		return resume.Session{}
	}
	if session.UpdatedAt.IsZero() {
		session.UpdatedAt = common.FileModTime(path)
	}
	session.Agent = "pi"
	session.ID = sessionIDFromPath(path)
	session.SourcePath = path
	session.ResumeHint = "pi"
	session.Confidence = "low"
	return session
}

func sessionIDFromPath(path string) string {
	base := filepath.Base(path)
	base = strings.TrimSuffix(base, filepath.Ext(base))
	if i := strings.LastIndex(base, "_"); i >= 0 && i+1 < len(base) {
		return base[i+1:]
	}
	return base
}
