package opencode

import (
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hmepas/resume/internal/adapters/common"
	"github.com/hmepas/resume/internal/resume"
	_ "modernc.org/sqlite"
)

type Adapter struct{}

func (Adapter) ID() string { return "opencode" }

func (Adapter) Sessions(ctx resume.Context) ([]resume.Session, error) {
	root, err := common.HomePath(".local", "share", "opencode")
	if err != nil {
		return nil, err
	}

	var sessions []resume.Session
	seen := map[string]bool{}

	dbSessions, err := dbSessions(filepath.Join(root, "opencode.db"))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	for _, session := range dbSessions {
		if !ctx.All && !resume.PathMatches(ctx.Project, session.Project) {
			continue
		}
		sessions = append(sessions, session)
		seen[session.ID] = true
	}

	for _, sessionRoot := range sessionRoots(root) {
		projectPaths := readProjects(filepath.Join(filepath.Dir(sessionRoot), "project"))
		_ = common.WalkFiles(sessionRoot, func(path string) {
			if !strings.HasSuffix(path, ".json") {
				return
			}
			session := parseSessionFile(path, projectPaths)
			if session.SourcePath == "" || seen[session.ID] {
				return
			}
			if !ctx.All && !resume.PathMatches(ctx.Project, session.Project) {
				return
			}
			sessions = append(sessions, session)
			seen[session.ID] = true
		})
	}
	return sessions, nil
}

func dbSessions(path string) ([]resume.Session, error) {
	if !common.Exists(path) {
		return nil, os.ErrNotExist
	}
	db, err := sql.Open("sqlite", path+"?mode=ro")
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.Query(`
select s.id, s.directory, s.title, s.time_created, s.time_updated
from session s
order by s.time_updated desc`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []resume.Session
	for rows.Next() {
		var (
			id        string
			project   string
			title     string
			createdMS int64
			updatedMS int64
		)
		if err := rows.Scan(&id, &project, &title, &createdMS, &updatedMS); err != nil {
			return nil, err
		}
		if id == "" || project == "" {
			continue
		}
		sessions = append(sessions, newSession(id, project, title, msTime(createdMS), msTime(updatedMS), path, "high"))
	}
	return sessions, rows.Err()
}

func sessionRoots(root string) []string {
	return []string{
		filepath.Join(root, "storage", "session"),
	}
}

func readProjects(root string) map[string]string {
	projects := map[string]string{}
	_ = common.WalkFiles(root, func(path string) {
		if !strings.HasSuffix(path, ".json") {
			return
		}
		var raw struct {
			ID       string `json:"id"`
			Worktree string `json:"worktree"`
		}
		if readJSON(path, &raw) == nil && raw.ID != "" && raw.Worktree != "" {
			projects[raw.ID] = raw.Worktree
		}
	})
	return projects
}

func parseSessionFile(path string, projects map[string]string) resume.Session {
	var raw struct {
		ID        string `json:"id"`
		ProjectID string `json:"projectID"`
		Directory string `json:"directory"`
		Title     string `json:"title"`
		Time      struct {
			Created int64 `json:"created"`
			Updated int64 `json:"updated"`
		} `json:"time"`
	}
	if readJSON(path, &raw) != nil {
		return resume.Session{}
	}

	project := raw.Directory
	if project == "" {
		project = projects[raw.ProjectID]
	}
	if raw.ID == "" || project == "" {
		return resume.Session{}
	}
	return newSession(raw.ID, project, raw.Title, msTime(raw.Time.Created), msTime(raw.Time.Updated), path, "high")
}

func readJSON(path string, v any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

func newSession(id, project, title string, startedAt, updatedAt time.Time, sourcePath, confidence string) resume.Session {
	if updatedAt.IsZero() {
		updatedAt = common.FileModTime(sourcePath)
	}
	return resume.Session{
		Agent:      "opencode",
		ID:         id,
		Project:    project,
		StartedAt:  startedAt,
		UpdatedAt:  updatedAt,
		Title:      title,
		SourcePath: sourcePath,
		ResumeHint: "opencode -s " + id,
		Confidence: confidence,
	}
}

func msTime(ms int64) time.Time {
	if ms == 0 {
		return time.Time{}
	}
	return time.UnixMilli(ms)
}
