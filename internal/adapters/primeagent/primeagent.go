package primeagent

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hmepas/resume/internal/adapters/common"
	"github.com/hmepas/resume/internal/resume"
)

const (
	maxHeaderLine = 1024 * 1024
	boundedHead   = 64 * 1024
	boundedTail   = 64 * 1024
)

type Adapter struct{}

func (Adapter) ID() string { return "prime-agent" }

func (Adapter) Sessions(ctx resume.Context) ([]resume.Session, error) {
	root, err := sessionRoot(ctx)
	if err != nil {
		return nil, err
	}
	// WalkDir does not follow a symlinked root (common with dotfile managers).
	if resolved, err := filepath.EvalSymlinks(root); err == nil {
		root = resolved
	}
	if !common.Exists(root) {
		return nil, nil
	}

	var sessions []resume.Session
	_ = common.WalkFiles(root, func(path string) {
		if !strings.HasSuffix(path, ".jsonl") {
			return
		}
		// The header on line 1 carries the cwd, so non-matching sessions are
		// skipped without reading the rest of the file.
		header, ok := readHeader(path)
		if !ok {
			return
		}
		if !ctx.All && !resume.PathMatches(ctx.Project, header.CWD) {
			return
		}
		sessions = append(sessions, buildSession(path, header))
	})
	return sessions, nil
}

func sessionRoot(ctx resume.Context) (string, error) {
	// An empty env value counts as unset so it cannot mask the fallback var.
	for _, key := range []string{"PRIME_AGENT_SESSION_DIR", "PRIME_AGENT_CODING_AGENT_SESSION_DIR"} {
		if value := os.Getenv(key); value != "" {
			return resume.ExpandHome(value), nil
		}
	}

	agentDir, err := agentRoot()
	if err != nil {
		return "", err
	}
	if configured := configuredSessionRoot(agentDir, ctx.Project); configured != "" {
		return configured, nil
	}
	return filepath.Join(agentDir, "sessions"), nil
}

func agentRoot() (string, error) {
	if value := os.Getenv("PRIME_AGENT_CODING_AGENT_DIR"); value != "" {
		return resume.ExpandHome(value), nil
	}
	return common.HomePath(".prime", "agent")
}

// configuredSessionRoot reads sessionDir from the settings files; later files
// win, so a project setting overrides the global one, and a configured empty
// value masks earlier files and selects the default root.
func configuredSessionRoot(agentDir string, project resume.Project) string {
	paths := []string{filepath.Join(agentDir, "settings.json")}
	if project.Root != "" {
		paths = append(paths, filepath.Join(project.Root, ".prime", "agent", "settings.json"))
	}
	if project.Path != "" && project.Path != project.Root {
		paths = append(paths, filepath.Join(project.Path, ".prime", "agent", "settings.json"))
	}

	root := ""
	for _, path := range paths {
		if value, ok := readSessionRoot(path); ok {
			root = value
		}
	}
	return root
}

func readSessionRoot(path string) (string, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	var settings map[string]any
	if json.Unmarshal(data, &settings) != nil {
		return "", false
	}
	value, ok := settings["sessionDir"].(string)
	if !ok || value == "" {
		return "", ok
	}
	value = resume.ExpandHome(value)
	if !filepath.IsAbs(value) {
		// Relative to the settings file, not the process cwd.
		value = filepath.Join(filepath.Dir(path), value)
	}
	return value, true
}

type sessionHeader struct {
	Type      string `json:"type"`
	ID        string `json:"id"`
	CWD       string `json:"cwd"`
	Timestamp string `json:"timestamp"`
}

// readHeader parses the first line, which must be the session header; a file
// that opens with anything else is not a prime session.
func readHeader(path string) (sessionHeader, bool) {
	file, err := os.Open(path)
	if err != nil {
		return sessionHeader{}, false
	}
	defer file.Close()

	reader := bufio.NewReaderSize(file, 64*1024)
	var line []byte
	for {
		part, readErr := reader.ReadSlice('\n')
		line = append(line, part...)
		if len(line) > maxHeaderLine {
			return sessionHeader{}, false
		}
		if readErr == bufio.ErrBufferFull {
			continue
		}
		if readErr != nil && readErr != io.EOF {
			return sessionHeader{}, false
		}
		break
	}

	var header sessionHeader
	if json.Unmarshal(bytes.TrimSpace(line), &header) != nil {
		return sessionHeader{}, false
	}
	if header.Type != "session" || header.ID == "" || header.CWD == "" {
		return sessionHeader{}, false
	}
	return header, true
}

// parseSession is the single-file entry point used by tests; Sessions calls
// readHeader separately to prefilter by cwd before scanning the body.
func parseSession(path string) resume.Session {
	header, ok := readHeader(path)
	if !ok {
		return resume.Session{}
	}
	return buildSession(path, header)
}

func buildSession(path string, header sessionHeader) resume.Session {
	startedAt := common.ParseTime(header.Timestamp)
	name, title, updatedAt := scanBody(path, true)
	if name == "" && title == "" {
		// The bounded scan can miss a title buried mid-file; retry in full.
		if info, err := os.Stat(path); err == nil && info.Size() > boundedHead+boundedTail {
			name, title, updatedAt = scanBody(path, false)
		}
	}
	if name != "" {
		title = name
	}
	if updatedAt.IsZero() {
		updatedAt = startedAt
	}
	if updatedAt.IsZero() {
		updatedAt = common.FileModTime(path)
	}

	return resume.Session{
		Agent:      "prime-agent",
		ID:         header.ID,
		Project:    header.CWD,
		StartedAt:  startedAt,
		UpdatedAt:  updatedAt,
		Title:      title,
		SourcePath: path,
		ResumeHint: "prime-agent --resume " + header.ID,
		Confidence: "high",
	}
}

func scanBody(path string, bounded bool) (name, title string, updatedAt time.Time) {
	scan := common.JSONLLines
	if bounded {
		scan = func(path string, fn func(map[string]any)) error {
			return common.JSONLBounded(path, boundedHead, boundedTail, fn)
		}
	}
	_ = scan(path, func(obj map[string]any) {
		switch common.String(obj, "type") {
		case "session":
			return
		case "session_info":
			// An empty name clears an earlier rename, so the last entry wins.
			name = strings.TrimSpace(common.String(obj, "name"))
		}
		if ts := entryTime(obj); ts.After(updatedAt) {
			updatedAt = ts
		}
		if title == "" {
			if text := common.FirstUserText(obj); common.UsefulTitle(text) {
				title = text
			}
		}
	})
	return name, title, updatedAt
}

// entryTime prefers the inner message timestamp and falls back to the outer
// entry timestamp when it is absent, null, or unparseable.
func entryTime(obj map[string]any) time.Time {
	if msg, ok := obj["message"].(map[string]any); ok {
		switch v := msg["timestamp"].(type) {
		case float64:
			if v > 0 {
				return time.UnixMilli(int64(v))
			}
		case string:
			if ts := common.ParseTime(v); !ts.IsZero() {
				return ts
			}
		}
	}
	return common.ParseTime(common.String(obj, "timestamp"))
}
