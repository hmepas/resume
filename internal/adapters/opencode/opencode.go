package opencode

import (
	"path/filepath"
	"strings"
	"time"

	"github.com/hmepas/resume/internal/adapters/common"
	"github.com/hmepas/resume/internal/resume"
)

type Adapter struct{}

func (Adapter) ID() string { return "opencode" }

func (Adapter) Sessions(ctx resume.Context) ([]resume.Session, error) {
	roots := [][]string{
		{".local", "share", "opencode"},
		{".config", "opencode"},
		{".opencode"},
	}

	var sessions []resume.Session
	for _, parts := range roots {
		root, err := common.HomePath(parts...)
		if err != nil || !common.Exists(root) {
			continue
		}
		_ = common.WalkFiles(root, func(path string) {
			if !strings.HasSuffix(path, ".json") && !strings.HasSuffix(path, ".jsonl") {
				return
			}
			session := parseLoose("opencode", path)
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

func parseLoose(agent, path string) resume.Session {
	var project string
	var title string
	var startedAt time.Time
	var updatedAt time.Time

	_ = common.JSONLLines(path, func(obj map[string]any) {
		if project == "" {
			project = firstString(obj, "cwd", "project", "projectPath", "workspace", "workspacePath")
		}
		if title == "" {
			title = firstString(obj, "title", "summary", "prompt")
		}
		if title == "" {
			if text := common.FirstUserText(obj); common.UsefulTitle(text) {
				title = text
			}
		}
		if ts := firstTime(obj, "updatedAt", "updated_at", "timestamp", "createdAt", "created_at"); !ts.IsZero() {
			if startedAt.IsZero() || ts.Before(startedAt) {
				startedAt = ts
			}
			if ts.After(updatedAt) {
				updatedAt = ts
			}
		}
	})

	if project == "" {
		project = inferProjectFromPath(path)
	}
	if project == "" {
		return resume.Session{}
	}
	if updatedAt.IsZero() {
		updatedAt = common.FileModTime(path)
	}

	return resume.Session{
		Agent:      agent,
		ID:         sessionIDFromPath(path),
		Project:    project,
		StartedAt:  startedAt,
		UpdatedAt:  updatedAt,
		Title:      title,
		SourcePath: path,
		ResumeHint: agent,
		Confidence: "low",
	}
}

func sessionIDFromPath(path string) string {
	base := filepath.Base(path)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

func firstString(obj map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := obj[key].(string); ok && value != "" {
			return value
		}
	}
	return ""
}

func firstTime(obj map[string]any, keys ...string) time.Time {
	for _, key := range keys {
		if value, ok := obj[key].(string); ok {
			if ts := common.ParseTime(value); !ts.IsZero() {
				return ts
			}
		}
	}
	return time.Time{}
}

func inferProjectFromPath(path string) string {
	dir := filepath.Dir(path)
	for {
		if common.Exists(filepath.Join(dir, ".git")) {
			return dir
		}
		next := filepath.Dir(dir)
		if next == dir {
			return ""
		}
		dir = next
	}
}
