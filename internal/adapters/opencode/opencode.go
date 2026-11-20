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
)

type Adapter struct{}

func (Adapter) ID() string { return "opencode" }

func (Adapter) Color() resume.Color { return resume.ColorYellow }

func (Adapter) Command(session resume.Session) (resume.Command, error) {
	return common.ResumeCommand(session, "-s")
}

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
	db, err := common.OpenSQLiteRO(path)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	table, err := sessionTable(db)
	if err != nil {
		return nil, err
	}
	if table == "" {
		return nil, nil
	}

	rows, err := db.Query(`
select s.id, s.directory, s.title, s.time_created, s.time_updated
from ` + table + ` s
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
			title     sql.NullString
			createdMS int64
			updatedMS int64
		)
		if err := rows.Scan(&id, &project, &title, &createdMS, &updatedMS); err != nil {
			return nil, err
		}
		if id == "" || project == "" {
			continue
		}
		sessions = append(sessions, newSession(id, project, title.String, msTime(createdMS), msTime(updatedMS), path, "high"))
	}
	return sessions, rows.Err()
}

// sessionTable returns the sessions table for the opencode schema in use:
// session_v2 (opencode 2.x) or the legacy session table. Empty if neither exists.
func sessionTable(db *sql.DB) (string, error) {
	var name string
	err := db.QueryRow(`
select name from sqlite_master
where type = 'table' and name in ('session_v2', 'session')
order by name = 'session_v2' desc
limit 1`).Scan(&name)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return name, err
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
		Confidence: confidence,
	}
}

func msTime(ms int64) time.Time {
	if ms == 0 {
		return time.Time{}
	}
	return time.UnixMilli(ms)
}
