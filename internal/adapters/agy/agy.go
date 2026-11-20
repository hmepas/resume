package agy

import (
	"bufio"
	"bytes"
	"database/sql"
	"encoding/json"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hmepas/resume/internal/adapters/common"
	"github.com/hmepas/resume/internal/resume"
	_ "modernc.org/sqlite"
)

const (
	summariesDB       = "conversation_summaries.db"
	maxTranscriptLine = 1024 * 1024
)

type Adapter struct{}

func (Adapter) ID() string { return "agy" }

func (Adapter) Sessions(ctx resume.Context) ([]resume.Session, error) {
	root, err := dataRoot()
	if err != nil {
		return nil, err
	}
	return loadSessions(root, ctx)
}

// dataRoot returns the Antigravity CLI data directory, resolving a symlinked
// root (common with dotfile managers).
func dataRoot() (string, error) {
	root, err := common.HomePath(".gemini", "antigravity-cli")
	if err != nil {
		return "", err
	}
	if resolved, err := filepath.EvalSymlinks(root); err == nil {
		root = resolved
	}
	return root, nil
}

// sqliteURI builds a file: URI so paths with spaces or special characters are
// escaped correctly before the query parameters are appended.
func sqliteURI(path, query string) string {
	return (&url.URL{Scheme: "file", Path: filepath.ToSlash(path), RawQuery: query}).String()
}

func loadSessions(root string, ctx resume.Context) ([]resume.Session, error) {
	dbPath := filepath.Join(root, summariesDB)
	if !common.Exists(dbPath) {
		return nil, nil
	}

	db, err := sql.Open("sqlite", sqliteURI(dbPath, "mode=ro&_pragma=busy_timeout(2000)"))
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.Query(`
SELECT conversation_id, title, preview, step_count, last_modified_time,
       workspace_uris, parent_conversation_id, nesting_depth, last_user_input_time
FROM conversation_summaries
ORDER BY last_modified_time DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []resume.Session
	for rows.Next() {
		var (
			id, title, preview, workspaceURIs, parentID sql.NullString
			stepCount, nestingDepth                     sql.NullInt64
			lastModifiedRaw, lastUserInputRaw           any
		)
		if err := rows.Scan(&id, &title, &preview, &stepCount, &lastModifiedRaw,
			&workspaceURIs, &parentID, &nestingDepth, &lastUserInputRaw); err != nil {
			return nil, err
		}
		if id.String == "" {
			continue
		}
		// Subagent conversations are not resumable on their own.
		if parentID.String != "" || nestingDepth.Int64 > 0 {
			continue
		}
		if stepCount.Int64 <= 0 {
			continue
		}
		paths := workspacePaths(workspaceURIs.String)
		if len(paths) == 0 {
			continue
		}
		project := ""
		if ctx.All {
			project = paths[0]
		} else {
			for _, p := range paths {
				if resume.PathMatches(ctx.Project, p) {
					project = p
					break
				}
			}
		}
		if project == "" {
			continue
		}

		sessions = append(sessions, buildSession(root, dbPath, id.String, project,
			title.String, preview.String, toTime(lastModifiedRaw), toTime(lastUserInputRaw)))
	}
	return sessions, rows.Err()
}

func buildSession(root, dbPath, id, project, title, preview string, lastModified, lastUserInput time.Time) resume.Session {
	transcriptPath := filepath.Join(root, "brain", id, ".system_generated", "logs", "transcript.jsonl")
	convDBPath := filepath.Join(root, "conversations", id+".db")

	sourcePath := dbPath
	hasTranscript := common.Exists(transcriptPath)
	switch {
	case hasTranscript:
		sourcePath = transcriptPath
	case common.Exists(convDBPath):
		sourcePath = convDBPath
	}

	var startedAt time.Time
	var prompt string
	if hasTranscript {
		startedAt, prompt = readTranscriptHead(transcriptPath)
	}

	sessionTitle := strings.TrimSpace(title)
	if sessionTitle == "" {
		sessionTitle = strings.TrimSpace(preview)
	}
	if sessionTitle == "" {
		sessionTitle = prompt
	}

	updatedAt := lastModified
	if updatedAt.IsZero() {
		updatedAt = lastUserInput
	}
	if updatedAt.IsZero() {
		updatedAt = common.FileModTime(sourcePath)
	}
	if startedAt.IsZero() {
		startedAt = lastUserInput
	}

	return resume.Session{
		Agent:      "agy",
		ID:         id,
		Project:    project,
		StartedAt:  startedAt,
		UpdatedAt:  updatedAt,
		Title:      sessionTitle,
		SourcePath: sourcePath,
		ResumeHint: "agy --conversation " + id,
		Confidence: "high",
	}
}

// workspacePaths decodes the JSON array of workspace URIs into absolute paths.
func workspacePaths(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var uris []string
	if json.Unmarshal([]byte(raw), &uris) != nil {
		return nil
	}
	var paths []string
	for _, uri := range uris {
		uri = strings.TrimSpace(uri)
		if uri == "" {
			continue
		}
		if strings.HasPrefix(uri, "file://") {
			u, err := url.Parse(uri)
			if err != nil || u.Path == "" {
				continue
			}
			paths = append(paths, filepath.FromSlash(u.Path))
			continue
		}
		if filepath.IsAbs(uri) {
			paths = append(paths, uri)
		}
	}
	return paths
}

// readTranscriptHead parses only the first transcript line for the start time
// and the initial user prompt, so large transcripts are never read in full.
func readTranscriptHead(path string) (time.Time, string) {
	file, err := os.Open(path)
	if err != nil {
		return time.Time{}, ""
	}
	defer file.Close()

	reader := bufio.NewReaderSize(file, 64*1024)
	var line []byte
	for {
		part, readErr := reader.ReadSlice('\n')
		line = append(line, part...)
		if len(line) > maxTranscriptLine {
			return time.Time{}, ""
		}
		if readErr == bufio.ErrBufferFull {
			continue
		}
		if readErr != nil && readErr != io.EOF {
			return time.Time{}, ""
		}
		break
	}

	var entry struct {
		Type      string `json:"type"`
		CreatedAt string `json:"created_at"`
		Content   string `json:"content"`
	}
	if json.Unmarshal(bytes.TrimSpace(line), &entry) != nil {
		return time.Time{}, ""
	}
	startedAt := common.ParseTime(entry.CreatedAt)
	prompt := ""
	if entry.Type == "USER_INPUT" {
		prompt = userRequestText(entry.Content)
	}
	return startedAt, prompt
}

func userRequestText(content string) string {
	const open, close = "<USER_REQUEST>", "</USER_REQUEST>"
	if start := strings.Index(content, open); start >= 0 {
		rest := content[start+len(open):]
		if end := strings.Index(rest, close); end >= 0 {
			rest = rest[:end]
		}
		content = rest
	}
	content = strings.TrimSpace(content)
	if !common.UsefulTitle(content) {
		return ""
	}
	return content
}

var timeLayouts = []string{
	time.RFC3339Nano,
	time.RFC3339,
	"2006-01-02 15:04:05.999999999-07:00",
	"2006-01-02 15:04:05.999999999Z07:00",
	"2006-01-02 15:04:05-07:00",
	"2006-01-02 15:04:05",
}

func toTime(v any) time.Time {
	var t time.Time
	switch x := v.(type) {
	case time.Time:
		t = x
	case string:
		t = parseTimestamp(x)
	case []byte:
		t = parseTimestamp(string(x))
	default:
		return time.Time{}
	}
	if t.Year() <= 1 {
		return time.Time{}
	}
	return t
}

func parseTimestamp(s string) time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}
	}
	for _, layout := range timeLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Time{}
}
