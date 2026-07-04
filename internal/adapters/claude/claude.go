package claude

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/hmepas/resume/internal/adapters/common"
	"github.com/hmepas/resume/internal/resume"
)

type Adapter struct{}

func (Adapter) ID() string { return "claude" }

func (Adapter) Sessions(ctx resume.Context) ([]resume.Session, error) {
	root, err := common.HomePath(".claude", "projects")
	if err != nil {
		return nil, err
	}
	if !common.Exists(root) {
		return nil, nil
	}

	activeNames := activeSessionNames()
	var sessions []resume.Session
	_ = common.WalkFiles(root, func(path string) {
		if strings.HasSuffix(path, ".jsonl") && !isClaudeChildSession(path) {
			session := parseSession(path, activeNames)
			if session.SourcePath == "" {
				return
			}
			if !ctx.All && !resume.PathMatches(ctx.Project, session.Project) {
				return
			}
			sessions = append(sessions, session)
		}
	})
	return sessions, nil
}

func activeSessionNames() map[string]string {
	dir, err := common.HomePath(".claude", "sessions")
	if err != nil || !common.Exists(dir) {
		return nil
	}
	names := make(map[string]string)
	_ = common.WalkFiles(dir, func(path string) {
		if !strings.HasSuffix(path, ".json") {
			return
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return
		}
		var obj map[string]any
		if err := json.Unmarshal(data, &obj); err != nil {
			return
		}
		id := common.String(obj, "sessionId")
		name := strings.TrimSpace(common.String(obj, "name"))
		if id != "" && name != "" {
			names[id] = name
		}
	})
	return names
}

func isClaudeChildSession(path string) bool {
	parts := strings.Split(filepath.ToSlash(path), "/")
	for _, part := range parts {
		if part == "subagents" || part == "tool-results" {
			return true
		}
	}
	return false
}

func decodeProjectPath(dir string) string {
	base := filepath.Base(dir)
	if !strings.HasPrefix(base, "-") {
		return ""
	}
	return string(filepath.Separator) + strings.ReplaceAll(strings.TrimPrefix(base, "-"), "-", string(filepath.Separator))
}

func parseSession(path string, activeNames map[string]string) resume.Session {
	project := decodeProjectPath(filepath.Dir(path))
	var title string
	var aiTitle string
	var customTitle string
	var commandTitle string
	var startupTitle string
	var sessionID string
	var startedAt time.Time
	var updatedAt time.Time

	_ = common.JSONLLines(path, func(obj map[string]any) {
		if sessionID == "" {
			sessionID = common.String(obj, "sessionId")
		}
		if aiTitle == "" {
			aiTitle = common.String(obj, "aiTitle")
		}
		if common.String(obj, "type") == "custom-title" {
			if value := common.String(obj, "customTitle"); value != "" {
				customTitle = value
			}
		}
		if commandTitle == "" {
			commandTitle = commandName(common.FirstUserText(obj))
		}
		if startupTitle == "" {
			startupTitle = startupHookTitle(obj)
		}
		if cwd := common.String(obj, "cwd"); cwd != "" {
			project = cwd
		}
		if ts := common.ParseTime(common.String(obj, "timestamp")); !ts.IsZero() {
			if startedAt.IsZero() || ts.Before(startedAt) {
				startedAt = ts
			}
			if ts.After(updatedAt) {
				updatedAt = ts
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
	if project == "" {
		return resume.Session{}
	}
	if sessionID == "" {
		sessionID = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}
	displayName := strings.TrimSpace(activeNames[sessionID])
	if displayName == "" {
		displayName = customTitle
	}
	if displayName == "" {
		displayName = aiTitle
	}
	if displayName == "" {
		displayName = commandTitle
	}
	if displayName == "" {
		displayName = startupTitle
	}
	if displayName == "" && title == "" {
		return resume.Session{}
	}
	title = combineTitle(displayName, title)

	return resume.Session{
		Agent:      "claude",
		ID:         sessionID,
		Project:    project,
		StartedAt:  startedAt,
		UpdatedAt:  updatedAt,
		Title:      title,
		SourcePath: path,
		ResumeHint: "claude --resume " + sessionID,
		Confidence: "high",
	}
}

var commandNamePattern = regexp.MustCompile(`(?s)<command-name>\s*([^<]+?)\s*</command-name>`)

func commandName(text string) string {
	matches := commandNamePattern.FindStringSubmatch(text)
	if len(matches) != 2 {
		return ""
	}
	return strings.TrimSpace(matches[1])
}

func startupHookTitle(obj map[string]any) string {
	attachment, ok := obj["attachment"].(map[string]any)
	if !ok || common.String(attachment, "type") != "hook_success" {
		return ""
	}
	text := strings.TrimSpace(common.String(attachment, "content"))
	if !common.UsefulTitle(text) {
		return ""
	}
	return text
}

func combineTitle(name, text string) string {
	if name == "" {
		return text
	}
	if text == "" || text == name {
		return name
	}
	return name + ": " + text
}
