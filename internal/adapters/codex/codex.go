package codex

import (
	"path/filepath"
	"strings"
	"time"

	"github.com/hmepas/resume/internal/adapters/common"
	"github.com/hmepas/resume/internal/resume"
)

type Adapter struct{}

func (Adapter) ID() string { return "codex" }

func (Adapter) Sessions(ctx resume.Context) ([]resume.Session, error) {
	root, err := common.HomePath(".codex")
	if err != nil {
		return nil, err
	}
	if !common.Exists(root) {
		return nil, nil
	}

	names := readIndex(filepath.Join(root, "session_index.jsonl"))
	var sessions []resume.Session

	for _, dir := range []string{
		filepath.Join(root, "sessions"),
		filepath.Join(root, "archived_sessions"),
	} {
		_ = common.WalkFiles(dir, func(path string) {
			if !strings.HasSuffix(path, ".jsonl") {
				return
			}
			session := parseSession(path, names)
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

func readIndex(path string) map[string]string {
	out := map[string]string{}
	_ = common.JSONLLines(path, func(obj map[string]any) {
		id := common.String(obj, "id")
		title := common.String(obj, "thread_name")
		if id != "" && title != "" {
			out[id] = title
		}
	})
	return out
}

func parseSession(path string, names map[string]string) resume.Session {
	var (
		id        string
		project   string
		title     string
		startedAt time.Time
		updatedAt time.Time
	)

	_ = common.JSONLLines(path, func(obj map[string]any) {
		ts := common.ParseTime(common.String(obj, "timestamp"))
		if !ts.IsZero() {
			if startedAt.IsZero() || ts.Before(startedAt) {
				startedAt = ts
			}
			if ts.After(updatedAt) {
				updatedAt = ts
			}
		}

		if common.String(obj, "type") == "session_meta" {
			if v := common.String(obj, "payload", "id"); v != "" {
				id = v
			}
			if v := common.String(obj, "payload", "cwd"); v != "" {
				project = v
			}
		}
		if title == "" {
			if text := common.FirstUserText(obj); common.UsefulTitle(text) {
				title = text
			}
		}
	})

	if updatedAt.IsZero() {
		updatedAt = common.FileModTime(path)
	}
	if name := names[id]; name != "" {
		title = combineTitle(name, title)
	}
	if project == "" {
		return resume.Session{}
	}

	return resume.Session{
		Agent:      "codex",
		ID:         id,
		Project:    project,
		StartedAt:  startedAt,
		UpdatedAt:  updatedAt,
		Title:      title,
		SourcePath: path,
		ResumeHint: "codex resume " + id,
		Confidence: "high",
	}
}

func combineTitle(name, text string) string {
	if text == "" || text == name {
		return name
	}
	return name + ": " + text
}
